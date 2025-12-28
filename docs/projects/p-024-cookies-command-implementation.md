# P-023: Cookies Command Implementation

- Status: Proposed
- Started: (not yet started)
- Completed: (not yet completed)

## Overview

**This is a breaking redesign and migration project.** Refactor the existing cookies command implementation to follow the new unified observation pattern defined in DR-029.

The current cookies command will be restructured to:
- Support default/show/save output modes for observation (breaking: default changes to temp file)
- Add universal flags (--find, --raw, --json)
- Add cookies-specific filter flags (--domain, --name)
- Retain set/delete mutation subcommands unchanged

This migration establishes clear separation between read operations (observation pattern) and write operations (mutation subcommands) while bringing cookies into alignment with the universal observation pattern.

## Goals

1. Implement new cookies command interface per DR-029
2. Add default/show/save subcommands for observation (reading cookies)
3. Add universal flags (--find, --raw, --json)
4. Add cookies-specific filter flags (--domain, --name)
5. Retain set and delete subcommands for mutation (writing cookies)
6. Update CLI command file (internal/cli/cookies.go)
7. Update daemon handlers if needed
8. Add/update tests for new interface
9. Update CLI documentation

## Scope

In Scope:
- Cookies command interface redesign (DR-029)
- Default behavior for observation (save to temp)
- Show subcommand (output to stdout)
- Save subcommand (save to custom path)
- Universal flags for observation (--find, --raw, --json)
- Cookies-specific filter flags (--domain, --name)
- Retain set subcommand (cookie creation/update)
- Retain delete subcommand (cookie deletion)
- Path handling (directory vs file detection)
- File naming pattern updates
- Integration tests
- Documentation updates

Out of Scope:
- Changes to other observation commands
- Cookie storage/retrieval mechanism changes
- CDP protocol changes

## Success Criteria

- [ ] Default (no subcommand) saves cookies to temp
- [ ] Show subcommand outputs cookies to stdout
- [ ] Save <path> subcommand saves to custom path
- [ ] Directory paths auto-generate filenames
- [ ] --find flag searches within cookie names and values
- [ ] --domain flag filters by domain
- [ ] --name flag filters by exact name
- [ ] --raw flag skips formatting
- [ ] --json flag outputs JSON format
- [ ] set subcommand works (create/update cookie)
- [ ] delete subcommand works (delete cookie)
- [ ] All existing tests pass
- [ ] New tests cover all modes and flags
- [ ] Documentation updated

## Deliverables

- Updated internal/cli/cookies.go
- Updated internal/daemon/handlers_cookies.go (if needed)
- Updated tests in internal/cli/cookies_test.go
- Updated docs/cli/cookies.md
- Updated AGENTS.md

## Technical Approach

Add default/show/save subcommands for cookie observation. Universal pattern applies only to read operations (getting cookies). Mutation subcommands (set/delete) remain separate to maintain clear distinction between read and write operations. Default behavior saves to /tmp/webctl-cookies/ with JSON format. Filter flags (--domain, --name, --find) apply only to observation modes, not mutations.

Testing follows DR-004 strategy with race detection and integration tests.

Command Structure:
- Root command (default): observation - save to temp
- show subcommand: observation - output to stdout
- save subcommand: observation - save to custom path
- set subcommand: mutation - create/update cookie
- delete subcommand: mutation - delete cookie

Flags:
- --find, --raw, --json: universal (observation only)
- --domain, --name: cookies-specific (observation only)
- set/delete have their own flags (--secure, --httponly, etc.)

## Dependencies

- DR-029: Cookies Command Interface (design authority)
- DR-004: Testing Strategy (testing approach consistency)
- P-019: Implementation Gotchas section (unknown subcommand validation pattern)
- Existing cookies command implementation

## Notes

- Breaking change: Default behavior switches from stdout to file (if cookies currently outputs to stdout)
- Mutation operations (set/delete) remain unchanged
- Clear separation: observation (read) vs mutation (write)

## Updates

- 2025-12-28: Project created
