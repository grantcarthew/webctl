# P-007: Observation Commands

- Status: Proposed
- Started: -

## Overview

Implement the observation commands that let agents inspect the current state: console, network, screenshot, html, eval, cookies.

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

Out of Scope:

- Navigation commands (P-008)
- Interaction commands (P-008)
- Wait-for commands (P-009)

## Success Criteria

- [ ] `webctl console` returns buffered logs as JSON
- [ ] `webctl network` returns buffered requests with bodies
- [ ] `webctl screenshot` outputs PNG to stdout (or file)
- [ ] `webctl html` returns full page HTML
- [ ] `webctl html ".selector"` returns element HTML
- [ ] `webctl eval "1+1"` returns `2`
- [ ] `webctl cookies` returns all cookies as JSON

## Deliverables

- `cmd/webctl/console.go`
- `cmd/webctl/network.go`
- `cmd/webctl/screenshot.go`
- `cmd/webctl/html.go`
- `cmd/webctl/eval.go`
- `cmd/webctl/cookies.go`
- Daemon-side handlers for each command

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

## Testing Strategy

1. **Unit tests** - Output formatting, parameter parsing
2. **Integration tests** - Real browser interaction

## Notes

These are the most valuable commands for AI agents debugging web apps. Prioritise console and network as they were the original motivation for webctl.
