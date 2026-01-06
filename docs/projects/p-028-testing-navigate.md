# P-028: Testing navigate Command

- Status: In Progress
- Started: 2026-01-06

## Overview

Test the webctl navigate command which navigates the browser to a specified URL. This is a core navigation command used frequently in browser automation workflows.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-navigate.sh
```

## Code References

- internal/cli/navigate.go

## Command Signature

```
webctl navigate <url> [--wait] [--timeout <seconds>]
```

Flags:
- --wait: Wait for page load to complete (default: true)
- --timeout: Timeout in seconds when waiting (default: 30)
- --debug: Enable debug output (global flag)
- --json: Output in JSON format (global flag)
- --no-color: Disable colored output (global flag)

## Test Checklist

Basic functionality:
- [ ] Navigate to a simple URL
- [ ] Verify page loads correctly
- [ ] Verify URL updates in REPL prompt
- [ ] Navigate to HTTPS site
- [ ] Navigate to HTTP site

URL handling:
- [ ] Full URL with protocol (https://example.com)
- [ ] URL without protocol (example.com) - should auto-add https://
- [ ] URL with path (example.com/page)
- [ ] URL with query params (example.com?foo=bar)

Wait behaviour:
- [ ] Default wait (should wait for load)
- [ ] Explicit --wait flag
- [ ] --wait=false (return immediately)
- [ ] --timeout with slow page

Output formats:
- [ ] Default output (text mode)
- [ ] JSON output with --json flag
- [ ] No color output with --no-color flag

Error cases:
- [ ] Invalid URL format
- [ ] Non-existent domain
- [ ] Timeout on slow/hanging page
- [ ] Navigate when daemon not running

CLI vs REPL:
- [ ] CLI: webctl navigate <url>
- [ ] REPL: navigate <url>
- [ ] REPL abbreviation: nav <url>

## Notes

- Navigate is one of the most frequently used commands
- URL without protocol should default to https://
- Wait behaviour is important for automation reliability
