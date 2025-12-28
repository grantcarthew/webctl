# P-022: Network Command Implementation

- Status: Proposed
- Started: (not yet started)
- Completed: (not yet completed)

## Overview

**This is a breaking redesign and migration project.** Refactor the existing network command implementation to follow the new unified observation pattern defined in DR-028.

The current network command (stdout-only) will be restructured to:
- Support default/show/save output modes (breaking: default changes from stdout to temp file)
- Add universal flags (--find, --raw, --json)
- Retain all network-specific filter flags (--type, --method, --status, --url, --mime, --min-duration, --min-size, --failed, --head, --tail, --range)

This migration brings the network command into alignment with the universal observation pattern.

## Goals

1. Implement new network command interface per DR-028
2. Add default/show/save subcommands for output mode control
3. Add universal flags (--find, --raw, --json)
4. Retain network-specific filter flags (--type, --method, --status, --url, --mime, --min-duration, --min-size, --failed, --head, --tail, --range)
5. Update CLI command file (internal/cli/network.go)
6. Update daemon handlers if needed
7. Add/update tests for new interface
8. Update CLI documentation

## Scope

In Scope:
- Network command interface redesign (DR-028)
- Default behavior (save to temp with auto-generated filename)
- Show subcommand (output to stdout)
- Save subcommand (save to custom path)
- Universal flags (--find, --raw, --json)
- Network-specific filter flags (all existing filters)
- Path handling (directory vs file detection)
- File naming pattern updates
- Color-coded output for show mode (status code colors)
- Integration tests
- Documentation updates

Out of Scope:
- Changes to other observation commands
- Network request buffering changes
- CDP event handling changes

## Success Criteria

- [ ] Default (no subcommand) saves requests to temp
- [ ] Show subcommand outputs requests to stdout
- [ ] Save <path> subcommand saves to custom path
- [ ] Directory paths auto-generate filenames
- [ ] --find flag searches within URLs and bodies
- [ ] All network-specific filters work (--type, --method, --status, --url, --mime, --min-duration, --min-size, --failed)
- [ ] --head/tail/range flags limit results
- [ ] --raw flag skips formatting
- [ ] --json flag outputs JSON format
- [ ] Show mode has color-coded output by status
- [ ] All filters AND-combine correctly
- [ ] All existing tests pass
- [ ] New tests cover all modes and flags
- [ ] Documentation updated

## Deliverables

- Updated internal/cli/network.go
- Updated internal/daemon/handlers_network.go (if needed)
- Updated tests in internal/cli/network_test.go
- Updated docs/cli/network.md
- Updated AGENTS.md

## Technical Approach

Add default/show/save subcommands to network command. Retain all existing filter flags and add universal --find flag. Default behavior saves to /tmp/webctl-network/ with JSON format. Show subcommand outputs formatted table with color-coding by status code (2xx green, 4xx yellow, 5xx red). All filters AND-combine for precise targeting.

Testing follows DR-004 strategy with race detection and integration tests.

## Dependencies

- DR-028: Network Command Interface (design authority)
- DR-004: Testing Strategy (testing approach consistency)
- Existing network command implementation

## Notes

- Breaking change: Default behavior switches from stdout to file
- Migration: Users must add `show` subcommand to maintain stdout behavior
- Network has most extensive filtering of all observation commands

## Updates

- 2025-12-28: Project created
