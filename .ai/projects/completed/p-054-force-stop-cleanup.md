# p-054: Force Stop and Cleanup

- Status: Complete
- Started: 2026-01-15
- Completed: 2026-01-15

## Overview

Implement comprehensive cleanup functionality for webctl via `stop --force` flag. This resolves orphaned process issues and provides clear error recovery paths for users.

## Goals

1. Add `--force` flag to stop command
2. Implement comprehensive cleanup (daemon, browser, serve, stale files)
3. Add contextual hints to error messages
4. Enable users to recover from any stuck state with single command

## Scope

In Scope:
- `--force` flag implementation on stop command
- Process discovery and termination (daemon, browser)
- Stale file cleanup (socket, pidfile)
- Error message hints in start and serve commands

Out of Scope:
- Remote process management
- Interactive prompts (rejected in DR-031)
- Automatic cleanup on start (rejected in DR-031)

## Success Criteria

- [x] `webctl stop --force` kills daemon process
- [x] `webctl stop --force` kills browser on CDP port
- [x] `webctl stop --force` removes stale socket file
- [x] `webctl stop --force` removes stale pidfile
- [x] `webctl start` shows hint on port-in-use error
- [x] `webctl start` shows hint on daemon-already-running error
- [x] `webctl serve` shows hint on relevant errors
- [x] All existing tests pass
- [x] Manual testing confirms cleanup works

## Deliverables

- Updated `internal/cli/stop.go` with --force flag
- Updated `internal/cli/start.go` with error hints
- Updated `internal/cli/serve.go` with error hints
- Process discovery utility (if needed)
- Updated command help text

## Technical Approach

Force stop execution:
1. Try graceful shutdown via IPC
2. If fails or --force: kill daemon from pidfile
3. Kill browser process on CDP port (`--port` flag, default 9222)
4. Remove socket and pidfile

Flags:
- `--force`: Enable forceful cleanup
- `--port`: CDP port for browser process discovery (default 9222)

Error hints:
- Add after outputError() calls for relevant errors
- Text mode only (not JSON)
- Format: `Hint: use 'webctl stop --force' to close existing processes`

## Code References

- internal/cli/stop.go - stop command implementation
- internal/cli/start.go - start command (add hints)
- internal/cli/serve.go - serve command (add hints)
- internal/daemon/daemon.go - daemon startup errors
- internal/browser/browser.go - ErrPortInUse definition

## Design Record

- DR-031: Force Stop and Cleanup

## Current State

### stop.go (56 lines)
- Simple IPC-based graceful shutdown via `shutdown` command
- No flags, no force capability
- Returns "daemon stopped" (JSON) or "OK" (text)

### start.go (76 lines)
- Line 33-35: Checks `execFactory.IsDaemonRunning()` and returns "daemon is already running"
- Line 70-72: `daemon.Run()` can fail with port-in-use wrapped as "failed to start browser: port is already in use: 9222"
- Hints needed at both error points

### serve.go (317 lines)
- Line 128: "failed to start daemon: %v" in `runServeWithDaemon()` - may contain port-in-use error
- Line 166: `resp.Error` from daemon IPC - may be "server already running"
- Line 281: `resp.Error` from daemon IPC - may be "server already running"
- "server already running" error originates at `internal/daemon/handlers_serve.go:41`
- Hints needed at lines 128, 166, and 281 for daemon/server errors

### daemon.go
- Line 518-526: `writePIDFile()` writes PID to `config.PIDPath`
- Line 529-531: `removePIDFile()` cleans up on shutdown
- Paths from `ipc.DefaultSocketPath()` and `ipc.DefaultPIDPath()`

### ipc/server.go and ipc/client.go
- `DefaultSocketPath()`: `$XDG_RUNTIME_DIR/webctl/webctl.sock` or `/tmp/webctl-<uid>/webctl.sock`
- `DefaultPIDPath()`: `$XDG_RUNTIME_DIR/webctl/webctl.pid` or `/tmp/webctl-<uid>/webctl.pid`
- `IsDaemonRunning()`: checks socket exists and can connect

### browser.go (226 lines)
- Line 31: `ErrPortInUse = errors.New("port is already in use")`
- Line 85: Returns `fmt.Errorf("%w: %d", ErrPortInUse, port)`

### Implementation Approach
1. Add `outputHint()` helper in root.go for text-mode hints (writes to stderr, skipped in JSON mode)
2. Add `--force` flag and `--port` flag to stop command
3. Implement force cleanup sequence: graceful IPC shutdown, kill daemon from pidfile, `lsof -i :PORT` for browser, remove socket/pidfile
4. Add hint calls in start.go after lines 34 and 71
5. Add hint calls in serve.go after lines 128, 166, and 281 (check error contains port/daemon/server keywords)

## Decision Points

1. **CDP port for force stop**: Should `webctl stop --force` accept a `--port` flag to specify which CDP port to check for browser processes?
   - A) Yes, add `--port` flag (default 9222) for explicit control ✓ **Selected**
   - B) No, always use default port 9222 only
   - C) Read last-used port from a state file (more complex)

## Notes

Created to resolve orphaned Chrome process blocking `webctl start` during p-037 testing.
