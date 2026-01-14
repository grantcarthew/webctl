# P-027: Testing stop Command

- Status: Complete
- Started: 2025-12-31
- Completed: 2026-01-06

## Overview

Test the webctl stop command which sends a shutdown signal to the running daemon, cleanly closing the browser and exiting the REPL.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-stop.sh
```

## Code References

- internal/cli/stop.go

## Command Signature

```
webctl stop
```

No command-specific flags. Global flags apply:
- --debug: Enable debug output (global flag)
- --json: Output in JSON format (global flag)
- --no-color: Disable colored output (global flag)

## Test Checklist

Basic functionality:
- [x] Stop running daemon from separate terminal
- [x] Verify browser closes
- [x] Verify daemon terminal exits cleanly
- [x] Verify daemon process terminates

Output formats:
- [x] Default output (text mode)
- [x] JSON output with --json flag
- [x] No color output with --no-color flag

Error cases:
- [x] Attempt stop when daemon not running
- [x] Verify appropriate error message

CLI vs REPL:
- [x] CLI: webctl stop (from separate terminal)
- [x] REPL: stop (from REPL itself)

Cleanup verification:
- [x] Verify socket file removed
- [x] Verify no orphaned browser processes
- [x] Verify webctl status shows "not running"

## Notes

- Stop can be called from CLI (separate terminal) or REPL
- Daemon should clean up all resources on shutdown
- Browser closes gracefully before daemon exits
- Alternative shutdown: Ctrl+C in daemon terminal

## Issues Discovered

### Issue #1: Terminal Echo Not Restored on External Stop

**Discovered:** 2026-01-05

**Description:**
When stopping the daemon from an external terminal using `webctl stop`, the terminal echo was not being restored, leaving the terminal in a state where typed input was not visible.

**Root Cause:**
The readline library's `Close()` method was not reliably restoring terminal state when called from a different goroutine while `Readline()` was blocked. The daemon exit triggered readline cleanup from the daemon goroutine, but the REPL goroutine was still blocked waiting for input, causing terminal state restoration to fail.

**Solution:**
Implemented explicit terminal state management at the daemon level:

1. **Save terminal state** before REPL starts (daemon.go:280)
   - Use `term.GetState()` to capture initial terminal state
   - Store in daemon struct for later restoration

2. **Restore terminal state explicitly** before daemon exits (daemon.go:308-322)
   - Added `restoreTerminalState()` method to daemon
   - Call explicitly in every exit path of `daemon.Run()` select statement
   - Handles all shutdown scenarios: external stop, SIGINT, context cancellation, errors, and REPL exit

3. **Made REPL.Close() idempotent** (repl.go:49-56)
   - Use `sync.Once` to prevent double-close issues
   - Both REPL goroutine and daemon can safely call Close()

**Files Modified:**
- `internal/daemon/daemon.go`: Added terminal state management
- `internal/daemon/repl.go`: Made Close() idempotent with sync.Once

**Testing:**
- Verified terminal echo restored when stopping from external terminal ✓
- Verified terminal echo restored when stopping from REPL ✓
- Verified terminal echo restored on Ctrl+C ✓
