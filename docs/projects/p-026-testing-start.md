# P-026: Testing start Command

- Status: In Progress
- Started: 2025-12-31

## Overview

Test the webctl start command which launches the daemon and browser. This command is foundational as it initializes the entire webctl system and enables REPL mode.

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

(Issues will be documented here during testing)
