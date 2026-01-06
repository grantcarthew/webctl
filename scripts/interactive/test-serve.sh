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

heading "Start test backend server"
echo "Starting test backend in background on port 3000..."
echo "In a separate terminal, run:"
cmd "cd \$(git rev-parse --show-toplevel)/testdata && ./start-backend.sh"

echo ""
echo "The backend provides endpoints:"
echo "  - GET  /                - Backend HTML page"
echo "  - GET  /api/hello       - Hello message (JSON)"
echo "  - GET  /api/users       - User list (JSON)"
echo "  - GET  /status/200      - 200 OK response"
echo "  - GET  /status/404      - 404 Not Found"
echo ""
echo "Start the backend, then continue"
read -p "Press Enter when backend is running..."

heading "Verify backend is accessible"
echo "Testing backend endpoints to ensure it's running correctly..."
cmd "curl -s http://localhost:3000/api/hello | head -1"

echo ""
echo "Should show: {\"message\":\"Hello from test backend!\"}"
read -p "Press Enter after verifying backend..."

heading "Test backend endpoints directly"
echo "Test a few endpoints to verify backend works before proxying:"
cmd "curl -s http://localhost:3000/api/users | head -5"
cmd "curl -s http://localhost:3000/status/404"

echo ""
echo "Verify responses are correct"
read -p "Press Enter to continue to proxy tests..."

heading "Proxy to localhost backend (shorthand)"
cmd "webctl serve --proxy localhost:3000"

echo ""
echo "The proxy server is now running on an auto-detected port."
echo "Note: Backend is on port 3000, proxy is on a different port (shown in output above)"
echo ""
echo "Test these in the browser using the PROXY port (not 3000):"
echo "  1. Visit http://localhost:PROXY_PORT/ - See backend HTML page (pink gradient)"
echo "  2. Visit http://localhost:PROXY_PORT/api/hello - See JSON response"
echo "  3. Visit http://localhost:PROXY_PORT/api/users - See user list"
echo ""
echo "In another terminal, you can also test with curl (replace PROXY_PORT):"
cmd "curl -s http://localhost:PROXY_PORT/api/hello"

echo ""
echo "Verify: Backend responses show through proxy"
read -p "Press Enter after testing (stop with Ctrl+C)..."

heading "Proxy with full URL"
cmd "webctl serve --proxy http://localhost:3000"

echo ""
echo "Verify: Same behavior as shorthand localhost:3000"
read -p "Press Enter after testing (stop with Ctrl+C)..."

heading "Stop backend server"
echo "Switch to backend terminal and press Ctrl+C to stop it"
read -p "Press Enter when backend stopped..."

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
