# P-051: Observation Commands Output Refactor

- Status: Completed
- Started: 2026-01-08
- Completed: 2026-01-09

## Overview

Refactor observation commands to output to stdout by default instead of saving to temp files. This aligns with Unix conventions and better supports AI agent workflows where direct output processing is preferred.

Current behaviour saves to temp by default, requiring `show` subcommand for stdout. New behaviour makes stdout the default, with `save` subcommand for file output.

## Goals

1. Change default output from save-to-temp to stdout for 5 commands
2. Remove redundant `show` subcommand from all observation commands
3. Make `save` subcommand handle both temp (no path) and custom path output
4. Update screenshot to use `save` subcommand pattern instead of `-o` flag
5. Update all affected design records

## Scope

In Scope:
- html, css, console, network, cookies command refactor
- screenshot command refactor (add `save` subcommand, remove `-o` flag)
- Update DR-025 through DR-029 and DR-011
- Update command help text and examples
- Update interactive test scripts

Out of Scope:
- Changing filter flags (--select, --find, --type, etc.)
- Changing other command behaviours
- Adding new features

## Success Criteria

- [ ] `html` outputs to stdout by default
- [ ] `html save` saves to temp with auto-filename
- [ ] `html save <path>` saves to custom path
- [ ] Same pattern works for css, console, network, cookies
- [ ] `screenshot` saves to temp by default (unchanged)
- [ ] `screenshot save` saves to temp (same as default)
- [ ] `screenshot save <path>` saves to custom path
- [ ] `show` subcommand removed from all commands
- [ ] `-o` flag removed from screenshot
- [ ] All affected DRs updated
- [ ] Interactive test scripts updated
- [ ] All existing tests pass or are updated

## Deliverables

Code changes:
- internal/cli/html.go
- internal/cli/css.go
- internal/cli/console.go
- internal/cli/network.go
- internal/cli/cookies.go
- internal/cli/screenshot.go
- internal/cli/cli_test.go (test updates)

Design record updates:
- docs/design/design-records/dr-025-html-command-interface.md
- docs/design/design-records/dr-026-css-command-interface.md
- docs/design/design-records/dr-027-console-command-interface.md
- docs/design/design-records/dr-028-network-command-interface.md
- docs/design/design-records/dr-029-cookies-command-interface.md
- docs/design/design-records/dr-011-screenshot-command.md

Test script updates:
- scripts/interactive/test-html.sh
- scripts/interactive/test-css.sh
- scripts/interactive/test-console.sh
- scripts/interactive/test-network.sh
- scripts/interactive/test-cookies.sh
- scripts/interactive/test-screenshot.sh

## Technical Approach

New command patterns:

For stdout-capable commands (html, css, console, network, cookies):
```
<cmd>                    # stdout (NEW default)
<cmd> save               # save to temp with auto-filename
<cmd> save <path>        # save to custom path
```

For screenshot (binary output):
```
screenshot               # save to temp (unchanged)
screenshot save          # save to temp (same as default)
screenshot save <path>   # save to custom path
```

Changes per command:
1. Remove `show` subcommand
2. Change root command RunE to output to stdout
3. Rename/refactor default handler to be the show handler
4. Update `save` subcommand to handle optional path argument
5. Remove `-o` flag from screenshot, add `save` subcommand

## Notes

Rationale for change:
- Unix convention: stdout default, explicit save
- AI agent workflows: direct output processing without file reads
- Reduces temp file clutter
- Removes extra `status` IPC call for auto-filename generation (when using stdout)
