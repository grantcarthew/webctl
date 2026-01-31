# P-065: CDP Connection Resilience

- Status: Active
- Started: 2026-01-31

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

- [ ] Heartbeat detects silent disconnects within 10 seconds
- [ ] WebSocket close events are properly classified by type
- [ ] `webctl status` shows connection health information
- [ ] `webctl reconnect` command enables manual recovery
- [ ] Automatic reconnection succeeds for recoverable disconnects
- [ ] Console and network buffers preserved across reconnection
- [ ] Session recovery re-navigates to last URL
- [ ] All existing tests pass
- [ ] Integration tests cover reconnection scenarios

## Deliverables

- Connection state machine in `internal/cdp/`
- Heartbeat mechanism using `Browser.getVersion`
- Enhanced `status` command output
- New `reconnect` command
- Reconnection logic with exponential backoff
- Updated daemon error handling
- Integration tests for disconnect scenarios

## Current State

The daemon currently:

- Uses `coder/websocket` for CDP communication
- Detects connection errors via `isConnectionError()` checking error strings
- On disconnect: clears sessions, sets `browserLost` flag, shuts down
- No heartbeat or proactive health monitoring
- No reconnection attempt - just exits with error message

Key files:

- `internal/cdp/client.go` - CDP client, request/response, events
- `internal/cdp/conn.go` - WebSocket connection interface
- `internal/daemon/daemon.go` - Daemon lifecycle, shutdown handling
- `internal/daemon/session.go` - Session state tracking

## Technical Approach

### Phase 1: Detection

Add proactive disconnect detection:

1. Implement heartbeat using `Browser.getVersion` (lightweight CDP call)
2. Add WebSocket close handler with status code classification
3. Track connection state explicitly
4. Update `status` command to show connection health

Heartbeat interval: 30 seconds with 5-second timeout.

### Phase 2: Manual Recovery

Enable user-initiated recovery:

1. Add `webctl reconnect` command
2. Preserve console/network buffers in memory during disconnect
3. Re-attach to existing browser if still running
4. Session recovery (re-navigate to last URL)

### Phase 3: Automatic Recovery

Implement transparent reconnection:

1. Exponential backoff with jitter (1s initial, 30s max, 5 attempts)
2. Connection state machine (Disconnected, Connecting, Connected, Reconnecting, Failed)
3. Configuration options for reconnect behaviour
4. Event notifications for disconnect/reconnect

### Connection State Machine

```
StateDisconnected → StateConnecting → StateConnected
                                           ↓
                                    StateReconnecting
                                      ↓         ↓
                              StateConnected  StateFailed
```

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

## Decision Points

1. Default reconnection behaviour

- A: Enabled by default (new behaviour)
- B: Disabled by default, opt-in via flag/config
- C: Enabled for automatic disconnects, disabled for explicit stop

2. Status command changes

- A: Always show connection health
- B: Only show with `--health` flag
- C: Show brief indicator, details with `--health`

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
