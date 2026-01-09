# DR-029: Cookies Command Interface

- Date: 2025-12-28
- Status: Accepted
- Category: CLI

## Problem

The current cookies command lacks output mode flexibility and filtering capabilities, making it inconsistent with other observation commands. Current limitations:

- No option to save cookies to file for later analysis
- Cannot preserve cookies for archival, session transfer, or testing
- Inconsistent with html, css, console, network commands that support file output
- Missing universal pattern (default/show/save) for output control
- No text search for filtering cookies by name or value
- Mutation operations (set/delete) exist but observation interface is incomplete

Users need cookies to follow the universal observation pattern for reading cookies while maintaining separate mutation operations for modifying cookies.

## Decision

Redesign cookies command to follow the universal observation pattern for cookie retrieval, with separate mutation subcommands:

```bash
# Universal pattern (observation)
webctl cookies                  # Output all cookies to stdout (Unix convention)
webctl cookies save [path]      # Save to file (temp if no path, custom if path given)

# Universal flags (observation)
--find, -f TEXT                 # Search within cookie names and values
--raw                           # Skip formatting
--json                          # JSON output

# Cookies-specific filter flags (observation)
--domain DOMAIN                 # Filter by domain
--name NAME                     # Filter by name (exact match)

# Cookies-specific mutation subcommands
webctl cookies set <name> <value> [flags]    # Set cookie
webctl cookies delete <name> [flags]         # Delete cookie
```

The cookies command outputs to stdout by default (Unix convention), with a save subcommand for file output. Mutation operations (set/delete) remain as separate subcommands to maintain clear separation between read and write operations.

Complete specification: docs/design/interface/cookies.md

## Why

Unix Convention (stdout by default):

Following Unix philosophy, observation commands output to stdout by default. This enables:
- Piping to other tools (grep, less, jq)
- Quick inspection without file management
- Consistent with standard CLI tools

Separation of Observation and Mutation:

Cookies have two distinct use cases:

1. Observation: Reading current cookies for debugging, analysis, session transfer
2. Mutation: Setting or deleting cookies for testing, authentication

The universal pattern applies to observation only. Mutation operations remain as separate subcommands (set/delete) to maintain clear distinction between read and write operations.

Save Subcommand for Files:

When file output is needed, the save subcommand provides flexibility:
- `cookies save` - saves to temp directory with auto-generated filename
- `cookies save ./cookies.json` - saves to custom path
- Directory paths auto-generate filenames, file paths use exact names

Cookies-Specific Filter Flags:

Cookie filtering requires domain and name filters:

- --domain: Filter by cookie domain (essential for multi-domain sites)
- --name: Filter by exact cookie name (find specific cookies)

These filters are cookie-specific and don't apply to other observation commands.

Text Search Integration:

The --find flag enables searching within cookie names and values, matching the pattern for other observation commands. Users can filter cookies by content, useful for finding session tokens or authentication cookies.

Mutation Subcommands Retained:

The set and delete subcommands handle cookie mutations:

- set: Create or update cookies with full attribute control
- delete: Remove cookies by name and domain

These operations are distinct from observation and remain as separate subcommands. This separation makes the command intent clear and prevents accidental mutations during observation.

## Trade-offs

Accept:

- Breaking change from current cookies command behavior
- Default to file may surprise users expecting stdout
- Temp files require eventual cleanup
- More complex command structure (observation + mutation)
- Users must learn both observation pattern and mutation subcommands
- Two separate behaviors within one command

Gain:

- Consistent observation interface across all commands
- Cookie preservation for debugging and session transfer
- Flexible output modes for different use cases
- File output for testing and automation
- Clear separation between read and write operations
- Integrated text search for cookie filtering
- Filter flags work across all observation modes
- Foundation matches other observation commands
- Mutation operations remain simple and clear

## Alternatives

Keep Current Behavior:

Current cookies command may output to stdout only without universal pattern support.

- Pro: No breaking changes
- Pro: Simple single behavior
- Con: No way to save cookies to file
- Con: Inconsistent with other observation commands
- Con: Cannot preserve cookies for session transfer
- Rejected: Fails to provide file output capability and consistency

Remove Mutation Subcommands:

```bash
webctl cookies              # Only observation
# Separate commands for mutation:
webctl cookie-set <name> <value>
webctl cookie-delete <name>
```

- Pro: Pure observation command
- Pro: Matches pattern more cleanly
- Con: Splits cookie functionality across multiple commands
- Con: Less discoverable
- Con: Breaking change for existing mutation usage
- Rejected: Keeping mutations as subcommands maintains grouping

Merge Set/Delete into Observation Pattern:

```bash
webctl cookies set <name> <value>     # Mutation as "output mode"?
webctl cookies delete <name>          # Mutation as "output mode"?
```

- Pro: Everything under one command
- Con: Confuses observation modes with mutations
- Con: set/delete aren't output modes
- Con: Violates clear separation principle
- Rejected: Mutations should not be treated as output modes

Separate Commands for Observation and Mutation:

```bash
webctl cookies-get              # Observation only
webctl cookies-set <name> <value>   # Mutation
webctl cookies-delete <name>        # Mutation
```

- Pro: Very clear separation
- Pro: No mixing of concerns
- Con: Three separate commands instead of one
- Con: Clutters namespace
- Con: Less discoverable
- Rejected: Subcommands group related functionality better

All Observation, No Mutations:

```bash
webctl cookies              # Only observation with universal pattern
# No set/delete support
```

- Pro: Pure observation command
- Pro: Simplest interface
- Con: Users need cookie mutation for testing
- Con: Loses valuable functionality
- Con: Incomplete cookie management
- Rejected: Mutation operations are essential for testing workflows

## Structure

Observation Pattern (Universal):

Default (no subcommand):
- Outputs all cookies to stdout (Unix convention)
- Formatted table with name, value, domain, path, expiry, flags
- Useful for piping to other tools

Save subcommand:
- Optional path argument
- No path: saves to /tmp/webctl-cookies/ with auto-generated filename
- Directory: auto-generates filename in that directory
- File: saves to exact path
- Creates parent directories if needed

Universal Flags (Observation):

--find, -f TEXT:
- Search for text within cookie names and values
- Filters cookies containing search text
- Works across all observation modes
- Case-insensitive search

--raw:
- Skips formatting/pretty-printing
- Returns cookies in raw format
- Useful for machine processing

--json:
- Global flag for JSON output format
- Array of cookie objects
- Each cookie: name, value, domain, path, expires, httpOnly, secure, sameSite, size, session

Cookies-Specific Filter Flags (Observation):

--domain DOMAIN:
- Filter by cookie domain
- Exact match or domain suffix match
- Example: --domain ".github.com"

--name NAME:
- Filter by exact cookie name
- Example: --name "session_id"

Mutation Subcommands:

set <name> <value>:
- Creates or updates a cookie
- Required arguments: name, value
- Optional flags:
  - --domain DOMAIN: Cookie domain (defaults to current page)
  - --path PATH: Cookie path (defaults to "/")
  - --secure: Require HTTPS
  - --httponly: Prevent JavaScript access
  - --max-age SECONDS: Expiry in seconds (0 = session cookie)
  - --samesite POLICY: SameSite policy (Strict, Lax, None)

delete <name>:
- Deletes a cookie by name
- Required argument: name
- Optional flags:
  - --domain DOMAIN: Required if multiple cookies match

## Usage Examples

Default behavior (stdout):

```bash
webctl cookies
# session_id | abc123 | .example.com | / | Session | Secure, HttpOnly
# remember   | xyz789 | example.com  | / | 2026-01-01 | Secure

webctl cookies --domain ".github.com"
# user_session | token123 | .github.com | / | 2026-01-01 | Secure, HttpOnly
# _gh_sess     | sess456  | github.com  | / | Session | Secure

webctl cookies --find "session"
# session_id   | abc123 | .example.com | / | Session | Secure, HttpOnly
# user_session | token123 | .github.com | / | 2026-01-01 | Secure, HttpOnly
```

Save to file:

```bash
webctl cookies save
# {"ok": true, "path": "/tmp/webctl-cookies/25-12-28-143052-cookies.json"}

webctl cookies save ./cookies.json
# {"ok": true, "path": "./cookies.json"}

webctl cookies save ./output/
# {"ok": true, "path": "./output/25-12-28-143052-cookies.json"}

webctl cookies save ./auth-cookies.json --find "auth"
# {"ok": true, "path": "./auth-cookies.json"}
```

Cookies-specific filters:

```bash
# Domain filtering
webctl cookies --domain ".example.com"
webctl cookies --domain "github.com"

# Name filtering
webctl cookies --name "session_id"

# Search within cookies
webctl cookies --find "session"
webctl cookies --find "token"

# Combined filtering
webctl cookies --domain ".github.com" --find "session"
webctl cookies save ./github-sessions.json --domain ".github.com" --find "session"
```

Mutation subcommands:

```bash
# Set session cookie
webctl cookies set session_id abc123
# OK

# Set persistent cookie (1 hour)
webctl cookies set remember_me yes --max-age 3600
# OK

# Set secure cookie
webctl cookies set auth_token xyz789 --secure --httponly
# OK

# Set cookie with domain
webctl cookies set tracking id123 --domain .example.com
# OK

# Set cookie with SameSite
webctl cookies set csrf_token xyz --samesite Strict
# OK

# Delete cookie
webctl cookies delete session_id
# OK

# Delete cookie with domain (if ambiguous)
webctl cookies delete session_id --domain api.example.com
# OK
```

Session transfer workflow:

```bash
# Save cookies from source environment
webctl cookies save ./production-cookies.json --domain ".example.com"

# Later, load and set cookies in test environment
# (implementation detail: read JSON and set each cookie)
```

JSON output:

```bash
webctl cookies --json
# {
#   "ok": true,
#   "cookies": [
#     {
#       "name": "session_id",
#       "value": "abc123",
#       "domain": ".example.com",
#       "path": "/",
#       "expires": null,
#       "httpOnly": true,
#       "secure": true,
#       "sameSite": "Lax",
#       "size": 18,
#       "session": true
#     },
#     ...
#   ]
# }
```

## File Naming

Auto-generated Filenames:

Pattern: /tmp/webctl-cookies/YY-MM-DD-HHMMSS-cookies.json

Default extension: .json (cookies are structured data)

Example filenames:
- 25-12-28-143052-cookies.json
- 25-12-28-143115-cookies.json
- 25-12-28-143120-cookies.json

Identifier: Fixed to "cookies" (no variation needed)

## Output Format

Text Mode (default):

Formatted table:
```
Name         | Value   | Domain        | Path | Expiry     | Flags
session_id   | abc123  | .example.com  | /    | Session    | Secure, HttpOnly
remember_me  | yes     | example.com   | /    | 2026-01-01 | Secure
tracking_id  | id123   | .example.com  | /    | 2025-02-01 | None
```

Use --raw to disable formatting.

JSON Mode (--json flag):

Array of cookie objects:
```json
{
  "ok": true,
  "cookies": [
    {
      "name": "session_id",
      "value": "abc123",
      "domain": ".example.com",
      "path": "/",
      "expires": null,
      "httpOnly": true,
      "secure": true,
      "sameSite": "Lax",
      "size": 18,
      "session": true
    }
  ]
}
```

## Breaking Changes

From DR-015 (Cookies Command Interface):

1. Changed: Default behavior now outputs to stdout (Unix convention)
2. Removed: show subcommand (not needed - stdout is default)
3. Changed: save subcommand now takes optional path (temp if no path)
4. Added: --find flag for text search within cookie names and values
5. Added: --domain and --name filter flags
6. Retained: set and delete subcommands (behavior unchanged)

Migration Guide:

Old pattern (DR-015 or current):
```bash
webctl cookies                       # Stdout or other default
webctl cookies set session abc123    # Set cookie
webctl cookies delete session        # Delete cookie
```

New pattern (DR-029 after P-051):
```bash
webctl cookies                       # Output to stdout (same as before)
webctl cookies save                  # Save to temp (new)
webctl cookies save ./cookies.json   # Save to custom path (new feature)
webctl cookies set session abc123    # Set cookie (same)
webctl cookies delete session        # Delete cookie (same)
```

The default stdout behavior is preserved. Use `webctl cookies save` when file output is needed.

Mutation subcommands (set/delete) remain unchanged for backward compatibility.

## Updates

- 2026-01-09: Updated to stdout default, removed show subcommand (P-051)
- 2025-12-28: Initial version (supersedes DR-015)
