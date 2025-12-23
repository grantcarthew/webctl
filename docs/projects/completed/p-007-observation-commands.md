# P-007: Observation Commands

- Status: Complete
- Started: 2025-12-15
- Completed: 2025-12-23

## Overview

Implement the observation commands that let agents inspect the current state: console, network, screenshot, html, eval, cookies.

Each command is broken into two phases: design (write DR) and implementation (write code and tests). This ensures design decisions are documented before implementation begins.

## Goals

1. Query buffered console logs and exceptions
2. Query buffered network requests with response bodies
3. Capture screenshots
4. Extract HTML content
5. Evaluate JavaScript expressions
6. Get/set cookies

## Scope

In Scope:

- `console` command
- `network` command
- `screenshot` command
- `html` command
- `eval` command
- `cookies` command
- `target` command (session listing and switching)
- Response body fetching for network entries
- Design records for each command interface
- Unit and integration tests for each command
- Daemon REPL interface (interactive commands via stdin)
- Browser-level CDP connection with session management (fixes BUG-001)

Out of Scope:

- Navigation commands (P-008)
- Interaction commands (P-009)
- Wait-for commands (P-010)

## Success Criteria

Console Command:

- [x] DR-007 written documenting console command interface
- [x] `webctl console` returns buffered logs as JSON
- [x] Console command has unit and integration tests

Daemon REPL:

- [x] DR-008 written documenting daemon REPL interface
- [x] `webctl start` accepts interactive commands via stdin when TTY
- [x] REPL supports: status, console, network, clear, stop
- [x] REPL has readline support with command history
- [x] REPL command abbreviations (s=status, n=network, t=target, etc.)
- [x] TTY-aware JSON output (pretty in terminal, compact when piped)

Network Command:

- [x] DR-009 written documenting network command interface
- [x] `webctl network` returns buffered requests with bodies
- [x] Network command has unit tests
- [x] Network command has integration tests

Browser-Level CDP Sessions (fixes BUG-001):

- [x] DR-010 written documenting browser-level CDP with session management
- [x] Daemon connects to browser WebSocket (not page target)
- [x] Target.setAutoAttach enables automatic session tracking
- [x] Target.setDiscoverTargets enables targetInfoChanged events
- [x] Session management with active session concept
- [x] `webctl target` command for listing/switching sessions
- [x] REPL prompt shows active session context
- [x] Entries tagged with sessionId and filtered to active session
- [x] Integration test verifies session URL updates after navigation

Screenshot Command:

- [x] DR-011 written documenting screenshot command interface
- [x] `webctl screenshot` saves PNG to /tmp/webctl-screenshots/ and returns JSON with path
- [x] `webctl screenshot --full-page` captures entire scrollable page
- [x] `webctl screenshot --output <path>` saves to custom path
- [x] Screenshot command has unit and integration tests

HTML Command:

- [x] DR-012 written documenting html command interface
- [x] `webctl html` returns full page HTML to /tmp/webctl-html/ as file
- [x] `webctl html ".selector"` returns element HTML with CSS selector support
- [x] `webctl html ".selector"` returns multiple matches with HTML comment separators
- [x] `webctl html --output <path>` saves to custom path
- [x] HTML command has unit and integration tests
- [x] BUG-002: HTML command fails with "client closed while waiting for response" in REPL - FIXED
- [x] BUG-003: HTML command extremely slow and times out in REPL - FIXED (see P-011)

Eval Command:

- [x] DR-014 written documenting eval command interface
- [x] `webctl eval "1+1"` returns `2`
- [x] `webctl eval "Promise.resolve(42)"` handles async expressions
- [x] Eval command has unit and integration tests (6 unit + 7 integration)

Cookies Command:

- [x] DR-015 written documenting cookies command interface
- [x] `webctl cookies` returns all cookies as JSON
- [x] `webctl cookies set` and `webctl cookies delete` subcommands work
- [x] Cookies command has unit and integration tests (7 unit + 7 integration)
- [x] Full implementation with smart delete behavior and all flags

## Deliverables

Design Records:

- DR-006: Action Command Flags (--clear for action commands)
- DR-007: Console Command Interface
- DR-008: Daemon REPL Interface
- DR-009: Network Command Interface
- DR-010: Browser-Level CDP Sessions (session management, fixes BUG-001)
- DR-011: Screenshot Command Interface
- DR-012: HTML Command Interface
- DR-013: Eval Command Interface
- DR-014: Cookies Command Interface

Implementation Files:

- `internal/cli/console.go` - COMPLETE
- `internal/cli/network.go` - COMPLETE (unit and integration tests pass)
- `internal/cli/target.go` - COMPLETE (session listing and switching)
- `internal/cli/screenshot.go` - COMPLETE (tests pending)
- `internal/cli/html.go` - TODO
- `internal/cli/eval.go` - COMPLETE
- `internal/cli/cookies.go` - TODO
- `internal/executor/` - Executor interface for CLI/REPL command execution - COMPLETE
- `internal/daemon/repl.go` - COMPLETE (session prompt, abbreviations, TTY-aware output)
- `internal/daemon/daemon.go` - COMPLETE (browser-level connection, session management)
- `internal/cdp/client.go` - COMPLETE (session ID support)
- `internal/cdp/message.go` - COMPLETE (sessionId field)
- Daemon-side handlers for each command (internal/daemon/)
- Test files for each command (internal/cli/cli_test.go)

## Technical Design

### Console Command

Output format:

```json
{
  "ok": true,
  "entries": [
    {"type": "log", "text": "Hello", "timestamp": 1702000000.123},
    {"type": "error", "text": "Failed to fetch", "timestamp": 1702000001.456},
    {"type": "exception", "text": "Uncaught Error: ...", "stack": "...", "timestamp": 1702000002.789}
  ]
}
```

Implementation:

- Daemon returns contents of console ring buffer
- Includes both `consoleAPICalled` and `exceptionThrown` entries
- Sorted by timestamp

### Network Command

Output format:

```json
{
  "ok": true,
  "entries": [
    {
      "requestId": "123",
      "method": "GET",
      "url": "https://api.example.com/data",
      "status": 200,
      "mimeType": "application/json",
      "requestHeaders": {...},
      "responseHeaders": {...},
      "body": "{\"data\": ...}",
      "bodyBase64": false,
      "timestamp": 1702000000.123,
      "duration": 0.234
    }
  ]
}
```

Implementation:

- Daemon returns contents of network ring buffer
- Bodies fetched at `loadingFinished` time and stored
- Large bodies may be truncated (configurable limit)

### Screenshot Command

Output: Binary PNG to stdout, or JSON with base64 if `--json` flag.

```bash
webctl screenshot > page.png
webctl screenshot --json  # {"ok": true, "data": "base64..."}
```

CDP: `Page.captureScreenshot`

Options:

- `--full-page` - capture full scrollable page
- `--selector ".foo"` - capture specific element (future)

### HTML Command

```bash
webctl html                    # Full page HTML
webctl html ".content"         # Specific element
```

Output:

```json
{"ok": true, "html": "<!DOCTYPE html>..."}
{"ok": true, "html": "<div class=\"content\">...</div>"}
```

CDP:

1. `DOM.getDocument` - get root node
2. `DOM.querySelector` - find element (if selector)
3. `DOM.getOuterHTML` - get HTML

### Eval Command

```bash
webctl eval "document.title"
webctl eval "1 + 1"
webctl eval "fetch('/api').then(r => r.json())"  # async
```

Output:

```json
{"ok": true, "value": "Page Title"}
{"ok": true, "value": 2}
{"ok": true, "value": {"data": "..."}}
```

CDP: `Runtime.evaluate`

- Use `awaitPromise: true` for async expressions
- Use `returnByValue: true` for JSON-serializable results

### Cookies Command

```bash
webctl cookies                     # Get all
webctl cookies --set "name=value"  # Set cookie (future)
```

Output:

```json
{
  "ok": true,
  "cookies": [
    {"name": "session", "value": "abc123", "domain": "example.com", "path": "/", ...}
  ]
}
```

CDP: `Network.getCookies`, `Network.setCookie`

## CDP Methods Used

| Command | CDP Methods |
|---------|-------------|
| console | (buffer only) |
| network | (buffer only) + `Network.getResponseBody` |
| screenshot | `Page.captureScreenshot` |
| html | `DOM.getDocument`, `DOM.querySelector`, `DOM.getOuterHTML` |
| eval | `Runtime.evaluate` |
| cookies | `Network.getCookies`, `Network.setCookie` |

## Dependencies

- P-006 (CLI Framework)

## Required Reading

Before starting any design phase:

- docs/design/dr-writing-guide.md - How to write design records
- docs/design/design-records/dr-001-core-architecture.md - Overall system architecture and command structure
- docs/design/design-records/dr-002-cli-browser-commands.md - CLI command patterns

Console Command Design (DR-007): COMPLETE

- docs/design/design-records/dr-003-cdp-eager-data-capture.md - Console buffer implementation
- docs/design/design-records/dr-006-action-command-flags.md - --clear flag behavior
- internal/daemon/daemon.go - Existing console buffer code

Daemon REPL Design (DR-008): COMPLETE

- docs/design/design-records/dr-008-daemon-repl.md - REPL interface and Executor pattern
- internal/executor/ - Executor interface implementation
- internal/daemon/repl.go - REPL implementation

Network Command Design (DR-009):

- docs/design/design-records/dr-003-cdp-eager-data-capture.md - Network buffer and response body fetching
- docs/design/design-records/dr-006-action-command-flags.md - --clear flag behavior
- internal/daemon/daemon.go - Existing network buffer code

Browser-Level CDP Sessions (DR-010): COMPLETE

- docs/design/design-records/dr-010-browser-level-cdp-sessions.md - Session management architecture
- Fixes BUG-001 (cross-origin navigation issue)

Screenshot Command Design (DR-011):

- docs/design/design-records/dr-006-action-command-flags.md - --clear flag behavior (if applicable)
- docs/design/design-records/dr-010-browser-level-cdp-sessions.md - Session context for commands
- No additional prerequisites

HTML Command Design (DR-012):

- docs/design/design-records/dr-006-action-command-flags.md - --clear flag behavior (if applicable)
- docs/design/design-records/dr-010-browser-level-cdp-sessions.md - Session context for commands
- No additional prerequisites

Eval Command Design (DR-013):

- docs/design/design-records/dr-006-action-command-flags.md - --clear flag behavior (if applicable)
- docs/design/design-records/dr-010-browser-level-cdp-sessions.md - Session context for commands
- No additional prerequisites

Cookies Command Design (DR-014):

- docs/design/design-records/dr-006-action-command-flags.md - --clear flag behavior (if applicable)
- docs/design/design-records/dr-010-browser-level-cdp-sessions.md - Session context for commands
- No additional prerequisites

Before starting any implementation phase:

- docs/design/design-records/dr-004-testing-strategy.md - Testing approach
- The corresponding DR written in the design phase
- internal/cli/root.go - Command registration patterns
- internal/daemon/daemon.go - Daemon handler patterns
- internal/executor/ - Executor interface for command execution

## Testing Strategy

1. Unit tests - Output formatting, parameter parsing
2. Integration tests - Real browser interaction

## Work Phases

Each command follows this two-phase approach:

Phase 1 - Design:

1. Read required documentation listed above
2. Design command interface (flags, arguments, output format)
3. Identify CDP methods and sequences needed
4. Document error cases and edge conditions
5. Write design record
6. Update DR index in docs/design/design-records/README.md

Phase 2 - Implementation:

1. Read the design record from Phase 1
2. Implement CLI command in cmd/webctl/
3. Implement daemon handler in internal/daemon/
4. Write unit tests
5. Write integration tests
6. Verify success criteria

Recommended order (prioritize console and network):

1. Console (DR-007 + implementation) - COMPLETE
2. Daemon REPL (DR-008 + implementation) - COMPLETE
3. Network (DR-009 + implementation) - COMPLETE (unit and integration tests pass)
4. Browser-Level CDP Sessions (DR-010 + implementation) - COMPLETE
5. Screenshot (DR-011 + implementation) - COMPLETE (tests pending)
6. HTML (DR-012 + implementation)
7. Eval (DR-013 + implementation)
8. Cookies (DR-014 + implementation)

## Notes

Console and network commands are the most valuable for AI agents debugging web apps. They were the original motivation for webctl and should be prioritized.

The Daemon REPL was prioritized because it changed the command execution architecture (Executor interface), which should be in place before implementing additional commands.

## Known Issues

BUG-001: Network events not captured after cross-origin navigation - FIXED

Symptom: After navigating to a second URL (different origin), `webctl network` shows no new requests. Only requests from the initial page load are captured.

Root cause: The CDP WebSocket connection is to a page-level target. Cross-origin navigation with Chrome's Site Isolation creates a new renderer process and target, leaving the daemon connected to a stale target.

Solution: DR-010 (Browser-Level CDP Sessions) addresses this by:

1. Connecting to the browser-level WebSocket instead of page target
2. Using Target.setAutoAttach with flatten: true for automatic session tracking
3. Using Target.setDiscoverTargets for targetInfoChanged events (URL/title updates)
4. Tracking sessions and automatically handling target attach/detach
5. Implementing session-based command routing

Status: Implementation complete. Integration test added to verify session URL updates.

---

BUG-002: HTML command fails with "client closed while waiting for response" - FIXED

Symptom: When running `webctl html` in the REPL, the command fails with error "failed to get document: client closed while waiting for response". The error appears twice in succession.

Root cause: Multiple issues identified:

1. **Primary fix**: `handleLoadingFinished` was calling `d.cdp.SendContext` (browser-level, no session ID) for `Network.getResponseBody`, but this method requires the session context where the network event originated. This could cause CDP errors that destabilized the connection. Fixed by using `d.cdp.SendToSession(ctx, evt.SessionID, ...)`.

2. **DOM domain not enabled**: The `enableDomainsForSession` function only enabled Runtime, Network, and Page domains. Added DOM.enable to ensure DOM methods work reliably.

3. **Double error output**: The error appeared twice because both `cli.outputError` (to stderr) and `daemon.outputError` (to stdout) were called. This was a side effect of how REPL handles Cobra command errors.

4. **Missing REPL abbreviations**: Added "html" to webctlCommands and updated abbreviation tests. Also added flag resets for screenshot and html commands.

Solution applied (2025-12-18):

- Changed `handleLoadingFinished` to use `d.cdp.SendToSession(ctx, evt.SessionID, ...)` for Network.getResponseBody
- Added "DOM.enable" to the domains list in `enableDomainsForSession`
- Added "html" to webctlCommands for REPL abbreviation support (h=html)
- Updated REPL help text with new abbreviations (st=status, sc=screenshot, h=html)
- Added flag resets for screenshot and html commands in `resetCommandFlags`
- Fixed test expectations for abbreviations (s is now ambiguous with status/screenshot)

Status: Fixed. All unit and integration tests pass.

---

BUG-003: HTML command extremely slow and times out in REPL - FIXED

Symptom: When running `webctl html` in REPL after navigating to a page, DOM.getDocument takes ~10-12 seconds or times out entirely.

Root Cause: Enabling the Network domain causes Chrome to block CDP method calls until `networkIdle` lifecycle event fires. This affected all CDP operations including `Runtime.evaluate` and `DOM.getDocument`.

Solution (2025-12-22):

1. Removed `Network.enable` from initial domain enablement - Network domain is now enabled lazily on first `webctl network` command
2. Changed `navigate` command to return immediately after `Page.navigate` without waiting for `frameNavigated`
3. Updated Chrome launch flags to prevent background throttling

Result: HTML extraction now completes in <10ms instead of 10-20 seconds.

Status: Fixed. See P-011 (completed) for full investigation details.
