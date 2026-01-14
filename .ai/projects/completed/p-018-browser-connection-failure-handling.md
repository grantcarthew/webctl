# P-018: Browser Connection Failure Handling

- Status: Completed
- Started: 2025-12-26
- Completed: 2025-12-27

## Overview

Implement clean daemon shutdown when browser connection is lost. Rather than attempting automatic recovery, the daemon detects connection loss (manual close or `kill -9`) and exits cleanly with a clear error message, allowing users to restart with `webctl start`. This fail-fast approach prioritizes simplicity and reliability over implicit recovery.

## Goals

1. Detect browser connection loss on any command execution
2. Handle both graceful close and forceful termination (`kill -9`)
3. Clear session state on disconnection
4. Exit daemon cleanly with clear error message
5. Prevent confusing "client is closed" errors from being shown to users
6. Maintain consistency across all command handlers

## Scope

In Scope:

- Proactive browser connection check at start of command handlers
- Reactive detection of "client is closed" errors during CDP calls
- Clean session state cleanup on disconnection
- Daemon shutdown triggered on connection loss
- Clear error message to user
- Consistent behavior across all 16+ command handlers
- Both manual close and `kill -9` scenarios

Out of Scope:

- Automatic browser recovery/relaunch
- URL restoration after relaunch
- Warning messages (clear error is sufficient)
- Retry limiting
- Attach mode special handling (uses same fail-fast approach)

## Success Criteria

- [x] Browser disconnection detected on next command execution
- [x] Both manual close and `kill -9` handled correctly
- [x] Session state cleared on disconnection
- [x] Daemon exits cleanly with error: "browser connection lost - daemon shutting down"
- [x] Error detection via proactive `requireBrowser()` check
- [x] Error detection via reactive `sendToSession()` wrapper
- [x] Consistent behavior across all command handlers
- [x] Works in both REPL and CLI modes
- [x] All tests pass

## Deliverables

- [x] Updated `internal/daemon/daemon.go`
  - `browserConnected()` - Check if browser and CDP client are alive
  - `requireBrowser()` - Proactive check at handler start, triggers shutdown if disconnected
  - `isConnectionError()` - Detect CDP connection failures
  - `sendToSession()` - Wrapper with reactive error detection
- [x] Updated `internal/daemon/session.go`
  - `Clear()` method - Clean session state on disconnection
- [x] Updated `internal/daemon/handlers_*.go`
  - Added `requireBrowser()` check to all 16+ handlers requiring browser
  - Replaced all 34 `d.cdp.SendToSession()` calls with `d.sendToSession()`
- [x] Tests
  - All existing daemon tests pass
  - Manual testing with browser close and `kill -9`
- [x] Updated documentation
  - DR-024: Browser Auto-relaunch Strategy (with fail-fast decision)
  - P-018: This project document
  - AGENTS.md: Updated project status

## Technical Approach

Fail-fast implementation with two-layer detection:

### 1. Proactive Detection

At the start of every command handler requiring the browser:
```go
func (d *Daemon) requireBrowser() (ok bool, resp ipc.Response) {
	if d.browserConnected() {
		return true, ipc.Response{}
	}

	// Browser is dead - clear state and trigger shutdown
	d.debugf("Browser not connected - clearing state and shutting down daemon")
	d.sessions.Clear()
	go d.shutdownOnce.Do(func() {
		close(d.shutdown)
	})

	return false, ipc.ErrorResponse("browser connection lost - daemon shutting down")
}
```

### 2. Reactive Detection

All 34 CDP calls wrapped to detect connection errors:
```go
func (d *Daemon) sendToSession(ctx context.Context, sessionID, method string, params any) (json.RawMessage, error) {
	result, err := d.cdp.SendToSession(ctx, sessionID, method, params)
	if err != nil && d.isConnectionError(err) {
		d.debugf("Connection error detected in %s: %v - shutting down daemon", method, err)
		d.sessions.Clear()
		go d.shutdownOnce.Do(func() {
			close(d.shutdown)
		})
		return nil, fmt.Errorf("browser connection lost - daemon shutting down")
	}
	return result, err
}
```

### 3. Error Detection

Recognizes all CDP connection error patterns:
```go
func (d *Daemon) isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "client is closed") ||
		strings.Contains(s, "client closed while waiting") ||
		strings.Contains(s, "failed to send request")
}
```

### 4. Scenarios Handled

**Manual browser close:** Detected by `requireBrowser()` → instant shutdown
**`kill -9` browser:** Detected on first CDP call → connection error → shutdown
**Browser dies mid-request:** Multiple handlers might detect error, but `sync.Once` ensures single shutdown
**Session cleanup:** Protected by `SessionManager` mutex for thread-safety

## Design Decision Rationale

### Why Fail-Fast Over Auto-Recovery?

Initial investigation showed that auto-relaunch with proper error handling, URL restoration, retry limiting, and warning messages would require:
- ~1200+ lines of changes across 33 files
- Complex double-checked locking for thread safety
- Retry limiting with sliding time windows
- Warning message propagation through IPC protocol
- Error detection and retry wrapper for every CDP call

The fail-fast approach achieves the core goal (clean error on browser disconnect) with:
- ~300 lines of changes across 7 files
- Simple, explicit error handling
- No edge cases to manage
- Easy to understand and maintain

**Trade-off:** Users must manually restart daemon vs. automatic recovery. This is acceptable because:
- Browser crashes are rare in practice
- Explicit is better than implicit (Go philosophy)
- Simpler code means fewer bugs
- Clear error message tells user exactly what to do

## Testing Performed

- ✅ Manual testing: Browser manual close → clean shutdown with error
- ✅ Manual testing: `kill -9` browser → clean shutdown with error
- ✅ All existing daemon unit tests pass
- ✅ Build successful with no compilation errors

## Notes

The fail-fast approach prioritizes reliability and maintainability over implicit magic. When the browser dies, the daemon exits cleanly rather than attempting to recover, which could introduce subtle bugs or unexpected behavior. Users can restart with `webctl start` if needed.
