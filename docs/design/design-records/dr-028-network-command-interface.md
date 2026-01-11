# DR-028: Network Command Interface

- Date: 2025-12-28
- Status: Accepted
- Category: CLI

## Problem

The current network command only outputs to stdout, making it inconsistent with other observation commands that support multiple output modes. Current limitations:

- No option to save network requests to file for later analysis
- Cannot preserve request data for archival, CI/CD, or performance analysis
- Inconsistent with html, css, console commands that support file output
- Missing universal pattern (default/show/save) for output control
- No integrated text search for filtering requests by URL or content

Users need network requests to follow the universal observation pattern with file output options while maintaining extensive network-specific filtering capabilities.

## Decision

Redesign network command to follow the universal observation pattern with comprehensive network-specific filter flags:

```bash
# Universal pattern
webctl network              # Output all requests to stdout (Unix convention)
webctl network save [path]  # Save to file (temp if no path, custom if path given)

# Universal flags
--find, -f TEXT             # Search within URLs and response bodies
--raw                       # Skip formatting
--json                      # JSON output

# Network-specific filter flags
--type TYPE                 # CDP resource type (xhr, fetch, document, script, etc.)
--method METHOD             # HTTP method (GET, POST, PUT, DELETE, etc.)
--status CODE               # Status code or range (200, 4xx, 5xx, 200-299)
--url PATTERN               # URL regex pattern
--mime TYPE                 # MIME type (application/json, text/html, etc.)
--min-duration DURATION     # Minimum request duration (1s, 500ms)
--min-size BYTES            # Minimum response size in bytes
--failed                    # Only failed requests (network errors, CORS)
--head N / --tail N / --range N-M   # Limit results
```

The network command outputs to stdout by default (Unix convention), with a save subcommand for file output. All filtering is provided through network-specific flags that apply to all output modes.

Complete specification: docs/design/interface/network.md

## Why

Unix Convention (stdout by default):

Following Unix philosophy, observation commands output to stdout by default. This enables:
- Piping to other tools (grep, less, jq)
- Quick inspection without file management
- Consistent with standard CLI tools

Save Subcommand for Files:

When file output is needed, the save subcommand provides flexibility:
- `network save` - saves to temp directory with auto-generated filename
- `network save ./requests.json` - saves to custom path (file)
- `network save ./output/` - saves to directory with auto-generated filename (trailing slash required)
- Trailing slash convention (like rsync): path with `/` suffix is directory, without is file

Extensive Network-Specific Filters:

Network requests require rich filtering capabilities due to high volume and diverse data:

- --type: Filter by resource type (xhr, fetch, document, script, etc.)
- --method: Filter by HTTP method (GET, POST, etc.)
- --status: Filter by status code or range (200, 4xx, 5xx)
- --url: Filter by URL pattern (regex matching)
- --mime: Filter by response MIME type
- --min-duration: Filter slow requests (performance analysis)
- --min-size: Filter large responses (performance analysis)
- --failed: Filter failed requests (network errors, CORS)

These filters are network-specific and don't apply to other observation commands. They enable precise targeting of requests for debugging and analysis.

No Network-Specific Subcommands:

Network monitoring doesn't require special operations like CSS does (computed/get/inject) or cookies does (set/delete). All network functionality is observation and filtering, which maps perfectly to the universal pattern with filter flags.

Text Search Integration:

The --find flag enables searching within URLs and response bodies, matching the pattern for other observation commands. Users can filter requests by content, useful for finding specific API calls or responses.

AND-Combining Filters:

All filter flags are AND-combined, allowing precise targeting:
```bash
webctl network show --status 5xx --method POST --url "api/" --min-duration 1s
```

This filters to: failed POST requests to API endpoints that took over 1 second.

## Trade-offs

Accept:

- Breaking change from current stdout-only behavior
- Default to file may surprise users expecting stdout
- Temp files require eventual cleanup
- Many filter flags increase learning curve
- More complex command structure with subcommands
- Users must learn new pattern for familiar command

Gain:

- Consistent interface across all observation commands
- Request preservation for debugging and analysis
- Flexible output modes for different use cases
- File output for CI/CD, performance analysis, security auditing
- Rich filtering for precise request targeting
- Integrated text search for request filtering
- Filter flags work across all output modes
- Foundation matches other observation commands
- Predictable behavior pattern (learn once, use everywhere)

## Alternatives

Keep Current Stdout-Only Behavior:

```bash
webctl network    # Always stdout
```

- Pro: No breaking changes, existing scripts work
- Pro: Simple single behavior
- Pro: Matches current user expectations
- Con: No way to save requests to file
- Con: Inconsistent with html/css/console commands
- Con: Cannot preserve requests for later analysis
- Rejected: Fails to provide file output capability and consistency

Add File Output Flag:

```bash
webctl network               # Stdout (current behavior)
webctl network -o <path>     # Save to file (new option)
```

- Pro: Minimal breaking change
- Pro: Adds file capability
- Con: Inconsistent with universal pattern
- Con: Stdout-first doesn't match other observation commands
- Con: Doesn't establish predictable pattern
- Rejected: Partial solution that doesn't achieve consistency

Network-Specific Subcommands for Filters:

```bash
webctl network errors        # Show failed requests
webctl network slow          # Show slow requests
webctl network api           # Show API requests
```

- Pro: Named filters are discoverable
- Pro: Simple for common cases
- Con: Limited to predefined filters
- Con: Flags provide more flexibility
- Con: Subcommands mix with output modes
- Rejected: Flags allow arbitrary combinations

Separate Commands for Output Modes:

```bash
webctl network-show          # Stdout
webctl network-save <path>   # File output
```

- Pro: Very explicit
- Pro: No subcommand complexity
- Con: Clutters command namespace
- Con: Two commands instead of one
- Con: Less discoverable
- Rejected: Subcommands group functionality better

Simplify Filter Flags:

```bash
# Only basic filters
--status CODE
--method METHOD
--find TEXT
```

- Pro: Fewer flags to learn
- Pro: Simpler interface
- Con: Insufficient for real-world debugging
- Con: Missing performance filters (duration, size)
- Con: Missing resource type filtering
- Rejected: Network debugging requires rich filtering

## Structure

Output Modes:

Default (no subcommand):
- Outputs network requests to stdout (Unix convention)
- Formatted table with method, status, URL, duration, size
- Color-coded by status (2xx green, 4xx yellow, 5xx red)
- Useful for piping to other tools

Save subcommand:
- Optional path argument
- No path: saves to /tmp/webctl-network/ with auto-generated filename
- Path with trailing slash (path/): auto-generates filename in that directory
- Path without trailing slash (path): saves to exact file path
- Creates parent directories if needed
- Trailing slash convention follows Unix tools like rsync

Universal Flags:

--find, -f TEXT:
- Search for text within URLs and response bodies
- Filters requests containing search text
- Works across all output modes
- Case-insensitive search

--raw:
- Skips formatting/pretty-printing
- Returns requests in raw format
- Useful for machine processing

--json:
- Global flag for JSON output format
- Array of network entry objects
- Each entry: requestId, url, method, status, type, mimeType, duration, size, headers, body, failed, errorText

Network-Specific Filter Flags:

--type TYPE:
- CDP resource type: xhr, fetch, document, script, stylesheet, image, font, websocket, media, manifest, texttrack, eventsource, prefetch, other
- Repeatable: --type xhr --type fetch
- CSV-supported: --type xhr,fetch
- Multiple values are OR-combined

--method METHOD:
- HTTP method: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS
- Repeatable: --method POST --method PUT
- CSV-supported: --method POST,PUT
- Multiple values are OR-combined

--status CODE:
- Status code or range: 200, 4xx, 5xx, 200-299
- Repeatable: --status 4xx --status 5xx
- CSV-supported: --status 4xx,5xx
- Multiple values are OR-combined
- Ranges: 200-299, 400-499, 500-599, or shorthand 2xx, 4xx, 5xx

--url PATTERN:
- URL regex pattern (Go regexp syntax)
- Filters requests matching pattern
- Example: --url "api/user"

--mime TYPE:
- Response MIME type: application/json, text/html, image/png, etc.
- Repeatable: --mime application/json --mime text/html
- CSV-supported: --mime application/json,text/html
- Multiple values are OR-combined

--min-duration DURATION:
- Minimum request duration: 1s, 500ms, 100ms
- Filters requests taking longer than duration
- Performance analysis

--min-size BYTES:
- Minimum response size in bytes
- Filters responses larger than size
- Performance analysis

--failed:
- Only failed requests (network errors, CORS, timeouts)
- Boolean flag
- Debugging filter

--head N:
- Return first N requests
- Applied after other filters
- Mutually exclusive with --tail and --range

--tail N:
- Return last N requests
- Applied after other filters
- Mutually exclusive with --head and --range
- Most common for recent requests

--range N-M:
- Return requests N through M (inclusive)
- Applied after other filters
- Mutually exclusive with --head and --tail

All filters are AND-combined for precise targeting.

## Usage Examples

Default behavior (stdout):

```bash
webctl network
# GET  | 200 | https://example.com/api/user | 250ms | 1.2KB
# POST | 201 | https://example.com/api/data | 150ms | 500B
# GET  | 404 | https://example.com/missing  | 50ms  | 200B

webctl network --status 4xx,5xx
# GET | 404 | https://example.com/missing | 50ms | 200B
# GET | 500 | https://example.com/error   | 1.2s | 150B

webctl network --type xhr,fetch
# (only AJAX requests)
```

Save to file:

```bash
webctl network save
# {"ok": true, "path": "/tmp/webctl-network/25-12-28-143052-network.json"}

webctl network save ./requests.json
# {"ok": true, "path": "./requests.json"}

webctl network save ./output/
# {"ok": true, "path": "./output/25-12-28-143052-network.json"}

webctl network save ./api-errors.json --status 5xx --url "api/"
# {"ok": true, "path": "./api-errors.json"}
```

Network-specific filters:

```bash
# Status filtering
webctl network --status 200
webctl network --status 4xx
webctl network --status 200-299
webctl network --status 4xx,5xx

# Type filtering
webctl network --type xhr,fetch
webctl network --type document
webctl network --type script,stylesheet

# Method filtering
webctl network --method POST
webctl network --method POST,PUT,DELETE

# URL filtering
webctl network --url "api/user"
webctl network --url "^https://example.com"

# MIME type filtering
webctl network --mime application/json
webctl network --mime image/png,image/jpeg

# Performance filtering
webctl network --min-duration 1s
webctl network --min-size 1048576  # 1MB+
webctl network --min-duration 500ms --min-size 500000

# Failed requests
webctl network --failed

# Search within requests
webctl network --find "api/"
webctl network --find "error"

# Limit results
webctl network --head 20
webctl network --tail 50
webctl network --range 10-30
```

Complex filtering (AND-combined):

```bash
# Slow API errors
webctl network --url "api/" --status 5xx --min-duration 500ms

# Large image requests
webctl network --type image --min-size 500000

# Failed POST requests to API
webctl network --method POST --url "api/" --failed

# Recent API errors
webctl network save ./recent-api-errors.json \
  --url "api/" \
  --status 4xx,5xx \
  --tail 100
```

JSON output:

```bash
webctl network --json
# {
#   "ok": true,
#   "requests": [
#     {
#       "requestId": "1234.5",
#       "url": "https://example.com/api/user",
#       "method": "GET",
#       "status": 200,
#       "type": "xhr",
#       "mimeType": "application/json",
#       "duration": 250,
#       "size": 1200,
#       "headers": {...},
#       "body": "{...}",
#       "failed": false
#     },
#     ...
#   ]
# }
```

## File Naming

Auto-generated Filenames:

Pattern: /tmp/webctl-network/YY-MM-DD-HHMMSS-network.json

Default extension: .json (requests are structured data)

Example filenames:
- 25-12-28-143052-network.json
- 25-12-28-143115-network.json
- 25-12-28-143120-network.json

Identifier: Fixed to "network" (no variation needed)

## Output Format

Text Mode (default):

Formatted table with color-coding:
```
GET  | 200 | https://example.com/api/user | 250ms | 1.2KB
POST | 201 | https://example.com/api/data | 150ms | 500B
GET  | 404 | https://example.com/missing  | 50ms  | 200B
GET  | 500 | https://example.com/error    | 1.2s  | 150B
```

Color scheme:
- 2xx: green
- 3xx: blue
- 4xx: yellow
- 5xx: red
- failed: red + bold

Use --raw to disable formatting and colors.

JSON Mode (--json flag):

Array of network entry objects:
```json
{
  "ok": true,
  "requests": [
    {
      "requestId": "1234.5",
      "url": "https://example.com/api/user",
      "method": "GET",
      "status": 200,
      "type": "xhr",
      "mimeType": "application/json",
      "duration": 250,
      "size": 1200,
      "headers": {
        "content-type": "application/json",
        "cache-control": "no-cache"
      },
      "body": "{\"user\": \"example\"}",
      "failed": false,
      "errorText": null
    }
  ]
}
```

## Breaking Changes

From DR-009 (Network Command Interface):

1. Changed: Default behavior now outputs to stdout (Unix convention)
2. Removed: show subcommand (not needed - stdout is default)
3. Changed: save subcommand now takes optional path (temp if no path)
4. Added: --find flag for text search within URLs and bodies
5. Retained: All filter flags (--type, --method, --status, --url, --mime, --min-duration, --min-size, --failed, --head, --tail, --range)
6. Retained: Color-coded output for default mode

Migration Guide:

Old pattern (DR-009):
```bash
webctl network                       # Stdout with all requests
webctl network --status 5xx          # Stdout with errors only
webctl network --type xhr --tail 20  # Recent AJAX requests
```

New pattern (DR-028 after P-051):
```bash
webctl network                       # Output to stdout (same as before)
webctl network --status 5xx          # Stdout with errors only (same)
webctl network --type xhr --tail 20  # Recent AJAX requests (same)
webctl network save                  # Save to temp (new)
webctl network save ./requests.json  # Save to custom path (new feature)
```

The default stdout behavior is preserved. Use `webctl network save` when file output is needed.

## Updates

- 2026-01-09: Updated to stdout default, removed show subcommand (P-051)
- 2025-12-28: Initial version (supersedes DR-009)
