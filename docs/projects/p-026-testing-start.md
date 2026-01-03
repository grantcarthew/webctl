# P-026: Testing start Command

- Status: In Progress
- Started: 2025-12-31

## Overview

Test the webctl start command which launches the daemon and browser. This command is foundational as it initializes the entire webctl system and enables REPL mode.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-start.sh
```

## Code References

- internal/cli/start.go

## Command Signature

```
webctl start [--headless] [--port 9222]
```

Flags:
- --headless: Run browser in headless mode (default: false)
- --port: CDP port for browser (default: 9222)
- --debug: Enable debug output (global flag)
- --json: Output in JSON format (global flag)
- --no-color: Disable colored output (global flag)

## Test Checklist

Basic functionality:
- [ ] Start daemon with default settings
- [ ] Verify browser window opens (headed mode)
- [ ] Verify REPL prompt appears
- [ ] Test REPL mode commands work

Headless mode:
- [ ] Start with --headless flag
- [ ] Verify no browser window appears
- [ ] Verify REPL still accessible
- [ ] Verify commands execute correctly

Custom port:
- [ ] Start with --port 9223
- [ ] Verify browser launches on custom port
- [ ] Verify daemon connects successfully

Global flags:
- [ ] Test --debug flag (verbose output)
- [ ] Test --json flag (JSON output format)
- [ ] Test --no-color flag (no ANSI colors)

Error cases:
- [ ] Attempt to start when already running
- [ ] Attempt to start with invalid port
- [ ] Attempt to start with port in use

CLI vs REPL:
- [ ] CLI: webctl start
- [ ] REPL: Not applicable (start launches REPL)

## Notes

- Start command is unique as it enters REPL mode and blocks
- REPL mode accessed through the started daemon terminal
- Stopping daemon (Ctrl+C or webctl stop) required to exit
- Browser launches automatically on start
- REPL prompt shows current page URL/title

## Issues Discovered

### Issue 1: Terminal echo disabled after REPL `stop` command

**Status**: FIXED ✓
**Severity**: High
**Reproducible**: Yes (before fix)
**Terminals affected**: Ghostty, Gnome Terminal (all terminals)

**Reproduction steps** (before fix):
1. Run `webctl start`
2. Type `stop` in the REPL
3. After daemon exits, terminal echo is disabled (text typed is invisible)
4. Workaround: Run `reset` command to restore terminal

**Does NOT occur when**:
- Exiting with Ctrl-C from REPL (terminal state correctly restored)

**Root cause**:
- `stop`/`exit`/`quit` commands called `r.shutdown()` which closed daemon shutdown channel
- Main daemon loop exited immediately on shutdown signal
- REPL goroutine didn't complete deferred cleanup: `defer r.readline.Close()`
- The readline library needs proper Close() to restore terminal echo settings

**Affected commands**:
- `stop`, `exit`, `quit` - All fixed

**Fix implemented**:
Changed `handleSpecialCommand()` to return `(bool, error)` instead of just `bool`. Exit commands now return `io.EOF` to signal clean exit, which allows the REPL.Run() loop to return normally and execute deferred cleanup before daemon shutdown.

**Files changed**:
- `internal/daemon/repl.go:207-237` - Modified handleSpecialCommand signature and implementation
- `internal/daemon/repl.go:85-92` - Updated call site to handle error return

**Test documentation**: `scripts/interactive/test-terminal-bug.md`
