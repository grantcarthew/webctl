# DR-001: Core Architecture

- Date: 2025-12-11
- Status: Accepted
- Category: Architecture

## Problem

AI agents debugging web applications need access to DevTools data (console logs, network requests, JS errors) that standard web fetching tools cannot access. CDP events are ephemeral - you must be listening when they occur. There is no `Runtime.getConsoleHistory()` method in CDP.

Existing tools have gaps:

| Tool                | Console Logs | Daemon | Single Binary | Cross-Platform  |
| ------------------- | ------------ | ------ | ------------- | --------------- |
| browser-console-tap | Yes          | No     | No            | No              |
| playd               | No           | Yes    | No            | No              |
| chrome-cli          | No           | No     | Yes           | No (macOS only) |

None solve the full problem: console capture + daemon + single binary + cross-platform.

## Decision

Build webctl as a CLI tool for browser automation and debugging, designed for AI agents:

- Daemon + stateless command model
- Single Go binary
- CDP-based browser control (minimal implementation from scratch)
- JSON output for agent consumption
- Unix socket IPC (local), TCP (remote)

## Why

- Single binary deployment simplifies distribution
- Daemon architecture is required because CDP console events are fire-and-forget
- A persistent process can subscribe to events before they occur and buffer them
- Go provides excellent cross-compilation and produces self-contained binaries
- JSON output is trivially parseable by agents
- Unix socket is fast, file-based (automatic cleanup), and avoids port conflicts

## Trade-offs

Accept:

- Daemon complexity (lifecycle management, IPC)
- Memory usage for event buffering
- Writing CDP implementation from scratch

Gain:

- Access to ephemeral CDP events (console, network, exceptions)
- Single binary distribution with zero dependencies
- Cross-platform support
- Agent-friendly JSON interface
- Full control over codebase, no third-party dependency maintenance

## Alternatives

Use rod as a Go module dependency:

- Pro: Full-featured, well-tested, MIT licensed
- Pro: Fastest path to working prototype
- Con: Pulls entire dependency tree
- Con: Most features unused (PDF, device emulation, complex selectors)
- Con: Dependency version management overhead
- Rejected: Too heavy for focused use case

Extract CDP code from rod:

- Pro: Proven, tested code
- Pro: Smaller than full rod dependency
- Con: Still maintaining someone else's code patterns
- Con: Harder to understand when debugging
- Rejected: Writing from scratch is cleaner for AI-agent development

Use chromedp:

- Pro: Mature CDP library
- Con: Different API style, still a dependency
- Rejected: Same dependency concerns as rod

Write a Node.js tool:

- Pro: Native CDP support, large ecosystem
- Con: Requires Node.js runtime
- Con: Distribution complexity
- Rejected: Single binary requirement

---

## CDP Implementation Strategy

Write minimal CDP implementation from scratch:

- Reference rod and devtools-protocol for patterns and correctness
- No copied code or external dependencies
- AI agents can generate the boilerplate efficiently
- Full understanding and control of the codebase

Required capabilities:

- Browser detection and launch (find Chrome, spawn with flags)
- CDP WebSocket connection management
- Event subscription (Runtime.consoleAPICalled, Runtime.exceptionThrown, Network.*)
- Command execution (~15 methods)
- Event buffering in memory

---

## Command Set

webctl exposes 18 commands across 5 categories:

### Observation Commands

| Command    | Purpose                                             |
| ---------- | --------------------------------------------------- |
| console    | Query buffered console logs and uncaught exceptions |
| network    | Query buffered network requests with full bodies    |
| screenshot | Capture current viewport or full page               |
| html       | Get DOM content (full page or selector)             |
| eval       | Run arbitrary JS and return result                  |
| cookies    | Get or set browser cookies                          |

### Navigation Commands

| Command  | Purpose                     |
| -------- | --------------------------- |
| navigate | Go to URL                   |
| reload   | Refresh current page        |
| back     | Navigate back in history    |
| forward  | Navigate forward in history |

### Interaction Commands

| Command | Purpose                       |
| ------- | ----------------------------- |
| click   | Click element by selector     |
| type    | Focus element and input text  |
| select  | Choose dropdown option        |
| scroll  | Scroll to element or position |

### Synchronisation Commands

| Command  | Purpose                                              |
| -------- | ---------------------------------------------------- |
| wait-for | Wait for selector, network idle, or custom condition |

### Lifecycle Commands

| Command | Purpose                                        |
| ------- | ---------------------------------------------- |
| start   | Launch daemon and browser                      |
| stop    | Clean shutdown of daemon and browser           |
| status  | Daemon health, current URL, page title         |
| clear   | Clear event buffers (console, network, or all) |

---

## IPC Protocol

webctl follows the XDG Base Directory Specification for file locations:

| Type | Variable | Default | Path |
|------|----------|---------|------|
| Runtime (socket, PID) | `XDG_RUNTIME_DIR` | `/run/user/<uid>` | `$XDG_RUNTIME_DIR/webctl/webctl.sock` |
| Config | `XDG_CONFIG_HOME` | `~/.config` | `$XDG_CONFIG_HOME/webctl/config.toml` |
| State/logs | `XDG_STATE_HOME` | `~/.local/state` | `$XDG_STATE_HOME/webctl/` |

Fallback when `XDG_RUNTIME_DIR` is unset: `/tmp/webctl-<uid>/webctl.sock`

Remote access uses TCP with `--listen :9444`.

JSON request/response format:

```json
{"cmd": "console"}
{"ok": true, "logs": [{"level": "error", "text": "...", "timestamp": 1702000000}]}

{"cmd": "click", "selector": ".button"}
{"ok": true}

{"cmd": "navigate", "url": "https://localhost:3000"}
{"ok": true}

{"cmd": "click", "selector": ".missing"}
{"ok": false, "error": "element not found: .missing"}
```

### Response Format

All responses include an `ok` field:

- `ok: true` - Command succeeded. Additional data fields depend on command.
- `ok: false` - Command failed. `error` field contains the error message.

---

## Event Buffering

### Buffer Configuration

| Buffer  | Default Size   | Contents                                    |
| ------- | -------------- | ------------------------------------------- |
| Console | 10,000 entries | Logs, warnings, errors, uncaught exceptions |
| Network | 10,000 entries | Full request/response including bodies      |

### Buffer Policy

Ring buffer with oldest-first eviction:

- When buffer is full, oldest entries are silently evicted
- Buffer size configurable via `--buffer-size` flag on start
- Queries return all buffered entries (non-destructive by default)

### Clear Command

```bash
webctl clear           # Clear all buffers
webctl clear console   # Clear console buffer only
webctl clear network   # Clear network buffer only
```

---

## Browser Lifecycle

Browser lifetime is tied to daemon:

- `webctl start` launches daemon and browser
- Browser closes when daemon exits
- Daemon exits when browser closes unexpectedly
- Single session only (no multi-browser support in v1)

### Timeout Behaviour

- No default timeout
- Optional `--timeout <seconds>` flag on start
- Daemon runs until explicit `stop` or timeout

---

## Remote Access

For cross-machine debugging:

```bash
# Machine A (browser host)
webctl start --listen :9444
# Warning displayed: no auth, network accessible

# Machine B (agent)
webctl --host 192.168.1.50:9444 console
```

Optional token authentication:

```bash
webctl start --listen :9444 --token mysecret
webctl --host 192.168.1.50:9444 --token mysecret console
```

---

## Usage Examples

Basic debugging workflow:

```bash
# Terminal 1 (or backgrounded)
webctl start --headless &

# Agent commands
webctl navigate https://localhost:3000
webctl console                             # Check for JS errors
# ... agent fixes code ...
webctl reload
webctl console                             # Verify fix
webctl stop
```

Debugging a specific interaction:

```bash
webctl start
webctl navigate https://myapp.local
webctl clear                               # Start fresh
webctl click "#submit-button"
webctl console                             # See only logs from this action
webctl network                             # See API calls from this action
webctl screenshot > debug.png
webctl stop
```
