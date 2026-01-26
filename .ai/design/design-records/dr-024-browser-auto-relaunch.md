# DR-024: Browser Auto-relaunch Strategy

- Date: 2025-12-26
- Status: Proposed
- Category: Daemon

## Problem

When the browser closes or crashes while the daemon is still running, users encounter a dead-end state with no recovery path:

Current broken flow:
```bash
webctl [example.com]> # Browser crashes or user closes it
webctl> navigate test.com
Error: no active session

webctl> status
No browser
pid: 42976

webctl> start
Error: daemon is already running

# User is stuck - must exit and restart entire daemon
```

The daemon and browser are separate processes with independent lifecycles. The browser can close (manually or crash) while the daemon continues running. Currently there is no way to recover without stopping the daemon and running `webctl start` again.

Issues with current state:
- Confusing error messages with no clear resolution
- Loss of daemon state (buffered console/network events)
- Poor user experience for browser crashes
- No distinction between temporary and permanent failures

## Decision

Implement automatic browser relaunching with intelligent retry limiting and state preservation.

Behavior:

When any command requires the browser:
1. Check if browser is connected
2. If not connected, show warning and auto-relaunch
3. Restore to last known URL
4. Execute the command
5. If relaunch fails, exit daemon

Commands affected (all commands requiring browser):
- Navigation: navigate, reload, back, forward
- Interaction: click, type, select, scroll, focus, key
- Observation: console, network, screenshot, html, eval, cookies, find
- Utility: ready, target

Commands NOT affected (daemon-only):
- status (shows browser state)
- stop (exits daemon)
- clear (clears buffers)

Example flow:
```bash
webctl [example.com]> # Browser crashes
webctl> screenshot
Warning: Browser not running. Relaunching to example.com...
/tmp/webctl-screenshots/screenshot.png
```

## Why

Seamless Recovery:

Users should not need to understand daemon vs browser architecture to recover from browser closure. Auto-relaunch provides transparent recovery.

State Preservation:

The daemon contains valuable state:
- Buffered console logs
- Buffered network requests
- Session history
- Configuration

Restarting the entire daemon loses this state. Auto-relaunch preserves it.

Better UX than Manual Commands:

Alternatives considered:
- Add `webctl browser` command to manually relaunch
- Make `webctl start` relaunch browser if daemon running

Both require user to understand the problem and know the solution. Auto-relaunch just works.

Predictable Failure Modes:

Clear failure modes with daemon shutdown:
- Relaunch fails → exit daemon (port conflict, Chrome missing, etc.)
- Crash loop → exit daemon (browser repeatedly crashes)
- Attach mode → exit daemon (cannot relaunch external browser)

Better than leaving daemon in broken state.

URL Restoration Provides Continuity:

Navigating back to last URL provides context continuity. User doesn't lose their place when browser crashes.

## Trade-offs

Accept:

- Additional complexity in daemon (state tracking, retry logic)
- Auto-relaunch might surprise users who closed browser intentionally
- Fresh browser profile (cookies/storage lost)
- Cannot restore multiple tabs
- Daemon exits on repeated failures (not self-healing)
- Attach mode cannot auto-relaunch

Gain:

- Transparent recovery from browser closure
- Better UX (no manual intervention needed)
- State preservation (buffered events retained)
- Clear failure modes (daemon exits vs broken state)
- Consistent behavior across REPL and CLI
- Crash loop protection with retry limiting

## Alternatives

Manual Browser Command:

Add `webctl browser` command for manual relaunch:

```bash
webctl> screenshot
Error: no active session
webctl> browser
Browser launched
webctl> screenshot
/tmp/screenshot.png
```

- Pro: Explicit, user controls when to relaunch
- Pro: No surprise auto-relaunches
- Con: Requires user to know the solution
- Con: Extra step, manual intervention
- Con: Still poor UX for crashes
- Rejected: Auto-relaunch is more seamless

Smart Start Command:

Make `webctl start` relaunch browser if daemon already running:

```bash
webctl> screenshot
Error: no active session
webctl> start
Browser relaunched
webctl> screenshot
/tmp/screenshot.png
```

- Pro: Reuses existing command
- Pro: No new concepts
- Con: Confusing semantics (`start` when already started)
- Con: Still requires manual intervention
- Con: Inconsistent with start command purpose
- Rejected: Auto-relaunch is better UX

Keep Broken State:

Do nothing, leave daemon in broken state:

```bash
webctl> screenshot
Error: no active session
# User must figure out daemon restart
```

- Pro: Simpler implementation
- Pro: No auto-behavior surprises
- Con: Terrible UX
- Con: User must understand architecture
- Con: Lose buffered state on restart
- Rejected: Unacceptable user experience

Auto-reconnect for Attach Mode:

Attempt to reconnect to external browser in attach mode:

```bash
# Attach mode, browser closes
webctl> screenshot
Warning: Browser connection lost. Reconnecting to :9222...
# Keeps trying to reconnect
```

- Pro: Could recover if external browser restarts
- Con: Infinite retry loop if browser gone
- Con: Unclear when to give up
- Con: Daemon doesn't control external browser
- Rejected: Clean exit is better than retry loop

## Structure

Daemon State Tracking:

```go
type Daemon struct {
  // ... existing fields ...

  // Browser launch configuration
  browserConfig struct {
    headless   bool
    port       int
    attachMode bool
    attachURL  string
  }

  // State for auto-relaunch
  lastURL          string
  relaunchAttempts []time.Time
}
```

Browser Connection Check:

Called by every command handler before execution:

```go
func (d *Daemon) ensureBrowserRunning() (warning string, err error) {
  // Check if browser is connected
  if d.isBrowserConnected() {
    return "", nil
  }

  // Cannot relaunch in attach mode
  if d.browserConfig.attachMode {
    return "", fmt.Errorf("browser connection lost (attach mode - cannot relaunch)")
  }

  // Check retry limit (3 relaunches in 60 seconds)
  if d.exceedsRetryLimit() {
    return "", fmt.Errorf("browser crash limit reached (3 relaunches in 60s)")
  }

  // Record attempt
  d.recordRelaunchAttempt()

  // Generate warning message
  if d.lastURL != "" {
    warning = fmt.Sprintf("Browser not running. Relaunching to %s...", d.lastURL)
  } else {
    warning = "Browser not running. Relaunching..."
  }

  // Relaunch browser
  if err := d.launchBrowser(d.browserConfig); err != nil {
    return "", fmt.Errorf("failed to launch browser: %w", err)
  }

  // Navigate to last URL if available
  if d.lastURL != "" {
    // Non-fatal if navigation fails (browser is running)
    _ = d.navigateToURL(d.lastURL)
  }

  return warning, nil
}
```

Retry Limit Logic:

Sliding 60-second window with 3 attempt limit:

```go
func (d *Daemon) exceedsRetryLimit() bool {
  const limit = 3
  const window = 60 * time.Second

  now := time.Now()

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

func (d *Daemon) recordRelaunchAttempt() {
  d.relaunchAttempts = append(d.relaunchAttempts, time.Now())
}
```

URL Tracking:

Track last successful navigation:

```go
func (d *Daemon) handleNavigate(req ipc.Request) ipc.Response {
  // Ensure browser running (may relaunch)
  warning, err := d.ensureBrowserRunning()
  if err != nil {
    d.Shutdown()
    return ipc.ErrorResponse(err.Error())
  }

  // Execute navigation
  var params ipc.NavigateParams
  json.Unmarshal(req.Params, &params)

  err = d.cdp.Navigate(params.URL)
  if err != nil {
    return ipc.ErrorResponse(err.Error())
  }

  // Track successful navigation
  d.lastURL = params.URL

  resp := ipc.SuccessResponse(nil)
  resp.Warning = warning
  return resp
}
```

Handler Integration Pattern:

Every command handler follows this pattern:

```go
func (d *Daemon) handleCommand(req ipc.Request) ipc.Response {
  // 1. Ensure browser running (auto-relaunch if needed)
  warning, err := d.ensureBrowserRunning()
  if err != nil {
    // Fatal error - shutdown daemon
    d.Shutdown()
    return ipc.ErrorResponse(err.Error())
  }

  // 2. Execute command logic
  // ... normal command implementation ...

  // 3. Include warning in response if relaunch occurred
  resp := ipc.SuccessResponse(data)
  if warning != "" {
    resp.Warning = warning
  }
  return resp
}
```

Response Structure:

Add warning field to IPC response:

```go
type Response struct {
  OK      bool            `json:"ok"`
  Data    json.RawMessage `json:"data,omitempty"`
  Error   string          `json:"error,omitempty"`
  Warning string          `json:"warning,omitempty"`  // New field
}
```

## Output Examples

Successful Auto-relaunch (Text Mode):

```bash
webctl [example.com]> # Browser crashes
webctl> screenshot
Warning: Browser not running. Relaunching to example.com...
/tmp/webctl-screenshots/25-12-26-143052-screenshot.png
```

Successful Auto-relaunch (JSON Mode):

```bash
$ webctl screenshot --json
{
  "ok": true,
  "warning": "Browser not running. Relaunched to example.com.",
  "data": {
    "path": "/tmp/webctl-screenshots/25-12-26-143052-screenshot.png"
  }
}
```

First Relaunch (No Previous URL):

```bash
webctl> status
No browser
pid: 42976
webctl> navigate test.com
Warning: Browser not running. Relaunching...
OK
```

Relaunch Failure (Port Conflict):

```bash
webctl> navigate example.com
Warning: Browser not running. Relaunching...
Error: failed to launch browser: port 9222 already in use
Daemon shutting down.
$
```

Retry Limit Exceeded:

```bash
webctl> navigate example.com
Warning: Browser not running. Relaunching... (attempt 1/3)
# Browser crashes immediately

webctl> screenshot
Warning: Browser not running. Relaunching... (attempt 2/3)
# Browser crashes again

webctl> html
Warning: Browser not running. Relaunching... (attempt 3/3)
# Browser crashes again

webctl> console
Error: browser crash limit reached (3 relaunches in 60s)
Daemon shutting down.
$
```

Attach Mode (Cannot Relaunch):

```bash
$ webctl start --attach :9222
OK
webctl> # External browser closes
webctl> navigate example.com
Error: browser connection lost (attach mode - cannot relaunch)
Daemon shutting down.
$
```

## Failure Modes

All failure modes result in daemon shutdown (clean exit):

1. Relaunch Failure:
   - Port conflict
   - Chrome not found
   - Permission denied
   - Action: Show error, shutdown daemon

2. Retry Limit Exceeded:
   - Browser crashes repeatedly
   - 3 relaunches within 60 seconds
   - Action: Show error, shutdown daemon

3. Attach Mode Disconnect:
   - External browser closes
   - Cannot relaunch (not our process)
   - Action: Show error, shutdown daemon

No broken states - daemon always exits cleanly on unrecoverable errors.

## Implementation Notes

Browser Connection Detection:

Multiple signals for browser disconnection:
- CDP WebSocket close event
- Failed CDP commands
- Empty session list
- CDP ping/pong timeout

Any of these triggers "not connected" state.

Warning Message Format:

Text mode (stderr, colored if terminal):
```
Warning: Browser not running. Relaunching to example.com...
```

With attempt counter during retry window:
```
Warning: Browser not running. Relaunching... (attempt 2/3)
```

Color: Yellow or cyan (warning level, not error)

Timeout Consistency:

Browser relaunch timeout matches initial browser launch timeout (currently 30-60 seconds based on CDP default).

URL Restoration Timing:

Navigation after relaunch uses same timeout as regular navigate command. If navigation fails, browser is still running so command can proceed (though may fail due to blank page).

## Security Considerations

No new security concerns introduced:
- Still using same browser launch mechanism
- No external input in auto-relaunch decision
- Retry limiting prevents resource exhaustion
- Daemon shutdown on failure prevents zombie states

## Updates

### 2025-12-27: Decision Reversed - Fail-Fast Approach

After initial implementation and testing review, the auto-relaunch feature was deemed too complex for the value it provides. The implementation required ~1200+ lines of changes across 33 files, including:
- Complex retry limiting with sliding time windows
- URL restoration logic
- Warning message propagation through IPC protocol
- Double-checked locking for thread safety
- Error detection and retry wrapper for every CDP call

**New Decision: Fail-Fast**

When browser connection is lost, daemon will:
1. Detect disconnection on next command
2. Clear session state
3. Show clear error: "browser connection lost - daemon shutting down"
4. Exit daemon cleanly

User must manually restart with `webctl start`.

**Implementation (~50 lines vs. 1200+):**

Added to `internal/daemon/daemon.go`:
```go
// browserConnected checks if the browser is currently running and connected.
func (d *Daemon) browserConnected() bool {
	if d.browser == nil || d.cdp == nil {
		return false
	}
	return d.sessions.Count() > 0
}

// requireBrowser checks if the browser is connected.
// If not connected, it triggers daemon shutdown and returns an error response.
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

Added to `internal/daemon/session.go`:
```go
// Clear removes all sessions and resets the manager state.
func (m *SessionManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions = make(map[string]*session)
	m.activeID = ""
	m.order = nil
}
```

Integration: Added `requireBrowser()` check at start of all 16 command handlers requiring browser connection.

**Connection Error Detection:**

The initial implementation had a race condition: `requireBrowser()` would pass, but the CDP connection could be dead (e.g., from `kill -9`). The CDP client object remains non-nil even when closed.

Added wrapper to detect connection errors during CDP calls:
```go
// isConnectionError checks if an error indicates a CDP connection failure.
func (d *Daemon) isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "client is closed") ||
		strings.Contains(s, "client closed while waiting") ||
		strings.Contains(s, "failed to send request")
}

// sendToSession wraps cdp.SendToSession with connection error detection.
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

All 34 `d.cdp.SendToSession()` calls replaced with `d.sendToSession()`.

**Behavior:**
- Manual browser close: Detected immediately by `requireBrowser()` → clean shutdown
- `kill -9` browser: Detected on first CDP call → connection error → clean shutdown

**Rationale:**

- **Simplicity**: Fail-fast is easy to understand and reason about
- **Clear UX**: Error message tells user exactly what to do
- **No Edge Cases**: No retry limits, race conditions, or URL restoration to handle
- **Maintainability**: Minimal code means minimal bugs
- **Browser crashes are rare**: Complex recovery for rare scenarios adds unnecessary complexity
- **Two-layer detection**: Proactive check + reactive error handling = robust

This aligns with Go's philosophy: simple, explicit error handling over implicit magic.
