# P-006: CLI Framework & Core Commands

- Status: Complete
- Started: 2025-12-12
- Completed: 2025-12-12

## Overview

Build the CLI interface using Cobra (or stdlib flags) and implement lifecycle commands: start, stop, status, clear.

## Goals

1. Clean CLI interface with subcommands
2. Working lifecycle commands
3. JSON output for all commands
4. Client-side IPC to communicate with daemon

## Scope

In Scope:

- CLI framework setup
- `start` command - launch daemon
- `stop` command - stop daemon
- `status` command - daemon status
- `clear` command - clear buffers
- JSON output formatting
- Error handling and exit codes

Out of Scope:

- Observation commands (P-007)
- Navigation/interaction commands (P-008)
- Remote access (`--host` flag)

## Success Criteria

- [x] `webctl start` launches daemon and browser
- [x] `webctl start --headless` launches headless
- [x] `webctl stop` cleanly shuts down daemon
- [x] `webctl status` returns JSON with daemon info
- [x] `webctl clear` clears event buffers
- [x] Proper exit codes (0 success, 1 error)
- [x] All output is valid JSON

## Deliverables

- `cmd/webctl/main.go` - entry point (calls cli.Execute())
- `internal/cli/root.go` - root command and JSON output helpers
- `internal/cli/start.go` - start command
- `internal/cli/stop.go` - stop command
- `internal/cli/status.go` - status command
- `internal/cli/clear.go` - clear command
- `internal/daemon/daemon.go` - shutdown handler added

## Technical Design

### Command Structure

```
webctl
├── start [--headless] [--port PORT]
├── stop
├── status
└── clear [console|network]
```

### CLI Framework Choice

**Cobra** - Standard for Go CLIs, good subcommand support, auto-generates help.

Minimal setup:
```go
var rootCmd = &cobra.Command{
    Use:   "webctl",
    Short: "Browser automation CLI for AI agents",
}

var startCmd = &cobra.Command{
    Use:   "start",
    Short: "Start daemon and browser",
    RunE:  runStart,
}
```

### Output Format

All commands output JSON to stdout:

```json
{"ok": true, "message": "Daemon started"}
{"ok": true, "status": "running", "url": "https://...", "title": "..."}
{"ok": false, "error": "Daemon not running"}
```

Errors go to stderr as JSON:
```json
{"error": "Failed to connect to daemon"}
```

### Start Command Flow

1. Check if daemon already running (PID file exists, process alive)
2. If running, error
3. Launch daemon in foreground (blocks) or background
4. Wait for socket to be ready
5. Output success

### Stop Command Flow

1. Connect to daemon socket
2. Send shutdown command
3. Wait for daemon to exit
4. Output success

### Status Command Flow

1. Check PID file
2. If no PID, output "not running"
3. If PID exists, connect to socket
4. Send status command
5. Output daemon status

### Client Package

```go
// internal/cli/client.go
type Client struct {
    socketPath string
}

func NewClient() (*Client, error)  // Finds socket path
func (c *Client) Send(cmd string, params map[string]any) (Response, error)
func (c *Client) Close() error
```

### Exit Codes

- 0: Success
- 1: Error (daemon not running, command failed, etc.)

## Dependencies

- P-005 (Daemon & IPC)
- `github.com/spf13/cobra`

## Testing Strategy

1. **Unit tests** - Command parsing, output formatting
2. **Integration tests** - Full start/stop/status cycle

## Notes

Keep the CLI simple. The daemon does the heavy lifting; CLI is just a thin client.

Consider adding `--json` flag in future for non-JSON output modes, but start JSON-only for v1.
