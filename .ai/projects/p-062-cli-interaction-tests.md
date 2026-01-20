# p-062: CLI Interaction Tests

- Status: Pending
- Started:
- Completed:
- Design Record: .ai/design/design-records/dr-032-test-framework-architecture.md

## Overview

Create automated tests for the webctl interaction commands. These tests verify browser interaction functionality including clicking elements, typing text, selecting options, scrolling, keyboard input, JavaScript evaluation, and waiting for page readiness.

## Goals

1. Test click command with various selectors and target elements
2. Test type command with text input, textareas, and special characters
3. Test select command with dropdown/select elements
4. Test scroll command with elements and positions
5. Test focus command with focusable elements
6. Test key command with keyboard events
7. Test eval command with JavaScript expressions
8. Test ready command with selectors and conditions
9. Test clear command for console and network buffers
10. Test find command for element discovery
11. Test target command for switching between pages/tabs

## Scope

In Scope:

- scripts/test/cli/test-interaction.sh
- Tests for: click, type, select, scroll, focus, key, eval, ready, clear, find, target commands
- Positive cases: successful interactions with valid selectors
- Error handling: invalid selectors, elements not found, interaction failures
- Verification: State changes after interactions (form values, scroll position, etc.)

Out of Scope:

- REPL tests (p-065)
- Workflow tests (p-066)
- JSON output mode testing (future)
- Performance benchmarks

## Success Criteria

- [ ] scripts/test/cli/test-interaction.sh created
- [ ] Tests pass with ./test-runner cli interaction
- [ ] click command tests: basic clicks, button interactions, link clicks, error cases
- [ ] type command tests: input fields, textareas, text entry, special characters, error cases
- [ ] select command tests: dropdown selection by value, text, index, error cases
- [ ] scroll command tests: scroll to element, scroll by position, scroll into view, error cases
- [ ] focus command tests: focus input elements, verify focus state, error cases
- [ ] key command tests: keyboard events (Enter, Tab, Escape, etc.), key sequences, error cases
- [ ] eval command tests: JavaScript execution, return values, DOM manipulation, error cases
- [ ] ready command tests: wait for selector, network idle, custom conditions, error cases
- [ ] clear command tests: clear console buffer, clear network buffer, verify emptied
- [ ] find command tests: element discovery, selector patterns, error cases
- [ ] target command tests: list targets, switch between pages/tabs, error cases

## Deliverables

- scripts/test/cli/test-interaction.sh

## Technical Approach

Implementation order:

1. Create test-interaction.sh following established patterns from test-observation.sh
2. Group tests by command type (interaction, evaluation, utility)
3. Use appropriate test pages for each command group

Test structure:

- Source shared modules (test-framework.sh, assertions.sh, setup.sh)
- Start daemon and test server for all tests
- Navigate to appropriate test pages before each interaction test
- Use run_test wrapper for consistent output capture
- Verify state changes after interactions where applicable
- Test error conditions for each command

Key test scenarios (comprehensive coverage):

**click command:**
- Basic button clicks
- Link navigation
- Click by various selectors (id, class, text)
- Click targets: buttons, links, divs with onclick
- Error cases: invalid selectors, element not found, element not clickable

**type command:**
- Type into input fields (text, email, password)
- Type into textareas
- Special characters and unicode
- Clear existing text before typing
- Error cases: invalid selectors, element not found, element not typeable

**select command:**
- Select by value
- Select by visible text
- Select by index
- Multiple selections (if supported)
- Error cases: invalid selectors, element not found, option not found

**scroll command:**
- Scroll to element by selector
- Scroll by position (x, y coordinates)
- Scroll into view
- Scroll on long pages
- Error cases: invalid selectors, element not found

**focus command:**
- Focus input elements
- Focus textareas
- Focus buttons
- Verify focus state with eval
- Error cases: invalid selectors, element not found, element not focusable

**key command:**
- Single key events (Enter, Tab, Escape)
- Key sequences
- Modifier keys (if supported)
- Error cases: invalid key names

**eval command:**
- Simple expressions (return values)
- DOM queries
- DOM manipulation
- Variable assignments
- Error cases: JavaScript errors, syntax errors

**ready command:**
- Wait for selector to appear
- Wait for network idle
- Wait with custom eval condition
- Timeout behavior
- Error cases: timeout, condition never met

**clear command:**
- Clear console buffer, verify empty with console command
- Clear network buffer, verify empty with network command
- Error cases: invalid buffer names

**find command:**
- Find elements by text content
- Find by selector patterns
- Multiple matches
- Error cases: no matches found

**target command:**
- List available targets
- Switch between targets (if multiple pages/tabs available)
- Error cases: invalid target id

## Current State

### Environment

- **Go Version**: 1.25.6 (project requires 1.25.5+)
- **Platform**: Linux/amd64 with bash support
- **Dependencies**: Standard (cobra, websocket, color, readline) - all available

### Test Framework (Ready)

All prerequisite projects completed (p-055 through p-061):

- **test-runner** at project root dispatches to scripts/test/cli/ directory
- **scripts/bash_modules/** contains shared testing infrastructure:
  - `test-framework.sh`: `run_test`, `test_section`, `test_summary`, test counters, color output
  - `assertions.sh`: `assert_success`, `assert_failure`, `assert_contains`, `assert_file_exists`, etc.
  - `setup.sh`: `setup_cleanup_trap`, `start_daemon`, `start_test_server`, `get_test_url`, cleanup handlers
- **Reference implementations**: test-observation.sh (127 tests), test-navigation.sh, test-start-stop.sh
- **Test runner integration**: `./test-runner cli interaction` will execute the test suite

### Test Pattern (from test-observation.sh)

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

### Interaction Commands (Mature Implementation)

All commands have extensive interactive test coverage (scripts/interactive/test-*.sh):

**click command** (8.5 KB interactive tests):
- Click buttons, links, elements with event handlers
- Various selector types (id, class, xpath, text)
- Tested: successful clicks, navigation, error cases

**type command** (11 KB interactive tests):
- Type into input fields, textareas
- Special characters, long text
- Tested: text entry, field clearing, error cases

**select command** (9.2 KB interactive tests):
- Select options from dropdowns
- Selection by value, text, index
- Tested: single selection, verification, error cases

**scroll command** (10 KB interactive tests):
- Scroll to elements
- Scroll by position
- Tested: scroll into view, position verification, error cases

**focus command** (10 KB interactive tests):
- Focus input elements, buttons, links
- Tested: focus state, focus verification, error cases

**key command** (12 KB interactive tests):
- Keyboard events (Enter, Tab, Escape, Arrow keys)
- Tested: single keys, sequences, event handling, error cases

**eval command** (9.6 KB interactive tests):
- Execute JavaScript expressions
- DOM queries and manipulation
- Tested: return values, DOM changes, error handling

**ready command** (8.5 KB interactive tests):
- Wait for selectors
- Network idle waiting
- Custom eval conditions
- Tested: successful waits, timeouts, error cases

**clear command** (9.3 KB interactive tests):
- Clear console buffer
- Clear network buffer
- Tested: buffer clearing, verification

**find command** (9.3 KB interactive tests):
- Find elements by selector patterns
- Text search within elements
- Tested: element discovery, multiple matches, error cases

**target command** (8.9 KB interactive tests):
- List available targets (pages/tabs)
- Switch between targets
- Tested: target listing, switching, error cases

### Test Pages (Available)

Test pages in testdata/pages/ are ready:

- **forms.html** - comprehensive form elements:
  - Input types: text, email, password, number, date, checkbox, radio
  - Textareas
  - Select/dropdown elements with multiple options
  - Buttons for submission and testing clicks
  - Form validation examples

- **click-targets.html** - various clickable elements:
  - Buttons with different styles
  - Links (internal and external)
  - Divs with onclick handlers
  - Nested clickable elements
  - Hidden/disabled elements for error testing

- **scroll-long.html** - long page for scroll testing:
  - Multiple sections with IDs
  - Elements at different scroll positions
  - Long content requiring scrolling
  - Anchors for scroll-to-element testing

- **navigation.html** - basic HTML structure for general testing
- **console-types.html** - for clear command testing (console buffer)
- **network-requests.html** - for clear command testing (network buffer, requires backend)

### Implementation Readiness

- ✅ All dependencies completed
- ✅ Test framework and modules ready
- ✅ Test pages available
- ✅ Reference test patterns established
- ✅ Commands are mature with known behaviors
- ✅ Interactive test scripts document expected outputs

## Dependencies

- p-055: Test Framework Bash Modules (completed)
- p-056: Test Library (completed)
- p-057: Test Runner (completed)
- p-058: Test Pages (completed)
- p-061: CLI Observation Tests (completed - provides reference pattern)

## Decision Points

**All decisions resolved:**

### 1. Target Command Test Scope → **Option B: Comprehensive multi-tab testing**
- Will test: Opening multiple tabs/windows, listing all targets, switching between targets, verifying context isolation
- Rationale: Full feature coverage for multi-context scenarios

### 2. JSON Output Mode Coverage → **Option B: Include JSON output tests**
- Will test: `--json` flag for each interaction command with JSON structure verification
- Rationale: Ensures consistency across all commands

### 3. Error Message Validation Depth → **Option B: Detailed error messages**
- Will test: Specific error message content and error codes/patterns
- Rationale: Documents expected error behavior and catches message regressions

## Coverage Matrix

Before implementation, read `.ai/tasks/cli-test-script.md` workflow. This section will track all test cases to ensure complete coverage.

### Commands to Test

| Command | Source File | Command-Specific Flags | JSON Mode | Notes |
|---------|-------------|------------------------|-----------|-------|
| click | internal/cli/click.go | None | Yes | Simple selector-based command |
| type | internal/cli/type.go | --key (string), --clear (bool) | Yes | Accepts 1-2 args (optional selector) |
| select | internal/cli/selectcmd.go | None | Yes | Requires selector and value args |
| scroll | internal/cli/scroll.go | --to (string), --by (string) | Yes | Three modes: element, absolute, relative |
| focus | internal/cli/focus.go | None | Yes | Simple selector-based command |
| key | internal/cli/key.go | --ctrl, --alt, --shift, --meta (all bool) | Yes | Modifier key combinations |
| eval | internal/cli/eval.go | --timeout/-t (duration, default 60s) | Yes | Supports async/Promise expressions |
| ready | internal/cli/ready.go | --timeout (duration, 60s), --network-idle (bool), --eval (string) | Yes | Four waiting modes |
| clear | internal/cli/clear.go | None | Yes | Optional arg: console or network |
| find | internal/cli/find.go | --regex/-E (bool), --case-sensitive/-c (bool), --limit/-l (int) | Yes | Text search with context display |
| target | internal/cli/target.go | None | Yes | Optional query arg for switching |

### Global Flags (All Commands)

| Flag | Description | Priority |
|------|-------------|----------|
| --json | Output in JSON format | High |
| --no-color | Disable color output | High |
| --debug | Enable debug output | Low |

### Test Coverage Requirements

Per `.ai/tasks/cli-test-script.md`, each command needs:

1. ✅ Basic functionality - Command works with no optional flags
2. ✅ Each flag individually - Every flag tested in isolation
3. ✅ Flag combinations - Commonly combined flags tested together
4. ✅ JSON output mode - `--json` flag produces valid JSON
5. ✅ JSON + flags - JSON output with other flags
6. ✅ No-color mode - `--no-color` produces plain text (no ANSI codes)
7. ✅ Error cases - All known error conditions
8. ✅ Error messages - Verify error text is helpful
9. ✅ Edge cases - Boundary conditions, empty inputs, etc.

**Estimated Test Count**: 150-250 tests across 11 commands (15-25 tests per command on average)

### Implementation Status

- [ ] Phase 1: Command analysis - Read all source files, document flags
- [ ] Phase 2: Complete coverage matrix with all test cases
- [ ] Phase 3: Implementation - Create test-interaction.sh
- [ ] Phase 4: Verification - All tests pass, coverage complete

## Notes

- Interaction tests require a running daemon and test server
- Some tests need specific page content (forms.html for type/select, click-targets.html for click, scroll-long.html for scroll)
- clear command tests need to populate console/network buffers before clearing
- ready command tests may need to trigger page events or navigation
- Verify state changes after interactions using eval or observation commands where applicable
- Reference interactive test scripts (scripts/interactive/test-*.sh) for expected command outputs and edge cases
- **IMPORTANT**: Follow `.ai/tasks/cli-test-script.md` workflow before starting implementation
- Test suite will be structured by command, with sections for basic, flags, JSON, no-color, and error cases
