# p-055: Test Framework Bash Modules

- Status: Done
- Started: 2026-01-15
- Completed: 2026-01-15
- Design Record: .ai/design/design-records/dr-032-test-framework-architecture.md

## Overview

Copy and adapt bash modules from ~/bin/scripts/bash_modules/ to scripts/bash_modules/ for use by the test framework. This provides the foundational logging, color, verification, and user input functions needed by all subsequent test framework projects.

## Goals

1. Create scripts/bash_modules/ directory
2. Copy colours.sh with NO_COLOR environment variable support
3. Copy terminal.sh adapted to be self-contained (source colours.sh locally)
4. Copy verify.sh for validation and dependency checking
5. Copy user-input.sh for optional prompts (--yes bypass support)

## Scope

In Scope:

- Copying and adapting the four bash module files
- Adding NO_COLOR support to colours.sh
- Updating source paths to be relative/local
- Basic validation that modules load correctly

Out of Scope:

- Test framework library (p-056)
- Test runner script (p-057)
- Test pages (p-058)
- Any actual test scripts

## Success Criteria

- [x] scripts/bash_modules/colours.sh exists and exports color variables
- [x] scripts/bash_modules/colours.sh respects NO_COLOR environment variable
- [x] scripts/bash_modules/terminal.sh exists and provides log_* functions
- [x] scripts/bash_modules/verify.sh exists and provides validation functions
- [x] scripts/bash_modules/user-input.sh exists and provides prompt functions
- [x] All modules can be sourced without errors
- [x] Simple test: source all modules, call log_title "Test", verify output

## Deliverables

- scripts/bash_modules/colours.sh
- scripts/bash_modules/terminal.sh
- scripts/bash_modules/verify.sh
- scripts/bash_modules/user-input.sh

## Technical Approach

1. Create scripts/bash_modules/ directory
2. Copy each file from ~/bin/scripts/bash_modules/
3. Modify colours.sh to check NO_COLOR and disable colors if set
4. Modify terminal.sh to source colours.sh from same directory (BASH_MODULES_DIR)
5. Test by creating a simple script that sources all modules and calls a few functions

## Source Files

From ~/bin/scripts/bash_modules/:

- colours.sh (358 bytes) - color variable exports
- terminal.sh (9.6 KB) - log_* functions
- verify.sh (7.0 KB) - validation/assertion functions
- user-input.sh (4.1 KB) - interactive prompt functions

## Implementation Summary

Completed 2026-01-15:

1. Created `scripts/bash_modules/` directory
2. Created `colours.sh` with NO_COLOR support (when set, all colour vars are empty strings)
3. Copied `terminal.sh` as-is (already uses BASH_MODULES_DIR for local sourcing)
4. Copied `verify.sh` with macOS fix: changed `head --lines=1 | grep -Pq` to `head -n 1 | grep -Eq`
5. Copied `user-input.sh` as-is (already uses BASH_MODULES_DIR for local sourcing)
6. Validated all modules source without errors
7. Tested log_title output with and without NO_COLOR

## Decision Points

1. Fix macOS compatibility in `is_bash_script()` function?
   - **A. Fix it** - changed to portable syntax (`head -n 1`, `grep -Eq`)

## Notes

- The bash_modules/ directory is at scripts/ level (not inside test/) so it can be shared by other scripts if needed
- NO_COLOR is a standard (https://no-color.org/) - when set, CLI tools should not emit color codes
- Terminal.sh already outputs to stderr which is correct for logging
