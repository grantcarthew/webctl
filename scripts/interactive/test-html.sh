#!/bin/bash
# Title: webctl html command tests

set -e

# Color output helpers
title() { echo -e "\n\033[1;34m=== $1 ===\033[0m"; }
heading() { echo -e "\n\033[1;32m## $1\033[0m"; }
cmd() {
    echo -e "\n\033[0;33m$ $1\033[0m"
    if [[ "$OSTYPE" == "darwin"* ]]; then echo "$1" | pbcopy; else echo "$1" | xclip -selection clipboard; fi
    echo "(Command copied to clipboard - paste and execute)"
    read -p "Press Enter to continue..."
}

clear
title "webctl html Command Test Suite"
echo "Project: P-034"
echo "Tests HTML extraction with stdout default and save mode"
echo ""
echo "Prerequisites:"
echo "  - webctl must be built"
echo "  - Clipboard tool (pbcopy on macOS, xclip on Linux)"
echo "  - Daemon running (or start one)"
echo ""
read -p "Press Enter to begin tests..."

# Setup
title "Setup"
echo "Ensure daemon is running and navigate to test page"
cmd "webctl start"

echo ""
echo "Start daemon if not running, then continue"
read -p "Press Enter when daemon ready..."

heading "Navigate to example.com for testing"
cmd "webctl navigate example.com --wait"

echo ""
echo "Wait for page to fully load"
read -p "Press Enter when page loaded..."

# Default mode tests
title "Default Mode (Output to Stdout)"

heading "Extract full page HTML to stdout"
cmd "webctl html"

echo ""
echo "Verify: HTML output to stdout"
echo "Verify: No file created"
read -p "Press Enter to continue..."

heading "Extract specific element to stdout"
cmd "webctl html --select \"h1\""

echo ""
echo "Verify: Only h1 element(s) shown"
read -p "Press Enter to continue..."

heading "Search for text in HTML"
cmd "webctl html --find \"Example\""

echo ""
echo "Verify: HTML containing 'Example' shown"
read -p "Press Enter to continue..."

# Save mode tests
title "Save Mode (File Output)"

heading "Save to temp (no path)"
cmd "webctl html save"

echo ""
echo "Verify: File saved to /tmp/webctl-html/"
echo "Verify: Auto-generated filename with timestamp"
echo "Verify: JSON response shows file path"
read -p "Press Enter to continue..."

heading "Save to custom file"
cmd "webctl html save ./page.html"

echo ""
echo "Verify: File saved to ./page.html"
read -p "Press Enter to continue..."

heading "Save to directory with auto-filename (trailing slash = directory)"
cmd "webctl html save ./output/"

echo ""
echo "Verify: File saved to ./output/ with auto-generated name"
echo "Note: Trailing slash (/) is REQUIRED for directory behavior"
echo "      Without slash, it would create a file named 'output'"
read -p "Press Enter to continue..."

heading "Save with selector and find"
cmd "webctl html save ./debug.html --select \"div\" --find \"example\""

echo ""
echo "Verify: File saved to ./debug.html with filtered content"
read -p "Press Enter to continue..."

# Select flag tests
title "Select Flag Tests"

heading "Select by ID"
cmd "webctl html --select \"#main\""

echo ""
echo "Check if element with id 'main' exists, or try another ID"
read -p "Press Enter to continue..."

heading "Select by class"
cmd "webctl html --select \"div\""

echo ""
echo "Verify: Only div elements shown"
read -p "Press Enter to continue..."

heading "Complex selector"
cmd "webctl html --select \"body > div\""

echo ""
echo "Verify: Only direct div children of body shown"
read -p "Press Enter to continue..."

# Find flag tests
title "Find Flag Tests"

heading "Find simple text"
cmd "webctl html --find \"Example\""

echo ""
echo "Verify: HTML contains 'Example'"
read -p "Press Enter to continue..."

heading "Find with no matches (should error)"
cmd "webctl html --find \"ThisTextDoesNotExist123\""

echo ""
echo "Verify: Error message about no matches"
read -p "Press Enter to continue..."

heading "Find combined with select"
cmd "webctl html --select \"h1\" --find \"Example\""

echo ""
echo "Verify: h1 elements containing 'Example'"
read -p "Press Enter to continue..."

# Raw flag tests
title "Raw Flag Tests"

heading "Raw output (no formatting)"
cmd "webctl html --raw"

echo ""
echo "Verify: Unformatted HTML as-is from browser"
read -p "Press Enter to continue..."

heading "Raw with select"
cmd "webctl html --raw --select \"h1\""

echo ""
echo "Verify: Raw HTML of selected elements"
read -p "Press Enter to continue..."

# Output format tests
title "Output Format Tests"

heading "JSON output"
cmd "webctl html --json"

echo ""
echo "Verify: JSON formatted output"
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl html --no-color"

echo ""
echo "Verify: No ANSI color codes"
read -p "Press Enter to continue..."

heading "Debug verbose output"
cmd "webctl html --debug"

echo ""
echo "Verify: Debug logging information"
read -p "Press Enter to continue..."

# Error cases
title "Error Cases"

heading "Selector matches no elements"
cmd "webctl html --select \"#nonexistent-id-xyz\""

echo ""
echo "Verify: Error message about selector not matching"
read -p "Press Enter to continue..."

heading "Save to invalid path"
cmd "webctl html save /root/invalid/path.html"

echo ""
echo "Verify: Permission denied or path error"
read -p "Press Enter to continue..."

# REPL mode tests
title "REPL Mode Tests"

heading "Test html in REPL"
echo "Switch to daemon terminal and execute:"
cmd "html"

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

heading "Test html with flags in REPL"
echo "In REPL, try:"
cmd "html --select \"h1\" --find \"Example\""

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

# Advanced scenarios
title "Advanced Scenarios"

heading "Navigate to complex page"
cmd "webctl navigate https://github.com --wait"

echo ""
echo "Wait for GitHub to load"
read -p "Press Enter when loaded..."

heading "Extract navigation menu"
cmd "webctl html --select \"nav\""

echo ""
echo "Verify: Navigation HTML shown"
read -p "Press Enter to continue..."

heading "Save page header"
cmd "webctl html save ./github-header.html --select \"header\""

echo ""
echo "Verify: Header saved to file"
read -p "Press Enter to continue..."

# Cleanup
title "Cleanup"
echo "Clean up test files if desired:"
cmd "rm -f ./page.html ./debug.html ./output/*.html ./github-header.html"

echo ""
echo "Remove test files when ready"
read -p "Press Enter to finish..."

title "Test Suite Complete"
echo "All html command tests finished"
echo ""
echo "Review checklist in docs/projects/p-034-testing-html.md"
echo "Document any issues discovered during testing"
