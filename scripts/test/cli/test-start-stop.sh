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

# Test constants
readonly DAEMON_START_WAIT=2
readonly DAEMON_STOP_WAIT=1

# Ensure clean state before tests
force_stop_daemon

# =============================================================================
# Status Command Tests (Not Running)
# =============================================================================

test_section "Status Command (Not Running)"

run_test "status when not running" "${WEBCTL_BINARY}" status
assert_success "${TEST_EXIT_CODE}" "Status returns success even when not running"
assert_contains "${TEST_STDOUT}" "Not running" "Output shows not running"

# Test: status --json when not running
run_test "status --json when not running" "${WEBCTL_BINARY}" status --json
assert_success "${TEST_EXIT_CODE}" "status --json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
assert_json_field "${TEST_STDOUT}" ".data.running" "false" "JSON running field is false"

# Test: status --no-color when not running
run_test "status --no-color not running" "${WEBCTL_BINARY}" status --no-color
assert_success "${TEST_EXIT_CODE}" "status --no-color returns success"
assert_contains "${TEST_STDOUT}" "Not running" "Output shows not running"
assert_not_contains "${TEST_STDOUT}" $'\e[' "Output has no ANSI codes"

# =============================================================================
# Start Command Tests
# =============================================================================

test_section "Start Command"

# Start daemon in headless mode
# The start command blocks, so we run it in background.
# --temp-profile keeps the test off the shared persistent profile.
"${WEBCTL_BINARY}" start --headless --temp-profile &
sleep "${DAEMON_START_WAIT}"

# Verify daemon started using assertion framework
if is_daemon_running; then
  TEST_EXIT_CODE=0
  DAEMON_STARTED_BY_TEST=true
else
  TEST_EXIT_CODE=1
fi
assert_success "${TEST_EXIT_CODE}" "Start command: daemon started successfully (headless)"

# Only continue with running tests if daemon started successfully
if [[ "${TEST_EXIT_CODE}" -ne 0 ]]; then
  log_failure "Cannot continue with running tests - daemon failed to start"
  test_summary
  exit 1
fi

# Note: We can't test start --json here because daemon is already running.
# The --json test will be done in a later section.

# =============================================================================
# Status Command Tests (Running)
# =============================================================================

test_section "Status Command (Running)"

run_test "status when running" "${WEBCTL_BINARY}" status
assert_success "${TEST_EXIT_CODE}" "Status returns success when running"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK when running"

# Test: status shows pid when running
run_test "status shows pid" "${WEBCTL_BINARY}" status
assert_success "${TEST_EXIT_CODE}" "status returns success"
assert_contains "${TEST_STDOUT}" "pid:" "Output contains pid"

# Test: status shows sessions/URL when running
# Note: daemon auto-navigates to about:blank on startup, so session already exists
run_test "status shows sessions" "${WEBCTL_BINARY}" status
assert_success "${TEST_EXIT_CODE}" "status returns success"
assert_contains "${TEST_STDOUT}" "sessions:" "Output contains sessions"

# Test: status shows URL in session list
run_test "status shows URL" "${WEBCTL_BINARY}" status
assert_success "${TEST_EXIT_CODE}" "status returns success"
assert_contains "${TEST_STDOUT}" "about:blank" "Output contains URL"

# Test: status --json when running
run_test "status --json when running" "${WEBCTL_BINARY}" status --json
assert_success "${TEST_EXIT_CODE}" "status --json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
assert_json_field "${TEST_STDOUT}" ".data.running" "true" "JSON running field is true"
assert_contains "${TEST_STDOUT}" "pid" "JSON contains pid field"

# Test: status --no-color when running
run_test "status --no-color running" "${WEBCTL_BINARY}" status --no-color
assert_success "${TEST_EXIT_CODE}" "status --no-color returns success"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"
assert_not_contains "${TEST_STDOUT}" $'\e[' "Output has no ANSI codes"

# =============================================================================
# Start Command Error Tests
# =============================================================================

test_section "Start Command (Already Running)"

# Try to start another daemon - this should fail immediately with an error
run_test "start when already running" "${WEBCTL_BINARY}" start --headless
assert_failure "${TEST_EXIT_CODE}" "Start fails when daemon already running"
assert_contains "${TEST_STDOUT}${TEST_STDERR}" "already running" "Error mentions already running"

# Test: start already running includes hint
run_test "start already running hint" "${WEBCTL_BINARY}" start --headless
assert_failure "${TEST_EXIT_CODE}" "start fails when already running"
assert_contains "${TEST_STDOUT}${TEST_STDERR}" "webctl stop" "Error includes stop hint"

# =============================================================================
# Stop Command Tests
# =============================================================================

test_section "Stop Command"

run_test "stop daemon (graceful)" "${WEBCTL_BINARY}" stop
assert_success "${TEST_EXIT_CODE}" "Stop returns success"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK on stop"

# Wait for daemon to fully stop
sleep "${DAEMON_STOP_WAIT}"

# Verify daemon stopped
run_test "verify daemon stopped" "${WEBCTL_BINARY}" status
assert_success "${TEST_EXIT_CODE}" "status returns success"
assert_contains "${TEST_STDOUT}" "Not running" "Daemon is no longer running"
DAEMON_STARTED_BY_TEST=false

# Test: stop --json output format (need to restart daemon first)
if start_daemon --headless; then
  run_test "stop --json output" "${WEBCTL_BINARY}" stop --json
  assert_success "${TEST_EXIT_CODE}" "stop --json returns success"
  assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
  assert_json_field "${TEST_STDOUT}" ".data.message" "daemon stopped" "JSON message field"
  sleep "${DAEMON_STOP_WAIT}"
  DAEMON_STARTED_BY_TEST=false
else
  log_failure "Failed to start daemon for stop --json test"
  increment_fail
fi

# =============================================================================
# Stop Command Error Tests
# =============================================================================

test_section "Stop Command (Not Running)"

run_test "stop when not running" "${WEBCTL_BINARY}" stop
assert_failure "${TEST_EXIT_CODE}" "Stop fails when daemon not running"
assert_contains "${TEST_STDOUT}${TEST_STDERR}" "not running" "Error mentions not running"

# =============================================================================
# Force Stop Tests
# =============================================================================

test_section "Force Stop Command"

# First, start a daemon to test force stop
if start_daemon --headless; then
  run_test "force stop daemon" "${WEBCTL_BINARY}" stop --force
  assert_success "${TEST_EXIT_CODE}" "Force stop returns success"

  # Verify daemon stopped
  sleep "${DAEMON_STOP_WAIT}"
  if ! is_daemon_running; then
    TEST_EXIT_CODE=0
  else
    TEST_EXIT_CODE=1
  fi
  assert_success "${TEST_EXIT_CODE}" "Daemon stopped after force stop"
  DAEMON_STARTED_BY_TEST=false
else
  log_failure "Failed to start daemon for force stop test"
  increment_fail
fi

# Force stop on clean state
force_stop_daemon
run_test "force stop when nothing running" "${WEBCTL_BINARY}" stop --force
assert_success "${TEST_EXIT_CODE}" "Force stop succeeds even when nothing to clean"

# Test: force stop reports cleanup actions
if start_daemon --headless; then
  run_test "force stop reports actions" "${WEBCTL_BINARY}" stop --force
  assert_success "${TEST_EXIT_CODE}" "force stop returns success"
  # Should contain at least one action (killed daemon, killed browser, removed socket, removed PID)
  assert_matches "killed|removed" "${TEST_STDOUT}" "Output reports cleanup action"
  sleep "${DAEMON_STOP_WAIT}"
  DAEMON_STARTED_BY_TEST=false
else
  log_failure "Failed to start daemon for action reporting test"
  increment_fail
fi

# Test: force stop --json reports actions array
if start_daemon --headless; then
  run_test "force stop --json actions" "${WEBCTL_BINARY}" stop --force --json
  assert_success "${TEST_EXIT_CODE}" "force stop --json returns success"
  assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
  assert_contains "${TEST_STDOUT}" "actions" "JSON contains actions array"
  sleep "${DAEMON_STOP_WAIT}"
  DAEMON_STARTED_BY_TEST=false
else
  log_failure "Failed to start daemon for JSON action test"
  increment_fail
fi

# =============================================================================
# Custom Port Tests
# =============================================================================

test_section "Custom Port Configuration"

# Ensure clean state
force_stop_daemon

# Start daemon on custom port (browser connects to 9333 instead of 9222)
"${WEBCTL_BINARY}" start --headless --port 9333 --temp-profile &
sleep "${DAEMON_START_WAIT}"

# Verify daemon started successfully
if is_daemon_running; then
  TEST_EXIT_CODE=0
  DAEMON_STARTED_BY_TEST=true
else
  TEST_EXIT_CODE=1
fi
assert_success "${TEST_EXIT_CODE}" "Start with custom port: daemon started on port 9333"

# Clean up - force stop on custom port (only if daemon started)
if [[ "${TEST_EXIT_CODE}" -eq 0 ]]; then
  run_test "force stop custom port" "${WEBCTL_BINARY}" stop --force --port 9333
  assert_success "${TEST_EXIT_CODE}" "Force stop on custom port succeeds"
  DAEMON_STARTED_BY_TEST=false
fi

# =============================================================================
# Start Command JSON Output Tests
# =============================================================================

test_section "Start Command JSON Output"

# Ensure clean state
force_stop_daemon

# Test: start --json output format
# Start command blocks, so we capture output to a temp file and run in background
JSON_OUTPUT_FILE=$(create_temp_file)
"${WEBCTL_BINARY}" start --headless --json --temp-profile >"${JSON_OUTPUT_FILE}" 2>&1 &

# Wait for daemon to start and output to be written
sleep "${DAEMON_START_WAIT}"

# Verify daemon started
if is_daemon_running; then
  TEST_EXIT_CODE=0
  DAEMON_STARTED_BY_TEST=true
else
  TEST_EXIT_CODE=1
fi
assert_success "${TEST_EXIT_CODE}" "start --json: daemon started"

if [[ "${TEST_EXIT_CODE}" -eq 0 ]]; then
  # Read the captured JSON output
  JSON_OUTPUT=$(cat "${JSON_OUTPUT_FILE}")

  # Validate JSON output was captured
  if [[ -n "${JSON_OUTPUT}" ]]; then
    TEST_EXIT_CODE=0
  else
    TEST_EXIT_CODE=1
  fi
  assert_success "${TEST_EXIT_CODE}" "start --json: captured output"

  # Parse and validate JSON fields
  if [[ "${TEST_EXIT_CODE}" -eq 0 ]]; then
    TEST_STDOUT="${JSON_OUTPUT}"
    assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
    assert_json_field "${TEST_STDOUT}" ".data.message" "daemon ready" "JSON message field"
    assert_json_field "${TEST_STDOUT}" ".data.port" "9222" "JSON port field"
  fi

  # Clean up (temp file will be auto-cleaned by test framework)
  run_test "cleanup stop --force" "${WEBCTL_BINARY}" stop --force
  assert_success "${TEST_EXIT_CODE}" "Cleanup succeeded"
  sleep "${DAEMON_STOP_WAIT}"
  DAEMON_STARTED_BY_TEST=false
fi

# =============================================================================
# Start Failure Contract Tests (no success line before error)
# =============================================================================

test_section "Start Failure Contract (port in use)"

# Regression guard for the success-before-error contract: a start that fails
# before the daemon is serving IPC must emit only the failure. We occupy the
# default CDP port so browser.Start takes the fail-fast ErrPortInUse branch and
# returns before any browser process spawns - a portable pre-readiness failure
# that needs no display server or Chrome.
force_stop_daemon

if command -v python3 >/dev/null 2>&1; then
  # Wait for any active listener on 9222 (a browser from a previous section still
  # tearing down) to exit. SO_REUSEADDR lets the holder bind over a leftover
  # TIME_WAIT socket but not over a live LISTEN (that needs SO_REUSEPORT), so an
  # un-dead prior listener would make the holder's bind fail and self-skip the test.
  attempts=0
  while [[ ${attempts} -lt 40 ]]; do
    lsof -iTCP:9222 -sTCP:LISTEN -P -n >/dev/null 2>&1 || break
    sleep 0.25
    ((attempts++))
  done

  # Hold 127.0.0.1:9222 with a listening socket. SO_REUSEADDR mirrors Go's
  # net.Listen so the holder can bind over a TIME_WAIT socket left by a Chrome
  # killed in a previous section, exactly as isPortAvailable would. With the
  # holder in LISTEN, isPortAvailable fails (two live listens need SO_REUSEPORT,
  # which neither sets), so the explicit --port 9222 start hits the fail-fast
  # ErrPortInUse branch before any browser process spawns.
  python3 -c 'import socket,time; s=socket.socket(); s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1); s.bind(("127.0.0.1",9222)); s.listen(1); time.sleep(60)' &
  PORT_HOLDER_PID=$!

  # Confirm the holder is actually listening before issuing the start.
  attempts=0
  holder_ready=false
  while [[ ${attempts} -lt 40 ]]; do
    if lsof -iTCP:9222 -sTCP:LISTEN -P -n >/dev/null 2>&1; then
      holder_ready=true
      break
    fi
    sleep 0.25
    ((attempts++))
  done

  if [[ "${holder_ready}" == "true" ]]; then
    # holder_ready confirms a live listener on 9222, so start is guaranteed to
    # fast-fail via ErrPortInUse before spawning a browser - that is the contract
    # under test. The timeout below is only a backstop against an infinite hang
    # should that guarantee ever regress, so it is optional: fall back to running
    # unguarded where neither timeout nor gtimeout exists (e.g. stock macOS).
    timeout_cmd=""
    if command -v timeout >/dev/null 2>&1; then
      timeout_cmd="timeout 15"
    elif command -v gtimeout >/dev/null 2>&1; then
      timeout_cmd="gtimeout 15"
    fi
    run_test "start --json with port held" ${timeout_cmd} "${WEBCTL_BINARY}" start --headless --json --port 9222 --temp-profile
    assert_failure "${TEST_EXIT_CODE}" "Start fails when CDP port is already in use"
    assert_not_contains "${TEST_STDOUT}" '"ok"' "stdout carries no machine-readable result"
    assert_json_field "${TEST_STDERR}" ".ok" "false" "stderr carries the single ok:false error"
  else
    log_failure "Could not hold port 9222 for start-failure test"
    increment_fail
  fi

  kill "${PORT_HOLDER_PID}" 2>/dev/null || true
  wait "${PORT_HOLDER_PID}" 2>/dev/null || true
  force_stop_daemon
else
  log_message "Skipping port-in-use start failure test (python3 not available)"
fi

# =============================================================================
# Persistent Default Profile Tests
# =============================================================================

test_section "Persistent Default Profile"

# The XDG sandbox (setup_cleanup_trap) points the persistent default profile
# here. These tests opt into the default explicitly (no profile flag).
PROFILE_DIR="${XDG_DATA_HOME}/webctl/profile"
SENTINEL_FILE="${PROFILE_DIR}/webctl-test-sentinel"

force_stop_daemon

# Start with the default persistent profile (no profile flag)
"${WEBCTL_BINARY}" start --headless &
attempts=0
while [[ ${attempts} -lt 20 ]]; do
  is_daemon_running && break
  sleep 0.5
  ((attempts++))
done
if is_daemon_running; then TEST_EXIT_CODE=0; DAEMON_STARTED_BY_TEST=true; else TEST_EXIT_CODE=1; fi
assert_success "${TEST_EXIT_CODE}" "Default profile: daemon started"

# The resolver must have created the persistent profile directory
assert_dir_exists "${PROFILE_DIR}" "Default profile: directory created under XDG_DATA_HOME"

# Write a sentinel into the profile directory
echo "persistent-profile-sentinel" >"${SENTINEL_FILE}"
assert_file_exists "${SENTINEL_FILE}" "Default profile: sentinel written"

# Graceful stop must NOT delete the persistent profile
run_test "default profile stop" "${WEBCTL_BINARY}" stop
assert_success "${TEST_EXIT_CODE}" "Default profile: graceful stop succeeds"
attempts=0
while [[ ${attempts} -lt 10 ]]; do
  is_daemon_running || break
  sleep 0.5
  ((attempts++))
done
DAEMON_STARTED_BY_TEST=false

assert_file_exists "${SENTINEL_FILE}" "Default profile: sentinel survives stop"
PROFILE_CONTENTS=$(ls -A "${PROFILE_DIR}" 2>/dev/null)
assert_not_empty "${PROFILE_CONTENTS}" "Default profile: directory non-empty after stop"

# Let the browser release the CDP port before relaunching on the same port.
sleep "${DAEMON_STOP_WAIT}"

# Restart with the default profile and confirm the same directory is reused
"${WEBCTL_BINARY}" start --headless &
attempts=0
while [[ ${attempts} -lt 20 ]]; do
  is_daemon_running && break
  sleep 0.5
  ((attempts++))
done
if is_daemon_running; then TEST_EXIT_CODE=0; DAEMON_STARTED_BY_TEST=true; else TEST_EXIT_CODE=1; fi
assert_success "${TEST_EXIT_CODE}" "Default profile: daemon restarted"
assert_file_exists "${SENTINEL_FILE}" "Default profile: sentinel reused across restart"

# Recover from an unclean exit. 'stop --force' SIGKILLs the browser, leaving a
# crashed exit_type and a stale singleton lock in the profile. The next start on
# the same persistent profile must come up without manual lock removal. This
# guards the stale-lock auto-recovery the implementation relies on; the
# crash-restore bubble it also suppresses is UI and cannot be asserted headlessly.
force_stop_daemon
sleep "${DAEMON_STOP_WAIT}"

"${WEBCTL_BINARY}" start --headless &
attempts=0
while [[ ${attempts} -lt 20 ]]; do
  is_daemon_running && break
  sleep 0.5
  ((attempts++))
done
if is_daemon_running; then TEST_EXIT_CODE=0; DAEMON_STARTED_BY_TEST=true; else TEST_EXIT_CODE=1; fi
assert_success "${TEST_EXIT_CODE}" "Default profile: recovers after unclean force-kill (stale lock)"
assert_file_exists "${SENTINEL_FILE}" "Default profile: profile preserved across unclean exit"

run_test "default profile cleanup stop" "${WEBCTL_BINARY}" stop
attempts=0
while [[ ${attempts} -lt 10 ]]; do
  is_daemon_running || break
  sleep 0.5
  ((attempts++))
done
DAEMON_STARTED_BY_TEST=false

# =============================================================================
# Temp Profile Non-Persistence Tests
# =============================================================================

test_section "Temp Profile Non-Persistence"

force_stop_daemon

# Snapshot existing webctl temp profiles, then start one and diff to find ours
TEMP_BEFORE=$(ls -d /tmp/webctl-chrome-* 2>/dev/null | sort)
"${WEBCTL_BINARY}" start --headless --temp-profile &
attempts=0
while [[ ${attempts} -lt 20 ]]; do
  is_daemon_running && break
  sleep 0.5
  ((attempts++))
done
if is_daemon_running; then TEST_EXIT_CODE=0; DAEMON_STARTED_BY_TEST=true; else TEST_EXIT_CODE=1; fi
assert_success "${TEST_EXIT_CODE}" "Temp profile: daemon started"

TEMP_AFTER=$(ls -d /tmp/webctl-chrome-* 2>/dev/null | sort)
TEMP_PROFILE_DIR=$(comm -13 <(echo "${TEMP_BEFORE}") <(echo "${TEMP_AFTER}") | grep -v '^$' | head -1)
assert_not_empty "${TEMP_PROFILE_DIR}" "Temp profile: throwaway directory created"
assert_dir_exists "${TEMP_PROFILE_DIR}" "Temp profile: directory exists while running"

# Graceful stop must delete the throwaway profile. The browser teardown that
# removes the directory runs after the daemon acknowledges shutdown, so poll for
# removal rather than asserting immediately.
run_test "temp profile stop" "${WEBCTL_BINARY}" stop
assert_success "${TEST_EXIT_CODE}" "Temp profile: graceful stop succeeds"
attempts=0
while [[ ${attempts} -lt 20 ]]; do
  [[ -d "${TEMP_PROFILE_DIR}" ]] || break
  sleep 0.5
  ((attempts++))
done
DAEMON_STARTED_BY_TEST=false

assert_dir_not_exists "${TEMP_PROFILE_DIR}" "Temp profile: directory deleted after stop"

# =============================================================================
# Summary
# =============================================================================

test_summary
