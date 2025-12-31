#!/bin/bash
# Title: webctl scroll command tests

set -e

# Color output helpers
title() { echo -e "\n\033[1;34m=== $1 ===\033[0m"; }
heading() { echo -e "\n\033[1;32m## $1\033[0m"; }
cmd() {
    echo -e "\n\033[0;33m$ $1\033[0m"
    echo "$1" | xclip -selection clipboard
    echo "(Command copied to clipboard - paste and execute)"
    read -p "Press Enter to continue..."
}

clear
title "webctl scroll Command Test Suite"
echo "Project: P-043"
echo "Tests scrolling to elements, absolute positions, and by offsets"
echo ""
echo "Prerequisites:"
echo "  - webctl must be built"
echo "  - xclip must be installed"
echo "  - Daemon running (or start one)"
echo ""
read -p "Press Enter to begin tests..."

# Setup
title "Setup"
echo "Ensure daemon is running and navigate to long page"
cmd "webctl start"

echo ""
echo "Start daemon if not running, then continue"
read -p "Press Enter when daemon ready..."

heading "Navigate to long page for scrolling (GitHub documentation)"
cmd "webctl navigate https://docs.github.com/en/get-started/start-your-journey/hello-world --wait"

echo ""
echo "Wait for page to load (long page with multiple sections)"
read -p "Press Enter when page loaded..."

# Element mode tests
title "Element Mode (Scroll to Element)"

heading "Scroll to footer"
cmd "webctl scroll \"footer\""

echo ""
echo "Verify: Page scrolled to footer (centered in viewport)"
read -p "Press Enter to continue..."

heading "Scroll back to top (scroll to header or nav)"
cmd "webctl scroll \"header\""

echo ""
echo "Verify: Page scrolled back to top"
read -p "Press Enter to continue..."

heading "Scroll to main content"
cmd "webctl scroll \"main\""

echo ""
echo "Verify: Main content area scrolled to center"
read -p "Press Enter to continue..."

heading "Scroll to specific heading"
cmd "webctl scroll \"h2\""

echo ""
echo "Verify: First h2 heading scrolled into view (centered)"
read -p "Press Enter to continue..."

heading "Scroll to article content"
cmd "webctl scroll \"article\""

echo ""
echo "Verify: Article scrolled to center of viewport"
read -p "Press Enter to continue..."

# Absolute mode tests
title "Absolute Mode (--to x,y)"

heading "Scroll to top of page (0,0)"
cmd "webctl scroll --to 0,0"

echo ""
echo "Verify: Page at very top, scroll position 0,0"
read -p "Press Enter to continue..."

heading "Scroll to 500px from top"
cmd "webctl scroll --to 0,500"

echo ""
echo "Verify: Page scrolled to 500px position"
read -p "Press Enter to continue..."

heading "Scroll to 1000px from top"
cmd "webctl scroll --to 0,1000"

echo ""
echo "Verify: Page scrolled to 1000px position"
read -p "Press Enter to continue..."

heading "Scroll to 2000px from top"
cmd "webctl scroll --to 0,2000"

echo ""
echo "Verify: Page scrolled to 2000px position"
read -p "Press Enter to continue..."

heading "Scroll to very large Y value (bottom of page)"
cmd "webctl scroll --to 0,99999"

echo ""
echo "Verify: Page scrolled to bottom (max scroll position)"
read -p "Press Enter to continue..."

heading "Scroll back to top"
cmd "webctl scroll --to 0,0"

echo ""
echo "Verify: Page back at top"
read -p "Press Enter to continue..."

# Relative mode tests
title "Relative Mode (--by x,y)"

heading "Scroll down 100px from current position"
cmd "webctl scroll --by 0,100"

echo ""
echo "Verify: Page scrolled down 100px"
read -p "Press Enter to continue..."

heading "Scroll down another 100px"
cmd "webctl scroll --by 0,100"

echo ""
echo "Verify: Page scrolled down another 100px (total 200px from top)"
read -p "Press Enter to continue..."

heading "Scroll down 500px"
cmd "webctl scroll --by 0,500"

echo ""
echo "Verify: Page scrolled down 500px more"
read -p "Press Enter to continue..."

heading "Scroll up 100px"
cmd "webctl scroll --by 0,-100"

echo ""
echo "Verify: Page scrolled up 100px"
read -p "Press Enter to continue..."

heading "Scroll up 500px"
cmd "webctl scroll --by 0,-500"

echo ""
echo "Verify: Page scrolled up 500px"
read -p "Press Enter to continue..."

heading "Scroll to top with large negative offset"
cmd "webctl scroll --by 0,-99999"

echo ""
echo "Verify: Page scrolled to top (min scroll position)"
read -p "Press Enter to continue..."

heading "Scroll to bottom with large positive offset"
cmd "webctl scroll --by 0,99999"

echo ""
echo "Verify: Page scrolled to bottom (max scroll position)"
read -p "Press Enter to continue..."

# Coordinate parsing tests
title "Coordinate Parsing Tests"

heading "Coordinates with spaces"
cmd "webctl scroll --to \"0, 500\""

echo ""
echo "Verify: Parsed correctly, scrolled to 500px"
read -p "Press Enter to continue..."

heading "Different X and Y values"
cmd "webctl scroll --to 0,0"

echo ""
read -p "Press Enter to continue..."

cmd "webctl scroll --to 100,200"

echo ""
echo "Verify: Scrolled to x=100, y=200"
echo "Note: Horizontal scroll only visible if page has horizontal content"
read -p "Press Enter to continue..."

heading "Negative coordinates in relative mode"
cmd "webctl scroll --to 0,1000"

echo ""
read -p "Press Enter to continue..."

cmd "webctl scroll --by -50,-200"

echo ""
echo "Verify: Scrolled left 50px and up 200px from previous position"
read -p "Press Enter to continue..."

# Navigation patterns
title "Navigation Patterns"

heading "Return to top pattern"
cmd "webctl scroll --to 0,0"

echo ""
echo "Verify: Quick return to top"
read -p "Press Enter to continue..."

heading "Jump to specific section (simulate table of contents)"
cmd "webctl scroll \"h2\""

echo ""
echo "Verify: Jumped to first h2 section"
read -p "Press Enter to continue..."

heading "Sequential section navigation"
cmd "webctl scroll \"h2\""

echo ""
read -p "Press Enter to continue..."

heading "Use relative scroll to move down page incrementally"
cmd "webctl scroll --by 0,300"

echo ""
read -p "Press Enter to continue..."

cmd "webctl scroll --by 0,300"

echo ""
read -p "Press Enter to continue..."

cmd "webctl scroll --by 0,300"

echo ""
echo "Verify: Scrolled down in increments"
read -p "Press Enter to continue..."

# Long page testing
title "Long Page Testing"

heading "Navigate to very long page"
cmd "webctl navigate https://en.wikipedia.org/wiki/Web_browser --wait"

echo ""
echo "Wait for Wikipedia page to load (very long page)"
read -p "Press Enter when loaded..."

heading "Scroll to bottom of long page"
cmd "webctl scroll --to 0,99999"

echo ""
echo "Verify: Scrolled to bottom of Wikipedia page"
read -p "Press Enter to continue..."

heading "Scroll to top of long page"
cmd "webctl scroll --to 0,0"

echo ""
echo "Verify: Scrolled to top"
read -p "Press Enter to continue..."

heading "Scroll to middle-ish of page"
cmd "webctl scroll --to 0,5000"

echo ""
echo "Verify: Scrolled to middle section"
read -p "Press Enter to continue..."

heading "Scroll to specific section heading"
cmd "webctl scroll \"h2\""

echo ""
echo "Verify: First h2 heading in view"
read -p "Press Enter to continue..."

# Horizontal scrolling (if page has it)
title "Horizontal Scrolling"

heading "Note about horizontal scrolling"
echo "Horizontal scrolling only works if page has horizontal content"
echo "Most pages don't have horizontal scroll, so these tests may not show visible effect"
echo ""
read -p "Press Enter to continue..."

heading "Try scrolling right"
cmd "webctl scroll --to 100,0"

echo ""
echo "Verify: Scrolled right 100px (if page has horizontal content)"
read -p "Press Enter to continue..."

heading "Try scrolling right by offset"
cmd "webctl scroll --by 100,0"

echo ""
echo "Verify: Scrolled right 100px more"
read -p "Press Enter to continue..."

heading "Scroll back to left"
cmd "webctl scroll --to 0,0"

echo ""
echo "Verify: Scrolled back to left edge"
read -p "Press Enter to continue..."

# Error cases
title "Error Cases"

heading "Scroll with non-existent selector"
cmd "webctl scroll \"#nonexistent-element-xyz\""

echo ""
echo "Verify: Error message 'element not found'"
read -p "Press Enter to continue..."

heading "Scroll with invalid CSS selector"
cmd "webctl scroll \"[[invalid]]\""

echo ""
echo "Verify: Error message about invalid selector"
read -p "Press Enter to continue..."

heading "Scroll --to with invalid coordinates (missing y)"
cmd "webctl scroll --to 100"

echo ""
echo "Verify: Error message about invalid coordinates format"
read -p "Press Enter to continue..."

heading "Scroll --to with non-numeric coordinates"
cmd "webctl scroll --to abc,def"

echo ""
echo "Verify: Error message about non-numeric coordinates"
read -p "Press Enter to continue..."

heading "Scroll --by with invalid format"
cmd "webctl scroll --by 100,200,300"

echo ""
echo "Verify: Error message about coordinate format"
read -p "Press Enter to continue..."

heading "Scroll with no arguments"
cmd "webctl scroll"

echo ""
echo "Verify: Error message 'provide a selector, --to x,y, or --by x,y'"
read -p "Press Enter to continue..."

# Output formats
title "Output Format Tests"

heading "JSON output"
cmd "webctl scroll --to 0,500 --json"

echo ""
echo "Verify: JSON formatted output"
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl scroll --to 0,0 --no-color"

echo ""
echo "Verify: No ANSI color codes"
read -p "Press Enter to continue..."

heading "Debug output"
cmd "webctl scroll \"h2\" --debug"

echo ""
echo "Verify: Debug logging information"
read -p "Press Enter to continue..."

# REPL mode tests
title "REPL Mode Tests"

heading "Test scroll in REPL"
echo "Switch to daemon terminal and execute:"
cmd "scroll \"footer\""

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

heading "Test scroll --to in REPL"
echo "In REPL, try:"
cmd "scroll --to 0,0"

echo ""
echo "Should scroll to top"
read -p "Press Enter when tested in REPL..."

heading "Test scroll --by in REPL"
echo "In REPL, try:"
cmd "scroll --by 0,500"

echo ""
echo "Should scroll down 500px"
read -p "Press Enter when tested in REPL..."

heading "Test scroll element in REPL"
echo "In REPL, try:"
cmd "scroll \"main\""

echo ""
echo "Should scroll main element into view"
read -p "Press Enter when tested in REPL..."

# Test completed
title "Test Suite Complete"
echo "All scroll command tests finished"
echo ""
echo "Review checklist in docs/projects/p-043-testing-scroll.md"
echo "Document any issues discovered during testing"
