# DR-020: Ready Command Extensions

- Date: 2025-12-25
- Status: Accepted
- Category: CLI

## Problem

The current `ready` command only supports waiting for page load (`loadEventFired`). AI agents automating web applications need more sophisticated synchronization primitives:

- SPAs with client-side routing don't fire load events for route transitions
- Dynamic content loads after page load completes
- Form submissions may trigger network requests without page navigation
- Custom application initialization states need to be detected

Requirements:

- Wait for CSS selectors (element presence)
- Wait for network idle (no pending requests)
- Wait for custom JavaScript conditions (application state)
- Maintain backwards compatibility with existing page load behavior
- Configurable timeouts for all modes

## Decision

Extend the `ready` command to support four synchronization modes:

| Mode | Syntax | Description |
|------|--------|-------------|
| Page load | `ready [--timeout 30s]` | Wait for page load (existing, default) |
| Selector | `ready <selector> [--timeout 30s]` | Wait for element to appear |
| Network idle | `ready --network-idle [--timeout 30s]` | Wait for no pending requests (500ms quiet) |
| JS condition | `ready --eval "expr" [--timeout 30s]` | Wait for JS expression to be truthy |

Mode detection logic (in order):

```go
if networkIdle {
    // Network idle mode
} else if evalExpression != "" {
    // JS condition mode
} else if selector != "" {
    // Element selector mode
} else {
    // Default: page load mode (existing behavior)
}
```

All modes support the `--timeout` flag (default: 30s).

## Why

Unified command for all synchronization needs:

A single `ready` command is easier to learn and remember than multiple commands (`ready`, `wait-for`, `wait-selector`, `wait-network`). Users learn one command with multiple modes instead of several commands.

Polling-based implementation:

Polling is simple, reliable, and easier to implement than event-based waiting:

- Element presence: Query DOM in a loop (100ms interval)
- Network idle: Check pending request count in a loop (50ms interval)
- JS condition: Evaluate expression in a loop (100ms interval)

Event-based waiting (MutationObserver, Network event tracking) is more efficient but adds complexity. Start simple; optimize later if needed.

Network idle threshold:

Network must be quiet for 500ms to be considered "idle". This prevents premature completion during rapid request sequences (redirects, chunked responses, etc.).

Backwards compatibility:

The existing `ready` command behavior is preserved as the default:

```bash
ready                  # Still waits for page load
ready --timeout 5s     # Still waits for page load with custom timeout
```

## Trade-offs

Accept:

- Polling adds slight overhead (50-100ms intervals)
- Network idle mode requires Network domain to be enabled (lazy init)
- Element presence only checks existence, not visibility/interactivity (v1)
- Cannot combine modes (e.g., can't wait for selector AND network idle)

Gain:

- Simple, reliable implementation
- Works consistently across all browser states
- Easy to understand and debug
- Backwards compatible
- Composable (chain multiple ready commands if needed)

## Alternatives

Separate wait-for command:

Create a new `wait-for` command for advanced waiting, keep `ready` for page load only.

- Pro: Cleaner separation of concerns
- Pro: Backwards compatible by default
- Con: Users must learn two commands
- Con: Less intuitive ("ready" vs "wait-for" distinction unclear)
- Rejected: Unified command simpler for users

Event-based waiting:

Use MutationObserver for elements, Network event tracking for idle.

- Pro: More efficient, no polling
- Pro: Instant response when condition met
- Con: More complex implementation
- Con: Harder to debug
- Con: Potential edge cases with event timing
- Rejected: Polling simpler for v1, can optimize later

Combined modes:

Allow combining modes: `ready .element --network-idle`.

- Pro: More flexible
- Pro: Single wait for multiple conditions
- Con: Complex mode interaction logic
- Con: Unclear semantics (AND vs OR?)
- Rejected: Keep modes mutually exclusive for simplicity

## Implementation

### IPC Protocol

Update `ReadyParams` structure:

```go
type ReadyParams struct {
    Timeout     int    `json:"timeout"`      // milliseconds
    Selector    string `json:"selector"`     // CSS selector (optional)
    NetworkIdle bool   `json:"networkIdle"`  // wait for network idle
    Eval        string `json:"eval"`         // JS expression (optional)
}
```

### CLI Command

Update flags and arguments:

```go
readyCmd.Flags().Duration("timeout", 30*time.Second, "Maximum time to wait")
readyCmd.Flags().Bool("network-idle", false, "Wait for network to be idle (500ms of no activity)")
readyCmd.Flags().String("eval", "", "JavaScript expression to evaluate")
```

Arguments:

- Zero arguments: page load mode (existing)
- One argument: selector mode

### Daemon Handlers

Extend `handleReady` with mode detection and delegation:

```go
func (d *Daemon) handleReady(req ipc.Request) ipc.Response {
    // Parse params
    var params ipc.ReadyParams
    // ... unmarshal logic ...

    // Mode detection
    if params.NetworkIdle {
        return d.handleReadyNetworkIdle(activeID, timeout)
    } else if params.Eval != "" {
        return d.handleReadyEval(activeID, params.Eval, timeout)
    } else if params.Selector != "" {
        return d.handleReadySelector(activeID, params.Selector, timeout)
    } else {
        // Existing page load logic
        return d.handleReadyPageLoad(activeID, timeout)
    }
}
```

### Element Waiting

Poll for selector with `DOM.querySelector`:

```go
func (d *Daemon) handleReadySelector(sessionID, selector string, timeout time.Duration) ipc.Response {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ipc.ErrorResponse(fmt.Sprintf("timeout waiting for: %s", selector))
        case <-ticker.C:
            found, err := d.querySelector(ctx, sessionID, selector)
            if err != nil {
                return ipc.ErrorResponse(err.Error())
            }
            if found {
                return ipc.SuccessResponse(nil)
            }
        }
    }
}
```

### Network Idle Waiting

Track pending requests and poll for idle state:

```go
func (d *Daemon) handleReadyNetworkIdle(sessionID string, timeout time.Duration) ipc.Response {
    // Ensure Network domain is enabled
    if err := d.ensureNetworkEnabled(sessionID); err != nil {
        return ipc.ErrorResponse(err.Error())
    }

    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    idleThreshold := 500 * time.Millisecond
    ticker := time.NewTicker(50 * time.Millisecond)
    defer ticker.Stop()

    var idleStart time.Time

    for {
        select {
        case <-ctx.Done():
            return ipc.ErrorResponse("timeout waiting for network idle")
        case <-ticker.C:
            pending := d.getPendingRequestCount(sessionID)
            if pending == 0 {
                if idleStart.IsZero() {
                    idleStart = time.Now()
                } else if time.Since(idleStart) >= idleThreshold {
                    return ipc.SuccessResponse(nil)
                }
            } else {
                idleStart = time.Time{} // Reset
            }
        }
    }
}
```

Network tracking uses existing buffer infrastructure:

```go
func (d *Daemon) getPendingRequestCount(sessionID string) int {
    // Count requests in networkBuf where LoadingFinished or LoadingFailed
    // has not been received (status == "pending")
    count := 0
    d.networkBuf.ForEach(func(entry ipc.NetworkEntry) bool {
        if entry.Type == "requestWillBeSent" {
            // Check if corresponding finished/failed event exists
            // If not, it's still pending
            // (This is simplified; actual implementation needs request ID tracking)
            count++
        }
        return true
    })
    return count
}
```

Better approach: Maintain a pending requests map in SessionManager:

```go
type SessionManager struct {
    // ... existing fields ...
    pendingRequests sync.Map // map[sessionID]map[requestID]bool
}
```

### JS Condition Waiting

Poll for expression evaluation:

```go
func (d *Daemon) handleReadyEval(sessionID, expression string, timeout time.Duration) ipc.Response {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ipc.ErrorResponse(fmt.Sprintf("timeout waiting for: %s", expression))
        case <-ticker.C:
            result, err := d.cdp.SendToSession(ctx, sessionID, "Runtime.evaluate", map[string]any{
                "expression":    expression,
                "returnByValue": true,
            })
            if err != nil {
                // Don't fail on evaluation error, keep polling
                continue
            }

            var resp struct {
                Result struct {
                    Value any `json:"value"`
                } `json:"result"`
            }
            if err := json.Unmarshal(result, &resp); err != nil {
                continue
            }

            // Check if truthy
            if isTruthy(resp.Result.Value) {
                return ipc.SuccessResponse(nil)
            }
        }
    }
}

func isTruthy(value any) bool {
    if value == nil {
        return false
    }
    switch v := value.(type) {
    case bool:
        return v
    case string:
        return v != ""
    case float64:
        return v != 0
    default:
        return true
    }
}
```

### Help Text

Comprehensive help following the select command style:

```
Waits for the page or application to be ready before continuing.

Supports multiple synchronization modes for different use cases:
- Page load (default): waits for browser load event
- Element presence: waits for CSS selector to match
- Network idle: waits for all network requests to complete
- JS condition: waits for custom JavaScript expression to be true

Only works with a single mode at a time. For complex conditions,
chain multiple ready commands or use custom JavaScript.

Page load mode (default):
  Checks document.readyState first - if already "complete", returns
  immediately. Otherwise, waits for the browser's load event to fire.

  This is useful after navigation to ensure all resources (images,
  scripts, stylesheets) have fully loaded.

Selector mode:
  Waits for an element matching the CSS selector to appear in the DOM.
  Only checks presence, not visibility or interactivity (use eval for that).

  Useful for dynamic content loading, SPAs, or lazy-loaded components.

Network idle mode:
  Waits for all pending network requests to complete and the network
  to be quiet for 500ms. Useful after form submissions, AJAX requests,
  or API calls.

Eval mode:
  Waits for a custom JavaScript expression to evaluate to a truthy value.
  Most flexible option for application-specific ready states.

Timeout:
  --timeout duration    Maximum time to wait (default 30s)
                        Accepts Go duration format: 10s, 1m, 500ms

Examples:
  # Page load mode - wait for full page load
  ready
  ready --timeout 10s

  # Selector mode - wait for element to appear
  ready ".content-loaded"              # Wait for element with class
  ready "#dashboard"                   # Wait for element with ID
  ready "[data-loaded=true]"           # Wait for attribute
  ready "button.submit:enabled"        # Wait for enabled button

  # Network idle mode - wait for requests to complete
  ready --network-idle                 # Default 30s timeout
  ready --network-idle --timeout 60s   # Longer timeout for slow APIs

  # Eval mode - wait for custom condition
  ready --eval "document.readyState === 'complete'"
  ready --eval "window.appReady === true"
  ready --eval "document.querySelector('.error') === null"
  ready --eval "fetch('/api/health').then(r => r.ok)"

Common patterns:
  # Navigate and wait for page load
  navigate example.com
  ready

  # SPA navigation - wait for route content
  click ".nav-dashboard"
  ready "#dashboard-content"

  # Form submission - wait for success message
  click "#submit"
  ready ".success-message"

  # API call - wait for network idle
  click "#load-data"
  ready --network-idle

  # Complex initialization - wait for app state
  navigate app.example.com
  ready --eval "window.app && window.app.initialized"

  # Dynamic content loading
  scroll "#load-more"
  ready --network-idle
  ready ".new-items"

  # Chaining multiple conditions
  ready                               # Page load
  ready --network-idle                # Then network idle
  ready --eval "window.dataLoaded"    # Then custom state

When to use each mode:
  - Page load: Full page navigation, browser reload
  - Selector: Dynamic content, SPA routes, lazy loading
  - Network idle: Form submissions, AJAX calls, API requests
  - Eval: Custom app states, complex conditions, visibility checks

For SPAs with client-side routing (React Router, Vue Router, etc.),
the page load event may not fire. Use selector or eval modes instead.

Error cases:
  - "timeout waiting for: <condition>" - condition not met within timeout
  - "no active session" - no browser page is open
  - "element not found" (in eval mode) - JS expression threw error
```

## Testing Strategy

Integration tests:

1. Page load mode:
   - ready on already-loaded page (immediate return)
   - ready during navigation (wait for load)
   - ready with timeout on slow page (error)

2. Selector mode:
   - ready with immediate element (quick return)
   - ready with delayed element (JS timeout creates element)
   - ready with non-existent element (timeout error)

3. Network idle mode:
   - ready on idle page (immediate return)
   - ready with pending requests (wait for completion)
   - ready with continuous requests (timeout error)

4. Eval mode:
   - ready with true expression (immediate return)
   - ready with delayed condition (JS timeout sets flag)
   - ready with always-false condition (timeout error)
   - ready with invalid expression (continue polling, timeout)

Test page setup:

```html
<!DOCTYPE html>
<html>
<head><title>Ready Test Page</title></head>
<body>
  <button id="load-element">Load Element</button>
  <button id="trigger-request">Trigger Request</button>
  <button id="set-flag">Set Flag</button>

  <script>
    // Test selector mode
    document.getElementById('load-element').onclick = () => {
      setTimeout(() => {
        const div = document.createElement('div');
        div.className = 'dynamic-content';
        div.textContent = 'Loaded!';
        document.body.appendChild(div);
      }, 2000);
    };

    // Test network idle mode
    document.getElementById('trigger-request').onclick = () => {
      fetch('/slow-endpoint');  // Takes 3 seconds
    };

    // Test eval mode
    window.appReady = false;
    document.getElementById('set-flag').onclick = () => {
      setTimeout(() => {
        window.appReady = true;
      }, 1500);
    };
  </script>
</body>
</html>
```

## Future Enhancements

Deferred from initial implementation:

Visibility checking:

```bash
ready .element --visible
```

Wait for element to be visible, not just present in DOM.

Element state checking:

```bash
ready button --enabled
ready .element --hidden
```

Wait for specific element states.

Text content matching:

```bash
ready .message --text "Success"
```

Wait for element with specific text content.

Combined modes (AND logic):

```bash
ready .element --network-idle
```

Wait for both selector and network idle.

Custom network idle threshold:

```bash
ready --network-idle --idle-time 1s
```

Configure how long network must be quiet.

Event-based optimization:

Replace polling with MutationObserver (selector) and Network event tracking (idle) for better performance.

## Updates

- 2025-12-25: Initial version
