#!/bin/bash
# Title: webctl focus command tests

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
title "webctl focus Command Test Suite"
echo "Project: P-044"
echo "Tests focusing elements with CSS selectors"
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

heading "Navigate to httpbin.org/forms/post (has focusable form elements)"
cmd "webctl navigate http://httpbin.org/forms/post --wait"

echo ""
echo "Wait for page to load"
read -p "Press Enter when page loaded..."

# Basic focusing
title "Basic Focusing"

heading "Focus customer name input"
cmd "webctl focus \"input[name=custname]\""

echo ""
echo "Verify: Input field focused (visible focus ring/outline)"
read -p "Press Enter to continue..."

heading "Focus telephone input"
cmd "webctl focus \"input[name=custtel]\""

echo ""
echo "Verify: Telephone field focused, previous field blurred"
read -p "Press Enter to continue..."

heading "Focus email input"
cmd "webctl focus \"input[name=custemail]\""

echo ""
echo "Verify: Email field focused"
read -p "Press Enter to continue..."

heading "Focus textarea"
cmd "webctl focus \"textarea[name=comments]\""

echo ""
echo "Verify: Textarea focused"
read -p "Press Enter to continue..."

heading "Focus submit button"
cmd "webctl focus \"button[type=submit]\""

echo ""
echo "Verify: Submit button focused"
read -p "Press Enter to continue..."

# Different input types
title "Different Input Types"

heading "Focus checkbox"
cmd "webctl focus \"input[type=checkbox][value=bacon]\""

echo ""
echo "Verify: Checkbox focused (may have focus ring)"
read -p "Press Enter to continue..."

heading "Focus radio button"
cmd "webctl focus \"input[type=radio][value=small]\""

echo ""
echo "Verify: Radio button focused"
read -p "Press Enter to continue..."

heading "Focus select dropdown"
cmd "webctl focus \"select[name=delivery]\""

echo ""
echo "Verify: Select dropdown focused"
read -p "Press Enter to continue..."

# Focusable elements
title "Focusable Elements"

heading "Navigate to page with links"
cmd "webctl navigate https://example.com --wait"

echo ""
read -p "Press Enter when loaded..."

heading "Focus link element"
cmd "webctl focus \"a\""

echo ""
echo "Verify: Link focused (may show focus outline)"
read -p "Press Enter to continue..."

# Complex selectors
title "Complex Selectors"

heading "Navigate back to form"
cmd "webctl navigate http://httpbin.org/forms/post --wait"

echo ""
read -p "Press Enter when loaded..."

heading "Focus using attribute selector"
cmd "webctl focus \"input[type=text]\""

echo ""
echo "Verify: First text input focused"
read -p "Press Enter to continue..."

heading "Focus using :first-of-type pseudo-selector"
cmd "webctl focus \"input:first-of-type\""

echo ""
echo "Verify: First input element focused"
read -p "Press Enter to continue..."

heading "Focus using :not pseudo-selector"
cmd "webctl focus \"input:not([type=checkbox])\""

echo ""
echo "Verify: First non-checkbox input focused"
read -p "Press Enter to continue..."

# Focus followed by typing
title "Focus + Type Workflow"

heading "Focus customer name field"
cmd "webctl focus \"input[name=custname]\""

echo ""
read -p "Press Enter to continue..."

heading "Type into focused field (no selector)"
cmd "webctl type \"Test User\""

echo ""
echo "Verify: Text typed into focused field"
read -p "Press Enter to continue..."

heading "Focus email field"
cmd "webctl focus \"input[name=custemail]\""

echo ""
read -p "Press Enter to continue..."

heading "Type into focused email field"
cmd "webctl type \"test@example.com\""

echo ""
echo "Verify: Email typed into focused field"
read -p "Press Enter to continue..."

heading "Focus textarea"
cmd "webctl focus \"textarea[name=comments]\""

echo ""
read -p "Press Enter to continue..."

heading "Type comment"
cmd "webctl type \"This is a comment typed into focused textarea\""

echo ""
echo "Verify: Comment typed successfully"
read -p "Press Enter to continue..."

# Focus state verification
title "Focus State Verification"

heading "Focus a field"
cmd "webctl focus \"input[name=custname]\""

echo ""
read -p "Press Enter to continue..."

heading "Verify focused element with eval"
cmd "webctl eval \"document.activeElement.name\""

echo ""
echo "Verify: Output shows 'custname'"
read -p "Press Enter to continue..."

heading "Focus different field"
cmd "webctl focus \"input[name=custtel]\""

echo ""
read -p "Press Enter to continue..."

heading "Verify new focused element"
cmd "webctl eval \"document.activeElement.name\""

echo ""
echo "Verify: Output shows 'custtel'"
read -p "Press Enter to continue..."

heading "Verify focused element tag name"
cmd "webctl eval \"document.activeElement.tagName\""

echo ""
echo "Verify: Output shows 'INPUT'"
read -p "Press Enter to continue..."

# Focus events
title "Focus Events"

heading "Set up focus event listener with eval"
cmd "webctl eval \"window.focusEventFired = false; document.querySelector('input[name=custemail]').addEventListener('focus', () => { window.focusEventFired = true; });\""

echo ""
echo "Event listener set up"
read -p "Press Enter to continue..."

heading "Focus the email field"
cmd "webctl focus \"input[name=custemail]\""

echo ""
read -p "Press Enter to continue..."

heading "Check if focus event fired"
cmd "webctl eval \"window.focusEventFired\""

echo ""
echo "Verify: Output shows true (focus event fired)"
read -p "Press Enter to continue..."

# Error cases
title "Error Cases"

heading "Focus non-existent selector"
cmd "webctl focus \"#nonexistent-element\""

echo ""
echo "Verify: Error message 'element not found'"
read -p "Press Enter to continue..."

heading "Focus invalid CSS selector"
cmd "webctl focus \"[[invalid]]\""

echo ""
echo "Verify: Error message about invalid selector"
read -p "Press Enter to continue..."

heading "Focus empty selector"
cmd "webctl focus \"\""

echo ""
echo "Verify: Error message"
read -p "Press Enter to continue..."

heading "Focus non-focusable element (div)"
cmd "webctl focus \"div\""

echo ""
echo "Verify: Error message 'element is not focusable'"
echo "Note: May succeed if div has tabindex attribute"
read -p "Press Enter to continue..."

heading "Navigate to example.com"
cmd "webctl navigate https://example.com --wait"

echo ""
read -p "Press Enter when loaded..."

heading "Try to focus paragraph element"
cmd "webctl focus \"p\""

echo ""
echo "Verify: Error message 'element is not focusable'"
read -p "Press Enter to continue..."

heading "Try to focus h1 element"
cmd "webctl focus \"h1\""

echo ""
echo "Verify: Error message 'element is not focusable'"
read -p "Press Enter to continue..."

heading "Try to focus div element"
cmd "webctl focus \"div\""

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
cmd "webctl focus \"input[name=custname]\" --json"

echo ""
echo "Verify: JSON formatted output"
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl focus \"input[name=custtel]\" --no-color"

echo ""
echo "Verify: No ANSI color codes"
read -p "Press Enter to continue..."

heading "Debug output"
cmd "webctl focus \"input[name=custemail]\" --debug"

echo ""
echo "Verify: Debug logging information"
read -p "Press Enter to continue..."

# REPL mode tests
title "REPL Mode Tests"

heading "Test focus in REPL"
echo "Switch to daemon terminal and execute:"
cmd "focus \"input[name=custname]\""

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

heading "Test focus different element in REPL"
echo "In REPL, try:"
cmd "focus \"textarea[name=comments]\""

echo ""
echo "Should focus textarea"
read -p "Press Enter when tested in REPL..."

heading "Test focus + type workflow in REPL"
echo "In REPL, try focusing then typing:"
cmd "focus \"input[name=custtel]\""

echo ""
read -p "Press Enter when focused..."

echo "Then type:"
cmd "type \"555-7777\""

echo ""
echo "Should type into focused field"
read -p "Press Enter when tested in REPL..."

heading "Verify focused element in REPL"
echo "In REPL, check active element:"
cmd "eval \"document.activeElement.name\""

echo ""
echo "Should show name of focused element"
read -p "Press Enter when tested in REPL..."

# Accessibility testing
title "Accessibility Testing"

heading "Test focus order through form"
echo "Manual test: Use Tab key to move through form"
echo "Then use focus command to jump to specific fields"
cmd "webctl focus \"input[name=custname]\""

echo ""
read -p "Press Enter to continue..."

cmd "webctl key Tab"

echo ""
echo "Verify: Focus moved to next field in tab order"
read -p "Press Enter to continue..."

cmd "webctl key Tab"

echo ""
echo "Verify: Focus moved to next field again"
read -p "Press Enter to continue..."

heading "Jump focus to non-sequential element"
cmd "webctl focus \"button[type=submit]\""

echo ""
echo "Verify: Focus jumped to submit button (breaking tab order)"
read -p "Press Enter to continue..."

heading "Focus back to first field"
cmd "webctl focus \"input[name=custname]\""

echo ""
echo "Verify: Focus back at first field"
read -p "Press Enter to continue..."

# Test completed
title "Test Suite Complete"
echo "All focus command tests finished"
echo ""
echo "Review checklist in docs/projects/p-044-testing-focus.md"
echo "Document any issues discovered during testing"
