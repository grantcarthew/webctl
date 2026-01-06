#!/bin/bash
# Title: webctl navigate command tests

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
title "webctl navigate Command Test Suite"
echo "Project: P-030"
echo "Tests URL navigation with protocol auto-detection"
echo ""
echo "Prerequisites:"
echo "  - webctl must be built"
echo "  - Clipboard tool (pbcopy on macOS, xclip on Linux)"
echo "  - Daemon running (or start one)"
echo ""
read -p "Press Enter to begin tests..."

# Setup
title "Setup"
echo "Ensure daemon is running"
cmd "webctl start"

echo ""
echo "Start daemon if not running, then continue"
read -p "Press Enter when daemon ready..."

# CLI Tests
title "Basic Navigation (Fast Return)"

heading "Navigate to example.com"
cmd "webctl navigate example.com"

echo ""
echo "Should return immediately, page loads in background"
read -p "Press Enter to continue..."

heading "Navigate to google.com"
cmd "webctl navigate google.com"

echo ""
echo "Verify quick return and background loading"
read -p "Press Enter to continue..."

title "URL Protocol Auto-Detection"

heading "URL without protocol (gets https://)"
cmd "webctl navigate example.com"
cmd "webctl status"

echo ""
echo "Status should show https://example.com"
read -p "Press Enter to continue..."

heading "localhost gets http://"
cmd "webctl navigate localhost:3000"
cmd "webctl status"

echo ""
echo "Status should show http://localhost:3000"
read -p "Press Enter to continue..."

heading "127.0.0.1 gets http://"
cmd "webctl navigate 127.0.0.1:8080"
cmd "webctl status"

echo ""
echo "Status should show http://127.0.0.1:8080"
read -p "Press Enter to continue..."

heading "Explicit https:// preserved"
cmd "webctl navigate https://example.com"
cmd "webctl status"

echo ""
echo "Verify https:// used"
read -p "Press Enter to continue..."

heading "Explicit http:// preserved"
cmd "webctl navigate http://example.com"
cmd "webctl status"

echo ""
echo "Verify http:// used"
read -p "Press Enter to continue..."

title "Wait Functionality"

heading "Navigate with --wait flag"
cmd "webctl navigate example.com --wait"

echo ""
echo "Should block until page fully loaded"
read -p "Press Enter to continue..."

heading "Navigate with custom timeout"
cmd "webctl navigate example.com --wait --timeout 60000"

echo ""
echo "Should wait up to 60 seconds for load"
read -p "Press Enter to continue..."

title "Output Formats"

heading "Default text output"
cmd "webctl navigate example.com"

echo ""
echo "Should show simple OK"
read -p "Press Enter to continue..."

heading "JSON output"
cmd "webctl navigate example.com --json"

echo ""
echo "Should include URL and title in JSON"
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl navigate example.com --no-color"

echo ""
echo "Should have no ANSI colors"
read -p "Press Enter to continue..."

title "Various Page Types"

heading "Navigate to static HTML page"
cmd "webctl navigate https://example.com"
read -p "Press Enter to continue..."

heading "Navigate to JavaScript SPA"
cmd "webctl navigate https://react.dev"
read -p "Press Enter to continue..."

heading "Navigate to page with redirects"
cmd "webctl navigate http://github.com"

echo ""
echo "Should follow redirect to https://github.com"
read -p "Press Enter to continue..."

title "Error Cases"

heading "Invalid domain"
cmd "webctl navigate nonexistent-domain-xyz.com"

echo ""
echo "Should show net::ERR_NAME_NOT_RESOLVED or similar"
read -p "Press Enter to continue..."

heading "Connection refused"
cmd "webctl navigate localhost:9999"

echo ""
echo "Should show net::ERR_CONNECTION_REFUSED"
read -p "Press Enter to continue..."

title "Common Workflow Patterns"

heading "navigate && ready"
cmd "webctl navigate example.com && webctl ready"

echo ""
echo "Should navigate and wait for ready state"
read -p "Press Enter to continue..."

heading "navigate && screenshot"
cmd "webctl navigate example.com && webctl screenshot"

echo ""
echo "Navigate and capture screenshot"
read -p "Press Enter to continue..."

heading "navigate --wait && html"
cmd "webctl navigate example.com --wait && webctl html show"

echo ""
echo "Navigate, wait, then show HTML"
read -p "Press Enter to continue..."

title "REPL Mode Tests"

heading "Navigate from REPL"
echo "Switch to daemon terminal and execute:"
cmd "navigate https://example.com"

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

title "Cleanup"
echo "Tests complete - daemon can remain running or be stopped"

title "Test Suite Complete"
echo "All navigate command tests finished"
echo ""
echo "Review checklist in docs/projects/p-030-testing-navigate.md"
echo "Document any issues discovered during testing"
