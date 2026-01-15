# DR-032: Test Framework Architecture

- Date: 2026-01-15
- Status: Accepted
- Category: testing

## Problem

Manual testing of webctl CLI commands is time-consuming and error-prone. The current interactive test scripts require copying/pasting commands and visual verification. This creates several issues:

- Cannot see the UX without typing many commands
- AI agents cannot run tests to verify changes
- No automated regression testing
- Inconsistent test coverage
- 13 manual testing projects (p-038 to p-050) pending

## Decision

Create a Bash-based automated test framework with:

1. A root-level `test-runner` controller script
2. Shared bash modules in `scripts/bash_modules/`
3. Test-specific library in `scripts/test/lib/`
4. Test suites organised by category (cli/, repl/, workflow/)
5. Test pages in `testdata/pages/`
6. Verbose output showing commands and results
7. Support for Go test integration

## Why

Bash for CLI testing:

- Tests the actual CLI binary as users would use it
- Natural fit for testing command-line tools
- Easy to capture stdout, stderr, exit codes
- Simple string/JSON assertions
- Portable across development environments

Verbose output:

- See UX without manually typing commands
- AI agents can run and observe test output
- Easier debugging when tests fail
- Documents expected behavior

Modular structure:

- Reusable logging and assertion functions
- Consistent test patterns across suites
- Easy to add new test files
- Clear separation of concerns

Root-level controller:

- Discoverable entry point (`./test-runner`)
- Unified interface for all test types
- Simple to run from any context

## Structure

```
webctl/
├── test-runner                      # Root controller script
│
├── scripts/
│   ├── bash_modules/                # Shared modules (from ~/bin/scripts/)
│   │   ├── colours.sh               # Color definitions
│   │   ├── terminal.sh              # Log functions
│   │   ├── verify.sh                # Validation functions
│   │   └── user-input.sh            # User prompts
│   │
│   ├── test/
│   │   ├── lib/
│   │   │   ├── test-framework.sh    # Test counters, run_test wrapper
│   │   │   ├── assertions.sh        # assert_*, test-specific checks
│   │   │   └── setup.sh             # Build, daemon, server management
│   │   │
│   │   ├── cli/                     # CLI command tests
│   │   │   ├── test-start-stop.sh
│   │   │   ├── test-navigation.sh
│   │   │   ├── test-observation.sh
│   │   │   ├── test-interaction.sh
│   │   │   ├── test-cookies.sh
│   │   │   ├── test-utility.sh
│   │   │   └── test-serve.sh
│   │   │
│   │   ├── repl/                    # REPL tests
│   │   │   ├── test-repl-basic.sh
│   │   │   └── test-repl-parity.sh
│   │   │
│   │   └── workflow/                # Multi-step integration tests
│   │       ├── test-workflow-form.sh
│   │       └── test-workflow-scrape.sh
│   │
│   └── interactive/                 # Existing manual tests (retained)
│
├── testdata/
│   ├── index.html                   # Existing test page
│   ├── backend.go                   # Existing backend
│   └── pages/                       # Test-specific pages
│       ├── forms.html
│       ├── navigation.html
│       ├── cookies.html
│       ├── console-types.html
│       ├── network-requests.html
│       ├── css-showcase.html
│       ├── slow-load.html
│       ├── scroll-long.html
│       └── click-targets.html
```

## Test Runner Interface

```bash
./test-runner                        # Run all (Go + automated bash)
./test-runner -y                     # Skip prompts
./test-runner --list                 # List available suites

# Go tests
./test-runner go                     # All Go tests
./test-runner go unit                # Unit tests (go test -short)
./test-runner go integration         # Integration tests
./test-runner go race                # With race detector
./test-runner go cover               # With coverage
./test-runner go bench               # Benchmarks

# Automated bash tests
./test-runner cli                    # All CLI tests
./test-runner cli navigation         # Specific test file
./test-runner repl                   # REPL tests
./test-runner workflow               # Workflow tests

# Interactive tests
./test-runner interactive            # Menu of interactive tests
./test-runner interactive cookies    # Specific interactive test

# Quality checks
./test-runner lint                   # go vet, staticcheck
./test-runner fmt                    # Check gofmt

# Shortcuts
./test-runner ci                     # CI-friendly (no prompts, no interactive)
./test-runner quick                  # Fast feedback (go unit + lint)
```

## Verbose Output Format

```
══════════════════════════════════════════════════════════════════════════════

 webctl Test Suite
══════════════════════════════════════════════════════════════════════════════

▶ Setup
────────────────────────────────────────────────────────────────────────────────
 ✔ Building webctl...
 ✔ Starting test server on :8888...
 ✔ Starting daemon (headless)...

▶ Suite: Navigation Commands
────────────────────────────────────────────────────────────────────────────────

▶ Test: navigate to local page
  $ webctl navigate http://localhost:8888/pages/forms.html
  {"ok":true}
 ✔ Exit code: 0
 ✔ Contains: "ok"
 ✔ PASS (0.8s)

▶ Test: back navigation
  $ webctl back
 ✔ Exit code: 0
 ✔ PASS (0.3s)

────────────────────────────────────────────────────────────────────────────────
Suite Results: 5/5 passed (3.2s)

▶ Teardown
────────────────────────────────────────────────────────────────────────────────
 ✔ Stopping daemon...
 ✔ Cleanup complete

══════════════════════════════════════════════════════════════════════════════

 RESULTS: 42/42 tests passed (28.4s)
══════════════════════════════════════════════════════════════════════════════
```

## Assertion Functions

```bash
assert_exit_code <expected> <actual>
assert_equals <expected> <actual>
assert_contains <haystack> <needle>
assert_not_contains <haystack> <needle>
assert_json_field <json> <jq-path> <expected>
assert_json_ok <json>
assert_file_exists <path>
assert_file_contains <path> <needle>
```

## Test Sites

Local (reliable, controlled):

- testdata/pages/* served via `webctl serve testdata`
- testdata/backend.go for proxy testing

External (real-world, stable):

- example.com - simple static HTML
- httpbin.org - HTTP testing (status codes, delays)
- jsonplaceholder.typicode.com - JSON API

## Trade-offs

Accept:

- Bash has limited error handling compared to Go
- String parsing for assertions is fragile
- Tests are slower than pure Go unit tests
- External site tests may be flaky

Gain:

- Tests actual CLI binary end-to-end
- Verbose output shows real UX
- AI agents can run and observe tests
- Easy to write and maintain
- Natural fit for CLI testing
- Reusable bash modules

## Alternatives

Go-based CLI testing:

- Pro: Type-safe, better error handling
- Pro: Integrated with existing test suite
- Con: More boilerplate to exec commands
- Con: Harder to show verbose command output
- Rejected: Bash is more natural for CLI testing

Expect/pexpect:

- Pro: Handles interactive REPL testing well
- Con: Additional dependency
- Con: More complex scripting
- Rejected: Can pipe to REPL stdin instead

Makefile:

- Pro: Standard build tool
- Con: Syntax is arcane
- Con: Less flexibility for test logic
- Rejected: Bash provides better control flow

## Implementation Notes

REPL testing approach:

- Pipe commands to stdin: `echo -e "cmd1\ncmd2\nexit" | webctl start`
- Or run daemon in background and use CLI commands
- Both approaches should be tested

Daemon lifecycle:

- Start once per test suite (not per test)
- Teardown on script exit or Ctrl+C
- Use trap for cleanup

Test isolation:

- Clear console/network buffers between tests
- Navigate to known page before each test group
- Use unique cookie names per test

## Implementation Projects

The framework will be implemented across multiple focused projects:

| Project | Scope | Deliverables |
|---------|-------|--------------|
| p-055 | Bash Modules | Copy bash_modules/ to scripts/, adapt for NO_COLOR |
| p-056 | Test Library | lib/test-framework.sh, assertions.sh, setup.sh |
| p-057 | Test Runner | test-runner script with --help, --list, go/cli/repl dispatch |
| p-058 | Test Pages | testdata/pages/*.html (forms, navigation, cookies, etc.) |
| p-059 | CLI Start/Stop Tests | scripts/test/cli/test-start-stop.sh |
| p-060 | CLI Navigation Tests | scripts/test/cli/test-navigation.sh |
| p-061 | CLI Observation Tests | scripts/test/cli/test-observation.sh |
| p-062 | CLI Interaction Tests | scripts/test/cli/test-interaction.sh |
| p-063 | CLI Cookies Tests | scripts/test/cli/test-cookies.sh |
| p-064 | CLI Utility Tests | scripts/test/cli/test-utility.sh, test-screenshot.sh, test-serve.sh |
| p-065 | REPL Tests | scripts/test/repl/test-repl-basic.sh, test-repl-parity.sh |
| p-066 | Workflow Tests | scripts/test/workflow/test-workflow-*.sh |
| p-067 | Archive Manual Tests | Archive p-038 to p-050 as superseded |

Dependencies:

- p-056, p-057, p-058 depend on p-055
- p-059 through p-066 depend on p-056, p-057, p-058
- p-067 should be done after automated tests prove working
