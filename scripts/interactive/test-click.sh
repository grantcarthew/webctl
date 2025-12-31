#!/bin/bash
# Title: webctl click command tests

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
title "webctl click Command Test Suite"
echo "Project: P-040"
echo "Tests element clicking with automatic scrolling and full event chain"
echo ""
echo "Prerequisites:"
echo "  - webctl must be built"
echo "  - xclip must be installed"
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

heading "Navigate to httpbin.org/forms/post (has form with buttons and inputs)"
cmd "webctl navigate http://httpbin.org/forms/post --wait"

echo ""
echo "Wait for page to load (has form elements for clicking)"
read -p "Press Enter when page loaded..."

# Basic element clicking
title "Basic Element Clicking"

heading "Click submit button by attribute selector"
cmd "webctl click \"button[type=submit]\""

echo ""
echo "Verify: Form submitted (page may reload or show result)"
echo "Reload page if needed to reset form state"
read -p "Press Enter to continue..."

heading "Navigate back to form"
cmd "webctl navigate http://httpbin.org/forms/post --wait"

echo ""
read -p "Press Enter when page loaded..."

heading "Click input field (focuses it)"
cmd "webctl click \"input[name=custname]\""

echo ""
echo "Verify: Input field receives focus"
read -p "Press Enter to continue..."

heading "Click checkbox input"
cmd "webctl click \"input[type=checkbox][value=bacon]\""

echo ""
echo "Verify: Checkbox toggles (check or uncheck)"
read -p "Press Enter to continue..."

heading "Click checkbox again to toggle"
cmd "webctl click \"input[type=checkbox][value=bacon]\""

echo ""
echo "Verify: Checkbox toggles back"
read -p "Press Enter to continue..."

heading "Click radio button"
cmd "webctl click \"input[type=radio][value=small]\""

echo ""
echo "Verify: Radio button selected"
read -p "Press Enter to continue..."

heading "Click different radio button in same group"
cmd "webctl click \"input[type=radio][value=large]\""

echo ""
echo "Verify: New radio selected, previous deselected"
read -p "Press Enter to continue..."

# Different element types
title "Different Element Types"

heading "Click text input to focus"
cmd "webctl click \"input[name=custtel]\""

echo ""
echo "Verify: Input focused"
read -p "Press Enter to continue..."

heading "Click textarea to focus"
cmd "webctl click \"textarea[name=comments]\""

echo ""
echo "Verify: Textarea focused"
read -p "Press Enter to continue..."

# Complex selectors
title "Complex Selectors"

heading "Navigate to page with multiple elements"
cmd "webctl navigate https://example.com --wait"

echo ""
read -p "Press Enter when page loaded..."

heading "Click link (will navigate)"
cmd "webctl click \"a\""

echo ""
echo "Verify: Navigation occurred"
echo "Note: Page likely navigated to IANA website"
read -p "Press Enter to continue..."

heading "Navigate back"
cmd "webctl back"

echo ""
read -p "Press Enter to continue..."

# Auto-scrolling behavior
title "Auto-Scrolling Behavior"

heading "Navigate to long page for scroll testing"
cmd "webctl navigate https://github.com --wait"

echo ""
read -p "Press Enter when page loaded..."

heading "Scroll to top first"
cmd "webctl scroll --to 0,0"

echo ""
read -p "Press Enter to continue..."

heading "Click element at bottom of page (footer)"
echo "This should scroll the element into view before clicking"
cmd "webctl click \"footer a\""

echo ""
echo "Verify: Page scrolled down to show footer"
echo "Verify: Link clicked (may navigate)"
read -p "Press Enter to continue..."

# Form interactions workflow
title "Form Interactions Workflow"

heading "Navigate back to form"
cmd "webctl navigate http://httpbin.org/forms/post --wait"

echo ""
read -p "Press Enter when page loaded..."

heading "Fill and submit complete form"
echo "Step 1: Click and type into customer name"
cmd "webctl click \"input[name=custname]\""

echo ""
read -p "Press Enter to continue..."

cmd "webctl type \"John Doe\""

echo ""
echo "Verify: Name typed into field"
read -p "Press Enter to continue..."

heading "Step 2: Click and type into telephone"
cmd "webctl click \"input[name=custtel]\""

echo ""
read -p "Press Enter to continue..."

cmd "webctl type \"555-1234\""

echo ""
read -p "Press Enter to continue..."

heading "Step 3: Click and type into email"
cmd "webctl click \"input[name=custemail]\""

echo ""
read -p "Press Enter to continue..."

cmd "webctl type \"john@example.com\""

echo ""
read -p "Press Enter to continue..."

heading "Step 4: Select pizza size"
cmd "webctl click \"input[type=radio][value=medium]\""

echo ""
echo "Verify: Medium size selected"
read -p "Press Enter to continue..."

heading "Step 5: Select toppings"
cmd "webctl click \"input[type=checkbox][value=bacon]\""

echo ""
read -p "Press Enter to continue..."

cmd "webctl click \"input[type=checkbox][value=cheese]\""

echo ""
read -p "Press Enter to continue..."

heading "Step 6: Click and add comments"
cmd "webctl click \"textarea[name=comments]\""

echo ""
read -p "Press Enter to continue..."

cmd "webctl type \"Please deliver by 6pm\""

echo ""
read -p "Press Enter to continue..."

heading "Step 7: Submit form"
cmd "webctl click \"button[type=submit]\""

echo ""
echo "Verify: Form submitted successfully"
echo "Verify: Page shows form data"
read -p "Press Enter to continue..."

# Error cases
title "Error Cases"

heading "Click non-existent selector"
cmd "webctl click \"#nonexistent-element-xyz\""

echo ""
echo "Verify: Error message 'element not found'"
read -p "Press Enter to continue..."

heading "Click with invalid CSS selector"
cmd "webctl click \"[[invalid]]\""

echo ""
echo "Verify: Error message about invalid selector"
read -p "Press Enter to continue..."

heading "Click with empty selector"
cmd "webctl click \"\""

echo ""
echo "Verify: Error message"
read -p "Press Enter to continue..."

# Output formats
title "Output Format Tests"

heading "Navigate back to example.com"
cmd "webctl navigate https://example.com --wait"

echo ""
read -p "Press Enter when loaded..."

heading "JSON output"
cmd "webctl click \"a\" --json"

echo ""
echo "Verify: JSON formatted output"
read -p "Press Enter to continue..."

heading "Navigate back"
cmd "webctl back"

echo ""
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl click \"a\" --no-color"

echo ""
echo "Verify: No ANSI color codes"
read -p "Press Enter to continue..."

heading "Navigate back"
cmd "webctl back"

echo ""
read -p "Press Enter to continue..."

heading "Debug output"
cmd "webctl click \"a\" --debug"

echo ""
echo "Verify: Debug logging information"
read -p "Press Enter to continue..."

# REPL mode tests
title "REPL Mode Tests"

heading "Test click in REPL"
cmd "webctl navigate http://httpbin.org/forms/post --wait"

echo ""
echo "Wait for page to load"
read -p "Press Enter when loaded..."

echo ""
echo "Switch to daemon terminal and execute:"
cmd "click \"input[type=checkbox][value=bacon]\""

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

heading "Test click with different elements in REPL"
echo "In REPL, try:"
cmd "click \"input[type=radio][value=large]\""

echo ""
echo "Should select radio button from REPL"
read -p "Press Enter when tested in REPL..."

heading "Test click submit in REPL"
echo "In REPL, try:"
cmd "click \"button[type=submit]\""

echo ""
echo "Should submit form from REPL"
read -p "Press Enter when tested in REPL..."

# Overlapping elements test
title "Overlapping Elements Test"

heading "Navigate to page with overlays (GitHub)"
cmd "webctl navigate https://github.com --wait"

echo ""
read -p "Press Enter when loaded..."

heading "Try clicking element that might be covered"
echo "Note: This test depends on page structure and may not show warning"
cmd "webctl click \"footer a\""

echo ""
echo "Verify: Click succeeds"
echo "Verify: Warning shown if element covered (not always present)"
read -p "Press Enter to continue..."

# Test completed
title "Test Suite Complete"
echo "All click command tests finished"
echo ""
echo "Review checklist in docs/projects/p-040-testing-click.md"
echo "Document any issues discovered during testing"
