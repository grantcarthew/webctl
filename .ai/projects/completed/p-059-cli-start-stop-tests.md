# p-059: CLI Start/Stop Tests

- Status: Done
- Started: 2026-01-16
- Completed: 2026-01-16
- Design Record: .ai/design/design-records/dr-032-test-framework-architecture.md

## Overview

Create automated tests for the webctl start and stop commands. These tests verify daemon lifecycle management including starting, stopping, status checking, and error handling.

## Goals

1. Test start command with various options (headless, custom ports)
2. Test stop command including force stop
3. Test status command output
4. Verify daemon lifecycle (start, check running, stop, confirm stopped)
5. Test error conditions (already running, not running)

## Scope

In Scope:

- scripts/test/cli/test-start-stop.sh
- Tests for: start, stop, status commands
- Headless mode (headed mode requires display)
- Custom port configuration
- Force stop functionality
- Error handling: already running, not running

Out of Scope:

- Navigation command tests (p-060)
- Observation command tests (p-061)
- Browser launch testing (covered by start command)

## Success Criteria

- [x] scripts/test/cli/test-start-stop.sh created
- [x] Tests pass with ./test-runner cli start-stop
- [x] start command tests: basic, headless, custom port
- [x] stop command tests: basic, force, already stopped
- [x] status command tests: running, not running
- [x] Error handling: daemon already running

## Deliverables

- scripts/test/cli/test-start-stop.sh

## Technical Approach

Test structure:

- Source shared modules (test-framework.sh, assertions.sh, setup.sh)
- Use run_test wrapper for consistent output capture
- Use force_stop_daemon before each test group to ensure clean state
- Use setup_cleanup_trap for proper teardown
- Test text output (default) - JSON output is consistent and lower priority

Script outline:

```bash
#!/usr/bin/env bash
source "${PROJECT_ROOT}/scripts/bash_modules/test-framework.sh"
source "${PROJECT_ROOT}/scripts/bash_modules/assertions.sh"
source "${PROJECT_ROOT}/scripts/bash_modules/setup.sh"

setup_cleanup_trap
require_webctl

# Test sections:
# 1. Status (not running) - baseline
# 2. Start (headless) - basic success
# 3. Status (running) - verify started
# 4. Start (already running) - error case
# 5. Stop (graceful) - basic success
# 6. Status (not running) - verify stopped
# 7. Stop (not running) - error case
# 8. Force stop (cleanup) - force mode
# 9. Custom port - verify port flag works

test_summary
```

Key test scenarios:

1. Status when not running - expect "Not running" text
2. Start daemon (headless) - expect "OK" and exit 0
3. Status when running - expect "OK" text
4. Start when already running - expect error and exit 1
5. Stop daemon - expect "OK" and exit 0
6. Stop when not running - expect error and exit 1
7. Force stop - expect cleanup or "Nothing to clean up"
8. Custom port (--port 9333) - verify daemon starts on alternate port

## Current State

### Test Framework

- `test-runner` at project root dispatches to `scripts/test/cli/` directory
- `scripts/bash_modules/` contains shared modules:
  - `test-framework.sh` - run_test wrapper, TEST_STDOUT/STDERR/EXIT_CODE vars, test_summary
  - `assertions.sh` - assert_success, assert_failure, assert_contains, assert_equals, assert_json_field, etc.
  - `setup.sh` - require_webctl, start_daemon, stop_daemon, force_stop_daemon, is_daemon_running, setup_cleanup_trap
- `scripts/test/cli/test-start-stop.sh` - 17 tests covering start, stop, status commands

### CLI Commands

**start command** (`internal/cli/start.go`):
- Flags: `--headless` (bool), `--port` (int, default 9222)
- Text output: "OK" on success
- JSON output: `{"ok":true,"data":{"message":"daemon starting","port":9222}}`
- Error output: "Error: daemon is already running" (exit 1)
- Daemon blocks until shutdown; REPL only starts if stdin is TTY

**stop command** (`internal/cli/stop.go`):
- Flags: `--force` (bool), `--port` (int, default 9222, used with --force)
- Text output (graceful): "OK"
- Text output (force): action list or "Nothing to clean up"
- JSON output: `{"ok":true,"data":{"message":"daemon stopped"}}`
- Error output: "Error: daemon not running or not responding" (exit 1)

**status command** (`internal/cli/status.go`):
- No command-specific flags
- Text output (not running): "Not running (start with: webctl start)"
- Text output (running): "OK" with URL, title, PID
- JSON output: `{"ok":true,"data":{"running":true,...}}` or `{"ok":true,"data":{"running":false}}`

### Key Helper Functions (setup.sh)

- `is_daemon_running` - checks status for "Not running" text
- `start_daemon [--headless]` - starts in background, waits up to 10s
- `stop_daemon` - graceful stop (only if test started it)
- `force_stop_daemon` - force stop regardless of who started it
- `setup_cleanup_trap` - registers EXIT/INT/TERM handlers

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

## Implementation Notes

Fixed a bug in `test-framework.sh` where `run_test` would enable `set -e` (errexit)
even when it wasn't previously enabled. The `set +e` / `set -e` pair now saves and
restores the errexit state properly.

Test coverage:
- Status command: 2 tests (not running, running)
- Start command: 3 tests (headless start, already running error, custom port)
- Stop command: 5 tests (graceful, verify stopped, not running error)
- Force stop: 3 tests (running daemon, verify cleanup, nothing to clean)
- Custom port: 2 tests (start on 9333, force stop)

Total: 17 tests, all passing
