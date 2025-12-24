# P-009: Design Review & Validation of P-008 Commands

- Status: In Progress
- Started: 2025-12-23

## Overview

Systematic review and validation of the 11 navigation and interaction commands implemented in P-008. The implementation was completed but design decisions were not fully reviewed and validated. This project ensures each command uses the best approach before proceeding with new features.

## Progress

### 2025-12-23: Command Abbreviation Support

Added command shortcuts/abbreviations for all navigation and interaction commands in the REPL.

**What was done:**
- Added all 11 P-008 commands to the `webctlCommands` abbreviation list in `repl.go`
- Updated help text to show categorized commands (Navigation, Interaction, Observation, Utility)
- Updated and expanded test coverage for all new abbreviations
- All tests passing

**Changes:**
- Modified: `/home/grant/Projects/webctl/internal/daemon/repl.go:108-112`
- Modified: `/home/grant/Projects/webctl/internal/daemon/repl.go:265-308` (help text)
- Modified: `/home/grant/Projects/webctl/internal/daemon/repl_test.go:238-418` (tests)

**Available shortcuts:**
- Single char: `h` (html), `k` (key)
- Two char: `ba` (back), `na` (navigate), `ne` (network), `st` (status), `se` (select), `ta` (target), `ty` (type), `ev` (eval)
- Three char: `con` (console), `coo` (cookies), `cle` (clear), `cli` (click), `foc` (focus), `for` (forward), `rea` (ready), `rel` (reload)
- Four char: `scre` (screenshot), `scro` (scroll)

**Ambiguities handled:**
- Single-letter conflicts (n, c, s, t, f, r) require 2+ character prefixes
- Users can type unique prefixes; system expands to full command name

### 2025-12-23: Navigation Commands Design Review & Refactor

Completed comprehensive review and refactor of Group 1 navigation commands based on testing and design discussion.

**Design Decisions:**
1. Navigation commands return immediately (no frameNavigated wait) - fast feedback
2. Added --wait flag to all nav commands for optional page load waiting
3. Added --timeout flag (default 30000ms) when using --wait
4. Reload always does hard reload (removed --ignore-cache flag)
5. Navigate auto-detects URL protocol (https:// default, http:// for localhost)
6. REPL prompt shows URL instead of title (available immediately)

**Implementation Changes:**
- Modified: `internal/cli/navigate.go` - URL normalization, --wait, --timeout flags
- Modified: `internal/cli/reload.go` - Hard reload always, --wait, --timeout flags
- Modified: `internal/cli/back.go` - Added --wait, --timeout flags
- Modified: `internal/cli/forward.go` - Added --wait, --timeout flags
- Modified: `internal/daemon/handlers_navigation.go` - Immediate return with optional wait
- Modified: `internal/daemon/daemon.go` - Updated handler signatures for back/forward
- Modified: `internal/daemon/repl.go` - URL-based prompt with cleanURLForDisplay()
- Modified: `internal/ipc/protocol.go` - Added HistoryParams, updated NavigateParams/ReloadParams
- Updated: `docs/design/design-records/dr-013-navigation-interaction-commands.md` - Major revision

**Rationale:**
- Fast return (<100ms) better for automation than blocking on page load
- URL more useful than title for automation (shows exact location)
- URL available immediately, no need to wait for frameNavigated
- Hard reload default matches automation/testing use case (always want fresh content)
- Protocol auto-detection reduces typing (example.com vs https://example.com)
- Users compose wait behavior explicitly: `navigate url --wait` or `navigate url && ready`

**Examples:**
```bash
# Fast navigation (immediate return)
navigate example.com           # auto-detects https://
navigate localhost:3000        # auto-detects http://
reload                         # hard reload always
back

# Wait for page load when needed
navigate example.com --wait
reload --wait --timeout 10000
back --wait

# REPL prompt shows URL
webctl [example.com]>          # instead of [Example Domain]>
webctl [localhost:3000/api]>   # port and path preserved
```

**Status:**
- Implementation: Complete ✓
- DR-013 documentation: Complete ✓
- Tests: Complete ✓

### 2025-12-24: Navigation Command Tests

Added comprehensive unit tests for all Group 1 navigation commands.

**Tests added to `internal/cli/cli_test.go`:**
- `TestNormalizeURL` - 16 table-driven test cases for URL protocol auto-detection
- `TestRunNavigate_DaemonNotRunning` - error handling when daemon not running
- `TestRunNavigate_Success` - successful navigation with URL normalization
- `TestRunNavigate_WithWaitFlag` - --wait and --timeout flag handling
- `TestRunNavigate_LocalhostUsesHTTP` - localhost uses http:// protocol
- `TestRunNavigate_Error` - navigation error handling
- `TestRunReload_DaemonNotRunning` - error handling
- `TestRunReload_Success` - hard reload (ignoreCache=true always)
- `TestRunReload_WithWaitFlag` - --wait and --timeout flag handling
- `TestRunBack_DaemonNotRunning` - error handling
- `TestRunBack_Success` - successful back navigation
- `TestRunBack_NoHistory` - "no previous page" error
- `TestRunBack_WithWaitFlag` - --wait and --timeout flag handling
- `TestRunForward_DaemonNotRunning` - error handling
- `TestRunForward_Success` - successful forward navigation
- `TestRunForward_NoHistory` - "no next page" error
- `TestRunForward_WithWaitFlag` - --wait and --timeout flag handling

**Test patterns followed (per DR-004):**
- Mock executor with `executeFunc` callback
- Mock factory with `daemonRunning` flag
- Race detection enabled (`go test -race`)
- goleak for goroutine leak detection (via main_test.go)

All tests passing with race detection.

## Goals

1. Review design of all 11 P-008 commands
2. Discuss alternatives and trade-offs for each
3. Validate or refactor implementation based on best practices
4. Update DR-013 with validated design decisions
5. Establish patterns for future command implementations

## Scope

In Scope:

Review 11 commands grouped by similarity:

**Group 1: Navigation Commands (4)**
- `navigate` - Navigate to URL
- `reload` - Reload page
- `back` - Previous history entry
- `forward` - Next history entry

**Group 2: Element Interaction (3)**
- `click` - Click element by selector
- `focus` - Focus element by selector
- `type` - Type text into element

**Group 3: Input Commands (2)**
- `key` - Send keyboard key
- `select` - Select dropdown option

**Group 4: Positioning (1)**
- `scroll` - Scroll to element or position

**Group 5: Synchronization (1)**
- `ready` - Wait for page load

Out of Scope:

- New features or commands
- Performance optimization (unless part of design decision)
- Complex refactoring not related to design validation

## Review Process

For each command/group:

1. Present current implementation design
2. Discuss alternative approaches with pros/cons
3. Recommend best option with rationale
4. User decides final approach
5. Refactor if design changes
6. Update DR-013 documentation

## Success Criteria

- [x] All 5 command groups reviewed (Group 1: Navigation complete)
- [x] Design decisions validated or corrected (Navigation commands refactored)
- [x] Any necessary refactoring completed (Navigation commands refactored)
- [x] DR-013 updated with validated designs (Major revision 2025-12-23)
- [x] All tests still passing after any refactoring (17 new tests added 2025-12-24)
- [x] Patterns documented for future commands (Documented in DR-013)

## Deliverables

- Updated implementation (if refactoring needed)
- Updated DR-013 with validated design decisions
- Design pattern documentation for future commands

## Dependencies

- P-008 (completed implementation to review)

## Notes

This retrospective design review ensures we build on a solid foundation before implementing P-010 (wait-for) and future features.
