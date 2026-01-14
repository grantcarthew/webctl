# p-043: Testing scroll Command

- Status: In Progress
- Started: 2025-12-31

## Overview

Test the webctl scroll command which scrolls to an element, absolute position, or by an offset. Supports three modes: element (scroll to element), absolute (scroll to x,y position), and relative (scroll by x,y offset).

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-scroll.sh
```

## Code References

- internal/cli/scroll.go

## Command Signature

```
webctl scroll <selector> | --to x,y | --by x,y
```

Arguments:
- selector: CSS selector to scroll element into view (element mode)

Flags:
- --to x,y: Scroll to absolute position (absolute mode)
- --by x,y: Scroll by offset from current position (relative mode)

Coordinates are specified as x,y where:
- x = horizontal position (0 = left edge)
- y = vertical position (0 = top edge)

## Test Checklist

Element mode (scroll to element):
- [ ] scroll "#footer" (scroll footer into view)
- [ ] scroll ".next-section" (scroll to section)
- [ ] scroll "article h2" (scroll to heading)
- [ ] scroll "[data-testid=results]" (scroll to test ID)
- [ ] scroll "nav" (scroll to navigation)
- [ ] Verify element scrolled to center of viewport
- [ ] Verify smooth scrolling behavior

Element mode - different page sections:
- [ ] Scroll to header
- [ ] Scroll to main content
- [ ] Scroll to sidebar
- [ ] Scroll to footer
- [ ] Scroll to specific section
- [ ] Scroll to specific heading

Absolute mode (--to x,y):
- [ ] scroll --to 0,0 (scroll to top-left, top of page)
- [ ] scroll --to 0,500 (scroll 500px from top)
- [ ] scroll --to 0,1000 (scroll 1000px from top)
- [ ] scroll --to 100,200 (scroll to x=100, y=200)
- [ ] scroll --to 0,9999 (scroll to bottom of page)
- [ ] Verify scrolls to exact coordinates

Absolute mode - specific positions:
- [ ] Scroll to top of page (0,0)
- [ ] Scroll to middle of page
- [ ] Scroll to bottom of page
- [ ] Scroll to specific x,y coordinate
- [ ] Verify position after scroll

Relative mode (--by x,y):
- [ ] scroll --by 0,100 (scroll down 100px)
- [ ] scroll --by 0,-100 (scroll up 100px)
- [ ] scroll --by 0,500 (scroll down 500px)
- [ ] scroll --by 200,0 (scroll right 200px)
- [ ] scroll --by -200,0 (scroll left 200px)
- [ ] scroll --by 100,100 (scroll diagonally)
- [ ] scroll --by 0,-9999 (scroll to top)
- [ ] Verify relative offset from current position

Relative mode - scrolling patterns:
- [ ] Scroll down incrementally (multiple --by 0,100)
- [ ] Scroll up incrementally (multiple --by 0,-100)
- [ ] Scroll to top with --by 0,-9999
- [ ] Scroll down to bottom with --by 0,9999

Navigation patterns:
- [ ] scroll --to 0,0 (return to top)
- [ ] scroll "#main-content" (skip to main content)
- [ ] scroll "#section-2" then scroll "#section-3" (sequential sections)
- [ ] Verify navigation between sections

Coordinate parsing:
- [ ] Valid coordinates: 0,0
- [ ] Valid coordinates: 100,200
- [ ] Valid coordinates with spaces: "100, 200"
- [ ] Negative coordinates: -100,-100
- [ ] Large coordinates: 9999,9999
- [ ] Invalid format: "100" (missing y) - should error
- [ ] Invalid format: "100,200,300" (too many) - should error
- [ ] Invalid format: "abc,def" (non-numeric) - should error

Error cases:
- [ ] Scroll with non-existent selector (error: element not found)
- [ ] Scroll with empty selector
- [ ] Scroll with invalid CSS selector
- [ ] Scroll --to with invalid coordinates (error message)
- [ ] Scroll --by with invalid coordinates (error message)
- [ ] Scroll with no arguments (error: provide selector or flags)
- [ ] Scroll with both selector and --to (which takes precedence?)
- [ ] Scroll with both --to and --by (which takes precedence?)
- [ ] Daemon not running (error message)

Mode precedence:
- [ ] Provide --to flag only (absolute mode)
- [ ] Provide --by flag only (relative mode)
- [ ] Provide selector only (element mode)
- [ ] Provide selector and --to (verify which is used)
- [ ] Provide selector and --by (verify which is used)
- [ ] Provide --to and --by (verify which is used)

Long page testing:
- [ ] Navigate to long page (e.g., documentation page)
- [ ] Scroll to various sections
- [ ] Scroll to bottom then back to top
- [ ] Verify scrolling on very long pages

Horizontal scrolling:
- [ ] Navigate to page with horizontal scroll
- [ ] scroll --to 500,0 (scroll horizontally)
- [ ] scroll --by 100,0 (scroll right)
- [ ] scroll --by -100,0 (scroll left)
- [ ] Verify horizontal scrolling works

Output formats:
- [ ] Default output: {"ok": true}
- [ ] --json output format
- [ ] --no-color output
- [ ] --debug verbose output
- [ ] Error output format

CLI vs REPL:
- [ ] CLI: webctl scroll "#footer"
- [ ] CLI: webctl scroll --to 0,0
- [ ] CLI: webctl scroll --by 0,100
- [ ] CLI: webctl scroll ".next-section"
- [ ] REPL: scroll "#footer"
- [ ] REPL: scroll --to 0,0
- [ ] REPL: scroll --by 0,100
- [ ] REPL: scroll ".next-section"

## Notes

- Three scroll modes: element, absolute (--to), relative (--by)
- Element mode scrolls element to center of viewport
- Absolute mode scrolls to exact x,y coordinates on page
- Relative mode scrolls by x,y offset from current position
- Coordinates format: x,y (e.g., "0,500" or "100,200")
- x = horizontal (0 = left edge), y = vertical (0 = top edge)
- Negative coordinates in relative mode scroll in opposite direction
- Useful for navigation, testing scroll-based interactions
- Can be used to test lazy-loading, infinite scroll, sticky headers
- Mode precedence: --to > --by > selector (verify this)

## Issues Discovered

(Issues will be documented here during testing)
