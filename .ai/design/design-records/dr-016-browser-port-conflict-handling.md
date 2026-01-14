# DR-016: Browser Port Conflict Handling

- Date: 2025-12-23
- Status: Accepted
- Category: Browser Launch

## Problem

When `webctl start` launches Chrome while another Chrome instance is already using the same CDP port, a second Chrome instance launches but cannot bind to the port. The daemon connects to the first Chrome instance (which successfully bound the port), but the user sees the second Chrome window. This causes confusing behavior:

- Navigation commands appear to not work (they control the invisible browser)
- Screenshots and HTML come from the wrong browser instance
- The visible browser does not respond to commands

This creates a poor user experience where the tool appears broken when multiple instances are accidentally started.

## Decision

Implement smart port conflict resolution with different behavior based on whether the port was explicitly specified:

Default port (user did not specify `--port`):
- Auto-find next available port starting from 9222
- Warn on stderr if a different port is used
- Continue with the found port

Explicit port (user specified `--port 9222`):
- Check if the port is available
- Error immediately with `ErrPortInUse` if port is busy
- Do not auto-select a different port

Implementation:
- Add `isPortAvailable()` to check TCP port binding
- Add `findFreePort()` to search for available port in range
- Modify `StartWithBinary()` to implement smart port selection
- Track actual port used in daemon config

## Why

This approach balances convenience with safety:

Convenience for default case:
- Users can run multiple `webctl start` commands without manually managing ports
- Common case (single instance) works seamlessly without warnings
- Multiple instances work automatically with clear warnings

Safety for explicit case:
- Users who specify a port expect that exact port to be used
- Silently using a different port would violate the principle of least surprise
- Explicit port requests likely have infrastructure dependencies (firewall rules, remote access)

The stderr warning provides visibility without breaking JSON output on stdout, maintaining the tool's scriptability.

## Trade-offs

Accept:
- Additional complexity in browser launch logic
- Port scanning could theoretically be slow (mitigated by small range)
- Warning messages on stderr when ports conflict
- Different behavior for default vs explicit ports

Gain:
- Prevents daemon from controlling the wrong browser instance
- Eliminates confusing UX where visible browser is unresponsive
- Allows multiple daemon instances for testing/development
- Clear error messages when explicit port is unavailable
- Maintains predictable behavior for scripting (explicit port always errors if unavailable)

## Alternatives

Always error on port conflict:
- Pro: Simple, predictable behavior
- Pro: Forces users to be explicit about port management
- Con: Poor UX for common case (forgot to stop previous instance)
- Con: Requires manual port management for multiple instances
- Rejected: Too strict for default port case

Always auto-find next available port:
- Pro: Maximum convenience, works in all cases
- Pro: Simplest implementation
- Con: Violates principle of least surprise when user specifies explicit port
- Con: Could silently ignore infrastructure requirements
- Rejected: Unsafe for explicit port specification

Kill existing process on port and reuse:
- Pro: Guarantees requested port is available
- Pro: Simple for user (no warnings)
- Con: Dangerous - could kill unrelated Chrome instances
- Con: Loses state from previous daemon
- Con: Could disrupt other users' sessions
- Rejected: Too destructive, unsafe

Return port in use error with suggestion:
- Pro: User decides how to handle conflict
- Pro: Clear error message
- Con: Requires user intervention for common mistake
- Con: Breaks workflow when accidentally starting multiple instances
- Rejected: Too much friction for default case

## Behavior Examples

Scenario 1 - Default port, no conflict:
```bash
webctl start
# Uses port 9222, no warning
```

Scenario 2 - Default port, 9222 in use:
```bash
webctl start
# Stderr: "Port 9222 in use, using port 9223 instead"
# Daemon uses port 9223
```

Scenario 3 - Explicit port, available:
```bash
webctl start --port 9500
# Uses port 9500
```

Scenario 4 - Explicit port, in use:
```bash
webctl start --port 9222
# Error: "port is already in use: 9222"
# Exit code: non-zero
```

## Implementation Notes

Port availability check:
- Use `net.Listen("tcp", ...)` to test if port can be bound
- Close listener immediately after check
- Race condition possible but acceptable (port could be taken between check and use)

Port search range:
- Start at requested port (default 9222)
- Check up to 10 ports (9222-9231)
- Error if no free port found in range

Error types:
- `ErrPortInUse`: Explicit port is unavailable
- `ErrNoFreePort`: No available ports in search range (default port only)

Modified files:
- `internal/browser/browser.go`: Port logic and helpers
- `internal/daemon/daemon.go`: Config update with actual port
- Tests updated to handle port auto-selection

## Related Issues

- BUG-004: Multiple Chrome instances on same port causes daemon to control wrong browser
