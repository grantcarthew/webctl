#!/bin/bash
# Title: webctl serve command tests

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
title "webctl serve Command Test Suite"
echo "Project: P-029"
echo "Tests development server with static and proxy modes"
echo ""
echo "Prerequisites:"
echo "  - webctl must be built"
echo "  - Clipboard tool (pbcopy on macOS, xclip on Linux)"
echo "  - testdata/index.html (comprehensive test page included)"
echo "  - Backend server for proxy tests (optional)"
echo ""
read -p "Press Enter to begin tests..."

# CLI Tests
title "Static Mode - Basic Functionality"

heading "Serve current directory (default)"
echo "Uses testdata directory with comprehensive test page"
cmd "cd \$(git rev-parse --show-toplevel)/testdata && webctl serve"

echo ""
echo "Watch for:"
echo "  1. Daemon auto-starts if not running"
echo "  2. Browser opens and navigates to served URL"
echo "  3. Test page displays"
echo "  4. Server runs in foreground"
echo "Stop with Ctrl+C"
read -p "Press Enter after testing..."

heading "Serve specified directory"
cmd "webctl serve \$(git rev-parse --show-toplevel)/testdata"

echo ""
echo "Verify same behavior as default mode"
read -p "Press Enter after testing..."

title "File Watching and Hot Reload"

heading "Setup for hot reload test"
echo "Start serve, then modify HTML file in another terminal"
cmd "webctl serve \$(git rev-parse --show-toplevel)/testdata"

echo ""
echo "While running:"
echo "  1. Modify testdata/index.html (e.g., change the title)"
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
cmd "webctl serve \$(git rev-parse --show-toplevel)/testdata --port 3000"

echo ""
echo "Verify server runs on port 3000"
read -p "Press Enter after testing..."

heading "Network binding"
cmd "webctl serve \$(git rev-parse --show-toplevel)/testdata --host 0.0.0.0"

echo ""
echo "Verify accessible from network (check local IP)"
read -p "Press Enter after testing..."

title "Auto-Start Behavior"

heading "Serve when daemon not running"
echo "First, ensure no daemon running"
cmd "webctl stop"
cmd "webctl serve \$(git rev-parse --show-toplevel)/testdata"

echo ""
echo "Verify daemon auto-starts with browser"
read -p "Press Enter after testing..."

title "Integration with webctl Commands"

heading "Use console while serving"
echo "Start serve, then use console command from another terminal"
cmd "webctl serve \$(git rev-parse --show-toplevel)/testdata"

echo ""
echo "In another terminal, test:"
cmd "webctl console show"
echo ""
echo "TIP: Click the console test buttons on the test page!"

echo ""
echo "Verify console command works while serving"
read -p "Press Enter after testing..."

title "Output Formats"

heading "JSON output"
cmd "webctl serve \$(git rev-parse --show-toplevel)/testdata --json"

echo ""
echo "Check for JSON formatted startup message"
read -p "Press Enter after testing..."

heading "Debug output"
cmd "webctl serve \$(git rev-parse --show-toplevel)/testdata --debug"

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
echo "No cleanup needed - testdata is part of the repository"
echo "If you modified testdata/index.html during testing, you may want to:"
cmd "git checkout testdata/index.html"

title "Test Suite Complete"
echo "All serve command tests finished"
echo ""
echo "Review checklist in docs/projects/p-029-testing-serve.md"
echo "Document any issues discovered during testing"
