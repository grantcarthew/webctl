# P-034: Testing html Command

- Status: In Progress
- Started: 2025-12-31

## Overview

Test the webctl html command which extracts HTML from the current page. This command outputs to stdout by default, with a save subcommand for file output.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-html.sh
```

## Code References

- internal/cli/html.go
- internal/htmlformat (HTML formatting)

## Command Signature

```
webctl html [save [path]] [--select sel] [--find text] [--raw]
```

Subcommands:
- (default): Output HTML to stdout
- save: Save to /tmp/webctl-html/ with auto-generated filename
- save <path>: Save to custom path

Flags:
- --select, -s <selector>: Filter to element(s) matching CSS selector
- --find, -f <text>: Search for text within HTML
- --raw: Skip HTML formatting (return as-is from browser)
- --debug: Enable debug output (global flag)
- --json: Output in JSON format (global flag)
- --no-color: Disable colored output (global flag)

## Test Checklist

Default mode (stdout):
- [ ] html (full page to stdout)
- [ ] html --select "#main" (element to stdout)
- [ ] html --find "login" (search and output)
- [ ] Verify HTML output to stdout
- [ ] Verify no file created

Save mode (file output):
- [ ] html save (full page to temp)
- [ ] html save --select ".content" (element to temp)
- [ ] html save --find "error" (search and save)
- [ ] Verify file saved to /tmp/webctl-html/
- [ ] Verify auto-generated filename format
- [ ] Verify JSON response with file path

Save mode (custom path):
- [ ] html save ./page.html (save to file)
- [ ] html save ./output/ (save to dir with auto-filename)
- [ ] html save ./debug.html --select "form" --find "password"
- [ ] Verify file saved to custom path

Select flag:
- [ ] --select with ID selector (#header)
- [ ] --select with class selector (.button)
- [ ] --select with element selector (form)
- [ ] --select with complex selector (div > p.intro)
- [ ] --select matching multiple elements
- [ ] --select matching no elements (error)

Find flag:
- [ ] --find with simple text
- [ ] --find with case sensitivity
- [ ] --find with no matches (error)
- [ ] --find combined with --select

Raw flag:
- [ ] --raw output (no formatting)
- [ ] Compare raw vs formatted output
- [ ] --raw with --select
- [ ] --raw with show/save modes

Combination tests:
- [ ] --select and --find together
- [ ] --raw and --select together
- [ ] Multiple flags with different modes

Output formats:
- [ ] Default JSON response (file path)
- [ ] --json with show mode
- [ ] --no-color output
- [ ] --debug verbose output

Error cases:
- [ ] Selector matches no elements
- [ ] Find text not in HTML
- [ ] Save to invalid path
- [ ] Daemon not running

CLI vs REPL:
- [ ] CLI: webctl html
- [ ] CLI: webctl html save
- [ ] CLI: webctl html save ./page.html
- [ ] REPL: html
- [ ] REPL: html save
- [ ] REPL: html save ./page.html

## Notes

- Default mode outputs to stdout (Unix convention)
- Save mode saves to temp or custom path
- Select flag extracts specific elements (computed styles for CSS)
- Find flag searches text content
- Raw flag skips formatting for exact browser HTML
- Auto-generated filenames include timestamp and page title

## Issues Discovered

### Fixed: --find filter applied before HTML formatting

**Problem:** The `--find` flag was applied to raw HTML (often a single long line) before formatting. This caused the entire page to match if any part contained the search text.

**Fix:** Moved `--find` filter to run after HTML formatting, so line-based search works correctly on prettified HTML.

**File:** `internal/cli/html.go:273-290`
