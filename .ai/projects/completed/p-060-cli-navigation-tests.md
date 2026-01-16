# p-060: CLI Navigation Tests

- Status: Done
- Started: 2026-01-16
- Completed: 2026-01-16
- Design Record: .ai/design/design-records/dr-032-test-framework-architecture.md

## Overview

Create automated tests for the webctl navigation commands. These tests verify browser navigation including URL navigation, reload, back, and forward functionality.

## Goals

1. Test navigate command with various URL types
2. Test reload command
3. Test back command (history navigation)
4. Test forward command (history navigation)
5. Test error conditions (invalid URLs, no history)

## Scope

In Scope:

- scripts/test/cli/test-navigation.sh
- Tests for: navigate, reload, back, forward commands
- URL navigation (http, file)
- History navigation
- Error handling: invalid URL, no history available

Out of Scope:

- Observation command tests (p-061)
- Interaction command tests (future)
- HTTPS certificate handling

## Success Criteria

- [x] scripts/test/cli/test-navigation.sh created
- [x] Tests pass with ./test-runner cli navigation
- [x] navigate command tests: basic URL, file URL, invalid URL
- [x] reload command tests: basic reload
- [x] back command tests: with history, without history
- [x] forward command tests: with history, without history

## Deliverables

- scripts/test/cli/test-navigation.sh

## Technical Approach

Test structure:

- Source shared modules (test-framework.sh, assertions.sh, setup.sh)
- Use setup_test_environment_with_server for tests requiring the test server
- Use run_test wrapper for consistent output capture
- Test text output (default)

Script outline:

```bash
#!/usr/bin/env bash
source "${PROJECT_ROOT}/scripts/bash_modules/test-framework.sh"
source "${PROJECT_ROOT}/scripts/bash_modules/assertions.sh"
source "${PROJECT_ROOT}/scripts/bash_modules/setup.sh"

setup_cleanup_trap
require_webctl
start_daemon --headless
start_test_server

# Test sections:
# 1. Navigate - basic URL navigation
# 2. Navigate - file URL
# 3. Navigate - invalid URL error
# 4. Reload - basic page reload
# 5. Back - navigate back in history
# 6. Back - no history error
# 7. Forward - navigate forward in history
# 8. Forward - no history error

test_summary
```

Key test scenarios:

1. Navigate to test server URL - expect "OK" in stdout, exit 0
2. Navigate to file:// URL - expect "OK" in stdout, exit 0
3. Navigate to invalid URL - expect error in stdout/stderr, exit 1
4. Reload current page - expect "OK" in stdout, exit 0
5. Back after navigation - expect "OK" in stdout, exit 0
6. Back with no history - expect "No previous page" in stderr, exit 1
7. Forward after back - expect "OK" in stdout, exit 0
8. Forward with no forward history - expect "No next page" in stderr, exit 1

## Current State

### Test Framework

- test-runner at project root dispatches to scripts/test/cli/ directory
- scripts/bash_modules/ contains shared modules:
  - test-framework.sh - run_test wrapper, TEST_STDOUT/STDERR/EXIT_CODE vars, test_summary
  - assertions.sh - assert_success, assert_failure, assert_contains, assert_equals, etc.
  - setup.sh - daemon and test server management, cleanup handlers
- scripts/test/cli/test-start-stop.sh - existing test file (17 tests) demonstrating the pattern

### CLI Commands

navigate command (internal/cli/navigate.go):
- Arguments: URL (required)
- Text output: "OK" on success (stdout)
- Error output: navigation failure message (stderr, exit 1)
- URL normalisation: adds https:// by default, http:// for localhost/127.0.0.1

reload command (internal/cli/reload.go):
- No arguments
- Text output: "OK" on success (stdout)
- Error output: reload failure message (stderr, exit 1)

back command (internal/cli/back.go):
- No arguments
- Text output: "OK" on success (stdout)
- No history: "No previous page" (stderr via outputNotice, exit 1)
- Note: Uses outputNotice - no "Error:" prefix, just the message

forward command (internal/cli/forward.go):
- No arguments
- Text output: "OK" on success (stdout)
- No history: "No next page" (stderr via outputNotice, exit 1)
- Note: Uses outputNotice - no "Error:" prefix, just the message

### Test Server

- start_test_server starts webctl serve on testdata directory (port 8888)
- get_test_url returns test server URLs (e.g., get_test_url "/pages/navigation.html")
- testdata/pages/ contains test pages:
  - navigation.html - links to other pages (ideal for history tests)
  - forms.html, cookies.html, console-types.html, etc.

### Test Pattern (from test-start-stop.sh)

```bash
source "${PROJECT_ROOT}/scripts/bash_modules/test-framework.sh"
source "${PROJECT_ROOT}/scripts/bash_modules/assertions.sh"
source "${PROJECT_ROOT}/scripts/bash_modules/setup.sh"

setup_cleanup_trap
require_webctl
# ... test code using run_test, assert_*, test_section, test_summary
```

## Dependencies

- p-055: Test Framework Bash Modules (completed)
- p-056: Test Library (completed)
- p-057: Test Runner (completed)
- p-058: Test Pages (completed)
- p-059: CLI Start/Stop Tests (completed)

## Notes

- Navigation tests require a running daemon and test server
- Test history navigation by navigating to multiple pages first
- Use headless mode for CI compatibility
- Clean up daemon state in cleanup trap
