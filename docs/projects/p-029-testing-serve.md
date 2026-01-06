# P-029: Testing serve Command

- Status: In Progress
- Started: 2025-12-31

## Overview

Test the webctl serve command which starts a development server with hot reload capabilities. This command has two modes: static file serving and proxy mode, and auto-starts the daemon if not running.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-serve.sh
```

## Code References

- internal/cli/serve.go
- internal/daemon/handlers_serve.go
- internal/daemon/daemon.go
- internal/daemon/repl.go
- internal/server/server.go
- internal/server/static.go
- testdata/index.html

## Command Signature

```
webctl serve [directory] [--proxy url]
```

Flags:
- --proxy <url>: Proxy mode (proxy requests to backend server)
- --port <number>: Server port (auto-detect if not specified)
- --host <address>: Network binding (default: localhost)
- --watch <paths>: Custom watch paths for hot reload
- --ignore <patterns>: Ignore patterns for file watching
- --headless: Browser headless mode (from auto-start)
- --debug: Enable debug output (global flag)
- --json: Output in JSON format (global flag)
- --no-color: Disable colored output (global flag)

## Test Checklist

Static mode - Basic functionality:
- [ ] Serve current directory (default)
- [ ] Serve specified directory
- [ ] Verify auto-start of daemon and browser
- [ ] Verify browser navigates to served URL
- [ ] Verify file serving works

Static mode - File watching and hot reload:
- [ ] Modify HTML file, verify auto-reload
- [ ] Modify CSS file, verify auto-reload
- [ ] Modify JS file, verify auto-reload
- [ ] Test custom watch paths (--watch)
- [ ] Test ignore patterns (--ignore)

Proxy mode:
- [ ] Proxy to localhost backend (--proxy localhost:3000)
- [ ] Proxy to full URL (--proxy http://api.example.com)
- [ ] Verify requests forwarded correctly
- [ ] Verify daemon auto-starts in proxy mode

Port and host options:
- [ ] Auto-detect available port (default)
- [ ] Custom port (--port 3000)
- [ ] Network binding (--host 0.0.0.0)
- [ ] Verify localhost vs network access

Auto-start behavior:
- [ ] Serve when daemon not running (auto-start)
- [ ] Serve when daemon already running (use existing)
- [ ] Headless mode (--headless flag)

Server lifecycle:
- [ ] Start server
- [ ] Stop with Ctrl+C
- [ ] Stop with webctl stop command
- [ ] Verify clean shutdown

Integration with webctl commands:
- [ ] Use console command while serving
- [ ] Use network command while serving
- [ ] Use html command while serving
- [ ] Verify all commands work during serve

Output formats:
- [ ] Default text output
- [ ] JSON output (--json)
- [ ] No color output (--no-color)
- [ ] Debug output (--debug)

Error cases:
- [ ] Serve non-existent directory
- [ ] Serve with port in use
- [ ] Proxy to unreachable backend
- [ ] Invalid --watch or --ignore patterns

CLI vs REPL:
- [ ] CLI: webctl serve ./public
- [ ] REPL: serve ./public (from already running daemon)

## Notes

- Serve command is unique with auto-start functionality
- Can serve static files or proxy to backend
- Includes file watching and hot reload for static mode
- Integrates with all other webctl commands during operation
- Primary use case for development workflow

## Issues Discovered

### Issue 1: Browser Navigation Failure on Auto-Start (FIXED)
**Severity:** High
**Status:** ✅ Fixed (2026-01-06)

**Problem:**
When `webctl serve` auto-started the daemon, the browser showed `about:blank` instead of loading the served URL. The navigation code checked for browser connection immediately, but the browser session hadn't been created yet (takes ~500ms after daemon starts).

**Root Cause:**
- `handleServeStart` in `internal/daemon/handlers_serve.go` checked `d.browserConnected()` immediately
- `browserConnected()` requires at least one session to exist
- Sessions are created asynchronously via `Target.attachedToTarget` events
- The check happened before the session was ready

**Fix:**
Modified `handleServeStart` to wait up to 10 seconds for a browser session before navigating, checking every 100ms. This ensures the session exists before attempting navigation.

**Files Modified:**
- `internal/daemon/handlers_serve.go` (lines 83-117)

---

### Issue 2: Ctrl+C Not Exiting Serve Mode (FIXED)
**Severity:** High
**Status:** ✅ Fixed (2026-01-06)

**Problem:**
Pressing Ctrl+C during `webctl serve` would stop the server but the process wouldn't exit, leaving the terminal hung. User had to kill the process manually.

**Root Cause:**
Two separate issues:
1. The readline library intercepts SIGINT before the daemon's signal handler
2. When daemon exits cleanly (returns nil), the goroutine in `runServeWithDaemon` didn't send to the `daemonErr` channel, causing line 195 to block forever

**Fix:**
1. Added SIGINT handler in REPL after readline creation that triggers shutdown and closes readline
2. Used `shutdownOnce.Do` to safely close shutdown channel (prevents double-close panic)
3. Changed `runServeWithDaemon` to always send daemon result to channel (not just errors)

**Files Modified:**
- `internal/daemon/repl.go` (added signal handling, lines 83-98)
- `internal/daemon/daemon.go` (safe shutdown callback, lines 300-304)
- `internal/cli/serve.go` (always send result, line 118)

---

### Issue 3: Directory Display Shows "." (FIXED)
**Severity:** Low
**Status:** ✅ Fixed (2026-01-06)

**Problem:**
When running `webctl serve` without arguments, the output showed `Directory: .` which is not informative about the actual directory being served.

**Fix:**
Added `filepath.Abs()` to resolve the directory to an absolute path before displaying it.

**Files Modified:**
- `internal/cli/serve.go` (lines 226-230)

---

### Enhancement: Comprehensive Test Page
**Status:** ✅ Added (2026-01-06)

Created `testdata/index.html` with comprehensive testing features:
- Visual design with gradient background and modern UI
- Live clock and page load counter (tests hot reload)
- Console test buttons (log, warn, error, info)
- Network request test button
- User agent display
- Console output mirroring

This provides a complete test environment for serve command functionality.
