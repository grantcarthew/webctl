#!/bin/bash
# Title: webctl clear command tests

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
title "webctl clear Command Test Suite"
echo "Project: P-048"
echo "Tests clearing event buffers (console and network)"
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

# Generate console and network events
heading "Navigate to page that generates events"
cmd "webctl navigate https://example.com --wait"

echo ""
read -p "Press Enter when page loaded..."

heading "Generate console logs"
cmd "webctl eval \"console.log('Test message 1'); console.warn('Warning 1'); console.error('Error 1')\""

echo ""
echo "Verify: Console messages logged"
read -p "Press Enter to continue..."

heading "Verify console events exist"
cmd "webctl console show"

echo ""
echo "Verify: Shows console messages"
read -p "Press Enter to continue..."

heading "Verify network events exist"
cmd "webctl network show"

echo ""
echo "Verify: Shows network requests"
read -p "Press Enter to continue..."

# Clear all buffers
title "Clear All Buffers (No Argument)"

heading "Clear both console and network"
cmd "webctl clear"

echo ""
echo "Verify: Success message or OK"
read -p "Press Enter to continue..."

heading "Verify console buffer cleared"
cmd "webctl console show"

echo ""
echo "Verify: No console messages (empty)"
read -p "Press Enter to continue..."

heading "Verify network buffer cleared"
cmd "webctl network show"

echo ""
echo "Verify: No network requests (empty)"
read -p "Press Enter to continue..."

# Clear console buffer only
title "Clear Console Buffer Only"

heading "Generate new console and network events"
cmd "webctl eval \"console.log('Test message 2')\""
cmd "webctl navigate https://example.com --wait"

echo ""
read -p "Press Enter when ready..."

heading "Verify both buffers have events"
cmd "webctl console show"
cmd "webctl network show"

echo ""
echo "Verify: Both show events"
read -p "Press Enter to continue..."

heading "Clear console buffer only"
cmd "webctl clear console"

echo ""
echo "Verify: Success message"
read -p "Press Enter to continue..."

heading "Verify console buffer cleared"
cmd "webctl console show"

echo ""
echo "Verify: Console empty"
read -p "Press Enter to continue..."

heading "Verify network buffer NOT cleared"
cmd "webctl network show"

echo ""
echo "Verify: Network requests still present"
read -p "Press Enter to continue..."

# Clear network buffer only
title "Clear Network Buffer Only"

heading "Generate new console logs"
cmd "webctl eval \"console.log('Test message 3')\""

echo ""
read -p "Press Enter to continue..."

heading "Verify console has events"
cmd "webctl console show"

echo ""
echo "Verify: Shows console messages"
read -p "Press Enter to continue..."

heading "Clear network buffer only"
cmd "webctl clear network"

echo ""
echo "Verify: Success message"
read -p "Press Enter to continue..."

heading "Verify network buffer cleared"
cmd "webctl network show"

echo ""
echo "Verify: Network empty"
read -p "Press Enter to continue..."

heading "Verify console buffer NOT cleared"
cmd "webctl console show"

echo ""
echo "Verify: Console messages still present"
read -p "Press Enter to continue..."

# Workflow scenarios
title "Workflow Scenarios"

heading "Test scenario 1: Generate events"
cmd "webctl navigate https://github.com --wait"
cmd "webctl eval \"console.log('Scenario 1 log')\""

echo ""
read -p "Press Enter to continue..."

heading "Observe events"
cmd "webctl console show"
cmd "webctl network show"

echo ""
echo "Verify: Events from scenario 1 present"
read -p "Press Enter to continue..."

heading "Clear all buffers"
cmd "webctl clear"

echo ""
read -p "Press Enter to continue..."

heading "Test scenario 2: Generate different events"
cmd "webctl navigate https://example.com --wait"
cmd "webctl eval \"console.log('Scenario 2 log')\""

echo ""
read -p "Press Enter to continue..."

heading "Verify only scenario 2 events present"
cmd "webctl console show"
cmd "webctl network show"

echo ""
echo "Verify: Only new events from scenario 2"
read -p "Press Enter to continue..."

# Multiple clears
title "Multiple Clears in Sequence"

heading "Clear console"
cmd "webctl clear console"

echo ""
read -p "Press Enter to continue..."

heading "Clear network"
cmd "webctl clear network"

echo ""
read -p "Press Enter to continue..."

heading "Clear console again (idempotent)"
cmd "webctl clear console"

echo ""
echo "Verify: Still succeeds (no error)"
read -p "Press Enter to continue..."

heading "Clear all"
cmd "webctl clear"

echo ""
echo "Verify: Still succeeds"
read -p "Press Enter to continue..."

# Empty buffer clears
title "Clear When Buffers Already Empty"

heading "Clear console when already empty"
cmd "webctl clear console"

echo ""
echo "Verify: Succeeds (idempotent)"
read -p "Press Enter to continue..."

heading "Clear network when already empty"
cmd "webctl clear network"

echo ""
echo "Verify: Succeeds (idempotent)"
read -p "Press Enter to continue..."

heading "Clear all when already empty"
cmd "webctl clear"

echo ""
echo "Verify: Succeeds (idempotent)"
read -p "Press Enter to continue..."

# Error cases
title "Error Cases"

heading "Invalid target (wrong name)"
cmd "webctl clear invalid-target"

echo ""
echo "Verify: Error - invalid target, must be 'console' or 'network'"
read -p "Press Enter to continue..."

heading "Case mismatch - Console"
cmd "webctl clear Console"

echo ""
echo "Verify: Error - case sensitive, must be lowercase"
read -p "Press Enter to continue..."

heading "Case mismatch - NETWORK"
cmd "webctl clear NETWORK"

echo ""
echo "Verify: Error - case sensitive, must be lowercase"
read -p "Press Enter to continue..."

# Output formats
title "Output Format Tests"

heading "Generate events for output tests"
cmd "webctl eval \"console.log('Output test')\""

echo ""
read -p "Press Enter to continue..."

heading "Default text output"
cmd "webctl clear console"

echo ""
echo "Verify: Shows OK or success"
read -p "Press Enter to continue..."

heading "JSON output - clear all"
cmd "webctl eval \"console.log('JSON test')\""
cmd "webctl clear --json"

echo ""
echo "Verify: JSON with message 'all buffers cleared'"
read -p "Press Enter to continue..."

heading "JSON output - clear console"
cmd "webctl eval \"console.log('JSON test 2')\""
cmd "webctl clear console --json"

echo ""
echo "Verify: JSON with message 'console buffer cleared'"
read -p "Press Enter to continue..."

heading "Generate network event"
cmd "webctl navigate https://example.com --wait"

echo ""
read -p "Press Enter to continue..."

heading "JSON output - clear network"
cmd "webctl clear network --json"

echo ""
echo "Verify: JSON with message 'network buffer cleared'"
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl eval \"console.log('No color test')\""
cmd "webctl clear console --no-color"

echo ""
echo "Verify: No ANSI color codes"
read -p "Press Enter to continue..."

heading "Debug output"
cmd "webctl eval \"console.log('Debug test')\""
cmd "webctl clear console --debug"

echo ""
echo "Verify: Debug information shown"
read -p "Press Enter to continue..."

# Independence verification
title "Verify Buffer Independence"

heading "Generate console events only"
cmd "webctl eval \"console.log('Console only')\""

echo ""
read -p "Press Enter to continue..."

heading "Clear network (should not affect console)"
cmd "webctl clear network"

echo ""
read -p "Press Enter to continue..."

heading "Verify console events still present"
cmd "webctl console show"

echo ""
echo "Verify: Console events still there"
read -p "Press Enter to continue..."

heading "Generate network events"
cmd "webctl navigate https://example.com --wait"

echo ""
read -p "Press Enter to continue..."

heading "Clear console (should not affect network)"
cmd "webctl clear console"

echo ""
read -p "Press Enter to continue..."

heading "Verify network events still present"
cmd "webctl network show"

echo ""
echo "Verify: Network events still there"
read -p "Press Enter to continue..."

# REPL mode tests
title "REPL Mode Tests"

heading "Clear in REPL (all buffers)"
cmd "webctl eval \"console.log('REPL test')\""

echo ""
echo "Switch to daemon terminal and execute:"
cmd "clear"

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

heading "Clear console in REPL"
cmd "webctl eval \"console.log('REPL test 2')\""

echo ""
echo "In REPL, try:"
cmd "clear console"

echo ""
read -p "Press Enter when tested in REPL..."

heading "Clear network in REPL"
cmd "webctl navigate https://example.com --wait"

echo ""
echo "In REPL, try:"
cmd "clear network"

echo ""
read -p "Press Enter when tested in REPL..."

# Test completed
title "Test Suite Complete"
echo "All clear command tests finished"
echo ""
echo "Review checklist in docs/projects/p-048-testing-clear.md"
echo "Document any issues discovered during testing"
