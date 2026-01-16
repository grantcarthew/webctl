# p-056: Test Library

- Status: Done
- Started: 2026-01-16
- Completed: 2026-01-16
- Design Record: .ai/design/design-records/dr-032-test-framework-architecture.md

## Overview

Create the test-specific library functions in scripts/bash_modules/. These build on the existing bash modules (p-055) to provide test counters, run_test wrapper, assertions, and setup/teardown helpers for daemon and server management.

## Goals

1. Create test-framework.sh with test counters and run_test wrapper
2. Create assertions.sh with assert_* functions for test validation
3. Create setup.sh for build, daemon, and server management

## Scope

In Scope:

- Test counter management (pass/fail/total)
- run_test wrapper that captures output and exit codes
- Assertion functions for exit codes, strings, JSON, files
- Build helpers (compile webctl)
- Daemon lifecycle management (start/stop)
- Test server management (start/stop webctl serve)
- Cleanup and trap handlers

Out of Scope:

- Test runner script (p-057)
- Test pages (p-058)
- Actual test scripts (p-059+)

## Success Criteria

- [x] scripts/bash_modules/test-framework.sh provides test counters and run_test
- [x] scripts/bash_modules/assertions.sh provides assert_* functions
- [x] scripts/bash_modules/setup.sh provides build, daemon, server helpers
- [x] All library files source without errors
- [x] Libraries integrate with bash_modules (colours, terminal, verify)
- [x] run_test captures stdout, stderr, and exit code
- [x] Assertions output clear pass/fail messages

## Deliverables

- scripts/bash_modules/test-framework.sh
- scripts/bash_modules/assertions.sh
- scripts/bash_modules/setup.sh

## Technical Approach

test-framework.sh:

- TEST_PASS, TEST_FAIL, TEST_TOTAL counters
- run_test "name" "command" - captures output, times execution
- test_summary - prints final pass/fail counts
- Integrates with terminal.sh for log_* output

assertions.sh (from DR-032):

- assert_exit_code expected actual
- assert_equals expected actual
- assert_contains haystack needle
- assert_not_contains haystack needle
- assert_json_field json jq-path expected
- assert_json_ok json
- assert_file_exists path
- assert_file_contains path needle

setup.sh:

- build_webctl - compiles binary
- start_daemon [--headless] - starts daemon, waits for ready
- stop_daemon - stops daemon gracefully
- start_test_server [port] - starts webctl serve on testdata
- stop_test_server - stops test server
- cleanup - trap handler for cleanup on exit/interrupt

## Dependencies

- p-055: Bash Modules (completed)

## Current State

Bash modules (p-055) are complete and ready:

- `scripts/bash_modules/colours.sh` - Colour constants (RED, GREEN, YELLOW, CYAN, BOLD, NORMAL) with NO_COLOR support
- `scripts/bash_modules/terminal.sh` - Log functions for test output:
  - `log_title`, `log_heading`, `log_subheading` - Section headers with lines
  - `log_success`, `log_failure` - Checkmark/cross with message (use for assertions)
  - `log_message`, `log_warning`, `log_error` - Basic output
  - `log_line`, `log_fullline` - Separator lines
  - All output to stderr (won't interfere with captured stdout)
- `scripts/bash_modules/verify.sh` - Validation functions to reuse:
  - `dependency_check cmd...` - Verify required commands exist
  - `is_json string` - Validate JSON (returns 0/1)
  - `file_exists path` - Check file exists (returns 0/1)
  - `directory_exists path` - Check directory exists (returns 0/1)
- `scripts/bash_modules/user-input.sh` - Interactive prompts (not needed for automated tests)

Existing resources:

- `testdata/index.html` - Test page with console buttons (log/warn/error/info), network request button
- `testdata/backend.go` - Backend server on port 3000 with /api/hello, /api/users, /status/{code}, /delay endpoints
- `scripts/interactive/` - 25 manual test scripts (patterns to reference)
- webctl binary builds and runs with all commands functional

External dependencies:

- `jq` - Required for JSON assertions (already used in verify.sh)
- `go` - Required for build_webctl function

Key integration points:

- terminal.sh provides log_success/log_failure for assertion output
- verify.sh provides is_json, file_exists, dependency_check for reuse
- `webctl status` returns "Not running (start with: webctl start)" when daemon is down
- `webctl status` returns daemon info with URL when running
- `webctl serve testdata` serves test pages on default port 8888 (per DR-032)
- `webctl stop` gracefully stops daemon, returns exit code 0

## Decisions

1. **Assertion output verbosity:** Verbose - always show pass/fail messages for each assertion (matches DR-032 output format)

2. **Library location:** Placed in `scripts/bash_modules/` rather than `scripts/test/lib/` to keep all bash modules centralised for this single project

## Notes

- Test library should use stderr for all logging (via terminal.sh)
- Daemon start should verify it's running before returning
- All temporary files should be cleaned up on exit
- Test failure behaviour: continue running remaining tests (collect all failures)
