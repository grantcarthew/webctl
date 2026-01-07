# P-033: Testing forward Command

- Status: Completed
- Started: 2025-12-31
- Completed: 2026-01-07

## Overview

Test the webctl forward command which navigates to the next page in browser history. Mirrors browser forward button functionality.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-forward.sh
```

## Code References

- internal/cli/forward.go

## Command Signature

```
webctl forward [--wait] [--timeout ms]
```

Flags:
- --wait: Wait for page load completion (default: false)
- --timeout <ms>: Timeout in milliseconds with --wait (default: 30000)
- --debug: Enable debug output (global flag)
- --json: Output in JSON format (global flag)
- --no-color: Disable colored output (global flag)

## Test Checklist

Basic forward navigation:
- [ ] Navigate to page A, page B, back, then forward
- [ ] Verify returns to page B
- [ ] Verify immediate return (no wait)
- [ ] Verify navigation happens in background

Wait functionality:
- [ ] Forward with --wait flag
- [ ] Verify blocks until page loads
- [ ] Test custom timeout (--timeout 60000)

History traversal:
- [ ] Navigate through 3+ pages, back 2 times, forward 2 times
- [ ] Verify correct history order
- [ ] Combine forward with back
- [ ] Verify browser state restored

Forward responses:
- [ ] Default text output (OK)
- [ ] JSON output includes URL and title (--json)
- [ ] No color output (--no-color)

Various scenarios:
- [ ] Forward from static page to static page
- [ ] Forward with JavaScript state changes
- [ ] Forward with form data
- [ ] Forward with hash navigation (#section)

Forward history cleared:
- [ ] Navigate A, B, back to A, navigate to C
- [ ] Verify forward from A fails (history to B cleared)
- [ ] Verify appropriate error message

Error cases:
- [ ] Forward when no next page
- [ ] Forward at end of history
- [ ] Forward when daemon not running
- [ ] Verify appropriate error messages

Common patterns:
- [ ] back && forward
- [ ] forward && ready
- [ ] forward --wait && html

CLI vs REPL:
- [ ] CLI: webctl forward
- [ ] REPL: forward

## Notes

- Returns error if no next page in history
- Behaves like browser forward button
- Forward history cleared when navigating from intermediate point
- May restore cached page state or reload depending on browser
- Returns immediately unless --wait specified
- JSON output includes destination URL and title

## Issues Discovered

(Issues will be documented here during testing)
