#!/bin/bash
# Title: webctl find command tests

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
title "webctl find Command Test Suite"
echo "Project: P-049"
echo "Tests searching HTML content for text patterns"
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

heading "Navigate to example.com (simple page for testing)"
cmd "webctl navigate https://example.com --wait"

echo ""
read -p "Press Enter when page loaded..."

# Basic text search (case-insensitive)
title "Basic Text Search (Case-Insensitive)"

heading "Find 'example'"
cmd "webctl find \"example\""

echo ""
echo "Verify: Finds matches (case-insensitive)"
echo "Verify: Shows context (line before and after)"
echo "Verify: Matching line prefixed with '>'"
echo "Verify: Matched text highlighted in yellow"
read -p "Press Enter to continue..."

heading "Find 'domain'"
cmd "webctl find \"domain\""

echo ""
echo "Verify: Finds 'domain', 'Domain', etc."
read -p "Press Enter to continue..."

heading "Find 'more'"
cmd "webctl find \"more\""

echo ""
echo "Verify: Finds 'more' and 'More'"
read -p "Press Enter to continue..."

# Case-sensitive search
title "Case-Sensitive Search"

heading "Navigate to GitHub for more complex content"
cmd "webctl navigate https://github.com --wait"

echo ""
read -p "Press Enter when page loaded..."

heading "Case-sensitive 'GitHub'"
cmd "webctl find -c \"GitHub\""

echo ""
echo "Verify: Finds only 'GitHub' (exact case)"
read -p "Press Enter to continue..."

heading "Case-sensitive 'github' (lowercase)"
cmd "webctl find --case-sensitive \"github\""

echo ""
echo "Verify: Finds only lowercase 'github'"
read -p "Press Enter to continue..."

# Regex pattern search
title "Regex Pattern Search"

heading "Navigate back to example.com"
cmd "webctl navigate https://example.com --wait"

echo ""
read -p "Press Enter when page loaded..."

heading "Regex alternation pattern"
cmd "webctl find -E \"example|domain\""

echo ""
echo "Verify: Finds both 'example' and 'domain'"
read -p "Press Enter to continue..."

heading "Regex with wildcard"
cmd "webctl find -E \"<div.*>\""

echo ""
echo "Verify: Finds div tags with any attributes"
read -p "Press Enter to continue..."

heading "Regex for URLs"
cmd "webctl find -E \"https?://[^\\s]+\""

echo ""
echo "Verify: Finds HTTP/HTTPS URLs"
read -p "Press Enter to continue..."

heading "Regex line start"
cmd "webctl find -E \"^\\s*<\""

echo ""
echo "Verify: Finds lines starting with HTML tags"
read -p "Press Enter to continue..."

# Limit results
title "Limit Results"

heading "Navigate to GitHub for many matches"
cmd "webctl navigate https://github.com --wait"

echo ""
read -p "Press Enter when page loaded..."

heading "Limit to first 5 matches"
cmd "webctl find --limit 5 \"the\""

echo ""
echo "Verify: Shows only first 5 matches"
read -p "Press Enter to continue..."

heading "Limit to first match"
cmd "webctl find -l 1 \"github\""

echo ""
echo "Verify: Shows only first match"
read -p "Press Enter to continue..."

heading "Limit to 10 matches"
cmd "webctl find --limit 10 \"href\""

echo ""
echo "Verify: Shows only first 10 matches"
read -p "Press Enter to continue..."

# Minimum query length validation
title "Minimum Query Length Validation"

heading "Query too short (2 chars)"
cmd "webctl find \"ab\""

echo ""
echo "Verify: Error - query must be at least 3 characters"
read -p "Press Enter to continue..."

heading "Query too short (1 char)"
cmd "webctl find \"a\""

echo ""
echo "Verify: Error - query must be at least 3 characters"
read -p "Press Enter to continue..."

heading "Query exactly 3 chars (should succeed)"
cmd "webctl find \"the\""

echo ""
echo "Verify: Succeeds (3 characters is minimum)"
read -p "Press Enter to continue..."

# JSON output format
title "JSON Output Format"

heading "Navigate to example.com"
cmd "webctl navigate https://example.com --wait"

echo ""
read -p "Press Enter when page loaded..."

heading "JSON output for 'example'"
cmd "webctl find --json \"example\""

echo ""
echo "Verify: JSON includes query field"
echo "Verify: JSON includes total count"
echo "Verify: JSON includes matches array"
echo "Verify: Each match has CSS selector"
echo "Verify: Each match has XPath"
read -p "Press Enter to continue..."

heading "JSON output with limit"
cmd "webctl find --json --limit 2 \"more\""

echo ""
echo "Verify: JSON shows limited results"
read -p "Press Enter to continue..."

# Special characters in search
title "Special Characters in Search"

heading "Find with dot"
cmd "webctl find \"example.com\""

echo ""
echo "Verify: Finds 'example.com'"
read -p "Press Enter to continue..."

heading "Find with @ sign (email-like)"
cmd "webctl find \"@example\""

echo ""
echo "Verify: Finds matches with @ sign"
read -p "Press Enter to continue..."

# Error cases
title "Error Cases"

heading "Invalid regex pattern"
cmd "webctl find -E \"[[invalid\""

echo ""
echo "Verify: Error - invalid regex pattern"
read -p "Press Enter to continue..."

heading "Case-sensitive with regex (invalid combination)"
cmd "webctl find --case-sensitive -E \"test\""

echo ""
echo "Verify: Error - cannot use --case-sensitive with --regex"
read -p "Press Enter to continue..."

# No matches found
title "No Matches Found"

heading "Search for non-existent text"
cmd "webctl find \"xyznonexistenttext123\""

echo ""
echo "Verify: No matches message"
echo "Verify: total: 0 in output"
read -p "Press Enter to continue..."

heading "JSON output with no matches"
cmd "webctl find --json \"nonexistent999\""

echo ""
echo "Verify: JSON with total: 0, empty matches array"
read -p "Press Enter to continue..."

# Many matches scenario
title "Many Matches Scenario"

heading "Navigate to GitHub (lots of content)"
cmd "webctl navigate https://github.com --wait"

echo ""
read -p "Press Enter when page loaded..."

heading "Find common word without limit"
cmd "webctl find \"the\""

echo ""
echo "Verify: Shows many matches"
read -p "Press Enter to continue..."

heading "Find common word with limit"
cmd "webctl find --limit 20 \"and\""

echo ""
echo "Verify: Limits to 20 matches"
read -p "Press Enter to continue..."

# Different HTML structures
title "Different HTML Structures"

heading "Navigate to example.com"
cmd "webctl navigate https://example.com --wait"

echo ""
read -p "Press Enter when page loaded..."

heading "Find in title tag"
cmd "webctl find \"Example\""

echo ""
echo "Verify: Finds text in <title> tag"
read -p "Press Enter to continue..."

heading "Find in meta description"
cmd "webctl find \"illustrative\""

echo ""
echo "Verify: Finds text in meta tags"
read -p "Press Enter to continue..."

heading "Find in heading"
cmd "webctl find -E \"<h1.*Example\""

echo ""
echo "Verify: Finds heading content"
read -p "Press Enter to continue..."

# Output format variations
title "Output Format Variations"

heading "Default text output"
cmd "webctl find \"example\""

echo ""
echo "Verify: Text format with context and highlighting"
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl find \"example\" --no-color"

echo ""
echo "Verify: No ANSI color codes, no highlighting"
read -p "Press Enter to continue..."

heading "Debug output"
cmd "webctl find \"example\" --debug"

echo ""
echo "Verify: Debug information shown"
read -p "Press Enter to continue..."

# Piping demonstration
title "Piping to Extract Selectors"

heading "Find with JSON and pipe to jq (if available)"
echo "This test requires jq to be installed"
cmd "webctl find --json \"more\" | jq -r '.matches[0].selector'"

echo ""
echo "Verify: Extracts CSS selector from first match"
echo "Note: Will error if jq not installed (that's expected)"
read -p "Press Enter to continue..."

heading "Extract XPath"
cmd "webctl find --json \"more\" | jq -r '.matches[0].xpath'"

echo ""
echo "Verify: Extracts XPath from first match"
echo "Note: Will error if jq not installed (that's expected)"
read -p "Press Enter to continue..."

# REPL mode tests
title "REPL Mode Tests"

heading "Find in REPL (basic)"
echo "Switch to daemon terminal and execute:"
cmd "find \"example\""

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

heading "Find with regex in REPL"
echo "In REPL, try:"
cmd "find -E \"example|domain\""

echo ""
read -p "Press Enter when tested in REPL..."

heading "Find with case-sensitive in REPL"
echo "In REPL, try:"
cmd "find --case-sensitive \"Example\""

echo ""
read -p "Press Enter when tested in REPL..."

heading "Find with limit in REPL"
echo "In REPL, try:"
cmd "find --limit 5 \"more\""

echo ""
read -p "Press Enter when tested in REPL..."

# Test completed
title "Test Suite Complete"
echo "All find command tests finished"
echo ""
echo "Review checklist in docs/projects/p-049-testing-find.md"
echo "Document any issues discovered during testing"
