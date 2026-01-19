# CLI Test Script Development

This task document guides agents through creating comprehensive CLI test scripts that achieve complete coverage of command functionality.

## Objective

Create automated bash test scripts that thoroughly test webctl CLI commands, ensuring all flags, output modes, error cases, and edge cases are covered.

## Why This Matters

Previous test script projects (p-059, p-060) were completed with significant gaps:
- Missing `--json` output mode tests
- Missing `--wait` and `--timeout` flag tests
- Missing `--no-color` flag tests
- Incomplete error case coverage

This resulted in rework to fix gaps after project completion. Following this workflow prevents that.

---

## Workflow

### Phase 1: Command Analysis

Before writing any test code, analyze each command to enumerate ALL testable behaviors.

#### Step 1: Read Command Source

For each command in scope, read the source file:

```bash
# Example for navigate command
internal/cli/navigate.go
```

Extract from the `init()` function:
- All command-specific flags (BoolVar, StringVar, IntVar, etc.)
- Flag defaults
- Flag descriptions

#### Step 2: Identify Global Flags

These apply to ALL commands and must be tested:

| Flag | Description |
|------|-------------|
| `--json` | Output in JSON format |
| `--no-color` | Disable color output |
| `--debug` | Enable debug output (lower priority) |

#### Step 3: Identify Output Modes

For each command, document:
- Text output format (default)
- JSON output format (`--json`)
- Success output
- Error output (stderr)
- Hint messages (if any)

#### Step 4: Identify Error Cases

- What errors can occur?
- What are the error messages?
- What exit codes are returned?
- Are there hint messages with errors?

---

### Phase 2: Coverage Matrix

Create a coverage matrix in the project file BEFORE implementation.

#### Template

```markdown
## Coverage Matrix

### Command: [command name]

**Source:** `internal/cli/[command].go`

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --flag1 | bool | false | Description |
| --flag2 | int | 60 | Description |

**Test Cases:**
| # | Category | Test Case | Flags/Args | Expected |
|---|----------|-----------|------------|----------|
| 1 | Basic | command succeeds | (none) | exit 0, "OK" |
| 2 | Flag | with --flag1 | --flag1 | exit 0 |
| 3 | Flag | with --flag2 | --flag2 30 | exit 0 |
| 4 | Combined | flag1 + flag2 | --flag1 --flag2 30 | exit 0 |
| 5 | JSON | json output | --json | exit 0, {"ok":true} |
| 6 | JSON+Flag | json with flag1 | --json --flag1 | exit 0, {"ok":true} |
| 7 | No-Color | no-color output | --no-color | exit 0, no ANSI |
| 8 | Error | error case 1 | invalid input | exit 1, error msg |
| 9 | Error | error case 2 | missing arg | exit 1, error msg |
```

#### Required Test Categories

Every command MUST have tests for:

1. **Basic functionality** - Command works with no optional flags
2. **Each flag individually** - Every flag tested in isolation
3. **Flag combinations** - Commonly combined flags tested together
4. **JSON output mode** - `--json` flag produces valid JSON
5. **JSON + flags** - JSON output with other flags
6. **No-color mode** - `--no-color` produces plain text (no ANSI codes)
7. **Error cases** - All known error conditions
8. **Error messages** - Verify error text is helpful
9. **Edge cases** - Boundary conditions, empty inputs, etc.

---

### Phase 3: Implementation

#### Test Script Structure

```bash
#!/usr/bin/env bash

# Test: CLI [Category] Commands
# -----------------------------
# Tests for webctl [commands].
# [Brief description of what's tested]

# Determine script location and project root
SCRIPT_DIR="$(cd "${BASH_SOURCE[0]%/*}" || exit 1; pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." || exit 1; pwd)"

# Import test modules
source "${PROJECT_ROOT}/scripts/bash_modules/test-framework.sh"
source "${PROJECT_ROOT}/scripts/bash_modules/assertions.sh"
source "${PROJECT_ROOT}/scripts/bash_modules/setup.sh"

# Setup
setup_cleanup_trap
require_webctl

# [Setup code - start daemon, server, etc.]

# =============================================================================
# [Command] - Basic Functionality
# =============================================================================

test_section "[Command] Command - Basic"

run_test "[command] basic" "${WEBCTL_BINARY}" [command] [args]
assert_success "${TEST_EXIT_CODE}" "[command] returns success"
assert_contains "${TEST_STDOUT}" "OK" "Output shows OK"

# =============================================================================
# [Command] - Flag Tests
# =============================================================================

test_section "[Command] Command - Flags"

run_test "[command] --flag1" "${WEBCTL_BINARY}" [command] --flag1 [args]
assert_success "${TEST_EXIT_CODE}" "--flag1 returns success"

# =============================================================================
# [Command] - JSON Output Mode
# =============================================================================

test_section "[Command] Command - JSON Output"

run_test "[command] --json" "${WEBCTL_BINARY}" [command] --json [args]
assert_success "${TEST_EXIT_CODE}" "--json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok field is true"

# =============================================================================
# [Command] - No-Color Mode
# =============================================================================

test_section "[Command] Command - No-Color"

run_test "[command] --no-color" "${WEBCTL_BINARY}" [command] --no-color [args]
assert_success "${TEST_EXIT_CODE}" "--no-color returns success"
assert_not_contains "${TEST_STDOUT}" $'\e[' "Output has no ANSI codes"

# =============================================================================
# [Command] - Error Cases
# =============================================================================

test_section "[Command] Command - Error Cases"

run_test "[command] error case" "${WEBCTL_BINARY}" [command] [bad-args]
assert_failure "${TEST_EXIT_CODE}" "Error case returns failure"
assert_contains "${TEST_STDERR}" "expected error text" "Error message is correct"

# =============================================================================
# Summary
# =============================================================================

test_summary
```

#### Assertion Functions Reference

```bash
# Exit code assertions
assert_success "${TEST_EXIT_CODE}" "message"
assert_failure "${TEST_EXIT_CODE}" "message"
assert_exit_code 2 "${TEST_EXIT_CODE}" "message"  # Specific code

# Content assertions
assert_contains "${TEST_STDOUT}" "needle" "message"
assert_not_contains "${TEST_STDOUT}" "needle" "message"
assert_equals "expected" "${TEST_STDOUT}" "message"

# JSON assertions
assert_json_field "${TEST_STDOUT}" ".ok" "true" "message"
assert_json_field "${TEST_STDOUT}" ".data.field" "value" "message"

# File assertions
assert_file_exists "/path/to/file" "message"
assert_file_contains "/path/to/file" "needle" "message"
```

---

### Phase 4: Verification

Before marking the project complete:

#### Checklist

- [ ] Every flag from command source is tested individually
- [ ] Flag combinations are tested
- [ ] `--json` output tested for all commands
- [ ] JSON output structure verified with `assert_json_field`
- [ ] `--no-color` tested (verify no ANSI escape codes)
- [ ] All error cases tested
- [ ] Error messages verified
- [ ] Edge cases covered
- [ ] Tests pass: `./test-runner cli [suite-name]`
- [ ] Coverage matrix in project file is complete (all rows have status)

#### Test Count Guideline

As a rough guide, expect:
- **Simple command** (few flags): 10-15 tests
- **Medium command** (several flags): 20-40 tests
- **Complex command** (many flags, subcommands): 50+ tests

If your test count is significantly lower, you likely have gaps.

---

## Common Patterns

### Testing JSON Output

```bash
test_section "Command - JSON Output"

run_test "command --json" "${WEBCTL_BINARY}" command --json
assert_success "${TEST_EXIT_CODE}" "--json returns success"
assert_json_field "${TEST_STDOUT}" ".ok" "true" "JSON ok is true"

# For commands with data fields:
assert_json_field "${TEST_STDOUT}" ".data.field" "expected" "JSON data.field correct"
assert_contains "${TEST_STDOUT}" "\"field\":" "JSON contains field"
```

### Testing No-Color Mode

```bash
test_section "Command - No-Color"

run_test "command --no-color" "${WEBCTL_BINARY}" command --no-color
assert_success "${TEST_EXIT_CODE}" "--no-color returns success"
# ANSI escape codes start with \e[ or \033[
assert_not_contains "${TEST_STDOUT}" $'\e[' "No ANSI escape codes in output"
```

### Testing Error Cases

```bash
test_section "Command - Error Cases"

run_test "command with invalid input" "${WEBCTL_BINARY}" command "invalid"
assert_failure "${TEST_EXIT_CODE}" "Invalid input returns failure"
assert_contains "${TEST_STDERR}" "Error:" "Error message on stderr"

# Verify error hints if applicable
assert_contains "${TEST_STDERR}" "hint text" "Error includes helpful hint"
```

### Testing Timeout Flags

```bash
test_section "Command - Timeout Behavior"

# Test flag is accepted
run_test "command --timeout 30" "${WEBCTL_BINARY}" command --timeout 30
assert_success "${TEST_EXIT_CODE}" "--timeout flag accepted"

# Test short timeout (may fail or succeed depending on operation)
run_test "command --timeout 1" "${WEBCTL_BINARY}" command --timeout 1
# Document expected behavior
```

---

## References

- Test framework: `scripts/bash_modules/test-framework.sh`
- Assertions: `scripts/bash_modules/assertions.sh`
- Setup helpers: `scripts/bash_modules/setup.sh`
- Example (comprehensive): `scripts/test/cli/test-observation.sh`
- Test runner: `./test-runner cli [suite]`
