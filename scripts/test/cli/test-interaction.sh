#!/usr/bin/env bash

# Test: CLI Interaction Commands
# --------------------------------
# Tests for webctl interaction commands: click, type, select, scroll, focus,
# key, eval, ready, clear, find, target.
# Verifies browser interaction functionality across all interaction modes.
#
# External Dependencies:
# ----------------------
# This test requires the following external components:
#
# 1. Test Server (start_test_server from setup.sh)
#    - Serves HTML test pages from PROJECT_ROOT/testdata
#    - Default port: 8888 (configurable via TEST_SERVER_PORT)
#    - Provides test pages: click-targets.html, forms.html, navigation.html,
#      scroll-long.html, console-types.html, blank.html
#
# 2. Daemon Auto-Navigation Behavior
#    - The 'serve' command automatically navigates the browser to the server URL
#    - Tests must wait for this auto-navigation to complete before proceeding
#    - See SERVE_AUTO_NAV_TIMEOUT in interaction-helpers.sh
#
# 3. Test Page Instrumentation
#    - Test HTML pages include data-* attributes for verification
#    - Example: clicking buttons sets data-clicked="true"
#    - This allows testing of browser interaction without complex assertions
#
# 4. JSON Output Format
#    - All commands support --json flag for structured output
#    - Expected format: {"ok": true/false, ...command-specific fields}
#
# 5. Daemon State Persistence
#    - Browser state (focus, scroll position) persists across CLI invocations
#    - This is intentional daemon behavior for interactive use
#    - Tests verify this persistence works correctly

# Determine script location and project root
SCRIPT_DIR="$(cd "${BASH_SOURCE[0]%/*}" || exit 1; pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." || exit 1; pwd)"

# Import test modules
source "${PROJECT_ROOT}/scripts/bash_modules/test-framework.sh"
source "${PROJECT_ROOT}/scripts/bash_modules/assertions.sh"
source "${PROJECT_ROOT}/scripts/bash_modules/setup.sh"
source "${PROJECT_ROOT}/scripts/bash_modules/interaction-helpers.sh"

# Setup
setup_cleanup_trap
require_webctl

# Force stop any existing daemon to ensure clean test state.
#
# Rationale: Interaction tests depend on predictable browser state (no history,
# no prior navigation, clean focus state). If a daemon is already running from
# a previous test or manual session, it may have accumulated state that
# interferes with test assumptions.
#
# This is defensive but necessary because:
# 1. Tests may abort before cleanup runs, leaving daemon running
# 2. Developers may have daemons running during test development
# 3. CI environments may have stale processes from failed runs
#
# Alternatives considered:
# - Check daemon state and only stop if "dirty" → Complex, unreliable
# - Let tests adapt to existing state → Breaks determinism
# - Require manual cleanup → Error-prone, bad DX
force_stop_daemon

# Start daemon and test server
start_daemon --headless
start_test_server

# Wait for the serve command's auto-navigation to complete
# The serve command automatically navigates to the server URL.
# We need to wait for this navigation before running tests.
# Using ready --eval to wait for the URL to contain our test server port.
run_setup_required "wait for serve auto-navigation" "${WEBCTL_BINARY}" ready --eval "$(eval_url_contains ":${TEST_SERVER_PORT}")" --timeout "${SERVE_AUTO_NAV_TIMEOUT}"

# =============================================================================
# Click Command - Basic Functionality
# =============================================================================

test_section "Click Command - Basic"

# Navigate to click-targets page using setup helper (includes --wait)
setup_navigate_to '/pages/click-targets.html'

# Test: Click a button
run_test "click simple button" "${WEBCTL_BINARY}" click "#btn-simple"
assert_success "${TEST_EXIT_CODE}" "click button returns success"

# Verify click worked - check button's data-clicked attribute
verify_element_attribute "#btn-simple" "data-clicked" "true" "Button was clicked successfully"

# Test: Click by class selector
run_test "click toggle button" "${WEBCTL_BINARY}" click "#btn-toggle-1"
assert_success "${TEST_EXIT_CODE}" "click toggle button returns success"

# Verify toggle worked - check if 'active' class was added
run_test "verify toggle active" "${WEBCTL_BINARY}" eval "document.querySelector('#btn-toggle-1').classList.contains('active')"
assert_success "${TEST_EXIT_CODE}" "check classList succeeded"
assert_contains "${TEST_STDOUT}" "true" "Toggle button has 'active' class"

# Test: Click a div (click area)
run_test "click div area" "${WEBCTL_BINARY}" click "#area-1"
assert_success "${TEST_EXIT_CODE}" "click div returns success"

# Verify area was clicked
verify_element_attribute "#area-1" "data-clicked" "true" "Clickable area was clicked successfully"

test_section "Click Command - JSON Output"

# Test: Click with JSON output
run_test "click --json" "${WEBCTL_BINARY}" click --json "#btn-primary"
assert_success "${TEST_EXIT_CODE}" "--json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"

test_section "Click Command - No-Color Mode"

# Test: Click with no-color
run_test "click --no-color" "${WEBCTL_BINARY}" click --no-color "#btn-danger"
assert_success "${TEST_EXIT_CODE}" "--no-color returns success"
assert_no_ansi_codes "${TEST_STDOUT}" "No ANSI escape codes in output"

# Verify the click actually worked (not just the flag)
verify_element_attribute "#btn-danger" "data-clicked" "true" "Button was clicked successfully with --no-color"

test_section "Click Command - Error Cases"

# Test: Click nonexistent element
run_test "click nonexistent element" "${WEBCTL_BINARY}" click ".nonexistent-element-xyz"
assert_failure "${TEST_EXIT_CODE}" "Nonexistent element returns failure"

# =============================================================================
# Type Command - Basic Functionality
# =============================================================================

test_section "Type Command - Basic"

# Navigate to forms page
# This starts fresh navigation, clearing any state from previous tests
setup_navigate_to '/pages/forms.html'

# Test: Type into text input
run_test "type into text input" "${WEBCTL_BINARY}" type "#text-input" "Hello World"
assert_success "${TEST_EXIT_CODE}" "type returns success"

# Verify text was typed
verify_input_value "#text-input" "Hello World" "Text input contains typed text"

# Test: Type into email input
run_test "type into email input" "${WEBCTL_BINARY}" type "#email-input" "test@example.com"
assert_success "${TEST_EXIT_CODE}" "type into email returns success"

# Test: Type into password input
run_test "type into password input" "${WEBCTL_BINARY}" type "#password-input" "secret123"
assert_success "${TEST_EXIT_CODE}" "type into password returns success"

# Test: Type into textarea
run_test "type into textarea" "${WEBCTL_BINARY}" type "#textarea" "Multi-line text content"
assert_success "${TEST_EXIT_CODE}" "type into textarea returns success"

test_section "Type Command - Flags"

# Test: Type with --clear flag
# Using a distinct value "ReplacementValue123" to verify both:
# 1. New value was typed
# 2. Old value "Hello World" was cleared (not appended)
run_test "type with --clear" "${WEBCTL_BINARY}" type --clear "#text-input" "ReplacementValue123"
assert_success "${TEST_EXIT_CODE}" "--clear returns success"

# Verify text was replaced completely
get_input_value "#text-input"
assert_success "${TEST_EXIT_CODE}" "get input value succeeded"
assert_contains "${TEST_STDOUT}" "ReplacementValue123" "Input contains new value"
assert_not_contains "${TEST_STDOUT}" "Hello World" "Old value was cleared (not appended)"

# Test: Type with --key flag (Tab)
run_test "type with --key Tab" "${WEBCTL_BINARY}" type --key Tab "#email-input" "tabtest@example.com"
assert_success "${TEST_EXIT_CODE}" "--key Tab returns success"

# Test: Type with --key flag (Enter)
run_test "type with --key Enter" "${WEBCTL_BINARY}" type --key Enter "#text-input" "Submit this"
assert_success "${TEST_EXIT_CODE}" "--key Enter returns success"

# Test: Type with both --clear and --key
run_test "type with --clear --key Enter" "${WEBCTL_BINARY}" type --clear --key Enter "#text-input" "Clear and submit"
assert_success "${TEST_EXIT_CODE}" "--clear --key returns success"

test_section "Type Command - Without Selector"

# Test: Focus first, then type without selector
# This test verifies that focus state persists across CLI invocations via daemon.
# This is intentional behavior: the daemon maintains browser state for interactive use.
run_test "focus input for typing" "${WEBCTL_BINARY}" focus "#number-input"
assert_success "${TEST_EXIT_CODE}" "focus returns success"

# Verify focus was actually set before proceeding
# This prevents false positives if focus command silently fails
verify_focused_element "number-input" "Number input is focused"

run_test "type without selector (into focused element)" "${WEBCTL_BINARY}" type "42"
assert_success "${TEST_EXIT_CODE}" "type without selector returns success"

# Verify number was typed into the focused element
verify_input_value "#number-input" "42" "Number input contains typed value"

test_section "Type Command - JSON Output"

# Test: Type with JSON output
run_test "type --json" "${WEBCTL_BINARY}" type --json "#text-input" "JSON test"
assert_success "${TEST_EXIT_CODE}" "--json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"

test_section "Type Command - No-Color Mode"

# Test: Type with no-color
run_test "type --no-color" "${WEBCTL_BINARY}" type --no-color "#text-input" "No color test"
assert_success "${TEST_EXIT_CODE}" "--no-color returns success"
assert_no_ansi_codes "${TEST_STDOUT}" "No ANSI escape codes in output"

# Verify typing actually worked (not just the flag)
verify_input_value "#text-input" "No color test" "Text was typed successfully with --no-color"

test_section "Type Command - Error Cases"

# Test: Type into nonexistent element
run_test "type into nonexistent element" "${WEBCTL_BINARY}" type ".nonexistent-input-xyz" "test"
assert_failure "${TEST_EXIT_CODE}" "Nonexistent element returns failure"

# =============================================================================
# Select Command - Basic Functionality
# =============================================================================

test_section "Select Command - Basic"

# Navigate to forms page (fresh page state)
setup_navigate_to '/pages/forms.html'

# Test: Select an option by value
run_test "select option by value" "${WEBCTL_BINARY}" select "#select" "option1"
assert_success "${TEST_EXIT_CODE}" "select returns success"

# Verify selection using eval to get the select element's value
run_test "verify selection" "${WEBCTL_BINARY}" eval "$(eval_element_property '#select' 'value')"
assert_success "${TEST_EXIT_CODE}" "get select value succeeded"
assert_equals "option1" "${TEST_STDOUT}" "Select has correct value"

# Test: Select different option
run_test "select option2" "${WEBCTL_BINARY}" select "#select" "option2"
assert_success "${TEST_EXIT_CODE}" "select option2 returns success"

# Verify new selection
run_test "verify new selection" "${WEBCTL_BINARY}" eval "$(eval_element_property '#select' 'value')"
assert_success "${TEST_EXIT_CODE}" "get select value succeeded"
assert_equals "option2" "${TEST_STDOUT}" "Select has option2 value"

# Test: Select option3
run_test "select option3" "${WEBCTL_BINARY}" select "#select" "option3"
assert_success "${TEST_EXIT_CODE}" "select option3 returns success"

test_section "Select Command - JSON Output"

# Test: Select with JSON output
run_test "select --json" "${WEBCTL_BINARY}" select --json "#select" "option1"
assert_success "${TEST_EXIT_CODE}" "--json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"

test_section "Select Command - No-Color Mode"

# Test: Select with no-color
run_test "select --no-color" "${WEBCTL_BINARY}" select --no-color "#select" "option2"
assert_success "${TEST_EXIT_CODE}" "--no-color returns success"
assert_no_ansi_codes "${TEST_STDOUT}" "No ANSI escape codes in output"

# Verify selection actually worked (not just the flag)
run_test "verify --no-color selection" "${WEBCTL_BINARY}" eval "$(eval_element_property '#select' 'value')"
assert_success "${TEST_EXIT_CODE}" "get select value succeeded"
assert_equals "option2" "${TEST_STDOUT}" "Select has correct value with --no-color"

test_section "Select Command - Error Cases"

# Test: Select nonexistent element
run_test "select nonexistent element" "${WEBCTL_BINARY}" select ".nonexistent-select-xyz" "value"
assert_failure "${TEST_EXIT_CODE}" "Nonexistent element returns failure"

# =============================================================================
# Scroll Command - Basic Functionality
# =============================================================================

test_section "Scroll Command - Basic (Element Mode)"

# Navigate to scroll-long page (fresh state, scroll position reset)
setup_navigate_to '/pages/scroll-long.html'

# Test: Scroll to element
run_test "scroll to element" "${WEBCTL_BINARY}" scroll "#marker-middle"
assert_success "${TEST_EXIT_CODE}" "scroll to element returns success"

# Verify scroll happened - check element is now visible in viewport
# Using ready --eval with helper function for viewport visibility check
run_test "verify element scrolled into view" "${WEBCTL_BINARY}" ready --eval "$(eval_element_visible_in_viewport '#marker-middle')"
assert_success "${TEST_EXIT_CODE}" "Element is now visible in viewport"

# Test: Scroll to another element
run_test "scroll to bottom marker" "${WEBCTL_BINARY}" scroll "#marker-bottom"
assert_success "${TEST_EXIT_CODE}" "scroll to bottom returns success"

# Verify scroll reached target - check element is visible in viewport
run_test "verify bottom marker visible" "${WEBCTL_BINARY}" ready --eval "$(eval_element_visible_in_viewport '#marker-bottom')"
assert_success "${TEST_EXIT_CODE}" "Bottom marker is now visible in viewport"

test_section "Scroll Command - Absolute Mode (--to)"

# Test: Scroll to absolute position (top)
run_test "scroll --to 0,0" "${WEBCTL_BINARY}" scroll --to 0,0
assert_success "${TEST_EXIT_CODE}" "scroll --to 0,0 returns success"

# Wait for scroll and verify at top
# Using TOP_SCROLL_TOLERANCE (5px) - see interaction-helpers.sh for rationale
run_test "wait and verify at top" "${WEBCTL_BINARY}" ready --eval "$(eval_scroll_at_top)"
assert_success "${TEST_EXIT_CODE}" "Scroll position is at top (within ${TOP_SCROLL_TOLERANCE}px)"

# Test: Scroll to specific position
run_test "scroll --to 0,500" "${WEBCTL_BINARY}" scroll --to 0,500
assert_success "${TEST_EXIT_CODE}" "scroll --to 0,500 returns success"

# Wait for scroll and verify position
# Using SCROLL_TOLERANCE (10px) - see interaction-helpers.sh for rationale
run_test "wait and verify position ~500" "${WEBCTL_BINARY}" ready --eval "$(eval_scroll_near_position 500)"
assert_success "${TEST_EXIT_CODE}" "Scroll position is approximately 500 (within ${SCROLL_TOLERANCE}px)"

test_section "Scroll Command - Relative Mode (--by)"

# Reset scroll position to top for predictable relative scrolling
run_test "reset scroll position" "${WEBCTL_BINARY}" scroll --to 0,0
assert_success "${TEST_EXIT_CODE}" "scroll reset returns success"

# Wait for reset and verify we're at top
run_test "wait for reset" "${WEBCTL_BINARY}" ready --eval "$(eval_scroll_at_top)"
assert_success "${TEST_EXIT_CODE}" "Reset complete - at top"

# Test: Scroll by offset (relative to current position)
run_test "scroll --by 0,200" "${WEBCTL_BINARY}" scroll --by 0,200
assert_success "${TEST_EXIT_CODE}" "scroll --by 0,200 returns success"

# Wait and verify relative scroll from top (0 + 200 = ~200)
run_test "wait and verify relative scroll ~200" "${WEBCTL_BINARY}" ready --eval "$(eval_scroll_near_position 200)"
assert_success "${TEST_EXIT_CODE}" "Scroll position is approximately 200 (within ${SCROLL_TOLERANCE}px)"

# Test: Scroll by another offset (cumulative: 200 + 300 = 500)
run_test "scroll --by 0,300 (cumulative)" "${WEBCTL_BINARY}" scroll --by 0,300
assert_success "${TEST_EXIT_CODE}" "scroll --by 0,300 returns success"

# Wait and verify cumulative scroll (200 + 300 = ~500)
run_test "wait and verify cumulative scroll ~500" "${WEBCTL_BINARY}" ready --eval "$(eval_scroll_near_position 500)"
assert_success "${TEST_EXIT_CODE}" "Scroll position is approximately 500 (200+300, within ${SCROLL_TOLERANCE}px)"

test_section "Scroll Command - JSON Output"

# Test: Scroll with JSON output
run_test "scroll --json --to 0,0" "${WEBCTL_BINARY}" scroll --json --to 0,0
assert_success "${TEST_EXIT_CODE}" "--json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"

test_section "Scroll Command - No-Color Mode"

# Test: Scroll with no-color
run_test "scroll --no-color" "${WEBCTL_BINARY}" scroll --no-color "#marker-middle"
assert_success "${TEST_EXIT_CODE}" "--no-color returns success"
assert_no_ansi_codes "${TEST_STDOUT}" "No ANSI escape codes in output"

# Verify scroll actually worked (not just the flag)
run_test "verify --no-color scroll" "${WEBCTL_BINARY}" ready --eval "$(eval_element_visible_in_viewport '#marker-middle')"
assert_success "${TEST_EXIT_CODE}" "Element scrolled into view with --no-color"

test_section "Scroll Command - Error Cases"

# Test: Scroll to nonexistent element
run_test "scroll nonexistent element" "${WEBCTL_BINARY}" scroll ".nonexistent-element-xyz"
assert_failure "${TEST_EXIT_CODE}" "Nonexistent element returns failure"

# Test: Scroll with invalid --to coordinates
run_test "scroll --to invalid format" "${WEBCTL_BINARY}" scroll --to "invalid"
assert_failure "${TEST_EXIT_CODE}" "Invalid --to format returns failure"

# Test: Scroll with invalid --by coordinates
run_test "scroll --by invalid format" "${WEBCTL_BINARY}" scroll --by "not,numbers"
assert_failure "${TEST_EXIT_CODE}" "Invalid --by format returns failure"

# =============================================================================
# Focus Command - Basic Functionality
# =============================================================================

test_section "Focus Command - Basic"

# Navigate to forms page (fresh state)
setup_navigate_to '/pages/forms.html'

# Test: Focus an input
run_test "focus text input" "${WEBCTL_BINARY}" focus "#text-input"
assert_success "${TEST_EXIT_CODE}" "focus returns success"

# Verify focus using helper
verify_focused_element "text-input" "Text input is focused"

# Test: Focus another input (verifies focus can move)
run_test "focus email input" "${WEBCTL_BINARY}" focus "#email-input"
assert_success "${TEST_EXIT_CODE}" "focus email returns success"

# Verify new focus
verify_focused_element "email-input" "Email input is focused"

# Test: Focus textarea
run_test "focus textarea" "${WEBCTL_BINARY}" focus "#textarea"
assert_success "${TEST_EXIT_CODE}" "focus textarea returns success"

# Test: Focus button
run_test "focus button" "${WEBCTL_BINARY}" focus "#submit-btn"
assert_success "${TEST_EXIT_CODE}" "focus button returns success"

test_section "Focus Command - JSON Output"

# Test: Focus with JSON output
run_test "focus --json" "${WEBCTL_BINARY}" focus --json "#text-input"
assert_success "${TEST_EXIT_CODE}" "--json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"

test_section "Focus Command - No-Color Mode"

# Test: Focus with no-color
run_test "focus --no-color" "${WEBCTL_BINARY}" focus --no-color "#password-input"
assert_success "${TEST_EXIT_CODE}" "--no-color returns success"
assert_no_ansi_codes "${TEST_STDOUT}" "No ANSI escape codes in output"

# Verify focus actually worked (not just the flag)
verify_focused_element "password-input" "Password input is focused with --no-color"

test_section "Focus Command - Error Cases"

# Test: Focus nonexistent element
run_test "focus nonexistent element" "${WEBCTL_BINARY}" focus ".nonexistent-element-xyz"
assert_failure "${TEST_EXIT_CODE}" "Nonexistent element returns failure"

# =============================================================================
# Key Command - Basic Functionality
# =============================================================================

test_section "Key Command - Basic"

# Navigate to forms page (fresh state)
setup_navigate_to '/pages/forms.html'

# Focus an input first to establish starting point for Tab navigation
run_test "focus text input for key tests" "${WEBCTL_BINARY}" focus "#text-input"
assert_success "${TEST_EXIT_CODE}" "focus returns success"

# Verify initial focus before testing Tab navigation
verify_focused_element "text-input" "Initial focus on text-input"

# Test: Send Tab key and verify focus moved
run_test "key Tab" "${WEBCTL_BINARY}" key Tab
assert_success "${TEST_EXIT_CODE}" "key Tab returns success"

# Verify Tab moved focus to next element (email-input is after text-input in tab order)
verify_focused_element "email-input" "Tab moved focus to email-input"

# Test: Send another Tab to verify continued navigation through tab order
run_test "key Tab (second)" "${WEBCTL_BINARY}" key Tab
assert_success "${TEST_EXIT_CODE}" "key Tab returns success"

# Verify focus moved to password-input (third element in tab order)
verify_focused_element "password-input" "Second Tab moved focus to password-input"

# Test: Send Escape key
run_test "key Escape" "${WEBCTL_BINARY}" key Escape
assert_success "${TEST_EXIT_CODE}" "key Escape returns success"

# Test: Send Space key (on a focusable element)
run_test "key Space" "${WEBCTL_BINARY}" key Space
assert_success "${TEST_EXIT_CODE}" "key Space returns success"

# Test navigation keys - these verify command execution
# Note: Full cursor movement verification would require text selection state
run_test "key ArrowDown" "${WEBCTL_BINARY}" key ArrowDown
assert_success "${TEST_EXIT_CODE}" "key ArrowDown returns success"

run_test "key ArrowUp" "${WEBCTL_BINARY}" key ArrowUp
assert_success "${TEST_EXIT_CODE}" "key ArrowUp returns success"

run_test "key ArrowLeft" "${WEBCTL_BINARY}" key ArrowLeft
assert_success "${TEST_EXIT_CODE}" "key ArrowLeft returns success"

run_test "key ArrowRight" "${WEBCTL_BINARY}" key ArrowRight
assert_success "${TEST_EXIT_CODE}" "key ArrowRight returns success"

run_test "key Backspace" "${WEBCTL_BINARY}" key Backspace
assert_success "${TEST_EXIT_CODE}" "key Backspace returns success"

run_test "key Delete" "${WEBCTL_BINARY}" key Delete
assert_success "${TEST_EXIT_CODE}" "key Delete returns success"

run_test "key Home" "${WEBCTL_BINARY}" key Home
assert_success "${TEST_EXIT_CODE}" "key Home returns success"

run_test "key End" "${WEBCTL_BINARY}" key End
assert_success "${TEST_EXIT_CODE}" "key End returns success"

# Test: Space key with behavior verification (checkbox toggle)
# Note: Enter doesn't toggle checkboxes in standard HTML, Space does
run_test "focus checkbox" "${WEBCTL_BINARY}" focus "#checkbox"
assert_success "${TEST_EXIT_CODE}" "focus checkbox returns success"

# Verify checkbox is initially unchecked
run_test "verify checkbox initially unchecked" "${WEBCTL_BINARY}" eval "$(eval_element_property '#checkbox' 'checked')"
assert_success "${TEST_EXIT_CODE}" "get checkbox state succeeded"
assert_contains "${TEST_STDOUT}" "false" "Checkbox initially unchecked"

# Space toggles checkbox (standard HTML behavior)
run_test "key Space to toggle checkbox" "${WEBCTL_BINARY}" key Space
assert_success "${TEST_EXIT_CODE}" "key Space returns success"

# Verify checkbox is now checked
run_test "verify checkbox toggled" "${WEBCTL_BINARY}" eval "$(eval_element_property '#checkbox' 'checked')"
assert_success "${TEST_EXIT_CODE}" "get checkbox state succeeded"
assert_contains "${TEST_STDOUT}" "true" "Space toggled checkbox to checked"

test_section "Key Command - Modifier Flags"

# Test: Key with --ctrl
run_test "key a --ctrl" "${WEBCTL_BINARY}" key --ctrl a
assert_success "${TEST_EXIT_CODE}" "--ctrl returns success"

# Test: Key with --shift
run_test "key a --shift" "${WEBCTL_BINARY}" key --shift a
assert_success "${TEST_EXIT_CODE}" "--shift returns success"

# Test: Key with --alt
run_test "key a --alt" "${WEBCTL_BINARY}" key --alt a
assert_success "${TEST_EXIT_CODE}" "--alt returns success"

# Test: Key with --meta (macOS Cmd)
run_test "key a --meta" "${WEBCTL_BINARY}" key --meta a
assert_success "${TEST_EXIT_CODE}" "--meta returns success"

# Test: Combined modifiers
run_test "key z --ctrl --shift" "${WEBCTL_BINARY}" key --ctrl --shift z
assert_success "${TEST_EXIT_CODE}" "combined modifiers returns success"

test_section "Key Command - JSON Output"

# Test: Key with JSON output
run_test "key --json Tab" "${WEBCTL_BINARY}" key --json Tab
assert_success "${TEST_EXIT_CODE}" "--json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"

test_section "Key Command - No-Color Mode"

# Test: Key with no-color
run_test "key --no-color Enter" "${WEBCTL_BINARY}" key --no-color Enter
assert_success "${TEST_EXIT_CODE}" "--no-color returns success"
assert_no_ansi_codes "${TEST_STDOUT}" "No ANSI escape codes in output"

# =============================================================================
# Eval Command - Basic Functionality
# =============================================================================

test_section "Eval Command - Basic"

# Navigate to a page (fresh state)
setup_navigate_to '/pages/navigation.html'

# Test: Simple arithmetic expression
run_test "eval simple expression" "${WEBCTL_BINARY}" eval "1 + 1"
assert_success "${TEST_EXIT_CODE}" "eval returns success"
assert_contains "${TEST_STDOUT}" "2" "Arithmetic result is correct"

# Test: String expression
run_test "eval string" "${WEBCTL_BINARY}" eval "'hello'.toUpperCase()"
assert_success "${TEST_EXIT_CODE}" "eval string returns success"
assert_contains "${TEST_STDOUT}" "HELLO" "Output contains uppercase result"

# Test: Array operations
run_test "eval array" "${WEBCTL_BINARY}" eval "[1,2,3].length"
assert_success "${TEST_EXIT_CODE}" "eval array returns success"
assert_contains "${TEST_STDOUT}" "3" "Output contains array length"

# Test: DOM query
run_test "eval DOM query" "${WEBCTL_BINARY}" eval "document.title"
assert_success "${TEST_EXIT_CODE}" "eval document.title returns success"
assert_contains "${TEST_STDOUT}" "Navigation Test Page" "Output contains page title"

# Test: DOM manipulation
run_test "eval DOM manipulation" "${WEBCTL_BINARY}" eval "document.body.style.background = 'white'; 'done'"
assert_success "${TEST_EXIT_CODE}" "eval DOM manipulation returns success"
assert_contains "${TEST_STDOUT}" "done" "Output contains result"

# Test: Get element count
run_test "eval element count" "${WEBCTL_BINARY}" eval "document.querySelectorAll('a').length"
assert_success "${TEST_EXIT_CODE}" "eval element count returns success"

# Test: Boolean expression
run_test "eval boolean" "${WEBCTL_BINARY}" eval "document.querySelector('h1') !== null"
assert_success "${TEST_EXIT_CODE}" "eval boolean returns success"
assert_contains "${TEST_STDOUT}" "true" "Output contains true"

# Test: Object return
run_test "eval object" "${WEBCTL_BINARY}" eval "({name: 'test', value: 42})"
assert_success "${TEST_EXIT_CODE}" "eval object returns success"
assert_contains "${TEST_STDOUT}" "test" "Output contains object name"
assert_contains "${TEST_STDOUT}" "42" "Output contains object value"

test_section "Eval Command - Flags"

# Test: Eval with --timeout flag
# Timeout is useful for potentially long-running evals (e.g., waiting for async operations)
# Using DAEMON_HEALTH_CHECK_TIMEOUT to allow for slower systems
run_test "eval with --timeout" "${WEBCTL_BINARY}" eval --timeout "${DAEMON_HEALTH_CHECK_TIMEOUT}" "1 + 1"
assert_success "${TEST_EXIT_CODE}" "--timeout flag works"

# Test: Eval with -t short flag (alias for --timeout)
run_test "eval with -t" "${WEBCTL_BINARY}" eval -t "${DEFAULT_READY_TIMEOUT}" "2 + 2"
assert_success "${TEST_EXIT_CODE}" "-t short flag works"

test_section "Eval Command - JSON Output"

# Test: Eval with JSON output
run_test "eval --json" "${WEBCTL_BINARY}" eval --json "42"
assert_success "${TEST_EXIT_CODE}" "--json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
assert_json_field "${TEST_STDOUT}" ".value" "42" "JSON value is 42"

# Test: Eval JSON with string result
run_test "eval --json string result" "${WEBCTL_BINARY}" eval --json "'hello'"
assert_success "${TEST_EXIT_CODE}" "--json string returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"

# Test: Eval JSON with null result
run_test "eval --json null result" "${WEBCTL_BINARY}" eval --json "null"
assert_success "${TEST_EXIT_CODE}" "--json null returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"

test_section "Eval Command - No-Color Mode"

# Test: Eval with no-color
run_test "eval --no-color" "${WEBCTL_BINARY}" eval --no-color "1 + 1"
assert_success "${TEST_EXIT_CODE}" "--no-color returns success"
assert_no_ansi_codes "${TEST_STDOUT}" "No ANSI escape codes in output"

# Verify eval actually worked (not just the flag)
assert_contains "${TEST_STDOUT}" "2" "Eval returned correct result with --no-color"

test_section "Eval Command - Error Cases"

# Test: Eval syntax error
run_test "eval syntax error" "${WEBCTL_BINARY}" eval "function("
assert_failure "${TEST_EXIT_CODE}" "Syntax error returns failure"

# Test: Eval reference error
run_test "eval reference error" "${WEBCTL_BINARY}" eval "nonexistentVariable"
assert_failure "${TEST_EXIT_CODE}" "Reference error returns failure"

# =============================================================================
# Ready Command - Basic Functionality
# =============================================================================

test_section "Ready Command - Basic (Page Load)"

# Navigate to a page (fresh state)
setup_navigate_to '/pages/navigation.html'

# Test: Ready (page load mode) - waits for DOMContentLoaded
run_test "ready (page load)" "${WEBCTL_BINARY}" ready
assert_success "${TEST_EXIT_CODE}" "ready returns success"

test_section "Ready Command - Selector Mode"

# Test: Ready with selector - waits for element to exist in DOM
run_test "ready with selector" "${WEBCTL_BINARY}" ready "h1"
assert_success "${TEST_EXIT_CODE}" "ready with selector returns success"

# Test: Ready with existing selector (should return immediately)
run_test "ready with existing element" "${WEBCTL_BINARY}" ready "body"
assert_success "${TEST_EXIT_CODE}" "ready with body selector returns success"

test_section "Ready Command - Flags"

# Test: Ready with --timeout (custom timeout for slow pages)
# Using DEFAULT_READY_TIMEOUT from interaction-helpers.sh
run_test "ready --timeout" "${WEBCTL_BINARY}" ready --timeout "${DEFAULT_READY_TIMEOUT}"
assert_success "${TEST_EXIT_CODE}" "--timeout returns success"

# Test: Ready with --network-idle (waits for network requests to finish)
run_test "ready --network-idle" "${WEBCTL_BINARY}" ready --network-idle
assert_success "${TEST_EXIT_CODE}" "--network-idle returns success"

# Test: Ready with combined flags
run_test "ready --network-idle --timeout" "${WEBCTL_BINARY}" ready --network-idle --timeout "${DAEMON_HEALTH_CHECK_TIMEOUT}"
assert_success "${TEST_EXIT_CODE}" "--network-idle --timeout returns success"

# Test: Ready with --eval (custom JavaScript condition)
run_test "ready --eval condition" "${WEBCTL_BINARY}" ready --eval "document.readyState === 'complete'"
assert_success "${TEST_EXIT_CODE}" "--eval with document.readyState returns success"

# Test: Ready with --eval true condition (should return immediately)
run_test "ready --eval true" "${WEBCTL_BINARY}" ready --eval "true"
assert_success "${TEST_EXIT_CODE}" "--eval true returns success immediately"

test_section "Ready Command - JSON Output"

# Test: Ready with JSON output
run_test "ready --json" "${WEBCTL_BINARY}" ready --json
assert_success "${TEST_EXIT_CODE}" "--json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"

test_section "Ready Command - No-Color Mode"

# Test: Ready with no-color
run_test "ready --no-color" "${WEBCTL_BINARY}" ready --no-color
assert_success "${TEST_EXIT_CODE}" "--no-color returns success"
assert_no_ansi_codes "${TEST_STDOUT}" "No ANSI escape codes in output"

test_section "Ready Command - Error Cases"

# Test: Ready with --eval false condition (should timeout)
# Using FORCED_FAILURE_TIMEOUT (2s) - long enough to prove failure, short enough to be fast
run_test "ready --eval false with timeout" "${WEBCTL_BINARY}" ready --eval "false" --timeout "${FORCED_FAILURE_TIMEOUT}"
assert_failure "${TEST_EXIT_CODE}" "--eval false with timeout returns failure"

# Test: Ready with nonexistent selector (should timeout)
run_test "ready nonexistent selector with timeout" "${WEBCTL_BINARY}" ready ".nonexistent-element-xyz" --timeout "${FORCED_FAILURE_TIMEOUT}"
assert_failure "${TEST_EXIT_CODE}" "Nonexistent selector with timeout returns failure"

# =============================================================================
# Daemon Health Check After Forced Timeout Tests
# =============================================================================
# WHY THIS CHECK EXISTS:
# ----------------------
# The timeout tests above deliberately force the browser to wait for conditions
# that will never be satisfied (--eval false, nonexistent selectors). This is
# necessary to verify error handling, but it can destabilize the daemon-browser
# connection.
#
# KNOWN ISSUES:
# -------------
# 1. CDP Protocol Limitations
#    - Chrome DevTools Protocol (CDP) doesn't handle long-running evals well
#    - Extended waits can cause CDP message queue backlog
#    - Connection may silently fail without proper error reporting
#
# 2. Browser-Side Timeouts
#    - Browser internal timeouts may fire independently of our timeout flags
#    - This can leave the browser in an inconsistent state
#    - Focus, scroll position, or page state may be corrupted
#
# 3. Rapid Repeated Failures
#    - Multiple forced timeouts in quick succession stress the connection
#    - Each timeout may leave cleanup tasks pending
#    - Accumulated pending tasks can crash the daemon
#
# WHY NOT JUST FIX THE ROOT CAUSE:
# --------------------------------
# This is a fundamental limitation of CDP and browser architecture. Options:
# - Don't test timeout behavior → Unacceptable (need error case coverage)
# - Use shorter timeouts → Already using minimum viable (2s)
# - Restart daemon between tests → Too slow (adds ~10s per test)
# - Skip health check → Tests fail mysteriously on remaining tests
#
# This health check is the pragmatic solution: verify daemon survived the
# stress test and recover if it didn't. If this block executes frequently,
# it indicates the timeout stress is worse than expected and should be
# investigated further.
#
# WHAT WE CHECK:
# --------------
# 1. Process existence (is daemon running?)
# 2. Functional health (does 'status' command respond?)
# If either check fails, we restart the daemon and re-serve the test server.
#
# REFERENCE:
# ----------
# See .ai/context/cdp-timeout-research.md for detailed research confirming
# these CDP limitations are architectural and documented across Chromium
# issue tracker, Puppeteer, Playwright, and ChromeDP projects.
# =============================================================================

# Check daemon health after timeout stress tests
# Forced timeouts can destabilize CDP connection (see comment block above)
DAEMON_HEALTHY=true

if ! is_daemon_running; then
  DAEMON_HEALTHY=false
  log_warning "Daemon process terminated after timeout tests"
elif ! "${WEBCTL_BINARY}" status >/dev/null 2>&1; then
  DAEMON_HEALTHY=false
  log_warning "Daemon process exists but is not responding - forcing restart"
  force_stop_daemon
fi

# Recover if daemon is unhealthy
if [[ "${DAEMON_HEALTHY}" != "true" ]]; then
  log_warning "Daemon stability issue detected after forced timeout tests"
  log_warning "This is a KNOWN ISSUE with CDP timeout handling - see comments above"
  log_message "Restarting daemon to continue remaining tests..."

  start_daemon --headless
  start_test_server

  run_setup_required "wait for serve auto-navigation after restart" \
    "${WEBCTL_BINARY}" ready --eval "$(eval_url_contains ":${TEST_SERVER_PORT}")" \
    --timeout "${DAEMON_HEALTH_CHECK_TIMEOUT}"
fi

# =============================================================================
# Clear Command - Basic Functionality
# =============================================================================

test_section "Clear Command - Basic"

# Navigate to console page to generate console logs
# This page automatically generates console.log output on page load
setup_navigate_to '/pages/console-types.html'

# Wait for page to load and console logs to be generated
run_test "wait for console logs" "${WEBCTL_BINARY}" ready --timeout "${DEFAULT_READY_TIMEOUT}"
assert_success "${TEST_EXIT_CODE}" "Page ready with console logs"

# Verify console has entries before clearing (text output mode)
run_test "verify console has entries" "${WEBCTL_BINARY}" console
assert_success "${TEST_EXIT_CODE}" "console command returns success"
assert_not_empty "${TEST_STDOUT}" "Console has output before clearing"

# Test: Clear all buffers (console + network)
run_test "clear (all buffers)" "${WEBCTL_BINARY}" clear
assert_success "${TEST_EXIT_CODE}" "clear all buffers returns success"

# Test: Clear console buffer specifically
run_test "clear console" "${WEBCTL_BINARY}" clear console
assert_success "${TEST_EXIT_CODE}" "clear console returns success"

# Test: Clear network buffer specifically
run_test "clear network" "${WEBCTL_BINARY}" clear network
assert_success "${TEST_EXIT_CODE}" "clear network returns success"

# Verify console is empty after clear using JSON output for precise verification
run_test "verify console empty" "${WEBCTL_BINARY}" console --json
assert_success "${TEST_EXIT_CODE}" "console --json after clear returns success"
assert_json_array_length "${TEST_STDOUT}" ".entries" "0" "Console entries array is empty"

test_section "Clear Command - JSON Output"

# Test: Clear with JSON output
run_test "clear --json" "${WEBCTL_BINARY}" clear --json
assert_success "${TEST_EXIT_CODE}" "--json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
assert_contains "${TEST_STDOUT}" "message" "JSON contains message field"

# Test: Clear console with JSON
run_test "clear console --json" "${WEBCTL_BINARY}" clear --json console
assert_success "${TEST_EXIT_CODE}" "clear console --json returns success"
assert_contains "${TEST_STDOUT}" "console" "JSON message mentions console"

# Test: Clear network with JSON
run_test "clear network --json" "${WEBCTL_BINARY}" clear --json network
assert_success "${TEST_EXIT_CODE}" "clear network --json returns success"
assert_contains "${TEST_STDOUT}" "network" "JSON message mentions network"

test_section "Clear Command - No-Color Mode"

# Test: Clear with no-color
run_test "clear --no-color" "${WEBCTL_BINARY}" clear --no-color
assert_success "${TEST_EXIT_CODE}" "--no-color returns success"
assert_no_ansi_codes "${TEST_STDOUT}" "No ANSI escape codes in output"

test_section "Clear Command - Error Cases"

# Test: Clear with invalid target
run_test "clear invalid target" "${WEBCTL_BINARY}" clear invalidtarget
assert_failure "${TEST_EXIT_CODE}" "Invalid target returns failure"
assert_matches "(console|network|invalid|target)" "${TEST_STDERR}" \
  "Error message provides helpful context about valid targets"

# =============================================================================
# Find Command - Basic Functionality
# =============================================================================

test_section "Find Command - Basic"

# Navigate to a page with content (fresh state)
# Page title: "Navigation Test Page"
# Page contains: "Navigation", "Test", "Page" and other text
setup_navigate_to '/pages/navigation.html'

# Test: Find text (case-insensitive by default)
run_test "find text (case-insensitive)" "${WEBCTL_BINARY}" find "Navigation"
assert_success "${TEST_EXIT_CODE}" "find returns success"
assert_contains "${TEST_STDOUT}" "Navigation" "Output contains matched text"

# Test: Find lowercase variant (should match due to case-insensitive default)
run_test "find lowercase (case-insensitive)" "${WEBCTL_BINARY}" find "navigation"
assert_success "${TEST_EXIT_CODE}" "find lowercase returns success"

test_section "Find Command - Flags"

# Test: Find with --case-sensitive (exact case match)
# Page contains "Navigation Test Page" (title case)
# Searching for "Navigation" (exact match) should find occurrences
run_test "find --case-sensitive exact match" "${WEBCTL_BINARY}" find --case-sensitive --json "Navigation"
assert_success "${TEST_EXIT_CODE}" "--case-sensitive with exact case returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"

# Verify we found matches by checking total > 0
run_test "verify case-sensitive match count" "${WEBCTL_BINARY}" find --case-sensitive --json "Navigation"
assert_success "${TEST_EXIT_CODE}" "find completed"

# Extract total from JSON output
MATCH_TOTAL=$(echo "${TEST_STDOUT}" | jq -r '.total' 2>/dev/null || echo "0")

# Validate extraction succeeded and value is numeric
if [[ ! "${MATCH_TOTAL}" =~ ^[0-9]+$ ]]; then
  log_failure "Failed to extract valid numeric match count from JSON"
  increment_fail
  MATCH_TOTAL=0
fi

assert_greater_than 0 "${MATCH_TOTAL}" "Found matches for case-sensitive 'Navigation'"

# Test: Find with --case-sensitive (wrong case - no match expected)
# Searching "navigation" (lowercase) when page has "Navigation" (title case)
# Should find ZERO matches because case must match exactly
run_test "find --case-sensitive wrong case (lowercase)" "${WEBCTL_BINARY}" find --case-sensitive --json "navigation"
assert_success "${TEST_EXIT_CODE}" "--case-sensitive lowercase search completes"
assert_json_field "${TEST_STDOUT}" ".total" "0" "Lowercase != Title case: 0 matches"

# Test: Find with --case-sensitive (wrong case - no match expected)
# Searching "NAVIGATION" (uppercase) when page has "Navigation" (title case)
# Should find ZERO matches because case must match exactly
run_test "find --case-sensitive wrong case (uppercase)" "${WEBCTL_BINARY}" find --case-sensitive --json "NAVIGATION"
assert_success "${TEST_EXIT_CODE}" "--case-sensitive uppercase search completes"
assert_json_field "${TEST_STDOUT}" ".total" "0" "Uppercase != Title case: 0 matches"

# Test: Find with -c short flag
run_test "find -c" "${WEBCTL_BINARY}" find -c "Navigation"
assert_success "${TEST_EXIT_CODE}" "-c returns success"

# Test: Find with --regex
run_test "find --regex" "${WEBCTL_BINARY}" find --regex "Nav.*tion"
assert_success "${TEST_EXIT_CODE}" "--regex returns success"

# Test: Find with -E short flag
run_test "find -E" "${WEBCTL_BINARY}" find -E "Nav[a-z]+"
assert_success "${TEST_EXIT_CODE}" "-E returns success"

# Test: Find with --limit
run_test "find --limit 1" "${WEBCTL_BINARY}" find --limit 1 "the"
assert_success "${TEST_EXIT_CODE}" "--limit returns success"

# Test: Find with -l short flag
run_test "find -l 2" "${WEBCTL_BINARY}" find -l 2 "the"
assert_success "${TEST_EXIT_CODE}" "-l returns success"

test_section "Find Command - JSON Output"

# Test: Find with JSON output
run_test "find --json" "${WEBCTL_BINARY}" find --json "Navigation"
assert_success "${TEST_EXIT_CODE}" "--json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
assert_contains "${TEST_STDOUT}" "matches" "JSON contains matches field"
assert_contains "${TEST_STDOUT}" "total" "JSON contains total field"

test_section "Find Command - No-Color Mode"

# Test: Find with no-color
run_test "find --no-color" "${WEBCTL_BINARY}" find --no-color "Navigation"
assert_success "${TEST_EXIT_CODE}" "--no-color returns success"
assert_no_ansi_codes "${TEST_STDOUT}" "No ANSI escape codes in output"

test_section "Find Command - Error Cases"

# Test: Find with no matches (command succeeds, just returns 0 matches)
run_test "find no matches" "${WEBCTL_BINARY}" find "NONEXISTENT_TEXT_XYZ_12345"
assert_success "${TEST_EXIT_CODE}" "No matches returns success (search worked)"

# Test: Find with query too short
run_test "find query too short" "${WEBCTL_BINARY}" find "ab"
assert_failure "${TEST_EXIT_CODE}" "Short query returns failure"
assert_matches "(minimum|least|characters|length)" "${TEST_STDERR}" \
  "Error message mentions minimum length requirement"

# Test: Find --case-sensitive with --regex (mutually exclusive)
# Note: This tests a CLI design decision that these flags are mutually exclusive.
# If the implementation changes to support both flags together, this test should be updated.
run_test "find --case-sensitive --regex (error)" "${WEBCTL_BINARY}" find --case-sensitive --regex "test"
assert_failure "${TEST_EXIT_CODE}" "Mutually exclusive flags return failure"

# Test: Find with invalid regex
run_test "find --regex invalid pattern" "${WEBCTL_BINARY}" find --regex "[invalid("
assert_failure "${TEST_EXIT_CODE}" "Invalid regex returns failure"

# =============================================================================
# Target Command - Basic Functionality
# =============================================================================

test_section "Target Command - Basic (List Targets)"

# Test: List targets
run_test "target (list)" "${WEBCTL_BINARY}" target
assert_success "${TEST_EXIT_CODE}" "target returns success"
assert_contains "${TEST_STDOUT}" "http" "Output contains URL"

test_section "Target Command - JSON Output"

# Test: Target with JSON output
run_test "target --json" "${WEBCTL_BINARY}" target --json
assert_success "${TEST_EXIT_CODE}" "--json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
assert_contains "${TEST_STDOUT}" "sessions" "JSON contains sessions field"
assert_contains "${TEST_STDOUT}" "activeSession" "JSON contains activeSession field"

test_section "Target Command - No-Color Mode"

# Test: Target with no-color
run_test "target --no-color" "${WEBCTL_BINARY}" target --no-color
assert_success "${TEST_EXIT_CODE}" "--no-color returns success"
assert_no_ansi_codes "${TEST_STDOUT}" "No ANSI escape codes in output"

test_section "Target Command - Error Cases"

# Test: Target with nonexistent query
run_test "target nonexistent query" "${WEBCTL_BINARY}" target "nonexistent-session-xyz-12345"
assert_failure "${TEST_EXIT_CODE}" "Nonexistent target returns failure"

# =============================================================================
# Target Command - Session Selection Tests
# =============================================================================

test_section "Target Command - Session Selection"

# Navigate to a known page first to establish predictable state
setup_navigate_to '/pages/navigation.html'

# Test: Target can match current session by URL pattern
# The target command searches session URLs for the given pattern
run_test "target match by URL pattern" "${WEBCTL_BINARY}" target "navigation"
assert_success "${TEST_EXIT_CODE}" "Target matches URL pattern"

# Verify we're still on the same page (target doesn't navigate, just selects)
run_test "verify still on navigation page" "${WEBCTL_BINARY}" eval "document.title"
assert_success "${TEST_EXIT_CODE}" "eval returns success"
assert_contains "${TEST_STDOUT}" "Navigation" "Still on Navigation page"

# Test: Target can select by common URL pattern (localhost)
run_test "target match by localhost" "${WEBCTL_BINARY}" target "localhost"
assert_success "${TEST_EXIT_CODE}" "Target matches localhost"

# Test: Target shows session info in JSON format
run_test "target --json session info" "${WEBCTL_BINARY}" target --json
assert_success "${TEST_EXIT_CODE}" "target --json returns success"
# Verify JSON structure:
# - activeSession: current session ID string
# - sessions: array of session objects with url, title, id fields
assert_json_field_exists "${TEST_STDOUT}" ".activeSession" "Active session ID exists"
assert_json_field_exists "${TEST_STDOUT}" ".sessions[0].url" "First session has URL"
assert_json_field_exists "${TEST_STDOUT}" ".sessions[0].title" "First session has title"

# =============================================================================
# Summary
# =============================================================================

test_summary
