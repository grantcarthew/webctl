#!/usr/bin/env bash

# Setup Module
# ------------
# Provides build, daemon lifecycle, and test server management helpers.
# Handles cleanup and trap handlers for test teardown.

# Environment setup
set -o pipefail

# Determine script location and project root
BASH_MODULES_DIR="$(cd "${BASH_SOURCE[0]%/*}" || exit 1; pwd)"
PROJECT_ROOT="$(cd "${BASH_MODULES_DIR}/../.." || exit 1; pwd)"

# Import test-framework if not already loaded
if ! declare -f log_message >/dev/null 2>&1; then
  if [[ ! -f "${BASH_MODULES_DIR}/test-framework.sh" ]]; then
    echo "ERROR: test-framework.sh not found at ${BASH_MODULES_DIR}" >&2
    return 1
  fi
  source "${BASH_MODULES_DIR}/test-framework.sh"
fi

# Configuration
WEBCTL_BINARY="${PROJECT_ROOT}/webctl"
TESTDATA_DIR="${PROJECT_ROOT}/testdata"
TEST_SERVER_PORT="${TEST_SERVER_PORT:-8888}"
TEST_SERVER_PID=""
DAEMON_STARTED_BY_TEST=false

# Configurable timeout for navigation operations (seconds)
# Can be overridden via environment variable
TEST_TIMEOUT="${TEST_TIMEOUT:-30}"

# Captured page state after test execution
# These are populated by capture_page_state and used by assert_on_page_captured
CAPTURED_PAGE_URL=""
CAPTURED_PAGE_TITLE=""

# Flag to track if we should abort on setup failures
ABORT_ON_SETUP_FAILURE=true

# Temporary files for cleanup
declare -a TEMP_FILES=()

# Build Functions
# -----------------------------------------------------------------------------

function build_webctl() {
  # build_webctl
  # Compiles the webctl binary. Returns 0 on success.

  log_message "Building webctl..."

  if ! command -v go >/dev/null 2>&1; then
    log_error "ERROR: 'go' is not installed"
    return 1
  fi

  local build_output
  if build_output=$(cd "${PROJECT_ROOT}" && go build -o webctl . 2>&1); then
    log_success "webctl built successfully"
    return 0
  else
    log_failure "webctl build failed"
    log_message "${build_output}"
    return 1
  fi
}

function require_webctl() {
  # require_webctl
  # Ensures webctl binary exists, building if necessary

  if [[ -x "${WEBCTL_BINARY}" ]]; then
    return 0
  fi

  log_message "webctl binary not found, building..."
  build_webctl
}

# Daemon Lifecycle
# -----------------------------------------------------------------------------

function is_daemon_running() {
  # is_daemon_running
  # Returns 0 if daemon is running, 1 otherwise

  local status_output
  status_output=$("${WEBCTL_BINARY}" status 2>&1)

  if [[ "${status_output}" == *"Not running"* ]]; then
    return 1
  else
    return 0
  fi
}

function start_daemon() {
  # start_daemon [--headless]
  # Starts the daemon, waits for ready. Returns 0 on success.

  local headless_flag=""
  if [[ "${1}" == "--headless" ]]; then
    headless_flag="--headless"
  fi

  # Check if already running
  if is_daemon_running; then
    log_message "Daemon already running"
    DAEMON_STARTED_BY_TEST=false
    return 0
  fi

  log_message "Starting daemon${headless_flag:+ (headless)}..."

  # Start daemon in background
  # Note: REPL is automatically skipped when stdin is not a TTY (background process)
  if [[ -n "${headless_flag}" ]]; then
    "${WEBCTL_BINARY}" start --headless &
  else
    "${WEBCTL_BINARY}" start &
  fi

  # Wait for daemon to be ready (max 10 seconds)
  local attempts=0
  local max_attempts=20
  while [[ ${attempts} -lt ${max_attempts} ]]; do
    if is_daemon_running; then
      log_success "Daemon started"
      DAEMON_STARTED_BY_TEST=true
      return 0
    fi
    sleep 0.5
    ((attempts++))
  done

  log_failure "Daemon failed to start within timeout"
  return 1
}

function stop_daemon() {
  # stop_daemon
  # Stops the daemon gracefully. Only stops if we started it.

  if [[ "${DAEMON_STARTED_BY_TEST}" != "true" ]]; then
    log_message "Daemon was not started by test, leaving running"
    return 0
  fi

  if ! is_daemon_running; then
    log_message "Daemon not running"
    return 0
  fi

  log_message "Stopping daemon..."

  if "${WEBCTL_BINARY}" stop >/dev/null 2>&1; then
    # Wait for daemon to fully stop
    local attempts=0
    local max_attempts=10
    while [[ ${attempts} -lt ${max_attempts} ]]; do
      if ! is_daemon_running; then
        log_success "Daemon stopped"
        DAEMON_STARTED_BY_TEST=false
        return 0
      fi
      sleep 0.5
      ((attempts++))
    done
    log_warning "Daemon stop timed out"
    return 1
  else
    log_failure "Daemon stop command failed"
    return 1
  fi
}

function force_stop_daemon() {
  # force_stop_daemon
  # Forces daemon to stop regardless of who started it

  if ! is_daemon_running; then
    return 0
  fi

  log_message "Force stopping daemon..."
  "${WEBCTL_BINARY}" stop --force >/dev/null 2>&1 || true
  DAEMON_STARTED_BY_TEST=false

  # Wait for daemon to fully stop
  local attempts=0
  while [[ ${attempts} -lt 10 ]]; do
    if ! is_daemon_running; then
      return 0
    fi
    sleep 0.5
    ((attempts++))
  done
  return 1
}

# Test Server Management
# -----------------------------------------------------------------------------

function start_test_server() {
  # start_test_server [port]
  # Starts webctl serve on testdata directory. Returns 0 on success.

  local port="${1:-${TEST_SERVER_PORT}}"

  if [[ -n "${TEST_SERVER_PID}" ]] && kill -0 "${TEST_SERVER_PID}" 2>/dev/null; then
    log_message "Test server already running (PID: ${TEST_SERVER_PID})"
    return 0
  fi

  if [[ ! -d "${TESTDATA_DIR}" ]]; then
    log_error "Testdata directory not found: ${TESTDATA_DIR}"
    return 1
  fi

  log_message "Starting test server on port ${port}..."

  # Start serve in background
  "${WEBCTL_BINARY}" serve "${TESTDATA_DIR}" --port "${port}" &
  TEST_SERVER_PID=$!

  # Wait for server to be ready (max 5 seconds)
  local attempts=0
  local max_attempts=10
  while [[ ${attempts} -lt ${max_attempts} ]]; do
    if curl -s "http://localhost:${port}/" >/dev/null 2>&1; then
      log_success "Test server started on port ${port}"
      return 0
    fi
    sleep 0.5
    ((attempts++))
  done

  log_failure "Test server failed to start"
  TEST_SERVER_PID=""
  return 1
}

function stop_test_server() {
  # stop_test_server
  # Stops the test server

  if [[ -z "${TEST_SERVER_PID}" ]]; then
    return 0
  fi

  if kill -0 "${TEST_SERVER_PID}" 2>/dev/null; then
    log_message "Stopping test server (PID: ${TEST_SERVER_PID})..."
    kill "${TEST_SERVER_PID}" 2>/dev/null || true
    wait "${TEST_SERVER_PID}" 2>/dev/null || true
    log_success "Test server stopped"
  fi

  TEST_SERVER_PID=""
}

function get_test_url() {
  # get_test_url [path]
  # Returns the test server URL for the given path

  local path="${1:-/}"
  echo "http://localhost:${TEST_SERVER_PORT}${path}"
}

function require_test_pages() {
  # require_test_pages page1 [page2...]
  # Verifies that required test pages exist in testdata directory.
  # Aborts with error if any page is missing.
  #
  # Args:
  #   page1, page2, ...: Paths relative to testdata (e.g., 'pages/navigation.html')
  #
  # Example: require_test_pages 'pages/navigation.html' 'pages/forms.html'

  local missing=()

  for page in "$@"; do
    local full_path="${TESTDATA_DIR}/${page}"
    if [[ ! -f "${full_path}" ]]; then
      missing+=("${page}")
    fi
  done

  if [[ ${#missing[@]} -gt 0 ]]; then
    log_error "FATAL: Required test pages not found in testdata:"
    for page in "${missing[@]}"; do
      log_error "       - ${page}"
    done
    exit 1
  fi
}

function capture_page_state() {
  # capture_page_state
  # Captures the current browser page URL and title into global variables.
  # Call this after run_test to capture state for subsequent assertions.
  # Use assert_captured_url and assert_captured_title to check these values.
  #
  # Sets: CAPTURED_PAGE_URL, CAPTURED_PAGE_TITLE

  local status_output
  status_output=$("${WEBCTL_BINARY}" status --json 2>/dev/null)
  CAPTURED_PAGE_URL=$(echo "${status_output}" | jq -r '.data.activeSession.url // empty' 2>/dev/null)
  CAPTURED_PAGE_TITLE=$(echo "${status_output}" | jq -r '.data.activeSession.title // empty' 2>/dev/null)
}

function get_captured_url() {
  # get_captured_url
  # Returns the URL captured by the last capture_page_state call
  echo "${CAPTURED_PAGE_URL}"
}

function get_captured_title() {
  # get_captured_title
  # Returns the title captured by the last capture_page_state call
  echo "${CAPTURED_PAGE_TITLE}"
}

function get_current_url() {
  # get_current_url
  # Returns the current page URL from the browser

  local status_output
  status_output=$("${WEBCTL_BINARY}" status --json 2>/dev/null)
  echo "${status_output}" | jq -r '.data.activeSession.url // empty' 2>/dev/null
}

function get_current_title() {
  # get_current_title
  # Returns the current page title from the browser

  local status_output
  status_output=$("${WEBCTL_BINARY}" status --json 2>/dev/null)
  echo "${status_output}" | jq -r '.data.activeSession.title // empty' 2>/dev/null
}

function verify_on_page() {
  # verify_on_page "url_pattern"
  # Verifies the browser is on a page matching the URL pattern.
  # Returns 0 if matched, 1 otherwise.

  local url_pattern="${1}"
  local current_url
  current_url=$(get_current_url)

  if [[ "${current_url}" == *"${url_pattern}"* ]]; then
    return 0
  else
    return 1
  fi
}

function verify_page_title() {
  # verify_page_title "expected_title"
  # Verifies the current page has the expected title.
  # Returns 0 if matched, 1 otherwise.

  local expected_title="${1}"
  local current_title
  current_title=$(get_current_title)

  if [[ "${current_title}" == "${expected_title}" ]]; then
    return 0
  else
    return 1
  fi
}

function navigate_and_verify() {
  # navigate_and_verify "url" "expected_url_pattern"
  # Navigates to URL and verifies we arrived at expected page.
  # Returns 0 on success, 1 on failure.

  local url="${1}"
  local expected_pattern="${2}"

  if ! "${WEBCTL_BINARY}" navigate --wait "${url}" >/dev/null 2>&1; then
    return 1
  fi

  verify_on_page "${expected_pattern}"
}

function setup_navigate_to() {
  # setup_navigate_to "url_path" ["expected_pattern"]
  # Navigates to a test server URL and verifies arrival. Aborts on failure.
  # Use this for setup steps that require navigation.
  #
  # Args:
  #   url_path: Path on test server (e.g., '/pages/navigation.html')
  #   expected_pattern: Optional URL pattern to verify (defaults to basename of url_path)
  #
  # Example: setup_navigate_to '/pages/forms.html'

  local url_path="${1}"
  local expected_pattern="${2:-}"

  # Default expected pattern to the filename
  if [[ -z "${expected_pattern}" ]]; then
    expected_pattern="${url_path##*/}"
  fi

  local full_url
  full_url=$(get_test_url "${url_path}")

  if ! "${WEBCTL_BINARY}" navigate --wait "${full_url}" >/dev/null 2>&1; then
    log_error "FATAL: setup_navigate_to failed to navigate to ${url_path}"
    exit 1
  fi

  if ! verify_on_page "${expected_pattern}"; then
    log_error "FATAL: setup_navigate_to verification failed - not on ${expected_pattern}"
    log_error "       Current URL: $(get_current_url)"
    exit 1
  fi
}

function setup_history_chain() {
  # setup_history_chain page1 page2 [page3...]
  # Creates a browser history chain by navigating through multiple pages.
  # Starts with navigate_to_blank for clean history boundary.
  # Aborts on any navigation failure.
  #
  # Args:
  #   page1, page2, ...: URL paths on test server (e.g., '/pages/forms.html')
  #
  # Example: setup_history_chain '/pages/nav.html' '/pages/forms.html' '/pages/cookies.html'

  if [[ $# -lt 2 ]]; then
    log_error "FATAL: setup_history_chain requires at least 2 pages"
    exit 1
  fi

  # Start with blank page for clean history boundary
  if ! navigate_to_blank; then
    log_error "FATAL: setup_history_chain failed at navigate_to_blank"
    exit 1
  fi

  # Navigate through each page
  for page in "$@"; do
    setup_navigate_to "${page}"
  done
}

function navigate_to_blank() {
  # navigate_to_blank
  # Creates a history boundary by navigating to a blank page on the test server.
  # WARNING: This ADDS to browser history, it does NOT clear it.
  # Use restart_daemon_clean for tests requiring truly empty history.
  #
  # Note: about:blank and data: URLs fail in modern Chrome via CDP due to
  # security restrictions. We use a minimal test server page instead.
  #
  # Requires: Test server must be running (start_test_server)
  # Returns: 0 on success, 1 on failure

  local blank_url
  blank_url=$(get_test_url '/pages/blank.html')

  if ! "${WEBCTL_BINARY}" navigate --wait --timeout "${TEST_TIMEOUT}" "${blank_url}" >/dev/null 2>&1; then
    log_error "navigate_to_blank failed"
    return 1
  fi
  return 0
}

# Alias for backwards compatibility (deprecated - use navigate_to_blank)
function reset_browser_state() {
  navigate_to_blank
}

function restart_daemon_clean() {
  # restart_daemon_clean [--headless]
  # Stops and restarts the daemon to get a truly clean browser state.
  # Use this for tests that require no prior navigation history.
  # Uses polling to verify daemon is fully stopped before restarting.

  local headless_flag=""
  if [[ "${1}" == "--headless" ]]; then
    headless_flag="--headless"
  fi

  # Stop daemon and wait for it to fully terminate
  force_stop_daemon

  # Poll to ensure daemon is truly stopped (max 5 seconds)
  local attempts=0
  local max_attempts=10
  while [[ ${attempts} -lt ${max_attempts} ]]; do
    if ! is_daemon_running; then
      break
    fi
    sleep 0.5
    ((attempts++))
  done

  if is_daemon_running; then
    log_error "FATAL: restart_daemon_clean failed - daemon still running after stop"
    exit 1
  fi

  # Start fresh daemon
  if ! start_daemon ${headless_flag}; then
    log_error "FATAL: restart_daemon_clean failed - could not start daemon"
    exit 1
  fi
}

# Backend Server Management
# -----------------------------------------------------------------------------

BACKEND_PID=""
BACKEND_PORT="${BACKEND_PORT:-3000}"

function start_backend() {
  # start_backend [port]
  # Starts the test backend server. Returns 0 on success.

  local port="${1:-${BACKEND_PORT}}"

  if [[ -n "${BACKEND_PID}" ]] && kill -0 "${BACKEND_PID}" 2>/dev/null; then
    log_message "Backend server already running (PID: ${BACKEND_PID})"
    return 0
  fi

  log_message "Starting backend server on port ${port}..."

  # Start backend in background
  go run "${PROJECT_ROOT}/scripts/test/backend.go" "${port}" >/dev/null 2>&1 &
  BACKEND_PID=$!

  # Wait for backend to be ready (max 5 seconds)
  local attempts=0
  local max_attempts=10
  while [[ ${attempts} -lt ${max_attempts} ]]; do
    if curl -s "http://localhost:${port}/" >/dev/null 2>&1; then
      log_success "Backend server started on port ${port}"
      return 0
    fi
    sleep 0.5
    ((attempts++))
  done

  log_failure "Backend server failed to start"
  BACKEND_PID=""
  return 1
}

function stop_backend() {
  # stop_backend
  # Stops the backend server

  if [[ -z "${BACKEND_PID}" ]]; then
    return 0
  fi

  if kill -0 "${BACKEND_PID}" 2>/dev/null; then
    log_message "Stopping backend server (PID: ${BACKEND_PID})..."
    kill "${BACKEND_PID}" 2>/dev/null || true
    wait "${BACKEND_PID}" 2>/dev/null || true
    log_success "Backend server stopped"
  fi

  BACKEND_PID=""
}

# Temporary File Management
# -----------------------------------------------------------------------------

function create_temp_file() {
  # create_temp_file [suffix]
  # Creates a temp file, registers for cleanup, returns path

  local suffix="${1:-}"
  local temp_file
  temp_file=$(mktemp "/tmp/webctl-test-XXXXXX${suffix}")
  TEMP_FILES+=("${temp_file}")
  echo "${temp_file}"
}

function create_temp_dir() {
  # create_temp_dir
  # Creates a temp directory, registers for cleanup, returns path

  local temp_dir
  temp_dir=$(mktemp -d "/tmp/webctl-test-XXXXXX")
  TEMP_FILES+=("${temp_dir}")
  echo "${temp_dir}"
}

# Cleanup Functions
# -----------------------------------------------------------------------------

function cleanup_temp_files() {
  # cleanup_temp_files
  # Removes all registered temp files and directories

  for temp_path in "${TEMP_FILES[@]}"; do
    if [[ -e "${temp_path}" ]]; then
      rm -rf "${temp_path}"
    fi
  done
  TEMP_FILES=()
}

function cleanup() {
  # cleanup
  # Full cleanup: stops servers, daemon (if we started it), removes temp files

  local exit_code=$?

  # Suppress output during cleanup
  exec 3>&2
  exec 2>/dev/null

  stop_backend
  stop_test_server
  stop_daemon
  cleanup_temp_files

  exec 2>&3
  exec 3>&-

  return ${exit_code}
}

function setup_cleanup_trap() {
  # setup_cleanup_trap
  # Sets up trap handlers for cleanup on exit/interrupt

  trap cleanup EXIT
  trap 'cleanup; exit 130' INT
  trap 'cleanup; exit 143' TERM
}

# Test Environment Setup
# -----------------------------------------------------------------------------

function setup_test_environment() {
  # setup_test_environment [--headless]
  # Full setup: builds webctl, starts daemon, sets up cleanup trap

  local headless_flag=""
  if [[ "${1}" == "--headless" ]]; then
    headless_flag="--headless"
  fi

  setup_cleanup_trap
  require_webctl || return 1
  start_daemon ${headless_flag} || return 1
}

function setup_test_environment_with_server() {
  # setup_test_environment_with_server [--headless]
  # Full setup including test server

  setup_test_environment "$@" || return 1
  start_test_server || return 1
}
