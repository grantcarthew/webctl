#!/usr/bin/env bash

# Test: CLI Navigation Commands
# -----------------------------
# Tests for webctl navigate, reload, back, and forward commands.
# Verifies browser navigation including URL navigation, history, and flags.
#
# Test Design:
# - Setup steps use run_setup/run_setup_required (don't count toward test totals)
# - Critical setup uses run_setup_required which aborts on failure
# - All navigations that precede page state assertions use --wait for reliability
# - Each section is fully self-contained with explicit setup
# - Page verification uses capture_page_state for reliable state capture
# - Error assertions use consistent JSON field access
# - Timeout behavior tested via flag acceptance (actual timeout tested in unit tests)

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

# Verify required test pages exist before running tests
require_test_pages \
  'pages/blank.html' \
  'pages/navigation.html' \
  'pages/forms.html' \
  'pages/cookies.html'

# Ensure clean state before tests
force_stop_daemon

# Start daemon and test server
start_daemon --headless
start_test_server

# =============================================================================
# Navigate Command - Basic Tests
# =============================================================================

test_section "Navigate Command (Basic)"

# Setup: Initial navigation after daemon startup
# The webctl serve command navigates the browser to the serve root URL. We need
# to wait for that navigation to complete before running tests, otherwise there's
# a race condition. This initial navigation with --wait ensures the browser is
# stable and ready for testing.
run_setup_required "initial navigation" "${WEBCTL_BINARY}" navigate --wait --timeout "${TEST_TIMEOUT}" "$(get_test_url '/pages/blank.html')"

# Test: Navigate to test server URL (with --wait for reliable page state check)
run_test "navigate to test server URL" "${WEBCTL_BINARY}" navigate --wait --timeout "${TEST_TIMEOUT}" "$(get_test_url '/pages/navigation.html')"
assert_success "${TEST_EXIT_CODE}" "Navigate returns success"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK on navigate"
capture_page_state
assert_captured_url "navigation.html" "Browser navigated to navigation.html"

# Test: Navigate to another page (with --wait for reliable page state check)
run_test "navigate to forms page" "${WEBCTL_BINARY}" navigate --wait --timeout "${TEST_TIMEOUT}" "$(get_test_url '/pages/forms.html')"
assert_success "${TEST_EXIT_CODE}" "Navigate returns success"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK on navigate"
capture_page_state
assert_captured_url "forms.html" "Browser navigated to forms.html"

# =============================================================================
# Navigate Command - Wait Flag Tests
# =============================================================================

test_section "Navigate Command (Wait Flag)"

# Test: Navigate with --wait flag
run_test "navigate --wait" "${WEBCTL_BINARY}" navigate --wait --timeout "${TEST_TIMEOUT}" "$(get_test_url '/pages/navigation.html')"
assert_success "${TEST_EXIT_CODE}" "navigate --wait returns success"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"
capture_page_state
assert_captured_url "navigation.html" "Browser on navigation.html after --wait"

# Test: Navigate with --wait --timeout flags (explicit short timeout)
run_test "navigate --wait --timeout" "${WEBCTL_BINARY}" navigate --wait --timeout 10 "$(get_test_url '/pages/forms.html')"
assert_success "${TEST_EXIT_CODE}" "navigate --wait --timeout returns success"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"
capture_page_state
assert_captured_url "forms.html" "Browser on forms.html after --wait --timeout"

# =============================================================================
# Navigate Command - JSON Output
# =============================================================================

test_section "Navigate Command (JSON Output)"

# Test: Navigate with --json flag (title may be empty without --wait)
run_test "navigate --json" "${WEBCTL_BINARY}" navigate --json "$(get_test_url '/pages/navigation.html')"
assert_success "${TEST_EXIT_CODE}" "navigate --json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
assert_json_field_exists "${TEST_STDOUT}" ".url" "JSON contains url field"
# Note: title may be empty without --wait since page is still loading

# Test: Navigate with --wait --json combined (title should be populated)
run_test "navigate --wait --json" "${WEBCTL_BINARY}" navigate --wait --json --timeout "${TEST_TIMEOUT}" "$(get_test_url '/pages/forms.html')"
assert_success "${TEST_EXIT_CODE}" "navigate --wait --json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
assert_json_field_exists "${TEST_STDOUT}" ".title" "JSON title populated with --wait"

# =============================================================================
# Navigate Command - File URL
# =============================================================================

test_section "Navigate Command (File URL)"

FILE_URL="file://${PROJECT_ROOT}/testdata/pages/navigation.html"
run_test "navigate to file URL" "${WEBCTL_BINARY}" navigate --wait --timeout "${TEST_TIMEOUT}" "${FILE_URL}"
assert_success "${TEST_EXIT_CODE}" "File URL navigation returns success"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK for file URL"
capture_page_state
assert_captured_url "navigation.html" "Browser on file URL page"

# =============================================================================
# Navigate Command - URL Normalization
# =============================================================================

test_section "Navigate Command (URL Normalization)"

# Test: Navigate to localhost without explicit protocol (should use http://)
run_test "navigate to localhost without protocol" "${WEBCTL_BINARY}" navigate --wait --timeout "${TEST_TIMEOUT}" "localhost:${TEST_SERVER_PORT}/pages/navigation.html"
assert_success "${TEST_EXIT_CODE}" "navigate to localhost works"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"
capture_page_state
assert_captured_url "navigation.html" "Browser navigated via localhost"

# Test: Navigate to 127.0.0.1 without explicit protocol (should use http://)
run_test "navigate to 127.0.0.1 without protocol" "${WEBCTL_BINARY}" navigate --wait --timeout "${TEST_TIMEOUT}" "127.0.0.1:${TEST_SERVER_PORT}/pages/forms.html"
assert_success "${TEST_EXIT_CODE}" "navigate to 127.0.0.1 works"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"
capture_page_state
assert_captured_url "forms.html" "Browser navigated via 127.0.0.1"

# =============================================================================
# Navigate Command - Error Cases
# =============================================================================

test_section "Navigate Command (Error Cases)"

# Test: Navigate to invalid/unreachable URL (with --wait for consistent error handling)
# We check for net::ERR pattern which Chrome uses for network errors
run_test "navigate to invalid URL" "${WEBCTL_BINARY}" navigate --wait --timeout "${TEST_TIMEOUT}" "not-a-valid-url-that-exists.invalid"
assert_failure "${TEST_EXIT_CODE}" "Invalid URL navigation fails"
assert_contains "${TEST_STDERR}" "net::ERR" "Error output contains Chrome network error"

# Test: Navigate --json to invalid URL - verify JSON error output using consistent assertions
run_test "navigate --json to invalid URL" "${WEBCTL_BINARY}" navigate --json --wait --timeout "${TEST_TIMEOUT}" "not-a-valid-url-that-exists.invalid"
assert_failure "${TEST_EXIT_CODE}" "Navigate --json fails on invalid URL"
assert_valid_json_error "${TEST_STDERR}" "JSON error response is valid"
assert_json_error_contains "${TEST_STDERR}" "net::ERR" "JSON error contains network error"

# =============================================================================
# Reload Command Tests
# =============================================================================

test_section "Reload Command"

# Setup: Navigate to a known page first (required for reload to work)
run_setup_required "navigate for reload test" "${WEBCTL_BINARY}" navigate --wait --timeout "${TEST_TIMEOUT}" "$(get_test_url '/pages/navigation.html')"

# Test: Reload current page (with --wait for reliable page state check)
run_test "reload current page" "${WEBCTL_BINARY}" reload --wait --timeout "${TEST_TIMEOUT}"
assert_success "${TEST_EXIT_CODE}" "Reload returns success"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK on reload"
capture_page_state
assert_captured_url "navigation.html" "Still on navigation.html after reload"

# Test: Reload with --wait flag
run_test "reload --wait" "${WEBCTL_BINARY}" reload --wait --timeout "${TEST_TIMEOUT}"
assert_success "${TEST_EXIT_CODE}" "reload --wait returns success"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"

# Test: Reload with --wait --timeout flags (explicit short timeout)
run_test "reload --wait --timeout" "${WEBCTL_BINARY}" reload --wait --timeout 10
assert_success "${TEST_EXIT_CODE}" "reload --wait --timeout returns success"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"

# Test: Reload with --json flag
run_test "reload --json" "${WEBCTL_BINARY}" reload --json
assert_success "${TEST_EXIT_CODE}" "reload --json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"

# Test: Reload with --wait --json combined
run_test "reload --wait --json" "${WEBCTL_BINARY}" reload --wait --json --timeout "${TEST_TIMEOUT}"
assert_success "${TEST_EXIT_CODE}" "reload --wait --json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"

# =============================================================================
# Back Command Tests
# =============================================================================

test_section "Back Command"

# Setup: Create controlled history chain for this section
# History: blank.html -> navigation.html -> forms.html -> cookies.html
# We navigate from cookies.html backward through this history.
setup_history_chain '/pages/navigation.html' '/pages/forms.html' '/pages/cookies.html'

# Test: Back to page 2 (with --wait for reliable page state check)
run_test "back with history" "${WEBCTL_BINARY}" back --wait --timeout "${TEST_TIMEOUT}"
assert_success "${TEST_EXIT_CODE}" "Back returns success"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK on back"
capture_page_state
assert_captured_url "forms.html" "Navigated back to forms.html (page 2)"

# Test: Back again to page 1
run_test "back again" "${WEBCTL_BINARY}" back --wait --timeout "${TEST_TIMEOUT}"
assert_success "${TEST_EXIT_CODE}" "Second back returns success"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK on second back"
capture_page_state
assert_captured_url "navigation.html" "Navigated back to navigation.html (page 1)"

# =============================================================================
# Back Command - Flag Tests
# =============================================================================

test_section "Back Command (Flags)"

# Setup: Create fresh history for each flag test to avoid fragile restore pattern
# Each test rebuilds the history chain to ensure consistent state

# Test: Back with --wait flag
setup_history_chain '/pages/navigation.html' '/pages/forms.html'
run_test "back --wait" "${WEBCTL_BINARY}" back --wait --timeout "${TEST_TIMEOUT}"
assert_success "${TEST_EXIT_CODE}" "back --wait returns success"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"
capture_page_state
assert_captured_url "navigation.html" "Back --wait navigated to navigation.html"

# Test: Back with --wait --timeout flags
setup_history_chain '/pages/navigation.html' '/pages/forms.html'
run_test "back --wait --timeout" "${WEBCTL_BINARY}" back --wait --timeout 10
assert_success "${TEST_EXIT_CODE}" "back --wait --timeout returns success"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"

# Test: Back with --json flag
setup_history_chain '/pages/navigation.html' '/pages/forms.html'
run_test "back --json" "${WEBCTL_BINARY}" back --json
assert_success "${TEST_EXIT_CODE}" "back --json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
assert_json_field_exists "${TEST_STDOUT}" ".url" "JSON contains url field"

# Test: Back with --wait --json combined
setup_history_chain '/pages/navigation.html' '/pages/forms.html'
run_test "back --wait --json" "${WEBCTL_BINARY}" back --wait --json --timeout "${TEST_TIMEOUT}"
assert_success "${TEST_EXIT_CODE}" "back --wait --json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"

# =============================================================================
# Forward Command Tests
# =============================================================================

test_section "Forward Command"

# Setup: Create history chain then navigate back to first page
# History: blank.html -> navigation.html -> forms.html -> cookies.html
# Position: At navigation.html with forward history to forms.html and cookies.html
setup_history_chain '/pages/navigation.html' '/pages/forms.html' '/pages/cookies.html'
run_setup_required "back to page 2" "${WEBCTL_BINARY}" back --wait --timeout "${TEST_TIMEOUT}"
run_setup_required "back to page 1" "${WEBCTL_BINARY}" back --wait --timeout "${TEST_TIMEOUT}"

# Test: Forward to page 2 (with --wait for reliable page state check)
run_test "forward with history" "${WEBCTL_BINARY}" forward --wait --timeout "${TEST_TIMEOUT}"
assert_success "${TEST_EXIT_CODE}" "Forward returns success"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK on forward"
capture_page_state
assert_captured_url "forms.html" "Navigated forward to forms.html (page 2)"

# Test: Forward again to page 3
run_test "forward again" "${WEBCTL_BINARY}" forward --wait --timeout "${TEST_TIMEOUT}"
assert_success "${TEST_EXIT_CODE}" "Second forward returns success"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK on second forward"
capture_page_state
assert_captured_url "cookies.html" "Navigated forward to cookies.html (page 3)"

# =============================================================================
# Forward Command - End of History Error
# =============================================================================

test_section "Forward Command (End of History)"

# Setup: Navigate to end of forward history (self-contained section)
setup_history_chain '/pages/navigation.html' '/pages/forms.html'

# Test: Forward at end of history should fail (with --wait for consistent behavior)
run_test "forward at end of history" "${WEBCTL_BINARY}" forward --wait --timeout "${TEST_TIMEOUT}"
assert_failure "${TEST_EXIT_CODE}" "Forward fails at end of history"
assert_contains "${TEST_STDERR}" "No next page" "Error shows 'No next page'"

# Test: Forward --json at end of history - verify JSON error output using consistent assertions
# Note: The failed forward above doesn't change browser state, so we can test
# the JSON variant immediately without needing to reset
run_test "forward --json at end of history" "${WEBCTL_BINARY}" forward --json --wait --timeout "${TEST_TIMEOUT}"
assert_failure "${TEST_EXIT_CODE}" "Forward --json fails at end of history"
assert_valid_json_error "${TEST_STDERR}" "JSON error response is valid"
assert_json_error_contains "${TEST_STDERR}" "No next page" "JSON error contains 'No next page'"

# =============================================================================
# Forward Command - Flag Tests
# =============================================================================

test_section "Forward Command (Flags)"

# Setup: Each test rebuilds history to avoid fragile restore pattern

# Test: Forward with --wait flag
setup_history_chain '/pages/navigation.html' '/pages/forms.html'
run_setup_required "back for forward test" "${WEBCTL_BINARY}" back --wait --timeout "${TEST_TIMEOUT}"
run_test "forward --wait" "${WEBCTL_BINARY}" forward --wait --timeout "${TEST_TIMEOUT}"
assert_success "${TEST_EXIT_CODE}" "forward --wait returns success"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"
capture_page_state
assert_captured_url "forms.html" "Forward --wait navigated to forms.html"

# Test: Forward with --wait --timeout flags
setup_history_chain '/pages/navigation.html' '/pages/forms.html'
run_setup_required "back for forward test" "${WEBCTL_BINARY}" back --wait --timeout "${TEST_TIMEOUT}"
run_test "forward --wait --timeout" "${WEBCTL_BINARY}" forward --wait --timeout 10
assert_success "${TEST_EXIT_CODE}" "forward --wait --timeout returns success"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"

# Test: Forward with --json flag
setup_history_chain '/pages/navigation.html' '/pages/forms.html'
run_setup_required "back for forward test" "${WEBCTL_BINARY}" back --wait --timeout "${TEST_TIMEOUT}"
run_test "forward --json" "${WEBCTL_BINARY}" forward --json
assert_success "${TEST_EXIT_CODE}" "forward --json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
assert_json_field_exists "${TEST_STDOUT}" ".url" "JSON contains url field"

# Test: Forward with --wait --json combined
setup_history_chain '/pages/navigation.html' '/pages/forms.html'
run_setup_required "back for forward test" "${WEBCTL_BINARY}" back --wait --timeout "${TEST_TIMEOUT}"
run_test "forward --wait --json" "${WEBCTL_BINARY}" forward --wait --json --timeout "${TEST_TIMEOUT}"
assert_success "${TEST_EXIT_CODE}" "forward --wait --json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"

# =============================================================================
# Back Command - Beginning of History Error
# =============================================================================
# Note: These tests require a completely clean browser history, which can only
# be achieved by restarting the daemon. This is by design - Chrome maintains
# history state that cannot be cleared without restarting the browser process.
# The daemon restart ensures no prior navigation entries exist.

test_section "Back Command (Beginning of History)"

# Restart daemon for truly clean history state (no prior navigation)
# NOTE: We do NOT start the test server here because webctl serve navigates
# the browser to the served URL, which creates a history entry. For testing
# "no history" state, we need just the daemon with no navigations.
restart_daemon_clean --headless

# Test: Back immediately after fresh start should fail (browser on about:blank, no history)
# Use --wait for consistent error handling behavior
run_test "back at beginning of history" "${WEBCTL_BINARY}" back --wait --timeout "${TEST_TIMEOUT}"
assert_failure "${TEST_EXIT_CODE}" "Back fails at beginning of history"
assert_contains "${TEST_STDERR}" "No previous page" "Error shows 'No previous page'"

# Test: Back --json at beginning of history - verify JSON error output
# Note: Failed back command above does not change browser state, so we can
# test the JSON variant immediately without needing to restart
run_test "back --json at beginning of history" "${WEBCTL_BINARY}" back --json --wait --timeout "${TEST_TIMEOUT}"
assert_failure "${TEST_EXIT_CODE}" "Back --json fails at beginning of history"
assert_valid_json_error "${TEST_STDERR}" "JSON error response is valid"
assert_json_error_contains "${TEST_STDERR}" "No previous page" "JSON error contains 'No previous page'"

# =============================================================================
# Timeout Flag Tests
# =============================================================================
# Note: These tests verify timeout flags are accepted and work correctly.
# Actual timeout expiration (when page load exceeds timeout) is tested in
# unit tests (internal/cli/cli_test.go) because it requires controlled slow
# responses which are impractical in integration tests.

test_section "Timeout Flag Tests"

# Restart test server (was not started in previous section) - self-contained setup
start_test_server

# Test: Navigate --wait with explicit timeout (should succeed on fast page)
run_test "navigate --wait --timeout on fast page" "${WEBCTL_BINARY}" navigate --wait --timeout 5 "$(get_test_url '/pages/navigation.html')"
assert_success "${TEST_EXIT_CODE}" "navigate --wait --timeout 5 succeeds on fast page"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"

# Test: Reload with --wait --timeout
run_test "reload --wait --timeout on fast page" "${WEBCTL_BINARY}" reload --wait --timeout 5
assert_success "${TEST_EXIT_CODE}" "reload --wait --timeout succeeds on fast page"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"

# Test: --timeout 0 means no timeout limit (wait indefinitely for page load)
run_test "navigate --timeout 0" "${WEBCTL_BINARY}" navigate --wait --timeout 0 "$(get_test_url '/pages/forms.html')"
assert_success "${TEST_EXIT_CODE}" "navigate --timeout 0 succeeds (no timeout limit)"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"

# =============================================================================
# Global Flags Tests
# =============================================================================

test_section "Global Flags"

# Navigate to a known page for consistent output
run_setup_required "setup for global flags" "${WEBCTL_BINARY}" navigate --wait --timeout "${TEST_TIMEOUT}" "$(get_test_url '/pages/navigation.html')"

# Test: --no-color flag (verify it's accepted and doesn't break output)
run_test "navigate --no-color" "${WEBCTL_BINARY}" navigate --no-color --wait --timeout "${TEST_TIMEOUT}" "$(get_test_url '/pages/forms.html')"
assert_success "${TEST_EXIT_CODE}" "--no-color flag is accepted"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK with --no-color"
# Verify no ANSI escape codes in output using dedicated assertion
assert_no_ansi_codes "${TEST_STDOUT}" "--no-color output has no ANSI escape codes"

# Test: --debug flag (verify it produces debug output on stderr)
run_test "navigate --debug" "${WEBCTL_BINARY}" navigate --debug --wait --timeout "${TEST_TIMEOUT}" "$(get_test_url '/pages/navigation.html')"
assert_success "${TEST_EXIT_CODE}" "--debug flag is accepted"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK with --debug"
# Debug output goes to stderr and contains [DEBUG] markers
assert_contains "${TEST_STDERR}" "[DEBUG]" "--debug produces debug output on stderr"

# Test: --json with other global flags
run_test "navigate --json --no-color" "${WEBCTL_BINARY}" navigate --json --no-color --wait --timeout "${TEST_TIMEOUT}" "$(get_test_url '/pages/forms.html')"
assert_success "${TEST_EXIT_CODE}" "--json --no-color flags work together"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON output with --no-color"

# =============================================================================
# Summary
# =============================================================================

test_summary
