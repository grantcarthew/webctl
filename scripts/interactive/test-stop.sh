#!/bin/bash
# Title: webctl stop command tests

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
title "webctl stop Command Test Suite"
echo "Project: P-027"
echo "Tests daemon shutdown functionality"
echo ""
echo "Prerequisites:"
echo "  - webctl must be built"
echo "  - xclip must be installed"
echo ""
read -p "Press Enter to begin tests..."

# Setup
title "Setup"
echo "Starting daemon for stop tests..."
echo "Run this command in another terminal:"
cmd "webctl start"

echo ""
echo "Verify daemon is running before continuing"
read -p "Press Enter when daemon is running..."

# CLI Tests
title "CLI Mode Tests - Basic Functionality"

heading "Stop daemon from separate terminal"
echo "This should cleanly shut down daemon and browser"
cmd "webctl stop"

echo ""
echo "After executing stop:"
echo "  1. Verify browser closed"
echo "  2. Verify daemon terminal exited cleanly"
echo "  3. Check for clean shutdown message"
read -p "Press Enter to continue..."

title "Output Format Tests"

heading "Setup for output tests"
echo "Start daemon again for next tests"
cmd "webctl start"
read -p "Press Enter when daemon is running..."

heading "Stop with JSON output"
cmd "webctl stop --json"

echo ""
echo "Check for JSON formatted response"
read -p "Press Enter to continue..."

heading "Setup for no-color test"
cmd "webctl start"
read -p "Press Enter when daemon is running..."

heading "Stop with no color output"
cmd "webctl stop --no-color"

echo ""
echo "Verify no ANSI color codes in output"
read -p "Press Enter to continue..."

title "REPL Mode Tests"

heading "Setup REPL test"
echo "Start daemon and switch to its terminal for REPL test"
cmd "webctl start"
read -p "Press Enter when ready to test REPL stop..."

heading "Stop from REPL"
echo "In the daemon terminal REPL, execute:"
cmd "stop"

echo ""
echo "Verify daemon stopped from within REPL"
read -p "Press Enter to continue..."

title "Error Cases"

heading "Attempt stop when daemon not running"
echo "Daemon should not be running now"
cmd "webctl stop"

echo ""
echo "Should show error: daemon not running or connection refused"
read -p "Press Enter to continue..."

title "Cleanup Verification"

heading "Start and stop daemon, then verify cleanup"
echo "Start daemon..."
cmd "webctl start"
read -p "Press Enter when daemon started..."

cmd "webctl stop"
echo ""
echo "After stop, verify:"
echo "  1. Check no browser processes: ps aux | grep chrome"
echo "  2. Check daemon status"
cmd "webctl status"

echo ""
echo "Status should show 'running: false'"
read -p "Press Enter to continue..."

title "Test Suite Complete"
echo "All stop command tests finished"
echo ""
echo "Review checklist in docs/projects/p-027-testing-stop.md"
echo "Document any issues discovered during testing"
