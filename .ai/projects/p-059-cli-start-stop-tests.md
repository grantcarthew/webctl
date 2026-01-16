# p-059: CLI Start/Stop Tests

- Status: Pending
- Started:
- Design Record: .ai/design/design-records/dr-032-test-framework-architecture.md

## Overview

Create automated tests for the webctl start and stop commands. These tests verify daemon lifecycle management including starting, stopping, status checking, and error handling.

## Goals

1. Test start command with various options (headless, ports, attach)
2. Test stop command including force stop
3. Test status command output
4. Verify daemon lifecycle (start, check running, stop, confirm stopped)
5. Test error conditions (already running, not running, port conflicts)

## Scope

In Scope:

- scripts/test/cli/test-start-stop.sh
- Tests for: start, stop, status commands
- Headless and headed modes
- Port configuration options
- Force stop functionality
- Error handling scenarios

Out of Scope:

- Navigation command tests (p-060)
- Observation command tests (p-061)
- Browser launch testing (covered by start command)

## Success Criteria

- [ ] scripts/test/cli/test-start-stop.sh created
- [ ] Tests pass with ./test-runner cli start-stop
- [ ] start command tests: basic, headless, custom ports
- [ ] stop command tests: basic, force, already stopped
- [ ] status command tests: running, not running
- [ ] Error handling: port in use, daemon already running

## Deliverables

- scripts/test/cli/test-start-stop.sh

## Technical Approach

Test structure:

- Source shared modules (test-framework.sh, assertions.sh, setup.sh)
- Use run_test wrapper for consistent output
- Clean state before each test group
- Test both success and error paths

Key test scenarios:

1. Start daemon (headless) - verify status shows running
2. Start when already running - expect error
3. Stop daemon - verify status shows not running
4. Stop when not running - expect graceful handling
5. Force stop - verify cleanup
6. Custom port configuration

## Current State

- test-runner script exists and can dispatch to cli tests
- bash_modules/ contains test-framework.sh, assertions.sh, setup.sh
- scripts/test/cli/ directory needs to be created
- testdata/pages/ has test pages for navigation tests

## Dependencies

- p-055: Test Framework Bash Modules (completed)
- p-056: Test Library (completed)
- p-057: Test Runner (completed)
- p-058: Test Pages (completed)

## Notes

- Start with simple success path tests
- Add error handling tests incrementally
- Use headless mode for CI compatibility
- Clean up daemon state between test groups
