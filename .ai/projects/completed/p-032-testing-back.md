# P-032: Testing back Command

- Status: Completed
- Started: 2025-12-31
- Completed: 2026-01-07

## Overview

Test the webctl back command which navigates to the previous page in browser history. Mirrors browser back button functionality.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-back.sh
```

## Code References

- internal/cli/back.go

## Command Signature

```
webctl back [--wait] [--timeout ms]
```

Flags:
- --wait: Wait for page load completion (default: false)
- --timeout <ms>: Timeout in milliseconds with --wait (default: 30000)
- --debug: Enable debug output (global flag)
- --json: Output in JSON format (global flag)
- --no-color: Disable colored output (global flag)

## Test Checklist

Basic back navigation:
- [ ] Navigate to page A, then page B, then back
- [ ] Verify returns to page A
- [ ] Verify immediate return (no wait)
- [ ] Verify navigation happens in background

Wait functionality:
- [ ] Back with --wait flag
- [ ] Verify blocks until page loads
- [ ] Test custom timeout (--timeout 60000)

History traversal:
- [ ] Navigate through 3+ pages, then back multiple times
- [ ] Verify correct history order
- [ ] Combine back with forward
- [ ] Verify browser state restored

Back responses:
- [ ] Default text output (OK)
- [ ] JSON output includes URL and title (--json)
- [ ] No color output (--no-color)

Various scenarios:
- [ ] Back from static page to static page
- [ ] Back with JavaScript state changes
- [ ] Back with form data (verify restored or lost)
- [ ] Back with hash navigation (#section)

Error cases:
- [ ] Back when no previous page (fresh start)
- [ ] Back at beginning of history
- [ ] Verify appropriate error message
- [ ] Back when daemon not running

Common patterns:
- [ ] navigate A && navigate B && back
- [ ] back && ready
- [ ] back --wait && html

CLI vs REPL:
- [ ] CLI: webctl back
- [ ] REPL: back

## Notes

- Returns error if no previous page in history
- Behaves like browser back button
- May restore cached page state or reload depending on browser
- Returns immediately unless --wait specified
- JSON output includes destination URL and title

## Issues Discovered

(Issues will be documented here during testing)
