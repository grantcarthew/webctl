#!/bin/bash
# Title: webctl target command tests

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
title "webctl target Command Test Suite"
echo "Project: P-050"
echo "Tests listing and switching between page sessions"
echo ""
echo "Prerequisites:"
echo "  - webctl must be built"
echo "  - xclip must be installed"
echo "  - Daemon running (or start one)"
echo ""
echo "Note: This command requires multi-tab/window support."
echo "If not yet fully implemented, test what's available."
echo ""
read -p "Press Enter to begin tests..."

# Setup
title "Setup"
echo "Ensure daemon is running"
cmd "webctl start"

echo ""
echo "Start daemon if not running, then continue"
read -p "Press Enter when daemon ready..."

heading "Navigate to initial page"
cmd "webctl navigate https://example.com --wait"

echo ""
read -p "Press Enter when page loaded..."

# List sessions with single session
title "List Sessions (Single Session)"

heading "List all sessions (should show one)"
cmd "webctl target"

echo ""
echo "Verify: Shows one session"
echo "Verify: Shows session ID (8 chars or truncated)"
echo "Verify: Shows title (truncated to 40 chars if needed)"
echo "Verify: Shows URL"
echo "Verify: Active session marked/indicated"
read -p "Press Enter to continue..."

heading "Target current session by ID"
echo "Copy the session ID from above output, then:"
echo "Note: Replace SESSION_ID with actual ID from previous command"
cmd "webctl target SESSION_ID"

echo ""
echo "Verify: Switches to session (even though it's already active)"
read -p "Press Enter to continue..."

# Multi-tab scenario (if supported)
title "Multi-Tab Scenario"

echo ""
echo "NOTE: Multi-tab creation may require manual browser interaction"
echo "or may not be fully implemented yet. Test what's available."
echo ""
echo "If multi-tab creation is not available:"
echo "  - Manually open new tab in Chrome at localhost:9222"
echo "  - Or skip multi-tab tests and document limitation"
echo ""
read -p "Press Enter to continue..."

heading "Attempt to create second session (manual or command)"
echo "Try one of these approaches:"
echo "1. Manually open new tab in the Chrome instance"
echo "2. Use webctl eval to open new window (if supported)"
echo ""
cmd "webctl eval \"window.open('https://github.com', '_blank')\""

echo ""
echo "If this opens a new tab/window, great!"
echo "If not, manually open a new tab in the browser"
read -p "Press Enter when second session ready (or skipping)..."

heading "List all sessions (should show multiple if multi-tab works)"
cmd "webctl target"

echo ""
echo "Verify: Shows multiple sessions (if multi-tab supported)"
echo "Verify: Each session has unique ID"
echo "Verify: Shows different titles/URLs"
echo "Verify: One session marked as active"
echo ""
echo "If only one session shown, multi-tab may not be supported yet"
read -p "Press Enter to continue..."

# Switch by session ID (if multi-tab works)
title "Switch by Session ID"

echo ""
echo "If multiple sessions are available:"
echo "Note the session IDs from the list above"
echo ""
read -p "Press Enter to continue..."

heading "Switch to first session by ID prefix"
echo "Replace SESSION_ID_PREFIX with actual prefix (e.g., '9A3E')"
cmd "webctl target SESSION_ID_PREFIX"

echo ""
echo "Verify: Switches to specified session"
echo "Verify: Subsequent commands execute in that session"
read -p "Press Enter to continue..."

heading "Verify active session changed"
cmd "webctl target"

echo ""
echo "Verify: Different session now marked as active"
read -p "Press Enter to continue..."

heading "Switch to second session by different prefix"
echo "Replace with another session ID prefix"
cmd "webctl target ANOTHER_PREFIX"

echo ""
echo "Verify: Switches to other session"
read -p "Press Enter to continue..."

# Switch by title substring
title "Switch by Title Substring"

heading "List sessions to see titles"
cmd "webctl target"

echo ""
echo "Note the titles of available sessions"
read -p "Press Enter to continue..."

heading "Switch by title substring (e.g., 'example')"
cmd "webctl target example"

echo ""
echo "Verify: Switches to session with 'example' in title"
echo "Verify: Case-insensitive matching"
read -p "Press Enter to continue..."

heading "Switch by different title substring (e.g., 'github')"
cmd "webctl target github"

echo ""
echo "Verify: Switches to session with 'github' in title"
read -p "Press Enter to continue..."

# Verify commands execute in correct session
title "Verify Commands Execute in Correct Session"

heading "List sessions"
cmd "webctl target"

echo ""
read -p "Press Enter to continue..."

heading "Switch to example.com session"
cmd "webctl target example"

echo ""
read -p "Press Enter to continue..."

heading "Get URL (should be example.com)"
cmd "webctl eval \"window.location.href\""

echo ""
echo "Verify: Returns example.com URL"
read -p "Press Enter to continue..."

heading "Switch to github.com session"
cmd "webctl target github"

echo ""
read -p "Press Enter to continue..."

heading "Get URL (should be github.com)"
cmd "webctl eval \"window.location.href\""

echo ""
echo "Verify: Returns github.com URL"
read -p "Press Enter to continue..."

# Error cases - multiple matches
title "Error Cases - Multiple Matches"

heading "Query that matches multiple sessions (e.g., 'com')"
cmd "webctl target com"

echo ""
echo "Verify: Error message about multiple matches"
echo "Verify: Shows list of matching sessions"
echo "Verify: Suggests more specific query"
read -p "Press Enter to continue..."

# Error cases - no matches
title "Error Cases - No Matches"

heading "Query with no matches"
cmd "webctl target nonexistent-xyz"

echo ""
echo "Verify: Error message - no matches found"
echo "Verify: Shows list of all sessions"
echo "Verify: Helpful guidance provided"
read -p "Press Enter to continue..."

# JSON output format
title "JSON Output Format"

heading "List sessions in JSON"
cmd "webctl target --json"

echo ""
echo "Verify: JSON with activeSession field"
echo "Verify: JSON with sessions array"
echo "Verify: Each session has id, title, url, active fields"
echo "Verify: Full IDs in JSON (not truncated)"
read -p "Press Enter to continue..."

heading "Switch session with JSON output"
cmd "webctl target example --json"

echo ""
echo "Verify: JSON response for successful switch"
read -p "Press Enter to continue..."

# Output formats
title "Output Format Tests"

heading "Default text output"
cmd "webctl target"

echo ""
echo "Verify: Formatted table/list"
echo "Verify: Truncated IDs (8 chars)"
echo "Verify: Truncated titles (40 chars)"
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl target --no-color"

echo ""
echo "Verify: No ANSI color codes"
read -p "Press Enter to continue..."

heading "Debug output"
cmd "webctl target --debug"

echo ""
echo "Verify: Debug information shown"
read -p "Press Enter to continue..."

# Edge cases
title "Edge Cases"

heading "Empty query string (should list all)"
cmd "webctl target \"\""

echo ""
echo "Verify: Lists all sessions (same as no argument)"
read -p "Press Enter to continue..."

heading "Target with very short ID prefix"
cmd "webctl target 9"

echo ""
echo "Verify: Works if unique, or shows multiple matches"
read -p "Press Enter to continue..."

# REPL mode tests
title "REPL Mode Tests"

heading "List sessions in REPL"
echo "Switch to daemon terminal and execute:"
cmd "target"

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

heading "Switch by ID in REPL"
echo "In REPL, try (replace with actual ID prefix):"
cmd "target SESSION_ID"

echo ""
read -p "Press Enter when tested in REPL..."

heading "Switch by title in REPL"
echo "In REPL, try:"
cmd "target example"

echo ""
read -p "Press Enter when tested in REPL..."

# Session persistence
title "Session Persistence"

heading "Create or switch between sessions"
echo "Verify sessions persist throughout daemon lifetime"
cmd "webctl target"

echo ""
echo "Verify: All previously created sessions still listed"
read -p "Press Enter to continue..."

heading "Navigate in current session"
cmd "webctl navigate https://example.com --wait"

echo ""
read -p "Press Enter to continue..."

heading "Verify session persists after navigation"
cmd "webctl target"

echo ""
echo "Verify: Same session, updated title/URL"
read -p "Press Enter to continue..."

# Test completed
title "Test Suite Complete"
echo "All target command tests finished"
echo ""
echo "Review checklist in docs/projects/p-050-testing-target.md"
echo "Document any issues discovered during testing"
echo ""
echo "Note: If multi-tab functionality is limited or not implemented,"
echo "document the current capabilities and limitations."
