# P-007: Observation Commands

- Status: Proposed
- Started: -

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
- Response body fetching for network entries
- Design records for each command interface
- Unit and integration tests for each command

Out of Scope:

- Navigation commands (P-008)
- Interaction commands (P-009)
- Wait-for commands (P-010)

## Success Criteria

Console Command:

- [x] DR-007 written documenting console command interface
- [ ] `webctl console` returns buffered logs as JSON
- [ ] Console command has unit and integration tests

Network Command:

- [ ] DR-008 written documenting network command interface
- [ ] `webctl network` returns buffered requests with bodies
- [ ] Network command has unit and integration tests

Screenshot Command:

- [ ] DR-009 written documenting screenshot command interface
- [ ] `webctl screenshot` outputs PNG to stdout or JSON with base64
- [ ] `webctl screenshot --full-page` captures entire scrollable page
- [ ] Screenshot command has unit and integration tests

HTML Command:

- [ ] DR-010 written documenting html command interface
- [ ] `webctl html` returns full page HTML
- [ ] `webctl html ".selector"` returns element HTML
- [ ] HTML command has unit and integration tests

Eval Command:

- [ ] DR-011 written documenting eval command interface
- [ ] `webctl eval "1+1"` returns `2`
- [ ] `webctl eval "Promise.resolve(42)"` handles async expressions
- [ ] Eval command has unit and integration tests

Cookies Command:

- [ ] DR-012 written documenting cookies command interface
- [ ] `webctl cookies` returns all cookies as JSON
- [ ] Cookies command has unit and integration tests

## Deliverables

Design Records:

- DR-006: Action Command Flags (--clear for action commands)
- DR-007: Console Command Interface
- DR-008: Network Command Interface
- DR-009: Screenshot Command Interface
- DR-010: HTML Command Interface
- DR-011: Eval Command Interface
- DR-012: Cookies Command Interface

Implementation Files:

- `cmd/webctl/console.go`
- `cmd/webctl/network.go`
- `cmd/webctl/screenshot.go`
- `cmd/webctl/html.go`
- `cmd/webctl/eval.go`
- `cmd/webctl/cookies.go`
- Daemon-side handlers for each command
- Test files for each command

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

Network Command Design (DR-008):

- docs/design/design-records/dr-003-cdp-eager-data-capture.md - Network buffer and response body fetching
- docs/design/design-records/dr-006-action-command-flags.md - --clear flag behavior
- internal/daemon/daemon.go - Existing network buffer code

Screenshot Command Design (DR-009):

- docs/design/design-records/dr-006-action-command-flags.md - --clear flag behavior (if applicable)
- No additional prerequisites

HTML Command Design (DR-010):

- docs/design/design-records/dr-006-action-command-flags.md - --clear flag behavior (if applicable)
- No additional prerequisites

Eval Command Design (DR-011):

- docs/design/design-records/dr-006-action-command-flags.md - --clear flag behavior (if applicable)
- No additional prerequisites

Cookies Command Design (DR-012):

- docs/design/design-records/dr-006-action-command-flags.md - --clear flag behavior (if applicable)
- No additional prerequisites

Before starting any implementation phase:

- docs/design/design-records/dr-004-testing-strategy.md - Testing approach
- The corresponding DR written in the design phase
- cmd/webctl/root.go - Command registration patterns
- internal/daemon/daemon.go - Daemon handler patterns

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

1. Console (DR-007 + implementation) - DESIGN COMPLETE
2. Network (DR-008 + implementation)
3. Screenshot (DR-009 + implementation)
4. HTML (DR-010 + implementation)
5. Eval (DR-011 + implementation)
6. Cookies (DR-012 + implementation)

## Notes

Console and network commands are the most valuable for AI agents debugging web apps. They were the original motivation for webctl and should be prioritized.
