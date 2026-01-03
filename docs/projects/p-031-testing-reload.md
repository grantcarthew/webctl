# P-031: Testing reload Command

- Status: In Progress
- Started: 2025-12-31

## Overview

Test the webctl reload command which performs a hard reload of the current page (ignores cache). Similar to navigate but reloads current URL.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-reload.sh
```

## Code References

- internal/cli/reload.go

## Command Signature

```
webctl reload [--wait] [--timeout ms]
```

Flags:
- --wait: Wait for page load completion (default: false)
- --timeout <ms>: Timeout in milliseconds with --wait (default: 30000)
- --debug: Enable debug output (global flag)
- --json: Output in JSON format (global flag)
- --no-color: Disable colored output (global flag)

## Test Checklist

Basic reload:
- [ ] Load page, then reload
- [ ] Verify hard reload (cache ignored)
- [ ] Verify immediate return (no wait)
- [ ] Verify page reloads in background

Wait functionality:
- [ ] Reload with --wait flag
- [ ] Verify blocks until reload complete
- [ ] Test custom timeout (--timeout 60000)
- [ ] Test timeout expiry

Cache behavior:
- [ ] Load page with cacheable resources
- [ ] Modify cached resource on server
- [ ] Reload and verify fresh content loaded
- [ ] Compare with soft reload (if possible)

Reload responses:
- [ ] Default text output (OK)
- [ ] JSON output includes URL and title (--json)
- [ ] No color output (--no-color)

Various page states:
- [ ] Reload static page
- [ ] Reload SPA with JavaScript state
- [ ] Reload page with form data
- [ ] Reload after script execution

Error cases:
- [ ] Reload when page fails to load
- [ ] Reload with network error
- [ ] Reload with timeout (--wait)
- [ ] Reload when daemon not running

Common patterns:
- [ ] reload && ready
- [ ] reload --wait && console show
- [ ] Reload to clear JavaScript state

CLI vs REPL:
- [ ] CLI: webctl reload
- [ ] REPL: reload

## Notes

- Always performs hard reload (ignores cache)
- Returns immediately unless --wait specified
- Useful for development when files change
- Reloads current URL (same as browser F5/Cmd+R with cache clear)
- JSON output includes URL and title after reload

## Issues Discovered

(Issues will be documented here during testing)
