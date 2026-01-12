#!/bin/bash
# Title: webctl css command tests

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
title "webctl css Command Test Suite"
echo "Project: P-052"
echo "Tests CSS extraction with all subcommands: save/computed/get/inline/matched"
echo "and flags: --select (rule filtering), --find, context flags"
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

heading "Extract all stylesheets to stdout"
cmd "webctl css"

echo ""
echo "Verify: CSS output to stdout"
echo "Verify: No file created"
read -p "Press Enter to continue..."

heading "Filter CSS rules by selector to stdout"
cmd "webctl css --select \"body\""

echo ""
echo "Verify: CSS rules with 'body' in selector shown"
echo "Note: --select now filters stylesheet rules, not computed styles"
read -p "Press Enter to continue..."

heading "Search for CSS text"
cmd "webctl css --find \"color\""

echo ""
echo "Verify: CSS containing 'color' shown"
read -p "Press Enter to continue..."

# Save mode tests
title "Save Mode (File Output)"

heading "Save to temp (no path)"
cmd "webctl css save"

echo ""
echo "Verify: File saved to /tmp/webctl-css/"
echo "Verify: Auto-generated filename with timestamp"
echo "Verify: JSON response shows file path"
read -p "Press Enter to continue..."

heading "Save to custom file"
cmd "webctl css save ./styles.css"

echo ""
echo "Verify: File saved to ./styles.css"
read -p "Press Enter to continue..."

heading "Save to directory with auto-filename (trailing slash = directory)"
cmd "webctl css save ./output/"

echo ""
echo "Verify: File saved to ./output/ with auto-generated name"
echo "Note: Trailing slash (/) is REQUIRED for directory behavior"
echo "      Without slash, it would create a file named 'output'"
read -p "Press Enter to continue..."

heading "Save with selector and find"
cmd "webctl css save ./debug.css --select \"div\" --find \"background\""

echo ""
echo "Verify: File saved with CSS rules matching 'div' containing 'background'"
read -p "Press Enter to continue..."

# Computed mode tests (supports multiple elements with -- separators)
title "Computed Mode (Get All Computed Styles)"

heading "Get all computed styles for body"
cmd "webctl css computed \"body\""

echo ""
echo "Verify: All computed CSS properties shown"
echo "Verify: Format is 'property: value' per line"
read -p "Press Enter to continue..."

heading "Get computed styles for h1 (may match multiple elements)"
cmd "webctl css computed \"h1\""

echo ""
echo "Verify: h1 computed styles shown"
echo "If multiple h1 elements exist, separated by '--'"
read -p "Press Enter to continue..."

heading "Get computed styles for multiple elements"
cmd "webctl css computed \"p\""

echo ""
echo "Verify: Computed styles for ALL p elements"
echo "Multiple elements separated by '--'"
read -p "Press Enter to continue..."

heading "Get computed styles with complex selector"
cmd "webctl css computed \"body > div\""

echo ""
echo "Verify: Computed styles for matching elements"
echo "Multiple matches separated by '--'"
read -p "Press Enter to continue..."

heading "Get computed styles as JSON (array format)"
cmd "webctl css computed \"body\" --json"

echo ""
echo "Verify: JSON formatted output with styles ARRAY"
echo "Format: {\"ok\": true, \"styles\": [{...}, {...}]}"
read -p "Press Enter to continue..."

# Get mode tests
title "Get Mode (Get Single Property)"

heading "Get background-color of body"
cmd "webctl css get \"body\" background-color"

echo ""
echo "Verify: Plain value output (e.g., 'rgba(255, 255, 255, 1)')"
read -p "Press Enter to continue..."

heading "Get display property of h1"
cmd "webctl css get \"h1\" display"

echo ""
echo "Verify: Plain value output (e.g., 'block')"
read -p "Press Enter to continue..."

heading "Get font-size of body"
cmd "webctl css get \"body\" font-size"

echo ""
echo "Verify: Font size value (e.g., '16px')"
read -p "Press Enter to continue..."

heading "Get invalid property (should error)"
cmd "webctl css get \"body\" nonexistent-property"

echo ""
echo "Verify: Error or empty value"
read -p "Press Enter to continue..."

# Select flag tests (NEW: filters CSS rules by selector pattern)
title "Select Flag Tests"

heading "Select by element type"
cmd "webctl css --select \"body\""

echo ""
echo "Verify: CSS rules with 'body' in selector"
echo "Note: --select now filters stylesheet rules, not computed styles"
read -p "Press Enter to continue..."

heading "Select by class pattern"
cmd "webctl css --select \".container\""

echo ""
echo "Verify: CSS rules with '.container' in selector"
read -p "Press Enter to continue..."

heading "Select by ID pattern"
cmd "webctl css --select \"#main\""

echo ""
echo "Verify: CSS rules with '#main' in selector"
read -p "Press Enter to continue..."

heading "Select by complex pattern"
cmd "webctl css --select \"h1\""

echo ""
echo "Verify: CSS rules matching h1, .class h1, div h1, etc."
read -p "Press Enter to continue..."

# Find flag tests
title "Find Flag Tests"

heading "Find text in CSS"
cmd "webctl css --find \"background\""

echo ""
echo "Verify: CSS rules containing 'background'"
read -p "Press Enter to continue..."

heading "Find with no matches (should error)"
cmd "webctl css --find \"ThisTextDoesNotExist123\""

echo ""
echo "Verify: Error message about no matches"
read -p "Press Enter to continue..."

heading "Find combined with select"
cmd "webctl css --select \"body\" --find \"color\""

echo ""
echo "Verify: CSS rules with 'body' selector containing 'color'"
read -p "Press Enter to continue..."

# Context flag tests
title "Context Flag Tests (-A, -B, -C)"

heading "Find with after context (-A)"
cmd "webctl css --find \"color\" -A 3"

echo ""
echo "Verify: Matching lines plus 3 lines after each match"
read -p "Press Enter to continue..."

heading "Find with before context (-B)"
cmd "webctl css --find \"color\" -B 3"

echo ""
echo "Verify: Matching lines plus 3 lines before each match"
read -p "Press Enter to continue..."

heading "Find with symmetric context (-C)"
cmd "webctl css --find \"background\" -C 2"

echo ""
echo "Verify: Matching lines plus 2 lines before AND after"
echo "Note: -C 2 is shorthand for -B 2 -A 2"
read -p "Press Enter to continue..."

heading "Find with asymmetric context (-B and -A)"
cmd "webctl css --find \"margin\" -B 1 -A 5"

echo ""
echo "Verify: 1 line before, 5 lines after each match"
read -p "Press Enter to continue..."

heading "Context with multiple matches (should merge overlapping)"
cmd "webctl css --find \"px\" -C 1"

echo ""
echo "Verify: Adjacent/overlapping contexts are merged"
echo "Verify: Non-adjacent regions separated by '--'"
read -p "Press Enter to continue..."

heading "Context to capture full CSS rule"
cmd "webctl css --find \"body\" -A 5"

echo ""
echo "Verify: Shows selector line plus following properties"
echo "Useful for seeing complete rules containing search term"
read -p "Press Enter to continue..."

# Inline mode tests (NEW)
title "Inline Mode Tests"

heading "Get inline styles for all elements with style attribute"
cmd "webctl css inline \"[style]\""

echo ""
echo "Verify: Inline style attribute content for each matching element"
echo "Multiple elements separated by '--'"
read -p "Press Enter to continue..."

heading "Get inline styles for specific element"
cmd "webctl css inline \"body\""

echo ""
echo "Verify: Shows body inline style (may be empty)"
read -p "Press Enter to continue..."

heading "Get inline styles for multiple elements"
cmd "webctl css inline \"div\""

echo ""
echo "Verify: Inline styles for all div elements, separated by '--'"
read -p "Press Enter to continue..."

heading "Inline styles as JSON"
cmd "webctl css inline \"[style]\" --json"

echo ""
echo "Verify: JSON array of inline style strings"
read -p "Press Enter to continue..."

# Matched mode tests (NEW)
title "Matched Mode Tests"

heading "Get matched CSS rules for element"
cmd "webctl css matched \"body\""

echo ""
echo "Verify: CSS rules from stylesheets that apply to body element"
echo "Each rule shows selector and properties, separated by '--'"
read -p "Press Enter to continue..."

heading "Get matched CSS rules for specific element"
cmd "webctl css matched \"h1\""

echo ""
echo "Verify: CSS rules that apply to h1 element"
read -p "Press Enter to continue..."

heading "Get matched CSS rules with ID selector"
cmd "webctl css matched \"#main\""

echo ""
echo "Verify: CSS rules matching #main (if exists)"
echo "Note: Adjust selector if #main doesn't exist on test page"
read -p "Press Enter to continue..."

heading "Matched rules as JSON"
cmd "webctl css matched \"body\" --json"

echo ""
echo "Verify: JSON array of matched rules with selector and properties"
read -p "Press Enter to continue..."

# Raw flag tests
title "Raw Flag Tests"

heading "Raw output (no formatting)"
cmd "webctl css --raw"

echo ""
echo "Verify: Unformatted CSS as-is from browser"
read -p "Press Enter to continue..."

heading "Raw with select"
cmd "webctl css --raw --select \"h1\""

echo ""
echo "Verify: Raw CSS rules matching h1 selector"
read -p "Press Enter to continue..."

# Output format tests
title "Output Format Tests"

heading "JSON output"
cmd "webctl css --json"

echo ""
echo "Verify: JSON formatted output"
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl css --no-color"

echo ""
echo "Verify: No ANSI color codes"
read -p "Press Enter to continue..."

heading "Debug verbose output"
cmd "webctl css --debug"

echo ""
echo "Verify: Debug logging information"
read -p "Press Enter to continue..."

# Error cases
title "Error Cases"

heading "Selector matches no elements (computed mode)"
cmd "webctl css computed \"#nonexistent-id-xyz\""

echo ""
echo "Verify: Error message about selector not matching"
read -p "Press Enter to continue..."

heading "Selector matches no elements (get mode)"
cmd "webctl css get \"#nonexistent-id-xyz\" color"

echo ""
echo "Verify: Error message about selector not matching"
read -p "Press Enter to continue..."

heading "Save to invalid path"
cmd "webctl css save /root/invalid/path.css"

echo ""
echo "Verify: Permission denied or path error"
read -p "Press Enter to continue..."

# REPL mode tests
title "REPL Mode Tests"

heading "Test css in REPL"
echo "Switch to daemon terminal and execute:"
cmd "css"

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

heading "Test css computed in REPL"
echo "In REPL, try:"
cmd "css computed \"body\""

echo ""
echo "Should show all computed styles"
read -p "Press Enter when tested in REPL..."

heading "Test css get in REPL"
echo "In REPL, try:"
cmd "css get \"body\" background-color"

echo ""
echo "Should show background-color value"
read -p "Press Enter when tested in REPL..."

# Advanced scenarios
title "Advanced Scenarios"

heading "Navigate to styled page"
cmd "webctl navigate https://github.com --wait"

echo ""
echo "Wait for GitHub to load"
read -p "Press Enter when loaded..."

heading "Get header background color"
cmd "webctl css get \"header\" background-color"

echo ""
echo "Verify: Background color value"
read -p "Press Enter to continue..."

heading "Extract button computed styles"
cmd "webctl css computed \"button\""

echo ""
echo "Verify: Button styles shown"
read -p "Press Enter to continue..."

heading "Save all stylesheets"
cmd "webctl css save ./github-styles.css"

echo ""
echo "Verify: All GitHub CSS saved"
read -p "Press Enter to continue..."

# Cleanup
title "Cleanup"
echo "Clean up test files if desired:"
cmd "rm -f ./styles.css ./debug.css ./output/*.css ./github-styles.css"

echo ""
echo "Remove test files when ready"
read -p "Press Enter to finish..."

title "Test Suite Complete"
echo "All css command tests finished"
echo ""
echo "Review checklist in docs/projects/p-035-testing-css.md"
echo "Document any issues discovered during testing"
