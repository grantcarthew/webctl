# webctl Agent Guide

Browser automation CLI for AI agents, written in Go. A persistent daemon launches Chrome/Chromium and buffers Chrome DevTools Protocol (CDP) events so stateless CLI commands can query them later.

This file is the entry point for AI coding agents. For a human-facing summary see `README.md`. For per-command human docs see `docs/`. For AI working files (projects, roles, research) see `.ai/`.

## Architecture

Daemon plus stateless command model.

- `webctl start` launches the browser, opens a CDP WebSocket, buffers events (console, network, etc.), and serves IPC over a Unix socket.
- All other commands (`console`, `network`, `click`, `navigate`, `reload`, ...) are short-lived processes that talk to the daemon via IPC and exit.
- `webctl stop` cleanly shuts the daemon and the browser it owns; `webctl stop --force` reaps orphaned processes (kills any browser bound to the configured CDP port via `lsof`).
- `webctl serve` is an optional dev HTTP server (static or reverse proxy) integrated with hot reload via CDP `Page.reload`.

Why a daemon: CDP events are ephemeral. The listener must be attached when they fire, otherwise they are lost. The daemon solves this by holding the CDP connection across many CLI invocations.

## Repository Layout

```
cmd/webctl          Main entry; thin wrapper over internal/cli.Execute
internal/browser    Chrome launch, detection, CDP target/version HTTP, attach (planned)
internal/cdp        CDP client, connection, message types
internal/cli        Cobra commands, output helpers, formatters, agent-help topics
internal/daemon     Daemon, event handlers, IPC server, REPL, session, buffers
internal/executor   Direct (in-process) and IPC executors for command dispatch
internal/ipc        Unix-socket protocol, client, server
internal/server     Dev HTTP server: static, proxy, file watcher
internal/cssformat  CSS pretty-printer
internal/htmlformat HTML pretty-printer
scripts             test-runner helpers, bash modules, CLI/interactive test scripts
testdata            Static fixtures and pages used by tests and the dev server
docs                Human-facing per-command documentation
.ai                 AI working files (roles, context, research, project docs)
```

Directory depth in `internal/cli/agent-help/` holds the markdown topics surfaced via `webctl help <topic>`.

## Build

Pure Go, no cgo. Module path: `github.com/grantcarthew/webctl`. Go 1.25.5 minimum (see `go.mod`).

```bash
go build -o webctl ./cmd/webctl
```

The repo-root `webctl` binary is gitignored. Tests that need it call `require_webctl` in `scripts/bash_modules/setup.sh`, which builds it on demand.

Cross-platform note: Linux and macOS are first-class. `webctl stop --force` and Chrome auto-detection use `lsof` and `ps`.

## Testing

Use the `./test-runner` script at the repo root. It is the unified entry point and wraps Go and bash test suites.

| Command | What it does |
|---------|--------------|
| `./test-runner go unit` | `go test -short ./internal/...` (no Chrome required) |
| `./test-runner go integration` | `go test -run Integration ./internal/...` (launches real Chrome) |
| `./test-runner go race` | `go test -short -race ./internal/...` |
| `./test-runner go cover` | Coverage report; writes `coverage.out` |
| `./test-runner go bench` | Benchmarks |
| `./test-runner cli [name]` | Bash CLI test suite under `scripts/test/cli/` |
| `./test-runner interactive [name]` | Manual/interactive tests in `scripts/interactive/` |
| `./test-runner lint` | `go vet` plus `staticcheck` if installed |
| `./test-runner fmt` | `gofmt -l .` (excluding `vendor/`) |
| `./test-runner ci` | All Go tests, lint, then CLI bash suite |
| `./test-runner quick` | Unit tests plus lint, fast feedback |

Direct equivalents (use these if `./test-runner` is unavailable):

```bash
go test -short ./internal/...
go test -run Integration ./internal/...
go test -coverprofile=coverage.out ./internal/...
go vet ./internal/...
gofmt -l .
```

Integration tests are gated by `testing.Short()`; they will spawn Chrome and need a working browser binary. Skip them with `-short` when running on machines without a display server unless headless Chrome is installed.

Bash tests source modules from `scripts/bash_modules/` (`setup.sh`, `assertions.sh`, `test-framework.sh`). They expect to find or build `./webctl` at the repo root, then start and stop a daemon as needed. They also start a local test server on `${TEST_SERVER_PORT:-8888}` serving `testdata/`.

Coverage baseline (2025-12-17): 65.6% overall. Detailed gaps and recommended approaches live in `docs/testing.md`. Lowest-coverage area is `internal/browser` launch/lifecycle (35.6%).

## Code Style

- Format with `gofmt`. CI checks via `./test-runner fmt`; fix with `gofmt -w .`.
- Static analysis: `go vet` always; `staticcheck` if installed (`./test-runner lint`).
- Pure Go only. Do not introduce cgo or C dependencies. Prefer the standard library and the existing dependency set in `go.mod`.
- Idiomatic Go: explicit error returns, no panics in library code, lower-case unexported names, exported identifiers documented with a comment beginning with the name.
- Avoid speculative abstraction. Prefer adding a concrete handler or command over a new framework.

## CLI Conventions

The CLI lives in `internal/cli`. Add new commands by creating `<name>.go`, defining a `*cobra.Command`, and registering it in `init()` via `rootCmd.AddCommand`. To group it in `webctl --help`, add an entry to `commandGroups` in `internal/cli/root.go`.

Output helpers in `internal/cli/root.go` (use these, do not write to stdout/stderr directly):

| Helper | Purpose |
|--------|---------|
| `outputSuccess(data)` | Stdout. Text mode prints `OK` when `data` is nil; JSON mode wraps in `{"ok": true, "data": ...}`. |
| `outputError(msg)` | Stderr with `Error:` prefix; JSON mode wraps in `{"ok": false, "error": ...}`. Returns `printedError` so `main.go` does not double-print. |
| `outputNotice(msg)` | Stderr informational message that still exits non-zero. |
| `outputHint(msg)` | Stderr `Hint:` line, suppressed in JSON mode. |
| `outputJSON(w, data)` | Pretty-prints when `w` is a TTY, compact otherwise. |

Global flags (`--debug`, `--json`, `--no-color`) bind to the package vars `Debug`, `JSONOutput`, `NoColor`. The REPL resets these between commands; per-command flags should be defined as locals or reset in `Execute`-style helpers.

Debug logging: use `debugf`, `debugRequest`, `debugResponse`, `debugFilter`, `debugFile`, `debugTiming`, `debugParam` rather than ad-hoc `fmt.Fprintln`. They no-op when `Debug` is false and use a consistent `[DEBUG] [HH:MM:SS.mmm] [CATEGORY]` prefix.

JSON output is the primary interface for AI agents. Text output is a convenience for humans on a TTY; many commands implement TTY-only formatters that are not exercised in JSON mode.

Command abbreviation: `Execute` expands a unique prefix (e.g. `webctl nav` to `navigate`). Keep top-level command names unambiguous when adding new ones.

## Daemon and IPC

- `internal/daemon/daemon.go` defines `Config` inline (no separate `config.go`) and runs the daemon. `Run` launches the browser via `internal/browser`, connects to CDP, starts the IPC server, and optionally enters a REPL.
- IPC payloads are JSON over a Unix socket (`internal/ipc/protocol.go`). Add a new command by extending the request handler dispatch in `internal/daemon/handlers_*.go`.
- The `CommandExecutor` field on `daemon.Config` lets the REPL execute commands in-process via `executor.Direct` instead of round-tripping through IPC.
- Browser lifecycle: `internal/browser/browser.go` `Browser.Close()` always SIGTERMs the process and removes the temp data dir. The attach-mode work in progress (see Active Project) introduces an `attached` flag that no-ops Close for externally-launched browsers.

## Active Project

The project document at the repo root drives current work:

- `attach-project.md` (current): implement `webctl start --attach` to connect to an existing CDP endpoint instead of launching a new browser.

Project documents follow `.ai/docs/project-writing-guide.md`. Status, Goals, Scope, Success Criteria, and Deliverables are required sections. Capture decisions inline rather than in source comments. When the active project changes, rename the root-level document and update this section.

## Documentation Conventions

Two audiences, two styles.

- Human-facing docs (`README.md`, `docs/*.md`): full CommonMark, including bold and emojis where useful.
- Agent-facing docs (this file, `.ai/`, `internal/cli/agent-help/`): token-efficient markdown. Avoid bold, italics, horizontal rules, emojis, HTML comments, image embeds, multi-blank-line gaps, task lists, nesting beyond 3 levels, headings beyond `###`, and directory trees beyond depth 3. Prefer headings, lists, tables, and `NOTE:`/`WARNING:`/`TIP:` callout prefixes without bold.

The agent-help system surfaces the `internal/cli/agent-help/*.md` topics through `webctl help <topic>` (see `internal/cli/help_agents.go`). When adding a topic file there, register it with `registerHelpTopics` so it appears under `webctl --help`.

## Common Pitfalls

- Stale daemon: a previous run left a socket or PID file. `webctl status` reports state; `webctl stop --force` reaps the daemon and any browser still bound to the CDP port.
- Port already in use: default CDP port is 9222. The launcher does not retry on conflict; `webctl serve` does cycle dev ports (3000, 8080, 8000, 5000, 4000) before falling back to OS-assigned.
- Background daemon in tests: `webctl start` blocks. Bash tests run it with `&` and poll `is_daemon_running` (see `scripts/bash_modules/setup.sh`). Do the same in any new shell automation.
- REPL flag bleed: REPL invocations share the global `Debug`, `JSONOutput`, `NoColor`. `ExecuteArgs` resets per-command flags after each call; if you add a new global persistent flag, reset it there too.
- Output helpers vs raw printing: use `outputSuccess`/`outputError`/`outputNotice` so JSON mode and the `printedError` no-double-print contract keep working.
- `lsof` and `ps` dependence: `webctl stop --force` and Chrome auto-detect call out to these. They are present on Linux and macOS by default; expect breakage elsewhere.
- Integration tests need Chrome: `./test-runner go unit` is the safe default in CI without a browser. Use `go integration` only where Chrome is available.
- CDP events are ephemeral: do not assume a fresh `webctl console` after the daemon was restarted contains historical logs. Buffers reset on daemon restart and on `webctl clear`.

## Pull Request Notes

- Run `./test-runner quick` before pushing, then `./test-runner ci` for anything that touches command surface, daemon, or IPC.
- Update `docs/testing.md` if you move coverage materially up or down for a package, and update the relevant `docs/<command>.md` when adding or changing a command.
- Do not add `Co-Authored-By` trailers to commit messages.
