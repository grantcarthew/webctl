# P-063: New Tab Command

Project: Implement command to open new tabs in existing browser

Status: Proposed
Started: -
Active: No

## Overview

Add ability to create new browser tabs from the CLI. Currently, users can only work with tabs that already exist (via target command) or tabs opened by the browser itself. There is no way to programmatically open a new tab.

This enables multi-tab workflows for agents and users, such as comparing pages side-by-side, testing multiple scenarios, or organizing work across tabs.

## Goals

1. Implement newtab command to create new browser tab
2. Support optional URL argument for immediate navigation
3. New tab becomes the active session automatically
4. Integrate with existing target command for tab management

## Scope

In Scope:

- newtab command with optional URL argument
- CDP Target.createTarget implementation
- Automatic session activation after creation
- Integration with daemon session tracking
- Text and JSON output formats

Out of Scope:

- Window management (creating new windows vs tabs)
- Tab positioning or ordering
- Closing tabs (separate feature)
- Tab groups or organization features

## Success Criteria

- [ ] newtab command creates new tab successfully
- [ ] newtab <url> creates tab and navigates to URL
- [ ] New tab becomes active session after creation
- [ ] target command shows new tab in list
- [ ] Command works in both text and JSON output modes
- [ ] Created DR documenting implementation approach

## Deliverables

- internal/cli/newtab.go - New tab command implementation
- internal/daemon/handler.go - CDP createTarget integration
- tests/cli/newtab.bats - Command tests
- DR documenting new tab implementation

## Current State

- target command lists and switches between existing sessions
- Daemon tracks all browser tabs via CDP Target domain
- No programmatic way to create new tabs
- Browser can open new tabs via user interaction or window.open()

## Technical Approach

Use CDP Target.createTarget to create new browser tab:

```
Target.createTarget({url: "about:blank", newWindow: false})
```

Command syntax:

```
webctl newtab                    # Opens blank tab
webctl newtab <url>              # Opens tab with URL
webctl newtab https://example.com
```

Integration points:

- Daemon handler adds "newtab" command
- Use existing session tracking to activate new tab
- Return new session ID in response
- Compatible with existing ready command for page load waiting

## Related Work

- Builds on: p-005 (Daemon & IPC)
- Builds on: dr-010 (Browser-level CDP sessions)
- Complements: target command (tab switching)
