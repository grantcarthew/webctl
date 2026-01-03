# P-025: Interactive Test Suite

- Status: Completed
- Started: 2025-12-31
- Completed: 2026-01-03

## Overview

Create a comprehensive interactive testing system for webctl that enables systematic manual testing of all commands, arguments, and flags through guided bash scripts. Each test script walks through test cases, copies commands to the clipboard, and waits for user validation before proceeding.

This project establishes a standardized workflow for discovering and documenting issues, determining fix complexity, and tracking tested features through project checklists.

## Goals

1. Document all webctl commands, subcommands, flags, and arguments through code inspection
2. Create individual project files for each command's test suite
3. Generate interactive bash test scripts that guide manual testing
4. Establish workflow for issue discovery, assessment, and resolution
5. Provide systematic test coverage across both CLI and REPL modes

## Scope

In Scope:

- All 25 current webctl commands (start, stop, status, navigate, html, css, console, network, cookies, screenshot, click, type, select, scroll, eval, ready, clear, back, forward, reload, serve, find, focus, key, target, selectcmd)
- Both CLI mode (webctl <command>) and REPL mode (<command>) testing
- Global flags (--debug, --json, --no-color)
- Command-specific flags and arguments
- Interactive test scripts with xclip clipboard integration

Out of Scope:

- Automated test execution (these are manual interactive tests)
- Test assertions or validation (relies on user observation)
- Performance testing or benchmarking
- Integration with CI/CD pipelines

## Success Criteria

- [x] Complete inventory of all commands and their options documented
- [x] One project file per command describing what to test
- [x] One interactive test script per command in scripts/interactive/
- [x] Test scripts handle both CLI and REPL modes
- [x] Setup sections included for tests requiring daemon/browser state
- [x] All scripts use xclip for clipboard integration
- [x] Workflow documented for handling discovered issues
- [x] README.md updated with new P-025 entry

## Deliverables

- docs/projects/p-025-testing-interactive-test-suite.md (this file)
- docs/projects/p-026-testing-{command}.md for each command
- scripts/interactive/test-{command}.sh for each command
- Updated docs/projects/README.md with P-025 entry
- Command inventory documentation (in this project file)

## Command Inventory

Based on code inspection of internal/cli/:

Daemon Management:
- start [--headless] [--port 9222]
- stop
- status

Server:
- serve <directory|--proxy url>

Navigation:
- navigate <url> [--wait] [--timeout ms]
- reload
- back
- forward

Observation (default/show/save pattern):
- html [show|save path] [--select sel] [--find text] [--raw]
- css [show|save path|computed sel|get sel prop] [--select sel] [--find text] [--raw]
- console [show|save path] [--find text] [--type type] [--head n] [--tail n] [--range n-m] [--raw]
- network [show|save path] [--status code] [--method method] [--find text] [--head n] [--tail n] [--range n-m] [--raw]
- cookies [show|save path] [--domain domain] [--find text] [--raw]
- cookies set <name> <value> [--domain d] [--path p] [--expires epoch] [--http-only] [--secure] [--same-site val]
- cookies delete <name>
- screenshot [save path]

Interaction:
- click <selector>
- type <selector> <text>
- select <selector> <value>
- scroll <selector|position>
- focus <selector>
- key <key-name>

Utility:
- eval <expression>
- ready [selector] [--network-idle] [--eval condition] [--timeout ms]
- clear [console|network]
- find <text>
- target

Global Flags (apply to all commands):
- --debug
- --json
- --no-color

## Testing Workflow

1. Run interactive test script for a command
2. Script displays test case title and command
3. Command copied to clipboard via xclip
4. User pastes and executes command (in CLI or REPL as indicated)
5. User observes behavior and presses Enter to continue
6. If issue discovered:
   - User reports issue to AI
   - AI assesses complexity
   - Simple fix: Implement immediately
   - Complex fix: Create new project file (P-NNN)
7. After all tests for a command:
   - Update that command's project file checklist
   - Mark tested features as complete

## Technical Approach

Test Script Structure:

```bash
#!/bin/bash
# Title: webctl {command} command tests

set -e

# Color output helpers
title() { echo -e "\n\033[1;34m=== $1 ===\033[0m"; }
heading() { echo -e "\n\033[1;32m## $1\033[0m"; }
cmd() {
    echo -e "\n\033[0;33m$ $1\033[0m"
    echo "$1" | xclip -selection clipboard
    echo "(Command copied to clipboard)"
    read -p "Press Enter to continue..."
}

# Setup section (if needed for this command)
title "Setup"
echo "Starting webctl daemon..."
# ... setup commands ...

# CLI Tests
title "CLI Mode Tests"

heading "Basic usage"
cmd "webctl {command}"

heading "With flags"
cmd "webctl {command} --flag value"

# REPL Tests
title "REPL Mode Tests"
echo "Switch to daemon terminal for REPL tests"
read -p "Press Enter when ready..."

heading "Basic usage in REPL"
cmd "{command}"

heading "With flags in REPL"
cmd "{command} --flag value"

# Cleanup
title "Cleanup"
echo "Test session complete"
```

Project File Structure (per command):

- Introduction to the command
- Code references (internal/cli/{command}.go)
- Documentation references (if any)
- Checklist of all variations to test
- Workflow notes

## Project Files to Create

One project file for each command (P-026 through P-050):

- P-026: start command
- P-027: stop command
- P-028: status command
- P-029: serve command
- P-030: navigate command
- P-031: reload command
- P-032: back command
- P-033: forward command
- P-034: html command
- P-035: css command
- P-036: console command
- P-037: network command
- P-038: cookies command
- P-039: screenshot command
- P-040: click command
- P-041: type command
- P-042: select command
- P-043: scroll command
- P-044: focus command
- P-045: key command
- P-046: eval command
- P-047: ready command
- P-048: clear command
- P-049: find command
- P-050: target command

## Test Scripts to Create

One script file for each command in scripts/interactive/:

- test-start.sh
- test-stop.sh
- test-status.sh
- test-serve.sh
- test-navigate.sh
- test-reload.sh
- test-back.sh
- test-forward.sh
- test-html.sh
- test-css.sh
- test-console.sh
- test-network.sh
- test-cookies.sh
- test-screenshot.sh
- test-click.sh
- test-type.sh
- test-select.sh
- test-scroll.sh
- test-focus.sh
- test-key.sh
- test-eval.sh
- test-ready.sh
- test-clear.sh
- test-find.sh
- test-target.sh

## Notes

- AGENTS.md appears outdated regarding browser command (not found in codebase)
- REPL mode accessed via daemon terminal, commands same as CLI but without "webctl" prefix
- Some commands (observation pattern) have complex subcommand structures to test
- Setup requirements vary by command (some need pages loaded, forms present, etc.)
- Global flags should be tested with at least one command from each category

## Questions & Uncertainties

- Should we create a master test-all.sh script that runs all individual scripts?
- How should we handle tests that require external websites (vs local serve)?
- Should test scripts validate xclip is installed before running?
- How to test error conditions systematically?

## Updates

2025-12-31: Project created, command inventory completed
2025-12-31: All 25 project files (P-026 through P-050) created
2025-12-31: All 25 interactive test scripts created in scripts/interactive/
2025-12-31: All success criteria met - project infrastructure complete
