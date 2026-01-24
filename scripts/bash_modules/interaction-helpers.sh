#!/usr/bin/env bash

# Interaction Test Helpers
# -------------------------
# Provides helper functions for common interactions and state verification
# in CLI interaction tests. Reduces duplication and improves readability.

# Environment setup
set -o pipefail

# Scroll Position Tolerances
# -----------------------------------------------------------------------------
# Modern browsers may have sub-pixel rendering and different font metrics
# across platforms, leading to slight variations in scroll positions.
#
# TOP_SCROLL_TOLERANCE: 5px - Used for "at top" checks (scrollY ≈ 0)
#   Rationale: Most browsers return exactly 0, but some may overshoot slightly
#   during scroll animations or due to rounding. 5px is imperceptible to users.
#
# SCROLL_TOLERANCE: 10px - Used for specific position checks (e.g., scrollY ≈ 500)
#   Rationale: Scroll positions are affected by:
#   - Font rendering differences across OSes
#   - Browser chrome (scrollbar width) variations
#   - Subpixel rendering in high-DPI displays
#   - Scroll animation timing/easing
#   10px tolerance accommodates these variations while ensuring scroll happened.
readonly TOP_SCROLL_TOLERANCE=5
readonly SCROLL_TOLERANCE=10

# Timeout Configuration
# -----------------------------------------------------------------------------
# All timeout values documented with rationale:
#
# SERVE_AUTO_NAV_TIMEOUT: 15s - Wait for serve command's auto-navigation
#   Rationale: Server startup + initial navigation can be slow on CI/cold start.
#   Covers: DNS resolution, server binding, browser navigation, page load
#
# DEFAULT_READY_TIMEOUT: 5s - Standard page ready/eval condition wait
#   Rationale: Most page loads complete within 2-3s, 5s provides headroom
#   for slower systems/CI environments
#
# FORCED_FAILURE_TIMEOUT: 2s - Deliberate timeout for error testing
#   Rationale: Long enough to prove condition fails, short enough to not
#   waste time in test runs
#
# DAEMON_HEALTH_CHECK_TIMEOUT: 10s - Post-recovery navigation timeout
#   Rationale: After daemon restart, needs extra time for full initialization
readonly SERVE_AUTO_NAV_TIMEOUT="15s"
readonly DEFAULT_READY_TIMEOUT="5s"
readonly FORCED_FAILURE_TIMEOUT="2s"
readonly DAEMON_HEALTH_CHECK_TIMEOUT="10s"

# JavaScript Evaluation Helpers
# -----------------------------------------------------------------------------

function js_escape() {
  # js_escape string
  # Escapes a string for safe inclusion in JavaScript single-quoted string literals
  #
  # Args:
  #   string: String to escape
  #
  # Returns: Escaped string safe for use in JavaScript 'string' literals
  #
  # Security: Prevents JavaScript injection by escaping special characters
  # Order matters: backslash must be escaped first to avoid double-escaping

  local str="${1}"
  # Escape backslashes first (must be first to avoid double-escaping)
  str="${str//\\/\\\\}"
  # Escape single quotes
  str="${str//\'/\\\'}"
  # Escape newlines
  str="${str//$'\n'/\\n}"
  # Escape carriage returns
  str="${str//$'\r'/\\r}"
  # Escape tabs
  str="${str//$'\t'/\\t}"
  echo "${str}"
}

function eval_element_visible_in_viewport() {
  # eval_element_visible_in_viewport selector
  # Returns JavaScript expression that evaluates to true if element is visible
  # in the current viewport.
  #
  # Args:
  #   selector: CSS selector for the element
  #
  # Usage:
  #   "${WEBCTL_BINARY}" ready --eval "$(eval_element_visible_in_viewport '#marker')"

  local selector
  selector="$(js_escape "${1}")"
  cat <<EOF
(() => {
  const el = document.querySelector('${selector}');
  if (!el) return false;
  const rect = el.getBoundingClientRect();
  return rect.top >= 0 && rect.top < window.innerHeight;
})()
EOF
}

function eval_scroll_at_top() {
  # eval_scroll_at_top
  # Returns JavaScript expression that checks if page is scrolled to top
  # (within TOP_SCROLL_TOLERANCE)

  echo "window.scrollY < ${TOP_SCROLL_TOLERANCE}"
}

function eval_scroll_near_position() {
  # eval_scroll_near_position position
  # Returns JavaScript expression that checks if scroll position is near
  # the target position (within SCROLL_TOLERANCE)
  #
  # Args:
  #   position: Target Y scroll position in pixels

  local position="${1}"
  echo "Math.abs(window.scrollY - ${position}) < ${SCROLL_TOLERANCE}"
}

function eval_element_has_attribute() {
  # eval_element_has_attribute selector attribute expected_value
  # Returns JavaScript expression to check element attribute value
  #
  # Args:
  #   selector: CSS selector
  #   attribute: Attribute name (e.g., 'data-clicked', 'value')
  #   expected_value: Expected attribute value

  local selector attribute expected_value
  selector="$(js_escape "${1}")"
  attribute="$(js_escape "${2}")"
  expected_value="$(js_escape "${3}")"
  cat <<EOF
document.querySelector('${selector}').getAttribute('${attribute}') === '${expected_value}'
EOF
}

function eval_element_property() {
  # eval_element_property selector property
  # Returns JavaScript expression to get element property value
  #
  # Args:
  #   selector: CSS selector
  #   property: Property name (e.g., 'value', 'checked', 'classList')
  #
  # Note: property is used as a JavaScript identifier, not a string,
  # so it's not escaped (e.g., 'value', 'checked' are direct property access)

  local selector
  selector="$(js_escape "${1}")"
  local property="${2}"
  echo "document.querySelector('${selector}').${property}"
}

function eval_active_element_id() {
  # eval_active_element_id
  # Returns JavaScript expression to get the ID of the currently focused element

  echo "document.activeElement.id"
}

function eval_url_contains() {
  # eval_url_contains pattern
  # Returns JavaScript expression to check if current URL contains pattern
  #
  # Args:
  #   pattern: String to search for in window.location.href

  local pattern
  pattern="$(js_escape "${1}")"
  echo "window.location.href.includes('${pattern}')"
}

# Input State Helpers
# -----------------------------------------------------------------------------

function clear_input_value() {
  # clear_input_value selector
  # Clears an input field completely by setting its value to empty string
  #
  # Args:
  #   selector: CSS selector for input element

  local selector escaped_selector
  selector="${1}"
  escaped_selector="$(js_escape "${selector}")"
  run_test "clear ${selector}" "${WEBCTL_BINARY}" eval "document.querySelector('${escaped_selector}').value = ''; 'cleared'"
}

function get_input_value() {
  # get_input_value selector
  # Gets the current value of an input element
  #
  # Args:
  #   selector: CSS selector for input element
  #
  # Returns: The input value in TEST_STDOUT

  local selector escaped_selector
  selector="${1}"
  escaped_selector="$(js_escape "${selector}")"
  run_test "get value of ${selector}" "${WEBCTL_BINARY}" eval "document.querySelector('${escaped_selector}').value"
}

# Verification Helpers
# -----------------------------------------------------------------------------

function verify_input_value() {
  # verify_input_value selector expected [message]
  # Verifies that an input element has the expected value
  #
  # Args:
  #   selector: CSS selector for input element
  #   expected: Expected value
  #   message: Optional assertion message

  local selector="${1}"
  local expected="${2}"
  local message="${3:-Input value matches expected}"

  get_input_value "${selector}"
  assert_success "${TEST_EXIT_CODE}" "get input value succeeded"
  assert_contains "${TEST_STDOUT}" "${expected}" "${message}"
}

function verify_element_attribute() {
  # verify_element_attribute selector attribute expected [message]
  # Verifies that an element has the expected attribute value
  #
  # Args:
  #   selector: CSS selector
  #   attribute: Attribute name
  #   expected: Expected value
  #   message: Optional assertion message

  local selector attribute escaped_selector escaped_attribute
  selector="${1}"
  attribute="${2}"
  local expected="${3}"
  local message="${4:-Attribute value matches expected}"

  escaped_selector="$(js_escape "${selector}")"
  escaped_attribute="$(js_escape "${attribute}")"

  run_test "get ${attribute} of ${selector}" "${WEBCTL_BINARY}" eval "document.querySelector('${escaped_selector}').getAttribute('${escaped_attribute}')"
  assert_success "${TEST_EXIT_CODE}" "get attribute succeeded"
  assert_contains "${TEST_STDOUT}" "${expected}" "${message}"
}

function verify_focused_element() {
  # verify_focused_element expected_id [message]
  # Verifies that the currently focused element has the expected ID
  #
  # Args:
  #   expected_id: Expected element ID (without '#' prefix)
  #               For example: "text-input" not "#text-input"
  #               This is compared against document.activeElement.id
  #   message: Optional assertion message
  #
  # Note: Unlike other helpers, this takes a plain ID string, not a CSS selector

  local expected_id="${1}"
  local message="${2:-Correct element is focused}"

  run_test "get focused element" "${WEBCTL_BINARY}" eval "$(eval_active_element_id)"
  assert_success "${TEST_EXIT_CODE}" "get focused element succeeded"
  assert_contains "${TEST_STDOUT}" "${expected_id}" "${message}"
}
