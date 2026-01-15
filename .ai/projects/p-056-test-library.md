# p-056: Test Library

- Status: Pending
- Started:
- Design Record: .ai/design/design-records/dr-032-test-framework-architecture.md

## Overview

Create the test-specific library functions in scripts/test/lib/. These build on the bash modules (p-055) to provide test counters, run_test wrapper, assertions, and setup/teardown helpers for daemon and server management.

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

- [ ] scripts/test/lib/test-framework.sh provides test counters and run_test
- [ ] scripts/test/lib/assertions.sh provides assert_* functions
- [ ] scripts/test/lib/setup.sh provides build, daemon, server helpers
- [ ] All library files source without errors
- [ ] Libraries integrate with bash_modules (colours, terminal, verify)
- [ ] run_test captures stdout, stderr, and exit code
- [ ] Assertions output clear pass/fail messages

## Deliverables

- scripts/test/lib/test-framework.sh
- scripts/test/lib/assertions.sh
- scripts/test/lib/setup.sh

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

- `scripts/bash_modules/colours.sh` - Colour definitions with NO_COLOR support
- `scripts/bash_modules/terminal.sh` - Log functions (log_success, log_failure, log_heading, etc.)
- `scripts/bash_modules/verify.sh` - Validation functions (is_json, file_exists, dependency_check, etc.)
- `scripts/bash_modules/user-input.sh` - User prompt functions

Directory structure to create:

- `scripts/test/` - Does not exist yet
- `scripts/test/lib/` - Target for test library files

Existing resources:

- `testdata/index.html` - Basic test page
- `testdata/backend.go` - Test backend server
- `scripts/interactive/` - 25 manual test scripts (to be replaced by automated tests)
- webctl binary builds and runs with all commands functional

Key integration points:

- terminal.sh provides log_success/log_failure for assertion output
- verify.sh provides is_json, file_exists for reuse in assertions
- webctl status command can verify daemon is running
- webctl serve provides test server capability

## Decisions

1. **Assertion output verbosity:** Verbose - always show pass/fail messages for each assertion (matches DR-032 output format)

## Notes

- Test library should use stderr for all logging (via terminal.sh)
- Daemon start should verify it's running before returning
- All temporary files should be cleaned up on exit
- Test failure behaviour: continue running remaining tests (collect all failures)
