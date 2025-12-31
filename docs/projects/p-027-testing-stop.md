# P-027: Testing stop Command

- Status: In Progress
- Started: 2025-12-31

## Overview

Test the webctl stop command which sends a shutdown signal to the running daemon, cleanly closing the browser and exiting the REPL.

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
- [ ] Stop running daemon from separate terminal
- [ ] Verify browser closes
- [ ] Verify daemon terminal exits cleanly
- [ ] Verify daemon process terminates

Output formats:
- [ ] Default output (text mode)
- [ ] JSON output with --json flag
- [ ] No color output with --no-color flag

Error cases:
- [ ] Attempt stop when daemon not running
- [ ] Verify appropriate error message

CLI vs REPL:
- [ ] CLI: webctl stop (from separate terminal)
- [ ] REPL: stop (from REPL itself)

Cleanup verification:
- [ ] Verify socket file removed
- [ ] Verify no orphaned browser processes
- [ ] Verify webctl status shows "not running"

## Notes

- Stop can be called from CLI (separate terminal) or REPL
- Daemon should clean up all resources on shutdown
- Browser closes gracefully before daemon exits
- Alternative shutdown: Ctrl+C in daemon terminal

## Issues Discovered

(Issues will be documented here during testing)
