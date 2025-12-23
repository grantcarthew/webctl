# Known Bugs

## BUG-003: HTML command extremely slow and times out after navigation

- **Status**: Fixed
- **Reported**: 2025-12-19
- **Fixed**: 2025-12-22
- **Severity**: High
- **Component**: HTML Command / CDP Operations

### Problem

When running `webctl html` in REPL after navigating to a page, the command takes 10-20 seconds to complete or times out entirely. This makes the tool unusable for interactive browser automation workflows.

### Reproduction

```bash
# Start daemon and navigate
webctl start
webctl navigate https://example.com

# Try to get HTML immediately
webctl html
# Takes 10-20 seconds instead of <1 second

# Even data URLs are affected
webctl navigate data:text/html,<h1>Hello</h1>
webctl html
# Takes 10+ seconds for trivial HTML
```

### Root Cause

The `Network.enable` CDP domain was causing Chrome to block ALL CDP method calls (including `Runtime.evaluate`, `DOM.getDocument`, etc.) until the `networkIdle` lifecycle event fired.

When Network domain is enabled, Chrome tracks all network activity and queues CDP responses until the page reaches a stable state (no network requests for ~500ms). Slow-loading resources like favicon 404s would delay `networkIdle` by 10-20 seconds, blocking even simple operations.

**Key findings from investigation**:
- `Runtime.evaluate` completed at the exact same millisecond as `networkIdle` event (not `loadEventFired`)
- `DOM.getDocument` showed identical blocking behavior
- ALL CDP method calls were blocked, even failing calls
- Rod library extracted HTML in <100ms because it didn't enable Network domain by default

### Solution

Three-part fix:

1. **Remove `Network.enable` from initial domain enablement**: Don't enable Network domain automatically when sessions start
2. **Make `navigate` return immediately**: Remove wait for `frameNavigated` event (like Rod does)
3. **Add lazy Network domain enablement**: Enable Network domain only when user runs `webctl network` command

### Fix Implemented

Modified `internal/daemon/daemon.go`:

1. **enableDomainsForSession()**: Removed `Network.enable` from initial domains list
   ```go
   // Before: domains := []string{"Runtime.enable", "Network.enable", "Page.enable", "DOM.enable"}
   // After:  domains := []string{"Runtime.enable", "Page.enable", "DOM.enable"}
   ```

2. **handleNavigate()**: Returns immediately after `Page.navigate` CDP call
   ```go
   // Before: Waited for frameNavigated event (added 5 seconds)
   // After:  Returns immediately after navigate command succeeds
   ```

3. **handleNetwork()**: Added lazy Network enablement
   ```go
   // Check if Network domain enabled for this session
   // If not, call Network.enable on first use
   // Track enabled sessions in networkEnabled sync.Map
   ```

4. **Added networkEnabled field**: Track which sessions have Network domain enabled

Modified `internal/daemon/integration_test.go`:
- Updated tests to explicitly enable Network domain before testing network entries

### Test Results

Created automated test `internal/daemon/html_timing_test.go` to verify fix:

**Before fix**:
```
Navigate + HTML: 20.00372675s (FAIL - expected <2s)
Data URL HTML:   10.00183233s (FAIL - expected <500ms)
```

**After fix**:
```
Navigate + HTML: 8.535458ms (PASS)
Data URL HTML:   5.096125ms (PASS)
```

### Behavior After Fix

```bash
# Scenario 1: Normal navigation + HTML extraction
webctl navigate https://example.com
webctl html
# Completes in <10ms (previously 10-20 seconds)

# Scenario 2: Data URL HTML extraction
webctl navigate data:text/html,<h1>Test</h1>
webctl html
# Completes in <10ms (previously 10+ seconds)

# Scenario 3: Network tracking still works
webctl navigate https://example.com
webctl network
# Network domain enabled on first use, subsequent calls work normally
```

### Performance Comparison

| Operation | Before Fix | After Fix | Rod Library |
|-----------|-----------|-----------|-------------|
| Navigate + HTML | 20+ seconds | 8ms | 14ms |
| Data URL HTML | 10+ seconds | 5ms | 18ms |

### Related Issues

- Investigated but not related:
  - CDP session flattening (Target.attachToTarget with flatten: true)
  - Chrome launch flags
  - Page lifecycle event timing
  - Runtime execution context management

### History

- 2025-12-19: Bug discovered during P-011 CDP navigation debugging
- 2025-12-19: Initial hypothesis - Runtime.evaluate blocks until page load
- 2025-12-20: Critical discovery - blocking until networkIdle, not loadEventFired
- 2025-12-20: Investigated Rod's session management (flatten: true) - not the cause
- 2025-12-22: Created automated test to reproduce bug
- 2025-12-22: Root cause identified - Network.enable domain causing blocking
- 2025-12-22: Fix implemented and tested - 2500x performance improvement (20s → 8ms)
- 2025-12-22: User validated fix in real-world usage

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
