# DR-002: CLI Browser Commands

- Date: 2025-12-11
- Status: Accepted
- Category: CLI

## Problem

webctl needs a clear command interface for browser and daemon lifecycle. There are multiple scenarios:

1. Simple local use: launch browser and daemon together
2. Attach to existing browser: user launched Chrome manually or via another tool
3. Remote access: daemon on server, CLI connects over network
4. Browser-only launch: user wants Chrome with CDP enabled, no daemon

These scenarios involve two distinct ports that can cause confusion:

- CDP port: Chrome's DevTools Protocol WebSocket (browser)
- Daemon port: webctl's IPC for remote CLI access

The interface must keep these concepts separate and unambiguous.

## Decision

Two distinct commands for browser and daemon lifecycle:

```
webctl browser [--headless] [--port 9222]
webctl start [--headless] [--attach :9222] [--listen :9444]
```

Command: webctl browser

Launches Chrome with CDP enabled, then exits. No daemon.

| Flag | Default | Description |
|------|---------|-------------|
| --headless | false | Run browser in headless mode |
| --port | 9222 | CDP port for remote debugging |

Behaviour:

- Spawns Chrome with `--remote-debugging-port=PORT`
- Prints CDP WebSocket URL to stdout
- Exits (browser continues running)
- No daemon started

Use case: Manual browser interaction, or preparation for `webctl start --attach`.

Command: webctl start

Launches the daemon with event buffering.

| Flag | Default | Description |
|------|---------|-------------|
| --headless | false | Run browser in headless mode (ignored if --attach) |
| --attach | - | Connect to existing CDP endpoint instead of launching browser |
| --listen | - | Expose daemon on TCP port for remote CLI access |

Behaviour without --attach:

- Launches browser with CDP
- Starts daemon
- Daemon connects to browser via CDP
- IPC via Unix socket (default) or TCP (if --listen)

Behaviour with --attach:

- Does not launch browser
- Starts daemon
- Daemon connects to specified CDP endpoint
- --headless flag is ignored

## Why

Separation of concerns:

- `webctl browser` deals only with browser launch (one concept: CDP port)
- `webctl start` deals with daemon lifecycle (two concepts: attach to CDP, listen for CLI)

Clear flag semantics:

- `--port` only appears on `browser` command (CDP port)
- `--attach` specifies CDP endpoint to connect to (not launch)
- `--listen` specifies daemon's TCP port for remote CLI access

No ambiguous `--port` flag that could mean either CDP or daemon port.

## Trade-offs

Accept:

- Two commands instead of one (slightly more to learn)
- Users must understand browser vs daemon distinction

Gain:

- No port confusion between CDP and daemon
- Each command has single responsibility
- Flags are unambiguous
- Supports all use cases cleanly

## Alternatives

Single command with mode flags:

```
webctl start --browser-only
webctl start --attach :9222
```

- Pro: Fewer commands
- Con: `--browser-only` is awkward (implies other modes)
- Con: Mixing browser-only and daemon modes in one command
- Rejected: Separation is cleaner

Implicit attach (auto-detect running browser):

- Pro: Magic "just works" experience
- Con: Unpredictable when multiple browsers running
- Con: Harder to debug connection issues
- Rejected: Explicit is better than implicit

## Usage Examples

Local development (launch everything):

```bash
webctl start
webctl navigate https://localhost:3000
webctl console
webctl stop
```

Headless on server with remote access:

```bash
# On server
webctl start --headless --listen :9444

# On local machine
webctl --host server.example.com:9444 console
```

Attach to manually-launched browser:

```bash
webctl browser --port 9222
# ... interact with browser manually ...
webctl start --attach :9222
webctl console
```

Attach to browser launched by another tool (e.g., snag):

```bash
snag --open-browser  # Launches Chrome with CDP on 9222
webctl start --attach :9222
webctl network
```

## Browser Detection

When launching browser (via `webctl browser` or `webctl start` without `--attach`):

1. Check `WEBCTL_CHROME` environment variable
2. Search common paths per platform (LookPath, similar to rod)
3. If not found, error with detailed message listing searched paths

No auto-download of browser binaries. User must have Chrome/Chromium installed.

## Updates

- 2026-01-25: Implementation note - The full design described above was not implemented. Current implementation (as of v0.1):
  - `webctl browser` command does not exist
  - `webctl start` only has `--headless` and `--port` flags
  - No `--attach` or `--listen` flags implemented
  - Simple use case only: start launches both daemon and browser locally
  - Remote access and attach scenarios remain unimplemented (future enhancement)
