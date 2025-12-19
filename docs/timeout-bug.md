# BUG-003: HTML Command Timeout During Page Navigation

## Status: ONGOING

The navigation blocking approach via CDP events is not working reliably. This document captures all debugging findings for future reference.

## Summary

The `html` command times out (30 seconds) when called immediately after `navigate` to complex pages. The root cause is that Chrome's `Runtime.evaluate` CDP method blocks until the page is ready for JavaScript execution.

## Symptoms

```bash
# This sequence times out
./webctl navigate https://abc.net.au/
./webctl html  # Times out after 30 seconds
```

Error message:
```json
{"error":"timeout waiting for page load - try 'webctl ready' first","ok":false}
```

## Root Cause Analysis

### CDP Event Timing

When navigating to a page, Chrome fires several events in sequence:

1. `Page.frameNavigated` - Navigation committed (URL changed, new document created)
2. `Page.domContentEventFired` - DOM is ready (DOMContentLoaded)
3. `Page.loadEventFired` - Page fully loaded (window.onload)

The `navigate` command returns after `Page.frameNavigated`. However, `Runtime.evaluate` blocks internally until the page is ready.

### The Blocking Behavior

**Key Finding**: `Runtime.evaluate` blocks until Chrome considers the page ready for JavaScript execution.

Even simple synchronous JavaScript like:
```javascript
document.documentElement.outerHTML
```

Will block if called between `Page.frameNavigated` and when the page is ready.

### Timing Measurements (2024-12-19)

From debug logs with example.com:
```
[DEBUG] handleNavigate called
[DEBUG] navigate: created navigating channel for session BE1E0F6D52309D86FD613ED480FB1D06
[DEBUG] Target.targetInfoChanged: url="https://example.com/"
[DEBUG] handleHTML called
[DEBUG] html: calling Runtime.evaluate
[DEBUG] Page.domContentEventFired: sessionID=BE1E0F6D52309D86FD613ED480FB1D06
[DEBUG] Page.loadEventFired: sessionID=BE1E0F6D52309D86FD613ED480FB1D06
[DEBUG] html: Runtime.evaluate completed in 15.416424916s
```

**Observations**:
- `Runtime.evaluate` takes 15+ seconds for example.com in headless mode
- The CDP events (`domContentEventFired`, `loadEventFired`) fire DURING the blocked `Runtime.evaluate` call
- Once the page is loaded, subsequent `html` calls are instant (14ms)

---

## Approaches Tried

### Approach 1: Navigation Channel with CDP Events (FAILED)

Wait for `Page.loadEventFired` or `Page.domContentEventFired` to close a navigation channel.

**Implementation**:
```go
// On navigate: create channel
navDoneCh := make(chan struct{})
d.navigating.Store(activeID, navDoneCh)

// On loadEventFired: close channel
if ch, ok := d.navigating.LoadAndDelete(evt.SessionID); ok {
    close(ch.(chan struct{}))
}

// In handleHTML: wait for channel
if navCh, ok := d.navigating.Load(activeID); ok {
    select {
    case <-navCh.(chan struct{}):
        // proceed
    case <-time.After(30 * time.Second):
        return error
    }
}
```

**Result**: FAILED - The `loadEventFired` event fires, but the navigation channel is never closed in practice. Possible causes:
- Race condition between storing channel and receiving event
- Event sessionID mismatch
- Event not being received for some pages

### Approach 2: Promise-Based JavaScript Wait (FAILED)

Use JavaScript Promise that waits for DOMContentLoaded:

```javascript
(function() {
    return new Promise((resolve) => {
        const getHTML = () => resolve(document.documentElement.outerHTML);
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', getHTML);
        } else {
            getHTML();
        }
    });
})()
```

With `awaitPromise: true` in `Runtime.evaluate`.

**Result**: FAILED - `Runtime.evaluate` itself blocks before the Promise can execute. The blocking happens at the Chrome level, not in JavaScript.

### Approach 3: Direct JavaScript (SLOW)

Just call `Runtime.evaluate` directly without any waiting:

```go
result, err := d.cdp.SendToSession(ctx, activeID, "Runtime.evaluate", map[string]any{
    "expression":    `document.documentElement.outerHTML`,
    "returnByValue": true,
})
```

**Result**: WORKS but SLOW - Takes 15-20 seconds for example.com because Chrome blocks the response.

---

## How Rod/Snag Works (Reference)

### Snag's Fetch Flow

```go
// 1. Navigate (returns after HTTP headers, fast)
err := pf.page.Timeout(pf.timeout).Navigate(opts.URL)

// 2. WaitStable (3-second timeout for stabilization)
err = pf.page.WaitStable(StabilizeTimeout)  // StabilizeTimeout = 3 * time.Second
if err != nil {
    logger.Warning("Page did not stabilize: %v", err)
    // Continues anyway!
}

// 3. Get HTML
html, err := pf.page.HTML()
```

### Rod's WaitStable Implementation

```go
func (p *Page) WaitStable(d time.Duration) error {
    utils.All(func() {
        p.WaitLoad()           // JavaScript Promise wait
    }, func() {
        p.WaitRequestIdle(d, nil, nil, nil)()  // Network idle
    }, func() {
        p.WaitDOMStable(d, 0)  // DOM stops changing
    })()
    return err
}
```

### Rod's Key Insight: Retry on Context Errors

From `context/rod/page_eval.go`:

```go
func (p *Page) Evaluate(opts *EvalOptions) (res *proto.RuntimeRemoteObject, err error) {
    var backoff utils.Sleeper

    // js context will be invalid if a frame is reloaded or not ready
    for {
        res, err = p.evaluate(opts)
        if err != nil && errors.Is(err, cdp.ErrCtxNotFound) {
            if backoff == nil {
                backoff = utils.BackoffSleeper(30*time.Millisecond, 3*time.Second, nil)
            } else {
                _ = backoff(p.ctx)
            }
            p.unsetJSCtxID()
            continue  // RETRY
        }
        return
    }
}
```

Rod RETRIES on `ErrCtxNotFound` with exponential backoff (30ms to 3s).

### Rod's HTML() Implementation

```go
func (p *Page) HTML() (string, error) {
    el, err := p.Element("html")  // Uses ElementByJS with retry
    if err != nil {
        return "", err
    }
    return el.HTML()
}
```

### Key Differences from webctl

1. **Snag creates a NEW page for each fetch** via `browser.Page(proto.TargetCreateTarget{})`, not reusing existing pages
2. **WaitStable has a short timeout** (3 seconds) and continues anyway if it fails
3. **Retry on context errors** instead of blocking waiting for CDP events
4. **Navigate returns early** (after HTTP headers), not after `frameNavigated`

---

## Important Observations

### 1. Snag Also Times Out on Heavy Pages

When starting fresh (no cached pages), snag also times out on abc.net.au:

```bash
$ time snag -f html https://abc.net.au/
✓ Chrome launched in headless mode
Fetching https://abc.net.au/...
✗ Page load timeout exceeded (30s)
✗ The page took too long to load
  Try: snag https://abc.net.au/ --timeout 60
Error: page load timeout exceeded

real    0m30.074s
```

### 2. "Fast" Results Were From Cached Pages

Earlier tests that showed snag working fast on abc.net.au were actually connecting to a browser that already had the page loaded. When both tools start fresh, they have similar behavior.

### 3. Headless Mode is Slower

In headless mode, `Runtime.evaluate` blocking times are much longer:
- example.com: ~15-20 seconds in headless, ~5 seconds in visible mode
- abc.net.au: >30 seconds (timeout) in both modes

### 4. Second Call is Instant

Once a page is loaded, subsequent `html` calls are instant (14ms):

```bash
./webctl html   # First call: 15-20 seconds
./webctl html   # Second call: 0.014 seconds
```

---

## CDP Event Details

| Event | Description | Has SessionID |
|-------|-------------|---------------|
| `Page.frameNavigated` | Navigation committed | Yes (in params.frame) |
| `Page.domContentEventFired` | DOMContentLoaded | Yes (in message envelope) |
| `Page.loadEventFired` | window.onload | Yes (in message envelope) |

Event params for `domContentEventFired` and `loadEventFired`:
```json
{
    "timestamp": 12345.678
}
```

The sessionID comes from the WebSocket message envelope, not the event params.

---

## Potential Fixes to Try

### Option A: Use DOM.getOuterHTML Instead of Runtime.evaluate

```go
// Get document node
doc, _ := d.cdp.SendToSession(ctx, sessionID, "DOM.getDocument", nil)
// Get outer HTML via DOM method
html, _ := d.cdp.SendToSession(ctx, sessionID, "DOM.getOuterHTML", map[string]any{
    "nodeId": rootNodeId,
})
```

**Status**: Not tested yet. May also block.

### Option B: Retry Loop with Short Timeout

```go
for attempts := 0; attempts < 10; attempts++ {
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    result, err := d.cdp.SendToSession(ctx, activeID, "Runtime.evaluate", ...)
    cancel()

    if err == nil {
        return result
    }
    time.Sleep(500 * time.Millisecond)
}
```

**Status**: Not tested yet.

### Option C: Wait for frameStoppedLoading Event

Subscribe to `Page.frameStoppedLoading` instead of/in addition to `loadEventFired`.

**Status**: Not tested yet.

### Option D: Use Page.lifecycleEvent

Enable lifecycle events and wait for specific lifecycle states:

```go
d.cdp.SendToSession(ctx, sessionID, "Page.setLifecycleEventsEnabled", map[string]any{
    "enabled": true,
})
// Then listen for "DOMContentLoaded", "load", "networkIdle" etc.
```

**Status**: Not tested yet.

### Option E: Match Rod's Approach

1. Create new page for each operation
2. Use retry with backoff on context errors
3. Don't wait for CDP events, just retry until it works

**Status**: Most promising but requires significant refactoring.

---

## Debug Commands

### Run with Debug Logging
```bash
./webctl start --headless --debug
```

### Check Navigation State
Look for these debug messages:
```
[DEBUG] navigate: created navigating channel for session XXX
[DEBUG] Page.domContentEventFired: sessionID=XXX
[DEBUG] Page.domContentEventFired: closing navigating channel for session XXX
[DEBUG] Page.loadEventFired: sessionID=XXX
```

### Time HTML Command
```bash
time ./webctl html
```

### Compare with Snag
```bash
time snag -f html https://example.com/
```

---

## Files Involved

- `internal/daemon/daemon.go`:
  - `handleNavigate` - creates navigation channel
  - `handleHTML` - waits for navigation channel (current approach)
  - `handleLoadEventFired` - closes navigation channel
  - `handleDOMContentEventFired` - also closes navigation channel
  - `subscribeEvents` - subscribes to CDP events

- `internal/cdp/client.go`:
  - `SendToSession` - sends CDP commands
  - Event handling and sessionID parsing

---

## Conclusion

The CDP event-based approach is unreliable. The most promising path forward is to match Rod's approach: retry on errors with backoff, and optionally create new pages for operations that need a clean context.

The fundamental issue is that Chrome's `Runtime.evaluate` blocks until the page is ready, and this blocking time is unpredictable (15+ seconds for simple pages in headless mode). No amount of event waiting fixes this - we need to either accept the blocking or use a retry-based approach.
