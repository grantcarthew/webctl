#!/bin/bash
# Title: webctl tab command tests

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
title "webctl tab Command Test Suite"
echo "Tests listing, switching, creating, and closing browser tabs"
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

heading "Navigate the active tab to a known page"
cmd "webctl navigate https://example.com --wait"

echo ""
read -p "Press Enter when page loaded..."

# List with single tab
title "List Tabs (Single Tab)"

heading "List all tabs (should show one)"
cmd "webctl tab"

echo ""
echo "Verify: Shows one tab"
echo "Verify: Session ID truncated to 8 chars"
echo "Verify: Title truncated to 40 chars if needed"
echo "Verify: URL shown"
echo "Verify: Active tab marked"
read -p "Press Enter to continue..."

# Last-tab close guard
title "Close Refused on Last Tab"

heading "Close the only tab (should refuse)"
cmd "webctl tab close"

echo ""
echo "Verify: Error 'cannot close the last tab; use webctl stop to shut down the browser'"
echo "Verify: Tab is NOT closed"
read -p "Press Enter to continue..."

# Open new tabs
title "Open New Tabs"

heading "tab new with no URL (about:blank)"
cmd "webctl tab new"

echo ""
echo "Verify: New blank tab opened and foregrounded"
echo "Verify: Returns OK (or JSON with id, url=about:blank)"
read -p "Press Enter to continue..."

heading "tab new with a URL (https auto-detection)"
cmd "webctl tab new github.com"

echo ""
echo "Verify: New tab opens at https://github.com"
echo "Verify: New tab becomes active and is foregrounded"
read -p "Press Enter to continue..."

heading "tab new with localhost (http auto-detection)"
cmd "webctl tab new localhost:9222"

echo ""
echo "Verify: Opens http://localhost:9222 (CDP page list)"
read -p "Press Enter to continue..."

heading "tab new with explicit protocol (preserved)"
cmd "webctl tab new http://example.com"

echo ""
echo "Verify: Uses http:// (not auto-rewritten)"
read -p "Press Enter to continue..."

# List with multiple tabs
title "List Tabs (Multiple Tabs)"

heading "List all tabs"
cmd "webctl tab"

echo ""
echo "Verify: Multiple tabs shown"
echo "Verify: Each tab has unique ID"
echo "Verify: Latest opened tab is the active one"
read -p "Press Enter to continue..."

# Switch by query
title "Switch Tabs"

heading "Switch by title substring (case-insensitive)"
cmd "webctl tab switch example"

echo ""
echo "Verify: Active tab switches to example.com tab"
echo "Verify: Browser foregrounds that tab"
read -p "Press Enter to continue..."

heading "Confirm active tab changed"
cmd "webctl tab"

echo ""
echo "Verify: example.com tab now marked active"
read -p "Press Enter to continue..."

heading "Switch by another title substring"
cmd "webctl tab switch github"

echo ""
echo "Verify: Switches to github.com tab and foregrounds it"
read -p "Press Enter to continue..."

heading "Switch by session ID prefix"
echo "Copy a session ID prefix (first few chars) from the list above, then:"
cmd "webctl tab switch SESSION_ID_PREFIX"

echo ""
echo "Verify: Switches to that tab"
read -p "Press Enter to continue..."

# Verify execution context follows active tab
title "Commands Execute in the Active Tab"

heading "Switch to example.com tab"
cmd "webctl tab switch example"
read -p "Press Enter to continue..."

heading "Read URL of active tab"
cmd "webctl eval \"window.location.href\""

echo ""
echo "Verify: Returns example.com URL"
read -p "Press Enter to continue..."

heading "Switch to github.com tab"
cmd "webctl tab switch github"
read -p "Press Enter to continue..."

heading "Read URL of active tab"
cmd "webctl eval \"window.location.href\""

echo ""
echo "Verify: Returns github.com URL"
read -p "Press Enter to continue..."

# Switch error cases
title "Switch Errors"

heading "Switch with no matching query"
cmd "webctl tab switch nonexistent-xyz"

echo ""
echo "Verify: Error 'no tab matches query: nonexistent-xyz'"
read -p "Press Enter to continue..."

heading "Switch with ambiguous query (matches multiple)"
echo "Choose a substring that matches more than one tab title (e.g. 'a' or 'com')"
cmd "webctl tab switch com"

echo ""
echo "Verify: Error 'ambiguous query'"
echo "Verify: Candidate list printed"
read -p "Press Enter to continue..."

# Close
title "Close Tabs"

heading "Close a tab by query"
cmd "webctl tab close github"

echo ""
echo "Verify: github.com tab closed"
echo "Verify: webctl tab no longer lists it"
read -p "Press Enter to continue..."

heading "Verify list reflects close immediately"
cmd "webctl tab"

echo ""
echo "Verify: Closed tab not present (close blocks until SessionManager updates)"
read -p "Press Enter to continue..."

heading "Close the active tab (no query)"
cmd "webctl tab close"

echo ""
echo "Verify: Active tab closed"
echo "Verify: Most-recently-opened remaining tab becomes active and is foregrounded"
read -p "Press Enter to continue..."

heading "List remaining tabs"
cmd "webctl tab"

echo ""
echo "Verify: New active tab marked"
read -p "Press Enter to continue..."

# Close error cases
title "Close Errors"

heading "Close with no matching query"
cmd "webctl tab close nonexistent-xyz"

echo ""
echo "Verify: Error 'no tab matches query: nonexistent-xyz'"
read -p "Press Enter to continue..."

heading "Close with ambiguous query"
echo "Open a couple of similar tabs first if needed, then run a substring matching multiple:"
cmd "webctl tab close com"

echo ""
echo "Verify: Error 'ambiguous query' with candidate list"
read -p "Press Enter to continue..."

# JSON output
title "JSON Output"

heading "List tabs as JSON"
cmd "webctl tab --json"

echo ""
echo "Verify: { ok, activeSession, sessions[] } structure"
echo "Verify: Each session has id, title, url, active fields"
read -p "Press Enter to continue..."

heading "tab switch with JSON output"
cmd "webctl tab switch example --json"

echo ""
echo "Verify: { ok: true, activeSession: <id> }"
read -p "Press Enter to continue..."

heading "tab new with JSON output"
cmd "webctl tab new about:blank --json"

echo ""
echo "Verify: { ok: true, id, url, title }"
read -p "Press Enter to continue..."

heading "tab close with JSON output"
cmd "webctl tab close --json"

echo ""
echo "Verify: { ok: true, activeSession: <id> }"
read -p "Press Enter to continue..."

heading "Ambiguous query in JSON form"
cmd "webctl tab switch com --json"

echo ""
echo "Verify: { ok: false, error, matches[] }"
read -p "Press Enter to continue..."

# REPL mode
title "REPL Mode"

heading "List tabs in REPL"
echo "Switch to the daemon terminal and run:"
cmd "tab"
read -p "Press Enter when tested in REPL..."

heading "Switch in REPL"
echo "In REPL:"
cmd "tab switch example"

echo ""
echo "Verify: REPL prompt updates to reflect new active session"
read -p "Press Enter when tested in REPL..."

heading "tab new in REPL"
cmd "tab new about:blank"
read -p "Press Enter when tested in REPL..."

heading "tab close in REPL"
cmd "tab close"
read -p "Press Enter when tested in REPL..."

# Cleanup
title "Cleanup"
echo "Tests complete - daemon can remain running or be stopped"

title "Test Suite Complete"
echo "All tab command tests finished"
echo ""
echo "Document any issues discovered during testing"
