#!/usr/bin/env bash

# Test: CLI Tab Commands
# ----------------------
# Tests for webctl tab, tab switch, tab new, tab close.
# Verifies tab listing, switching (including ambiguous/no-match queries),
# tab creation (about:blank, explicit URL, localhost auto-detection),
# and tab close (active tab, by query, last-tab guard, active promotion).

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

# Verify required test pages exist
require_test_pages \
  'pages/blank.html' \
  'pages/navigation.html' \
  'pages/forms.html'

# Ensure clean state
force_stop_daemon

# Start daemon and test server
start_daemon --headless
start_test_server

# Initial navigation to a known page so listings are predictable.
run_setup_required "initial navigation" "${WEBCTL_BINARY}" navigate --wait --timeout "${TEST_TIMEOUT}" "$(get_test_url '/pages/blank.html')"

# =============================================================================
# tab list
# =============================================================================

test_section "tab (list)"

# Bare list returns success and includes the active marker.
run_test "tab list (text)" "${WEBCTL_BINARY}" tab
assert_success "${TEST_EXIT_CODE}" "tab returns success"
assert_contains "${TEST_STDOUT}" "*" "tab output marks active tab with *"

# JSON listing exposes activeSession and sessions[].
run_test "tab list (json)" "${WEBCTL_BINARY}" tab --json
assert_success "${TEST_EXIT_CODE}" "tab --json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "json ok=true"
assert_json_field_exists "${TEST_STDOUT}" ".activeSession" "json includes activeSession"
assert_json_field_exists "${TEST_STDOUT}" ".sessions" "json includes sessions"

# =============================================================================
# tab close (last-tab guard)
# =============================================================================

test_section "tab close (last-tab guard)"

# With only one tab, close must refuse with the documented message.
run_test "tab close last tab refused" "${WEBCTL_BINARY}" tab close
assert_failure "${TEST_EXIT_CODE}" "tab close refuses last tab"
assert_contains "${TEST_STDOUT}${TEST_STDERR}" "cannot close the last tab" "error mentions last-tab guard"
assert_contains "${TEST_STDOUT}${TEST_STDERR}" "webctl stop" "error suggests webctl stop"

# =============================================================================
# tab new
# =============================================================================

test_section "tab new"

# tab new with no URL opens about:blank and makes it active.
run_test "tab new (no url)" "${WEBCTL_BINARY}" tab new --json
assert_success "${TEST_EXIT_CODE}" "tab new returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "tab new json ok=true"
assert_json_field_exists "${TEST_STDOUT}" ".id" "tab new returns id"

# Save the new id for later switching.
NEW_TAB1_ID=$(echo "${TEST_STDOUT}" | jq -r '.id')

# tab new with an explicit URL.
run_test "tab new with url" "${WEBCTL_BINARY}" tab new --json "$(get_test_url '/pages/forms.html')"
assert_success "${TEST_EXIT_CODE}" "tab new with url returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "tab new with url json ok=true"

NEW_TAB2_ID=$(echo "${TEST_STDOUT}" | jq -r '.id')

# tab new with localhost auto-detection (no protocol). The daemon receives
# the http:// URL because the CLI normalizes it before sending.
run_test "tab new localhost auto-detection" "${WEBCTL_BINARY}" tab new --json "localhost:${TEST_SERVER_PORT}/pages/navigation.html"
assert_success "${TEST_EXIT_CODE}" "tab new localhost returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "tab new localhost json ok=true"

# Verify there are now multiple tabs.
run_test "tab list after new" "${WEBCTL_BINARY}" tab --json
assert_success "${TEST_EXIT_CODE}" "tab list after new"
TAB_COUNT=$(echo "${TEST_STDOUT}" | jq '.sessions | length')
if [[ "${TAB_COUNT}" -ge 4 ]]; then
  log_success "Tab list contains expected number of tabs (${TAB_COUNT})"
  increment_pass
else
  log_failure "Tab list has only ${TAB_COUNT} tabs, expected >= 4"
  increment_fail
fi

# =============================================================================
# tab switch
# =============================================================================

test_section "tab switch"

# Switch to a tab by id prefix.
SHORT_ID="${NEW_TAB1_ID:0:8}"
run_test "tab switch by id prefix" "${WEBCTL_BINARY}" tab switch --json "${SHORT_ID}"
assert_success "${TEST_EXIT_CODE}" "tab switch returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "tab switch json ok=true"

# Verify the active session changed to the chosen tab.
run_test "tab list shows chosen active" "${WEBCTL_BINARY}" tab --json
ACTIVE=$(echo "${TEST_STDOUT}" | jq -r '.activeSession')
if [[ "${ACTIVE}" == "${NEW_TAB1_ID}" ]]; then
  log_success "Active session is the switched tab"
  increment_pass
else
  log_failure "Expected active=${NEW_TAB1_ID}, got ${ACTIVE}"
  increment_fail
fi

# Switch with no match returns an error.
run_test "tab switch no match" "${WEBCTL_BINARY}" tab switch ZZZZZZZZZZZZZZZZZZZZ-no-such-tab
assert_failure "${TEST_EXIT_CODE}" "tab switch no-match fails"
assert_contains "${TEST_STDOUT}${TEST_STDERR}" "no tab matches query" "error mentions no match"

# =============================================================================
# tab close (with query and active-tab promotion)
# =============================================================================

test_section "tab close (with query / active promotion)"

# Close a tab by id prefix.
SHORT_ID2="${NEW_TAB2_ID:0:8}"
run_test "tab close by id prefix" "${WEBCTL_BINARY}" tab close --json "${SHORT_ID2}"
assert_success "${TEST_EXIT_CODE}" "tab close by query returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "tab close json ok=true"

# Verify the closed tab is gone from the list.
run_test "tab list does not include closed tab" "${WEBCTL_BINARY}" tab --json
assert_success "${TEST_EXIT_CODE}" "tab list after close"
if echo "${TEST_STDOUT}" | jq -e --arg id "${NEW_TAB2_ID}" '.sessions | map(.id) | index($id)' >/dev/null 2>&1; then
  log_failure "Closed tab still present in list"
  increment_fail
else
  log_success "Closed tab removed from list"
  increment_pass
fi

# Close the active tab with no query — should succeed and promote a remaining tab.
run_test "tab close active (no query)" "${WEBCTL_BINARY}" tab close --json
assert_success "${TEST_EXIT_CODE}" "tab close active returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "tab close active json ok=true"

# Active session should still exist (something else got promoted).
run_test "tab list after closing active" "${WEBCTL_BINARY}" tab --json
NEW_ACTIVE=$(echo "${TEST_STDOUT}" | jq -r '.activeSession')
if [[ -n "${NEW_ACTIVE}" && "${NEW_ACTIVE}" != "null" ]]; then
  log_success "A new active tab was promoted (${NEW_ACTIVE})"
  increment_pass
else
  log_failure "No active tab after closing the active one"
  increment_fail
fi

# =============================================================================
# Cleanup
# =============================================================================

force_stop_daemon

test_summary
