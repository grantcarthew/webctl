#!/bin/bash
# Title: webctl serve command tests

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
title "webctl serve Command Test Suite"
echo "Project: P-029"
echo "Tests development server with static and proxy modes"
echo ""
echo "Prerequisites:"
echo "  - webctl must be built"
echo "  - xclip must be installed"
echo "  - Test directory with HTML files"
echo "  - Backend server for proxy tests (optional)"
echo ""
read -p "Press Enter to begin tests..."

# CLI Tests
title "Static Mode - Basic Functionality"

heading "Serve current directory (default)"
echo "Creates a test file first"
cmd "mkdir -p /tmp/webctl-serve-test && echo '<h1>Test Page</h1>' > /tmp/webctl-serve-test/index.html"
cmd "cd /tmp/webctl-serve-test && webctl serve"

echo ""
echo "Watch for:"
echo "  1. Daemon auto-starts if not running"
echo "  2. Browser opens and navigates to served URL"
echo "  3. Test page displays"
echo "  4. Server runs in foreground"
echo "Stop with Ctrl+C"
read -p "Press Enter after testing..."

heading "Serve specified directory"
cmd "webctl serve /tmp/webctl-serve-test"

echo ""
echo "Verify same behavior as default mode"
read -p "Press Enter after testing..."

title "File Watching and Hot Reload"

heading "Setup for hot reload test"
echo "Start serve, then modify HTML file in another terminal"
cmd "webctl serve /tmp/webctl-serve-test"

echo ""
echo "While running:"
echo "  1. Modify /tmp/webctl-serve-test/index.html"
echo "  2. Watch browser auto-reload"
echo "  3. Verify changes appear"
echo "Stop with Ctrl+C"
read -p "Press Enter after testing..."

title "Proxy Mode"

heading "Proxy to localhost backend"
echo "NOTE: Requires backend server running on localhost:3000"
echo "Skip if no backend available"
cmd "webctl serve --proxy localhost:3000"

echo ""
echo "Test only if backend available"
read -p "Press Enter to continue..."

heading "Proxy with full URL"
cmd "webctl serve --proxy http://localhost:8080"

echo ""
echo "Test only if backend available"
read -p "Press Enter to continue..."

title "Port and Host Options"

heading "Custom port"
cmd "webctl serve /tmp/webctl-serve-test --port 3000"

echo ""
echo "Verify server runs on port 3000"
read -p "Press Enter after testing..."

heading "Network binding"
cmd "webctl serve /tmp/webctl-serve-test --host 0.0.0.0"

echo ""
echo "Verify accessible from network (check local IP)"
read -p "Press Enter after testing..."

title "Auto-Start Behavior"

heading "Serve when daemon not running"
echo "First, ensure no daemon running"
cmd "webctl stop"
cmd "webctl serve /tmp/webctl-serve-test"

echo ""
echo "Verify daemon auto-starts with browser"
read -p "Press Enter after testing..."

title "Integration with webctl Commands"

heading "Use console while serving"
echo "Start serve, then use console command from another terminal"
cmd "webctl serve /tmp/webctl-serve-test"

echo ""
echo "In another terminal, test:"
cmd "webctl console show"

echo ""
echo "Verify console command works while serving"
read -p "Press Enter after testing..."

title "Output Formats"

heading "JSON output"
cmd "webctl serve /tmp/webctl-serve-test --json"

echo ""
echo "Check for JSON formatted startup message"
read -p "Press Enter after testing..."

heading "Debug output"
cmd "webctl serve /tmp/webctl-serve-test --debug"

echo ""
echo "Watch for verbose debug logs"
read -p "Press Enter after testing..."

title "Error Cases"

heading "Serve non-existent directory"
cmd "webctl serve /nonexistent/directory"

echo ""
echo "Should show appropriate error"
read -p "Press Enter to continue..."

heading "Serve with port in use"
echo "Start serve on port 3000, then try again"
echo "Test manually if desired"
read -p "Press Enter to continue..."

title "Cleanup"
echo "Removing test directory..."
cmd "rm -rf /tmp/webctl-serve-test"

title "Test Suite Complete"
echo "All serve command tests finished"
echo ""
echo "Review checklist in docs/projects/p-029-testing-serve.md"
echo "Document any issues discovered during testing"
