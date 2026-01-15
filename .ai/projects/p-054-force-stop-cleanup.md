# p-054: Force Stop and Cleanup

- Status: In Progress
- Started: 2026-01-15

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

- [ ] `webctl stop --force` kills daemon process
- [ ] `webctl stop --force` kills browser on CDP port
- [ ] `webctl stop --force` removes stale socket file
- [ ] `webctl stop --force` removes stale pidfile
- [ ] `webctl start` shows hint on port-in-use error
- [ ] `webctl start` shows hint on daemon-already-running error
- [ ] `webctl serve` shows hint on relevant errors
- [ ] All existing tests pass
- [ ] Manual testing confirms cleanup works

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
3. Kill browser process on CDP port (lsof -i :PORT)
4. Remove socket and pidfile

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

## Notes

Created to resolve orphaned Chrome process blocking `webctl start` during p-037 testing.
