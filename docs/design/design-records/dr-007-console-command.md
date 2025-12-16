# DR-007: Console Command Interface

- Date: 2025-12-14
- Status: Accepted
- Category: CLI

## Problem

AI agents debugging web applications need access to browser console logs, warnings, errors, and uncaught exceptions. The CDP events (Runtime.consoleAPICalled, Runtime.exceptionThrown) are already being buffered by the daemon (see DR-003), but agents need a CLI command to query these buffered entries.

Requirements:

- Query all buffered console entries
- Filter by entry type (log, warn, error, etc.)
- Limit or paginate results for large buffers
- Output in both machine-readable (JSON) and human-readable (text) formats
- Support the common debugging workflow of clearing before actions (see DR-006)

## Decision

Implement `webctl console` command with the following interface:

```bash
webctl console [flags]
```

Flags:

| Flag | Type | Description |
|------|------|-------------|
| --format | string | Output format: json or text (auto-detect by default) |
| --type | []string | Filter by entry type (repeatable, CSV-supported) |
| --head | int | Return first N entries |
| --tail | int | Return last N entries |
| --range | string | Return entries in range (format: START-END) |

Note: --head, --tail, and --range are mutually exclusive.

Output formats:

JSON (default to terminal, raw to pipes):

```json
{
  "ok": true,
  "entries": [
    {
      "sessionId": "9A3E8D71...",
      "type": "error",
      "text": "Uncaught TypeError: Cannot read property 'foo'",
      "args": ["Uncaught TypeError: Cannot read property 'foo'"],
      "timestamp": 1734151712450,
      "url": "https://example.com/app.js",
      "line": 123,
      "column": 45
    }
  ],
  "count": 1
}
```

Text (human-readable):

```
[2025-12-14 16:08:32.450] app.js:123 Uncaught TypeError: Cannot read property 'foo'
[2025-12-14 16:08:32.789] main.js:456 User clicked submit button
```

## Why

Auto-detecting output format:

Agents typically consume JSON, but humans debugging via terminal benefit from readable text. Auto-detection (pretty JSON to TTY, raw JSON to pipes) provides the best default experience for both use cases while allowing explicit override via --format.

Multiple filtering and limiting options:

Different debugging scenarios require different query patterns:

- `--type error` - "Show me only errors"
- `--tail 50` - "What happened recently?"
- `--head 1000` - "What happened at page load?"
- `--range 2000-3000` - "Let me page through a large buffer to find the issue"

Supporting all patterns makes the command flexible for various debugging workflows.

Type filtering with StringSlice:

Cobra's StringSlice supports both CSV (`--type error,warn`) and repeatable (`--type error --type warn`) syntax. This provides maximum flexibility and follows the principle of least surprise - users can use whichever syntax feels natural.

Timestamps:

- JSON: Unix milliseconds matches CDP format, easy for agents to work with programmatically
- Text: ISO-style timestamp in local timezone is human-readable and matches system time

Chronological sort order:

Matches browser DevTools console behavior. Oldest entries first allows agents to see the sequence of events as they occurred.

## Trade-offs

Accept:

- Complex flag interface (5 flags with mutual exclusivity rules)
- Auto-detection adds magic (though predictable)
- Text format requires timestamp conversion from Unix milliseconds
- Large JSON output for full buffer (up to 10,000 entries)

Gain:

- Flexible querying for different debugging scenarios
- Smart defaults (auto-format, no limits)
- Works well for both agents (JSON) and humans (text)
- No hidden behavior (default returns all entries)
- Pagination support for large buffers

## Alternatives

Default to limited entries:

```bash
webctl console          # Return last 100 entries by default
webctl console --all    # Return everything
```

- Pro: Prevents overwhelming output
- Con: Hidden behavior (agents might not realize entries are missing)
- Con: Common workflow uses --clear, so buffers are usually small
- Rejected: Explicit is better than implicit

Add time-based filtering:

```bash
webctl console --since 5m
webctl console --after 1702000000
```

- Pro: Time-based queries possible
- Con: --clear provides workflow-level time isolation
- Con: Adds complexity for marginal benefit
- Rejected: Not needed for v1, can add later if use cases emerge

Separate flags for each type:

```bash
webctl console --errors --warnings
```

- Pro: More explicit than --type
- Con: More flags to maintain
- Con: Less flexible (can't easily add new types)
- Rejected: --type with StringSlice is more flexible

Always output raw JSON:

```bash
webctl console              # Raw JSON always
webctl console | jq '.'     # User pipes through jq for pretty
```

- Pro: Simpler implementation
- Con: Poor terminal experience
- Con: Agents rarely need pretty JSON
- Rejected: Auto-detection provides better UX

Table format for text output:

```
TYPE    TIME                 SOURCE        MESSAGE
error   2025-12-14 16:08:32  app.js:123   Uncaught TypeError...
log     2025-12-14 16:08:33  main.js:456  User clicked...
```

- Pro: Structured, aligned columns
- Con: Console messages can be very long (stack traces, objects)
- Con: Table breaks with multi-line messages
- Con: Less like browser console
- Rejected: Simple line format is more flexible

## Usage Examples

Basic query (get all entries):

```bash
webctl console
```

Filter by type:

```bash
webctl console --type error                    # Only errors
webctl console --type error,warn               # Errors and warnings (CSV)
webctl console --type error --type warn        # Errors and warnings (repeatable)
```

Limit results:

```bash
webctl console --tail 50      # Last 50 entries
webctl console --head 100     # First 100 entries
```

Pagination for large buffers:

```bash
webctl console --head 1000              # First batch
webctl console --range 1000-1999        # Second batch
webctl console --range 2000-2999        # Third batch
```

Debug infinite loop:

```bash
webctl console --head 20      # See what triggered the loop
```

Format control:

```bash
webctl console --format json  # Force JSON even to terminal
webctl console --format text  # Force text even to pipe
webctl console | jq '.'       # Pipe to jq (gets raw JSON automatically)
```

Combined with action command --clear (see DR-006):

```bash
webctl navigate https://example.com --clear
webctl console --type error   # Only errors from this navigation

webctl click "#submit" --clear=console
webctl console --tail 10      # Last 10 entries from the click
```

## Implementation Notes

CLI implementation:

- Parse flags with Cobra
- Validate mutual exclusivity of --head/--tail/--range
- Detect output destination (TTY vs pipe) for auto-format
- Send IPC request to daemon: `{"cmd": "console"}`
- Parse daemon response
- Format output based on --format flag or auto-detection
- Convert timestamps for text output (Unix ms → local time)

Daemon implementation:

- Already implemented (daemon.go:382-388)
- Returns: `{"ok": true, "entries": [...], "count": N}`
- No changes needed for basic functionality
- Filtering and limiting handled CLI-side (simpler, keeps daemon focused)

Text format timestamp conversion (Go):

```go
ts := time.UnixMilli(entry.Timestamp).Local()
fmt.Sprintf("[%s]", ts.Format("2006-01-02 15:04:05.000"))
```

Error cases:

Daemon not running:

```json
{"ok": false, "error": "daemon not running: connection refused"}
```

CLI error message: "Error: daemon not running. Start with: webctl start"

Empty buffer:

```json
{"ok": true, "entries": [], "count": 0}
```

Text output: (no output, exit 0)

Invalid range format:

```bash
webctl console --range abc
```

Error: "invalid range format: use START-END (e.g., 100-200)"

Mutually exclusive flags:

```bash
webctl console --head 50 --tail 50
```

Error: "Error: --head, --tail, and --range are mutually exclusive"

## CDP Methods

None - console command queries buffered data only. Buffer is populated by daemon event handlers for Runtime.consoleAPICalled and Runtime.exceptionThrown (see DR-003).

## Testing Strategy

Unit tests:

- Flag parsing and validation
- Mutual exclusivity checking (--head/--tail/--range)
- Output format detection (TTY vs pipe)
- Text formatting (timestamp conversion, line format)
- JSON formatting (pretty vs raw)
- Type filtering logic
- Range parsing and validation

Integration tests:

- Start daemon, generate console logs, query with webctl console
- Verify all entry types captured (log, warn, error, info)
- Verify filtering by type works
- Verify --head/--tail/--range limiting works
- Verify empty buffer returns count: 0
- Verify error when daemon not running

## Session Filtering

Console entries are filtered to the active session by default. Each entry includes a sessionId field identifying which page session produced it. When multiple browser tabs are open, only entries from the active session are returned.

Entries from a session are discarded when that session detaches (tab closed or cross-origin navigation). See DR-010 for full session management details.

## Updates

- 2025-12-14: Initial version
- 2025-12-16: Added sessionId field to entries, added session filtering (see DR-010)
