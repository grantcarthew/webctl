# P-030: Testing navigate Command

- Status: In Progress
- Started: 2025-12-31

## Overview

Test the webctl navigate command which loads URLs in the browser. Features automatic protocol detection and optional wait-for-load functionality.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-navigate.sh
```

## Code References

- internal/cli/navigate.go

## Command Signature

```
webctl navigate <url> [--wait] [--timeout ms]
```

Flags:
- --wait: Wait for page load completion (default: false)
- --timeout <ms>: Timeout in milliseconds with --wait (default: 30000)
- --debug: Enable debug output (global flag)
- --json: Output in JSON format (global flag)
- --no-color: Disable colored output (global flag)

## Test Checklist

Basic navigation (fast return):
- [ ] Navigate to example.com
- [ ] Navigate to google.com
- [ ] Verify immediate return (no wait)
- [ ] Verify page loads in background

URL protocol auto-detection:
- [ ] URL without protocol gets https:// (example.com)
- [ ] localhost gets http:// (localhost:3000)
- [ ] 127.0.0.1 gets http:// (127.0.0.1:8080)
- [ ] 0.0.0.0 gets http:// (0.0.0.0:5000)
- [ ] Explicit http:// preserved
- [ ] Explicit https:// preserved
- [ ] file:// protocol works

Wait functionality:
- [ ] Navigate with --wait flag
- [ ] Verify blocks until page loads
- [ ] Test custom timeout (--timeout 60000)
- [ ] Test timeout expiry on slow page

Navigation responses:
- [ ] Default text output (OK)
- [ ] JSON output includes URL and title (--json)
- [ ] No color output (--no-color)

Various page types:
- [ ] Navigate to static HTML page
- [ ] Navigate to JavaScript-heavy SPA
- [ ] Navigate to page with redirects
- [ ] Navigate to page with forms
- [ ] Navigate to page with media

Error cases:
- [ ] Invalid domain (net::ERR_NAME_NOT_RESOLVED)
- [ ] Connection refused (net::ERR_CONNECTION_REFUSED)
- [ ] Timeout with --wait (slow page)
- [ ] Malformed URL
- [ ] Navigate when daemon not running

Common workflow patterns:
- [ ] navigate example.com && ready
- [ ] navigate example.com && screenshot
- [ ] navigate example.com --wait && html

CLI vs REPL:
- [ ] CLI: webctl navigate https://example.com
- [ ] REPL: navigate https://example.com

## Notes

- Returns immediately by default for fast feedback
- Use --wait or follow with ready command for synchronous behavior
- Protocol auto-detection handles common development scenarios
- localhost and local IPs automatically get http://
- JSON output includes final URL and page title

## Issues Discovered

(Issues will be documented here during testing)
