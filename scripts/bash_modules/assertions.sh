#!/usr/bin/env bash

# Assertions Module
# -----------------
# Provides assert_* functions for test validation.
# All assertions output pass/fail messages and update test counters.

# Environment setup
set -o pipefail

# Determine script location
BASH_MODULES_DIR="$(cd "${BASH_SOURCE[0]%/*}" || exit 1; pwd)"

# Import test-framework if not already loaded (for counters)
if ! declare -f increment_pass >/dev/null 2>&1; then
  if [[ ! -f "${BASH_MODULES_DIR}/test-framework.sh" ]]; then
    echo "ERROR: test-framework.sh not found at ${BASH_MODULES_DIR}" >&2
    return 1
  fi
  source "${BASH_MODULES_DIR}/test-framework.sh"
fi

# Exit Code Assertions
# -----------------------------------------------------------------------------

function assert_exit_code() {
  # assert_exit_code expected actual [message]
  # Asserts that exit code matches expected value

  local expected="${1}"
  local actual="${2}"
  local message="${3:-Exit code}"

  if [[ "${actual}" -eq "${expected}" ]]; then
    log_success "${message}: expected ${expected}, got ${actual}"
    increment_pass
    return 0
  else
    log_failure "${message}: expected ${expected}, got ${actual}"
    increment_fail
    return 1
  fi
}

function assert_success() {
  # assert_success actual [message]
  # Asserts that exit code is 0

  local actual="${1}"
  local message="${2:-Command succeeded}"

  assert_exit_code 0 "${actual}" "${message}"
}

function assert_failure() {
  # assert_failure actual [message]
  # Asserts that exit code is non-zero

  local actual="${1}"
  local message="${2:-Command failed}"

  if [[ "${actual}" -ne 0 ]]; then
    log_success "${message}: exit code ${actual} (non-zero)"
    increment_pass
    return 0
  else
    log_failure "${message}: expected non-zero, got 0"
    increment_fail
    return 1
  fi
}

# String Assertions
# -----------------------------------------------------------------------------

function assert_equals() {
  # assert_equals expected actual [message]
  # Asserts that two strings are equal

  local expected="${1}"
  local actual="${2}"
  local message="${3:-Values equal}"

  if [[ "${actual}" == "${expected}" ]]; then
    log_success "${message}"
    increment_pass
    return 0
  else
    log_failure "${message}"
    log_message "    Expected: '${expected}'"
    log_message "    Actual:   '${actual}'"
    increment_fail
    return 1
  fi
}

function assert_not_equals() {
  # assert_not_equals unexpected actual [message]
  # Asserts that two strings are not equal

  local unexpected="${1}"
  local actual="${2}"
  local message="${3:-Values not equal}"

  if [[ "${actual}" != "${unexpected}" ]]; then
    log_success "${message}"
    increment_pass
    return 0
  else
    log_failure "${message}: values should differ but both are '${actual}'"
    increment_fail
    return 1
  fi
}

function assert_contains() {
  # assert_contains haystack needle [message]
  # Asserts that haystack contains needle

  local haystack="${1}"
  local needle="${2}"
  local message="${3:-String contains}"

  if [[ "${haystack}" == *"${needle}"* ]]; then
    log_success "${message}: found '${needle}'"
    increment_pass
    return 0
  else
    log_failure "${message}: '${needle}' not found"
    log_message "    In: '${haystack:0:200}'"
    increment_fail
    return 1
  fi
}

function assert_not_contains() {
  # assert_not_contains haystack needle [message]
  # Asserts that haystack does not contain needle

  local haystack="${1}"
  local needle="${2}"
  local message="${3:-String does not contain}"

  if [[ "${haystack}" != *"${needle}"* ]]; then
    log_success "${message}: '${needle}' not present"
    increment_pass
    return 0
  else
    log_failure "${message}: '${needle}' was found but should not be"
    increment_fail
    return 1
  fi
}

function assert_matches() {
  # assert_matches pattern actual [message]
  # Asserts that actual matches regex pattern

  local pattern="${1}"
  local actual="${2}"
  local message="${3:-String matches pattern}"

  if [[ "${actual}" =~ ${pattern} ]]; then
    log_success "${message}"
    increment_pass
    return 0
  else
    log_failure "${message}"
    log_message "    Pattern: '${pattern}'"
    log_message "    Actual:  '${actual:0:200}'"
    increment_fail
    return 1
  fi
}

function assert_empty() {
  # assert_empty value [message]
  # Asserts that value is empty

  local value="${1}"
  local message="${2:-Value is empty}"

  if [[ -z "${value}" ]]; then
    log_success "${message}"
    increment_pass
    return 0
  else
    log_failure "${message}: expected empty, got '${value:0:100}'"
    increment_fail
    return 1
  fi
}

function assert_not_empty() {
  # assert_not_empty value [message]
  # Asserts that value is not empty

  local value="${1}"
  local message="${2:-Value is not empty}"

  if [[ -n "${value}" ]]; then
    log_success "${message}"
    increment_pass
    return 0
  else
    log_failure "${message}: expected non-empty value"
    increment_fail
    return 1
  fi
}

# JSON Assertions
# -----------------------------------------------------------------------------

function assert_json_valid() {
  # assert_json_valid json [message]
  # Asserts that string is valid JSON

  local json="${1}"
  local message="${2:-Valid JSON}"

  if echo "${json}" | jq empty >/dev/null 2>&1; then
    log_success "${message}"
    increment_pass
    return 0
  else
    log_failure "${message}: invalid JSON"
    log_message "    Value: '${json:0:200}'"
    increment_fail
    return 1
  fi
}

function assert_json_field() {
  # assert_json_field json jq_path expected [message]
  # Asserts that JSON field at jq_path equals expected value

  local json="${1}"
  local jq_path="${2}"
  local expected="${3}"
  local message="${4:-JSON field}"

  if ! echo "${json}" | jq empty >/dev/null 2>&1; then
    log_failure "${message}: invalid JSON input"
    increment_fail
    return 1
  fi

  local actual
  actual=$(echo "${json}" | jq -r "${jq_path}" 2>/dev/null)

  if [[ "${actual}" == "${expected}" ]]; then
    log_success "${message}: ${jq_path} = '${expected}'"
    increment_pass
    return 0
  else
    log_failure "${message}: ${jq_path}"
    log_message "    Expected: '${expected}'"
    log_message "    Actual:   '${actual}'"
    increment_fail
    return 1
  fi
}

function assert_json_field_exists() {
  # assert_json_field_exists json jq_path [message]
  # Asserts that JSON field at jq_path exists and is not null

  local json="${1}"
  local jq_path="${2}"
  local message="${3:-JSON field exists}"

  if ! echo "${json}" | jq empty >/dev/null 2>&1; then
    log_failure "${message}: invalid JSON input"
    increment_fail
    return 1
  fi

  local value
  value=$(echo "${json}" | jq -r "${jq_path}" 2>/dev/null)

  if [[ "${value}" != "null" ]] && [[ -n "${value}" ]]; then
    log_success "${message}: ${jq_path} exists"
    increment_pass
    return 0
  else
    log_failure "${message}: ${jq_path} not found or null"
    increment_fail
    return 1
  fi
}

function assert_json_array_length() {
  # assert_json_array_length json jq_path expected_length [message]
  # Asserts that JSON array at jq_path has expected length

  local json="${1}"
  local jq_path="${2}"
  local expected="${3}"
  local message="${4:-JSON array length}"

  if ! echo "${json}" | jq empty >/dev/null 2>&1; then
    log_failure "${message}: invalid JSON input"
    increment_fail
    return 1
  fi

  local actual
  actual=$(echo "${json}" | jq -r "${jq_path} | length" 2>/dev/null)

  if [[ "${actual}" == "${expected}" ]]; then
    log_success "${message}: ${jq_path} has ${expected} elements"
    increment_pass
    return 0
  else
    log_failure "${message}: ${jq_path} length"
    log_message "    Expected: ${expected}"
    log_message "    Actual:   ${actual}"
    increment_fail
    return 1
  fi
}

# File Assertions
# -----------------------------------------------------------------------------

function assert_file_exists() {
  # assert_file_exists path [message]
  # Asserts that file exists

  local path="${1}"
  local message="${2:-File exists}"

  if [[ -f "${path}" ]]; then
    log_success "${message}: ${path}"
    increment_pass
    return 0
  else
    log_failure "${message}: ${path} not found"
    increment_fail
    return 1
  fi
}

function assert_file_not_exists() {
  # assert_file_not_exists path [message]
  # Asserts that file does not exist

  local path="${1}"
  local message="${2:-File does not exist}"

  if [[ ! -f "${path}" ]]; then
    log_success "${message}: ${path}"
    increment_pass
    return 0
  else
    log_failure "${message}: ${path} exists but should not"
    increment_fail
    return 1
  fi
}

function assert_file_contains() {
  # assert_file_contains path needle [message]
  # Asserts that file contains needle

  local path="${1}"
  local needle="${2}"
  local message="${3:-File contains}"

  if [[ ! -f "${path}" ]]; then
    log_failure "${message}: file ${path} not found"
    increment_fail
    return 1
  fi

  if grep -q "${needle}" "${path}"; then
    log_success "${message}: '${needle}' in ${path}"
    increment_pass
    return 0
  else
    log_failure "${message}: '${needle}' not in ${path}"
    increment_fail
    return 1
  fi
}

function assert_dir_exists() {
  # assert_dir_exists path [message]
  # Asserts that directory exists

  local path="${1}"
  local message="${2:-Directory exists}"

  if [[ -d "${path}" ]]; then
    log_success "${message}: ${path}"
    increment_pass
    return 0
  else
    log_failure "${message}: ${path} not found"
    increment_fail
    return 1
  fi
}

# Numeric Assertions
# -----------------------------------------------------------------------------

function assert_greater_than() {
  # assert_greater_than expected actual [message]
  # Asserts that actual > expected

  local expected="${1}"
  local actual="${2}"
  local message="${3:-Value greater than}"

  if [[ "${actual}" -gt "${expected}" ]]; then
    log_success "${message}: ${actual} > ${expected}"
    increment_pass
    return 0
  else
    log_failure "${message}: ${actual} is not greater than ${expected}"
    increment_fail
    return 1
  fi
}

function assert_less_than() {
  # assert_less_than expected actual [message]
  # Asserts that actual < expected

  local expected="${1}"
  local actual="${2}"
  local message="${3:-Value less than}"

  if [[ "${actual}" -lt "${expected}" ]]; then
    log_success "${message}: ${actual} < ${expected}"
    increment_pass
    return 0
  else
    log_failure "${message}: ${actual} is not less than ${expected}"
    increment_fail
    return 1
  fi
}
