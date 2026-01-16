#!/usr/bin/env bash

# Test: CLI Start/Stop Commands
# -----------------------------
# Tests for webctl start, stop, and status commands.
# Verifies daemon lifecycle management including starting, stopping,
# status checking, and error handling.

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

# =============================================================================
# Status Command Tests (Not Running)
# =============================================================================

test_section "Status Command (Not Running)"

run_test "status when not running" "${WEBCTL_BINARY}" status
assert_contains "${TEST_STDOUT}" "Not running" "Output shows not running"
assert_success "${TEST_EXIT_CODE}" "Status returns success even when not running"

# =============================================================================
# Start Command Tests
# =============================================================================

test_section "Start Command"

# Start daemon in headless mode using the helper (starts in background, waits for ready)
# The start command blocks, so we use start_daemon helper which handles background start
"${WEBCTL_BINARY}" start --headless &
DAEMON_PID=$!
sleep 2

if is_daemon_running; then
  log_success "Start command: daemon started successfully (headless)"
  increment_pass
  DAEMON_STARTED_BY_TEST=true
else
  log_failure "Start command: daemon failed to start"
  increment_fail
fi

# =============================================================================
# Status Command Tests (Running)
# =============================================================================

test_section "Status Command (Running)"

run_test "status when running" "${WEBCTL_BINARY}" status
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK when running"
assert_success "${TEST_EXIT_CODE}" "Status returns success when running"

# =============================================================================
# Start Command Error Tests
# =============================================================================

test_section "Start Command (Already Running)"

# Try to start another daemon - this should fail immediately with an error
run_test "start when already running" "${WEBCTL_BINARY}" start --headless
assert_contains "${TEST_STDOUT}${TEST_STDERR}" "already running" "Error mentions already running"
assert_failure "${TEST_EXIT_CODE}" "Start fails when daemon already running"

# =============================================================================
# Stop Command Tests
# =============================================================================

test_section "Stop Command"

run_test "stop daemon (graceful)" "${WEBCTL_BINARY}" stop
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK on stop"
assert_success "${TEST_EXIT_CODE}" "Stop returns success"

# Wait for daemon to fully stop
sleep 1

# Verify daemon stopped
run_test "verify daemon stopped" "${WEBCTL_BINARY}" status
assert_contains "${TEST_STDOUT}" "Not running" "Daemon is no longer running"
DAEMON_STARTED_BY_TEST=false

# =============================================================================
# Stop Command Error Tests
# =============================================================================

test_section "Stop Command (Not Running)"

run_test "stop when not running" "${WEBCTL_BINARY}" stop
assert_contains "${TEST_STDOUT}${TEST_STDERR}" "not running" "Error mentions not running"
assert_failure "${TEST_EXIT_CODE}" "Stop fails when daemon not running"

# =============================================================================
# Force Stop Tests
# =============================================================================

test_section "Force Stop Command"

# First, start a daemon to test force stop
"${WEBCTL_BINARY}" start --headless &
sleep 2

if is_daemon_running; then
  DAEMON_STARTED_BY_TEST=true
  run_test "force stop daemon" "${WEBCTL_BINARY}" stop --force
  assert_success "${TEST_EXIT_CODE}" "Force stop returns success"

  # Verify daemon stopped
  sleep 1
  if ! is_daemon_running; then
    log_success "Daemon stopped after force stop"
    increment_pass
  else
    log_failure "Daemon still running after force stop"
    increment_fail
  fi
  DAEMON_STARTED_BY_TEST=false
else
  log_message "Skipping force stop test - daemon didn't start"
fi

# Force stop on clean state
force_stop_daemon
run_test "force stop when nothing running" "${WEBCTL_BINARY}" stop --force
assert_success "${TEST_EXIT_CODE}" "Force stop succeeds even when nothing to clean"

# =============================================================================
# Custom Port Tests
# =============================================================================

test_section "Custom Port Configuration"

# Ensure clean state
force_stop_daemon

# Start daemon on custom port (browser connects to 9333 instead of 9222)
"${WEBCTL_BINARY}" start --headless --port 9333 &
sleep 2

# Verify daemon started successfully
if is_daemon_running; then
  log_success "Start with custom port: daemon started on port 9333"
  increment_pass
  DAEMON_STARTED_BY_TEST=true
else
  log_failure "Start with custom port: daemon failed to start"
  increment_fail
fi

# Clean up - force stop on custom port
run_test "force stop custom port" "${WEBCTL_BINARY}" stop --force --port 9333
assert_success "${TEST_EXIT_CODE}" "Force stop on custom port succeeds"
DAEMON_STARTED_BY_TEST=false

# =============================================================================
# Summary
# =============================================================================

test_summary
