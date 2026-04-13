# Heartbeat Disconnect Detection

- Status: Pending
- Started: -

## Overview

Add proactive CDP disconnect detection to the daemon using a heartbeat goroutine. Currently, when the browser disconnects silently (crash, network drop, memory exhaustion), the daemon sits idle indefinitely until the next user command fails. This project adds a lightweight heartbeat that detects disconnection within 10 seconds and shuts the daemon down with a clear, classified error message.

This replaces the abandoned P-065 (CDP Connection Resilience) feature branch, which attempted full automatic reconnection with a connection state machine, exponential backoff, and session recovery. That approach introduced significant concurrency complexity (CDP client hot-swapping, TOCTOU races on client pointers, sync.Map clearing during reconnection, heuristic timing delays) for marginal benefit. The simple alternative: detect the disconnect, tell the user what happened, exit cleanly. The user runs `webctl start` again.

## Goals

1. Detect silent browser disconnections within 10 seconds
2. Classify disconnect cause (graceful close vs crash vs timeout)
3. Display a clear, specific error message on daemon exit
4. No new concurrency complexity (no client replacement, no state machine, no reconnection)

## Scope

In Scope:

- Heartbeat goroutine sending `Browser.getVersion` at a fixed interval
- WebSocket close code classification for error messages
- Enhanced daemon exit message based on classification
- Unit tests for close code classification
- Integration test for heartbeat detection

Out of Scope:

- Automatic reconnection (deliberate omission)
- Manual reconnect command
- Connection state machine
- CDP client replacement at runtime
- Buffer preservation across disconnects
- Session recovery
- Changes to `webctl status` output

## Success Criteria

- [ ] Heartbeat detects silent disconnects within 10 seconds (5s interval + 5s timeout)
- [ ] Browser crash produces: "browser crashed (connection lost unexpectedly)"
- [ ] Browser closed by user produces: "browser closed normally"
- [ ] Heartbeat timeout produces: "browser unresponsive (heartbeat timeout)"
- [ ] Daemon shuts down cleanly after detection (no hang, no panic)
- [ ] Existing tests pass unchanged
- [ ] Unit tests cover close code classification
- [ ] No new exported types or fields on Daemon struct

## Deliverables

- Modified `internal/daemon/daemon.go` - heartbeat startup in `Run()`, shutdown on disconnect
- New `internal/daemon/heartbeat.go` - heartbeat goroutine and close code classification
- New `internal/daemon/heartbeat_test.go` - unit tests

## Current State

The daemon currently detects disconnection reactively:

- `isConnectionError()` checks error strings for known patterns (daemon.go:133-142)
- `sendToSession()` triggers shutdown when a connection error is detected during a CDP call (daemon.go:144-160)
- `requireBrowser()` checks session count and triggers shutdown if zero (daemon.go:115-131)
- No proactive monitoring exists; silent disconnects go undetected indefinitely
- On disconnect, exit message is generic: "browser connection lost - daemon shutting down"

CDP client details (internal/cdp/client.go):

- `readLoop()` runs in a goroutine, sets `closeErr` and closes `closedCh` on read failure (line 154-183)
- `Err()` returns the error that caused closure (line 147-152)
- `SendContext()` returns early if client is closed (line 77-79)
- `closed` is an `atomic.Bool` for lock-free status checks (line 29)
- WebSocket library: `coder/websocket v1.8.14`
  - `websocket.CloseStatus(err)` extracts close code from error (-1 if not a CloseError)
  - Close codes: 1000=Normal, 1001=GoingAway, 1006=Abnormal

Shutdown path (daemon.go):

- `shutdown` channel (line 61) signals daemon exit
- `shutdownOnce` (line 62) prevents double-close panic
- `browserLost` flag (line 63) controls exit message
- `Run()` select loop (line 327-345) waits on shutdown channel

## Technical Approach

### Heartbeat Goroutine

A single goroutine started in `Run()` after CDP connection is established. It:

1. Ticks every 5 seconds
2. Sends `Browser.getVersion` with a 5-second timeout (worst case 10s detection)
3. On failure, classifies the error and triggers shutdown
4. Stops when the daemon context is cancelled or shutdown is signalled

`Browser.getVersion` is used instead of WebSocket ping because it validates the full CDP stack (protocol layer, browser responsiveness), not just transport. It is lightweight (~2ms on local connections).

The goroutine sends the disconnect error to a channel. The `Run()` select loop receives from this channel alongside the existing shutdown/signal/error cases. On receipt, it sets `browserLost` with a classified message and triggers shutdown through the existing path.

### Close Code Classification

A pure function that takes an error and returns a human-readable disconnect reason:

| Input | Message |
|-------|---------|
| Close code 1000 (Normal) | "browser closed normally" |
| Close code 1001 (GoingAway) | "browser closed normally" |
| Close code 1006 (Abnormal) | "browser crashed (connection lost unexpectedly)" |
| Close code -1 (not a WebSocket error) | "browser unresponsive (heartbeat timeout)" |
| Context deadline exceeded | "browser unresponsive (heartbeat timeout)" |

This classification is used only for the exit message. It does not drive any branching logic (no reconnection, no state transitions).

### Changes to Run()

Add a `disconnectCh` case to the existing select in `Run()`:

```
case err := <-disconnectCh:
    msg := classifyDisconnect(err)
    fmt.Fprintf(os.Stderr, "Error: %s - daemon shutting down\n", msg)
    return nil
```

The heartbeat goroutine is started after `enableAutoAttach()` and before the IPC server. It receives the daemon context for cancellation. No new fields on the Daemon struct are exported; `disconnectCh` is a local variable in `Run()`.

### What This Does NOT Do

- Does not add a `connectionManager`, `ConnectionState`, or state machine
- Does not replace or swap the CDP client at runtime
- Does not add mutexes around `d.cdp` access
- Does not add a `reconnect` command or IPC handler
- Does not modify `webctl status` output
- Does not preserve buffers or session state across disconnects
- Does not add any fields to `ipc.StatusData` or `ipc.protocol.go`

## Testing Strategy

Unit tests (heartbeat_test.go):

- `TestClassifyDisconnect` - table-driven test covering all close code paths
  - nil error
  - Normal closure (1000)
  - GoingAway (1001)
  - Abnormal closure (1006)
  - Non-WebSocket error (timeout, network error)
  - Context deadline exceeded

Integration test (in existing integration_test.go):

- Start daemon with headless browser
- Kill browser process
- Verify daemon exits within 15 seconds (heartbeat interval + timeout + margin)
- Verify stderr contains classified error message

## Decisions

1. Heartbeat method: `Browser.getVersion` via CDP, not WebSocket `Ping()`. CDP validates the full stack; WebSocket ping only validates transport.
2. Heartbeat timing: 5s interval, 5s timeout. Worst case 10s detection. Matches P-065 research findings.
3. On disconnect: always shut down. No distinction between graceful and abnormal for behaviour (both exit). The classification is purely for the error message.
4. No new exported API surface. The heartbeat is an internal implementation detail of `Run()`.
5. No changes to status output. Connection health reporting adds complexity for minimal value when the daemon exits on disconnect anyway.
