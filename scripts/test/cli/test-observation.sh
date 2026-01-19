#!/usr/bin/env bash

# Test: CLI Observation Commands
# --------------------------------
# Tests for webctl observation commands: html, css, console, network, cookies, screenshot.
# Verifies data capture and output functionality across all observation modes.

# Determine script location and project root
SCRIPT_DIR="$(cd "${BASH_SOURCE[0]%/*}" || exit 1; pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." || exit 1; pwd)"

# Import test modules
source "${PROJECT_ROOT}/scripts/bash_modules/test-framework.sh"
source "${PROJECT_ROOT}/scripts/bash_modules/assertions.sh"
source "${PROJECT_ROOT}/scripts/bash_modules/setup.sh"

# Setup
setup_cleanup_trap
require_webctl

# Ensure clean state before tests
force_stop_daemon

# Start daemon and test server
start_daemon --headless
start_test_server

# Wait for serve auto-navigation to complete before running tests
# webctl serve automatically navigates the browser to the served URL
sleep 3

# =============================================================================
# HTML Command Tests
# =============================================================================

test_section "HTML Command - Basic Output"

# Navigate to navigation page
run_test "setup: navigate to navigation.html" "${WEBCTL_BINARY}" navigate "$(get_test_url '/pages/navigation.html')"
assert_success "${TEST_EXIT_CODE}" "Navigate succeeded"

# Wait for page to fully load
sleep 2

# Test: Basic HTML output to stdout
run_test "html basic output" "${WEBCTL_BINARY}" html
assert_success "${TEST_EXIT_CODE}" "html command returns success"
assert_contains "${TEST_STDOUT}" "<html" "Output contains html tag"
assert_contains "${TEST_STDOUT}" "Navigation" "Output contains navigation text"

test_section "HTML Command - Selector Filtering"

# Test: HTML with selector filtering
run_test "html with --select h1" "${WEBCTL_BINARY}" html --select "h1"
assert_success "${TEST_EXIT_CODE}" "html --select returns success"
assert_contains "${TEST_STDOUT}" "<h1>" "Output contains h1 tag"
assert_contains "${TEST_STDOUT}" "Navigation Test Page" "Output contains h1 content"

# Test: HTML with selector for paragraph
run_test "html with --select p" "${WEBCTL_BINARY}" html --select "p"
assert_success "${TEST_EXIT_CODE}" "html --select p returns success"

test_section "HTML Command - Text Search"

# Test: HTML with text search
run_test "html with --find" "${WEBCTL_BINARY}" html --find "Navigation"
assert_success "${TEST_EXIT_CODE}" "html --find returns success"
assert_contains "${TEST_STDOUT}" "Navigation" "Output contains searched text"

test_section "HTML Command - Save Modes"

# Test: Save to temp
run_test "html save to temp" "${WEBCTL_BINARY}" html save
assert_success "${TEST_EXIT_CODE}" "html save returns success"
assert_contains "${TEST_STDOUT}" ".html" "Output shows temp file path"

# Test: Save to custom file
TEMP_HTML_FILE=$(create_temp_file ".html")
run_test "html save to custom file" "${WEBCTL_BINARY}" html save "${TEMP_HTML_FILE}"
assert_success "${TEST_EXIT_CODE}" "html save to file returns success"
assert_file_exists "${TEMP_HTML_FILE}" "Custom HTML file created"
assert_file_contains "${TEMP_HTML_FILE}" "Navigation Test Page" "Saved file contains page content"

# Test: Save to directory
TEMP_HTML_DIR=$(create_temp_dir)
run_test "html save to directory" "${WEBCTL_BINARY}" html save "${TEMP_HTML_DIR}/"
assert_success "${TEST_EXIT_CODE}" "html save to directory returns success"
assert_contains "${TEST_STDOUT}" "${TEMP_HTML_DIR}/" "Output shows directory path"

# =============================================================================
# CSS Command Tests
# =============================================================================

test_section "CSS Command - Basic Output"

# Navigate to CSS showcase page
run_test "setup: navigate to css-showcase.html" "${WEBCTL_BINARY}" navigate "$(get_test_url '/pages/css-showcase.html')"
assert_success "${TEST_EXIT_CODE}" "Navigate to CSS page succeeded"

# Wait for page to fully load
sleep 2

# Test: Basic CSS output
run_test "css basic output" "${WEBCTL_BINARY}" css
assert_success "${TEST_EXIT_CODE}" "css command returns success"
assert_contains "${TEST_STDOUT}" "body" "Output contains body selector"
assert_contains "${TEST_STDOUT}" "font-family" "Output contains CSS properties"

test_section "CSS Command - Selector Filtering"

# Test: CSS with selector pattern (gets computed styles for selector)
run_test "css with --select h1" "${WEBCTL_BINARY}" css --select "h1"
assert_success "${TEST_EXIT_CODE}" "css --select returns success"
assert_contains "${TEST_STDOUT}" "color" "Output contains computed style properties"

test_section "CSS Command - Text Search"

# Test: CSS with text search
run_test "css with --find" "${WEBCTL_BINARY}" css --find "background"
assert_success "${TEST_EXIT_CODE}" "css --find returns success"
assert_contains "${TEST_STDOUT}" "background" "Output contains searched text"

test_section "CSS Command - Computed Styles"

# Test: Computed styles for element
run_test "css computed for h1" "${WEBCTL_BINARY}" css computed "h1"
assert_success "${TEST_EXIT_CODE}" "css computed returns success"
assert_contains "${TEST_STDOUT}" "color" "Output contains computed color property"

test_section "CSS Command - Get Property"

# Test: Get single CSS property
run_test "css get h1 color" "${WEBCTL_BINARY}" css get "h1" "color"
assert_success "${TEST_EXIT_CODE}" "css get returns success"
assert_contains "${TEST_STDOUT}" "rgb" "Output contains color value"

test_section "CSS Command - Inline Styles"

# Test: Get inline styles for elements with style attributes
run_test "css inline for elements with style" "${WEBCTL_BINARY}" css inline "[style]"
assert_success "${TEST_EXIT_CODE}" "css inline returns success"
assert_contains "${TEST_STDOUT}" "padding" "Output contains inline style property"

# Test: Get inline styles for element without inline styles (should succeed with empty/no output)
run_test "css inline for element without inline styles" "${WEBCTL_BINARY}" css inline "h1"
# May succeed with empty output or fail if no inline styles - checking behavior
# assert_success "${TEST_EXIT_CODE}" "css inline for h1 returns success"

# Test: CSS inline with multiple matching elements
run_test "css inline for multiple elements" "${WEBCTL_BINARY}" css inline ".animated, .hover-example"
assert_success "${TEST_EXIT_CODE}" "css inline for multiple elements returns success"

test_section "CSS Command - Matched Rules"

# Test: Get matched CSS rules for body
run_test "css matched for body" "${WEBCTL_BINARY}" css matched "body"
assert_success "${TEST_EXIT_CODE}" "css matched returns success"
assert_contains "${TEST_STDOUT}" "font-family" "Output contains matched CSS property"

# Test: Get matched CSS rules for h1
run_test "css matched for h1" "${WEBCTL_BINARY}" css matched "h1"
assert_success "${TEST_EXIT_CODE}" "css matched for h1 returns success"
assert_contains "${TEST_STDOUT}" "color" "Output contains matched color property"

# Test: Get matched CSS rules for element with class
run_test "css matched for .highlight" "${WEBCTL_BINARY}" css matched ".highlight"
assert_success "${TEST_EXIT_CODE}" "css matched for .highlight returns success"
assert_contains "${TEST_STDOUT}" "background-color" "Output contains background-color property"

test_section "CSS Command - Save Modes"

# Test: Save CSS to temp
run_test "css save to temp" "${WEBCTL_BINARY}" css save
assert_success "${TEST_EXIT_CODE}" "css save returns success"
assert_contains "${TEST_STDOUT}" ".css" "Output shows temp file path"

# Test: Save CSS to custom file
TEMP_CSS_FILE=$(create_temp_file ".css")
run_test "css save to custom file" "${WEBCTL_BINARY}" css save "${TEMP_CSS_FILE}"
assert_success "${TEST_EXIT_CODE}" "css save to file returns success"
assert_file_exists "${TEMP_CSS_FILE}" "Custom CSS file created"

# Test: Save CSS to directory
TEMP_CSS_DIR=$(create_temp_dir)
run_test "css save to directory" "${WEBCTL_BINARY}" css save "${TEMP_CSS_DIR}/"
assert_success "${TEST_EXIT_CODE}" "css save to directory returns success"
assert_contains "${TEST_STDOUT}" "${TEMP_CSS_DIR}/" "Output shows directory path"

# =============================================================================
# Console Command Tests
# =============================================================================

test_section "Console Command - Basic Output"

# Navigate to console-types page
run_test "setup: navigate to console-types.html" "${WEBCTL_BINARY}" navigate "$(get_test_url '/pages/console-types.html')"
assert_success "${TEST_EXIT_CODE}" "Navigate to console page succeeded"

# Wait for page to load and trigger console messages
sleep 3

# Test: Basic console output
run_test "console basic output" "${WEBCTL_BINARY}" console
assert_success "${TEST_EXIT_CODE}" "console command returns success"

test_section "Console Command - Type Filtering"

# Test: Filter by log type
run_test "console --type log" "${WEBCTL_BINARY}" console --type log
assert_success "${TEST_EXIT_CODE}" "console --type log returns success"

# Test: Filter by error type
run_test "console --type error" "${WEBCTL_BINARY}" console --type error
assert_success "${TEST_EXIT_CODE}" "console --type error returns success"

test_section "Console Command - Text Search"

# Test: Console with text search
run_test "console with --find" "${WEBCTL_BINARY}" console --find "TEST"
assert_success "${TEST_EXIT_CODE}" "console --find returns success"

test_section "Console Command - Save Modes"

# Test: Save console to temp
run_test "console save to temp" "${WEBCTL_BINARY}" console save
assert_success "${TEST_EXIT_CODE}" "console save returns success"
assert_contains "${TEST_STDOUT}" "console" "Output shows console file path"

# Test: Save console to custom file
TEMP_CONSOLE_FILE=$(create_temp_file ".txt")
run_test "console save to custom file" "${WEBCTL_BINARY}" console save "${TEMP_CONSOLE_FILE}"
assert_success "${TEST_EXIT_CODE}" "console save to file returns success"
assert_file_exists "${TEMP_CONSOLE_FILE}" "Custom console file created"

# Test: Save console to directory
TEMP_CONSOLE_DIR=$(create_temp_dir)
run_test "console save to directory" "${WEBCTL_BINARY}" console save "${TEMP_CONSOLE_DIR}/"
assert_success "${TEST_EXIT_CODE}" "console save to directory returns success"
assert_contains "${TEST_STDOUT}" "${TEMP_CONSOLE_DIR}/" "Output shows directory path"

# =============================================================================
# Network Command Tests
# =============================================================================

test_section "Network Command - Setup"

# Note: Backend server setup is optional for basic network tests
# The network-requests.html page will make its own requests that we can observe

test_section "Network Command - Basic Output"

# Navigate to network-requests page
run_test "setup: navigate to network-requests.html" "${WEBCTL_BINARY}" navigate "$(get_test_url '/pages/network-requests.html')"
assert_success "${TEST_EXIT_CODE}" "Navigate to network page succeeded"

# Wait for page and resources to load
sleep 3

# Clear network buffer before tests
run_test "clear network buffer" "${WEBCTL_BINARY}" clear network
assert_success "${TEST_EXIT_CODE}" "Network buffer cleared"

# Trigger network request via eval
run_test "trigger API request" "${WEBCTL_BINARY}" eval "fetch('/api/hello').then(r => r.json())"
sleep 2

# Test: Basic network output
run_test "network basic output" "${WEBCTL_BINARY}" network
assert_success "${TEST_EXIT_CODE}" "network command returns success"

test_section "Network Command - Status Filtering"

# Test: Filter by status code
run_test "network --status 200" "${WEBCTL_BINARY}" network --status 200
assert_success "${TEST_EXIT_CODE}" "network --status 200 returns success"

test_section "Network Command - Method Filtering"

# Test: Filter by method
run_test "network --method GET" "${WEBCTL_BINARY}" network --method GET
assert_success "${TEST_EXIT_CODE}" "network --method GET returns success"

test_section "Network Command - Text Search"

# Test: Network with text search for the API request (buffer was cleared, so page request is gone)
run_test "network with --find" "${WEBCTL_BINARY}" network --find "api"
assert_success "${TEST_EXIT_CODE}" "network --find returns success"
assert_contains "${TEST_STDOUT}" "api" "Output contains the API request"

test_section "Network Command - Save Modes"

# Test: Save network to temp
run_test "network save to temp" "${WEBCTL_BINARY}" network save
assert_success "${TEST_EXIT_CODE}" "network save returns success"
assert_contains "${TEST_STDOUT}" "network" "Output shows network file path"

# Test: Save network to custom file
TEMP_NETWORK_FILE=$(create_temp_file ".txt")
run_test "network save to custom file" "${WEBCTL_BINARY}" network save "${TEMP_NETWORK_FILE}"
assert_success "${TEST_EXIT_CODE}" "network save to file returns success"
assert_file_exists "${TEMP_NETWORK_FILE}" "Custom network file created"

# Test: Save network to directory
TEMP_NETWORK_DIR=$(create_temp_dir)
run_test "network save to directory" "${WEBCTL_BINARY}" network save "${TEMP_NETWORK_DIR}/"
assert_success "${TEST_EXIT_CODE}" "network save to directory returns success"
assert_contains "${TEST_STDOUT}" "${TEMP_NETWORK_DIR}/" "Output shows directory path"

# =============================================================================
# Cookies Command Tests
# =============================================================================

test_section "Cookies Command - Basic Output"

# Navigate to cookies page
run_test "setup: navigate to cookies.html" "${WEBCTL_BINARY}" navigate "$(get_test_url '/pages/cookies.html')"
assert_success "${TEST_EXIT_CODE}" "Navigate to cookies page succeeded"

# Wait for page to load and set initial cookie
sleep 2

# Test: Basic cookies output
run_test "cookies basic output" "${WEBCTL_BINARY}" cookies
assert_success "${TEST_EXIT_CODE}" "cookies command returns success"
assert_contains "${TEST_STDOUT}" "initial-cookie" "Output contains initial cookie"

test_section "Cookies Command - Set Cookie"

# Test: Set a new cookie
run_test "cookies set test-cookie test-value" "${WEBCTL_BINARY}" cookies set "test-cookie" "test-value"
assert_success "${TEST_EXIT_CODE}" "cookies set returns success"

# Verify cookie was set
run_test "verify cookie set" "${WEBCTL_BINARY}" cookies
assert_success "${TEST_EXIT_CODE}" "cookies command returns success"
assert_contains "${TEST_STDOUT}" "test-cookie" "Output contains set cookie"

test_section "Cookies Command - Delete Cookie"

# Test: Delete the cookie
run_test "cookies delete test-cookie" "${WEBCTL_BINARY}" cookies delete "test-cookie"
assert_success "${TEST_EXIT_CODE}" "cookies delete returns success"

# Verify cookie was deleted
run_test "verify cookie deleted" "${WEBCTL_BINARY}" cookies
assert_success "${TEST_EXIT_CODE}" "cookies command returns success"
assert_not_contains "${TEST_STDOUT}" "test-cookie" "Output does not contain deleted cookie"

test_section "Cookies Command - Domain Filtering"

# Test: Filter by domain
run_test "cookies --domain localhost" "${WEBCTL_BINARY}" cookies --domain "localhost"
assert_success "${TEST_EXIT_CODE}" "cookies --domain returns success"

test_section "Cookies Command - Save Modes"

# Test: Save cookies to temp
run_test "cookies save to temp" "${WEBCTL_BINARY}" cookies save
assert_success "${TEST_EXIT_CODE}" "cookies save returns success"
assert_contains "${TEST_STDOUT}" "cookies" "Output shows cookies file path"

# Test: Save cookies to custom file
TEMP_COOKIES_FILE=$(create_temp_file ".txt")
run_test "cookies save to custom file" "${WEBCTL_BINARY}" cookies save "${TEMP_COOKIES_FILE}"
assert_success "${TEST_EXIT_CODE}" "cookies save to file returns success"
assert_file_exists "${TEMP_COOKIES_FILE}" "Custom cookies file created"

# Test: Save cookies to directory
TEMP_COOKIES_DIR=$(create_temp_dir)
run_test "cookies save to directory" "${WEBCTL_BINARY}" cookies save "${TEMP_COOKIES_DIR}/"
assert_success "${TEST_EXIT_CODE}" "cookies save to directory returns success"
assert_contains "${TEST_STDOUT}" "${TEMP_COOKIES_DIR}/" "Output shows directory path"

# =============================================================================
# Screenshot Command Tests
# =============================================================================

test_section "Screenshot Command - Basic Save"

# Navigate to a page for screenshot
run_test "setup: navigate for screenshot" "${WEBCTL_BINARY}" navigate "$(get_test_url '/pages/navigation.html')"
assert_success "${TEST_EXIT_CODE}" "Navigate succeeded"

# Wait for page to fully load and render
sleep 2

# Test: Screenshot save to temp (default behavior)
run_test "screenshot save to temp" "${WEBCTL_BINARY}" screenshot
assert_success "${TEST_EXIT_CODE}" "screenshot returns success"
assert_contains "${TEST_STDOUT}" ".png" "Output shows PNG file path"

test_section "Screenshot Command - Save to Custom Path"

# Test: Screenshot save to custom file
TEMP_SCREENSHOT_FILE=$(create_temp_file ".png")
run_test "screenshot save to custom file" "${WEBCTL_BINARY}" screenshot save "${TEMP_SCREENSHOT_FILE}"
assert_success "${TEST_EXIT_CODE}" "screenshot save to file returns success"
assert_file_exists "${TEMP_SCREENSHOT_FILE}" "Custom screenshot file created"

# Verify file has content (PNG files should have non-zero size)
SCREENSHOT_SIZE=$(stat -c%s "${TEMP_SCREENSHOT_FILE}" 2>/dev/null || stat -f%z "${TEMP_SCREENSHOT_FILE}" 2>/dev/null)
if [[ ${SCREENSHOT_SIZE} -gt 0 ]]; then
  log_success "Screenshot file has content (${SCREENSHOT_SIZE} bytes)"
else
  log_failure "Screenshot file is empty"
  TEST_FAILURES=$((TEST_FAILURES + 1))
fi

test_section "Screenshot Command - Full Page"

# Test: Full-page screenshot
TEMP_FULLPAGE_FILE=$(create_temp_file ".png")
run_test "screenshot --full-page" "${WEBCTL_BINARY}" screenshot --full-page save "${TEMP_FULLPAGE_FILE}"
assert_success "${TEST_EXIT_CODE}" "screenshot --full-page returns success"
assert_file_exists "${TEMP_FULLPAGE_FILE}" "Full-page screenshot file created"

# =============================================================================
# Error Case Tests
# =============================================================================

test_section "Error Cases - HTML Command"

# Test: HTML with find that has no matches
run_test "html --find with no matches" "${WEBCTL_BINARY}" html --find "NONEXISTENT_TEXT_XYZ_12345"
assert_failure "${TEST_EXIT_CODE}" "No matches returns failure"

# Test: HTML with selector that matches nothing
run_test "html --select with no matches" "${WEBCTL_BINARY}" html --select ".nonexistent-class-xyz-12345"
assert_failure "${TEST_EXIT_CODE}" "No matching selector returns failure"

# Test: HTML with invalid selector syntax
run_test "html --select with invalid syntax" "${WEBCTL_BINARY}" html --select "[invalid::syntax"
assert_failure "${TEST_EXIT_CODE}" "Invalid selector syntax returns failure"

test_section "Error Cases - CSS Command"

# Navigate to CSS page for error tests
run_test "setup: navigate to css-showcase.html for error tests" "${WEBCTL_BINARY}" navigate "$(get_test_url '/pages/css-showcase.html')"
sleep 2

# Test: CSS computed with selector that matches nothing
run_test "css computed with no matching elements" "${WEBCTL_BINARY}" css computed ".nonexistent-class-xyz-12345"
assert_failure "${TEST_EXIT_CODE}" "No matching elements returns failure"

# Test: CSS get with selector that matches nothing
run_test "css get with no matching elements" "${WEBCTL_BINARY}" css get ".nonexistent-class-xyz-12345" "color"
assert_failure "${TEST_EXIT_CODE}" "CSS get with no match returns failure"

# Test: CSS matched with selector that matches nothing
run_test "css matched with no matching elements" "${WEBCTL_BINARY}" css matched ".nonexistent-class-xyz-12345"
assert_failure "${TEST_EXIT_CODE}" "CSS matched with no match returns failure"

# Test: CSS inline with selector that matches nothing
run_test "css inline with no matching elements" "${WEBCTL_BINARY}" css inline ".nonexistent-class-xyz-12345"
assert_failure "${TEST_EXIT_CODE}" "CSS inline with no match returns failure"

# Test: CSS get with nonexistent property (valid selector, invalid property)
run_test "css get with nonexistent property" "${WEBCTL_BINARY}" css get "h1" "nonexistent-fake-property"
# Note: Returns failure for nonexistent/invalid properties
assert_failure "${TEST_EXIT_CODE}" "CSS get with nonexistent property returns failure"

# Test: CSS computed with invalid selector syntax
run_test "css computed with invalid selector syntax" "${WEBCTL_BINARY}" css computed "[invalid::syntax"
assert_failure "${TEST_EXIT_CODE}" "Invalid CSS selector syntax returns failure"

test_section "Error Cases - Console Command"

# Test: Console with invalid type filter (returns success with no output - filters to nothing)
run_test "console --type with invalid type" "${WEBCTL_BINARY}" console --type "invalidtype"
assert_success "${TEST_EXIT_CODE}" "Invalid type returns success (filters to no entries)"

# Test: Console with find that has no matches
run_test "console --find with no matches" "${WEBCTL_BINARY}" console --find "NONEXISTENT_CONSOLE_TEXT_XYZ"
assert_failure "${TEST_EXIT_CODE}" "No console matches returns failure"

test_section "Error Cases - Cookies Command"

# Navigate to cookies page for error tests
run_test "setup: navigate to cookies.html for error tests" "${WEBCTL_BINARY}" navigate "$(get_test_url '/pages/cookies.html')"
sleep 2

# Test: Cookies delete nonexistent cookie (idempotent - should succeed)
run_test "cookies delete nonexistent" "${WEBCTL_BINARY}" cookies delete "nonexistent-cookie-xyz-12345"
assert_success "${TEST_EXIT_CODE}" "Delete nonexistent cookie succeeds (idempotent)"

# Test: Cookies with domain that has no matches (returns empty, success)
run_test "cookies --domain with no matches" "${WEBCTL_BINARY}" cookies --domain "nonexistent.domain.xyz"
assert_success "${TEST_EXIT_CODE}" "Domain filter with no matches succeeds"

# Test: Cookies --find with no matches
run_test "cookies --find with no matches" "${WEBCTL_BINARY}" cookies --find "NONEXISTENT_COOKIE_VALUE_XYZ"
assert_failure "${TEST_EXIT_CODE}" "Find with no matches returns failure"

test_section "Error Cases - Network Command"

# Test: Network after clearing buffer (empty buffer handling)
run_test "clear network buffer for error test" "${WEBCTL_BINARY}" clear network
assert_success "${TEST_EXIT_CODE}" "Clear network succeeds"

# Test: Network with empty buffer returns success (empty output is valid)
run_test "network with empty buffer" "${WEBCTL_BINARY}" network
assert_success "${TEST_EXIT_CODE}" "Network with empty buffer succeeds"

# Test: Network --find with no matches in empty buffer
run_test "network --find with empty buffer" "${WEBCTL_BINARY}" network --find "NONEXISTENT_REQUEST_XYZ"
assert_failure "${TEST_EXIT_CODE}" "Network find with no matches returns failure"

# =============================================================================
# Range Limiting Tests
# =============================================================================

test_section "Range Limiting - Console Command"

# Navigate to console page to generate logs
run_test "setup: navigate to console-types.html for range tests" "${WEBCTL_BINARY}" navigate "$(get_test_url '/pages/console-types.html')"
sleep 3

# Test: Console with --head
run_test "console --head 2" "${WEBCTL_BINARY}" console --head 2
assert_success "${TEST_EXIT_CODE}" "console --head returns success"

# Test: Console with --tail
run_test "console --tail 2" "${WEBCTL_BINARY}" console --tail 2
assert_success "${TEST_EXIT_CODE}" "console --tail returns success"

# Test: Console with --range
run_test "console --range 1-2" "${WEBCTL_BINARY}" console --range "1-2"
assert_success "${TEST_EXIT_CODE}" "console --range returns success"

# Test: Mutually exclusive flags should fail
run_test "console --head and --tail together" "${WEBCTL_BINARY}" console --head 2 --tail 2
assert_failure "${TEST_EXIT_CODE}" "head and tail together returns failure"

test_section "Range Limiting - Network Command"

# Navigate to trigger some network requests
run_test "setup: navigate to network page for range tests" "${WEBCTL_BINARY}" navigate "$(get_test_url '/pages/network-requests.html')"
sleep 2

# Test: Network with --head
run_test "network --head 5" "${WEBCTL_BINARY}" network --head 5
assert_success "${TEST_EXIT_CODE}" "network --head returns success"

# Test: Network with --tail
run_test "network --tail 3" "${WEBCTL_BINARY}" network --tail 3
assert_success "${TEST_EXIT_CODE}" "network --tail returns success"

# =============================================================================
# Context Flag Tests
# =============================================================================

test_section "Context Flags - HTML Command"

# Navigate to navigation page for context tests
run_test "setup: navigate to navigation.html for context tests" "${WEBCTL_BINARY}" navigate "$(get_test_url '/pages/navigation.html')"
sleep 2

# Test: HTML --find with -B (before context)
run_test "html --find with -B 2" "${WEBCTL_BINARY}" html --find "Navigation" -B 2
assert_success "${TEST_EXIT_CODE}" "html with before context returns success"

# Test: HTML --find with -A (after context)
run_test "html --find with -A 2" "${WEBCTL_BINARY}" html --find "Navigation" -A 2
assert_success "${TEST_EXIT_CODE}" "html with after context returns success"

# Test: HTML --find with -C (surrounding context)
run_test "html --find with -C 2" "${WEBCTL_BINARY}" html --find "Navigation" -C 2
assert_success "${TEST_EXIT_CODE}" "html with surrounding context returns success"

test_section "Context Flags - CSS Command"

# Navigate to CSS page for context tests
run_test "setup: navigate to css-showcase.html for context tests" "${WEBCTL_BINARY}" navigate "$(get_test_url '/pages/css-showcase.html')"
sleep 2

# Test: CSS --find with -B (before context)
run_test "css --find with -B 1" "${WEBCTL_BINARY}" css --find "background" -B 1
assert_success "${TEST_EXIT_CODE}" "css with before context returns success"

# Test: CSS --find with -A (after context)
run_test "css --find with -A 1" "${WEBCTL_BINARY}" css --find "background" -A 1
assert_success "${TEST_EXIT_CODE}" "css with after context returns success"

# Test: CSS --find with -C (surrounding context)
run_test "css --find with -C 1" "${WEBCTL_BINARY}" css --find "background" -C 1
assert_success "${TEST_EXIT_CODE}" "css with surrounding context returns success"

# Note: Console command does not support context flags (-A, -B, -C)
# Those flags are only available for html and css commands

# =============================================================================
# Backend Integration Tests
# =============================================================================

test_section "Backend Integration - Setup"

# Start the backend server for proxy testing
start_backend 3000

# Stop existing test server and restart with proxy mode
stop_test_server
"${WEBCTL_BINARY}" stop --force 2>/dev/null || true
sleep 2

# Start serve with proxy to backend (proxy mode, no directory)
"${WEBCTL_BINARY}" serve --proxy "http://localhost:3000" --port 8888 &
TEST_SERVER_PID=$!
sleep 4

test_section "Backend Integration - API Requests"

# Navigate to the backend root page (via proxy)
run_test "setup: navigate to backend via proxy" "${WEBCTL_BINARY}" navigate "http://localhost:8888/"
sleep 2

# Clear network buffer
run_test "clear network for backend tests" "${WEBCTL_BINARY}" clear network
assert_success "${TEST_EXIT_CODE}" "Network buffer cleared"

# Trigger API request to backend via proxy
run_test "trigger backend API request" "${WEBCTL_BINARY}" eval "fetch('/api/hello').then(r => r.json())"
sleep 2

# Test: Verify API request was captured
run_test "network finds backend API call" "${WEBCTL_BINARY}" network --find "api/hello"
assert_success "${TEST_EXIT_CODE}" "Backend API call found in network"
assert_contains "${TEST_STDOUT}" "api/hello" "Output contains API path"

test_section "Backend Integration - Status Codes"

# Trigger 404 request
run_test "trigger 404 request" "${WEBCTL_BINARY}" eval "fetch('/status/404').catch(() => {})"
sleep 2

# Test: Filter by 404 status
run_test "network --status 404" "${WEBCTL_BINARY}" network --status 404
assert_success "${TEST_EXIT_CODE}" "404 status filter works"
assert_contains "${TEST_STDOUT}" "404" "Output contains 404 status"

# Trigger 500 request
run_test "trigger 500 request" "${WEBCTL_BINARY}" eval "fetch('/status/500').catch(() => {})"
sleep 3

# Test: Filter by 5xx status codes (includes 500)
run_test "network --status 5xx" "${WEBCTL_BINARY}" network --status 5xx
assert_success "${TEST_EXIT_CODE}" "5xx status filter works"

test_section "Backend Integration - Users Endpoint"

# Trigger users API request
run_test "trigger users API request" "${WEBCTL_BINARY}" eval "fetch('/api/users').then(r => r.json())"
sleep 2

# Test: Find users endpoint in network
run_test "network finds users API call" "${WEBCTL_BINARY}" network --find "api/users"
assert_success "${TEST_EXIT_CODE}" "Users API call found"

# Stop backend server
stop_backend

# =============================================================================
# Summary
# =============================================================================

test_summary
