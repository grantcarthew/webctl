# P-021: Console Command Implementation

- Status: Proposed
- Started: (not yet started)
- Completed: (not yet completed)

## Overview

**This is a breaking redesign and migration project.** Refactor the existing console command implementation to follow the new unified observation pattern defined in DR-027.

The current console command (stdout-only) will be restructured to:
- Support default/show/save output modes (breaking: default changes from stdout to temp file)
- Add universal flags (--find, --raw, --json)
- Retain console-specific filter flags (--type, --head, --tail, --range)

This migration brings the console command into alignment with the universal observation pattern.

## Goals

1. Implement new console command interface per DR-027
2. Add default/show/save subcommands for output mode control
3. Add universal flags (--find, --raw, --json)
4. Retain console-specific filter flags (--type, --head, --tail, --range)
5. Update CLI command file (internal/cli/console.go)
6. Update daemon handlers if needed
7. Add/update tests for new interface
8. Update CLI documentation

## Scope

In Scope:
- Console command interface redesign (DR-027)
- Default behavior (save to temp with auto-generated filename)
- Show subcommand (output to stdout)
- Save subcommand (save to custom path)
- Universal flags (--find, --raw, --json)
- Console-specific filter flags (--type, --head, --tail, --range)
- Path handling (directory vs file detection)
- File naming pattern updates
- Color-coded output for show mode
- Integration tests
- Documentation updates

Out of Scope:
- Changes to other observation commands
- Console log buffering changes
- CDP event handling changes

## Success Criteria

- [ ] Default (no subcommand) saves logs to temp
- [ ] Show subcommand outputs logs to stdout
- [ ] Save <path> subcommand saves to custom path
- [ ] Directory paths auto-generate filenames
- [ ] --find flag searches within log messages
- [ ] --type flag filters by log type
- [ ] --head/tail/range flags limit results
- [ ] --raw flag skips formatting
- [ ] --json flag outputs JSON format
- [ ] Show mode has color-coded output
- [ ] All existing tests pass
- [ ] New tests cover all modes and flags
- [ ] Documentation updated

## Deliverables

- Updated internal/cli/console.go
- Updated internal/daemon/handlers_console.go (if needed)
- Updated tests in internal/cli/console_test.go
- Updated docs/cli/console.md
- Updated AGENTS.md

## Technical Approach

Add default/show/save subcommands to console command. Retain existing filter flags (--type, --head, --tail, --range) and add universal flags (--find). Default behavior saves to /tmp/webctl-console/ with JSON format. Show subcommand outputs formatted table with color-coding.

Testing follows DR-004 strategy with race detection and integration tests.

## Dependencies

- DR-027: Console Command Interface (design authority)
- DR-004: Testing Strategy (testing approach consistency)
- P-019: Implementation Gotchas section (unknown subcommand validation pattern)
- Existing console command implementation

## Notes

- Breaking change: Default behavior switches from stdout to file
- Migration: Users must add `show` subcommand to maintain stdout behavior

## Updates

- 2025-12-28: Project created
