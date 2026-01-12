# P-034: Testing html Command

- Status: Completed
- Started: 2025-12-31
- Completed: 2026-01-12

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
- save <path>: Save to file or directory/ (trailing slash = directory)

Flags:
- --select, -s <selector>: Filter to element(s) matching CSS selector
- --find, -f <text>: Search for text within HTML
- --raw: Skip HTML formatting (return as-is from browser)
- --debug: Enable debug output (global flag)
- --json: Output in JSON format (global flag)
- --no-color: Disable colored output (global flag)

## Test Checklist

Default mode (stdout):
- [x] html (full page to stdout)
- [x] html --select "#main" (element to stdout)
- [x] html --find "login" (search and output)
- [x] Verify HTML output to stdout
- [x] Verify no file created

Save mode (file output):
- [x] html save (full page to temp)
- [x] html save --select ".content" (element to temp)
- [x] html save --find "error" (search and save)
- [x] Verify file saved to /tmp/webctl-html/
- [x] Verify auto-generated filename format
- [x] Verify JSON response with file path

Save mode (custom path):
- [x] html save ./page.html (save to file)
- [x] html save ./output/ (save to dir with auto-filename, creates dir)
- [x] html save ./output (save to file named "output", NOT a directory)
- [x] html save ./debug.html --select "form" --find "password"
- [x] Verify trailing slash behavior

Select flag:
- [x] --select with ID selector (#header)
- [x] --select with class selector (.button)
- [x] --select with element selector (form)
- [x] --select with complex selector (div > p.intro)
- [x] --select matching multiple elements
- [x] --select matching no elements (error)

Find flag:
- [x] --find with simple text
- [x] --find with case sensitivity
- [x] --find with no matches (error)
- [x] --find combined with --select

Raw flag:
- [x] --raw output (no formatting)
- [x] Compare raw vs formatted output
- [x] --raw with --select
- [x] --raw with show/save modes

Combination tests:
- [x] --select and --find together
- [x] --raw and --select together
- [x] Multiple flags with different modes

Output formats:
- [x] Default JSON response (file path)
- [x] --json with show mode
- [x] --no-color output
- [x] --debug verbose output

Error cases:
- [x] Selector matches no elements
- [x] Find text not in HTML
- [x] Save to invalid path
- [x] Daemon not running

CLI vs REPL:
- [x] CLI: webctl html
- [x] CLI: webctl html save
- [x] CLI: webctl html save ./page.html
- [x] REPL: html
- [x] REPL: html save
- [x] REPL: html save ./page.html

## Notes

- Default mode outputs to stdout (Unix convention)
- Save mode saves to temp or custom path
- Trailing slash convention: path/ = directory (auto-filename), path = file (like rsync)
- Select flag extracts specific elements (computed styles for CSS)
- Find flag searches text content
- Raw flag skips formatting for exact browser HTML
- Auto-generated filenames include timestamp and page title

## Issues Discovered and Fixed

### Fixed: --find filter applied before HTML formatting

Problem: The `--find` flag was applied to raw HTML (often a single long line) before formatting. This caused the entire page to match if any part contained the search text.

Fix: Moved `--find` filter to run after HTML formatting, so line-based search works correctly on prettified HTML.

File: `internal/cli/html.go:273-290`

### Fixed: Error messages for no matches/no elements

Problem: "No matches found" and "selector matched no elements" were prefixed with "Error:" which is misleading since the command worked, just found nothing.

Fix: Created `outputNotice()` function and `ErrNoMatches`/`ErrNoElements` sentinels. These cases now output without "Error:" prefix but still return exit code 1 for scripting.

Files: `internal/cli/root.go`, `internal/cli/html.go`, `internal/cli/css.go`, `internal/cli/cookies.go`, `internal/cli/console.go`, `internal/cli/network.go`

### Fixed: --json flag not producing JSON output

Problem: `webctl html --json` output raw HTML instead of JSON structure.

Fix: Added `JSONOutput` check in `runHTML` and `runCSS` to wrap output in JSON structure.

Files: `internal/cli/html.go`, `internal/cli/css.go`

### Fixed: REPL not parsing quoted arguments

Problem: REPL used `strings.Fields()` which kept literal quotes. `html --select "h1"` passed `"h1"` (with quotes) as the selector, causing JavaScript errors.

Fix: Created `parseArgs()` function that properly handles single and double quoted strings, stripping quotes from the result.

File: `internal/daemon/repl.go`

## New Project Created

P-035: Debug Output - Add comprehensive debug output throughout the CLI codebase (currently sparse/missing).
