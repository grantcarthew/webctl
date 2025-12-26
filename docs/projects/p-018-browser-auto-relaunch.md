# P-018: Browser Auto-relaunch

- Status: Proposed
- Started: TBD
- Completed: TBD

## Overview

Implement automatic browser relaunching when the browser closes or crashes while the daemon is still running. This eliminates the current dead-end state where users cannot recover from browser closure without restarting the entire daemon, providing seamless recovery and better user experience.

## Goals

1. Detect when browser connection is lost while daemon is running
2. Automatically relaunch browser when any command requires it
3. Restore browser to last known URL after relaunch
4. Preserve original browser settings (headless, port) on relaunch
5. Show warning message to user when auto-relaunch occurs
6. Handle relaunch failures gracefully with daemon shutdown
7. Prevent crash loops with retry limiting
8. Support both normal and attach modes appropriately

## Scope

In Scope:

- Browser connection monitoring and detection of disconnection
- Automatic browser relaunch on any command requiring browser
- Warning message before relaunch (stderr, colored if terminal)
- Preservation of browser launch settings (headless flag, CDP port)
- Restoration of last active URL after relaunch
- Retry limit tracking (3 relaunches within 60 second window)
- Daemon shutdown on relaunch failure
- Daemon shutdown when retry limit exceeded
- Attach mode handling (cannot relaunch external browser)
- Same behavior in REPL and CLI modes
- JSON mode support for warnings

Out of Scope:

- Cookie/storage persistence across relaunches (fresh profile)
- Navigation history restoration
- Multiple tab restoration
- Automatic reconnection attempts for attach mode
- Configurable retry limits (hardcoded to 3/60s)
- Browser process health monitoring
- Graceful vs ungraceful shutdown detection

## Success Criteria

- [ ] Browser disconnection is detected immediately
- [ ] Any command auto-relaunches browser if disconnected
- [ ] Warning message shown before relaunch
- [ ] Browser relaunches with same settings (headless, port)
- [ ] Browser navigates to last URL after relaunch
- [ ] Command executes after successful relaunch
- [ ] Relaunch failure causes daemon to exit with error
- [ ] Retry limit (3 in 60s) prevents crash loops
- [ ] Exceeding retry limit exits daemon with error
- [ ] Attach mode shows appropriate error and exits
- [ ] Works identically in REPL and CLI
- [ ] JSON output includes warning field
- [ ] Text mode shows colored warning (if terminal supports color)

## Deliverables

- Updated `internal/daemon/daemon.go` - Browser state tracking and relaunch logic
  - Track browser launch settings
  - Track last active URL
  - Retry counter with time window
  - Auto-relaunch function
- Updated `internal/daemon/handlers_*.go` - Add relaunch check to all handlers
  - Check browser connection before executing
  - Call relaunch if disconnected
  - Show warning message
- Updated `internal/cli/format/text.go` - Warning message formatter
  - Warning output function
  - Color support for warnings
- Updated `internal/ipc/protocol.go` - Add warning field to responses
  - Warning string in Response struct
  - JSON serialization support
- Tests for auto-relaunch logic
  - Browser disconnection detection
  - Successful relaunch and command execution
  - Relaunch failure handling
  - Retry limit enforcement
  - Attach mode behavior
- Updated documentation
  - User-facing behavior docs
  - Error message documentation
- DR-024: Browser Auto-relaunch Strategy
- Updated AGENTS.md

## Technical Approach

High-level implementation strategy:

1. Browser State Tracking

Store in daemon:
- Launch settings: headless (bool), port (int), attach mode (bool)
- Last active URL: string
- Relaunch attempts: []time.Time (sliding window)

2. Connection Monitoring

Detect browser disconnection:
- CDP connection close events
- Failed CDP commands
- Session list becomes empty

3. Auto-relaunch Logic

Before executing any command requiring browser:
```go
func (d *Daemon) ensureBrowserRunning() (warning string, err error) {
  if d.browserConnected() {
    return "", nil
  }

  // Check attach mode
  if d.config.AttachMode {
    return "", errors.New("browser connection lost (attach mode - cannot relaunch)")
  }

  // Check retry limit
  if d.exceedsRetryLimit() {
    return "", errors.New("browser crash limit reached (3 relaunches in 60s)")
  }

  // Record attempt
  d.recordRelaunchAttempt()

  // Relaunch browser
  warning = "Browser not running. Relaunching..."
  if err := d.launchBrowser(); err != nil {
    return "", fmt.Errorf("failed to launch browser: %w", err)
  }

  // Navigate to last URL if available
  if d.lastURL != "" {
    warning = fmt.Sprintf("Browser not running. Relaunching to %s...", d.lastURL)
    if err := d.navigateToURL(d.lastURL); err != nil {
      // Navigation failure is non-fatal, browser is running
      return warning, nil
    }
  }

  return warning, nil
}
```

4. Handler Integration

Every command handler calls ensureBrowserRunning first:
```go
func (d *Daemon) handleNavigate(req ipc.Request) ipc.Response {
  warning, err := d.ensureBrowserRunning()
  if err != nil {
    // Fatal error - shutdown daemon
    d.Shutdown()
    return ipc.ErrorResponse(err.Error())
  }

  // Continue with normal command execution
  // Include warning in response if present
  resp := d.executeNavigate(req)
  if warning != "" {
    resp.Warning = warning
  }
  return resp
}
```

5. Retry Limiting

Sliding window approach:
```go
func (d *Daemon) exceedsRetryLimit() bool {
  now := time.Now()
  window := 60 * time.Second
  limit := 3

  // Remove attempts outside window
  var recent []time.Time
  for _, t := range d.relaunchAttempts {
    if now.Sub(t) < window {
      recent = append(recent, t)
    }
  }
  d.relaunchAttempts = recent

  return len(recent) >= limit
}
```

6. URL Tracking

Update last URL on successful navigation:
```go
func (d *Daemon) handleNavigate(req ipc.Request) ipc.Response {
  // ... relaunch check ...

  // Navigate
  err := d.cdp.Navigate(params.URL)
  if err == nil {
    d.lastURL = params.URL  // Track successful navigation
  }

  // ... return response ...
}
```

7. Warning Output

Text mode (stderr):
```
Warning: Browser not running. Relaunching to example.com...
```

JSON mode (response field):
```json
{
  "ok": true,
  "warning": "Browser not running. Relaunched to example.com.",
  "data": {...}
}
```

## Questions & Uncertainties

- Should we track URLs for all commands (html with selector, etc.)?
- What if last URL is unreachable on relaunch (404, network error)?
- Should retry limit be configurable or hardcoded?
- Should we distinguish between manual close vs crash?
- Should attach mode attempt to reconnect instead of immediate error?
- How to handle browser launch that succeeds but navigation fails?
- Should we preserve browser window size/position on relaunch?

## Testing Strategy

- Unit tests for retry limit logic (sliding window)
- Unit tests for relaunch decision logic
- Integration tests for browser disconnection detection
- Integration tests for successful relaunch flow
- Integration tests for relaunch failure scenarios
- Integration tests for retry limit enforcement
- Integration tests for attach mode behavior
- Integration tests for URL restoration
- Manual testing with browser crashes
- Manual testing in both REPL and CLI modes

## Notes

This feature transforms a dead-end error state into automatic recovery, significantly improving user experience. Users no longer need to understand the daemon/browser relationship to recover from browser closure.

The retry limit prevents infinite crash loops while still allowing legitimate reconnections (browser closed manually, port conflict resolved, etc.).

URL restoration provides continuity - users don't lose their place when the browser crashes.

Attach mode cannot auto-relaunch because the daemon doesn't control the external browser. Clean daemon shutdown is the right behavior.

Same behavior in REPL and CLI maintains consistency and predictability.
