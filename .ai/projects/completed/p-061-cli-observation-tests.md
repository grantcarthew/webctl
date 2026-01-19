# p-061: CLI Observation Tests

- Status: Done
- Started: 2026-01-17
- Completed: 2026-01-17
- Design Record: .ai/design/design-records/dr-032-test-framework-architecture.md

## Overview

Create automated tests for the webctl observation commands. These tests verify data capture and output functionality for html, css, console, network, cookies, and screenshot commands.

## Goals

1. Test html command with various selectors and output modes
2. Test css command with selectors and property queries
3. Test console command with type filtering
4. Test network command with status and method filtering
5. Test cookies command with domain filtering and set/delete
6. Test screenshot command with save functionality
7. Test error conditions for each command

## Scope

In Scope:

- scripts/test/cli/test-observation.sh
- Tests for: html, css, console, network, cookies, screenshot commands
- Output modes: stdout, save to file, save to temp
- Filtering options (--select, --find, --type, --status, etc.)
- Error handling: invalid selectors, no data

Out of Scope:

- Interaction command tests (p-062)
- JSON output mode testing (future)
- Performance benchmarks

## Success Criteria

- [x] scripts/test/cli/test-observation.sh created
- [x] Tests pass with ./test-runner cli observation (80/80 passing)
- [x] html command tests: basic output, selector, find, save
- [x] css command tests: basic output, selector, computed, get (inline/matched commands don't exist)
- [x] console command tests: basic output, type filter
- [x] network command tests: basic output, status filter, method filter
- [x] cookies command tests: basic output, set, delete
- [x] screenshot command tests: save to file

## Deliverables

- Backend management functions in scripts/bash_modules/setup.sh (start_backend, stop_backend)
- scripts/test/cli/test-observation.sh

## Technical Approach

Implementation order:

1. **First**: Add backend management to setup.sh:
   - `start_backend([port])` - starts scripts/test/backend.go, waits for ready
   - `stop_backend()` - stops backend gracefully
   - Pattern mirrors start_test_server/stop_test_server (see Decision 1)
   - Required for network command tests

2. **Then**: Create test-observation.sh following established patterns

Test structure:

- Source shared modules (test-framework.sh, assertions.sh, setup.sh)
- Start daemon and test server for all tests
- Start backend server using start_backend() helper for network tests
- Navigate to appropriate test pages before each observation test
- Use run_test wrapper for consistent output capture
- Test all output modes: stdout (default), save to temp, save to file, save to directory

Key test scenarios (comprehensive coverage):

**html command:**
- Basic output to stdout
- Selector filtering (--select)
- Text search (--find)
- Save modes: temp, custom file, directory with trailing slash
- Error cases: invalid selectors, no matches

**css command:**
- Basic stylesheets to stdout
- Selector filtering for rules (--select)
- Text search (--find)
- Subcommands: computed, get, inline, matched
- Save modes: temp, custom file, directory with trailing slash
- Error cases: invalid selectors, no matches

**console command:**
- Basic output to stdout
- Type filtering (--type log,warn,error,info,debug)
- Text search (--find)
- Range limiting (--head, --tail)
- Save modes: temp, custom file, directory with trailing slash
- Error cases: no logs, invalid type

**network command:**
- Basic output to stdout
- Status filtering (--status 200, 4xx, 5xx)
- Method filtering (--method GET, POST)
- Text search (--find)
- Save modes: temp, custom file, directory with trailing slash
- Backend-triggered requests via proxy

**cookies command:**
- Basic output to stdout
- Set/delete operations
- Domain filtering (--domain)
- Save modes: temp, custom file, directory with trailing slash
- Verify initial-cookie from cookies.html page load

**screenshot command:**
- Default viewport to temp
- Save to custom path
- Full-page capture (--full-page)
- Verify file exists and has non-zero size
- Test both PNG file modes

## Current State

### Environment

- **Go Version**: 1.25.6 (project requires 1.25.5+)
- **Platform**: Linux/amd64 with bash support
- **Dependencies**: Standard (cobra, websocket, color, readline) - all available

### Test Framework (Ready)

All prerequisite projects completed (p-055 through p-060):

- **test-runner** at project root dispatches to scripts/test/cli/ directory
- **scripts/bash_modules/** contains shared testing infrastructure:
  - `test-framework.sh`: `run_test`, `test_section`, `test_summary`, test counters, color output
  - `assertions.sh`: `assert_success`, `assert_failure`, `assert_contains`, `assert_file_exists`, etc.
  - `setup.sh`: `setup_cleanup_trap`, `start_daemon`, `start_test_server`, `get_test_url`, cleanup handlers
- **Reference implementations**: test-navigation.sh (148 lines) and test-start-stop.sh (171 lines) demonstrate patterns
- **Test runner integration**: `./test-runner cli observation` will execute the test suite

### Test Pattern (from test-navigation.sh)

```bash
# 1. Import modules
source "${PROJECT_ROOT}/scripts/bash_modules/test-framework.sh"
source "${PROJECT_ROOT}/scripts/bash_modules/assertions.sh"
source "${PROJECT_ROOT}/scripts/bash_modules/setup.sh"

# 2. Setup
setup_cleanup_trap
require_webctl
force_stop_daemon
start_daemon --headless
start_test_server

# 3. Test sections
test_section "Command Name"
run_test "test description" "${WEBCTL_BINARY}" command args
assert_success "${TEST_EXIT_CODE}" "message"
assert_contains "${TEST_STDOUT}" "expected" "message"

# 4. Summary
test_summary
```

### Observation Commands (Mature Implementation)

All commands have extensive interactive test coverage (scripts/interactive/test-*.sh) showing mature implementations:

**html command** (314 lines interactive tests):
- Flags: `--select <selector>`, `--find <text>`, `-A`, `-B`, `-C` (context), `--raw`, `save [path]`
- Modes: stdout (default), save to temp, save to file, save to directory
- Tested: selector filtering, text search, context lines, error cases

**css command** (486 lines interactive tests):
- Flags: `--select <pattern>` (filters CSS rules), `--find <text>`, `-A`, `-B`, `-C`, `--raw`, `save [path]`
- Modes: default (all stylesheets), `computed <selector>`, `get <selector> <property>`, `inline <selector>`, `matched <selector>`
- Tested: rule filtering, computed styles for multiple elements, single property queries, inline styles, matched rules

**console command** (407 lines interactive tests):
- Flags: `--type <type>` (log,info,warn,error,debug), `--find <text>`, `--head N`, `--tail N`, `--range N-M`, `save [path]`
- Modes: stdout (default), save to temp, save to file
- Tested: type filtering, text search, range selection, mutual exclusivity of head/tail/range

**network command** (extensive interactive tests):
- Flags: `--status <code>`, `--method <method>`, `--type <type>`, `--find <text>`, `--head N`, `--tail N`, `save [path]`
- Modes: stdout (default), save to temp, save to file
- Tested: status code filtering (4xx, 5xx), method filtering, type filtering

**cookies command** (interactive tests available):
- Flags: `--domain <domain>`, `--find <text>`, `save [path]`
- Commands: `set <name> <value>`, `delete <name>`
- Tested: domain filtering, set/delete operations

**screenshot command** (interactive tests available):
- Flags: `--full-page`, `save [path]`
- Modes: default (viewport to temp), save to custom path, full-page capture
- Output: Binary PNG files with auto-generated filenames (YY-MM-DD-HHMMSS-{title}.png)

### Test Pages (Available)

Test pages in testdata/pages/ are ready:

- **navigation.html** - basic HTML structure for html command tests
- **css-showcase.html** - comprehensive CSS styling with testable selectors:
  - IDs: `#unique-element`, `#special-box`
  - Classes: `.highlight`, `.primary-btn`, `.secondary-btn`, `.card`, `.hover-example`
  - Elements: `h1`, `h2`, `body`, `p`, `div`, `span`, `input`, `button`
  - Advanced: flexbox, grid, animations, pseudo-classes, media queries, CSS variables
- **console-types.html** - auto-triggers console messages on load:
  - `AUTOLOAD_LOG`, `AUTOLOAD_INFO`, `AUTOLOAD_WARN`, `AUTOLOAD_ERROR`
  - Interactive buttons for all console methods (log, info, warn, error, debug, table, group, count, time, assert)
- **cookies.html** - sets `initial-cookie=loaded` on page load
  - Interactive buttons for set, get, delete, clear operations
  - Session and persistent cookie testing
- **network-requests.html** - trigger API requests via fetch
  - Requires proxy to backend for API endpoints

### Test Backend (Available)

`scripts/test/backend.go` provides API endpoints for network testing:
- **API endpoints**: `/api/hello`, `/api/users`, `/api/echo` - JSON responses
- **Status testing**: `/status/200`, `/status/400`, `/status/404`, `/status/500`
- **Delay testing**: `/delay` - 2s delayed response
- **CORS enabled**: All endpoints have CORS headers for cross-origin testing

Usage pattern for network tests:
```bash
# Start backend on port 3000
go run scripts/test/backend.go 3000 &
BACKEND_PID=$!

# Start webctl serve with proxy to backend
webctl serve testdata --proxy http://localhost:3000 &
SERVER_PID=$!

# Navigate to network-requests.html and interact
webctl navigate http://localhost:8888/pages/network-requests.html

# Test network command
webctl network
webctl network --status 200
webctl network --method GET

# Cleanup
kill $BACKEND_PID $SERVER_PID
```

### Implementation Readiness

- ✅ All dependencies completed
- ✅ Test framework and modules ready
- ✅ Test pages and backend available
- ⚠️ Backend management functions (start_backend/stop_backend) need to be added to setup.sh first
- ✅ Reference test patterns established
- ✅ Commands are mature with known behaviors
- ✅ Interactive test scripts document expected outputs
- ✅ All implementation decisions approved (see Decisions Made)

## Decisions Made

The following decisions have been approved:

### 1. Backend Server Management for Network Tests
**Selected: Option A - Add helpers to setup.sh**

Implementation:
- Create `start_backend()` and `stop_backend()` functions in scripts/bash_modules/setup.sh
- Backend management will be reusable for future test projects
- Keeps test-observation.sh focused on assertions
- Consistent with existing patterns (start_daemon, start_test_server)

### 2. Screenshot Test Scope
**Selected: Option A - Include comprehensive screenshot tests**

Implementation:
- Test default behavior (viewport to temp)
- Test save to custom path
- Test --full-page flag
- Verify file exists and has non-zero size
- Matches project success criteria

### 3. Save Mode Testing Strategy
**Selected: Option B - Test all save modes for all commands**

Implementation:
- Test all save modes (stdout, temp, file, directory) for html, css, console, network, cookies
- Screenshot saves by default, test both default and custom path
- Comprehensive coverage even though save logic is shared
- Test suite will exceed 500+ lines but provides thorough validation

## Dependencies

- p-055: Test Framework Bash Modules (completed)
- p-056: Test Library (completed)
- p-057: Test Runner (completed)
- p-058: Test Pages (completed)
- p-060: CLI Navigation Tests (completed)

## Notes

- Observation tests require a running daemon and test server
- Some tests need specific page content (console-types.html for console logs, css-showcase.html for CSS tests)
- Screenshot tests verify file exists and has non-zero size (binary PNG output)
- Clean up saved files and temp directories in cleanup trap
- Network tests require backend server managed via start_backend()/stop_backend() helpers in setup.sh
- Test suite will be comprehensive (500+ lines) testing all save modes for all commands
- Reference interactive test scripts (scripts/interactive/test-*.sh) for expected command outputs and edge cases

### Backend Function Implementation Guide

The start_backend() and stop_backend() functions should follow the same pattern as start_test_server():

```bash
# Configuration
BACKEND_PID=""
BACKEND_PORT="${BACKEND_PORT:-3000}"

start_backend([port]) {
  # Check if already running
  # Start with: go run scripts/test/backend.go <port> &
  # Store PID in BACKEND_PID
  # Wait for ready: curl -s http://localhost:<port>/ >/dev/null
  # Log success
}

stop_backend() {
  # Kill BACKEND_PID if set
  # Wait for process to stop
  # Clear BACKEND_PID
}
```

Add to cleanup() function to ensure backend stops on test exit.
