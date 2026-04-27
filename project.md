# P-064: Start Attach Mode

Project: Implement --attach flag for connecting to existing browser

Status: Proposed
Started: -
Active: No

## Overview

Add ability to attach webctl daemon to an existing browser CDP endpoint instead of launching a new browser. This enables use cases like:

- Connecting to browser launched by another tool
- Debugging existing browser sessions
- Manual browser setup before automation
- Working with browsers that have specific profiles or extensions

Currently webctl start always launches a new browser instance. The --attach flag was designed in DR-002 but never implemented.

## Goals

1. Implement --attach flag on start command
2. Support connecting to CDP endpoint by port
3. Auto-detect running Chrome/Chromium browsers
4. Validate connection before starting daemon
5. Work with both local and remote CDP endpoints

## Scope

In Scope:

- webctl start --attach :9222 (explicit port)
- webctl start --attach (auto-detect)
- Connection validation before daemon start
- Support for both headed and headless browsers
- Error handling for connection failures

Out of Scope:

- Browser profile management
- Extension installation
- Reconnection on disconnect
- Multi-browser attachment

## Success Criteria

- [ ] webctl start --attach :9222 connects to browser on port 9222
- [ ] webctl start --attach auto-detects running Chrome/Chromium
- [ ] Daemon validates CDP connection before starting
- [ ] Clear error messages for connection failures
- [ ] Compatible with all existing webctl commands
- [ ] Created DR documenting attach mode implementation

## Deliverables

- internal/cli/start.go - Add --attach flag
- internal/daemon/browser.go - Connection logic for existing browser
- internal/daemon/config.go - AttachURL configuration
- tests/cli/start-attach.bats - Attach mode tests
- DR documenting attach mode design

## Current State

- start command only launches new browser via launcher.Launch()
- DR-002 designed --attach flag but not implemented
- Config has Port field but no AttachURL field
- Daemon assumes it owns the browser lifecycle

## Technical Approach

Add AttachURL to daemon config:

```go
type Config struct {
    AttachURL string  // If set, connect instead of launch
    Port      int     // Ignored if AttachURL set
    // ... existing fields
}
```

Connection logic:

1. If --attach flag provided, set AttachURL
2. If --attach with no argument, auto-detect via:
   - Check default Chrome ports (9222, 9223, etc.)
   - Query process list for Chrome --remote-debugging-port
3. Connect to CDP endpoint instead of launching
4. Validate connection with Target.getTargets
5. Start daemon with existing browser

Command syntax:

```
webctl start --attach :9222          # Connect to port 9222
webctl start --attach localhost:9222 # Explicit host:port
webctl start --attach                # Auto-detect
```

Error handling:

- Connection refused: Clear message with port number
- No browser found: Suggest launching Chrome manually
- Invalid endpoint: Validate URL format

## Decision Points

1. Auto-detect behavior when multiple browsers running

- A: Connect to first found (port order priority)
- B: List all and require user selection
- C: Return error and require explicit port

2. Browser lifecycle management

- A: Never close attached browser on stop
- B: Add flag to control close behavior
- C: Always ask user on stop

## Related Work

- Builds on: p-004 (Browser Launch)
- Builds on: p-005 (Daemon & IPC)
- Implements: dr-002 (CLI Browser Commands design)
- Enables: Manual browser setup workflows
