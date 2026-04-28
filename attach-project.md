# Start Attach Mode

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

Currently webctl start always launches a new browser instance. The --attach flag was previously scoped but never implemented.

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

## Deliverables

- internal/cli/start.go - Add --attach flag and CLI parsing of `--attach :PORT` / `host:port` / empty for auto-detect
- internal/browser/attach.go - New file: attach to an existing CDP endpoint (and auto-detect helper)
- internal/browser/browser.go - Refactor Browser to support attach mode (see Issue 3)
- internal/daemon/daemon.go - Add AttachURL to Config; branch in Run() between Start() and Attach()
- internal/browser/attach_test.go - Go unit tests for endpoint parsing, validation, auto-detect
- scripts/test/cli/test-start-attach.sh - Shell-driven CLI tests covering flag combinations and error messages

## Current State

- internal/cli/start.go calls daemon.New(cfg) and d.Run(ctx); the daemon launches a browser via browser.Start() in internal/daemon/daemon.go (Run method).
- internal/browser/browser.go defines Browser with cmd, port, dataDir, ownsData. Close() always SIGTERMs the process and removes the temp data dir.
- internal/daemon/daemon.go defines Config inline (no separate config.go) with Headless, Port, SocketPath, PIDPath, BufferSize, Debug, CommandExecutor — no AttachURL field yet.
- internal/browser/target.go provides FetchVersion(ctx, host, port) and FetchTargets(ctx, host, port) — these already accept a host parameter and can be reused for attach validation without a separate browser process.
- internal/cli/stop.go has a --force path that kills any browser on the configured port via lsof. This would kill an attached browser.
- The previous Tab Command project has shipped (commits d557737, 9b3cd63); project.md was renamed from cli-start-attach-mode.md in commit 5106aa4 and AGENTS.md has been updated to point at this project.

## Issues Discovered

1. Stale prior-design references removed (gap) — Resolved.

   The Overview and Deliverables previously pointed at a separate design document for attach mode that is no longer part of this repository. Those references and the matching deliverable have been removed. Design decisions for attach mode will be captured inline in this project document.

2. Deliverables paths do not match the actual codebase (gap) — Resolved.

   - internal/daemon/browser.go does not exist. Browser launch and CDP plumbing live in internal/browser/browser.go and internal/daemon/daemon.go (Run method).
   - internal/daemon/config.go does not exist. daemon.Config is defined inline in internal/daemon/daemon.go.
   - tests/cli/start-attach.bats assumes a bats test framework. The project uses scripts/test/cli/test-*.sh shell scripts plus Go *_test.go files; no bats infrastructure is present.
   Resolution: Deliverables updated to match the actual layout — `internal/cli/start.go`, new `internal/browser/attach.go`, refactor of `internal/browser/browser.go`, edits to `internal/daemon/daemon.go` (Config + Run branch), `internal/browser/attach_test.go`, and `scripts/test/cli/test-start-attach.sh`.

3. browser.Browser struct is launch-centric and Daemon owns the lifecycle (design) — Resolved: attached flag on existing struct.

   internal/browser/browser.go's Browser embeds *exec.Cmd and a dataDir it deletes on Close(). internal/daemon/daemon.go always calls defer d.browser.Close(). The struct has no field signalling "connected but not owned", so without a change every daemon-exit path (Ctrl-C, REPL EOF, IPC shutdown, server error) would SIGTERM the user's externally-launched browser.
   Resolution: Add `host` and `attached` fields to the existing Browser struct. Introduce `browser.Attach(host string, port int) (*Browser, error)` that validates via FetchVersion and returns `&Browser{host, port, attached: true}`. Gate Close() to no-op the SIGTERM/SIGKILL and dataDir cleanup when attached. Default `host` to "127.0.0.1" for launched browsers; this also pays for Issue 6 (remote endpoints). The daemon's `defer d.browser.Close()` stays as-is. stop.go --force semantics are tracked separately under Issue 5.

4. Auto-detect behavior when multiple browsers running (decision)

   A. Connect to the first found (port-order priority).
   B. List all and require explicit user selection.
   C. Return error and require an explicit endpoint.

5. Browser lifecycle management on stop (decision)

   A. Never close an attached browser on stop (and never delete its profile).
   B. Add a flag (e.g. --close-on-stop) to control close behaviour.
   C. Prompt the user on stop (only viable in interactive mode).
   This decision also constrains how internal/cli/stop.go --force behaves (it currently kills any browser on the CDP port via lsof).

6. Remote CDP endpoint scope is under-specified (gap)

   Scope says "Work with both local and remote CDP endpoints" and the syntax examples include localhost:9222, but the document does not address: non-loopback hosts (CDP grants full browser control with no auth), wss/TLS-fronted endpoints, URL forms accepted (host:port vs http(s)://host:port vs ws(s)://...), and how Browser.Version() — currently hardcoded to 127.0.0.1 — is reached for remote hosts. Without bounds, an implementer will either ship a security footgun or scope-creep into URL parsing and TLS plumbing.
   Suggested resolution: Either remove "remote" from Scope and constrain attach mode to loopback, or specify the accepted URL forms, document the lack of CDP authentication, and decide whether wss is in scope.

7. --headless interaction with --attach is unspecified (decision)

   With --attach, the daemon no longer controls browser launch, so --headless has no effect. The project does not say what happens when both flags are passed.
   A. Error if --headless and --attach are combined.
   B. Silently ignore --headless when --attach is set.
   C. Document --headless as a no-op under --attach without erroring.

8. Auto-detect strategy lacks concrete bounds (gap)

   The Technical Approach says "Check default Chrome ports (9222, 9223, etc.)" and "Query process list for Chrome --remote-debugging-port" without specifying the port range, the per-probe timeout, or whether process-list scanning is required (or limited to platforms supported elsewhere — Linux/macOS via lsof and ps). The implementer will pick something arbitrary that may need to be revisited.
   Suggested resolution: Fix the probe range (e.g. 9222–9229) and a short per-probe timeout (e.g. 200 ms), and decide whether process-list scanning is in scope or whether port probing alone is sufficient.

9. AGENTS.md was out of sync with the active project (gap) — Resolved.

   AGENTS.md previously listed the wrong active project and pointed readers at a projects directory that no longer exists. AGENTS.md and `.ai/workflow.md` have been updated to reflect the current layout (project.md at the repository root) and to name this project as active.

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

- Builds on the existing browser launch and daemon/IPC infrastructure
- Enables manual browser setup workflows
