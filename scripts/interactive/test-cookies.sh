#!/bin/bash
# Title: webctl cookies command tests

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
title "webctl cookies Command Test Suite"
echo "Project: P-038"
echo "Tests cookie extraction and manipulation (stdout/save/set/delete)"
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

heading "Navigate to site that sets cookies"
echo "We'll navigate to GitHub which sets cookies"
cmd "webctl navigate https://github.com --wait"

echo ""
echo "Wait for page to load (cookies will be set)"
read -p "Press Enter when page loaded..."

# Default mode tests
title "Default Mode (Output to Stdout)"

heading "Show all cookies"
cmd "webctl cookies"

echo ""
echo "Verify: Formatted cookies to stdout"
echo "Verify: Shows name, value, domain, path, expiry, flags"
echo "Verify: No file created"
read -p "Press Enter to continue..."

heading "Show only GitHub cookies"
cmd "webctl cookies --domain \".github.com\""

echo ""
echo "Verify: Only GitHub domain cookies shown"
read -p "Press Enter to continue..."

heading "Show cookies by name"
cmd "webctl cookies --name \"logged_in\""

echo ""
echo "Verify: Only 'logged_in' cookie shown (if exists)"
read -p "Press Enter to continue..."

heading "Show with text search"
cmd "webctl cookies --find \"session\""

echo ""
echo "Verify: Cookies containing 'session' shown"
read -p "Press Enter to continue..."

# Save mode tests
title "Save Mode (File Output)"

heading "Save to temp (no path)"
cmd "webctl cookies save"

echo ""
echo "Verify: File saved to /tmp/webctl-cookies/"
echo "Verify: Auto-generated filename (YY-MM-DD-HHMMSS-cookies.json)"
echo "Verify: JSON response shows file path"
read -p "Press Enter to continue..."

heading "Save to custom file"
cmd "webctl cookies save ./cookies.json"

echo ""
echo "Verify: File saved to ./cookies.json"
read -p "Press Enter to continue..."

heading "Save to directory with auto-filename (trailing slash = directory)"
cmd "webctl cookies save ./output/"

echo ""
echo "Verify: File saved to ./output/ with auto-generated name"
echo "Note: Trailing slash (/) is REQUIRED for directory behavior"
echo "      Without slash, it would create a file named 'output'"
read -p "Press Enter to continue..."

heading "Save filtered cookies"
cmd "webctl cookies save ./auth-cookies.json --find \"auth\""

echo ""
echo "Verify: Cookies containing 'auth' saved"
read -p "Press Enter to continue..."

# Domain filter tests
title "Domain Filter Tests"

heading "Filter by exact domain"
cmd "webctl cookies --domain \"github.com\""

echo ""
echo "Verify: Cookies for github.com shown"
read -p "Press Enter to continue..."

heading "Filter by domain with leading dot"
cmd "webctl cookies --domain \".github.com\""

echo ""
echo "Verify: Cookies for github.com and subdomains shown"
read -p "Press Enter to continue..."

# Name filter tests
title "Name Filter Tests"

heading "Filter by exact name"
cmd "webctl cookies --name \"logged_in\""

echo ""
echo "Verify: Exact match for 'logged_in' cookie"
read -p "Press Enter to continue..."

heading "Name combined with domain"
cmd "webctl cookies --name \"logged_in\" --domain \".github.com\""

echo ""
echo "Verify: Named cookie from specific domain"
read -p "Press Enter to continue..."

# Find flag tests
title "Find Flag Tests"

heading "Find in cookie name"
cmd "webctl cookies --find \"session\""

echo ""
echo "Verify: Case-insensitive search in names and values"
read -p "Press Enter to continue..."

heading "Find with no matches (should error)"
cmd "webctl cookies --find \"ThisTextDoesNotExist123\""

echo ""
echo "Verify: Error message about no matches"
read -p "Press Enter to continue..."

heading "Find combined with domain"
cmd "webctl cookies --find \"git\" --domain \".github.com\""

echo ""
echo "Verify: GitHub cookies containing 'git'"
read -p "Press Enter to continue..."

# Raw flag tests
title "Raw Flag Tests"

heading "Raw output (JSON format)"
cmd "webctl cookies --raw"

echo ""
echo "Verify: Raw JSON output instead of formatted text"
read -p "Press Enter to continue..."

heading "Raw with filters"
cmd "webctl cookies --raw --domain \".github.com\""

echo ""
echo "Verify: Raw JSON with filtered cookies"
read -p "Press Enter to continue..."

# Set subcommand tests
title "Set Subcommand - Session Cookies"

heading "Set basic session cookie"
cmd "webctl cookies set test_session abc123"

echo ""
echo "Verify: Cookie created with default domain and path"
read -p "Press Enter to continue..."

heading "Verify cookie was set"
cmd "webctl cookies --name test_session"

echo ""
echo "Verify: test_session cookie shown with value abc123"
read -p "Press Enter to continue..."

heading "Set another session cookie"
cmd "webctl cookies set auth_token token456"

echo ""
echo "Verify: Cookie created"
read -p "Press Enter to continue..."

# Set subcommand - Persistent cookies
title "Set Subcommand - Persistent Cookies"

heading "Set cookie with 1 hour expiry"
cmd "webctl cookies set remember_me yes --max-age 3600"

echo ""
echo "Verify: Cookie created with expiry set"
read -p "Press Enter to continue..."

heading "Verify persistent cookie"
cmd "webctl cookies --name remember_me"

echo ""
echo "Verify: Cookie shows expiry time"
read -p "Press Enter to continue..."

heading "Set cookie with 24 hour expiry"
cmd "webctl cookies set long_lived xyz --max-age 86400"

echo ""
echo "Verify: Cookie created with 24h expiry"
read -p "Press Enter to continue..."

# Set subcommand - Secure cookies
title "Set Subcommand - Secure Cookies"

heading "Set secure cookie"
cmd "webctl cookies set secure_test abc --secure"

echo ""
echo "Verify: Cookie created with Secure flag"
read -p "Press Enter to continue..."

heading "Set HttpOnly cookie"
cmd "webctl cookies set httponly_test abc --httponly"

echo ""
echo "Verify: Cookie created with HttpOnly flag"
read -p "Press Enter to continue..."

heading "Set both Secure and HttpOnly"
cmd "webctl cookies set both_flags abc --secure --httponly"

echo ""
echo "Verify: Cookie created with both flags"
read -p "Press Enter to continue..."

heading "Verify secure flags"
cmd "webctl cookies --find \"_test\""

echo ""
echo "Verify: Cookies show Secure and HttpOnly flags"
read -p "Press Enter to continue..."

# Set subcommand - Domain and Path
title "Set Subcommand - Domain and Path"

heading "Set cookie with custom domain"
cmd "webctl cookies set custom_domain value --domain github.com"

echo ""
echo "Verify: Cookie created for github.com domain"
read -p "Press Enter to continue..."

heading "Set cookie with custom path"
cmd "webctl cookies set path_test value --path /api"

echo ""
echo "Verify: Cookie created with /api path"
read -p "Press Enter to continue..."

heading "Set cookie with domain and path"
cmd "webctl cookies set full_custom value --domain github.com --path /admin"

echo ""
echo "Verify: Cookie created with custom domain and path"
read -p "Press Enter to continue..."

# Set subcommand - SameSite
title "Set Subcommand - SameSite"

heading "Set SameSite Strict cookie"
cmd "webctl cookies set samesite_strict value --samesite Strict"

echo ""
echo "Verify: Cookie created with SameSite=Strict"
read -p "Press Enter to continue..."

heading "Set SameSite Lax cookie"
cmd "webctl cookies set samesite_lax value --samesite Lax"

echo ""
echo "Verify: Cookie created with SameSite=Lax"
read -p "Press Enter to continue..."

heading "Set SameSite None cookie (requires Secure)"
cmd "webctl cookies set samesite_none value --samesite None --secure"

echo ""
echo "Verify: Cookie created with SameSite=None and Secure"
read -p "Press Enter to continue..."

heading "Verify SameSite cookies"
cmd "webctl cookies --find \"samesite_\""

echo ""
echo "Verify: Cookies show SameSite attribute"
read -p "Press Enter to continue..."

# Set subcommand - Full attributes
title "Set Subcommand - All Attributes"

heading "Set cookie with all attributes"
cmd "webctl cookies set full_featured abc --domain github.com --path /api --secure --httponly --max-age 86400 --samesite Strict"

echo ""
echo "Verify: Cookie created with all attributes"
read -p "Press Enter to continue..."

heading "Verify full featured cookie"
cmd "webctl cookies --name full_featured"

echo ""
echo "Verify: All attributes set correctly"
read -p "Press Enter to continue..."

# Delete subcommand tests
title "Delete Subcommand - Unambiguous"

heading "Delete test_session cookie"
cmd "webctl cookies delete test_session"

echo ""
echo "Verify: Cookie deleted successfully"
read -p "Press Enter to continue..."

heading "Verify deletion"
cmd "webctl cookies --name test_session"

echo ""
echo "Verify: Cookie no longer exists (error or empty)"
read -p "Press Enter to continue..."

heading "Delete auth_token cookie"
cmd "webctl cookies delete auth_token"

echo ""
echo "Verify: Cookie deleted"
read -p "Press Enter to continue..."

heading "Delete non-existent cookie (should succeed - idempotent)"
cmd "webctl cookies delete nonexistent_cookie"

echo ""
echo "Verify: Success even though cookie doesn't exist"
read -p "Press Enter to continue..."

# Delete subcommand - With domain
title "Delete Subcommand - Domain Specific"

heading "Delete cookie with domain"
cmd "webctl cookies delete custom_domain --domain github.com"

echo ""
echo "Verify: Domain-specific deletion"
read -p "Press Enter to continue..."

heading "Delete path-specific cookie"
cmd "webctl cookies delete path_test"

echo ""
echo "Verify: Cookie deleted (may need path if ambiguous)"
read -p "Press Enter to continue..."

# Delete subcommand - Ambiguous (if applicable)
title "Delete Subcommand - Ambiguous Cases"

heading "Create cookies with same name, different domains"
cmd "webctl cookies set duplicate_name value1 --domain github.com"
cmd "webctl cookies set duplicate_name value2 --domain .github.com"

echo ""
echo "Two cookies with same name created"
read -p "Press Enter to continue..."

heading "Try to delete ambiguous name (should error)"
cmd "webctl cookies delete duplicate_name"

echo ""
echo "Verify: Error message listing multiple matches"
echo "Verify: Error suggests using --domain to disambiguate"
read -p "Press Enter to continue..."

heading "Delete with domain to disambiguate"
cmd "webctl cookies delete duplicate_name --domain github.com"

echo ""
echo "Verify: Specific cookie deleted"
read -p "Press Enter to continue..."

heading "Delete remaining duplicate"
cmd "webctl cookies delete duplicate_name --domain .github.com"

echo ""
echo "Verify: Other cookie deleted"
read -p "Press Enter to continue..."

# Output format tests
title "Output Format Tests"

heading "JSON output"
cmd "webctl cookies --json"

echo ""
echo "Verify: JSON formatted output"
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl cookies --no-color"

echo ""
echo "Verify: No ANSI color codes"
read -p "Press Enter to continue..."

heading "Debug verbose output"
cmd "webctl cookies --debug"

echo ""
echo "Verify: Debug logging information"
read -p "Press Enter to continue..."

# REPL mode tests
title "REPL Mode Tests"

heading "Test cookies in REPL"
echo "Switch to daemon terminal and execute:"
cmd "cookies"

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

heading "Test cookies set in REPL"
echo "In REPL, try:"
cmd "cookies set repl_cookie repl_value"

echo ""
echo "Should create cookie from REPL"
read -p "Press Enter when tested in REPL..."

heading "Test cookies delete in REPL"
echo "In REPL, try:"
cmd "cookies delete repl_cookie"

echo ""
echo "Should delete cookie from REPL"
read -p "Press Enter when tested in REPL..."

# Advanced scenarios
title "Advanced Scenarios"

heading "Navigate to different site"
cmd "webctl navigate https://example.com --wait"

echo ""
echo "Wait for example.com to load"
read -p "Press Enter when loaded..."

heading "Set cookies on example.com"
cmd "webctl cookies set example_cookie test_value"

echo ""
echo "Cookie set on example.com"
read -p "Press Enter to continue..."

heading "Show all cookies from all domains"
cmd "webctl cookies"

echo ""
echo "Verify: Cookies from both github.com and example.com"
read -p "Press Enter to continue..."

heading "Filter by domain"
cmd "webctl cookies --domain example.com"

echo ""
echo "Verify: Only example.com cookies shown"
read -p "Press Enter to continue..."

heading "Save cookies by domain"
cmd "webctl cookies save ./example-cookies.json --domain example.com"

echo ""
echo "Verify: Only example.com cookies saved"
read -p "Press Enter to continue..."

# Cleanup test cookies
title "Cleanup Test Cookies"

heading "Delete test cookies"
echo "Cleaning up test cookies we created..."
cmd "webctl cookies delete remember_me"
cmd "webctl cookies delete long_lived"
cmd "webctl cookies delete secure_test"
cmd "webctl cookies delete httponly_test"
cmd "webctl cookies delete both_flags"

echo ""
echo "Test cookies deleted"
read -p "Press Enter to continue..."

heading "Delete more test cookies"
cmd "webctl cookies delete full_custom"
cmd "webctl cookies delete samesite_strict"
cmd "webctl cookies delete samesite_lax"
cmd "webctl cookies delete samesite_none"
cmd "webctl cookies delete full_featured"

echo ""
echo "More test cookies deleted"
read -p "Press Enter to continue..."

heading "Delete remaining test cookies"
cmd "webctl cookies delete example_cookie"

echo ""
echo "All test cookies cleaned up"
read -p "Press Enter to continue..."

# Cleanup files
title "Cleanup Files"
echo "Clean up test files if desired:"
cmd "rm -f ./cookies.json ./auth-cookies.json ./output/*.json ./example-cookies.json"

echo ""
echo "Remove test files when ready"
read -p "Press Enter to finish..."

title "Test Suite Complete"
echo "All cookies command tests finished"
echo ""
echo "Review checklist in docs/projects/p-038-testing-cookies.md"
echo "Document any issues discovered during testing"
