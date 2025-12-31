#!/bin/bash
# Title: webctl select command tests

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
title "webctl select Command Test Suite"
echo "Project: P-042"
echo "Tests selecting options in native HTML select dropdowns"
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

heading "Navigate to W3Schools select example (or similar page with select elements)"
cmd "webctl navigate https://www.w3schools.com/tags/tryit.asp?filename=tryhtml_select --wait"

echo ""
echo "This page has a select dropdown for testing"
echo "Wait for page to load"
read -p "Press Enter when page loaded..."

# Note: W3Schools tryit pages have iframes - may not work with current webctl
# Let's create a better test setup

heading "Actually, navigate to a simpler page with select elements"
cmd "webctl navigate http://httpbin.org/forms/post --wait"

echo ""
echo "This form has select elements we can test"
read -p "Press Enter when page loaded..."

# First, let's check what select elements exist on this page
heading "Check page structure"
echo "Let's see what select elements are available"
cmd "webctl html show --select \"select\""

echo ""
echo "Review the select elements in output"
echo "Note: httpbin form may have delivery time select"
read -p "Press Enter to continue..."

# Basic selection tests
title "Basic Selection Tests"

heading "Select an option from delivery time dropdown"
echo "First, let's check the select element name/id"
echo "From the form, we'll select by name attribute"
cmd "webctl select \"select[name=delivery]\" \"\""

echo ""
echo "Verify: Empty/default option selected"
echo "Note: May error if empty value doesn't exist"
read -p "Press Enter to continue..."

heading "Select specific delivery time"
cmd "webctl select \"select[name=delivery]\" \"12pm\""

echo ""
echo "Verify: 12pm option selected in dropdown"
read -p "Press Enter to continue..."

heading "Select different delivery time"
cmd "webctl select \"select[name=delivery]\" \"8am\""

echo ""
echo "Verify: 8am option selected, 12pm deselected"
read -p "Press Enter to continue..."

heading "Select another time"
cmd "webctl select \"select[name=delivery]\" \"4pm\""

echo ""
echo "Verify: 4pm option selected"
read -p "Press Enter to continue..."

# Verify selection with eval
title "Verify Selection with eval"

heading "Check selected value with JavaScript"
cmd "webctl eval \"document.querySelector('select[name=delivery]').value\""

echo ""
echo "Verify: Output shows '4pm' (last selected value)"
read -p "Press Enter to continue..."

heading "Check selectedIndex"
cmd "webctl eval \"document.querySelector('select[name=delivery]').selectedIndex\""

echo ""
echo "Verify: Output shows index of selected option"
read -p "Press Enter to continue..."

# Different selector types
title "Different Selector Types"

heading "Select by element tag only"
cmd "webctl select \"select\" \"12pm\""

echo ""
echo "Verify: First select element's option changed to 12pm"
read -p "Press Enter to continue..."

heading "Select by name attribute"
cmd "webctl select \"select[name=delivery]\" \"8am\""

echo ""
echo "Verify: Specific select by name changed to 8am"
read -p "Press Enter to continue..."

# Change event verification
title "Change Event Verification"

heading "Select option and check if change event fired"
echo "We'll select a new option to trigger change event"
cmd "webctl select \"select[name=delivery]\" \"4pm\""

echo ""
echo "Verify: Change event fired (may trigger form validation or scripts)"
read -p "Press Enter to continue..."

# Form workflow with select
title "Form Workflow with Select"

heading "Fill complete form including select"
cmd "webctl type \"input[name=custname]\" \"Select Test User\" --clear"

echo ""
read -p "Press Enter to continue..."

cmd "webctl type \"input[name=custtel]\" \"555-1111\" --clear"

echo ""
read -p "Press Enter to continue..."

cmd "webctl type \"input[name=custemail]\" \"select@example.com\" --clear"

echo ""
read -p "Press Enter to continue..."

cmd "webctl select \"select[name=delivery]\" \"12pm\""

echo ""
read -p "Press Enter to continue..."

heading "Submit form with select value"
cmd "webctl click \"button[type=submit]\""

echo ""
echo "Verify: Form submitted with selected delivery time"
echo "Check result shows delivery: 12pm"
read -p "Press Enter to continue..."

# Navigate to page with more select elements
title "Testing with Different Select Elements"

heading "Navigate to page with multiple selects if available"
echo "Note: We'll use a page that's known to have select elements"
echo "For comprehensive testing, you may need to serve a custom HTML page"
cmd "webctl navigate http://httpbin.org/forms/post --wait"

echo ""
read -p "Press Enter when loaded..."

# Error cases
title "Error Cases"

heading "Select with non-existent selector"
cmd "webctl select \"#nonexistent-select\" \"value\""

echo ""
echo "Verify: Error message 'element not found'"
read -p "Press Enter to continue..."

heading "Select with invalid CSS selector"
cmd "webctl select \"[[invalid]]\" \"value\""

echo ""
echo "Verify: Error message about invalid selector"
read -p "Press Enter to continue..."

heading "Select on non-select element (try with input)"
cmd "webctl select \"input[name=custname]\" \"value\""

echo ""
echo "Verify: Error message 'element is not a select'"
read -p "Press Enter to continue..."

heading "Select on button element"
cmd "webctl select \"button\" \"value\""

echo ""
echo "Verify: Error message 'element is not a select'"
read -p "Press Enter to continue..."

heading "Select with empty selector"
cmd "webctl select \"\" \"value\""

echo ""
echo "Verify: Error message"
read -p "Press Enter to continue..."

heading "Select with non-existent option value"
cmd "webctl select \"select[name=delivery]\" \"nonexistent-time\""

echo ""
echo "Verify: May succeed with no change or show error"
echo "Behavior depends on browser implementation"
read -p "Press Enter to continue..."

# Output formats
title "Output Format Tests"

heading "JSON output"
cmd "webctl select \"select[name=delivery]\" \"8am\" --json"

echo ""
echo "Verify: JSON formatted output"
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl select \"select[name=delivery]\" \"12pm\" --no-color"

echo ""
echo "Verify: No ANSI color codes"
read -p "Press Enter to continue..."

heading "Debug output"
cmd "webctl select \"select[name=delivery]\" \"4pm\" --debug"

echo ""
echo "Verify: Debug logging information"
read -p "Press Enter to continue..."

# REPL mode tests
title "REPL Mode Tests"

heading "Test select in REPL"
echo "Switch to daemon terminal and execute:"
cmd "select \"select[name=delivery]\" \"8am\""

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

heading "Test select with different value in REPL"
echo "In REPL, try:"
cmd "select \"select[name=delivery]\" \"12pm\""

echo ""
echo "Should change selection to 12pm"
read -p "Press Enter when tested in REPL..."

heading "Verify selection in REPL with eval"
echo "In REPL, verify with:"
cmd "eval \"document.querySelector('select[name=delivery]').value\""

echo ""
echo "Should show '12pm'"
read -p "Press Enter when tested in REPL..."

# Advanced scenarios
title "Advanced Scenarios"

heading "Custom HTML page with multiple selects (if available)"
echo "For more comprehensive testing, create a custom HTML page with:"
echo "  - Multiple select elements"
echo "  - Selects with different value types"
echo "  - Dependent selects (country -> state)"
echo ""
echo "You can serve such a page with: webctl serve <directory>"
echo ""
echo "Skip this section if custom page not available"
read -p "Press Enter to continue..."

# Note about custom dropdowns
title "Note: Custom Dropdowns"

heading "Custom JavaScript dropdowns won't work with select command"
echo "The select command ONLY works with native HTML <select> elements"
echo ""
echo "For custom dropdowns (React Select, Material UI, etc.), use:"
echo "  1. click to open dropdown"
echo "  2. type to filter/search (if applicable)"
echo "  3. click on option or key Enter to select"
echo ""
echo "Example custom dropdown workflow:"
echo "  webctl click \".custom-dropdown\""
echo "  webctl type \"Australia\""
echo "  webctl key Enter"
echo ""
read -p "Press Enter to continue..."

# Test completed
title "Test Suite Complete"
echo "All select command tests finished"
echo ""
echo "Review checklist in docs/projects/p-042-testing-select.md"
echo "Document any issues discovered during testing"
echo ""
echo "Note: For more comprehensive testing, consider creating a custom HTML page"
echo "with multiple select elements of different types and serving it with webctl serve"
