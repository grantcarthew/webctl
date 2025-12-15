# DR-008: Daemon REPL Interface

- Date: 2025-12-15
- Status: Accepted
- Category: CLI

## Problem

When `webctl start` runs, the daemon occupies the terminal but only outputs log messages. Users must open a second terminal to run commands like `webctl console`. This is inconvenient for interactive debugging workflows.

The terminal running the daemon could accept commands directly, providing a REPL (Read-Eval-Print Loop) interface for immediate interaction without requiring a separate terminal or IPC round-trip.

Additionally, the current command architecture tightly couples CLI commands to IPC transport. Commands cannot be executed directly within the daemon process, which limits flexibility for both REPL support and future features like `webctl serve` (daemon + web server).

## Decision

Add a REPL interface to the daemon that activates when stdin is a TTY. Refactor command architecture to use an Executor interface, allowing commands to run via IPC, TCP, or direct call.

REPL activation:

- When `webctl start` runs and stdin is a TTY, show `webctl> ` prompt
- Accept commands interactively using liner for readline functionality
- When stdin is not a TTY (pipe, background), daemon runs silently with IPC only

Executor interface:

```
Executor:
  Execute(request Request) (Response, error)

Implementations:
  IPCExecutor   - connects via Unix socket (default CLI)
  TCPExecutor   - connects via TCP (--host flag)
  DirectExecutor - calls daemon handler directly (REPL)
```

Commands use the Executor interface rather than directly calling IPC. This decouples command logic from transport.

REPL-specific commands:

| Command | Action |
|---------|--------|
| help, ? | Show available commands |
| exit, quit | Stop daemon and exit |
| history | List session command history |

All standard webctl commands work in REPL: console, network, clear, status, etc.

History:

- Up/down arrow keys navigate command history
- `history` command lists session commands
- History is not persisted between sessions

Output format:

- Same as CLI (JSON, respects --format flags)
- Pretty-printed when stdout is TTY

Prompt: `webctl> `

## Why

Single terminal workflow:

The daemon terminal is otherwise idle. Making it interactive eliminates the need for a second terminal during debugging sessions.

Executor abstraction:

Decoupling commands from transport enables:

- REPL support (direct execution)
- Remote debugging (TCP)
- Future features like `webctl serve` (daemon + web server with REPL)
- Easier testing (mock executor)

Cobra reuse:

Reusing Cobra for REPL command parsing provides:

- Full flag support (--head, --type, etc.) for free
- Consistent command syntax between CLI and REPL
- Help text generation
- Tab completion potential

TTY-only activation:

- Interactive terminal: show prompt, accept commands
- Non-TTY (pipe, background, CI): silent daemon, IPC only
- Avoids complexity of stdin exhaustion in pipelines

liner for readline:

- Simple, focused library
- Provides arrow key history navigation
- MIT licensed
- No heavy TUI framework needed

## Trade-offs

Accept:

- New dependency (liner)
- Refactoring commands to use Executor interface
- Two code paths for command execution (though unified by interface)
- REPL output interleaved with any daemon log messages

Gain:

- Single terminal debugging workflow
- Consistent command experience (CLI and REPL identical)
- Foundation for remote debugging and future features
- Cleaner architecture with transport abstraction

## Alternatives

Self-IPC approach:

REPL sends commands through its own Unix socket.

- Pro: Minimal code changes
- Pro: Identical code path for all commands
- Con: Unnecessary round-trip through socket
- Con: Feels like a hack
- Rejected: Direct execution is cleaner

Simple bufio.Scanner without readline:

- Pro: No external dependency
- Pro: Simpler implementation
- Con: No arrow key history navigation
- Con: Poor interactive UX
- Rejected: History navigation is essential for REPL usability

Dot-prefix for REPL commands:

Use `.help`, `.exit`, `.clear` for REPL meta-commands (Node.js style).

- Pro: Avoids potential command name conflicts
- Con: `clear` is not a conflict (clears buffers in both contexts)
- Con: Extra syntax to learn
- Rejected: No actual conflicts exist

Always-on REPL (ignore TTY check):

- Pro: Simpler detection logic
- Con: Breaks when stdin is a pipe or closed
- Con: Complicates headless/CI usage
- Rejected: TTY detection is standard practice

## Usage Examples

Interactive session:

```
$ webctl start
{"ok":true,"data":{"message":"daemon started","port":9222}}
webctl> console --tail 5
{
  "ok": true,
  "entries": [...],
  "count": 5
}
webctl> clear console
{"ok":true,"data":{"message":"console buffer cleared"}}
webctl> history
  1  console --tail 5
  2  clear console
webctl> exit
{"ok":true,"data":{"message":"daemon stopped"}}
$
```

Headless (no REPL):

```
$ webctl start --headless &
[1] 12345
$ webctl console
{"ok":true,"entries":[...],"count":42}
$ webctl stop
```

## Implementation Notes

Executor interface location: `internal/executor/`

```
internal/executor/
  executor.go      # interface definition
  ipc.go           # IPCExecutor (Unix socket)
  tcp.go           # TCPExecutor (remote host)
  direct.go        # DirectExecutor (REPL)
```

Command refactoring:

- Commands receive Executor via dependency injection or package-level variable
- RunE functions call `executor.Execute(request)` instead of `ipc.Dial()`
- Response handling and output formatting unchanged

REPL loop location: `internal/daemon/repl.go`

- Uses liner for input
- Parses input line as Cobra command
- Executes via DirectExecutor
- Handles special commands (help, exit, history)

liner setup:

- Create liner.State
- Configure history (in-memory only)
- Prompt: `webctl> `
- Handle Ctrl+C (cancel current line, not exit)
- Handle Ctrl+D (exit)

Help command output:

```
webctl> help

Commands:
  status      Show daemon status
  console     Show console log entries
  network     Show network requests
  clear       Clear event buffers
  screenshot  Capture screenshot
  html        Get page HTML
  eval        Evaluate JavaScript
  cookies     Get browser cookies

REPL:
  help, ?     Show this help
  history     Show command history
  exit, quit  Stop daemon and exit
```

## Updates

- 2025-12-15: Initial version
