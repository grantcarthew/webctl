#!/bin/bash
# Title: webctl status command tests

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
title "webctl status Command Test Suite"
echo "Project: P-028"
echo "Tests daemon status reporting"
echo ""
echo "Prerequisites:"
echo "  - webctl must be built"
echo "  - xclip must be installed"
echo ""
read -p "Press Enter to begin tests..."

# Test when not running
title "Status When Daemon Not Running"

heading "Check status with no daemon"
cmd "webctl status"

echo ""
echo "Should show: running: false"
read -p "Press Enter to continue..."

# Setup
title "Setup"
echo "Starting daemon for remaining tests..."
cmd "webctl start"

echo ""
echo "Start daemon in another terminal, then continue"
read -p "Press Enter when daemon is running..."

# CLI Tests
title "CLI Mode Tests - When Running"

heading "Basic status check"
cmd "webctl status"

echo ""
echo "Should show:"
echo "  - running: true"
echo "  - Current URL"
echo "  - Page title (if available)"
read -p "Press Enter to continue..."

heading "Status with JSON output"
cmd "webctl status --json"

echo ""
echo "Should show JSON formatted status with all fields"
read -p "Press Enter to continue..."

heading "Status with no color"
cmd "webctl status --no-color"

echo ""
echo "Should show status without ANSI colors"
read -p "Press Enter to continue..."

title "Status During Various States"

heading "Status immediately after start"
echo "Already tested above - daemon just started"
echo "URL might be about:blank or default page"
read -p "Press Enter to continue..."

heading "Status after navigation"
echo "In daemon terminal or another terminal, navigate to a page:"
cmd "webctl navigate https://example.com"

echo ""
echo "Wait for page to load, then check status:"
cmd "webctl status"

echo ""
echo "Should show example.com URL and 'Example Domain' title"
read -p "Press Enter to continue..."

heading "Status with error page (404)"
echo "Navigate to non-existent page:"
cmd "webctl navigate https://example.com/nonexistent"

echo ""
echo "Then check status:"
cmd "webctl status"

echo ""
echo "Check how status reports error page"
read -p "Press Enter to continue..."

title "REPL Mode Tests"

heading "Status from REPL"
echo "Switch to daemon terminal and execute:"
cmd "status"

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

title "Debug Output Test"

heading "Status with debug flag"
cmd "webctl status --debug"

echo ""
echo "Should show additional debug information"
read -p "Press Enter to continue..."

title "Cleanup"
echo "Stopping daemon..."
cmd "webctl stop"

read -p "Press Enter when daemon stopped..."

title "Test Suite Complete"
echo "All status command tests finished"
echo ""
echo "Review checklist in docs/projects/p-028-testing-status.md"
echo "Document any issues discovered during testing"
