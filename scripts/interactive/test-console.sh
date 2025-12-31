#!/bin/bash
# Title: webctl console command tests

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
title "webctl console Command Test Suite"
echo "Project: P-036"
echo "Tests console log extraction with filtering and range limiting"
echo ""
echo "Prerequisites:"
echo "  - webctl must be built"
echo "  - xclip must be installed"
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

heading "Navigate to page with console logs"
echo "We'll navigate to a page and inject console logs for testing"
cmd "webctl navigate example.com --wait"

echo ""
echo "Wait for page to load"
read -p "Press Enter when page loaded..."

heading "Inject console logs for testing"
echo "Execute these in REPL or via evaluate command:"
cmd "webctl evaluate \"console.log('Test log message'); console.warn('Test warning'); console.error('Test error'); console.info('Test info'); console.debug('Test debug');\""

echo ""
echo "Console logs injected"
read -p "Press Enter to continue..."

# Default mode tests
title "Default Mode (Save to Temp)"

heading "Save all console logs to temp"
cmd "webctl console"

echo ""
echo "Verify: File saved to /tmp/webctl-console/"
echo "Verify: Auto-generated filename (YY-MM-DD-HHMMSS-console.json)"
echo "Verify: JSON response shows file path"
echo "Verify: JSON file contains logs array"
read -p "Press Enter to continue..."

heading "Save only errors to temp"
cmd "webctl console --type error"

echo ""
echo "Verify: Only error type logs saved"
read -p "Press Enter to continue..."

heading "Search and save logs"
cmd "webctl console --find \"Test\""

echo ""
echo "Verify: Logs containing 'Test' saved"
read -p "Press Enter to continue..."

# Show mode tests
title "Show Mode (Output to Stdout)"

heading "Show all console logs"
cmd "webctl console show"

echo ""
echo "Verify: Formatted logs to stdout"
echo "Verify: Shows timestamp, type, message"
echo "Verify: No file created"
read -p "Press Enter to continue..."

heading "Show only errors"
cmd "webctl console show --type error"

echo ""
echo "Verify: Only error logs shown"
read -p "Press Enter to continue..."

heading "Show errors and warnings (CSV)"
cmd "webctl console show --type error,warn"

echo ""
echo "Verify: Both errors and warnings shown"
read -p "Press Enter to continue..."

heading "Show errors and warnings (repeatable)"
cmd "webctl console show --type error --type warn"

echo ""
echo "Verify: Both errors and warnings shown"
read -p "Press Enter to continue..."

heading "Show with text search"
cmd "webctl console show --find \"warning\""

echo ""
echo "Verify: Logs containing 'warning' shown"
read -p "Press Enter to continue..."

# Save mode tests
title "Save Mode (Custom Path)"

heading "Save to custom file"
cmd "webctl console save ./logs.json"

echo ""
echo "Verify: File saved to ./logs.json"
read -p "Press Enter to continue..."

heading "Save to directory with auto-filename"
cmd "webctl console save ./output/"

echo ""
echo "Verify: File saved to ./output/ with auto-generated name"
read -p "Press Enter to continue..."

heading "Save errors with tail limit"
cmd "webctl console save ./errors.json --type error --tail 50"

echo ""
echo "Verify: Last 50 errors saved to ./errors.json"
read -p "Press Enter to continue..."

# Type filter tests
title "Type Filter Tests"

heading "Filter by log type"
cmd "webctl console show --type log"

echo ""
echo "Verify: Only 'log' entries shown"
read -p "Press Enter to continue..."

heading "Filter by warn type"
cmd "webctl console show --type warn"

echo ""
echo "Verify: Only 'warn' entries shown"
read -p "Press Enter to continue..."

heading "Filter by error type"
cmd "webctl console show --type error"

echo ""
echo "Verify: Only 'error' entries shown"
read -p "Press Enter to continue..."

heading "Filter by info type"
cmd "webctl console show --type info"

echo ""
echo "Verify: Only 'info' entries shown"
read -p "Press Enter to continue..."

heading "Filter by debug type"
cmd "webctl console show --type debug"

echo ""
echo "Verify: Only 'debug' entries shown"
read -p "Press Enter to continue..."

# Find flag tests
title "Find Flag Tests"

heading "Find simple text"
cmd "webctl console show --find \"Test\""

echo ""
echo "Verify: Case-insensitive search for 'Test'"
read -p "Press Enter to continue..."

heading "Find with no matches (should error)"
cmd "webctl console show --find \"ThisTextDoesNotExist123\""

echo ""
echo "Verify: Error message about no matches"
read -p "Press Enter to continue..."

heading "Find combined with type filter"
cmd "webctl console show --type error --find \"Test\""

echo ""
echo "Verify: Only errors containing 'Test'"
read -p "Press Enter to continue..."

# Head flag tests
title "Head Flag Tests"

heading "Get first 10 entries"
cmd "webctl console show --head 10"

echo ""
echo "Verify: First 10 log entries shown"
read -p "Press Enter to continue..."

heading "Get first entry"
cmd "webctl console show --head 1"

echo ""
echo "Verify: Only first log entry shown"
read -p "Press Enter to continue..."

heading "Head with type filter"
cmd "webctl console show --head 5 --type error"

echo ""
echo "Verify: First 5 error entries"
read -p "Press Enter to continue..."

# Tail flag tests
title "Tail Flag Tests"

heading "Get last 20 entries"
cmd "webctl console show --tail 20"

echo ""
echo "Verify: Last 20 log entries shown"
read -p "Press Enter to continue..."

heading "Get last entry"
cmd "webctl console show --tail 1"

echo ""
echo "Verify: Only last log entry shown"
read -p "Press Enter to continue..."

heading "Tail with type filter"
cmd "webctl console show --tail 5 --type warn"

echo ""
echo "Verify: Last 5 warning entries"
read -p "Press Enter to continue..."

# Range flag tests
title "Range Flag Tests"

heading "Get entries 10-20"
cmd "webctl console show --range 10-20"

echo ""
echo "Verify: Entries from index 10 to 20"
read -p "Press Enter to continue..."

heading "Get first 10 entries via range"
cmd "webctl console show --range 0-10"

echo ""
echo "Verify: First 10 entries"
read -p "Press Enter to continue..."

heading "Range with type filter"
cmd "webctl console show --range 0-5 --type error"

echo ""
echo "Verify: First 5 error entries"
read -p "Press Enter to continue..."

# Mutual exclusivity tests
title "Mutual Exclusivity Tests"

heading "Head and tail together (should error)"
cmd "webctl console show --head 10 --tail 10"

echo ""
echo "Verify: Error message about mutually exclusive flags"
read -p "Press Enter to continue..."

heading "Head and range together (should error)"
cmd "webctl console show --head 10 --range 0-10"

echo ""
echo "Verify: Error message about mutually exclusive flags"
read -p "Press Enter to continue..."

heading "Tail and range together (should error)"
cmd "webctl console show --tail 10 --range 0-10"

echo ""
echo "Verify: Error message about mutually exclusive flags"
read -p "Press Enter to continue..."

# Raw flag tests
title "Raw Flag Tests"

heading "Raw output (JSON format)"
cmd "webctl console show --raw"

echo ""
echo "Verify: Raw JSON output instead of formatted text"
read -p "Press Enter to continue..."

heading "Raw with filters"
cmd "webctl console show --raw --type error --tail 10"

echo ""
echo "Verify: Raw JSON with filtered results"
read -p "Press Enter to continue..."

# Output format tests
title "Output Format Tests"

heading "JSON output with show mode"
cmd "webctl console show --json"

echo ""
echo "Verify: JSON formatted output"
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl console show --no-color"

echo ""
echo "Verify: No ANSI color codes"
read -p "Press Enter to continue..."

heading "Debug verbose output"
cmd "webctl console show --debug"

echo ""
echo "Verify: Debug logging information"
read -p "Press Enter to continue..."

# REPL mode tests
title "REPL Mode Tests"

heading "Test console in REPL"
echo "Switch to daemon terminal and execute:"
cmd "console show"

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

heading "Test console with filters in REPL"
echo "In REPL, try:"
cmd "console show --type error --tail 10"

echo ""
echo "Should show last 10 errors"
read -p "Press Enter when tested in REPL..."

# Advanced scenarios
title "Advanced Scenarios"

heading "Navigate to page with natural console logs"
cmd "webctl navigate https://github.com --wait"

echo ""
echo "Wait for GitHub to load (may have console messages)"
read -p "Press Enter when loaded..."

heading "Check for console logs"
cmd "webctl console show"

echo ""
echo "Verify: Any console logs from GitHub shown"
read -p "Press Enter to continue..."

heading "Inject more logs for variety"
cmd "webctl evaluate \"for(let i=0; i<20; i++) { if(i%2===0) console.log('Even:', i); else console.warn('Odd:', i); }\""

echo ""
echo "20 logs injected (mix of log and warn)"
read -p "Press Enter to continue..."

heading "Get last 10 warnings"
cmd "webctl console show --type warn --tail 10"

echo ""
echo "Verify: Last 10 warning messages"
read -p "Press Enter to continue..."

heading "Find logs containing numbers"
cmd "webctl console show --find \"10\""

echo ""
echo "Verify: Logs containing '10'"
read -p "Press Enter to continue..."

# Cleanup
title "Cleanup"
echo "Clean up test files if desired:"
cmd "rm -f ./logs.json ./errors.json ./output/*.json"

echo ""
echo "Remove test files when ready"
read -p "Press Enter to finish..."

title "Test Suite Complete"
echo "All console command tests finished"
echo ""
echo "Review checklist in docs/projects/p-036-testing-console.md"
echo "Document any issues discovered during testing"
