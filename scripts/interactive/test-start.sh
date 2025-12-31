#!/bin/bash
# Title: webctl start command tests

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
title "webctl start Command Test Suite"
echo "Project: P-026"
echo "Tests daemon startup, browser launch, and REPL mode"
echo ""
echo "Prerequisites:"
echo "  - webctl must be built (./webctl or go build)"
echo "  - xclip must be installed"
echo "  - No daemon currently running"
echo ""
read -p "Press Enter to begin tests..."

# CLI Tests
title "CLI Mode Tests - Basic Functionality"

heading "Start daemon with default settings"
echo "This will start the daemon with headed browser and REPL"
echo "Watch for: browser window opens, REPL prompt appears"
cmd "webctl start"

echo ""
echo "After starting daemon:"
echo "  1. Verify browser window opened"
echo "  2. Verify REPL prompt shows in terminal"
echo "  3. Try a simple REPL command: status"
echo "  4. Stop daemon with Ctrl+C or 'stop' command"
read -p "Press Enter when daemon is stopped..."

heading "Start daemon in headless mode"
echo "Browser should not appear but REPL should work"
cmd "webctl start --headless"

echo ""
echo "After starting headless daemon:"
echo "  1. Verify NO browser window appeared"
echo "  2. Verify REPL prompt works"
echo "  3. Try REPL command: status"
echo "  4. Stop daemon"
read -p "Press Enter when daemon is stopped..."

heading "Start daemon on custom port"
echo "Browser should launch on port 9223"
cmd "webctl start --port 9223"

echo ""
echo "After starting on custom port:"
echo "  1. Verify REPL works"
echo "  2. Check browser launched correctly"
echo "  3. Stop daemon"
read -p "Press Enter when daemon is stopped..."

title "Global Flags Tests"

heading "Start with debug output"
echo "Should show verbose logging"
cmd "webctl start --debug"

echo ""
echo "Watch for detailed debug logs during startup"
echo "Stop daemon when reviewed"
read -p "Press Enter when daemon is stopped..."

heading "Start with JSON output"
echo "Startup message should be JSON formatted"
cmd "webctl start --json"

echo ""
echo "Check for JSON output on startup"
echo "Stop daemon when reviewed"
read -p "Press Enter when daemon is stopped..."

heading "Start with no color output"
echo "Output should have no ANSI color codes"
cmd "webctl start --no-color"

echo ""
echo "Verify no colored output"
echo "Stop daemon when reviewed"
read -p "Press Enter when daemon is stopped..."

title "Error Cases"

heading "Attempt to start when already running"
echo "First, start a daemon normally"
read -p "Start daemon in another terminal, then press Enter..."
cmd "webctl start"

echo ""
echo "Should show error: 'daemon is already running'"
read -p "Stop the running daemon, then press Enter..."

heading "Start with port already in use"
echo "This requires manually blocking port 9222"
echo "Skip if unable to test port conflicts"
read -p "Press Enter to skip or test manually..."

title "Test Suite Complete"
echo "All start command tests finished"
echo ""
echo "Review checklist in docs/projects/p-026-testing-start.md"
echo "Document any issues discovered during testing"
