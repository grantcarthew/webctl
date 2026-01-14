# p-038: Testing cookies Command

- Status: In Progress
- Started: 2025-12-31

## Overview

Test the webctl cookies command which extracts and manipulates cookies from the current page. This command outputs to stdout by default, with save/set/delete subcommands for file output and mutations.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-cookies.sh
```

## Code References

- internal/cli/cookies.go
- internal/cli/format (cookies formatting)

## Command Signature

```
webctl cookies [save [path]|set <name> <value>|delete <name>] [--find text] [--domain domain] [--name name] [--raw]
```

Subcommands:
- (default): Output cookies to stdout
- save: Save to /tmp/webctl-cookies/ with auto-generated filename
- save <path>: Save to file or directory/ (trailing slash = directory)
- set <name> <value>: Set a cookie (mutation)
- delete <name>: Delete a cookie (mutation)

Universal flags (work with default/show/save modes):
- --find, -f <text>: Search for text within cookie names and values
- --raw: Skip formatting (return raw JSON)
- --json: Output in JSON format (global flag)

Cookies-specific filter flags (observation only - default/show/save):
- --domain <DOMAIN>: Filter by cookie domain
- --name <NAME>: Filter by exact cookie name

Set subcommand flags:
- --domain <DOMAIN>: Cookie domain (defaults to current page domain)
- --path <PATH>: Cookie path (defaults to "/")
- --secure: Require HTTPS connection
- --httponly: Prevent JavaScript access
- --max-age <SECONDS>: Expiry in seconds from now (0 = session cookie)
- --samesite <POLICY>: SameSite policy (Strict, Lax, None)

Delete subcommand flags:
- --domain <DOMAIN>: Cookie domain (required if ambiguous)

## Test Checklist

Default mode (stdout):
- [ ] cookies (all cookies to stdout)
- [ ] cookies --domain ".github.com" (only GitHub cookies)
- [ ] cookies --find "session" (search and output)
- [ ] cookies --name "session_id" (exact name match)
- [ ] Verify formatted text output to stdout
- [ ] Verify no file created

Save mode (file output):
- [ ] cookies save (all cookies to temp)
- [ ] cookies save --domain ".github.com" (only GitHub cookies)
- [ ] cookies save --find "session" (search and save)
- [ ] Verify file saved to /tmp/webctl-cookies/
- [ ] Verify auto-generated filename format (YY-MM-DD-HHMMSS-cookies.json)
- [ ] Verify JSON response with file path
- [ ] Verify JSON file structure (ok, cookies, count)

Save mode (custom path):
- [ ] cookies save ./cookies.json (save to file)
- [ ] cookies save ./output/ (save to dir with auto-filename, creates dir)
- [ ] cookies save ./output (save to file named "output", NOT a directory)
- [ ] cookies save ./auth-cookies.json --find "auth"
- [ ] Verify trailing slash behavior

Domain filter:
- [ ] --domain exact match (example.com)
- [ ] --domain with leading dot (.example.com)
- [ ] --domain matching subdomains
- [ ] --domain with no matches
- [ ] Case insensitivity of domain filter

Name filter:
- [ ] --name exact match
- [ ] --name with no matches
- [ ] --name case sensitivity
- [ ] --name combined with --domain

Find flag:
- [ ] --find in cookie name
- [ ] --find in cookie value
- [ ] --find case insensitive
- [ ] --find with no matches (error)
- [ ] --find combined with --domain
- [ ] --find combined with --name

Raw flag:
- [ ] --raw output (JSON format)
- [ ] Compare raw vs formatted output
- [ ] --raw with show mode
- [ ] --raw with filters

Set subcommand (session cookies):
- [ ] cookies set session abc123 (basic session cookie)
- [ ] cookies set auth token123 (another session cookie)
- [ ] Verify cookie created
- [ ] Verify default domain (current page)
- [ ] Verify default path (/)
- [ ] Verify session cookie (no expiry)

Set subcommand (persistent cookies):
- [ ] cookies set remember_me yes --max-age 3600 (1 hour)
- [ ] cookies set auth_token xyz --max-age 86400 (24 hours)
- [ ] Verify cookie expiry set correctly
- [ ] Verify persistent cookie attributes

Set subcommand (secure cookies):
- [ ] cookies set session abc --secure
- [ ] cookies set session abc --httponly
- [ ] cookies set session abc --secure --httponly
- [ ] Verify Secure flag
- [ ] Verify HttpOnly flag

Set subcommand (domain and path):
- [ ] cookies set tracking id --domain .example.com
- [ ] cookies set api_key xyz --domain api.example.com
- [ ] cookies set csrf token --path /api
- [ ] cookies set data val --domain .example.com --path /admin
- [ ] Verify domain attribute
- [ ] Verify path attribute

Set subcommand (SameSite):
- [ ] cookies set csrf_token xyz --samesite Strict
- [ ] cookies set analytics id --samesite Lax
- [ ] cookies set tracking id --samesite None --secure
- [ ] Verify SameSite attribute
- [ ] SameSite None requires --secure

Set subcommand (full attributes):
- [ ] cookies set auth abc --domain example.com --path /api --secure --httponly --max-age 86400 --samesite Strict
- [ ] Verify all attributes set correctly

Set subcommand (errors):
- [ ] Set on HTTPS-only page without HTTPS
- [ ] Invalid SameSite value
- [ ] SameSite None without --secure
- [ ] Daemon not running

Delete subcommand (unambiguous):
- [ ] cookies delete session (single match)
- [ ] cookies delete auth_token (single match)
- [ ] Verify cookie deleted
- [ ] Delete non-existent cookie (success - idempotent)

Delete subcommand (ambiguous):
- [ ] cookies delete session (multiple matches - error)
- [ ] Verify error message lists matches
- [ ] cookies delete session --domain example.com (disambiguate)
- [ ] Verify correct cookie deleted

Delete subcommand (domain):
- [ ] cookies delete session --domain api.example.com
- [ ] cookies delete tracking --domain .example.com
- [ ] Verify domain-specific deletion

Delete subcommand (errors):
- [ ] Delete with ambiguous name (error with matches)
- [ ] Daemon not running

Combination tests (observation):
- [ ] --domain and --find together
- [ ] --name and --find together
- [ ] --domain and --name together
- [ ] All filters together

Output formats:
- [ ] Default JSON response (file path)
- [ ] Show mode text format (name, value, domain, path, expiry, flags)
- [ ] --json with show mode
- [ ] --raw output format
- [ ] Set/delete success response
- [ ] Delete error response with matches
- [ ] --no-color output
- [ ] --debug verbose output

Error cases:
- [ ] Find text not in cookies (no matches error)
- [ ] Save to invalid path
- [ ] Set with invalid attributes
- [ ] Delete ambiguous name without domain
- [ ] Daemon not running

CLI vs REPL:
- [ ] CLI: webctl cookies
- [ ] CLI: webctl cookies save
- [ ] CLI: webctl cookies save ./cookies.json
- [ ] CLI: webctl cookies --domain ".github.com"
- [ ] CLI: webctl cookies set session abc123
- [ ] CLI: webctl cookies set auth xyz --secure --httponly --max-age 3600
- [ ] CLI: webctl cookies delete session
- [ ] CLI: webctl cookies delete session --domain example.com
- [ ] REPL: cookies
- [ ] REPL: cookies save
- [ ] REPL: cookies save ./cookies.json
- [ ] REPL: cookies set session abc123
- [ ] REPL: cookies delete session

## Notes

- Default mode outputs to stdout (Unix convention)
- Save mode saves to temp or custom path
- Trailing slash convention: path/ = directory (auto-filename), path = file (like rsync)
- Set subcommand mutates browser state (creates/updates cookie)
- Delete subcommand mutates browser state (removes cookie)
- Domain filter matches exact domain and subdomains
- Name filter requires exact match (case sensitive)
- Find flag searches both names and values (case insensitive)
- Raw flag outputs JSON instead of formatted text
- Auto-generated filenames use timestamp with "cookies" identifier
- Saved files contain JSON with ok, cookies array, and count
- Session cookies (max-age 0) expire when browser closes
- Persistent cookies use max-age for expiry in seconds
- Secure flag requires HTTPS
- HttpOnly flag prevents JavaScript access via document.cookie
- SameSite controls cross-site request behavior (Strict, Lax, None)
- Delete is idempotent (success even if cookie doesn't exist)
- Ambiguous delete requires --domain to disambiguate

## Issues Discovered

(Issues will be documented here during testing)
