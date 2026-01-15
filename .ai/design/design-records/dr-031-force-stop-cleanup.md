# DR-031: Force Stop and Cleanup

- Date: 2026-01-15
- Status: Accepted
- Category: CLI

## Problem

When webctl processes (daemon, browser, serve) become orphaned or unresponsive, users have no built-in way to clean up. Common scenarios:

1. Daemon crashes but browser remains running on CDP port
2. Previous session not cleaned up properly (stale socket/pidfile)
3. Daemon becomes unresponsive and won't accept shutdown command
4. User runs `webctl start` and gets "port in use" with no clear remedy

Currently users must manually:
- Find process IDs using `lsof -i :9222`
- Kill processes manually with `kill`
- Remove stale socket/pid files

This creates friction and confusion, especially for users unfamiliar with process management.

## Decision

Add `--force` flag to the `stop` command that performs comprehensive cleanup:

```bash
webctl stop --force
```

Force stop terminates all webctl-related processes and cleans up all state:

1. Daemon process (from pidfile or process discovery)
2. Browser process on CDP port (default 9222 or configured)
3. Serve HTTP server (if running)
4. Stale socket file
5. Stale pidfile

Add contextual hints to error messages that suggest the remedy:

| Error | Hint |
|-------|------|
| Port in use (CDP) | `Hint: use 'webctl stop --force' to close existing processes` |
| Daemon already running | Same |
| IPC socket exists | Same |
| Server already running | Same |
| HTTP port in use | `Hint: use --port to specify a different port` |

## Why

Single command recovery:
- Users don't need to understand process management
- One command fixes any stuck state
- Consistent with "webctl manages its own lifecycle" philosophy

Contextual hints:
- Error messages become actionable
- Users learn the tool naturally
- Reduces support burden and documentation needs

The `--force` flag follows established CLI conventions (git, docker, systemctl) where force implies "override safety checks and do it anyway."

## Trade-offs

Accept:
- Force flag could kill processes user didn't intend to stop
- Process discovery may match wrong Chrome instances (mitigated by port check)
- Additional complexity in stop command

Gain:
- Self-contained recovery without external tools
- Clear path from error to resolution
- Reduced user frustration with orphaned processes
- Predictable cleanup behavior

## Alternatives

Separate `webctl kill` command:
- Pro: Clear distinction between graceful and forceful
- Pro: Follows Docker's stop/kill pattern
- Con: More commands to learn
- Con: `stop --force` is more discoverable
- Rejected: Flag on existing command is simpler

Interactive prompt on start failure:
- Pro: One-step recovery
- Pro: Convenient for interactive use
- Con: Requires TTY detection
- Con: Different behavior in scripts vs interactive
- Con: Risk of accidental kills
- Rejected: Hint message is safer and simpler

Automatic cleanup on start:
- Pro: Zero friction
- Con: Could kill important processes without consent
- Con: Violates principle of least surprise
- Rejected: Too dangerous

## Execution Flow

When `webctl stop --force` is executed:

1. Try graceful shutdown:
   - Send shutdown command to daemon via IPC
   - If successful, done

2. Kill daemon process:
   - Read PID from pidfile if exists
   - Send SIGTERM, wait briefly, then SIGKILL if needed
   - Remove pidfile

3. Kill browser on CDP port:
   - Find process listening on CDP port (default 9222)
   - Send SIGTERM, wait briefly, then SIGKILL if needed

4. Clean up files:
   - Remove IPC socket file
   - Remove any other stale state files

5. Report results:
   - Text mode: "OK" or list of what was cleaned
   - JSON mode: structured response with details

## Implementation Notes

Process discovery:
- Use pidfile first (most reliable)
- Fall back to `lsof -i :PORT` for orphaned browsers
- On macOS/Linux, use signals for termination

Port configuration:
- Use default CDP port (9222) unless --port specified
- Consider reading last-used port from state file

Error hint placement:
- Add hints in CLI layer, not daemon layer
- Hints only shown in text mode (not JSON)
- Use consistent format: `Hint: <action>`

Files affected:
- `internal/cli/stop.go`: Add --force flag and cleanup logic
- `internal/cli/start.go`: Add hint to port-in-use error
- `internal/cli/serve.go`: Add hints to relevant errors
