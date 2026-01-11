#!/bin/bash
# Title: webctl network command tests

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
title "webctl network Command Test Suite"
echo "Project: P-037"
echo "Tests network request extraction with extensive filtering"
echo ""
echo "Prerequisites:"
echo "  - webctl must be built"
echo "  - Clipboard tool (pbcopy on macOS, xclip on Linux)"
echo "  - testdata/index.html (comprehensive test page with network button)"
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

heading "Option 1: Use testdata page with network test button (Recommended)"
echo "Navigate to testdata page which has a network request test button"
cmd "webctl serve \$(git rev-parse --show-toplevel)/testdata"

echo ""
echo "The test page has a 'Network Request' button that:"
echo "  - Makes API call to JSONPlaceholder"
echo "  - Fetches POST data from external API"
echo "  - Provides realistic AJAX request for testing"
echo ""
echo "Click the 'Network Request' button to generate network activity"
echo "This provides realistic user-triggered network requests"
read -p "Press Enter when you've made some requests (or skip to Option 2)..."

heading "Option 2: Use example.com"
echo "Alternatively, navigate to example.com (simpler page)"
cmd "webctl navigate https://example.com --wait"

echo ""
echo "Wait for page to load (network requests will be captured)"
read -p "Press Enter when page loaded..."

# Default mode tests
title "Default Mode (Output to Stdout)"

heading "Show all network requests"
cmd "webctl network"

echo ""
echo "Verify: Formatted requests to stdout"
echo "Verify: Shows method, URL, status, duration"
echo "Verify: No file created"
read -p "Press Enter to continue..."

heading "Show only errors (4xx, 5xx)"
cmd "webctl network --status 4xx,5xx"

echo ""
echo "Verify: Only error status codes shown"
read -p "Press Enter to continue..."

heading "Show last 20 requests"
cmd "webctl network --tail 20"

echo ""
echo "Verify: Last 20 network requests shown"
read -p "Press Enter to continue..."

# Save mode tests
title "Save Mode (File Output)"

heading "Save to temp (no path)"
cmd "webctl network save"

echo ""
echo "Verify: File saved to /tmp/webctl-network/"
echo "Verify: Auto-generated filename (YY-MM-DD-HHMMSS-network.json)"
echo "Verify: JSON response shows file path"
read -p "Press Enter to continue..."

heading "Save to custom file"
cmd "webctl network save ./requests.json"

echo ""
echo "Verify: File saved to ./requests.json"
read -p "Press Enter to continue..."

heading "Save to directory with auto-filename (trailing slash = directory)"
cmd "webctl network save ./output/"

echo ""
echo "Verify: File saved to ./output/ with auto-generated name"
echo "Note: Trailing slash (/) is REQUIRED for directory behavior"
echo "      Without slash, it would create a file named 'output'"
read -p "Press Enter to continue..."

heading "Save errors with tail limit"
cmd "webctl network save ./errors.json --status 5xx --tail 50"

echo ""
echo "Verify: Last 50 5xx errors saved"
read -p "Press Enter to continue..."

# Type filter tests
title "Type Filter Tests"

heading "Filter by document type"
cmd "webctl network --type document"

echo ""
echo "Verify: Only HTML document requests shown"
read -p "Press Enter to continue..."

heading "Filter by script type"
cmd "webctl network --type script"

echo ""
echo "Verify: Only JavaScript requests shown"
read -p "Press Enter to continue..."

heading "Filter by stylesheet type"
cmd "webctl network --type stylesheet"

echo ""
echo "Verify: Only CSS requests shown"
read -p "Press Enter to continue..."

heading "Filter by image type"
cmd "webctl network --type image"

echo ""
echo "Verify: Only image requests shown"
read -p "Press Enter to continue..."

heading "Filter by xhr type"
cmd "webctl network --type xhr"

echo ""
echo "Verify: Only XMLHttpRequest shown"
read -p "Press Enter to continue..."

heading "Filter by fetch type"
cmd "webctl network --type fetch"

echo ""
echo "Verify: Only Fetch API requests shown"
read -p "Press Enter to continue..."

heading "Filter multiple types (CSV)"
cmd "webctl network --type xhr,fetch"

echo ""
echo "Verify: Both XHR and Fetch requests shown"
read -p "Press Enter to continue..."

heading "Filter multiple types (repeatable)"
cmd "webctl network --type xhr --type fetch"

echo ""
echo "Verify: Both XHR and Fetch requests shown"
read -p "Press Enter to continue..."

# Method filter tests
title "Method Filter Tests"

heading "Filter by GET method"
cmd "webctl network --method GET"

echo ""
echo "Verify: Only GET requests shown"
read -p "Press Enter to continue..."

heading "Filter by POST method"
cmd "webctl network --method POST"

echo ""
echo "Verify: Only POST requests shown (if any)"
read -p "Press Enter to continue..."

heading "Filter multiple methods (CSV)"
cmd "webctl network --method GET,POST"

echo ""
echo "Verify: Both GET and POST requests shown"
read -p "Press Enter to continue..."

heading "Filter multiple methods (repeatable)"
cmd "webctl network --method GET --method POST"

echo ""
echo "Verify: Both GET and POST requests shown"
read -p "Press Enter to continue..."

# Status filter tests
title "Status Filter Tests"

heading "Filter exact status code (200)"
cmd "webctl network --status 200"

echo ""
echo "Verify: Only 200 OK responses shown"
read -p "Press Enter to continue..."

heading "Filter status wildcard (4xx)"
cmd "webctl network --status 4xx"

echo ""
echo "Verify: All 4xx client errors shown"
read -p "Press Enter to continue..."

heading "Filter status wildcard (5xx)"
cmd "webctl network --status 5xx"

echo ""
echo "Verify: All 5xx server errors shown"
read -p "Press Enter to continue..."

heading "Filter status range (200-299)"
cmd "webctl network --status 200-299"

echo ""
echo "Verify: All 2xx success responses shown"
read -p "Press Enter to continue..."

heading "Filter multiple status patterns (CSV)"
cmd "webctl network --status 4xx,5xx"

echo ""
echo "Verify: All 4xx and 5xx errors shown"
read -p "Press Enter to continue..."

heading "Filter multiple status patterns (repeatable)"
cmd "webctl network --status 4xx --status 5xx"

echo ""
echo "Verify: All 4xx and 5xx errors shown"
read -p "Press Enter to continue..."

# URL filter tests
title "URL Filter Tests"

heading "Filter URL with simple pattern"
cmd "webctl network --url \"example\""

echo ""
echo "Verify: URLs containing 'example' shown"
read -p "Press Enter to continue..."

heading "Filter URL with regex (starts with https)"
cmd "webctl network --url \"^https://\""

echo ""
echo "Verify: Only HTTPS URLs shown"
read -p "Press Enter to continue..."

heading "Filter URL with regex (ends with .com)"
cmd "webctl network --url \"\\.com\""

echo ""
echo "Verify: URLs ending with .com shown"
read -p "Press Enter to continue..."

# MIME filter tests
title "MIME Filter Tests"

heading "Filter by text/html"
cmd "webctl network --mime text/html"

echo ""
echo "Verify: Only HTML documents shown"
read -p "Press Enter to continue..."

heading "Filter by application/json"
cmd "webctl network --mime application/json"

echo ""
echo "Verify: Only JSON responses shown (if any)"
read -p "Press Enter to continue..."

heading "Filter multiple MIME types (CSV)"
cmd "webctl network --mime text/html,text/css"

echo ""
echo "Verify: HTML and CSS responses shown"
read -p "Press Enter to continue..."

# Duration filter tests
title "Duration Filter Tests"

heading "Filter requests over 1 second"
cmd "webctl network --min-duration 1s"

echo ""
echo "Verify: Only slow requests (>1s) shown"
read -p "Press Enter to continue..."

heading "Filter requests over 500ms"
cmd "webctl network --min-duration 500ms"

echo ""
echo "Verify: Requests taking >500ms shown"
read -p "Press Enter to continue..."

heading "Filter requests over 100ms"
cmd "webctl network --min-duration 100ms"

echo ""
echo "Verify: Requests taking >100ms shown"
read -p "Press Enter to continue..."

# Size filter tests
title "Size Filter Tests"

heading "Filter requests over 1KB"
cmd "webctl network --min-size 1024"

echo ""
echo "Verify: Only responses >1KB shown"
read -p "Press Enter to continue..."

heading "Filter requests over 100KB"
cmd "webctl network --min-size 102400"

echo ""
echo "Verify: Only responses >100KB shown"
read -p "Press Enter to continue..."

# Failed filter tests
title "Failed Filter Tests"

heading "Show only failed requests"
cmd "webctl network --failed"

echo ""
echo "Verify: Only failed requests shown (network errors, CORS, etc.)"
read -p "Press Enter to continue..."

heading "Failed with other filters"
cmd "webctl network --failed --type xhr"

echo ""
echo "Verify: Only failed XHR requests shown"
read -p "Press Enter to continue..."

# Find flag tests
title "Find Flag Tests"

heading "Find in URL"
cmd "webctl network --find \"example\""

echo ""
echo "Verify: Case-insensitive search in URLs"
read -p "Press Enter to continue..."

heading "Find with no matches (should error)"
cmd "webctl network --find \"ThisTextDoesNotExist123\""

echo ""
echo "Verify: Error message about no matches"
read -p "Press Enter to continue..."

heading "Find combined with filters"
cmd "webctl network --find \"example\" --type document"

echo ""
echo "Verify: Documents containing 'example' in URL"
read -p "Press Enter to continue..."

# Head flag tests
title "Head Flag Tests"

heading "Get first 10 requests"
cmd "webctl network --head 10"

echo ""
echo "Verify: First 10 network requests shown"
read -p "Press Enter to continue..."

heading "Head with filters"
cmd "webctl network --head 5 --type script"

echo ""
echo "Verify: First 5 script requests"
read -p "Press Enter to continue..."

# Tail flag tests
title "Tail Flag Tests"

heading "Get last 20 requests"
cmd "webctl network --tail 20"

echo ""
echo "Verify: Last 20 network requests shown"
read -p "Press Enter to continue..."

heading "Tail with filters"
cmd "webctl network --tail 5 --method GET"

echo ""
echo "Verify: Last 5 GET requests"
read -p "Press Enter to continue..."

# Range flag tests
title "Range Flag Tests"

heading "Get requests 10-20"
cmd "webctl network --range 10-20"

echo ""
echo "Verify: Requests from index 10 to 20"
read -p "Press Enter to continue..."

heading "Range with filters"
cmd "webctl network --range 0-5 --type document"

echo ""
echo "Verify: First 5 document requests"
read -p "Press Enter to continue..."

# Mutual exclusivity tests
title "Mutual Exclusivity Tests"

heading "Head and tail together (should error)"
cmd "webctl network --head 10 --tail 10"

echo ""
echo "Verify: Error message about mutually exclusive flags"
read -p "Press Enter to continue..."

# Raw flag tests
title "Raw Flag Tests"

heading "Raw output (JSON format)"
cmd "webctl network --raw"

echo ""
echo "Verify: Raw JSON output instead of formatted text"
read -p "Press Enter to continue..."

heading "Raw with filters"
cmd "webctl network --raw --status 200 --tail 10"

echo ""
echo "Verify: Raw JSON with filtered results"
read -p "Press Enter to continue..."

# Max body size tests
title "Max Body Size Tests"

heading "Limit body size to 1KB"
cmd "webctl network --max-body-size 1024"

echo ""
echo "Verify: Large bodies truncated in output"
read -p "Press Enter to continue..."

heading "No bodies (max-body-size 0)"
cmd "webctl network --max-body-size 0"

echo ""
echo "Verify: No request/response bodies included"
read -p "Press Enter to continue..."

# Complex filter combinations
title "Complex Filter Combinations"

heading "Type + Method + Status"
cmd "webctl network --type xhr --method POST --status 200"

echo ""
echo "Verify: Only successful POST XHR requests"
read -p "Press Enter to continue..."

heading "Multiple filters with AND logic"
cmd "webctl network --type document --method GET --status 200 --min-duration 100ms"

echo ""
echo "Verify: Successful GET documents taking >100ms"
read -p "Press Enter to continue..."

# Output format tests
title "Output Format Tests"

heading "JSON output"
cmd "webctl network --json"

echo ""
echo "Verify: JSON formatted output"
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl network --no-color"

echo ""
echo "Verify: No ANSI color codes"
read -p "Press Enter to continue..."

# REPL mode tests
title "REPL Mode Tests"

heading "Test network in REPL"
echo "Switch to daemon terminal and execute:"
cmd "network"

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

heading "Test network with filters in REPL"
echo "In REPL, try:"
cmd "network --status 4xx --tail 10"

echo ""
echo "Should show last 10 4xx errors"
read -p "Press Enter when tested in REPL..."

# Advanced scenarios
title "Advanced Scenarios"

heading "Navigate to API-heavy page"
cmd "webctl navigate https://github.com --wait"

echo ""
echo "Wait for GitHub to load (many network requests)"
read -p "Press Enter when loaded..."

heading "Show all XHR/Fetch API requests"
cmd "webctl network --type xhr,fetch"

echo ""
echo "Verify: API requests from GitHub"
read -p "Press Enter to continue..."

heading "Find API endpoints"
cmd "webctl network --url \"api\""

echo ""
echo "Verify: URLs containing 'api'"
read -p "Press Enter to continue..."

heading "Show JSON responses"
cmd "webctl network --mime application/json"

echo ""
echo "Verify: JSON API responses"
read -p "Press Enter to continue..."

heading "Save all network data for analysis"
cmd "webctl network save ./github-network.json"

echo ""
echo "Verify: Complete network log saved"
read -p "Press Enter to continue..."

# Cleanup
title "Cleanup"
echo "Clean up test files if desired:"
cmd "rm -f ./requests.json ./errors.json ./output/*.json ./github-network.json"

echo ""
echo "Remove test files when ready"
read -p "Press Enter to finish..."

title "Test Suite Complete"
echo "All network command tests finished"
echo ""
echo "Review checklist in docs/projects/p-037-testing-network.md"
echo "Document any issues discovered during testing"
