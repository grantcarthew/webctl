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

# Note: css inline and css matched subcommands don't exist in webctl
# Skipping these tests

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

# Test: Network with text search (may have no matches if no API calls made)
run_test "network with --find" "${WEBCTL_BINARY}" network --find "network"
# Don't assert success - may be no matches, which is valid

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
# Summary
# =============================================================================

test_summary
