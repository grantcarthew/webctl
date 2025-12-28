# DR-015: Cookies Command Interface

- Date: 2025-12-22
- Status: Superseded by [DR-029](../dr-029-cookies-command-interface.md)
- Category: CLI

## Problem

AI agents debugging web applications need to inspect and manipulate cookies to understand authentication state, session data, and tracking behavior. Cookies affect page behavior and are critical for diagnosing login issues, session problems, and security configurations.

Requirements:

- Retrieve all cookies for the current page
- Set cookies with full attribute control
- Delete cookies by name
- Display all cookie attributes from CDP
- Work with active session in multi-tab scenarios

## Decision

Implement `webctl cookies` command with subcommands:

```bash
webctl cookies                      # List all cookies
webctl cookies set <name> <value>   # Set a cookie
webctl cookies delete <name>        # Delete a cookie
```

### List (default)

```bash
webctl cookies
```

Output:

```json
{
  "ok": true,
  "cookies": [
    {
      "name": "session",
      "value": "abc123",
      "domain": "example.com",
      "path": "/",
      "expires": 1735084800,
      "size": 20,
      "httpOnly": true,
      "secure": true,
      "sameSite": "Lax",
      "priority": "Medium",
      "sameParty": false,
      "sourceScheme": "Secure",
      "sourcePort": 443,
      "session": false
    }
  ],
  "count": 1
}
```

All CDP cookie attributes are included in the output.

### Set

```bash
webctl cookies set <name> <value> [flags]
```

Flags:

| Flag | Type | Description |
|------|------|-------------|
| --domain | string | Cookie domain (defaults to current page domain) |
| --path | string | Cookie path (defaults to "/") |
| --secure | bool | Require HTTPS |
| --httponly | bool | HTTP-only (no JavaScript access) |
| --max-age | int | Expiry in seconds from now (0 = session cookie) |
| --samesite | string | SameSite policy: "Strict", "Lax", or "None" |

Output:

```json
{"ok": true}
```

### Delete

```bash
webctl cookies delete <name> [flags]
```

Flags:

| Flag | Type | Description |
|------|------|-------------|
| --domain | string | Cookie domain (required if ambiguous) |

Behavior:

- If one cookie matches name: delete it
- If multiple cookies match name: return error listing matches, require --domain
- If no cookie matches: silent success (idempotent)

Output (success):

```json
{"ok": true}
```

Output (ambiguous):

```json
{
  "ok": false,
  "error": "multiple cookies named 'session' found",
  "matches": [
    {"name": "session", "domain": "example.com"},
    {"name": "session", "domain": "api.example.com"}
  ]
}
```

## Why

Subcommand structure:

Using `cookies set` and `cookies delete` instead of flags like `--set` follows established CLI patterns (git, docker, kubectl). Subcommands are more extensible and allow distinct flag sets per operation.

All CDP attributes:

Include every attribute CDP returns rather than curating a subset. This provides complete information for debugging without guessing what agents might need. Attributes can be ignored if not relevant.

Smart delete behavior:

Requiring --domain only when ambiguous balances convenience with safety. Single-match deletes are common and should be easy. Multi-match deletes could cause unintended data loss, so requiring disambiguation protects users.

Idempotent delete:

Deleting a non-existent cookie returns success. This matches standard REST semantics and simplifies scripts that ensure a cookie is gone without checking first.

max-age over expires:

Using --max-age (seconds from now) is more intuitive than absolute timestamps. Matches the HTTP Set-Cookie max-age directive. CDP converts this internally.

JSON only:

Text format deferred to reduce initial scope. Cookie data is structured and typically parsed by agents. Text format can be added later for human debugging.

## Trade-offs

Accept:

- Subcommand complexity (three operations in one command)
- No text format initially
- No filtering options (agents filter JSON themselves)
- Must handle ambiguous delete edge case

Gain:

- Complete cookie control (get/set/delete)
- Full CDP attribute visibility
- Safe delete behavior
- Intuitive duration-based expiry
- Extensible subcommand structure

## Alternatives

Flags instead of subcommands:

```bash
webctl cookies --set "name=value"
webctl cookies --delete "name"
```

- Pro: Single command structure
- Con: Awkward flag value parsing
- Con: Hard to add operation-specific flags
- Rejected: Subcommands are cleaner

Separate commands:

```bash
webctl cookies
webctl cookie-set <name> <value>
webctl cookie-delete <name>
```

- Pro: Simple individual commands
- Con: Pollutes command namespace
- Con: Less discoverable as related operations
- Rejected: Subcommands group related functionality

Always require --domain for delete:

- Pro: Always explicit
- Con: Tedious when unambiguous
- Rejected: Smart behavior is more user-friendly

## Usage Examples

List all cookies:

```bash
webctl cookies
```

Set a session cookie:

```bash
webctl cookies set session abc123
```

Set a persistent secure cookie:

```bash
webctl cookies set auth_token xyz789 --domain example.com --secure --httponly --max-age 86400
```

Set a cookie with SameSite:

```bash
webctl cookies set tracking id123 --samesite None --secure
```

Delete a cookie (unambiguous):

```bash
webctl cookies delete session
```

Delete a cookie (ambiguous):

```bash
webctl cookies delete session --domain api.example.com
```

Agent workflow - clear and set test cookie:

```bash
webctl cookies delete test_flag
webctl cookies set test_flag enabled --max-age 3600
webctl reload
webctl cookies  # Verify cookie is set
```

## Implementation Notes

CLI implementation:

- Root `cookies` command with no args calls list
- `set` subcommand parses name, value, and flags
- `delete` subcommand parses name and optional --domain
- Connect to daemon via IPC for each operation

Daemon implementation:

List:

- Call Network.getCookies (no URL = current page)
- Return all cookie objects as-is from CDP

Set:

- Call Network.setCookie with provided attributes
- Domain defaults to current page URL's domain
- Path defaults to "/"
- Convert max-age to expires timestamp

Delete:

- First call Network.getCookies to find matches
- If zero matches: return success (idempotent)
- If one match: call Network.deleteCookies with name and domain
- If multiple matches and no --domain: return error with matches
- If multiple matches with --domain: delete the specified one

Network domain:

- Cookies operations require Network domain enabled
- Use existing lazy enablement pattern from network command

REPL abbreviation:

- Add "cookies" to webctlCommands list
- "coo" expands to "cookies"

## CDP Methods

Network.getCookies:

```json
{
  "method": "Network.getCookies",
  "params": {},
  "sessionId": "..."
}
```

Network.setCookie:

```json
{
  "method": "Network.setCookie",
  "params": {
    "name": "session",
    "value": "abc123",
    "domain": "example.com",
    "path": "/",
    "secure": true,
    "httpOnly": true,
    "sameSite": "Lax",
    "expires": 1735084800
  },
  "sessionId": "..."
}
```

Network.deleteCookies:

```json
{
  "method": "Network.deleteCookies",
  "params": {
    "name": "session",
    "domain": "example.com"
  },
  "sessionId": "..."
}
```

## Testing Strategy

Unit tests:

- Subcommand routing (cookies, cookies set, cookies delete)
- Flag parsing for set command
- Flag parsing for delete command
- Ambiguous delete error formatting

Integration tests:

- List cookies on page with cookies
- List cookies on page with no cookies (empty array)
- Set cookie and verify it appears in list
- Set cookie with all flags
- Delete existing cookie
- Delete non-existent cookie (idempotent)
- Delete ambiguous cookie without --domain (error)
- Delete ambiguous cookie with --domain (success)
- Verify cookies use active session

## Session Context

Cookies command operates on the active session. When multiple browser tabs are open, cookies are retrieved/set/deleted for the currently active tab's context.

Session selection via `webctl target` command allows choosing which tab to operate on.

Cookies command does not support --clear flag as it is an observation command.

## Updates

- 2025-12-22: Initial version
