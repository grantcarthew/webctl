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
webctl network              # Save all requests to temp
webctl network show         # Output all requests to stdout
webctl network save <path>  # Save all requests to custom path

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

The network command uses the universal pattern with no network-specific subcommands. All filtering is provided through network-specific flags that apply to all output modes.

Complete specification: docs/design/interface/network.md

## Why

Universal Pattern Adoption:

Applying the default/show/save pattern to network requests provides consistent behavior across all observation commands. Users get predictable output mode control and file preservation capabilities for request/response data.

Default to Temp File:

Saving network requests to temp by default preserves debugging data for later analysis. Network request data is often needed for troubleshooting API issues, performance analysis, and security auditing. Automatic preservation prevents data loss.

Show Subcommand for Interactive Debugging:

Explicit show subcommand outputs requests to stdout for real-time monitoring and piping to analysis tools. This matches the current network command behavior while making the intent explicit.

Save Subcommand for Analysis:

The save subcommand enables saving requests to specific locations for performance analysis, CI/CD validation, security auditing, or long-term analysis. This fills a critical gap in the current implementation.

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
- Saves all network requests to /tmp/webctl-network/
- Auto-generates filename: YY-MM-DD-HHMMSS-network.json
- Returns JSON with file path
- Formatted text or JSON based on --json flag

Show subcommand:
- Outputs network requests to stdout
- Formatted table with method, status, URL, duration, size
- Color-coded by status (2xx green, 4xx yellow, 5xx red)
- Current behavior users expect

Save subcommand:
- Requires path argument
- Directory: auto-generates filename
- File: saves to exact path
- Creates parent directories if needed

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

Default behavior (save to temp):

```bash
webctl network
# {"ok": true, "path": "/tmp/webctl-network/25-12-28-143052-network.json"}

webctl network --status 4xx,5xx
# {"ok": true, "path": "/tmp/webctl-network/25-12-28-143115-network.json"}
# (only error responses)
```

Show to stdout:

```bash
webctl network show
# GET  | 200 | https://example.com/api/user | 250ms | 1.2KB
# POST | 201 | https://example.com/api/data | 150ms | 500B
# GET  | 404 | https://example.com/missing  | 50ms  | 200B

webctl network show --status 4xx,5xx
# GET | 404 | https://example.com/missing | 50ms | 200B
# GET | 500 | https://example.com/error   | 1.2s | 150B

webctl network show --type xhr,fetch
# (only AJAX requests)
```

Save to custom path:

```bash
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
webctl network show --status 200
webctl network show --status 4xx
webctl network show --status 200-299
webctl network show --status 4xx,5xx

# Type filtering
webctl network show --type xhr,fetch
webctl network show --type document
webctl network show --type script,stylesheet

# Method filtering
webctl network show --method POST
webctl network show --method POST,PUT,DELETE

# URL filtering
webctl network show --url "api/user"
webctl network show --url "^https://example.com"

# MIME type filtering
webctl network show --mime application/json
webctl network show --mime image/png,image/jpeg

# Performance filtering
webctl network show --min-duration 1s
webctl network show --min-size 1048576  # 1MB+
webctl network show --min-duration 500ms --min-size 500000

# Failed requests
webctl network show --failed

# Search within requests
webctl network show --find "api/"
webctl network show --find "error"

# Limit results
webctl network show --head 20
webctl network show --tail 50
webctl network show --range 10-30
```

Complex filtering (AND-combined):

```bash
# Slow API errors
webctl network show --url "api/" --status 5xx --min-duration 500ms

# Large image requests
webctl network show --type image --min-size 500000

# Failed POST requests to API
webctl network show --method POST --url "api/" --failed

# Recent API errors
webctl network save ./recent-api-errors.json \
  --url "api/" \
  --status 4xx,5xx \
  --tail 100
```

JSON output:

```bash
webctl network show --json
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

Text Mode (default for show):

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

1. Changed: Default behavior now saves to temp instead of stdout
2. Added: show subcommand for explicit stdout output (matches old default)
3. Added: save subcommand for custom path specification
4. Added: --find flag for text search within URLs and bodies
5. Added: Default output to JSON file format
6. Retained: All filter flags (--type, --method, --status, --url, --mime, --min-duration, --min-size, --failed, --head, --tail, --range)
7. Retained: Color-coded output for show mode

Migration Guide:

Old pattern (DR-009):
```bash
webctl network                       # Stdout with all requests
webctl network --status 5xx          # Stdout with errors only
webctl network --type xhr --tail 20  # Recent AJAX requests
```

New pattern (DR-028):
```bash
webctl network show                       # Stdout with all requests (changed)
webctl network show --status 5xx          # Stdout with errors only (changed)
webctl network show --type xhr --tail 20  # Recent AJAX requests (changed)
webctl network                            # Save to temp (new behavior)
webctl network save ./requests.json       # Save to custom path (new feature)
```

For users who want the old default behavior (stdout), update scripts to use `webctl network show`.

## Updates

- 2025-12-28: Initial version (supersedes DR-009)
