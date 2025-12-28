# Network Command Design - LOCKED

## Universal Pattern

```bash
# Default: save all requests to temp file
webctl network
# → /tmp/webctl-network/25-12-28-HHMMSS-network.json

# Show: output all requests to stdout
webctl network show

# Save: save all requests to custom path
webctl network save <path>
# If <path> is a directory, auto-generate filename
webctl network save ./output/
# → ./output/25-12-28-HHMMSS-network.json
```

## Universal Flags

```bash
--find, -f TEXT          # Search within URLs and response bodies
--raw                    # Skip formatting/pretty-printing
--json                   # JSON output (global flag)
```

## Network-Specific Flags

These filters are specific to network requests:

```bash
--type TYPE              # CDP resource type: xhr, fetch, document, script,
                         # stylesheet, image, font, websocket, media, manifest,
                         # texttrack, eventsource, prefetch, other
                         # Repeatable/CSV-supported
--method METHOD          # HTTP method: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS
                         # Repeatable/CSV-supported
--status CODE            # Status code or range: 200, 4xx, 5xx, 200-299
                         # Repeatable/CSV-supported
--url PATTERN            # URL regex pattern (Go regexp syntax)
--mime TYPE              # MIME type: application/json, text/html, image/png, etc.
                         # Repeatable/CSV-supported
--min-duration DURATION  # Minimum request duration: 1s, 500ms, 100ms
--min-size BYTES         # Minimum response size in bytes
--failed                 # Show only failed requests (network errors, CORS, etc.)
--head N                 # Return first N entries
--tail N                 # Return last N entries
--range N-M              # Return entries N through M
```

Note: `--head`, `--tail`, and `--range` are mutually exclusive.

## Examples

```bash
# All requests to temp
webctl network
# → /tmp/webctl-network/25-12-28-HHMMSS-network.json

# All requests to stdout
webctl network show

# Filter by status
webctl network show --status 4xx
webctl network show --status 4xx,5xx
webctl network show --status 200-299

# Filter by type
webctl network show --type xhr,fetch
webctl network show --type document

# Filter by method
webctl network show --method POST,PUT

# Search in URLs
webctl network show --url "api/user"
webctl network show --find "api/"

# Failed requests only
webctl network show --failed

# Slow requests
webctl network show --min-duration 1s

# Large responses
webctl network show --min-size 1048576  # 1MB+

# Combine filters
webctl network show --status 5xx --method POST --find "api/"
webctl network save ./api-errors.json --status 4xx,5xx --url "api/"

# Limit results
webctl network show --head 20
webctl network show --tail 50
webctl network show --range 10-30

# Complex filtering
webctl network save ./slow-api-errors.json \
  --url "api/" \
  --status 5xx \
  --min-duration 500ms \
  --tail 100
```

## Output Format

**Text mode:**
- Formatted table with method, status, URL, duration, size
- Color-coded by status (2xx green, 4xx yellow, 5xx red)

**JSON mode:**
- Array of network entry objects
- Each entry includes: requestId, url, method, status, type, mimeType,
  duration, size, headers, body, failed, errorText, etc.

## Network-Specific Subcommands

None. Network uses only the universal pattern with network-specific filter flags.

## Design Rationale

**Universal pattern:**
- Consistent with html, css, console, cookies
- Default saves to temp (preserves requests for analysis)
- `show` for interactive debugging
- `save <path>` for CI/CD or archival

**Network-specific flags:**
- Extensive filtering needed for large request volumes
- Status codes, HTTP methods, resource types are network-specific
- Performance filters (duration, size) unique to network requests
- All filters are AND-combined for precise targeting

**No specific subcommands:**
- Network doesn't need operations like CSS's `computed/get/inject`
- Filtering and output control covers all use cases
- Rich filtering via flags is more flexible than subcommands
