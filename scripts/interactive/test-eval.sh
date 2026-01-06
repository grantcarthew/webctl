#!/bin/bash
# Title: webctl eval command tests

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
title "webctl eval Command Test Suite"
echo "Project: P-046"
echo "Tests JavaScript evaluation in browser context"
echo ""
echo "Prerequisites:"
echo "  - webctl must be built"
echo "  - Clipboard tool (pbcopy on macOS, xclip on Linux)"
echo "  - Daemon running (or start one)"
echo ""
read -p "Press Enter to begin tests..."

# Setup
title "Setup"
echo "Ensure daemon is running and navigate to test page"
cmd "webctl start"

echo ""
echo "Start daemon if not running, then continue"
read -p "Press Enter when daemon ready..."

heading "Navigate to example.com for testing"
cmd "webctl navigate https://example.com --wait"

echo ""
read -p "Press Enter when page loaded..."

# Simple expressions
title "Simple Expressions"

heading "Arithmetic: 1 + 1"
cmd "webctl eval \"1 + 1\""

echo ""
echo "Verify: Returns 2"
read -p "Press Enter to continue..."

heading "Get document title"
cmd "webctl eval \"document.title\""

echo ""
echo "Verify: Returns page title"
read -p "Press Enter to continue..."

heading "Get current URL"
cmd "webctl eval \"window.location.href\""

echo ""
echo "Verify: Returns https://example.com/"
read -p "Press Enter to continue..."

heading "Get timestamp"
cmd "webctl eval \"Date.now()\""

echo ""
echo "Verify: Returns current timestamp (large number)"
read -p "Press Enter to continue..."

heading "Math operation"
cmd "webctl eval \"Math.random()\""

echo ""
echo "Verify: Returns random number between 0 and 1"
read -p "Press Enter to continue..."

heading "Boolean logic"
cmd "webctl eval \"true && false\""

echo ""
echo "Verify: Returns false"
read -p "Press Enter to continue..."

# Object and array results
title "Object and Array Results"

heading "Array map operation"
cmd "webctl eval \"[1, 2, 3].map(x => x * 2)\""

echo ""
echo "Verify: Returns [2, 4, 6]"
read -p "Press Enter to continue..."

heading "Object literal"
cmd "webctl eval \"({name: 'test', count: 42})\""

echo ""
echo "Verify: Returns object with name and count"
read -p "Press Enter to continue..."

heading "Array of link URLs"
cmd "webctl eval \"Array.from(document.querySelectorAll('a')).map(a => a.href)\""

echo ""
echo "Verify: Returns array of URLs"
read -p "Press Enter to continue..."

heading "Simple array"
cmd "webctl eval \"[1, 2, 3, 4, 5]\""

echo ""
echo "Verify: Returns [1, 2, 3, 4, 5]"
read -p "Press Enter to continue..."

# DOM inspection
title "DOM Inspection (Values Only)"

heading "Count anchor elements"
cmd "webctl eval \"document.querySelectorAll('a').length\""

echo ""
echo "Verify: Returns number of links on page"
read -p "Press Enter to continue..."

heading "Get text content"
cmd "webctl eval \"document.querySelector('h1').textContent.trim()\""

echo ""
echo "Verify: Returns heading text"
read -p "Press Enter to continue..."

heading "Check element existence"
cmd "webctl eval \"document.querySelector('div') !== null\""

echo ""
echo "Verify: Returns true"
read -p "Press Enter to continue..."

heading "Get computed style"
cmd "webctl eval \"getComputedStyle(document.body).backgroundColor\""

echo ""
echo "Verify: Returns background color value"
read -p "Press Enter to continue..."

# Async/Promise expressions
title "Async/Promise Expressions"

heading "Navigate to httpbin for async testing"
cmd "webctl navigate https://httpbin.org --wait"

echo ""
read -p "Press Enter when page loaded..."

heading "Fetch API (async)"
cmd "webctl eval \"fetch('https://httpbin.org/json').then(r => r.json())\""

echo ""
echo "Verify: Returns JSON object from API"
read -p "Press Enter to continue..."

heading "Promise with delay"
cmd "webctl eval \"new Promise(r => setTimeout(() => r('done'), 1000))\""

echo ""
echo "Verify: Waits ~1 second, then returns 'done'"
read -p "Press Enter to continue..."

heading "Resolved promise"
cmd "webctl eval \"Promise.resolve('immediate')\""

echo ""
echo "Verify: Returns 'immediate'"
read -p "Press Enter to continue..."

# Multi-statement with IIFE
title "Multi-Statement with IIFE"

heading "IIFE with variables"
cmd "webctl eval \"(function() { const x = 1; const y = 2; return x + y; })()\""

echo ""
echo "Verify: Returns 3"
read -p "Press Enter to continue..."

heading "IIFE with loop"
cmd "webctl eval \"(() => { let sum = 0; for(let i=0; i<10; i++) sum += i; return sum; })()\""

echo ""
echo "Verify: Returns 45 (sum of 0-9)"
read -p "Press Enter to continue..."

# Modify page state
title "Modify Page State"

heading "Change background color"
cmd "webctl eval \"document.body.style.background = 'red'\""

echo ""
echo "Verify: Background turns red"
read -p "Press Enter to continue..."

heading "Set localStorage"
cmd "webctl eval \"localStorage.setItem('debug', 'true')\""

echo ""
echo "Verify: Returns undefined (no error)"
read -p "Press Enter to continue..."

heading "Verify localStorage was set"
cmd "webctl eval \"localStorage.getItem('debug')\""

echo ""
echo "Verify: Returns 'true'"
read -p "Press Enter to continue..."

heading "Change document title"
cmd "webctl eval \"document.title = 'New Title'\""

echo ""
echo "Verify: Browser tab title changes to 'New Title'"
read -p "Press Enter to continue..."

heading "Reset background"
cmd "webctl eval \"document.body.style.background = ''\""

echo ""
read -p "Press Enter to continue..."

# Timeout handling
title "Timeout Handling"

heading "Async with 5s timeout (should succeed)"
cmd "webctl eval --timeout 5s \"new Promise(r => setTimeout(() => r('done'), 1000))\""

echo ""
echo "Verify: Completes successfully after ~1 second"
read -p "Press Enter to continue..."

heading "Async with 1s timeout on 5s promise (should timeout)"
cmd "webctl eval --timeout 1s \"new Promise(r => setTimeout(() => r('done'), 5000))\""

echo ""
echo "Verify: Times out with timeout error"
read -p "Press Enter to continue..."

heading "Short timeout flag"
cmd "webctl eval -t 10s \"fetch('https://httpbin.org/delay/2').then(r => r.json())\""

echo ""
echo "Verify: Completes successfully (delay is 2s, timeout is 10s)"
read -p "Press Enter to continue..."

# Return values
title "Return Value Handling"

heading "Undefined return value"
cmd "webctl eval \"undefined\""

echo ""
echo "Verify: Returns {ok: true} with no value field"
read -p "Press Enter to continue..."

heading "Null return value"
cmd "webctl eval \"null\""

echo ""
echo "Verify: Returns {ok: true, value: null}"
read -p "Press Enter to continue..."

heading "False return value"
cmd "webctl eval \"false\""

echo ""
echo "Verify: Returns {ok: true, value: false}"
read -p "Press Enter to continue..."

# Error cases
title "Error Cases"

heading "Syntax error"
cmd "webctl eval \"invalid syntax {{\""

echo ""
echo "Verify: SyntaxError message"
read -p "Press Enter to continue..."

heading "Reference error"
cmd "webctl eval \"undefinedVariable\""

echo ""
echo "Verify: ReferenceError message"
read -p "Press Enter to continue..."

heading "Runtime error"
cmd "webctl eval \"throw new Error('test error')\""

echo ""
echo "Verify: Error message shown"
read -p "Press Enter to continue..."

# Output formats
title "Output Format Tests"

heading "Default text output"
cmd "webctl eval \"document.title\""

echo ""
echo "Verify: Shows raw value only"
read -p "Press Enter to continue..."

heading "JSON output"
cmd "webctl eval \"document.title\" --json"

echo ""
echo "Verify: JSON with ok and value fields"
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl eval \"1 + 1\" --no-color"

echo ""
echo "Verify: No ANSI color codes"
read -p "Press Enter to continue..."

heading "Debug output"
cmd "webctl eval \"1 + 1\" --debug"

echo ""
echo "Verify: Debug information shown"
read -p "Press Enter to continue..."

# Multi-arg expressions
title "Multi-Arg Expression Handling"

heading "Expression without quotes (multi-arg)"
cmd "webctl eval 1 + 1"

echo ""
echo "Verify: Args joined as '1 + 1', returns 2"
read -p "Press Enter to continue..."

heading "Property access without quotes"
cmd "webctl eval document.title"

echo ""
echo "Verify: Returns page title"
read -p "Press Enter to continue..."

# Application state
title "Application State Access"

heading "Get sessionStorage"
cmd "webctl eval \"sessionStorage.setItem('token', 'abc123')\""
cmd "webctl eval \"sessionStorage.getItem('token')\""

echo ""
echo "Verify: Returns 'abc123'"
read -p "Press Enter to continue..."

heading "Get cookies"
cmd "webctl eval \"document.cookie\""

echo ""
echo "Verify: Returns cookie string (may be empty)"
read -p "Press Enter to continue..."

# REPL mode tests
title "REPL Mode Tests"

heading "Simple eval in REPL"
echo "Switch to daemon terminal and execute:"
cmd "eval \"document.title\""

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

heading "Arithmetic in REPL"
echo "In REPL, try:"
cmd "eval \"1 + 1\""

echo ""
read -p "Press Enter when tested in REPL..."

heading "Array operation in REPL"
echo "In REPL, try:"
cmd "eval \"[1,2,3].map(x => x * 2)\""

echo ""
read -p "Press Enter when tested in REPL..."

heading "Async in REPL"
echo "In REPL, try:"
cmd "eval \"Promise.resolve('test')\""

echo ""
read -p "Press Enter when tested in REPL..."

# Test completed
title "Test Suite Complete"
echo "All eval command tests finished"
echo ""
echo "Review checklist in docs/projects/p-046-testing-eval.md"
echo "Document any issues discovered during testing"
