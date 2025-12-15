# DR-009: Network Command Interface

- Date: 2025-12-15
- Status: Accepted
- Category: CLI

## Problem

AI agents debugging web applications need access to network requests and responses: API calls, resource loading, failed requests, and response bodies. The CDP events (Network.requestWillBeSent, Network.responseReceived, Network.loadingFinished, Network.loadingFailed) are being buffered by the daemon, but agents need a CLI command to query these buffered entries with filtering and formatting options.

Requirements:

- Query all buffered network entries
- Filter by resource type, HTTP method, status code, URL pattern, MIME type
- Filter by performance characteristics (slow requests, large responses)
- Filter for failed requests (network errors, CORS, etc.)
- Include request and response headers
- Include response bodies (with size limits for large responses)
- Save binary bodies to disk (images, fonts, etc.)
- Output in both machine-readable (JSON) and human-readable (text) formats

## Decision

Implement `webctl network` command with the following interface:

```bash
webctl network [flags]
```

Base Flags (matching console command):

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --format | string | auto | Output format: json, text, or auto (TTY=pretty JSON, pipe=raw JSON) |
| --head | int | 0 | Return first N entries (0=unlimited) |
| --tail | int | 0 | Return last N entries (0=unlimited) |
| --range | string | - | Return entries in range (format: START-END, 0-indexed) |

Note: --head, --tail, and --range are mutually exclusive.

Network-Specific Filters:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --type | []string | - | Filter by CDP resource type (xhr, fetch, document, script, stylesheet, image, font, websocket, etc.) |
| --method | []string | - | Filter by HTTP method (GET, POST, PUT, DELETE, PATCH, etc.) |
| --status | []string | - | Filter by status code or range (200, 4xx, 5xx, 200-299) |
| --url | string | - | Filter by URL regex pattern |
| --mime | []string | - | Filter by MIME type (application/json, text/html, etc.) |
| --min-duration | duration | - | Filter by minimum request duration (e.g., 1s, 500ms) |
| --min-size | int | - | Filter by minimum response size in bytes |
| --failed | bool | false | Show only failed requests |

Body Control:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --max-body-size | int | 102400 | Maximum body size in bytes before truncation (default 100KB) |

Filter Behavior:

- []string flags support CSV (--status 4xx,5xx) and repeatable (--status 4xx --status 5xx) syntax
- All filters are AND-combined (entry must match all specified filters)
- Type, method, and MIME matching is case-insensitive
- Status ranges use patterns: 2xx (200-299), 3xx (300-399), 4xx (400-499), 5xx (500-599)
- URL filter uses Go regexp syntax

## Why

Consistent interface with console command:

The base flags (--format, --head, --tail, --range) match the console command exactly. Users learn one pattern and apply it across observation commands. Auto-detecting output format provides the best default for both agents (raw JSON to pipes) and humans (pretty JSON to TTY).

Comprehensive filtering:

Network buffers can contain thousands of entries. Filters let agents focus on relevant requests without parsing the full buffer. Common debugging scenarios:

- `--type xhr,fetch` - API calls only
- `--status 4xx,5xx` - Errors only
- `--failed` - Network failures (CORS, connection refused)
- `--min-duration 1s` - Slow requests
- `--url "api/users"` - Specific endpoint

Bodies included by default:

Response bodies are critical for debugging API issues. Agents typically want to see what the server returned. The --max-body-size limit prevents memory issues with large responses while still capturing most API responses in full.

Binary bodies saved to disk:

Binary content (images, fonts, etc.) cannot be represented in JSON. Saving to disk with a path reference allows debugging asset loading issues while keeping JSON output clean.

Separate request and response headers:

Both are valuable for debugging. Request headers show what was sent (auth tokens, content-type). Response headers show server behavior (caching, CORS, content-type actually served).

## Trade-offs

Accept:

- Complex flag interface (12 flags total)
- Body storage uses disk space
- Large JSON output for full buffer with bodies
- Binary file cleanup is manual (webctl clear network)
- Regex learning curve for URL filtering

Gain:

- Flexible querying for all debugging scenarios
- Complete request/response data including bodies
- Binary assets accessible for inspection
- Consistent interface across observation commands
- Self-documenting intent (flags express query purpose)

## Alternatives

Exclude bodies by default:

```bash
webctl network              # No bodies
webctl network --body       # Include bodies
```

- Pro: Smaller default output
- Con: Bodies are usually what agents need for debugging
- Con: Extra flag for common case
- Rejected: Include by default, agents typically need bodies

Simple substring match for URL:

- Pro: Simpler, no regex knowledge required
- Con: Can't express complex patterns (e.g., /api/v[12]/)
- Con: Simple substrings still work as regex
- Rejected: Regex provides more power, simple cases still work

Base64 encode binary bodies:

- Pro: Everything in JSON output
- Con: Bloated output (33% overhead)
- Con: Agents rarely need raw binary bytes
- Rejected: File path reference is cleaner

Time-based filtering (--since 5m):

- Pro: Query by time window
- Con: --clear flag on action commands provides workflow-level isolation
- Con: Adds complexity for marginal benefit
- Rejected: Not needed for v1, --clear workflow is sufficient

## Structure

NetworkEntry Schema:

```json
{
  "requestId": "123.45",
  "url": "https://api.example.com/users",
  "method": "POST",
  "type": "fetch",
  "status": 201,
  "statusText": "Created",
  "mimeType": "application/json",
  "requestTime": 1734151712450,
  "responseTime": 1734151712684,
  "duration": 0.234,
  "size": 1523,
  "requestHeaders": {
    "Authorization": "Bearer token...",
    "Content-Type": "application/json"
  },
  "responseHeaders": {
    "Content-Type": "application/json",
    "Cache-Control": "no-cache"
  },
  "body": "{\"id\": 42, \"name\": \"Alice\"}",
  "bodyTruncated": false,
  "failed": false,
  "error": ""
}
```

Binary Body Entry (body saved to disk):

```json
{
  "requestId": "456.78",
  "url": "https://example.com/logo.png",
  "method": "GET",
  "type": "image",
  "status": 200,
  "mimeType": "image/png",
  "size": 45678,
  "bodyPath": "~/.local/state/webctl/bodies/2025-12-15-143045-456.78-logo.png",
  "failed": false
}
```

Failed Request Entry:

```json
{
  "requestId": "789.01",
  "url": "https://api.blocked.com/data",
  "method": "GET",
  "type": "xhr",
  "status": 0,
  "requestTime": 1734151712450,
  "responseTime": 1734151713684,
  "duration": 1.234,
  "failed": true,
  "error": "net::ERR_CONNECTION_REFUSED"
}
```

Field Descriptions:

requestId (string, required):
- CDP request identifier
- Unique within a session

url (string, required):
- Full request URL

method (string, required):
- HTTP method (GET, POST, PUT, DELETE, etc.)

type (string, optional):
- CDP resource type: document, stylesheet, image, media, font, script, texttrack, xhr, fetch, prefetch, eventsource, websocket, manifest, other

status (int, optional):
- HTTP status code
- 0 for failed requests that never received a response

statusText (string, optional):
- HTTP status text (OK, Created, Not Found, etc.)

mimeType (string, optional):
- Response MIME type

requestTime (int64, required):
- Unix timestamp in milliseconds when request was sent

responseTime (int64, optional):
- Unix timestamp in milliseconds when response headers were received
- Absent for failed requests

duration (float64, optional):
- Request duration in seconds
- Time from request sent to response headers received (or failure)

size (int64, optional):
- Response body size in bytes

requestHeaders (map[string]string, optional):
- Request headers sent to server

responseHeaders (map[string]string, optional):
- Response headers received from server

body (string, optional):
- Response body as string (for text content)
- Omitted for binary content (use bodyPath)

bodyTruncated (bool, optional):
- True if body exceeded --max-body-size and was truncated

bodyPath (string, optional):
- Path to saved binary body file
- Present instead of body for binary content types

failed (bool, required):
- True if request failed (network error, CORS, timeout, etc.)

error (string, optional):
- Error description for failed requests (net::ERR_*, CORS error text)

## Usage Examples

Basic query (get all entries):

```bash
webctl network
```

Filter by type:

```bash
webctl network --type xhr                     # Only XHR
webctl network --type xhr,fetch               # XHR and fetch (CSV)
webctl network --type xhr --type fetch        # XHR and fetch (repeatable)
```

Filter by status:

```bash
webctl network --status 200                   # Only 200
webctl network --status 4xx                   # All 4xx errors
webctl network --status 4xx,5xx               # All errors
webctl network --status 200-299               # Success range
```

Filter by URL pattern:

```bash
webctl network --url "api/users"              # Contains api/users
webctl network --url "^https://api\."         # Starts with https://api.
webctl network --url "/v[12]/users"           # v1 or v2 users endpoint
```

Find slow requests:

```bash
webctl network --min-duration 1s              # Requests taking > 1 second
webctl network --min-duration 500ms           # Requests taking > 500ms
```

Find large responses:

```bash
webctl network --min-size 1048576             # Responses > 1MB
```

Find failed requests:

```bash
webctl network --failed                       # All failures
webctl network --failed --type fetch          # Failed API calls
```

Limit results:

```bash
webctl network --tail 50                      # Last 50 entries
webctl network --head 100                     # First 100 entries
webctl network --range 100-199                # Entries 100-199
```

Format control:

```bash
webctl network --format json                  # Force JSON to terminal
webctl network --format text                  # Force text to pipe
webctl network | jq '.entries[].url'          # Pipe gets raw JSON
```

Combined filters:

```bash
webctl network --type fetch --status 5xx --min-duration 500ms
# Failed API calls that were also slow

webctl network --mime application/json --status 200
# Successful JSON responses
```

Debug workflow with --clear (action commands):

```bash
webctl navigate https://example.com --clear
webctl network --type fetch                   # Only API calls from this page

webctl click "#submit" --clear=network
webctl network                                # Only requests from the click
```

Text Output Format:

```
[2025-12-15 14:30:45.450] GET 200 234ms application/json https://api.example.com/users
[2025-12-15 14:30:45.789] POST 201 567ms application/json https://api.example.com/users
[2025-12-15 14:30:46.012] GET 404 89ms text/html https://example.com/missing
[2025-12-15 14:30:46.456] GET ERR 1234ms - https://api.blocked.com/data (net::ERR_CONNECTION_REFUSED)
```

## Validation

Flag validation:

- --head, --tail, and --range are mutually exclusive
- --range format must be START-END where START < END
- --status patterns must be valid (NNN, Nxx, NNN-NNN)
- --url must be valid Go regexp
- --min-duration must be valid Go duration string
- --min-size must be non-negative integer
- --max-body-size must be positive integer

## Error Cases

Daemon not running:

```json
{"ok": false, "error": "daemon not running: connection refused"}
```

CLI message: "Error: daemon not running. Start with: webctl start"

Empty buffer:

```json
{"ok": true, "entries": [], "count": 0}
```

Text output: (no output, exit 0)

Invalid range format:

```bash
webctl network --range abc
```

Error: "invalid range format: use START-END (e.g., 100-200)"

Invalid regex:

```bash
webctl network --url "[invalid"
```

Error: "invalid URL pattern: error parsing regexp: ..."

Mutually exclusive flags:

```bash
webctl network --head 50 --tail 50
```

Error: "--head, --tail, and --range are mutually exclusive"

## Binary Body Handling

Binary content types (bodies saved to disk):

- image/* (image/png, image/jpeg, image/gif, image/webp, image/svg+xml, etc.)
- font/* (font/woff, font/woff2, font/ttf, etc.)
- audio/* (audio/mpeg, audio/ogg, etc.)
- video/* (video/mp4, video/webm, etc.)
- application/octet-stream
- application/pdf
- application/zip

Body file location:

- Directory: $XDG_STATE_HOME/webctl/bodies/ (default: ~/.local/state/webctl/bodies/)
- Filename: YYYY-MM-DD-HHMMSS-<requestId>-<basename>.<ext>
- Extension derived from MIME type

Cleanup:

- `webctl clear network` deletes all body files
- `webctl clear` (no argument) deletes all body files

## Testing Strategy

Unit tests:

- Flag parsing and validation
- Mutual exclusivity checking (--head/--tail/--range)
- Status pattern parsing (200, 4xx, 200-299)
- URL regex compilation and matching
- Duration parsing
- Output format detection (TTY vs pipe)
- Text formatting (timestamp, method, status, duration, URL)
- JSON formatting (pretty vs raw)
- All filter logic (type, method, status, url, mime, min-duration, min-size, failed)
- Filter AND-combination

Integration tests:

- Start daemon, generate network requests, query with webctl network
- Verify all entry fields populated correctly
- Verify filtering by each criterion works
- Verify --head/--tail/--range limiting works
- Verify body capture and truncation
- Verify binary body file saving
- Verify failed request capture
- Verify clear command deletes body files
- Verify empty buffer returns count: 0
- Verify error when daemon not running
