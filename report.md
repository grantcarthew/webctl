# Test Coverage Report: p-061 CLI Observation Tests

**Date:** 2026-01-17
**Project:** p-061-cli-observation-tests
**Status:** 80/80 tests passing (happy path coverage complete)
**Issues:** Missing error cases, range limiting tests, and CSS inline/matched subcommands

---

## Executive Summary

The CLI observation test suite (scripts/test/cli/test-observation.sh) successfully implements comprehensive happy path testing for all observation commands. However, investigation reveals significant gaps between planned scope and implemented tests, particularly around error handling and advanced features.

**Critical Finding:** The webctl binary is outdated. Source code contains `css inline` and `css matched` subcommands that are not present in the current binary (built Jan 11, 2026).

---

## 1. CSS Inline/Matched Commands - Source vs Binary Mismatch

### Current Situation

**Binary Status (./webctl built Jan 11):**
```bash
$ ./webctl css --help | grep "Available Commands:" -A 5
Available Commands:
  computed    Get computed styles to stdout
  get         Get single CSS property to stdout
  save        Save CSS to file
```

**Source Code Status (internal/cli/css.go):**
```go
// Line 178: cssInlineCmd fully implemented
var cssInlineCmd = &cobra.Command{
    Use:   "inline <selector>",
    Short: "Get inline style attributes",
    ...
}

// Line 211: cssMatchedCmd fully implemented
var cssMatchedCmd = &cobra.Command{
    Use:   "matched <selector>",
    Short: "Get matched CSS rules from stylesheets",
    ...
}

// Line 256: Both commands registered
cssCmd.AddCommand(cssSaveCmd, cssComputedCmd, cssGetCmd, cssInlineCmd, cssMatchedCmd)
```

### What These Commands Do

**css inline <selector>:**
- Extracts inline style attributes (the `style="..."` attribute content)
- Returns style attribute content for matching elements
- Multiple elements separated by `--` markers
- Useful for debugging inline styles vs stylesheet rules

**css matched <selector>:**
- Gets matched CSS rules from stylesheets for an element
- Shows which stylesheet rules actually apply to the element
- Includes selector and property/value pairs
- Multiple elements separated by `--` markers
- Useful for understanding CSS cascade and specificity

### Evidence in Code

**Interactive test file exists:**
```bash
$ grep -E "css (inline|matched)" scripts/interactive/test-css.sh
cmd "webctl css inline \"[style]\""
cmd "webctl css inline \"body\""
cmd "webctl css inline \"div\""
cmd "webctl css inline \"[style]\" --json"
cmd "webctl css matched \"body\""
cmd "webctl css matched \"h1\""
cmd "webctl css matched \"#main\""
cmd "webctl css matched \"body\" --json"
```

**Git history shows implementation:**
```bash
$ git log --oneline | grep "inline and matched"
c07730e feat(css): add inline and matched commands; rework selector logic
```

### Action Required

1. **Rebuild the binary:**
   ```bash
   go build -o webctl ./cmd/webctl
   ```

2. **Add tests to test-observation.sh:**
   - CSS inline basic test
   - CSS inline with multiple elements
   - CSS inline with no inline styles (edge case)
   - CSS matched basic test
   - CSS matched with multiple elements
   - CSS matched with no matched rules (edge case)

3. **Update project documentation** to reflect these commands were tested

---

## 2. Tests Skipped or Simplified

### 2.1 Error Case Tests - MAJOR GAP

**What was planned (from p-061 project file):**

**HTML:**
- Invalid selectors
- No matches found

**CSS:**
- Invalid selectors
- No matches
- Property does not exist (for `css get`)

**Console:**
- No logs
- Invalid type

**Network:**
- Network errors
- CORS failures

**What was implemented:**
- **NONE** - Zero error case tests

**Impact:**
- No validation that commands fail gracefully
- No testing of error messages
- No confirmation that exit codes are non-zero for errors
- Users may encounter untested error paths

**Test cases needed (~15-20 tests):**
```bash
# HTML error cases
run_test "html with invalid selector" "${WEBCTL_BINARY}" html --select "::invalid::"
assert_failure "${TEST_EXIT_CODE}" "Invalid selector fails"
assert_contains "${TEST_STDERR}" "invalid selector" "Error message shown"

run_test "html find with no matches" "${WEBCTL_BINARY}" html --find "NONEXISTENT_TEXT_12345"
assert_failure "${TEST_EXIT_CODE}" "No matches returns error"

# CSS error cases
run_test "css get with invalid property" "${WEBCTL_BINARY}" css get "body" "nonexistent-property"
assert_failure "${TEST_EXIT_CODE}" "Invalid property fails"

run_test "css computed with no elements" "${WEBCTL_BINARY}" css computed ".nonexistent-class-12345"
assert_failure "${TEST_EXIT_CODE}" "No matching elements fails"

# Console error cases
run_test "console with invalid type" "${WEBCTL_BINARY}" console --type invalid
assert_failure "${TEST_EXIT_CODE}" "Invalid type fails"

# Network error cases - check for empty buffer handling
run_test "clear network then query" "${WEBCTL_BINARY}" clear network
run_test "network with no requests" "${WEBCTL_BINARY}" network
# Should succeed with empty output, not error

# Cookies error cases
run_test "cookies delete nonexistent" "${WEBCTL_BINARY}" cookies delete "nonexistent-cookie-12345"
# May succeed (idempotent operation) - verify behavior
```

### 2.2 Range Limiting Tests (--head, --tail)

**What was planned:**
- Console command: `--head N`, `--tail N`, `--range N-M`
- Network command: `--head N`, `--tail N`, `--range N-M`

**What was implemented:**
- **NONE** - These flags were not tested at all

**Why it matters:**
These are important features for handling large datasets:
- Console logs can be hundreds of entries
- Network requests can be extensive
- Users need to limit output for performance and readability

**Test cases needed (~8 tests):**
```bash
# Console range tests
run_test "console --head 5" "${WEBCTL_BINARY}" console --head 5
assert_success "${TEST_EXIT_CODE}" "head returns success"
# Count output lines, verify ≤ 5

run_test "console --tail 3" "${WEBCTL_BINARY}" console --tail 3
assert_success "${TEST_EXIT_CODE}" "tail returns success"
# Count output lines, verify ≤ 3

run_test "console --range 2-5" "${WEBCTL_BINARY}" console --range 2-5
assert_success "${TEST_EXIT_CODE}" "range returns success"
# Verify entries 2-5 returned

# Network range tests
run_test "network --head 10" "${WEBCTL_BINARY}" network --head 10
assert_success "${TEST_EXIT_CODE}" "head returns success"

run_test "network --tail 5" "${WEBCTL_BINARY}" network --tail 5
assert_success "${TEST_EXIT_CODE}" "tail returns success"

# Error cases: mutually exclusive flags
run_test "console --head 5 --tail 3" "${WEBCTL_BINARY}" console --head 5 --tail 3
assert_failure "${TEST_EXIT_CODE}" "head and tail together fails"
```

### 2.3 Context Flags (-A, -B, -C) Tests

**What was planned (from DR-026 and interactive tests):**
- HTML: `--before, -B N`, `--after, -A N`, `--context, -C N`
- CSS: `--before, -B N`, `--after, -A N`, `--context, -C N`
- Console: Context flags for --find results

**What was implemented:**
- **NONE** - Context flags not tested

**Why it matters:**
Context flags are crucial for debugging - they show surrounding lines when searching:
```bash
webctl html --find "error" -C 3
# Shows error line plus 3 lines before and after
```

**Test cases needed (~9 tests):**
```bash
# HTML context tests
run_test "html --find with -B 2" "${WEBCTL_BINARY}" html --find "Navigation" -B 2
assert_success "${TEST_EXIT_CODE}" "Before context works"
# Verify 2 lines before match are included

run_test "html --find with -A 3" "${WEBCTL_BINARY}" html --find "Navigation" -A 3
assert_success "${TEST_EXIT_CODE}" "After context works"

run_test "html --find with -C 2" "${WEBCTL_BINARY}" html --find "Navigation" -C 2
assert_success "${TEST_EXIT_CODE}" "Context works"
# Verify 2 lines before AND after

# CSS context tests
run_test "css --find with context" "${WEBCTL_BINARY}" css --find "background" -C 1
assert_success "${TEST_EXIT_CODE}" "CSS context works"

# Console context tests (if supported)
run_test "console --find with context" "${WEBCTL_BINARY}" console --find "TEST" -C 2
assert_success "${TEST_EXIT_CODE}" "Console context works"
```

### 2.4 Backend Server Integration - Simplified

**What was planned:**
```
**network command:**
- Backend-triggered requests via proxy
```

**What was implemented:**
- Backend management functions added to setup.sh (start_backend, stop_backend)
- NOT used in actual tests
- Network tests rely only on page resource loading

**Why it was simplified:**
- Page resource requests (HTML, CSS, images) sufficient for basic testing
- Backend adds complexity and potential flakiness
- Core network observation functionality validated without it

**What's missing:**
Without backend server testing, we don't validate:
- Proxy functionality (`webctl serve --proxy`)
- Specific status code testing (404, 500, etc.)
- API endpoint requests vs page resources
- Delayed responses (`/delay` endpoint)
- JSON API response handling

**Test cases that could be added (~8 tests):**
```bash
# Backend-based network tests
start_backend 3000
stop_test_server
webctl serve testdata --proxy http://localhost:3000 &
sleep 2

# Test specific status codes
run_test "navigate to trigger backend call" "${WEBCTL_BINARY}" navigate "$(get_test_url '/pages/network-requests.html')"
run_test "trigger 404 request" "${WEBCTL_BINARY}" eval "fetch('/status/404')"
sleep 1
run_test "network --status 404" "${WEBCTL_BINARY}" network --status 404
assert_success "${TEST_EXIT_CODE}" "404 filter works"
assert_contains "${TEST_STDOUT}" "404" "404 status shown"

# Test API endpoints
run_test "trigger API hello" "${WEBCTL_BINARY}" eval "fetch('/api/hello').then(r => r.json())"
sleep 1
run_test "network finds API call" "${WEBCTL_BINARY}" network --find "api/hello"
assert_success "${TEST_EXIT_CODE}" "API call found"
```

### 2.5 One Test Simplified (Not Skipped)

**Network --find test:**
```bash
# Test: Network with text search (may have no matches if no API calls made)
run_test "network with --find" "${WEBCTL_BINARY}" network --find "network"
# Don't assert success - may be no matches, which is valid
```

**Issue:** Test runs but doesn't assert success/failure or validate output

**Should be:**
```bash
run_test "network with --find" "${WEBCTL_BINARY}" network --find "network-requests"
assert_success "${TEST_EXIT_CODE}" "Network find returns success"
assert_contains "${TEST_STDOUT}" "network-requests" "Found the page request"
```

---

## 3. Test Count Analysis

### Planned vs Implemented

**From project file "Key test scenarios":**

| Command | Planned Tests | Implemented | Missing |
|---------|---------------|-------------|---------|
| HTML | Basic, selector, find, save (3 modes), errors | 17 | ~3 error tests |
| CSS | Basic, selector, find, computed, get, **inline, matched**, save (3 modes), errors | 14 | **inline/matched (6)**, ~3 error tests |
| Console | Basic, type filter, find, **--head, --tail**, save (3 modes), errors | 12 | **range tests (3)**, ~2 error tests |
| Network | Basic, status, method, find, save (3 modes), **backend tests** | 10 | **range tests (2)**, **backend tests (8)** |
| Cookies | Basic, set, delete, domain, find, save (3 modes) | 17 | None (complete) |
| Screenshot | Basic, custom path, full-page | 11 | None (complete) |

### Gap Summary

**Total tests implemented:** 80
**Total tests missing:** ~35-40
**Estimated complete coverage:** 115-120 tests

**Categories of missing tests:**
1. **Error cases:** ~15 tests (all commands)
2. **CSS inline/matched:** ~6 tests (after rebuild)
3. **Range limiting:** ~8 tests (console, network)
4. **Context flags:** ~9 tests (html, css, console)
5. **Backend integration:** ~8 tests (network with proxy)

---

## 4. Why Tests Were Simplified

### Pragmatic Decisions

**Good reasons:**
1. **Binary outdated** - css inline/matched couldn't be tested
2. **Time constraints** - 80 tests is substantial for initial coverage
3. **Happy path priority** - Core functionality validated first
4. **Reduced flakiness** - Skipping backend eliminated potential test flakiness

**Questionable decisions:**
1. **No error testing** - This is a significant gap
2. **No range limiting** - Common feature left untested
3. **No context flags** - Useful feature ignored

### Impact Assessment

**Low Risk (acceptable gaps):**
- Backend integration tests - core network observation works without it
- Context flags - advanced feature, less commonly used

**Medium Risk (should add soon):**
- Range limiting tests - --head/--tail are important for large datasets
- CSS inline/matched tests - need binary rebuild first

**High Risk (should be added immediately):**
- Error case tests - users will encounter errors, these paths are untested
- No validation that error messages are helpful
- No confirmation that exit codes are correct

---

## 5. Recommendations

### Immediate Actions (High Priority)

1. **Rebuild webctl binary:**
   ```bash
   go build -o webctl ./cmd/webctl
   ```

2. **Add error case tests** (~15 tests):
   - Invalid selectors for html, css
   - No matches scenarios
   - Invalid types/properties
   - Verify error messages and exit codes

3. **Add CSS inline/matched tests** (~6 tests):
   - Basic inline styles
   - Basic matched rules
   - Multiple elements
   - No inline styles (edge case)
   - No matched rules (edge case)

### Short-term Actions (Medium Priority)

4. **Add range limiting tests** (~8 tests):
   - Console --head, --tail, --range
   - Network --head, --tail
   - Mutual exclusivity validation

5. **Add context flag tests** (~9 tests):
   - HTML/CSS/Console with -A, -B, -C flags
   - Verify context lines included

### Long-term Actions (Lower Priority)

6. **Add backend integration tests** (~8 tests):
   - Proxy functionality
   - Specific status codes (404, 500)
   - API endpoint requests
   - Delayed responses

7. **Improve network --find test**:
   - Add proper assertions
   - Ensure reproducible matches

### Updated Test Count Target

**Current:** 80 tests
**With high priority additions:** ~101 tests
**With medium priority additions:** ~118 tests
**With all additions:** ~126 tests

---

## 6. Project File Accuracy Issues

### Inaccuracies Found in p-061-cli-observation-tests.md

**Line 88:** Lists "inline, matched" as subcommands to test
- **Issue:** Binary didn't have these (source code does)
- **Resolution:** Rebuild binary and add tests

**Line 96:** Lists "--head, --tail" for console
- **Issue:** Not tested
- **Resolution:** Add 3 tests for head/tail/range

**Line 172-179:** Claims comprehensive CSS testing including inline/matched
- **Issue:** Misleading - these weren't tested
- **Resolution:** Update to reflect actual state

**Success Criteria (Line 43):**
```markdown
- [x] css command tests: basic output, selector, computed, get (inline/matched commands don't exist)
```
- **Issue:** Says they "don't exist" but they do in source code
- **Resolution:** Change to "inline/matched not tested - binary needs rebuild"

---

## 7. Test Quality Assessment

### What Was Done Well

**Strengths:**
1. **Comprehensive happy path coverage** - All major features tested in success scenarios
2. **Save mode testing** - Thorough validation of stdout, temp, file, directory modes
3. **Consistent patterns** - Tests follow established framework conventions
4. **Good test structure** - Clear sections, descriptive test names
5. **Proper setup/teardown** - Daemon, server, and resource cleanup
6. **Generous timing** - 2-3 second delays prevent flakiness
7. **Cookie mutations tested** - Set/delete operations validated with verification
8. **Screenshot validation** - File existence AND size checked
9. **Backend infrastructure** - Functions added even though not used yet

**Test reliability:** 80/80 passing consistently

### What Needs Improvement

**Gaps:**
1. **No error testing** - Biggest weakness
2. **No edge cases** - Only sunny day scenarios
3. **No negative tests** - Don't verify failures fail correctly
4. **Missing features** - Range limiting, context flags, inline/matched
5. **Incomplete backend usage** - Infrastructure built but unused
6. **One assertion-less test** - network --find doesn't verify anything

**Test maturity:** Good for initial coverage, needs error/edge case hardening

---

## 8. Binary vs Source Code Investigation

### How This Happened

**Timeline:**
1. **Jan 11, 2026** - webctl binary last built
2. **Commit c07730e** - CSS inline/matched added to source code (after Jan 11)
3. **Jan 17, 2026** - Tests written against old binary
4. **Result** - Source code has features that binary doesn't

**Evidence:**
```bash
$ ls -la ./webctl
-rwxr-xr-x grant grant 14 MB Sun Jan 11 22:02:11 2026 ./webctl

$ git log --oneline | grep "inline and matched"
c07730e feat(css): add inline and matched commands; rework selector logic
```

### Verification Steps Taken

1. Checked binary help output - no inline/matched
2. Checked source code - both commands fully implemented
3. Checked git history - commit exists adding them
4. Checked interactive tests - tests exist for them
5. Verified registration - commands added to cobra

**Conclusion:** Binary is stale, source code is current

---

## 9. Next Steps

### For Test Suite Completion

**Phase 1: Critical (Complete p-061 properly)**
1. Rebuild binary
2. Add CSS inline/matched tests (6 tests)
3. Add error case tests (15 tests)
4. Update project file to reflect actual coverage
5. Re-run suite: target 101/101 passing

**Phase 2: Feature Completion**
6. Add range limiting tests (8 tests)
7. Add context flag tests (9 tests)
8. Re-run suite: target 118/118 passing

**Phase 3: Advanced Testing**
9. Implement backend integration tests (8 tests)
10. Add property edge cases
11. Add selector edge cases
12. Re-run suite: target 126+/126+ passing

### For Project Management

1. **Update p-061 status:**
   - Change from "Done" to "Needs revision"
   - Document gaps in project file
   - Create checklist for completion

2. **Create follow-up project (p-061-b?):**
   - Title: "CLI Observation Tests - Error Cases and Missing Features"
   - Scope: Add the 35-40 missing tests
   - Block p-062 until this is complete?

3. **Update AGENTS.md:**
   - Note p-061 needs revision
   - Block next project or proceed?

---

## 10. Summary

### Current State
- ✅ 80 tests passing
- ✅ Happy path coverage excellent
- ✅ Test framework working well
- ❌ Binary needs rebuild (css inline/matched)
- ❌ No error case testing (major gap)
- ❌ Missing range limiting tests
- ❌ Missing context flag tests
- ❌ Backend infrastructure unused

### Key Findings
1. **CSS inline/matched exist in code but not binary** - rebuild needed
2. **35-40 tests missing from planned scope** - significant gap
3. **Zero error tests implemented** - risky for production use
4. **Project marked complete prematurely** - needs revision

### Recommended Action
**Option A (Thorough):** Reopen p-061, add missing tests, re-complete properly
**Option B (Pragmatic):** Accept current state, document gaps, move forward with p-062

**Recommendation:** Option A - Test quality matters more than schedule
