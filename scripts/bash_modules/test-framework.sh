#!/usr/bin/env bash

# Test Framework Module
# ---------------------
# Provides test counters, run_test wrapper, and summary functions.
# Integrates with bash_modules for terminal output.

# Environment setup
set -o pipefail

# Determine script location and project root
BASH_MODULES_DIR="$(cd "${BASH_SOURCE[0]%/*}" || exit 1; pwd)"
PROJECT_ROOT="$(cd "${BASH_MODULES_DIR}/../.." || exit 1; pwd)"

# Import bash modules
if [[ ! -f "${BASH_MODULES_DIR}/terminal.sh" ]]; then
  echo "ERROR: terminal.sh module not found at ${BASH_MODULES_DIR}" >&2
  return 1
fi
source "${BASH_MODULES_DIR}/terminal.sh"

if [[ ! -f "${BASH_MODULES_DIR}/verify.sh" ]]; then
  echo "ERROR: verify.sh module not found at ${BASH_MODULES_DIR}" >&2
  return 1
fi
source "${BASH_MODULES_DIR}/verify.sh"

# Test counters
TEST_PASS=0
TEST_FAIL=0
TEST_TOTAL=0

# Test output capture variables
TEST_STDOUT=""
TEST_STDERR=""
TEST_EXIT_CODE=0
TEST_DURATION=""

# Current test name for context in assertions
CURRENT_TEST_NAME=""

function reset_test_counters() {
  # reset_test_counters
  # Resets all test counters to zero
  TEST_PASS=0
  TEST_FAIL=0
  TEST_TOTAL=0
}

function increment_pass() {
  # increment_pass
  # Increments pass and total counters
  ((TEST_PASS++))
  ((TEST_TOTAL++))
}

function increment_fail() {
  # increment_fail
  # Increments fail and total counters
  ((TEST_FAIL++))
  ((TEST_TOTAL++))
}

function run_command() {
  # run_command "description" command [args...]
  # Internal helper that runs a command and captures output.
  # Sets TEST_STDOUT, TEST_STDERR, TEST_EXIT_CODE, TEST_DURATION.
  # Returns the exit code of the command.
  # Does NOT update test counters - used by both run_test and run_setup.

  if [[ $# -lt 2 ]]; then
    log_error "ERROR: run_command requires description and command"
    return 1
  fi

  local description="${1}"
  shift
  local cmd=("$@")

  CURRENT_TEST_NAME="${description}"

  # Create temp files for output capture
  local stdout_file stderr_file
  stdout_file=$(mktemp)
  stderr_file=$(mktemp)

  # Capture start time
  local start_time end_time
  start_time=$(date +%s.%N 2>/dev/null || date +%s)

  # Run command and capture outputs
  # Save errexit state and disable it for command execution
  local errexit_was_set=false
  if [[ $- == *e* ]]; then
    errexit_was_set=true
  fi
  set +e
  "${cmd[@]}" >"${stdout_file}" 2>"${stderr_file}"
  TEST_EXIT_CODE=$?
  # Restore errexit state
  if [[ "${errexit_was_set}" == "true" ]]; then
    set -e
  fi

  # Capture end time and calculate duration
  end_time=$(date +%s.%N 2>/dev/null || date +%s)

  # Calculate duration (handle both GNU and BSD date)
  if command -v bc >/dev/null 2>&1; then
    TEST_DURATION=$(echo "${end_time} - ${start_time}" | bc)
  else
    # Fallback to integer seconds
    TEST_DURATION=$((${end_time%.*} - ${start_time%.*}))
  fi

  # Read captured output
  TEST_STDOUT=$(cat "${stdout_file}")
  TEST_STDERR=$(cat "${stderr_file}")

  # Clean up temp files
  rm -f "${stdout_file}" "${stderr_file}"

  return ${TEST_EXIT_CODE}
}

function run_test() {
  # run_test "test name" command [args...]
  # Runs a command, captures stdout, stderr, exit code, and timing.
  # Sets TEST_STDOUT, TEST_STDERR, TEST_EXIT_CODE, TEST_DURATION.
  # Returns the exit code of the command.

  run_command "$@"
}

function run_setup() {
  # run_setup "setup description" command [args...]
  # Like run_test but for setup steps - does NOT count toward test totals.
  # Captures output but only logs on failure.
  # Returns the exit code of the command.

  if [[ $# -lt 2 ]]; then
    log_error "ERROR: run_setup requires description and command"
    return 1
  fi

  local description="${1}"

  run_command "$@"
  local exit_code=${TEST_EXIT_CODE}

  if [[ ${exit_code} -ne 0 ]]; then
    log_failure "Setup failed: ${description}"
    if [[ -n "${TEST_STDERR}" ]]; then
      log_message "    stderr: ${TEST_STDERR:0:200}"
    fi
  fi

  return ${exit_code}
}

function run_setup_required() {
  # run_setup_required "setup description" command [args...]
  # Like run_setup but ABORTS the test suite if setup fails.
  # Use this for critical setup steps that must succeed.

  run_setup "$@"
  local exit_code=$?

  if [[ ${exit_code} -ne 0 ]]; then
    log_error "FATAL: Required setup failed, aborting test suite"
    log_error "       Failed step: ${1}"
    exit 1
  fi

  return 0
}

function require_var() {
  # require_var "VAR_NAME" ["error message"]
  # Checks that a variable is set and non-empty.
  # Aborts with error if not set.

  local var_name="${1}"
  local message="${2:-Required variable ${var_name} is not set}"

  if [[ -z "${!var_name:-}" ]]; then
    log_error "FATAL: ${message}"
    exit 1
  fi
}

function run_test_expect_fail() {
  # run_test_expect_fail "test name" command [args...]
  # Like run_test, but expects the command to fail.
  # Returns 0 if command fails, 1 if it succeeds.

  run_test "$@"
  local actual_exit=${TEST_EXIT_CODE}

  if [[ ${actual_exit} -ne 0 ]]; then
    return 0
  else
    return 1
  fi
}

function test_section() {
  # test_section "section name"
  # Prints a section header for grouping related tests
  local section_name="${1}"
  log_heading "${section_name}"
}

function test_case() {
  # test_case "test case name"
  # Prints a test case header
  local case_name="${1}"
  log_subheading "${case_name}"
}

function test_summary() {
  # test_summary
  # Prints final pass/fail counts and returns appropriate exit code

  log_fullline
  log_newline

  if [[ ${TEST_FAIL} -eq 0 ]]; then
    log_success "All tests passed: ${TEST_PASS}/${TEST_TOTAL}"
    return 0
  else
    log_failure "Tests failed: ${TEST_FAIL}/${TEST_TOTAL}"
    log_message "  Passed: ${TEST_PASS}"
    log_message "  Failed: ${TEST_FAIL}"
    return 1
  fi
}

function get_test_stats() {
  # get_test_stats
  # Outputs test statistics as "pass:fail:total"
  echo "${TEST_PASS}:${TEST_FAIL}:${TEST_TOTAL}"
}
