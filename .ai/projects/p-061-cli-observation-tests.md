# p-061: CLI Observation Tests

- Status: Pending
- Started:
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

- [ ] scripts/test/cli/test-observation.sh created
- [ ] Tests pass with ./test-runner cli observation
- [ ] html command tests: basic output, selector, find, save
- [ ] css command tests: basic output, selector, computed, get, matched
- [ ] console command tests: basic output, type filter
- [ ] network command tests: basic output, status filter, method filter
- [ ] cookies command tests: basic output, set, delete
- [ ] screenshot command tests: save to file

## Deliverables

- scripts/test/cli/test-observation.sh

## Technical Approach

Test structure:

- Source shared modules (test-framework.sh, assertions.sh, setup.sh)
- Start daemon and test server for all tests
- Navigate to appropriate test pages before each observation test
- Use run_test wrapper for consistent output capture
- Test text output (default)

Key test scenarios:

1. html - navigate to page, verify output contains expected content
2. html --select - verify filtered output
3. css - verify stylesheet output
4. css computed - verify computed styles for element
5. console - navigate to console-types.html, verify log capture
6. network - trigger requests, verify capture
7. cookies - set cookie, read, delete, verify
8. screenshot save - verify file created

## Current State

### Test Framework

- test-runner at project root dispatches to scripts/test/cli/ directory
- scripts/bash_modules/ contains shared modules:
  - test-framework.sh: `run_test`, `test_section`, `test_summary`, test counters
  - assertions.sh: `assert_success`, `assert_failure`, `assert_contains`, `assert_file_exists`, etc.
  - setup.sh: `setup_test_environment_with_server`, `start_daemon`, `start_test_server`, `get_test_url`, cleanup
- scripts/test/cli/test-navigation.sh demonstrates test patterns

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

### Command Flags to Test

**html**: `--select <selector>`, `--find <text>`, `save [path]`
**css**: `--select <selector>`, `--find <text>`, `save`, `computed <sel>`, `get <sel> <prop>`, `matched <sel>`
**console**: `--type <type>`, `--find <text>`, `--head N`, `--tail N`, `save`
**network**: `--status <code>`, `--method <method>`, `--type <type>`, `--find <text>`, `save`
**cookies**: `--domain <domain>`, `--find <text>`, `set <name> <value>`, `delete <name>`, `save`
**screenshot**: default (to temp), `save [path]`, `--full-page`

### Test Pages

Available test pages in testdata/pages/:

- **navigation.html** - basic HTML structure for html command tests
- **css-showcase.html** - CSS styling with testable selectors:
  - IDs: `#unique-element`, `#special-box`
  - Classes: `.highlight`, `.primary-btn`, `.card`
  - Elements: `h1`, `body`, `p`
- **console-types.html** - auto-triggers on load:
  - `AUTOLOAD_LOG`, `AUTOLOAD_INFO`, `AUTOLOAD_WARN`, `AUTOLOAD_ERROR`
- **cookies.html** - sets `initial-cookie=loaded` on page load
- **network-requests.html** - uses API endpoints served via proxy

### Test Backend

`scripts/test/backend.go` provides API endpoints for network testing:
- `/api/hello`, `/api/users`, `/api/echo` - JSON responses
- `/status/200|400|404|500` - status code testing
- `/delay` - 2s delayed response

Usage in tests:
```bash
# Start backend on port 3000
go run scripts/test/backend.go 3000 &
BACKEND_PID=$!

# Start webctl with proxy to backend
webctl serve testdata --proxy http://localhost:3000

# Navigate to network-requests.html, trigger requests, test network command
```

## Dependencies

- p-055: Test Framework Bash Modules (completed)
- p-056: Test Library (completed)
- p-057: Test Runner (completed)
- p-058: Test Pages (completed)
- p-060: CLI Navigation Tests (completed)

## Notes

- Observation tests require a running daemon and test server
- Some tests need specific page content (console-types for console logs)
- Screenshot tests should verify file exists and has content
- Clean up saved files in cleanup trap
- Network tests require starting scripts/test/backend.go and using --proxy mode
- Add backend start/stop helpers to setup.sh or handle in test script
