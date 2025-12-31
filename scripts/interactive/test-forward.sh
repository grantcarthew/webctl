#!/bin/bash
# Title: webctl forward command tests

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
title "webctl forward Command Test Suite"
echo "Project: P-033"
echo "Tests browser history forward navigation"
echo ""
echo "Prerequisites:"
echo "  - webctl must be built"
echo "  - xclip must be installed"
echo "  - Daemon running"
echo ""
read -p "Press Enter to begin tests..."

# Setup
title "Setup"
echo "Ensure daemon running"
cmd "webctl start"

read -p "Press Enter when daemon ready..."

# CLI Tests
title "Basic Forward Navigation"

heading "Navigate, back, then forward"
cmd "webctl navigate example.com"
cmd "webctl navigate wikipedia.org"
cmd "webctl back"
cmd "webctl status"

echo ""
echo "Should be at example.com"
read -p "Press Enter to continue..."

cmd "webctl forward"
cmd "webctl status"

echo ""
echo "Should be back at wikipedia.org"
read -p "Press Enter to continue..."

heading "Verify immediate return (no wait)"
cmd "webctl navigate google.com"
cmd "webctl navigate github.com"
cmd "webctl back"
cmd "webctl forward"

echo ""
echo "Should return immediately, navigation in background"
read -p "Press Enter to continue..."

title "Wait Functionality"

heading "Forward with --wait flag"
cmd "webctl navigate example.com"
cmd "webctl navigate wikipedia.org"
cmd "webctl back"
cmd "webctl forward --wait"

echo ""
echo "Should block until page loads"
read -p "Press Enter to continue..."

heading "Forward with custom timeout"
cmd "webctl forward --wait --timeout 60000"

echo ""
echo "Should wait up to 60 seconds"
read -p "Press Enter to continue..."

title "History Traversal"

heading "Navigate through multiple pages, back, then forward"
cmd "webctl navigate example.com"
cmd "webctl navigate wikipedia.org"
cmd "webctl navigate github.com"
cmd "webctl back"
cmd "webctl back"
cmd "webctl status"

echo ""
echo "Should be at example.com"
read -p "Press Enter to continue..."

cmd "webctl forward"
cmd "webctl status"

echo ""
echo "Should be at wikipedia.org"
read -p "Press Enter to continue..."

cmd "webctl forward"
cmd "webctl status"

echo ""
echo "Should be at github.com"
read -p "Press Enter to continue..."

heading "Combine forward with back"
cmd "webctl back"
cmd "webctl status"

echo ""
echo "Should be back at wikipedia.org"
read -p "Press Enter to continue..."

title "Forward History Cleared"

heading "Navigate from intermediate point clears forward history"
cmd "webctl navigate example.com"
cmd "webctl navigate wikipedia.org"
cmd "webctl navigate github.com"
cmd "webctl back"
cmd "webctl status"

echo ""
echo "Should be at wikipedia.org"
read -p "Press Enter to continue..."

cmd "webctl navigate google.com"
echo "Just navigated to new page from intermediate point"
cmd "webctl forward"

echo ""
echo "Forward should fail - history to github.com was cleared"
read -p "Press Enter to continue..."

title "Output Formats"

heading "Default text output"
cmd "webctl navigate example.com"
cmd "webctl navigate google.com"
cmd "webctl back"
cmd "webctl forward"

echo ""
echo "Should show simple OK"
read -p "Press Enter to continue..."

heading "JSON output"
cmd "webctl forward --json"

echo ""
echo "Should include URL and title in JSON"
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl forward --no-color"

echo ""
echo "Should have no ANSI colors"
read -p "Press Enter to continue..."

title "Various Scenarios"

heading "Forward from static to static page"
cmd "webctl navigate example.com"
cmd "webctl navigate wikipedia.org"
cmd "webctl back"
cmd "webctl forward"

echo ""
echo "Should navigate between static pages cleanly"
read -p "Press Enter to continue..."

heading "Forward with hash navigation"
cmd "webctl navigate https://developer.mozilla.org/en-US/"
cmd "webctl navigate https://developer.mozilla.org/en-US/#footer"
cmd "webctl back"
cmd "webctl forward"

echo ""
echo "Test hash navigation in history"
read -p "Press Enter to continue..."

title "Error Cases"

heading "Forward when no next page"
cmd "webctl navigate example.com"
cmd "webctl forward"

echo ""
echo "Should show error - no next page in history"
read -p "Press Enter to continue..."

heading "Forward at end of history"
cmd "webctl navigate example.com"
cmd "webctl navigate google.com"
cmd "webctl back"
cmd "webctl forward"
cmd "webctl forward"

echo ""
echo "Should show error - already at end"
read -p "Press Enter to continue..."

title "Common Patterns"

heading "back && forward"
cmd "webctl navigate example.com"
cmd "webctl navigate wikipedia.org"
cmd "webctl back"
cmd "webctl forward"

echo ""
echo "Common pattern for testing history"
read -p "Press Enter to continue..."

heading "forward && ready"
cmd "webctl navigate google.com"
cmd "webctl navigate github.com"
cmd "webctl back"
cmd "webctl forward && webctl ready"

echo ""
echo "Forward and wait for ready state"
read -p "Press Enter to continue..."

heading "forward --wait && html"
cmd "webctl forward --wait && webctl html show"

echo ""
echo "Forward, wait, then show HTML"
read -p "Press Enter to continue..."

title "REPL Mode Tests"

heading "Forward from REPL"
echo "Switch to daemon terminal and execute:"
cmd "navigate example.com"
cmd "navigate github.com"
cmd "back"
cmd "forward"

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

title "Test Suite Complete"
echo "All forward command tests finished"
echo ""
echo "Review checklist in docs/projects/p-033-testing-forward.md"
echo "Document any issues discovered during testing"
