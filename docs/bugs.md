# Known Bugs

## BUG-004: Multiple Chrome instances on same port causes daemon to control wrong browser

- **Status**: Fixed
- **Reported**: 2025-12-23
- **Fixed**: 2025-12-23
- **Severity**: High
- **Component**: Browser Launch

### Problem

When `webctl start` is run while Chrome is already running on the same CDP port (default 9222), a second Chrome instance launches but cannot bind to the port. The daemon connects to the first Chrome instance, but the user sees the second Chrome window. This causes:

1. Navigation commands appear to not work (they control invisible browser)
2. Screenshots/HTML come from wrong browser instance
3. Confusing UX - visible browser doesn't respond to commands

### Reproduction

```bash
# Terminal 1
webctl start --headless

# Terminal 2 (without stopping first daemon)
webctl start  # Launches visible Chrome

# Terminal 2
webctl navigate https://google.com
# Visible browser stays on about:blank (wrong instance)
# Headless browser navigates to google.com (correct instance, invisible)
```

### Root Cause

`browser.Start()` in `internal/daemon/daemon.go:124` unconditionally launches Chrome without checking if the requested port is already in use.

### Solution

Implement automatic free port finding:

- **Default port** (user didn't specify `--port`): Auto-find next available port starting from 9222
- **Explicit port** (user specified `--port 9222`): Error out if that port is busy

Files to modify:
- `internal/browser/launch.go`: Add port availability check and auto-increment logic
- `internal/daemon/daemon.go`: Handle port conflict errors
- `internal/cli/start.go`: Output actual port used when different from requested

### Related Issues

- None

### Fix Implemented

Modified `internal/browser/browser.go`:

1. Added `isPortAvailable()` function to check if a TCP port can be bound
2. Added `findFreePort()` function to find next available port in range
3. Modified `StartWithBinary()` to implement smart port selection:
   - **Default port** (opts.Port == 0): Auto-find free port starting from 9222, warn on stderr if different
   - **Explicit port** (opts.Port != 0): Verify port is available, error with `ErrPortInUse` if not

Modified `internal/daemon/daemon.go`:
- Update daemon config with actual port used after browser starts

### Behavior After Fix

```bash
# Scenario 1: Default port, no conflict
webctl start
# Uses port 9222, no warning

# Scenario 2: Default port, 9222 in use
webctl start
# Stderr: "Port 9222 in use, using port 9223 instead"
# Daemon uses port 9223

# Scenario 3: Explicit port, available
webctl start --port 9500
# Uses port 9500

# Scenario 4: Explicit port, in use
webctl start --port 9222
# Error: "port is already in use: 9222"
```

### History

- 2025-12-23: Bug discovered during P-009 design review of navigate command
- 2025-12-23: Fix implemented and tested
