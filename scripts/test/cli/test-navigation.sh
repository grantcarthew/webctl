#!/usr/bin/env bash

# Test: CLI Navigation Commands
# -----------------------------
# Tests for webctl navigate, reload, back, and forward commands.
# Verifies browser navigation including URL navigation and history.

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
# History Error Cases (Test First - Before Any Navigation)
# =============================================================================

test_section "Back Command (No History - Fresh Start)"

# Test: Back with no history - immediately after daemon start
run_test "back with no history (fresh start)" "${WEBCTL_BINARY}" back
assert_contains "${TEST_STDERR}" "No previous page" "Error shows no previous page"
assert_failure "${TEST_EXIT_CODE}" "Back fails when no history"

test_section "Forward Command (No History - Fresh Start)"

# Test: Forward with no history - immediately after daemon start
run_test "forward with no history (fresh start)" "${WEBCTL_BINARY}" forward
assert_contains "${TEST_STDERR}" "No next page" "Error shows no next page"
assert_failure "${TEST_EXIT_CODE}" "Forward fails when no forward history"

# =============================================================================
# Navigate Command Tests
# =============================================================================

test_section "Navigate Command"

# Test: Navigate to test server URL
run_test "navigate to test server URL" "${WEBCTL_BINARY}" navigate "$(get_test_url '/pages/navigation.html')"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK on navigate"
assert_success "${TEST_EXIT_CODE}" "Navigate returns success"

# Test: Navigate to another page (for history tests later)
run_test "navigate to forms page" "${WEBCTL_BINARY}" navigate "$(get_test_url '/pages/forms.html')"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK on navigate"
assert_success "${TEST_EXIT_CODE}" "Navigate returns success"

# Test: Navigate to file URL
test_section "Navigate Command (File URL)"

FILE_URL="file://${PROJECT_ROOT}/testdata/pages/navigation.html"
run_test "navigate to file URL" "${WEBCTL_BINARY}" navigate "${FILE_URL}"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK for file URL"
assert_success "${TEST_EXIT_CODE}" "File URL navigation returns success"

# Test: Navigate to invalid URL
test_section "Navigate Command (Error Cases)"

run_test "navigate to invalid URL" "${WEBCTL_BINARY}" navigate "not-a-valid-url-at-all"
assert_failure "${TEST_EXIT_CODE}" "Invalid URL navigation fails"

# =============================================================================
# Reload Command Tests
# =============================================================================

test_section "Reload Command"

# First navigate to a known page
run_test "setup: navigate for reload test" "${WEBCTL_BINARY}" navigate "$(get_test_url '/pages/navigation.html')"
assert_success "${TEST_EXIT_CODE}" "Setup navigation succeeded"

# Test: Reload current page
run_test "reload current page" "${WEBCTL_BINARY}" reload
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK on reload"
assert_success "${TEST_EXIT_CODE}" "Reload returns success"

# =============================================================================
# Back Command Tests (With History)
# =============================================================================

test_section "Back Command (With History)"

# Setup: Navigate to create history
# Page 1
run_test "setup: navigate to page 1" "${WEBCTL_BINARY}" navigate "$(get_test_url '/pages/navigation.html')"
assert_success "${TEST_EXIT_CODE}" "Navigate to page 1 succeeded"

# Page 2
run_test "setup: navigate to page 2" "${WEBCTL_BINARY}" navigate "$(get_test_url '/pages/forms.html')"
assert_success "${TEST_EXIT_CODE}" "Navigate to page 2 succeeded"

# Page 3
run_test "setup: navigate to page 3" "${WEBCTL_BINARY}" navigate "$(get_test_url '/pages/cookies.html')"
assert_success "${TEST_EXIT_CODE}" "Navigate to page 3 succeeded"

# Test: Back with history
run_test "back with history" "${WEBCTL_BINARY}" back
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK on back"
assert_success "${TEST_EXIT_CODE}" "Back returns success"

# Test: Back again
run_test "back again" "${WEBCTL_BINARY}" back
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK on second back"
assert_success "${TEST_EXIT_CODE}" "Second back returns success"

# =============================================================================
# Forward Command Tests (With History)
# =============================================================================

test_section "Forward Command (With History)"

# After going back twice, we should have forward history
# Test: Forward with history
run_test "forward with history" "${WEBCTL_BINARY}" forward
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK on forward"
assert_success "${TEST_EXIT_CODE}" "Forward returns success"

# Forward again
run_test "forward again" "${WEBCTL_BINARY}" forward
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK on second forward"
assert_success "${TEST_EXIT_CODE}" "Second forward returns success"

# Now at the end of history - forward should fail
test_section "Forward Command (End of History)"

run_test "forward at end of history" "${WEBCTL_BINARY}" forward
assert_contains "${TEST_STDERR}" "No next page" "Error shows no next page"
assert_failure "${TEST_EXIT_CODE}" "Forward fails at end of history"

# =============================================================================
# Summary
# =============================================================================

test_summary
