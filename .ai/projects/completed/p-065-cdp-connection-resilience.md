# P-065: CDP Connection Resilience

- Status: Done
- Started: 2026-01-31
- Completed: 2026-02-01

## Overview

Improve webctl's handling of browser disconnections and crashes. Currently, when the browser connection is lost (crash, network issue, or silent disconnect), the daemon shuts down entirely, requiring manual restart and losing buffered console/network data.

This project implements detection, recovery, and persistence mechanisms to handle CDP connection failures gracefully.

## Goals

1. Detect browser disconnection promptly (within 10 seconds)
2. Provide clear error messages about connection state
3. Allow graceful recovery without full daemon restart
4. Preserve buffered data across reconnection
5. Support both automatic and manual reconnection strategies

## Scope

In Scope:

- Connection health monitoring (heartbeat mechanism)
- WebSocket close handler with status code classification
- Connection state machine
- Manual reconnection command
- Automatic reconnection with exponential backoff
- Buffer preservation during disconnect
- Enhanced status reporting

Out of Scope:

- Browser crash recovery (relaunching browser process)
- Disk persistence of buffers (future enhancement)
- Remote daemon reconnection
- Multi-browser session management

## Success Criteria

- [x] Heartbeat detects silent disconnects within 10 seconds
- [x] WebSocket close events are properly classified by type
- [x] `webctl status` shows connection health information
- [x] `webctl reconnect` command enables manual recovery
- [x] Automatic reconnection succeeds for recoverable disconnects
- [x] Console and network buffers preserved across reconnection
- [x] Session recovery re-navigates to last URL
- [x] All existing tests pass
- [x] Integration tests cover reconnection scenarios

## Deliverables

- Connection state machine in `internal/daemon/`
- Heartbeat goroutine using `Browser.getVersion` CDP command
- Close code classification for disconnect handling
- Enhanced `status` command output with connection health
- New `reconnect` CLI command
- Reconnection logic with exponential backoff
- Updated daemon error handling (no immediate shutdown)
- Last URL preservation for session recovery
- Integration tests for disconnect scenarios

## Current State

The daemon currently:

- Uses `coder/websocket v1.8.14` for CDP communication
- Detects connection errors via `isConnectionError()` checking error strings (daemon.go:133-142)
- On disconnect: clears sessions, sets `browserLost` flag, shuts down (daemon.go:144-160)
- No heartbeat or proactive health monitoring
- No reconnection attempt - exits with error message

Key files and locations:

- `internal/cdp/client.go`
  - CDP client with read loop (readLoop:154-183)
  - closedCh channel signals client closure (line 30)
  - closeErr stores error that caused closure (line 31)
  - Err() method returns close error (line 147-152)
  - NewClient starts readLoop goroutine (line 39-47)
- `internal/cdp/conn.go`
  - Conn interface: Read, Write, Close only (line 12-22)
  - No changes needed - heartbeat uses CDP command not WebSocket ping
- `internal/daemon/daemon.go`
  - browserConnected() checks session count (line 103-110)
  - requireBrowser() triggers shutdown if disconnected (line 115-131)
  - sendToSession() wraps CDP calls with error detection (line 146-160)
  - shutdownOnce ensures single shutdown (line 62)
  - browserLost flag tracks disconnect cause (line 63)
  - Run() method coordinates lifecycle (line 199-346)
- `internal/daemon/session.go`
  - SessionManager.Clear() called on disconnect (line 202-210)
  - Sessions track URL/Title but lost on clear
  - No separate "last URL" preservation mechanism
- `internal/ipc/protocol.go`
  - StatusData struct needs health fields added (line 34-39)
- `internal/browser/browser.go`
  - Browser.Version() uses HTTP not WebSocket (line 168-170)
  - Browser process separate from CDP connection

WebSocket library capabilities (coder/websocket v1.8.14):

- `CloseError{Code, Reason}` - structured close info
- `CloseStatus(err) StatusCode` - extract code from error (-1 if not CloseError)
- Status codes: 1000=Normal, 1001=GoingAway, 1006=Abnormal, 1011=InternalError

Note: WebSocket-level `Ping()` only validates transport layer. We use CDP command (`Browser.getVersion`) for heartbeat instead, which validates the full stack including CDP protocol and browser responsiveness.

## Technical Approach

### Phase 1: Detection

Add proactive disconnect detection:

1. Implement heartbeat using `Browser.getVersion` CDP command (lightweight, ~2ms)
2. Add WebSocket close handler with status code classification
3. Track connection state explicitly
4. Update `status` command to show connection health

Heartbeat interval: 5 seconds with 5-second timeout (10s worst case detection).

Implementation details:

- Heartbeat goroutine is the single source of truth for disconnect detection
- Heartbeat calls `Browser.getVersion` via CDP (validates full stack)
- Heartbeat goroutine uses defer/recover to prevent panic from crashing daemon
- CDP client readLoop signals errors to heartbeat via channel
- Heartbeat decides whether to trigger reconnection based on close code
- Close handler uses `websocket.CloseStatus(err)` to classify disconnect type
- Add `ConnectionState` type and tracking field to daemon
- Log state transitions to stderr: "Connection lost (code X)", "Reconnecting...", "Reconnected"

### Phase 2: Manual Recovery

Enable user-initiated recovery:

1. Add `webctl reconnect` command
2. Preserve console/network buffers in memory during disconnect
3. Re-attach to existing browser if still running
4. Session recovery (re-navigate to last URL)

Implementation details:

- Add `lastURL` field to daemon (preserved before session clear)
- Buffers (consoleBuf, networkBuf) already persist - just don't clear them
- New CLI command: `internal/cli/reconnect.go`
- New IPC command: `reconnect` handled by daemon

Reconnection sequence:

1. Check browser alive via HTTP (`/json/version`)
2. Dial new CDP WebSocket connection
3. Call `subscribeEvents()` to register event handlers on new client
4. Call `enableAutoAttach()` to discover and attach to existing targets
5. For each attached session, call `enableDomainsForSession()`:
   - Runtime.enable, Page.enable, DOM.enable, Network.enable
   - Page.setLifecycleEventsEnabled
6. Re-navigate active session to `lastURL`

### Phase 3: Automatic Recovery

Implement transparent reconnection:

1. Exponential backoff with jitter (1s initial, 30s max, 5 attempts)
2. Connection state machine in daemon (Connected, Reconnecting, Disconnected)
3. Close code classification determines reconnection behaviour
4. Event notifications for disconnect/reconnect

Implementation details:

- Add `ConnectionState` type and `connState` field to Daemon struct
- Reconnect goroutine triggered on abnormal disconnect only
- State transitions controlled by daemon methods
- CDP client is disposable: create new client on each reconnect attempt
- Attempt counter tracks retries within Reconnecting state

### Connection State Machine

```
StateConnected ←→ StateReconnecting → StateDisconnected
                    (with attempt counter)
```

State transitions:

- Connected → Reconnecting: Abnormal close (code 1006) or heartbeat timeout
- Connected → Disconnected: Graceful close (code 1000, 1001) - user closed browser
- Reconnecting → Connected: Successful reconnection
- Reconnecting → Disconnected: Max attempts exceeded or manual stop

### Close Code Classification

| Code | Meaning | Action |
|------|---------|--------|
| 1000 (Normal) | Graceful close | No reconnect - user intent |
| 1001 (GoingAway) | Browser shutting down | No reconnect - user intent |
| 1006 (Abnormal) | No close frame | Attempt reconnect |
| Timeout | Heartbeat failed | Attempt reconnect |

### Reconnection Strategy

- Initial delay: 1 second
- Max delay: 30 seconds
- Max attempts: 5 (0 = infinite)
- Backoff factor: 2.0
- Jitter: 10%

## Testing Strategy

Unit tests:

- Connection state machine transitions
- Exponential backoff timing calculation
- WebSocket close code classification

Integration tests:

- Simulate browser crash (kill Chrome process)
- Simulate silent disconnect
- Verify buffer preservation across reconnect
- Verify session recovery

## Decisions

1. Heartbeat timing: 5s interval, 5s timeout (10s worst case detection)
2. Heartbeat method: CDP command (`Browser.getVersion`) only, not WebSocket ping. CDP validates full stack; WebSocket ping only validates transport and can cause false positives.
3. Single source of truth: Heartbeat goroutine is the authority for disconnect detection. ReadLoop errors flow to heartbeat, which decides action.
4. Close code classification: Graceful close (1000, 1001) means user intent - no reconnection. Abnormal close (1006) or timeout means crash/network issue - attempt reconnection.
5. State machine: Three states (Connected, Reconnecting, Disconnected) with attempt counter. Simpler than 5-state model, same functionality.
6. Session recovery: Always re-navigate to last URL on reconnection (dev tool optimised for resuming work)
7. In-flight commands: Fail immediately with connection error (caller handles retry)
8. Status command: Always show connection health
9. State machine location: `internal/daemon/` - daemon coordinates state with buffers, sessions, and reconnection workflow. CDP client is disposable (new client created on reconnect).
10. Panic recovery: Heartbeat goroutine uses defer/recover to prevent panics from crashing the daemon. On panic, log error and transition to Disconnected state.
11. Logging: State transitions logged to stderr for visibility. Format: "Connection lost (code 1006)", "Reconnecting (attempt 1/5)...", "Reconnected successfully".

## Related Work

- Bug report: Browser crash during rapid command execution (bugs.md)
- P-018: Browser Connection Failure Handling (basic error messages)
- DR-002: Daemon Architecture (connection management context)

## Research

Common causes of CDP disconnection:

- Browser memory exhaustion (screenshots, heap profiling)
- Long-running connections (24h+ silent disconnect)
- Network issues disrupting WebSocket
- Rapid command succession overwhelming CDP

Approaches from other libraries:

- Puppeteer: `browser.on('disconnected')` event, `browser.isConnected()` method
- Playwright: `browser.on('disconnected')` for browser process close
- go-rod: Known silent disconnect issues, users request configurable retry
- chromedp: No built-in disconnect detection, users implement custom monitoring
