# P-037: Testing network Command

- Status: In Progress
- Started: 2025-12-31

## Overview

Test the webctl network command which extracts network requests from the current page. This command supports three modes (default/show/save) with extensive filtering capabilities including type, method, status, URL patterns, MIME types, duration, size, and failure state.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-network.sh
```

## Code References

- internal/cli/network.go
- internal/cli/format (network formatting)

## Command Signature

```
webctl network [show|save <path>] [--find text] [--type type] [--method method] [--status status] [--url pattern] [--mime mime] [--min-duration duration] [--min-size bytes] [--failed] [--head N] [--tail N] [--range N-M] [--max-body-size bytes] [--raw]
```

Subcommands:
- (default): Save to /tmp/webctl-network/ with auto-generated filename
- show: Output network requests to stdout
- save <path>: Save to custom path

Universal flags (work with default/show/save modes):
- --find, -f <text>: Search for text within URLs and bodies
- --raw: Skip formatting (return raw JSON)
- --json: Output in JSON format (global flag)

Network-specific filter flags:
- --type <TYPE>: CDP resource type (xhr, fetch, document, script, stylesheet, image, font, websocket, media, manifest, texttrack, eventsource, prefetch, other) - repeatable, CSV-supported
- --method <METHOD>: HTTP method (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS) - repeatable, CSV-supported
- --status <STATUS>: Status code or range (200, 4xx, 5xx, 200-299) - repeatable, CSV-supported
- --url <PATTERN>: URL regex pattern (Go regexp syntax)
- --mime <MIME>: MIME type (application/json, text/html, image/png) - repeatable, CSV-supported
- --min-duration <DURATION>: Minimum request duration (1s, 500ms, 100ms)
- --min-size <BYTES>: Minimum response size in bytes
- --failed: Show only failed requests (network errors, CORS, etc.)
- --max-body-size <BYTES>: Maximum body size before truncation (default 102400)
- --head <N>: Return first N entries
- --tail <N>: Return last N entries
- --range <N-M>: Return entries N through M (mutually exclusive with head/tail)

## Test Checklist

Default mode (save to temp):
- [ ] network (all requests to temp)
- [ ] network --status 4xx (only 4xx to temp)
- [ ] network --find "api" (search and save)
- [ ] Verify file saved to /tmp/webctl-network/
- [ ] Verify auto-generated filename format (YY-MM-DD-HHMMSS-network.json)
- [ ] Verify JSON response with file path
- [ ] Verify JSON file structure (ok, entries, count)

Show mode (stdout):
- [ ] network show (all requests to stdout)
- [ ] network show --status 4xx,5xx (only errors)
- [ ] network show --find "fetch" (search and show)
- [ ] network show --tail 20 (last 20 entries)
- [ ] Verify formatted text output to stdout
- [ ] Verify no file created

Save mode (custom path):
- [ ] network save ./logs/requests.json (save to file)
- [ ] network save ./output/ (save to dir with auto-filename)
- [ ] network save ./errors.json --status 5xx --tail 50
- [ ] Verify file saved to custom path

Type filter:
- [ ] --type xhr (XMLHttpRequest)
- [ ] --type fetch (Fetch API)
- [ ] --type document (HTML documents)
- [ ] --type script (JavaScript files)
- [ ] --type stylesheet (CSS files)
- [ ] --type image (images)
- [ ] --type font (fonts)
- [ ] --type websocket (WebSocket)
- [ ] --type media (video/audio)
- [ ] --type manifest (web manifests)
- [ ] --type other (other types)
- [ ] --type xhr,fetch (CSV format)
- [ ] --type xhr --type fetch (repeatable format)
- [ ] Case insensitivity of type filter

Method filter:
- [ ] --method GET
- [ ] --method POST
- [ ] --method PUT
- [ ] --method DELETE
- [ ] --method PATCH
- [ ] --method HEAD
- [ ] --method OPTIONS
- [ ] --method GET,POST (CSV format)
- [ ] --method GET --method POST (repeatable format)
- [ ] Case insensitivity of method filter

Status filter:
- [ ] --status 200 (exact match)
- [ ] --status 404 (exact match)
- [ ] --status 4xx (wildcard pattern)
- [ ] --status 5xx (wildcard pattern)
- [ ] --status 200-299 (range pattern)
- [ ] --status 400-499 (range pattern)
- [ ] --status 4xx,5xx (CSV format)
- [ ] --status 4xx --status 5xx (repeatable format)
- [ ] Invalid status pattern (error)

URL filter:
- [ ] --url "api" (simple pattern)
- [ ] --url "^https://example.com" (starts with)
- [ ] --url "\.json$" (ends with)
- [ ] --url "api/v[0-9]+/" (regex with groups)
- [ ] Invalid regex pattern (error)

MIME filter:
- [ ] --mime application/json
- [ ] --mime text/html
- [ ] --mime image/png
- [ ] --mime text/css
- [ ] --mime application/javascript
- [ ] --mime application/json,text/html (CSV format)
- [ ] --mime application/json --mime text/html (repeatable format)
- [ ] Case insensitivity of mime filter

Duration filter:
- [ ] --min-duration 1s (seconds)
- [ ] --min-duration 500ms (milliseconds)
- [ ] --min-duration 100ms
- [ ] --min-duration with no matches
- [ ] Invalid duration format (error)

Size filter:
- [ ] --min-size 1024 (1KB)
- [ ] --min-size 102400 (100KB)
- [ ] --min-size 1048576 (1MB)
- [ ] --min-size with no matches

Failed filter:
- [ ] --failed (only failed requests)
- [ ] --failed with other filters
- [ ] Verify network errors included
- [ ] Verify CORS errors included

Max body size:
- [ ] --max-body-size 1024 (1KB truncation)
- [ ] --max-body-size 0 (no bodies)
- [ ] Default 102400 (100KB)
- [ ] Verify BodyTruncated field set when truncated

Find flag:
- [ ] --find in URL
- [ ] --find in body
- [ ] --find case insensitive
- [ ] --find with no matches (error)
- [ ] --find combined with other filters

Head flag:
- [ ] --head 10 (first 10 entries)
- [ ] --head 1 (first entry)
- [ ] --head 100 when fewer entries exist
- [ ] --head with filters

Tail flag:
- [ ] --tail 20 (last 20 entries)
- [ ] --tail 1 (last entry)
- [ ] --tail 100 when fewer entries exist
- [ ] --tail with filters

Range flag:
- [ ] --range 10-20 (entries 10 through 20)
- [ ] --range 0-10 (first 10 entries)
- [ ] --range 100-200 when fewer entries exist
- [ ] --range with invalid format (error)
- [ ] --range START-END where START >= END
- [ ] --range with filters

Mutual exclusivity:
- [ ] --head and --tail together (error)
- [ ] --head and --range together (error)
- [ ] --tail and --range together (error)

Raw flag:
- [ ] --raw output (JSON format)
- [ ] Compare raw vs formatted output
- [ ] --raw with show mode
- [ ] --raw with filters

Combination tests:
- [ ] --type and --method together
- [ ] --status and --method together
- [ ] --type, --method, --status together
- [ ] --url and --find together
- [ ] --min-duration and --min-size together
- [ ] --failed with other filters
- [ ] Multiple filters creating AND logic
- [ ] Complex filter: --type xhr --method POST --status 4xx --find "error"

Output formats:
- [ ] Default JSON response (file path)
- [ ] Show mode text format (method, URL, status, duration)
- [ ] --json with show mode
- [ ] --raw output format
- [ ] --no-color output
- [ ] --debug verbose output

Error cases:
- [ ] Find text not in requests (no matches error)
- [ ] Save to invalid path
- [ ] Invalid range format
- [ ] Invalid status pattern
- [ ] Invalid URL regex
- [ ] Invalid duration format
- [ ] Mutually exclusive flags used together
- [ ] Daemon not running

CLI vs REPL:
- [ ] CLI: webctl network
- [ ] CLI: webctl network show
- [ ] CLI: webctl network save ./requests.json
- [ ] CLI: webctl network show --status 4xx --tail 10
- [ ] CLI: webctl network show --type xhr,fetch --method POST
- [ ] REPL: network
- [ ] REPL: network show
- [ ] REPL: network save ./requests.json
- [ ] REPL: network show --status 4xx --tail 10

## Notes

- Default mode saves to temp for quick debugging
- Show mode useful for real-time monitoring and piping
- All filters are AND-combined (must match all specified filters)
- StringSlice flags support both CSV and repeatable syntax
- Status filter supports exact match (200), wildcard (4xx), and range (200-299)
- URL filter uses Go regexp syntax for powerful pattern matching
- Min-duration accepts duration strings (1s, 500ms)
- Failed flag filters for network errors, CORS failures, timeouts
- Max-body-size truncates large response bodies in JSON output
- Head/tail/range flags mutually exclusive
- Raw flag outputs JSON instead of formatted text
- Auto-generated filenames use timestamp with "network" identifier
- Saved files contain JSON with ok, entries array, and count
- Network requests captured via Chrome DevTools Protocol

## Issues Discovered

(Issues will be documented here during testing)
