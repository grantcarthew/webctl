#!/bin/bash
# Title: webctl key command tests

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
title "webctl key Command Test Suite"
echo "Project: P-045"
echo "Tests sending keyboard keys with modifier support"
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

heading "Navigate to httpbin.org/forms/post (has form for keyboard testing)"
cmd "webctl navigate http://httpbin.org/forms/post --wait"

echo ""
echo "Wait for page to load"
read -p "Press Enter when page loaded..."

# Basic keys (no modifiers)
title "Basic Keys (No Modifiers)"

heading "Focus a text field first"
cmd "webctl focus \"input[name=custname]\""

echo ""
read -p "Press Enter to continue..."

heading "Send Enter key"
cmd "webctl key Enter"

echo ""
echo "Verify: Enter key sent (may trigger form submission if on submit button)"
echo "Note: Since we're on text input, it may just add newline or do nothing"
read -p "Press Enter to continue..."

heading "Type some text first"
cmd "webctl type \"Test User\" --clear"

echo ""
read -p "Press Enter to continue..."

heading "Send Tab key to move to next field"
cmd "webctl key Tab"

echo ""
echo "Verify: Focus moved to next field (telephone)"
read -p "Press Enter to continue..."

heading "Type in new field"
cmd "webctl type \"555-1234\""

echo ""
read -p "Press Enter to continue..."

heading "Send Tab again"
cmd "webctl key Tab"

echo ""
echo "Verify: Focus moved to email field"
read -p "Press Enter to continue..."

heading "Send Escape key"
cmd "webctl key Escape"

echo ""
echo "Verify: Escape key sent (may clear field or cancel action depending on handlers)"
read -p "Press Enter to continue..."

heading "Focus textarea"
cmd "webctl focus \"textarea[name=comments]\""

echo ""
read -p "Press Enter to continue..."

heading "Send Space key"
cmd "webctl key Space"

echo ""
echo "Verify: Space character added to textarea"
read -p "Press Enter to continue..."

heading "Type some text"
cmd "webctl type \"Hello World\""

echo ""
read -p "Press Enter to continue..."

heading "Send Backspace key"
cmd "webctl key Backspace"

echo ""
echo "Verify: Last character deleted ('d' removed from 'World')"
read -p "Press Enter to continue..."

heading "Send Delete key"
cmd "webctl key Delete"

echo ""
echo "Verify: Character after cursor deleted (if any)"
read -p "Press Enter to continue..."

# Arrow keys
title "Arrow Keys"

heading "Focus text field with some text"
cmd "webctl focus \"input[name=custname]\""

echo ""
read -p "Press Enter to continue..."

cmd "webctl type \"Arrow Test\" --clear"

echo ""
read -p "Press Enter to continue..."

heading "Send ArrowLeft to move cursor"
cmd "webctl key ArrowLeft"

echo ""
echo "Verify: Cursor moved left one character"
read -p "Press Enter to continue..."

heading "Send ArrowRight to move cursor back"
cmd "webctl key ArrowRight"

echo ""
echo "Verify: Cursor moved right one character"
read -p "Press Enter to continue..."

heading "Send Home to move to start"
cmd "webctl key Home"

echo ""
echo "Verify: Cursor moved to start of line"
read -p "Press Enter to continue..."

heading "Send End to move to end"
cmd "webctl key End"

echo ""
echo "Verify: Cursor moved to end of line"
read -p "Press Enter to continue..."

# Page navigation keys
title "Page Navigation Keys"

heading "Navigate to long page"
cmd "webctl navigate https://docs.github.com/en --wait"

echo ""
echo "Wait for page to load"
read -p "Press Enter when loaded..."

heading "Send PageDown key"
cmd "webctl key PageDown"

echo ""
echo "Verify: Page scrolled down one viewport height"
read -p "Press Enter to continue..."

heading "Send PageDown again"
cmd "webctl key PageDown"

echo ""
echo "Verify: Page scrolled down more"
read -p "Press Enter to continue..."

heading "Send PageUp key"
cmd "webctl key PageUp"

echo ""
echo "Verify: Page scrolled up one viewport height"
read -p "Press Enter to continue..."

heading "Send Home key (document start)"
cmd "webctl key Home"

echo ""
echo "Verify: Scrolled to top of document"
echo "Note: Behavior depends on focus - may just move cursor if in text field"
read -p "Press Enter to continue..."

heading "Send End key (document end)"
cmd "webctl key End"

echo ""
echo "Verify: Scrolled to bottom of document or moved to end of line"
read -p "Press Enter to continue..."

# Text editing shortcuts (Linux)
title "Text Editing Shortcuts (Linux/Ctrl)"

heading "Navigate back to form"
cmd "webctl navigate http://httpbin.org/forms/post --wait"

echo ""
read -p "Press Enter when loaded..."

heading "Focus and type some text"
cmd "webctl type \"input[name=custname]\" \"Select All Test\" --clear"

echo ""
read -p "Press Enter to continue..."

heading "Send Ctrl+A to select all"
cmd "webctl key a --ctrl"

echo ""
echo "Verify: All text selected (highlighted)"
echo "Note: On macOS, use --meta instead of --ctrl"
read -p "Press Enter to continue..."

heading "Type to replace selected text"
cmd "webctl type \"Replaced Text\""

echo ""
echo "Verify: Selected text replaced with 'Replaced Text'"
read -p "Press Enter to continue..."

heading "Send Ctrl+Z to undo"
cmd "webctl key z --ctrl"

echo ""
echo "Verify: Undo action performed (text reverted)"
echo "Note: May not work in all browsers or contexts"
read -p "Press Enter to continue..."

heading "Send Ctrl+Shift+Z to redo"
cmd "webctl key z --ctrl --shift"

echo ""
echo "Verify: Redo action performed"
echo "Note: May not work in all browsers or contexts"
read -p "Press Enter to continue..."

# Single character keys
title "Single Character Keys"

heading "Focus field and clear"
cmd "webctl focus \"input[name=custname]\""

echo ""
read -p "Press Enter to continue..."

cmd "webctl key a --ctrl"

echo ""
read -p "Press Enter to continue..."

heading "Send individual character keys"
cmd "webctl key a"

echo ""
echo "Verify: 'a' character typed"
read -p "Press Enter to continue..."

cmd "webctl key b"

echo ""
echo "Verify: 'b' character typed"
read -p "Press Enter to continue..."

cmd "webctl key 1"

echo ""
echo "Verify: '1' character typed"
read -p "Press Enter to continue..."

cmd "webctl key 5"

echo ""
echo "Verify: '5' character typed"
read -p "Press Enter to continue..."

# Shift modifier
title "Shift Modifier"

heading "Clear field"
cmd "webctl type \"input[name=custname]\" \"Selection Test\" --clear"

echo ""
read -p "Press Enter to continue..."

heading "Send Home to move to start"
cmd "webctl key Home"

echo ""
read -p "Press Enter to continue..."

heading "Send Shift+End to select to end"
cmd "webctl key End --shift"

echo ""
echo "Verify: Text from cursor to end selected"
read -p "Press Enter to continue..."

heading "Send Shift+Home to change selection"
cmd "webctl key Home --shift"

echo ""
echo "Verify: Selection changed"
read -p "Press Enter to continue..."

heading "Send Shift+ArrowRight to extend selection"
cmd "webctl key ArrowRight --shift"

echo ""
echo "Verify: Selection extended one character right"
read -p "Press Enter to continue..."

heading "Send Shift+ArrowLeft to reduce selection"
cmd "webctl key ArrowLeft --shift"

echo ""
echo "Verify: Selection reduced one character left"
read -p "Press Enter to continue..."

# Multiple modifiers
title "Multiple Modifiers"

heading "Clear and type new text"
cmd "webctl type \"input[name=custname]\" \"Multi-Modifier Test\" --clear"

echo ""
read -p "Press Enter to continue..."

heading "Send Ctrl+Shift+A (example multi-modifier)"
cmd "webctl key a --ctrl --shift"

echo ""
echo "Verify: Multi-modifier key combination sent"
echo "Note: Effect depends on browser/application shortcuts"
read -p "Press Enter to continue..."

# Form navigation with Tab
title "Form Navigation with Tab"

heading "Focus first field"
cmd "webctl focus \"input[name=custname]\""

echo ""
read -p "Press Enter to continue..."

heading "Tab through form fields"
cmd "webctl key Tab"

echo ""
echo "Verify: Moved to telephone field"
read -p "Press Enter to continue..."

cmd "webctl key Tab"

echo ""
echo "Verify: Moved to email field"
read -p "Press Enter to continue..."

cmd "webctl key Tab"

echo ""
echo "Verify: Moved to next field (size radio or other)"
read -p "Press Enter to continue..."

heading "Shift+Tab to go backwards"
cmd "webctl key Tab --shift"

echo ""
echo "Verify: Moved back to previous field"
read -p "Press Enter to continue..."

cmd "webctl key Tab --shift"

echo ""
echo "Verify: Moved back again"
read -p "Press Enter to continue..."

# Error cases
title "Error Cases"

heading "Send invalid key name"
cmd "webctl key InvalidKeyName"

echo ""
echo "Verify: Error message about invalid key"
read -p "Press Enter to continue..."

heading "Send empty key string"
cmd "webctl key \"\""

echo ""
echo "Verify: Error message"
read -p "Press Enter to continue..."

# Output formats
title "Output Format Tests"

heading "JSON output"
cmd "webctl key Tab --json"

echo ""
echo "Verify: JSON formatted output"
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl key Enter --no-color"

echo ""
echo "Verify: No ANSI color codes"
read -p "Press Enter to continue..."

heading "Debug output"
cmd "webctl key Escape --debug"

echo ""
echo "Verify: Debug logging information"
read -p "Press Enter to continue..."

# REPL mode tests
title "REPL Mode Tests"

heading "Test key in REPL"
echo "Switch to daemon terminal and execute:"
cmd "key Enter"

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

heading "Test key Tab in REPL"
echo "In REPL, try:"
cmd "key Tab"

echo ""
echo "Should move focus to next field"
read -p "Press Enter when tested in REPL..."

heading "Test key with modifier in REPL"
echo "In REPL, try:"
cmd "key a --ctrl"

echo ""
echo "Should select all (Ctrl+A on Linux)"
read -p "Press Enter when tested in REPL..."

heading "Test arrow keys in REPL"
echo "In REPL, try:"
cmd "key ArrowDown"

echo ""
echo "Should send arrow down key"
read -p "Press Enter when tested in REPL..."

# Form submission with Enter
title "Form Submission with Enter"

heading "Focus submit button"
cmd "webctl focus \"button[type=submit]\""

echo ""
read -p "Press Enter to continue..."

heading "Send Enter to submit form"
cmd "webctl key Enter"

echo ""
echo "Verify: Form submitted (Space also works on buttons)"
read -p "Press Enter to continue..."

heading "Navigate back to form"
cmd "webctl navigate http://httpbin.org/forms/post --wait"

echo ""
read -p "Press Enter when loaded..."

heading "Focus submit button again"
cmd "webctl focus \"button[type=submit]\""

echo ""
read -p "Press Enter to continue..."

heading "Send Space to click/submit"
cmd "webctl key Space"

echo ""
echo "Verify: Button clicked, form submitted"
read -p "Press Enter to continue..."

# Checkbox/Radio with Space
title "Checkbox/Radio with Space Key"

heading "Navigate back to form"
cmd "webctl navigate http://httpbin.org/forms/post --wait"

echo ""
read -p "Press Enter when loaded..."

heading "Focus checkbox"
cmd "webctl focus \"input[type=checkbox][value=bacon]\""

echo ""
read -p "Press Enter to continue..."

heading "Send Space to toggle checkbox"
cmd "webctl key Space"

echo ""
echo "Verify: Checkbox toggled (checked/unchecked)"
read -p "Press Enter to continue..."

heading "Send Space again to toggle back"
cmd "webctl key Space"

echo ""
echo "Verify: Checkbox toggled back"
read -p "Press Enter to continue..."

heading "Focus radio button"
cmd "webctl focus \"input[type=radio][value=medium]\""

echo ""
read -p "Press Enter to continue..."

heading "Send Space to select radio"
cmd "webctl key Space"

echo ""
echo "Verify: Radio button selected"
read -p "Press Enter to continue..."

# Test completed
title "Test Suite Complete"
echo "All key command tests finished"
echo ""
echo "Review checklist in docs/projects/p-045-testing-key.md"
echo "Document any issues discovered during testing"
echo ""
echo "Note: Some keyboard shortcuts (Ctrl+C, Ctrl+V, browser shortcuts)"
echo "may not work in headless mode or may require browser permissions"
