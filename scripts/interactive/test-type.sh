#!/bin/bash
# Title: webctl type command tests

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
title "webctl type Command Test Suite"
echo "Project: P-041"
echo "Tests typing text into elements with optional clearing and key sending"
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

heading "Navigate to httpbin.org/forms/post (has form inputs)"
cmd "webctl navigate http://httpbin.org/forms/post --wait"

echo ""
echo "Wait for page to load"
read -p "Press Enter when page loaded..."

# Basic typing with selector
title "Basic Typing (With Selector)"

heading "Type into customer name field"
cmd "webctl type \"input[name=custname]\" \"John Doe\""

echo ""
echo "Verify: 'John Doe' appears in customer name field"
read -p "Press Enter to continue..."

heading "Type into telephone field"
cmd "webctl type \"input[name=custtel]\" \"555-1234\""

echo ""
echo "Verify: '555-1234' appears in telephone field"
read -p "Press Enter to continue..."

heading "Type into email field"
cmd "webctl type \"input[name=custemail]\" \"john@example.com\""

echo ""
echo "Verify: 'john@example.com' appears in email field"
read -p "Press Enter to continue..."

heading "Type into textarea"
cmd "webctl type \"textarea[name=comments]\" \"This is a test comment\""

echo ""
echo "Verify: Text appears in comments textarea"
read -p "Press Enter to continue..."

# Basic typing without selector
title "Basic Typing (Without Selector - Focused Element)"

heading "Focus an input field first"
cmd "webctl focus \"input[name=custname]\""

echo ""
echo "Verify: Customer name field focused"
read -p "Press Enter to continue..."

heading "Type into focused element (no selector)"
cmd "webctl type \"Jane Smith\""

echo ""
echo "Verify: 'Jane Smith' typed into focused field"
echo "Note: This appends to existing text"
read -p "Press Enter to continue..."

# Clear flag
title "Clear Flag Tests"

heading "Clear and replace customer name"
cmd "webctl type \"input[name=custname]\" \"Alice Johnson\" --clear"

echo ""
echo "Verify: Previous text cleared, 'Alice Johnson' now in field"
read -p "Press Enter to continue..."

heading "Clear and replace email"
cmd "webctl type \"input[name=custemail]\" \"alice@example.com\" --clear"

echo ""
echo "Verify: Previous email cleared, 'alice@example.com' now in field"
read -p "Press Enter to continue..."

heading "Clear empty field (should work without error)"
cmd "webctl type \"input[name=custtel]\" \"\" --clear"

echo ""
echo "Verify: Field cleared, no error"
read -p "Press Enter to continue..."

heading "Type into cleared field"
cmd "webctl type \"input[name=custtel]\" \"555-5678\" --clear"

echo ""
echo "Verify: New phone number in field"
read -p "Press Enter to continue..."

# Key flag (submit actions)
title "Key Flag - Submit Actions"

heading "Clear form and prepare for key flag test"
cmd "webctl reload"

echo ""
echo "Wait for page to reload"
read -p "Press Enter when loaded..."

heading "Type into search-like field with Enter key"
echo "Note: This form doesn't have a search field, but we'll use email field"
cmd "webctl type \"input[name=custemail]\" \"test@example.com\" --key Enter"

echo ""
echo "Verify: Text typed and Enter sent (may trigger form validation)"
read -p "Press Enter to continue..."

# Key flag (Tab navigation)
title "Key Flag - Tab Navigation"

heading "Reload page for clean state"
cmd "webctl reload"

echo ""
read -p "Press Enter when loaded..."

heading "Type into first field and Tab to next"
cmd "webctl type \"input[name=custname]\" \"Bob Smith\" --key Tab"

echo ""
echo "Verify: Name typed and focus moved to next field (telephone)"
read -p "Press Enter to continue..."

heading "Type into focused field (no selector) and Tab again"
cmd "webctl type \"555-9999\" --key Tab"

echo ""
echo "Verify: Phone typed and focus moved to email field"
read -p "Press Enter to continue..."

heading "Type into focused field and submit with Enter"
cmd "webctl type \"bob@example.com\" --key Enter"

echo ""
echo "Verify: Email typed and form attempted submission"
read -p "Press Enter to continue..."

# Combined flags
title "Combined Flags (Clear + Key)"

heading "Navigate back to form"
cmd "webctl navigate http://httpbin.org/forms/post --wait"

echo ""
read -p "Press Enter when loaded..."

heading "Clear, type, and submit with Enter"
cmd "webctl type \"input[name=custemail]\" \"combined@example.com\" --clear --key Enter"

echo ""
echo "Verify: Field cleared, new text typed, Enter sent"
read -p "Press Enter to continue..."

heading "Reload for next test"
cmd "webctl reload"

echo ""
read -p "Press Enter when loaded..."

heading "Clear, type, and Tab to next field"
cmd "webctl type \"input[name=custname]\" \"Test User\" --clear --key Tab"

echo ""
echo "Verify: Field cleared, text typed, focus moved to next field"
read -p "Press Enter to continue..."

# Different input types
title "Different Input Types"

heading "Type into text input (already tested)"
echo "Skip - already tested with custname field"
read -p "Press Enter to continue..."

heading "Navigate to page with different input types"
cmd "webctl navigate https://www.w3schools.com/html/html_form_input_types.asp --wait"

echo ""
echo "This page may have various input types for testing"
echo "If page has moved, use appropriate test page or skip"
read -p "Press Enter to continue or skip..."

heading "Navigate back to reliable form"
cmd "webctl navigate http://httpbin.org/forms/post --wait"

echo ""
read -p "Press Enter when loaded..."

# Special characters
title "Special Characters"

heading "Type text with spaces"
cmd "webctl type \"input[name=custname]\" \"First Last\" --clear"

echo ""
echo "Verify: Text with spaces typed correctly"
read -p "Press Enter to continue..."

heading "Type text with punctuation"
cmd "webctl type \"input[name=custemail]\" \"user+test@example.com\" --clear"

echo ""
echo "Verify: Email with + character typed correctly"
read -p "Press Enter to continue..."

heading "Type text with special characters"
cmd "webctl type \"textarea[name=comments]\" \"Testing: !@#$%^&*()\" --clear"

echo ""
echo "Verify: Special characters typed correctly"
read -p "Press Enter to continue..."

heading "Type text with quotes"
cmd "webctl type \"textarea[name=comments]\" \"She said \\\"hello\\\" to me\" --clear"

echo ""
echo "Verify: Text with quotes typed correctly"
read -p "Press Enter to continue..."

heading "Type multiline text in textarea"
cmd "webctl type \"textarea[name=comments]\" \"Line 1\nLine 2\nLine 3\" --clear"

echo ""
echo "Verify: Multiline text in textarea (may show as single line depending on rendering)"
read -p "Press Enter to continue..."

# Form workflow
title "Complete Form Workflow"

heading "Reload page for clean form"
cmd "webctl reload"

echo ""
read -p "Press Enter when loaded..."

heading "Fill complete form with type commands"
echo "Step 1: Type customer name with Tab"
cmd "webctl type \"input[name=custname]\" \"Complete Test\" --clear --key Tab"

echo ""
read -p "Press Enter to continue..."

heading "Step 2: Type telephone with Tab"
cmd "webctl type \"555-0000\" --key Tab"

echo ""
read -p "Press Enter to continue..."

heading "Step 3: Type email with Tab"
cmd "webctl type \"complete@example.com\" --key Tab"

echo ""
read -p "Press Enter to continue..."

heading "Step 4: Type comments and submit"
cmd "webctl type \"textarea[name=comments]\" \"Final test comments\" --clear"

echo ""
read -p "Press Enter to continue..."

heading "Step 5: Submit form by clicking submit button"
cmd "webctl click \"button[type=submit]\""

echo ""
echo "Verify: Complete form submitted successfully"
read -p "Press Enter to continue..."

# Error cases
title "Error Cases"

heading "Type with non-existent selector"
cmd "webctl type \"#nonexistent-input\" \"text\""

echo ""
echo "Verify: Error message 'element not found'"
read -p "Press Enter to continue..."

heading "Type with invalid CSS selector"
cmd "webctl type \"[[invalid]]\" \"text\""

echo ""
echo "Verify: Error message about invalid selector"
read -p "Press Enter to continue..."

heading "Type with invalid --key value"
cmd "webctl type \"input[name=custname]\" \"text\" --key InvalidKey"

echo ""
echo "Verify: Error message about invalid key"
read -p "Press Enter to continue..."

heading "Navigate to page and try typing into non-focusable element"
cmd "webctl navigate https://example.com --wait"

echo ""
read -p "Press Enter when loaded..."

cmd "webctl type \"p\" \"text\""

echo ""
echo "Verify: Error message 'element is not focusable'"
read -p "Press Enter to continue..."

# Output formats
title "Output Format Tests"

heading "Navigate back to form"
cmd "webctl navigate http://httpbin.org/forms/post --wait"

echo ""
read -p "Press Enter when loaded..."

heading "JSON output"
cmd "webctl type \"input[name=custname]\" \"JSON Test\" --json --clear"

echo ""
echo "Verify: JSON formatted output"
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl type \"input[name=custname]\" \"No Color\" --no-color --clear"

echo ""
echo "Verify: No ANSI color codes"
read -p "Press Enter to continue..."

heading "Debug output"
cmd "webctl type \"input[name=custname]\" \"Debug Test\" --debug --clear"

echo ""
echo "Verify: Debug logging information"
read -p "Press Enter to continue..."

# REPL mode tests
title "REPL Mode Tests"

heading "Test type in REPL"
echo "Switch to daemon terminal and execute:"
cmd "type \"input[name=custname]\" \"REPL Test\" --clear"

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

heading "Test type with flags in REPL"
echo "In REPL, try:"
cmd "type \"input[name=custemail]\" \"repl@example.com\" --clear --key Tab"

echo ""
echo "Should clear, type, and tab to next field"
read -p "Press Enter when tested in REPL..."

heading "Test type without selector in REPL"
echo "First focus a field, then in REPL, try:"
cmd "focus \"input[name=custname]\""

echo ""
read -p "Press Enter to continue..."

echo "Then type without selector:"
cmd "type \"Focused typing\""

echo ""
echo "Should type into focused field"
read -p "Press Enter when tested in REPL..."

# Test completed
title "Test Suite Complete"
echo "All type command tests finished"
echo ""
echo "Review checklist in docs/projects/p-041-testing-type.md"
echo "Document any issues discovered during testing"
