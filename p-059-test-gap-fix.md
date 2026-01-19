# p-059 Test Gap Fix

This document details missing tests in `scripts/test/cli/test-start-stop.sh` that need to be added for comprehensive coverage comparable to p-061 (CLI Observation Tests).

## Current State

- **Test file:** `scripts/test/cli/test-start-stop.sh` (171 lines)
- **Current tests:** 17
- **Commands covered:** start, stop, status

## Commands Under Test

### start command (`internal/cli/start.go`)

**Flags:**
- `--headless` (bool, default false) - Run browser in headless mode
- `--port` (int, default 9222) - CDP port for browser

**Output modes:**
- Text: "OK" on success
- JSON: `{"ok":true,"data":{"message":"daemon starting","port":9222}}`

**Error cases:**
- Already running: "Error: daemon is already running" (exit 1)
- Hint: "use 'webctl stop' to stop the daemon, or 'webctl stop --force' to force cleanup"

### stop command (`internal/cli/stop.go`)

**Flags:**
- `--force` (bool, default false) - Force kill processes and clean up stale files
- `--port` (int, default 9222) - CDP port for browser process discovery (used with --force)

**Output modes:**
- Text (graceful): "OK"
- Text (force): List of actions or "Nothing to clean up"
- JSON (graceful): `{"ok":true,"data":{"message":"daemon stopped"}}`
- JSON (force): `{"ok":true,"data":{"message":"force cleanup complete","actions":[...]}}`

**Force cleanup actions reported:**
- "killed daemon (PID X)"
- "killed browser (PID X) on port Y"
- "removed socket file"
- "removed PID file"

**Error cases:**
- Not running: "Error: daemon not running or not responding" (exit 1)

### status command (`internal/cli/status.go`)

**Flags:**
- None command-specific (uses global --json, --no-color)

**Output modes:**
- Text (not running): "Not running (start with: webctl start)"
- Text (running): "OK" followed by "pid: X" and "sessions:" with URLs
- Text (no browser): "No browser" followed by "pid: X"
- Text (no session): "No session" followed by "pid: X"
- JSON (not running): `{"ok":true,"data":{"running":false}}`
- JSON (running): `{"ok":true,"data":{"running":true,"pid":X,"activeSession":{...},"sessions":[...]}}`

### Global Flags

- `--json` - Output in JSON format
- `--no-color` - Disable color output
- `--debug` - Enable verbose debug output

## Currently Tested

| # | Test | Status |
|---|------|--------|
| 1 | Status when not running | ✓ |
| 2 | Start daemon (headless) | ✓ |
| 3 | Status when running | ✓ |
| 4 | Start when already running (error) | ✓ |
| 5 | Stop daemon (graceful) | ✓ |
| 6 | Verify daemon stopped | ✓ |
| 7 | Stop when not running (error) | ✓ |
| 8 | Force stop running daemon | ✓ |
| 9 | Verify force stop worked | ✓ |
| 10 | Force stop when nothing running | ✓ |
| 11 | Start with custom port (9333) | ✓ |
| 12 | Force stop custom port | ✓ |

## Missing Tests

### JSON Output Mode (5 tests)

1. **start --json output format**
   ```bash
   run_test "start --json output" "${WEBCTL_BINARY}" start --headless --json
   assert_success "${TEST_EXIT_CODE}" "start --json returns success"
   assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
   assert_json_field "${TEST_STDOUT}" ".data.message" "daemon starting" "JSON message field"
   assert_json_field "${TEST_STDOUT}" ".data.port" "9222" "JSON port field"
   ```

2. **stop --json output format**
   ```bash
   run_test "stop --json output" "${WEBCTL_BINARY}" stop --json
   assert_success "${TEST_EXIT_CODE}" "stop --json returns success"
   assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
   assert_json_field "${TEST_STDOUT}" ".data.message" "daemon stopped" "JSON message field"
   ```

3. **stop --force --json output format**
   ```bash
   run_test "stop --force --json output" "${WEBCTL_BINARY}" stop --force --json
   assert_success "${TEST_EXIT_CODE}" "stop --force --json returns success"
   assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
   ```

4. **status --json when running**
   ```bash
   run_test "status --json when running" "${WEBCTL_BINARY}" status --json
   assert_success "${TEST_EXIT_CODE}" "status --json returns success"
   assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
   assert_json_field "${TEST_STDOUT}" ".data.running" "true" "JSON running field is true"
   assert_contains "${TEST_STDOUT}" "pid" "JSON contains pid field"
   ```

5. **status --json when not running**
   ```bash
   run_test "status --json when not running" "${WEBCTL_BINARY}" status --json
   assert_success "${TEST_EXIT_CODE}" "status --json returns success"
   assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
   assert_json_field "${TEST_STDOUT}" ".data.running" "false" "JSON running field is false"
   ```

### Status Content Verification (3 tests)

6. **status shows pid when running**
   ```bash
   run_test "status shows pid" "${WEBCTL_BINARY}" status
   assert_success "${TEST_EXIT_CODE}" "status returns success"
   assert_contains "${TEST_STDOUT}" "pid:" "Output contains pid"
   ```

7. **status shows sessions/URL when running**
   ```bash
   # After navigating to a page
   run_test "status shows sessions" "${WEBCTL_BINARY}" status
   assert_success "${TEST_EXIT_CODE}" "status returns success"
   assert_contains "${TEST_STDOUT}" "sessions:" "Output contains sessions"
   ```

8. **status shows URL in session list**
   ```bash
   run_test "status shows URL" "${WEBCTL_BINARY}" status
   assert_contains "${TEST_STDOUT}" "http" "Output contains URL"
   ```

### Force Stop Action Reporting (2 tests)

9. **force stop reports cleanup actions**
   ```bash
   # Start daemon, then force stop
   run_test "force stop reports actions" "${WEBCTL_BINARY}" stop --force
   assert_success "${TEST_EXIT_CODE}" "force stop returns success"
   # Should contain at least one action (killed daemon, killed browser, removed socket, removed PID)
   assert_contains "${TEST_STDOUT}" "killed\|removed" "Output reports cleanup action"
   ```

10. **force stop --json reports actions array**
    ```bash
    run_test "force stop --json actions" "${WEBCTL_BINARY}" stop --force --json
    assert_success "${TEST_EXIT_CODE}" "force stop --json returns success"
    assert_contains "${TEST_STDOUT}" "actions" "JSON contains actions array"
    ```

### Error Hints (1 test)

11. **start already running includes hint**
    ```bash
    run_test "start already running hint" "${WEBCTL_BINARY}" start --headless
    assert_failure "${TEST_EXIT_CODE}" "start fails when already running"
    assert_contains "${TEST_STDOUT}${TEST_STDERR}" "webctl stop" "Error includes stop hint"
    ```

### No-Color Mode (2 tests)

12. **status --no-color when not running**
    ```bash
    run_test "status --no-color not running" "${WEBCTL_BINARY}" status --no-color
    assert_success "${TEST_EXIT_CODE}" "status --no-color returns success"
    assert_contains "${TEST_STDOUT}" "Not running" "Output shows not running"
    assert_not_contains "${TEST_STDOUT}" $'\e[' "Output has no ANSI codes"
    ```

13. **status --no-color when running**
    ```bash
    run_test "status --no-color running" "${WEBCTL_BINARY}" status --no-color
    assert_success "${TEST_EXIT_CODE}" "status --no-color returns success"
    assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"
    assert_not_contains "${TEST_STDOUT}" $'\e[' "Output has no ANSI codes"
    ```

## Summary

| Category | Missing Tests |
|----------|---------------|
| JSON output mode | 5 |
| Status content verification | 3 |
| Force stop action reporting | 2 |
| Error hints | 1 |
| No-color mode | 2 |
| **Total** | **13** |

## Implementation Notes

1. JSON tests require `assert_json_field` function from assertions.sh
2. No-color tests need to verify absence of ANSI escape sequences (`\e[` or `\033[`)
3. Force stop action tests may need to create stale files to verify all cleanup actions
4. Status content tests should navigate to a page first to ensure sessions are populated
5. Test sections should follow existing patterns from test-start-stop.sh

## Test Section Order (Recommended)

1. Status Command (Not Running) - existing + add --json, --no-color
2. Start Command - existing + add --json
3. Status Command (Running) - existing + add content verification, --json, --no-color
4. Start Command (Already Running) - existing + add hint verification
5. Stop Command - existing + add --json
6. Stop Command (Not Running) - existing
7. Force Stop Command - existing + add action reporting, --json
8. Custom Port Configuration - existing
