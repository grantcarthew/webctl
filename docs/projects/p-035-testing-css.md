# P-035: Testing css Command

- Status: In Progress
- Started: 2025-12-31

## Overview

Test the webctl css command which extracts CSS from the current page. This command outputs to stdout by default, with save/computed/get subcommands for specific operations.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-css.sh
```

## Code References

- internal/cli/css.go
- internal/cssformat (CSS formatting)

## Command Signature

```
webctl css [save [path]|computed <selector>|get <selector> <property>] [--select sel] [--find text] [--raw]
```

Subcommands:
- (default): Output CSS to stdout
- save: Save to /tmp/webctl-css/ with auto-generated filename
- save <path>: Save to file or directory/ (trailing slash = directory)
- computed <selector>: Get computed styles to stdout
- get <selector> <property>: Get single CSS property to stdout

Flags (default/show/save modes):
- --select, -s <selector>: Filter to element's computed styles
- --find, -f <text>: Search for text within CSS
- --raw: Skip CSS formatting (return as-is from browser)
- --debug: Enable debug output (global flag)
- --json: Output in JSON format (global flag)
- --no-color: Disable colored output (global flag)

Flags (computed mode):
- --json: Output in JSON format (global flag)

## Test Checklist

Default mode (stdout):
- [ ] css (all stylesheets to stdout)
- [ ] css --select "#main" (computed styles to stdout)
- [ ] css --find "background" (search and output)
- [ ] Verify CSS output to stdout
- [ ] Verify no file created

Save mode (file output):
- [ ] css save (all stylesheets to temp)
- [ ] css save --select ".content" (computed styles to temp)
- [ ] css save --find "color" (search and save)
- [ ] Verify file saved to /tmp/webctl-css/
- [ ] Verify auto-generated filename format
- [ ] Verify JSON response with file path

Save mode (custom path):
- [ ] css save ./styles.css (save to file)
- [ ] css save ./output/ (save to dir with auto-filename, creates dir)
- [ ] css save ./output (save to file named "output", NOT a directory)
- [ ] css save ./debug.css --select "form" --find "border"
- [ ] Verify trailing slash behavior

Computed mode (stdout):
- [ ] css computed "#header" (all computed styles)
- [ ] css computed ".button" (class selector)
- [ ] css computed "nav > ul" (complex selector)
- [ ] css computed "#main" --json (JSON output)
- [ ] Verify text format output (property: value)
- [ ] Verify JSON format output (styles object)
- [ ] Selector matches no elements (error)

Get mode (stdout):
- [ ] css get "#header" background-color (single property)
- [ ] css get ".button" display (class selector)
- [ ] css get "body" font-size (element selector)
- [ ] Verify plain value output for scripting
- [ ] Invalid property (error)
- [ ] Selector matches no elements (error)

Select flag:
- [ ] --select with ID selector (#header)
- [ ] --select with class selector (.button)
- [ ] --select with element selector (form)
- [ ] --select with complex selector (div > p.intro)
- [ ] --select matching no elements (error)
- [ ] Compare stylesheets vs computed styles output

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
- [ ] --json with computed mode
- [ ] --no-color output
- [ ] --debug verbose output

Error cases:
- [ ] Selector matches no elements
- [ ] Find text not in CSS
- [ ] Save to invalid path
- [ ] Property does not exist (get mode)
- [ ] Invalid selector syntax
- [ ] Daemon not running

CLI vs REPL:
- [ ] CLI: webctl css
- [ ] CLI: webctl css save
- [ ] CLI: webctl css save ./styles.css
- [ ] CLI: webctl css computed "#main"
- [ ] CLI: webctl css get "#header" color
- [ ] REPL: css
- [ ] REPL: css save
- [ ] REPL: css save ./styles.css
- [ ] REPL: css computed "#main"
- [ ] REPL: css get "#header" color

## Notes

- Default mode outputs to stdout (Unix convention)
- Save mode saves to temp or custom path
- Trailing slash convention: path/ = directory (auto-filename), path = file (like rsync)
- Select flag extracts computed styles for specific elements
- Find flag searches text within CSS rules
- Raw flag skips formatting for exact browser CSS
- Auto-generated filenames include timestamp and page title or selector
- Computed mode returns all CSS properties computed by browser
- Get mode returns single property value for scripting
- Computed styles differ from stylesheet CSS (includes inherited/default values)

## Issues Discovered

(Issues will be documented here during testing)
