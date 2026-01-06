# P-028: Testing navigate Command

- Status: Completed
- Started: 2026-01-06
- Completed: 2026-01-06

## Overview

Test the webctl navigate command which navigates the browser to a specified URL. This is a core navigation command used frequently in browser automation workflows.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-navigate.sh
```

## Code References

- internal/cli/navigate.go
- internal/daemon/handlers_navigation.go

## Command Signature

```
webctl navigate <url> [--wait] [--timeout <seconds>]
```

Flags:
- --wait: Wait for page load to complete (default: false)
- --timeout: Timeout in seconds when waiting (default: 60)
- --debug: Enable debug output (global flag)
- --json: Output in JSON format (global flag)
- --no-color: Disable colored output (global flag)

## Test Checklist

Basic functionality:
- [x] Navigate to a simple URL
- [x] Verify page loads correctly
- [x] Verify URL updates in REPL prompt
- [x] Navigate to HTTPS site
- [x] Navigate to HTTP site

URL handling:
- [x] Full URL with protocol (https://example.com)
- [x] URL without protocol (example.com) - should auto-add https://
- [x] URL with path (example.com/page)
- [x] URL with query params (example.com?foo=bar)

Wait behaviour:
- [x] Default wait (returns immediately, page loads in background)
- [x] Explicit --wait flag
- [x] --wait=false (return immediately)
- [x] --timeout with slow page

Output formats:
- [x] Default output (text mode)
- [x] JSON output with --json flag
- [x] No color output with --no-color flag

Error cases:
- [x] Invalid URL format
- [x] Non-existent domain
- [x] Timeout on slow/hanging page
- [x] Navigate when daemon not running

CLI vs REPL:
- [x] CLI: webctl navigate <url>
- [x] REPL: navigate <url>
- [x] REPL abbreviation: nav <url>

## Bugs Found and Fixed

### Bug 1: REPL prompt not updating after external navigate command

**Symptom:** When running `webctl navigate example.com` from CLI while REPL is running, the prompt showed old URL until Enter was pressed.

**Root Cause:** In daemon.go, `displayExternalCommand` was called BEFORE `handleRequest`, so the prompt was refreshed before the session URL was updated.

**Fix:** Moved `displayExternalCommand` to run AFTER `handleRequest` returns:
```go
ipcHandler := func(req ipc.Request) ipc.Response {
    resp := d.handleRequest(req)
    // Notify REPL AFTER handling (so prompt reflects updated state)
    if d.repl != nil {
        summary := formatCommandSummary(req)
        d.repl.displayExternalCommand(summary)
    }
    return resp
}
```

### Bug 2: REPL prompt not updating from CDP events

**Symptom:** After navigation, the prompt didn't update until the next command even when session URL changed via CDP events.

**Root Cause:** `Target.targetInfoChanged` events updated the session URL but didn't refresh the REPL prompt.

**Fix:** Added `refreshPrompt()` method to REPL and called it after `Target.targetInfoChanged` updates the session.

### Bug 3: --wait flag not working (returning immediately)

**Symptom:** `webctl navigate example.com --wait` returned immediately without waiting for page load.

**Root Cause:** Race condition - for cached/fast pages, `Page.domContentEventFired` fired and removed the session from `d.navigating` BEFORE `waitForLoadEvent` could register its waiter channel. The waiter check saw navigation was "complete" and returned immediately.

**Fix:** Register the `loadWaiter` channel BEFORE sending `Page.navigate`/`Page.reload` command:
```go
// Register BEFORE sending command to avoid race
if params.Wait {
    loadWaiterCh = make(chan struct{}, 1)
    d.loadWaiters.Store(activeID, loadWaiterCh)
}

// Send command
d.sendToSession(ctx, activeID, "Page.navigate", ...)

// Wait on pre-registered channel
select {
case <-loadWaiterCh:
    // Load event fired
case <-time.After(timeout):
    return ipc.ErrorResponse("timeout waiting for page load")
}
```

Applied same fix to `handleReload`.

## Notes

- Navigate is one of the most frequently used commands
- URL without protocol defaults to https:// (except localhost/127.0.0.1/0.0.0.0 which get http://)
- Default behaviour is fast return (~100ms) - use --wait for blocking navigation
- Wait behaviour is critical for automation reliability
