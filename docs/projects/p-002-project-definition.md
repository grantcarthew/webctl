# P-002: Project Definition

- Status: Complete
- Started: 2025-12-11
- Completed: 2025-12-11

## Overview

Before building webctl, we need to validate core assumptions and map out the work ahead. This project investigates whether the fundamental premise works (capturing console/network via CDP), explores the design space, and produces a concrete project roadmap.

This is a "project to define the project" - research, validate, and plan.

## Goals

1. Validate that CDP console and network event capture works as expected
2. Understand the full scope of CDP capabilities we need
3. Identify technical risks and unknowns
4. Define the project phases and their dependencies
5. Create a roadmap of concrete projects (P-003, P-004, etc.)

## Scope

In Scope:

- Proof-of-concept: CDP connection and event capture (console, network)
- Research: CDP protocol for all planned commands
- Design: Project breakdown and sequencing
- Documentation: Update DR-001 with any new findings

Out of Scope:

- Full implementation of any command
- IPC/daemon architecture implementation
- CLI framework setup
- Production-quality code

## Success Criteria

- [x] Working proof-of-concept that captures console.log events via CDP
- [x] Working proof-of-concept that captures network requests via CDP
- [x] Documented understanding of CDP methods needed for each command
- [x] Identified any commands that may be problematic or need redesign
- [x] Created project documents for next 3-5 projects with clear sequencing
- [x] Updated AGENTS.md with P-003 as active project

## Deliverables

- ~~`poc/` directory with minimal Go code demonstrating CDP event capture~~ (validated, code discarded)
- `docs/research/cdp-commands.md` - mapping of webctl commands to CDP methods
- `docs/projects/p-003-*.md` through `docs/projects/p-00N-*.md` - next projects
- Updated DR-001 if any architectural changes needed

---

## PoC Validation Results

Proof-of-concept completed 2025-12-11. All core CDP assumptions validated.

### Test Environment

- Browser: Chrome launched via `snag --open-browser` (CDP on port 9222)
- Connection: WebSocket to `ws://localhost:9222/devtools/page/{id}`
- Target discovery: HTTP GET `http://localhost:9222/json`
- Test page: Local HTTP server serving HTML with console/network test buttons

### Console Event Capture

**Status: Working**

| Console Method | CDP Event | Result |
|----------------|-----------|--------|
| `console.log()` | `Runtime.consoleAPICalled` type=log | Captured |
| `console.warn()` | `Runtime.consoleAPICalled` type=warning | Captured |
| `console.error()` | `Runtime.consoleAPICalled` type=error | Captured |
| `console.info()` | `Runtime.consoleAPICalled` type=info | Captured |
| `console.debug()` | `Runtime.consoleAPICalled` type=debug | Captured |
| `console.table()` | `Runtime.consoleAPICalled` type=table | Captured |

**Notes:**
- Primitive values (strings, numbers) returned directly in `args[].value`
- Complex objects (objects, arrays) return `nil` in value - require `Runtime.getProperties` with the `objectId` for full inspection
- Timestamps included as Unix epoch floats

### Exception Capture

**Status: Working**

| Exception Type | CDP Event | Result |
|----------------|-----------|--------|
| Uncaught Error (setTimeout) | `Runtime.exceptionThrown` | Captured with full stack trace |
| Unhandled Promise Rejection | `Runtime.exceptionThrown` | Captured with full stack trace |

**Notes:**
- Stack traces include file path and line numbers
- Exception description includes error message and type

### Network Event Capture

**Status: Working**

| Event | CDP Method | Data Available |
|-------|------------|----------------|
| Request sent | `Network.requestWillBeSent` | URL, method, headers, requestId, timestamp |
| Response received | `Network.responseReceived` | Status code, URL, mimeType, headers, requestId |
| Loading complete | `Network.loadingFinished` | requestId, encodedDataLength, timestamp |
| Response body | `Network.getResponseBody` | Full body content, base64Encoded flag |

**Test Results:**

| Request Type | Result |
|--------------|--------|
| Local 200 OK | Full cycle captured (request → response → body) |
| Local 404 | Status and body captured correctly |
| Remote HTTPS (httpbin.org) | Full cycle captured including response body |
| CORS request | Works when page served via HTTP (fails from file://) |

**Notes:**
- Response bodies retrieved via `Network.getResponseBody` command after `loadingFinished` event
- Must pass `requestId` from the event to retrieve corresponding body
- Bodies available immediately after `loadingFinished` - no timing issues observed
- Large bodies work fine (tested with ~1KB JSON response)

### CDP Domain Enablement

Required enable commands before receiving events:

```
Runtime.enable  - for console and exception events
Network.enable  - for network events
Console.enable  - for legacy console.messageAdded events (optional)
```

### Questions Answered

| Question | Answer |
|----------|--------|
| Can we get network response bodies reliably? | Yes - `Network.getResponseBody` works after `loadingFinished` |
| Does console capture work for all methods? | Yes - log, warn, error, info, debug, table all captured |
| Are uncaught exceptions captured? | Yes - including unhandled promise rejections with stack traces |
| Does CORS affect capture? | No - CDP captures regardless of CORS (page must be served via HTTP, not file://) |

### Implementation Notes

1. **WebSocket Thread Safety**: gorilla/websocket is not safe for concurrent writes. Production code needs a write mutex or channel-based write queue.

2. **Command/Response Correlation**: CDP responses include the same `id` as the request. Use a sync.Map or similar to track pending commands and their response channels.

3. **Complex Console Arguments**: To get full object contents from console.log, need to:
   - Check if `args[].objectId` is present
   - Call `Runtime.getProperties` with that objectId
   - Parse the returned property descriptors

4. **Event Flow for Network**:
   ```
   requestWillBeSent → [requestWillBeSentExtraInfo] →
   responseReceived → [responseReceivedExtraInfo] →
   [dataReceived...] → loadingFinished
   ```
   The "ExtraInfo" events contain additional headers/security info.

5. **Target Selection**: The `/json` endpoint returns multiple targets (pages, service workers, extensions). Filter by `type == "page"` for main page targets.

---

## Research Areas

CDP Event Capture:

- Runtime.consoleAPICalled - does it capture all console methods? **YES**
- Runtime.exceptionThrown - uncaught exceptions **YES**
- Network.requestWillBeSent, Network.responseReceived - full request/response cycle **YES**
- Network.loadingFinished - response bodies **YES (via getResponseBody)**

CDP Commands (still to validate in implementation):

- Page.navigate, Page.reload - navigation
- Runtime.evaluate - JS execution
- DOM.querySelector + Input.dispatchMouseEvent - clicking
- Input.insertText, Input.dispatchKeyEvent - typing
- Page.captureScreenshot - screenshots
- DOM.getOuterHTML - HTML extraction
- Network.getCookies, Network.setCookies - cookie management

Browser Launch:

- How to find Chrome/Chromium on different platforms
- Required launch flags for CDP (--remote-debugging-port, etc.)
- Headless vs headful considerations

## Questions & Uncertainties (Remaining)

- ~~Can we get network response bodies reliably? (Some are streamed)~~ **ANSWERED: Yes**
- How do we handle multiple frames/iframes?
- What happens if the page navigates while we're executing a command?
- Do we need to handle browser crashes/disconnects specially?
- Performance: is 10,000 network entries with bodies actually reasonable?

## Technical Approach

1. Create minimal Go program that:
   - ~~Launches Chrome with CDP enabled~~ (used snag --open-browser instead)
   - Connects via WebSocket
   - Subscribes to Runtime and Network events
   - Logs events to stdout

2. Test with a simple HTML page that:
   - Logs to console (log, warn, error)
   - Makes fetch requests
   - Throws an uncaught error

3. Document findings and any surprises

4. Based on findings, break remaining work into projects:
   - CDP core library
   - Daemon/IPC layer
   - CLI framework
   - Individual command groups
   - Testing/polish

## Notes

This project exists because we want to validate before committing to full implementation. If CDP event capture doesn't work as expected, we need to know early.

**Outcome**: All core assumptions validated. CDP provides everything needed for webctl's console and network capture features. Ready to proceed with implementation.

---

## CDP Command Mapping

Complete mapping of webctl commands to CDP methods. Based on DR-001 command set.

### Lifecycle Commands

| webctl Command | CDP Methods | Notes |
|----------------|-------------|-------|
| `start` | Browser launch (not CDP) | Find Chrome binary, spawn with `--remote-debugging-port=9222` |
| `stop` | `Browser.close` | Clean shutdown; also terminates daemon |
| `status` | `Target.getTargetInfo` | Returns URL, title, attached state |
| `clear` | N/A (daemon internal) | Clears in-memory event buffers |

### Navigation Commands

| webctl Command | CDP Methods | Notes |
|----------------|-------------|-------|
| `navigate <url>` | `Page.navigate` | Returns frameId, loaderId; may need `Page.loadEventFired` for completion |
| `reload` | `Page.reload` | Optional `ignoreCache` param |
| `back` | `Page.navigateToHistoryEntry` | Need `Page.getNavigationHistory` first to get entryId |
| `forward` | `Page.navigateToHistoryEntry` | Same as back, different direction |

### Observation Commands

| webctl Command | CDP Methods | Notes |
|----------------|-------------|-------|
| `console` | N/A (daemon buffer) | Return buffered `Runtime.consoleAPICalled` and `Runtime.exceptionThrown` events |
| `network` | N/A (daemon buffer) + `Network.getResponseBody` | Return buffered network events; fetch bodies on demand or at `loadingFinished` |
| `screenshot` | `Page.captureScreenshot` | Returns base64 PNG; options: `clip`, `fullPage`, `format` |
| `html [selector]` | `DOM.getDocument` + `DOM.querySelector` + `DOM.getOuterHTML` | Without selector: full document; with selector: specific element |
| `eval <js>` | `Runtime.evaluate` | Returns `result.value` or `result.objectId` for complex types |
| `cookies` | `Network.getCookies` / `Network.setCookie` | Get all or set specific cookies |

### Interaction Commands

| webctl Command | CDP Methods | Notes |
|----------------|-------------|-------|
| `click <selector>` | `DOM.getDocument` + `DOM.querySelector` + `DOM.getBoxModel` + `Input.dispatchMouseEvent` | Get element coords, dispatch mousePressed + mouseReleased |
| `type <selector> <text>` | `DOM.focus` + `Input.insertText` or `Input.dispatchKeyEvent` | Focus element first, then insert text |
| `select <selector> <value>` | `Runtime.evaluate` | Execute JS: `document.querySelector(sel).value = val` + dispatch change event |
| `scroll <target>` | `Runtime.evaluate` or `Input.dispatchMouseEvent` (wheel) | JS `scrollIntoView()` or `scrollTo()` is simplest |

### Synchronisation Commands

| webctl Command | CDP Methods | Notes |
|----------------|-------------|-------|
| `wait-for <selector>` | Poll `DOM.querySelector` or use `DOM.setNodeValue` mutation observer | Simple polling is reliable |
| `wait-for network-idle` | Monitor `Network.*` events | Wait for no pending requests for N ms |
| `wait-for <js-condition>` | Poll `Runtime.evaluate` | Evaluate JS expression until truthy |

### CDP Domains Required

| Domain | Purpose |
|--------|---------|
| `Runtime` | Console events, JS evaluation, exceptions |
| `Network` | Network events, cookies, response bodies |
| `Page` | Navigation, screenshots, lifecycle events |
| `DOM` | Element queries, HTML extraction |
| `Input` | Click, type, scroll interactions |
| `Browser` | Browser-level control (close) |
| `Target` | Target info, attach/detach |

### Methods Count

| Category | Count |
|----------|-------|
| Enable commands | 4 (`Runtime.enable`, `Network.enable`, `Page.enable`, `DOM.enable`) |
| Navigation | 4 |
| Observation | 6 |
| Interaction | ~8 |
| Queries | 5 |
| **Total unique methods** | ~25-30 |

---

## Problematic Commands & Risks

### Low Risk (straightforward CDP mapping)

- `navigate`, `reload` - Direct CDP methods
- `screenshot` - Single CDP call, returns base64
- `eval` - Direct CDP method
- `cookies` - Direct CDP methods
- `console`, `network` - Already validated in PoC

### Medium Risk (some complexity)

| Command | Risk | Mitigation |
|---------|------|------------|
| `click` | Requires coordinate calculation from box model | Well-documented pattern; rod/chromedp do this |
| `type` | May need to handle special keys (Enter, Tab) | Use `Input.dispatchKeyEvent` for special keys |
| `html` | DOM node IDs can become stale after navigation | Re-query document before each operation |
| `back`/`forward` | Requires history lookup first | Two-step: get history, then navigate |

### Higher Risk (needs careful design)

| Command | Risk | Mitigation |
|---------|------|------------|
| `wait-for` | Polling vs events; timeout handling; multiple conditions | Start with simple polling; add event-based later if needed |
| `select` | Dropdowns vary (native select vs custom) | JS-based approach covers most cases; document limitations |
| Network body storage | 10,000 entries with bodies = memory pressure | Lazy body fetching; size limits; LRU eviction |

### Iframe Handling

**Risk**: Most CDP commands operate on main frame only. Iframes require frame-aware targeting.

**Recommendation**: V1 targets main frame only. Document limitation. Add frame support in future version if needed.

### Navigation Race Conditions

**Risk**: If page navigates while executing a command (e.g., `click` triggers navigation), subsequent commands may fail.

**Recommendation**:
- Commands should be atomic where possible
- Document that users should `wait-for` after navigation-triggering actions
- Consider adding `Page.frameNavigated` / `Page.loadEventFired` listeners

---

## Project Roadmap

Based on findings, the implementation breaks into these projects:

### P-003: CDP Core Library

**Goal**: Minimal CDP client library in Go

**Scope**:
- WebSocket connection management (thread-safe writes)
- Command/response correlation (ID tracking)
- Event subscription and dispatch
- JSON message encoding/decoding
- Error handling

**Deliverables**:
- `internal/cdp/` package
- Connection, send command, receive events
- Unit tests with mock WebSocket

**Dependencies**: None

---

### P-004: Browser Launch & Target Management

**Goal**: Find, launch, and connect to Chrome

**Scope**:
- Chrome binary detection (macOS, Linux, Windows paths)
- Launch with CDP flags (`--remote-debugging-port`, `--headless`, etc.)
- Target discovery via `/json` endpoint
- Page target selection and attachment

**Deliverables**:
- `internal/browser/` package
- Cross-platform Chrome detection
- Process lifecycle management

**Dependencies**: P-003 (CDP library)

---

### P-005: Daemon & IPC

**Goal**: Persistent daemon with Unix socket IPC

**Scope**:
- Daemon process management (start, stop, status)
- Unix socket server (XDG paths)
- JSON request/response protocol
- PID file management
- Event buffer storage (ring buffers for console/network)

**Deliverables**:
- `internal/daemon/` package
- `internal/ipc/` package
- Socket communication working

**Dependencies**: P-003, P-004

---

### P-006: CLI Framework & Core Commands

**Goal**: CLI interface with lifecycle commands

**Scope**:
- CLI framework setup (cobra or stdlib)
- `start`, `stop`, `status`, `clear` commands
- Client-side IPC (connect to daemon socket)
- JSON output formatting

**Deliverables**:
- `cmd/webctl/` main package
- Lifecycle commands working end-to-end
- Can start daemon, check status, stop

**Dependencies**: P-005

---

### P-007: Observation Commands

**Goal**: Console, network, screenshot, html, eval, cookies

**Scope**:
- `console` - query buffered events
- `network` - query buffered events + response bodies
- `screenshot` - `Page.captureScreenshot`
- `html` - `DOM.getDocument` + `DOM.getOuterHTML`
- `eval` - `Runtime.evaluate`
- `cookies` - `Network.getCookies`/`setCookie`

**Deliverables**:
- All observation commands working
- JSON output for each

**Dependencies**: P-006

---

### P-008: Navigation & Interaction Commands

**Goal**: Navigate, reload, back, forward, click, type, select, scroll

**Scope**:
- Navigation commands (Page.navigate, history)
- Click (coordinate calculation, mouse events)
- Type (focus, text input)
- Select (JS-based)
- Scroll (JS-based or wheel events)

**Deliverables**:
- All navigation and interaction commands working

**Dependencies**: P-007

---

### P-009: Wait-For & Synchronisation

**Goal**: Robust waiting/synchronisation

**Scope**:
- `wait-for <selector>` - poll DOM
- `wait-for network-idle` - monitor network events
- `wait-for <js-condition>` - poll Runtime.evaluate
- Timeout handling
- Configurable poll intervals

**Deliverables**:
- `wait-for` command with multiple condition types

**Dependencies**: P-008

---

### P-010: Polish & Release

**Goal**: Production-ready release

**Scope**:
- Error messages and edge cases
- Documentation (README, man page)
- Cross-platform testing (macOS, Linux)
- CI/CD pipeline
- Release binaries (goreleaser)
- Performance testing (10k event buffers)

**Deliverables**:
- v1.0.0 release
- Published binaries

**Dependencies**: P-009

---

## Project Dependency Graph

```
P-003 (CDP Core)
  │
  ▼
P-004 (Browser Launch)
  │
  ▼
P-005 (Daemon & IPC)
  │
  ▼
P-006 (CLI & Lifecycle)
  │
  ▼
P-007 (Observation)
  │
  ▼
P-008 (Navigation & Interaction)
  │
  ▼
P-009 (Wait-For)
  │
  ▼
P-010 (Polish & Release)
```

Linear dependency chain - each project builds on the previous. No parallelisation in v1.

---

## Summary

P-002 complete. Key outcomes:

1. **PoC validated** - CDP console/network capture works as expected
2. **All 18 commands mapped** - CDP methods identified for each
3. **Risks identified** - Iframes (defer), wait-for (polling first), memory (lazy bodies)
4. **8 implementation projects defined** - P-003 through P-010
5. **Linear dependency chain** - Sequential implementation path

**Next step**: Begin P-003 (CDP Core Library)
