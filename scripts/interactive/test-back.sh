#!/bin/bash
# Title: webctl back command tests

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
title "webctl back Command Test Suite"
echo "Project: P-032"
echo "Tests browser history back navigation"
echo ""
echo "Prerequisites:"
echo "  - webctl must be built"
echo "  - Clipboard tool (pbcopy on macOS, xclip on Linux)"
echo "  - Daemon running"
echo ""
read -p "Press Enter to begin tests..."

# Setup
title "Setup"
echo "Ensure daemon running"
cmd "webctl start"

read -p "Press Enter when daemon ready..."

# CLI Tests
title "Basic Back Navigation"

heading "Navigate to two pages, then back"
cmd "webctl navigate example.com"
cmd "webctl navigate wikipedia.org"
cmd "webctl back"
cmd "webctl status"

echo ""
echo "Status should show example.com (returned to first page)"
read -p "Press Enter to continue..."

heading "Verify immediate return (no wait)"
cmd "webctl navigate google.com"
cmd "webctl navigate github.com"
cmd "webctl back"

echo ""
echo "Should return immediately, navigation in background"
read -p "Press Enter to continue..."

title "Wait Functionality"

heading "Back with --wait flag"
cmd "webctl navigate example.com"
cmd "webctl navigate wikipedia.org"
cmd "webctl back --wait"

echo ""
echo "Should block until page loads"
read -p "Press Enter to continue..."

heading "Back with custom timeout"
cmd "webctl back --wait --timeout 60"

echo ""
echo "Should wait up to 60 seconds"
read -p "Press Enter to continue..."

title "History Traversal"

heading "Navigate through multiple pages, then back multiple times"
cmd "webctl navigate example.com"
cmd "webctl navigate wikipedia.org"
cmd "webctl navigate github.com"
cmd "webctl back"
cmd "webctl status"

echo ""
echo "Should be at wikipedia.org"
read -p "Press Enter to continue..."

cmd "webctl back"
cmd "webctl status"

echo ""
echo "Should be at example.com"
read -p "Press Enter to continue..."

heading "Combine back with forward"
cmd "webctl forward"
cmd "webctl status"

echo ""
echo "Should be back at wikipedia.org"
read -p "Press Enter to continue..."

title "Output Formats"

heading "Default text output"
cmd "webctl navigate example.com"
cmd "webctl navigate google.com"
cmd "webctl back"

echo ""
echo "Should show simple OK"
read -p "Press Enter to continue..."

heading "JSON output"
cmd "webctl back --json"

echo ""
echo "Should include URL and title in JSON"
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl back --no-color"

echo ""
echo "Should have no ANSI colors"
read -p "Press Enter to continue..."

title "Various Scenarios"

heading "Back from static to static page"
cmd "webctl navigate example.com"
cmd "webctl navigate wikipedia.org"
cmd "webctl back"

echo ""
echo "Should navigate between static pages cleanly"
read -p "Press Enter to continue..."

heading "Back with hash navigation"
cmd "webctl navigate https://developer.mozilla.org/en-US/"
cmd "webctl navigate https://developer.mozilla.org/en-US/#content"
cmd "webctl back"

echo ""
echo "Test hash navigation in history"
read -p "Press Enter to continue..."

title "Error Cases"

heading "Back when no previous page"
cmd "webctl stop"
cmd "webctl start"
cmd "webctl back"

echo ""
echo "Should show error - no previous page in history"
read -p "Press Enter to continue..."

heading "Back at beginning of history"
cmd "webctl navigate example.com"
cmd "webctl back"

echo ""
echo "Should navigate to about:blank (OK)"
read -p "Press Enter to continue..."

cmd "webctl back"

echo ""
echo "Should show error - already at beginning"
read -p "Press Enter to continue..."

title "Common Patterns"

heading "navigate && navigate && back"
cmd "webctl navigate example.com"
cmd "webctl navigate wikipedia.org"
cmd "webctl back"

echo ""
echo "Common workflow for testing navigation"
read -p "Press Enter to continue..."

heading "back && ready"
cmd "webctl navigate google.com"
cmd "webctl back && webctl ready"

echo ""
echo "Back and wait for ready state"
read -p "Press Enter to continue..."

heading "back --wait && html"
cmd "webctl navigate example.com"
cmd "webctl back --wait && webctl html"

echo ""
echo "Back, wait, then show HTML"
read -p "Press Enter to continue..."

title "REPL Mode Tests"

heading "Back from REPL"
echo "Switch to daemon terminal and execute:"
cmd "navigate github.com"
cmd "back"

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

title "Test Suite Complete"
echo "All back command tests finished"
echo ""
echo "Review checklist in docs/projects/p-032-testing-back.md"
echo "Document any issues discovered during testing"
