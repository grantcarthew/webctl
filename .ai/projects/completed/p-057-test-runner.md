# p-057: Test Runner

- Status: Done
- Started: 2026-01-16
- Completed: 2026-01-16
- Design Record: .ai/design/design-records/dr-032-test-framework-architecture.md

## Overview

Create the root-level test-runner script that provides a unified interface for running all test types (Go tests, automated bash tests, interactive tests, and quality checks).

## Goals

1. Create test-runner script with help and list functionality
2. Implement Go test dispatch (unit, integration, race, cover, bench)
3. Implement CLI/REPL/workflow test dispatch
4. Implement interactive test menu
5. Add quality check commands (lint, fmt)
6. Add convenience shortcuts (ci, quick)

## Scope

In Scope:

- Root-level test-runner script
- Command dispatch to Go tests
- Command dispatch to bash test suites
- Interactive test menu selection
- Quality check wrappers
- Help output and usage
- List available test suites

Out of Scope:

- Actual test scripts (p-059+)
- Test pages (p-058)
- Test library (p-056, completed)

## Success Criteria

- [x] ./test-runner shows help when run without args
- [x] ./test-runner --help shows full usage
- [x] ./test-runner --list shows available suites
- [x] ./test-runner go runs all Go tests
- [x] ./test-runner go unit runs short tests
- [x] ./test-runner go race runs with race detector
- [x] ./test-runner go cover runs with coverage
- [x] ./test-runner cli dispatches to CLI tests
- [x] ./test-runner repl dispatches to REPL tests
- [x] ./test-runner interactive shows menu
- [x] ./test-runner lint runs go vet and staticcheck
- [x] ./test-runner ci runs non-interactive tests
- [x] ./test-runner quick runs fast feedback loop

## Deliverables

- test-runner (root-level executable script)

## Technical Approach

Script structure:

- Bash script with subcommand dispatch
- Sources bash_modules for terminal output
- Uses case statement for command routing
- Discovers test files via glob patterns

Commands:

```
./test-runner                        # Show help
./test-runner -y                     # Skip prompts (future use)
./test-runner --list                 # List available suites

# Go tests
./test-runner go                     # All Go tests
./test-runner go unit                # go test -short
./test-runner go integration         # go test (full)
./test-runner go race                # go test -race
./test-runner go cover               # go test -cover
./test-runner go bench               # go test -bench

# Bash tests
./test-runner cli                    # All CLI tests
./test-runner cli navigation         # Specific test file
./test-runner repl                   # REPL tests
./test-runner workflow               # Workflow tests

# Interactive
./test-runner interactive            # Menu of interactive tests
./test-runner interactive cookies    # Specific interactive test

# Quality
./test-runner lint                   # go vet, staticcheck
./test-runner fmt                    # gofmt check

# Shortcuts
./test-runner ci                     # CI-friendly (go + lint + cli)
./test-runner quick                  # Fast feedback (go unit + lint)
```

Test discovery:

- CLI tests: scripts/test/cli/test-*.sh
- REPL tests: scripts/test/repl/test-*.sh
- Workflow tests: scripts/test/workflow/test-*.sh
- Interactive: scripts/interactive/test-*.sh

## Dependencies

- p-056: Test Library (completed)

## Current State

### Environment

- Go: 1.25.6
- staticcheck: 2025.1.1

### Bash Modules (scripts/bash_modules/)

All 7 modules present:

| Module | Purpose |
|--------|---------|
| colours.sh | Colour definitions, NO_COLOR support |
| terminal.sh | log_* functions (success, failure, heading, etc.) |
| verify.sh | Input validation helpers |
| user-input.sh | User prompts |
| test-framework.sh | TEST_PASS/FAIL/TOTAL counters, run_test(), test_summary() |
| assertions.sh | 20+ assert_* functions (exit codes, strings, JSON, files) |
| setup.sh | build_webctl(), start/stop daemon, start/stop test server, cleanup |

Bug found: setup.sh:49 uses `SETUP_PROJECT_ROOT` but variable is `PROJECT_ROOT`. Fix during implementation.

### Existing Scripts (to be replaced)

scripts/test.sh (239 lines):
- Provides Go test functionality - will be replaced by `test-runner go`
- Delete after test-runner is complete

scripts/interactive/test-*.sh (25 files):
- Manual testing scripts - will be superseded by automated tests
- Retained for now, archived in future project (p-067)

### Missing Infrastructure

- scripts/test/ directory does not exist (create empty cli/, repl/, workflow/ subdirs)
- No root-level test-runner script

## Decision Points

1. **No-args behaviour**: DR-032 specifies `./test-runner` runs all tests (Go + automated bash), but this project specifies it shows help. Which is correct?
   - **A. Show help (as specified in this project)** ✓
   - B. Run all tests (as specified in DR-032)

## Notes

- Script should be executable (chmod +x)
- Use set -e for early exit on errors
- Provide clear error messages for missing test suites
- Support running from project root only
- Go tests target `./internal/...` (matching existing scripts/test.sh)
- `go bench` should use `-bench=.` pattern for all benchmarks
