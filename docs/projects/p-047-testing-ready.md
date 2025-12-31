# P-047: Testing ready Command

- Status: In Progress
- Started: 2025-12-31

## Overview

Test the webctl ready command which waits for the page or application to be ready before continuing. Supports multiple synchronization modes: page load (default), element presence (selector), network idle, and custom JavaScript condition. Only works with one mode at a time.

## Code References

- internal/cli/ready.go

## Command Signature

```
webctl ready [selector]
```

Arguments:
- selector: Optional CSS selector to wait for element presence

Flags:
- --timeout: Maximum time to wait (default 60s, accepts Go duration format)
- --network-idle: Wait for network to be idle (500ms of no activity)
- --eval: JavaScript expression to evaluate (wait for truthy value)

Global flags:
- --json: JSON output format
- --no-color: Disable color output
- --debug: Enable debug output

## Test Checklist

Page load mode (default):
- [ ] ready (wait for page load)
- [ ] ready --timeout 10s (with custom timeout)
- [ ] ready after navigation (ensure page fully loaded)
- [ ] ready on already loaded page (should return immediately)
- [ ] Verify checks document.readyState first
- [ ] Verify waits for load event if not complete

Selector mode (wait for element):
- [ ] ready ".content-loaded" (wait for class)
- [ ] ready "#dashboard" (wait for ID)
- [ ] ready "[data-loaded=true]" (wait for attribute)
- [ ] ready "button.submit:enabled" (wait for enabled button)
- [ ] ready "div.dynamic-content" (wait for dynamic content)
- [ ] Verify only checks presence, not visibility

Network idle mode:
- [ ] ready --network-idle (default 60s timeout)
- [ ] ready --network-idle --timeout 120s (custom timeout)
- [ ] ready --network-idle after form submission
- [ ] ready --network-idle after AJAX call
- [ ] Verify waits for 500ms of no network activity

Eval mode (custom condition):
- [ ] ready --eval "document.readyState === 'complete'"
- [ ] ready --eval "window.appReady === true"
- [ ] ready --eval "document.querySelector('.error') === null"
- [ ] ready --eval "!!document.querySelector('#loaded')"
- [ ] ready --eval "document.querySelectorAll('img').length > 0"

Timeout handling:
- [ ] ready --timeout 5s (short timeout)
- [ ] ready "#nonexistent" --timeout 2s (should timeout)
- [ ] ready --network-idle --timeout 30s
- [ ] Verify timeout error message includes condition

Common navigation patterns:
- [ ] navigate example.com && ready (navigate then wait)
- [ ] navigate example.com && ready ".content" (wait for element)
- [ ] navigate example.com && ready --network-idle (wait for network)
- [ ] click ".nav-link" && ready "#new-content" (SPA navigation)

Form submission patterns:
- [ ] click "#submit" && ready ".success-message" (wait for success)
- [ ] click "#submit" && ready --network-idle (wait for API call)
- [ ] type and click then ready for response

Dynamic content patterns:
- [ ] scroll "#load-more" && ready --network-idle (infinite scroll)
- [ ] scroll "#load-more" && ready ".new-items" (wait for new content)
- [ ] click "#load-data" && ready --eval "window.dataLoaded" (custom state)

SPA patterns:
- [ ] ready after client-side routing (React Router, Vue Router)
- [ ] ready --eval for custom app initialization state
- [ ] Verify page load event may not fire for SPA routes

Chaining conditions:
- [ ] ready (page load)
- [ ] ready --network-idle (then network idle)
- [ ] ready --eval "window.dataLoaded" (then custom state)
- [ ] Verify multiple ready commands in sequence work

Error cases:
- [ ] ready "#nonexistent" --timeout 2s (timeout error)
- [ ] ready --network-idle --timeout 1s on slow page (timeout)
- [ ] ready --eval "false" --timeout 2s (condition never true)
- [ ] ready with no active session (error message)
- [ ] ready with daemon not running (error message)

Mode conflicts (should only accept one mode):
- [ ] Verify selector and --network-idle can't be combined
- [ ] Verify selector and --eval can't be combined
- [ ] Verify --network-idle and --eval can't be combined

Output formats:
- [ ] Default text output (just OK)
- [ ] --json output format
- [ ] --no-color output
- [ ] --debug verbose output
- [ ] Timeout error format

Long-running operations:
- [ ] ready --timeout 120s on slow loading page
- [ ] ready --network-idle on page with many resources
- [ ] ready --eval on condition that takes time to become true

CLI vs REPL:
- [ ] CLI: webctl ready
- [ ] CLI: webctl ready ".content"
- [ ] CLI: webctl ready --network-idle
- [ ] CLI: webctl ready --eval "window.ready === true"
- [ ] REPL: ready
- [ ] REPL: ready ".content"
- [ ] REPL: ready --network-idle
- [ ] REPL: ready --eval "window.ready === true"

## Notes

- Only one synchronization mode allowed at a time
- Page load mode checks document.readyState first, returns immediately if complete
- Network idle waits for 500ms of no network activity
- Eval mode polls custom JavaScript expression until truthy
- Default timeout is 60 seconds
- Useful after navigation, form submission, AJAX calls, SPA routing
- For SPAs with client-side routing, page load event may not fire (use selector or eval)

## Issues Discovered

(Issues will be documented here during testing)
