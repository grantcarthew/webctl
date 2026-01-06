#!/bin/bash
# Title: webctl ready command tests

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
title "webctl ready Command Test Suite"
echo "Project: P-047"
echo "Tests waiting for page/application ready state"
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

# Page load mode (default)
title "Page Load Mode (Default)"

heading "Navigate and wait for page load"
cmd "webctl navigate https://example.com"
cmd "webctl ready"

echo ""
echo "Verify: Waits for page load event (or returns immediately if already loaded)"
read -p "Press Enter to continue..."

heading "Ready with custom timeout"
cmd "webctl navigate https://example.com"
cmd "webctl ready --timeout 10s"

echo ""
echo "Verify: Waits up to 10 seconds for page load"
read -p "Press Enter to continue..."

heading "Ready on already loaded page (immediate return)"
cmd "webctl ready"

echo ""
echo "Verify: Returns immediately (page already loaded)"
read -p "Press Enter to continue..."

# Selector mode
title "Selector Mode (Wait for Element)"

heading "Navigate to GitHub and wait for specific element"
cmd "webctl navigate https://github.com"
cmd "webctl ready \"footer\""

echo ""
echo "Verify: Waits for footer element to appear"
read -p "Press Enter to continue..."

heading "Wait for element by class"
cmd "webctl navigate https://example.com --wait"
cmd "webctl ready \"p\""

echo ""
echo "Verify: Waits for paragraph element"
read -p "Press Enter to continue..."

heading "Wait for element by ID (if exists)"
cmd "webctl navigate https://example.com --wait"
cmd "webctl eval \"document.body.innerHTML += '<div id=\\\"test-ready\\\">Test</div>'\""
cmd "webctl ready \"#test-ready\""

echo ""
echo "Verify: Waits for dynamically added element"
read -p "Press Enter to continue..."

heading "Wait for attribute selector"
cmd "webctl navigate https://example.com --wait"
cmd "webctl ready \"[href]\""

echo ""
echo "Verify: Waits for element with href attribute"
read -p "Press Enter to continue..."

# Network idle mode
title "Network Idle Mode"

heading "Navigate and wait for network idle"
cmd "webctl navigate https://example.com"
cmd "webctl ready --network-idle"

echo ""
echo "Verify: Waits for network to be quiet for 500ms"
read -p "Press Enter to continue..."

heading "Network idle with custom timeout"
cmd "webctl navigate https://github.com"
cmd "webctl ready --network-idle --timeout 120s"

echo ""
echo "Verify: Waits up to 120s for network idle"
read -p "Press Enter to continue..."

heading "Network idle after page already loaded"
cmd "webctl ready --network-idle --timeout 30s"

echo ""
echo "Verify: Should return quickly if no active requests"
read -p "Press Enter to continue..."

# Eval mode
title "Eval Mode (Custom JavaScript Condition)"

heading "Wait for document ready state"
cmd "webctl navigate https://example.com"
cmd "webctl ready --eval \"document.readyState === 'complete'\""

echo ""
echo "Verify: Waits until document.readyState is 'complete'"
read -p "Press Enter to continue..."

heading "Wait for custom window variable"
cmd "webctl navigate https://example.com --wait"
cmd "webctl eval \"window.testReady = true\""
cmd "webctl ready --eval \"window.testReady === true\""

echo ""
echo "Verify: Returns immediately (variable already set)"
read -p "Press Enter to continue..."

heading "Wait for no error element"
cmd "webctl ready --eval \"document.querySelector('.error') === null\""

echo ""
echo "Verify: Waits until no error element exists"
read -p "Press Enter to continue..."

heading "Wait for element existence check"
cmd "webctl ready --eval \"!!document.querySelector('h1')\""

echo ""
echo "Verify: Waits for h1 element to exist"
read -p "Press Enter to continue..."

# Common navigation patterns
title "Common Navigation Patterns"

heading "Navigate then ready"
cmd "webctl navigate https://example.com && webctl ready"

echo ""
echo "Verify: Navigate and wait for load"
read -p "Press Enter to continue..."

heading "Navigate then ready for element"
cmd "webctl navigate https://example.com && webctl ready \"p\""

echo ""
echo "Verify: Navigate and wait for paragraph"
read -p "Press Enter to continue..."

heading "Navigate then network idle"
cmd "webctl navigate https://example.com && webctl ready --network-idle"

echo ""
echo "Verify: Navigate and wait for network quiet"
read -p "Press Enter to continue..."

# SPA patterns
title "SPA Patterns"

heading "Navigate to React site"
cmd "webctl navigate https://react.dev"
cmd "webctl ready --eval \"document.querySelector('main') !== null\""

echo ""
echo "Verify: Waits for main content to appear"
read -p "Press Enter to continue..."

# Timeout scenarios
title "Timeout Scenarios"

heading "Wait for non-existent element with short timeout (should timeout)"
cmd "webctl ready \"#nonexistent-element-xyz\" --timeout 2s"

echo ""
echo "Verify: Times out after 2 seconds with timeout error"
read -p "Press Enter to continue..."

heading "Eval condition that's false with timeout"
cmd "webctl ready --eval \"false\" --timeout 2s"

echo ""
echo "Verify: Times out after 2 seconds"
read -p "Press Enter to continue..."

# Chaining conditions
title "Chaining Multiple Ready Commands"

heading "Page load, then network idle, then custom eval"
cmd "webctl navigate https://example.com"
cmd "webctl ready"
cmd "webctl ready --network-idle"
cmd "webctl ready --eval \"document.querySelector('p') !== null\""

echo ""
echo "Verify: Each ready command completes in sequence"
read -p "Press Enter to continue..."

# Dynamic content scenario
title "Dynamic Content Loading"

heading "Create dynamic content scenario"
cmd "webctl navigate https://example.com --wait"
cmd "webctl eval \"setTimeout(() => { document.body.innerHTML += '<div class=\\\"new-content\\\">Loaded</div>'; }, 2000)\""

heading "Wait for dynamically loaded content"
cmd "webctl ready \".new-content\" --timeout 5s"

echo ""
echo "Verify: Waits ~2 seconds for element to appear"
read -p "Press Enter to continue..."

# Error cases
title "Error Cases"

heading "Ready with no active session (stop daemon first)"
echo "Note: This will stop the daemon"
cmd "webctl stop"

echo ""
echo "Wait a moment for daemon to stop"
read -p "Press Enter to continue..."

cmd "webctl ready"

echo ""
echo "Verify: Error message about no daemon or no session"
read -p "Press Enter to continue..."

heading "Restart daemon for remaining tests"
cmd "webctl start"

echo ""
read -p "Press Enter when daemon ready..."

cmd "webctl navigate https://example.com --wait"

echo ""
read -p "Press Enter when page loaded..."

# Output formats
title "Output Format Tests"

heading "Default text output"
cmd "webctl ready"

echo ""
echo "Verify: Shows OK or success message"
read -p "Press Enter to continue..."

heading "JSON output"
cmd "webctl ready --json"

echo ""
echo "Verify: JSON formatted {ok: true}"
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl ready --no-color"

echo ""
echo "Verify: No ANSI color codes"
read -p "Press Enter to continue..."

heading "Debug output"
cmd "webctl ready --debug"

echo ""
echo "Verify: Debug information shown"
read -p "Press Enter to continue..."

# REPL mode tests
title "REPL Mode Tests"

heading "Ready in REPL (default mode)"
echo "Switch to daemon terminal and execute:"
cmd "ready"

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

heading "Ready with selector in REPL"
echo "In REPL, try:"
cmd "ready \"p\""

echo ""
read -p "Press Enter when tested in REPL..."

heading "Ready with network-idle in REPL"
echo "In REPL, try:"
cmd "ready --network-idle"

echo ""
read -p "Press Enter when tested in REPL..."

heading "Ready with eval in REPL"
echo "In REPL, try:"
cmd "ready --eval \"document.readyState === 'complete'\""

echo ""
read -p "Press Enter when tested in REPL..."

# Test completed
title "Test Suite Complete"
echo "All ready command tests finished"
echo ""
echo "Review checklist in docs/projects/p-047-testing-ready.md"
echo "Document any issues discovered during testing"
