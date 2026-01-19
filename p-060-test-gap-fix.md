# p-060 Test Gap Fix

This document details missing tests in `scripts/test/cli/test-navigation.sh` that need to be added for comprehensive coverage comparable to p-061 (CLI Observation Tests).

## Current State

- **Test file:** `scripts/test/cli/test-navigation.sh` (149 lines)
- **Current tests:** ~16
- **Commands covered:** navigate, reload, back, forward

## Commands Under Test

### navigate command (`internal/cli/navigate.go`)

**Arguments:**
- URL (required) - The URL to navigate to

**Flags:**
- `--wait` (bool, default false) - Wait for page load completion
- `--timeout` (int, default 60) - Timeout in seconds (used with --wait)

**Output modes:**
- Text: "OK" on success
- JSON: `{"ok":true}`

**URL normalization:**
- Adds `https://` by default
- Adds `http://` for localhost/127.0.0.1

**Error cases:**
- Invalid URL: navigation failure message (stderr, exit 1)

### reload command (`internal/cli/reload.go`)

**Flags:**
- `--wait` (bool, default false) - Wait for page load completion
- `--timeout` (int, default 60) - Timeout in seconds (used with --wait)

**Output modes:**
- Text: "OK" on success
- JSON: `{"ok":true}`

**Error cases:**
- Reload failure: error message (stderr, exit 1)

### back command (`internal/cli/back.go`)

**Flags:**
- `--wait` (bool, default false) - Wait for page load completion
- `--timeout` (int, default 60) - Timeout in seconds (used with --wait)

**Output modes:**
- Text: "OK" on success
- JSON: `{"ok":true}`

**Error cases:**
- No history: "No previous page" (stderr via outputNotice, exit 1)

### forward command (`internal/cli/forward.go`)

**Flags:**
- `--wait` (bool, default false) - Wait for page load completion
- `--timeout` (int, default 60) - Timeout in seconds (used with --wait)

**Output modes:**
- Text: "OK" on success
- JSON: `{"ok":true}`

**Error cases:**
- No history: "No next page" (stderr via outputNotice, exit 1)

### Global Flags

- `--json` - Output in JSON format
- `--no-color` - Disable color output
- `--debug` - Enable verbose debug output

## Currently Tested

| # | Test | Status |
|---|------|--------|
| 1 | Back with no history (fresh start) | ✓ |
| 2 | Forward with no history (fresh start) | ✓ |
| 3 | Navigate to test server URL | ✓ |
| 4 | Navigate to forms page | ✓ |
| 5 | Navigate to file URL | ✓ |
| 6 | Navigate to invalid URL | ✓ |
| 7 | Setup navigate for reload test | ✓ |
| 8 | Reload current page | ✓ |
| 9-11 | Setup navigation for history (3 pages) | ✓ |
| 12 | Back with history | ✓ |
| 13 | Back again | ✓ |
| 14 | Forward with history | ✓ |
| 15 | Forward again | ✓ |
| 16 | Forward at end of history | ✓ |

## Missing Tests

### Wait Flag Tests (8 tests)

1. **navigate --wait basic**
   ```bash
   run_test "navigate --wait" "${WEBCTL_BINARY}" navigate --wait "$(get_test_url '/pages/navigation.html')"
   assert_success "${TEST_EXIT_CODE}" "navigate --wait returns success"
   assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"
   ```

2. **navigate --wait --timeout**
   ```bash
   run_test "navigate --wait --timeout" "${WEBCTL_BINARY}" navigate --wait --timeout 10 "$(get_test_url '/pages/navigation.html')"
   assert_success "${TEST_EXIT_CODE}" "navigate --wait --timeout returns success"
   ```

3. **reload --wait basic**
   ```bash
   run_test "reload --wait" "${WEBCTL_BINARY}" reload --wait
   assert_success "${TEST_EXIT_CODE}" "reload --wait returns success"
   assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"
   ```

4. **reload --wait --timeout**
   ```bash
   run_test "reload --wait --timeout" "${WEBCTL_BINARY}" reload --wait --timeout 10
   assert_success "${TEST_EXIT_CODE}" "reload --wait --timeout returns success"
   ```

5. **back --wait basic**
   ```bash
   # After navigating to multiple pages
   run_test "back --wait" "${WEBCTL_BINARY}" back --wait
   assert_success "${TEST_EXIT_CODE}" "back --wait returns success"
   assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"
   ```

6. **back --wait --timeout**
   ```bash
   run_test "back --wait --timeout" "${WEBCTL_BINARY}" back --wait --timeout 10
   assert_success "${TEST_EXIT_CODE}" "back --wait --timeout returns success"
   ```

7. **forward --wait basic**
   ```bash
   # After going back
   run_test "forward --wait" "${WEBCTL_BINARY}" forward --wait
   assert_success "${TEST_EXIT_CODE}" "forward --wait returns success"
   assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"
   ```

8. **forward --wait --timeout**
   ```bash
   run_test "forward --wait --timeout" "${WEBCTL_BINARY}" forward --wait --timeout 10
   assert_success "${TEST_EXIT_CODE}" "forward --wait --timeout returns success"
   ```

### JSON Output Mode (4 tests)

9. **navigate --json output**
   ```bash
   run_test "navigate --json" "${WEBCTL_BINARY}" navigate --json "$(get_test_url '/pages/navigation.html')"
   assert_success "${TEST_EXIT_CODE}" "navigate --json returns success"
   assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
   ```

10. **reload --json output**
    ```bash
    run_test "reload --json" "${WEBCTL_BINARY}" reload --json
    assert_success "${TEST_EXIT_CODE}" "reload --json returns success"
    assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
    ```

11. **back --json output**
    ```bash
    run_test "back --json" "${WEBCTL_BINARY}" back --json
    assert_success "${TEST_EXIT_CODE}" "back --json returns success"
    assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
    ```

12. **forward --json output**
    ```bash
    run_test "forward --json" "${WEBCTL_BINARY}" forward --json
    assert_success "${TEST_EXIT_CODE}" "forward --json returns success"
    assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
    ```

### Timeout Error Cases (2 tests)

13. **navigate --wait timeout on slow page**
    ```bash
    # Use a slow-loading page or a page that triggers long JS
    run_test "navigate --wait timeout" "${WEBCTL_BINARY}" navigate --wait --timeout 1 "$(get_test_url '/pages/slow-load.html')"
    # May succeed or fail depending on page load time - test the flag is accepted
    ```

14. **reload --wait --timeout with short timeout**
    ```bash
    run_test "reload --wait short timeout" "${WEBCTL_BINARY}" reload --wait --timeout 1
    # Verify timeout flag is accepted
    ```

### URL Normalization (2 tests)

15. **navigate adds https:// to bare domain**
    ```bash
    run_test "navigate normalizes URL" "${WEBCTL_BINARY}" navigate "example.com"
    # Note: This will attempt to navigate to https://example.com
    # Test may need network access or could verify via status command
    ```

16. **navigate adds http:// to localhost**
    ```bash
    run_test "navigate localhost uses http" "${WEBCTL_BINARY}" navigate "localhost:8888/pages/navigation.html"
    assert_success "${TEST_EXIT_CODE}" "navigate to localhost works"
    ```

### Combined Flags (2 tests)

17. **navigate --wait --json combined**
    ```bash
    run_test "navigate --wait --json" "${WEBCTL_BINARY}" navigate --wait --json "$(get_test_url '/pages/navigation.html')"
    assert_success "${TEST_EXIT_CODE}" "navigate --wait --json returns success"
    assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
    ```

18. **reload --wait --json combined**
    ```bash
    run_test "reload --wait --json" "${WEBCTL_BINARY}" reload --wait --json
    assert_success "${TEST_EXIT_CODE}" "reload --wait --json returns success"
    assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"
    ```

## Summary

| Category | Missing Tests |
|----------|---------------|
| Wait flag (`--wait`) | 8 |
| JSON output mode | 4 |
| Timeout error cases | 2 |
| URL normalization | 2 |
| Combined flags | 2 |
| **Total** | **18** |

## Implementation Notes

1. All four navigation commands share the same `--wait` and `--timeout` flags
2. JSON tests require `assert_json_field` function from assertions.sh
3. Timeout tests may be timing-sensitive - use `testdata/pages/slow-load.html` if available
4. URL normalization tests may require external network access (example.com) or can be verified indirectly
5. Wait flag tests should verify the command waits for page load before returning
6. Combined flag tests ensure flags don't conflict with each other

## Test Section Order (Recommended)

1. History Error Cases (Fresh Start) - existing
2. Navigate Command - existing + add --wait, --wait --timeout, --json, URL normalization
3. Reload Command - existing + add --wait, --wait --timeout, --json
4. Back Command (With History) - existing + add --wait, --json
5. Forward Command (With History) - existing + add --wait, --json
6. Combined Flags - new section for --wait --json combinations
7. Timeout Edge Cases - new section for timeout behavior

## Test Page Requirements

If not already present, may need:
- `testdata/pages/slow-load.html` - A page with intentional delay for timeout testing

Existing pages that can be used:
- `testdata/pages/navigation.html` - Basic page for navigation tests
- `testdata/pages/forms.html` - Alternative page for history building
- `testdata/pages/cookies.html` - Another alternative for history tests
