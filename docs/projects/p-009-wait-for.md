# P-009: Wait-For & Synchronisation

- Status: Proposed
- Started: -

## Overview

Implement the wait-for command for synchronisation - waiting for elements to appear, network to settle, or custom conditions to be met.

## Goals

1. Wait for CSS selector to match an element
2. Wait for network activity to settle
3. Wait for custom JavaScript condition
4. Configurable timeouts

## Scope

In Scope:

- `wait-for <selector>` - element presence
- `wait-for --network-idle` - no pending requests
- `wait-for --eval "condition"` - JS condition
- `--timeout` flag for all modes
- Polling-based implementation

Out of Scope:

- Event-based waiting (optimisation for future)
- Complex conditions (element visible, enabled, etc.)

## Success Criteria

- [ ] `webctl wait-for ".loaded"` waits for element
- [ ] `webctl wait-for --network-idle` waits for no pending requests
- [ ] `webctl wait-for --eval "window.ready === true"` waits for condition
- [ ] `--timeout 10` times out after 10 seconds
- [ ] Returns success when condition met
- [ ] Returns error on timeout

## Deliverables

- `cmd/webctl/waitfor.go`
- Daemon-side wait-for handler

## Technical Design

### Command Syntax

```bash
# Wait for element
webctl wait-for ".content-loaded"
webctl wait-for "#results" --timeout 30

# Wait for network idle
webctl wait-for --network-idle
webctl wait-for --network-idle --timeout 10

# Wait for JS condition
webctl wait-for --eval "document.readyState === 'complete'"
webctl wait-for --eval "window.appLoaded === true" --timeout 60
```

### Output

Success:

```json
{"ok": true, "waited": 1.234}  // seconds waited
```

Timeout:

```json
{"ok": false, "error": "timeout waiting for: .content-loaded", "waited": 30.0}
```

### Element Wait Implementation

Polling approach:

```go
func waitForSelector(selector string, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        found, err := querySelector(selector)
        if err != nil {
            return err
        }
        if found {
            return nil
        }
        time.Sleep(100 * time.Millisecond)  // poll interval
    }
    return ErrTimeout
}
```

CDP: `DOM.querySelector` in a loop.

### Network Idle Implementation

Track pending requests:

```go
type NetworkTracker struct {
    pending map[string]bool  // requestId -> true
    mu      sync.Mutex
}

func (t *NetworkTracker) OnRequestSent(id string) {
    t.mu.Lock()
    t.pending[id] = true
    t.mu.Unlock()
}

func (t *NetworkTracker) OnLoadingFinished(id string) {
    t.mu.Lock()
    delete(t.pending, id)
    t.mu.Unlock()
}

func (t *NetworkTracker) IsIdle() bool {
    t.mu.Lock()
    defer t.mu.Unlock()
    return len(t.pending) == 0
}
```

Wait for idle:

```go
func waitForNetworkIdle(timeout time.Duration, idleTime time.Duration) error {
    deadline := time.Now().Add(timeout)
    idleStart := time.Time{}

    for time.Now().Before(deadline) {
        if tracker.IsIdle() {
            if idleStart.IsZero() {
                idleStart = time.Now()
            } else if time.Since(idleStart) >= idleTime {
                return nil  // Idle for long enough
            }
        } else {
            idleStart = time.Time{}  // Reset
        }
        time.Sleep(50 * time.Millisecond)
    }
    return ErrTimeout
}
```

Default idle time: 500ms of no network activity.

### JS Condition Implementation

```go
func waitForCondition(expression string, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        result, err := evaluate(expression)
        if err != nil {
            return err
        }
        if isTruthy(result) {
            return nil
        }
        time.Sleep(100 * time.Millisecond)
    }
    return ErrTimeout
}
```

CDP: `Runtime.evaluate` in a loop.

### Configuration

| Parameter | Default | Description |
|-----------|---------|-------------|
| timeout | 30s | Maximum wait time |
| poll-interval | 100ms | Time between checks |
| network-idle-time | 500ms | How long network must be quiet |

### Timeout Handling

- Default timeout: 30 seconds
- `--timeout 0` means wait forever (not recommended)
- On timeout, return error with time waited

## CDP Methods Used

| Mode | CDP Methods |
|------|-------------|
| selector | `DOM.getDocument`, `DOM.querySelector` |
| network-idle | (internal tracking of Network events) |
| eval | `Runtime.evaluate` |

## Dependencies

- P-008 (Navigation & Interaction)

## Testing Strategy

1. **Integration tests** - Test page with delayed element appearance, network requests

## Notes

Polling is simple and reliable. Event-based waiting (MutationObserver) is more efficient but adds complexity. Start with polling; optimise later if needed.

Consider future enhancements:

- `--visible` - wait for element to be visible (not just present)
- `--hidden` - wait for element to disappear
- `--enabled` - wait for element to be enabled
- `--text "foo"` - wait for element to contain text
