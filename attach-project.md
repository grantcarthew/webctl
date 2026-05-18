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

- `webctl start --attach` and `--attach VALUE` for connecting to an existing CDP endpoint (full value matrix in Issue 10).
- Loopback and remote hosts (IPv4, DNS, bracketed IPv6) per Issue 6.
- Auto-detect across ports 9222–9229 on the resolved host (Issue 8).
- `--port` composing with `--attach` when the attach value has no port (Issue 7).
- Connection validation before daemon start; clear error messages.
- Daemon refusing to close or kill the attached browser on `stop` or `stop --force` (Issue 5).

Out of Scope:

- `http(s)://` / `ws(s)://` URL schemes and TLS plumbing (Issue 6).
- Browser profile management, extension installation.
- Reconnection on disconnect, multi-browser attachment.
- Process-list scanning for non-default Chrome ports (Issue 8).

## Success Criteria

- [ ] All `--attach` value forms in Issue 10 parse and resolve correctly.
- [ ] Auto-detect picks the lowest-port responder in 9222–9229 (Issue 4).
- [ ] Daemon validates the CDP connection before declaring startup complete.
- [ ] `webctl stop` and `webctl stop --force` leave the attached browser running.
- [ ] WebSocketURL host rewrite makes remote endpoints usable when Chrome reports `127.0.0.1` (Issue 6).
- [ ] `--headless` combined with `--attach` is rejected at flag-validation time (Issue 7).
- [ ] Compatible with all existing webctl commands.

## Deliverables

- `internal/cli/start.go` — `--attach` flag with `NoOptDefVal="auto"`, `PreRunE` arg-merge, value parser, composition with `--port`, and `--headless` rejection.
- `internal/browser/attach.go` — `Attach(host string, port int)` plus parallel auto-detect helper using `FetchVersion`; WebSocketURL host rewrite helper.
- `internal/browser/browser.go` — `Browser` gains `host` and `attached` fields; `Close()` no-ops when `attached`; hardcoded `127.0.0.1` replaced with `b.host`.
- `internal/daemon/daemon.go` — `Config` gains `AttachHost` / `AttachPort`; `Run()` branches between `browser.Start()` and `browser.Attach()`; sidecar state file written at `ipc.DefaultStatePath()` (next to the PID file) with `{pid, host, port, attached}` on startup in BOTH launch and attach modes, and removed in the same defer as the PID file.
- `internal/cli/stop.go` — `forceCleanup` reads the sidecar state file and skips `findBrowserOnPort` / `killProcess` when `attached==true`.
- `internal/browser/attach_test.go` — Go unit tests for value parsing (full Issue 10 matrix), validation, auto-detect ranking, WebSocketURL rewrite.
- `internal/browser/integration_test.go` — extend with an attach integration test gated by `testing.Short()`: launch a real Chrome via `browser.Start`, call `browser.Attach` against the same port from a fresh struct, exercise `Targets`/`Version`, and confirm `Close` on the attached handle leaves the launched browser alive. Protects the WebSocketURL rewrite against regressions that unit tests cannot see.
- `internal/cli/start_test.go` (or extension of existing test file) — flag parsing tests for the merge logic and the `--headless` / `--port` composition rules.
- `scripts/test/cli/test-start-attach.sh` — shell CLI tests covering attach flag combinations, error messages, and the stop-does-not-kill-attached invariant.
- `docs/start.md` and a new `internal/cli/agent-help/attach.md` topic registered via `registerHelpTopics` in `internal/cli/help_agents.go`, documenting the new flag, the full Issue 10 value matrix, and the CDP-no-auth warning for remote use.
- `internal/ipc/protocol.go` and `internal/cli/format/text.go` — extend `StatusData` with `Attached` / `Host` / `Port` and surface a `Mode:` line in text output (Issue 11).

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

4. Auto-detect behavior when multiple browsers running (decision) — Resolved: lowest port wins.

   When `webctl start --attach` (no value) finds more than one CDP endpoint in the 9222–9229 scan, the daemon connects to the one on the lowest port number. The user can always override with an explicit `--attach=:PORT` when they want a specific browser.
   Resolution: Probe results are ranked by ascending port; the first responder is selected. Auto-detect failure (no candidates found) returns the existing "no browser found" error.

5. Browser lifecycle management on stop (decision) — Resolved: never close attached browser.

   webctl owns only the browsers it launches. When the daemon was started with --attach, neither `webctl stop` nor `webctl stop --force` may terminate the browser or touch its profile.
   Resolution:
   - Normal stop: the daemon's `defer d.browser.Close()` is already a no-op for attached browsers per Issue 3's resolution, so this path requires no further change.
   - Force stop: `forceCleanup` in `internal/cli/stop.go` must skip the `findBrowserOnPort` / `killProcess` step when the daemon was in attach mode. Because `--force` is the path used when the daemon is unresponsive, attach-mode state cannot be fetched via IPC at that point and must be persisted by the daemon at startup. Recommended storage: a JSON sidecar at `${cfg.PIDPath}.json` (same directory and permissions as the PID file, removed in the same `removePIDFile` defer path) containing `{pid, host, port, attached}`. Exposing the path via `ipc.DefaultStatePath()` alongside `DefaultPIDPath()` keeps the discovery story consistent. When the sidecar is missing — a pre-feature or crashed daemon — `stop --force` falls back to today's launch-mode behaviour (kill on port) for backwards compatibility; this fallback is safe because no pre-feature daemon was ever attached.

6. Remote CDP endpoint scope is under-specified (gap) — Resolved: remote hosts in scope; no schemes/TLS; WebSocketURL host rewrite required.

   Scope says "Work with both local and remote CDP endpoints" and that intent stands. Open questions raised under this issue are answered as follows.
   Accepted host forms in `--attach` values:
   - Empty (no host, e.g. `:9222` or bare port) — means loopback (127.0.0.1).
   - `localhost`, `127.0.0.1` — loopback.
   - Any DNS name or IPv4 address — remote host.
   - IPv6 in bracketed form: `[::1]:9222`, `[2001:db8::1]:9222`, or `[::1]` (no port → auto-detect).
   Rejected forms: `http://`, `https://`, `ws://`, `wss://` schemes. No TLS plumbing; users front their remote CDP with their own tunnel (SSH, VPN) if they need encryption.
   CDP authentication: CDP grants full browser control with no auth. This is a security footgun for any non-loopback host and must be called out prominently — not buried in a flag description. Required placements: a `WARNING:` block at the top of `docs/start.md`'s attach section, the same warning at the top of the `internal/cli/agent-help/` topic for attach, and a single-line warning emitted on stderr when `runStart` resolves a non-loopback host (suppressed in JSON mode). Users choosing to expose CDP remotely are making an explicit, informed decision.
   Port range: 9222–9229 for auto-detect (Issue 8). Explicit endpoints can use any port outside that range too — the range only constrains auto-detect.
   `Browser` gets `host` and `attached` fields (Issue 3 resolution). `Browser.Version()`, `Targets()`, and any other call currently hardcoded to `127.0.0.1` must use `b.host` instead.
   WebSocketURL host rewrite: Chrome's `/json/version` returns `webSocketDebuggerUrl` containing whatever host Chrome believes it is bound to. With `--remote-debugging-address=0.0.0.0` Chrome often emits `ws://127.0.0.1:PORT/...`, which is unreachable from a remote daemon. The attach path must parse the returned URL and substitute the `--attach` host before calling `cdp.Dial`. Use `net/url`'s `URL.Host = net.JoinHostPort(host, port)` so IPv6 brackets are correct.
   Pre-existing IPv6 URL construction bug: `FetchVersion` and `FetchTargets` in `internal/browser/target.go` build URLs via `fmt.Sprintf("http://%s:%d/...", host, port)`, which produces an invalid URL for any IPv6 host (the colons in the address collide with the port separator). Today this is latent because the only callers pass `"127.0.0.1"`. The attach implementation must switch both functions to `net.JoinHostPort(host, strconv.Itoa(port))` before any IPv6 form in Issue 10 can succeed.

7. --headless interaction with --attach is unspecified (decision) — Resolved: --headless errors; --port is allowed and composes.

   With --attach the daemon does not launch a browser, so --headless has no meaning. --port, by contrast, can legitimately supply the port when the --attach value does not (bare --attach, or value is a host only).
   Resolution:
   - `--headless` combined with `--attach` → error at flag-validation time. `runStart` checks `cmd.Flags().Changed("headless")` and rejects regardless of value (so `--headless=false` passed explicitly is still rejected, matching the principle that the user expressed an intent that does not apply).
   - `--port` combined with `--attach` is valid. Resolution rule for the value sources:
     - If the `--attach` value carries a port (`:PORT`, `PORT`, `HOST:PORT`, `[v6]:PORT`) AND `--port` is also passed → error ("port specified twice").
     - If the `--attach` value carries no port (bare `--attach`, or value is host-only like `localhost`, `somehost`, `[::1]`) → `--port` supplies it.
     - If neither carries a port → auto-detect across 9222–9229 on the resolved host.

8. Auto-detect strategy lacks concrete bounds (gap) — Resolved: 9222–9229, three timeout cases, no process scan.

   The Technical Approach previously said "Check default Chrome ports (9222, 9223, etc.)" and "Query process list for Chrome --remote-debugging-port" without specifying the port range, per-probe timeout, or whether process-list scanning was required.
   Resolution:
   - Port range for auto-detect: 9222–9229 (8 ports). Explicit endpoints can use any port.
   - Probe mechanism: HTTP `/json/version` via the existing `FetchVersion`. A bare TCP probe would only confirm something is listening; `/json/version` confirms it is a CDP endpoint and returns the WebSocketURL we need next anyway. Probes run in parallel.
   - Timeouts:
     - Loopback auto-detect: 200 ms per-probe context timeout. Worst-case total scan ~200 ms (parallel).
     - Remote auto-detect (`--attach=somehost` with no port): 2 s per-probe context timeout to absorb realistic LAN/WAN latency. Worst-case total scan ~2 s (parallel).
     - Explicit endpoint validation (loopback or remote, port specified): a single `FetchVersion` call with a 5 s context timeout.
   - Process-list scanning: out of scope. Adds platform-specific `ps`/`lsof` parsing for no benefit — port probing covers the entire valid range, and remote hosts have no `ps` accessible anyway.

9. AGENTS.md was out of sync with the active project (gap) — Resolved.

   AGENTS.md previously listed the wrong active project and pointed readers at a projects directory that no longer exists. AGENTS.md and `.ai/workflow.md` have been updated to reflect the current layout (project.md at the repository root) and to name this project as active.

10. --attach flag parsing pattern not specified (gap) — Resolved: NoOptDefVal + PreRunE arg-merge.

    pflag cannot natively support a flag that is both bare (`--attach`) and space-separated (`--attach 9222`). `NoOptDefVal` supports bare and `=value`; plain `StringVar` supports space-separated and `=value`; neither alone supports all three forms required.
    Resolution: declare `--attach` as `StringVar` with `Flag.NoOptDefVal = "auto"`, then add a `PreRunE` on `startCmd` that absorbs the next positional argument into the attach value when the parsed value is still `"auto"` and the next positional looks like an attach value. Concrete "looks like" rule, evaluated in order:
    - digits-only (`^[0-9]+$`) → port shorthand
    - starts with `:` followed by digits (`^:[0-9]+$`) → port shorthand
    - starts with `[` → IPv6 bracketed form, with or without `:PORT`
    - matches `^[A-Za-z0-9._-]+(:[0-9]+)?$` → hostname or hostname:port
    - anything else → leave as positional; Cobra rejects it (start accepts no positionals).
    Accepted forms (all parse to host + optional port):
    - `--attach` → loopback, auto-detect port range
    - `--attach=9222`, `--attach 9222`, `--attach=:9222`, `--attach :9222` → loopback, port 9222
    - `--attach=localhost`, `--attach localhost` → loopback, auto-detect port range
    - `--attach=localhost:9222`, `--attach localhost:9222` → loopback, port 9222
    - `--attach --port=9222`, `--attach --port 9222` → loopback, port 9222 (composes with `--port`, per Issue 7)
    - `--attach=somehost`, `--attach somehost` → remote host, auto-detect port range (2 s per probe per Issue 8)
    - `--attach=somehost:9222`, `--attach somehost:9222` → remote host, port 9222
    - `--attach=[::1]:9222`, `--attach [::1]:9222`, `--attach=[2001:db8::1]:9222` → IPv6, port 9222
    - `--attach=[::1]`, `--attach [::1]` → IPv6 loopback, auto-detect port range
    Value parser: bracketed IPv6 forms route through `net.SplitHostPort`. All-digits values are normalised to `127.0.0.1:PORT`. Values starting with `:` are normalised the same way. Values with no port and no bracket are treated as host-only.
    Port range enforcement: auto-detect always scans 9222–9229; explicit ports accept any value (Issue 8).
    Cobra Args validation order: Cobra validates `cobra.Command.Args` BEFORE `PreRunE` runs, so `startCmd` must leave `Args` unset (the default permits any positionals) or set it to `cobra.ArbitraryArgs`. Setting `cobra.NoArgs` would reject `webctl start --attach 9222` at Args validation before the merge can absorb the positional. The `PreRunE` itself must, after the absorb step, reject any remaining positionals so unrelated stray arguments still produce a clear error.

11. Status command does not surface attach mode (gap) — Resolved: extend StatusData.

    `ipc.StatusData` exposes `{Running, PID, ActiveSession, Sessions}`. An agent inspecting status cannot tell whether the daemon is in launch or attach mode, which host it is bound to, or whether `stop --force` will leave the browser running. Without this, agents have to remember their own startup behaviour to make safe decisions.
    Resolution: extend `ipc.StatusData` with `Attached bool`, `Host string`, and `Port int`. The daemon populates them from its own state (the same fields that go to the sidecar file in Issue 5). The text formatter in `internal/cli/format/text.go` prints `Mode: attached (HOST:PORT)` or `Mode: launched (HOST:PORT)` as a single line. JSON output carries the new fields unconditionally so agents have a stable shape. Update `internal/ipc/protocol_test.go` and `internal/cli/format/format_test.go` for the new fields.

## Technical Approach

Daemon config gains two fields:

```go
type Config struct {
    AttachHost string  // If non-empty, attach to this host:port instead of launching
    AttachPort int     // 0 means "auto-detect across the range"
    // ... existing fields (Port retained for launch mode)
}
```

`start.go` parses `--attach` plus `--port` into `AttachHost` and `AttachPort` per the matrix in Issue 10 and the composition rule in Issue 7. The implementer keeps `daemon.Config.Port` only meaningful when `AttachHost == ""`.

Browser struct (in `internal/browser/browser.go`) gains `host string` and `attached bool` fields. All call sites currently passing `"127.0.0.1"` literally (`Browser.Version`, `Browser.Targets`, `waitForCDP`) switch to `b.host`. `Browser.Close()` returns early when `b.attached == true` so the daemon's `defer d.browser.Close()` becomes a no-op for attached endpoints (Issue 3).

Connection logic in `daemon.Run`. Steps 1–2 are the mutually-exclusive browser-acquisition branch; steps 3–6 are common post-acquisition logic that apply to both modes.

1. Launch branch — if `cfg.AttachHost == ""`, call `browser.Start(LaunchOptions{...})` as today.
2. Attach branch — otherwise call `browser.Attach(host, port)` (new), which:
   - When `port != 0`: validates with one `FetchVersion` call, 5 s timeout (Issue 8).
   - When `port == 0`: probes 9222–9229 in parallel via `FetchVersion`, 200 ms per probe for loopback and 2 s per probe for remote hosts (Issue 8); lowest-port responder wins (Issue 4); error with the discovered candidate list if none respond.
   - Returns `&Browser{host, port, attached: true}` on success.
3. Sync `d.config.Port = b.Port()` immediately after acquisition (replaces today's line at daemon.go:242 and now covers both modes). This keeps the sidecar file (Issue 5), the status response (Issue 11), and any in-process consumers of `cfg.Port` aligned with the actual port in use after auto-detect.
4. Fetch the browser's CDP info via `d.browser.Version(ctx)` and connect via `version.WebSocketURL`. In attach mode only, rewrite the host portion of that URL to `cfg.AttachHost` before calling `cdp.Dial` to handle Chrome's `0.0.0.0` binding returning `ws://127.0.0.1:...` (Issue 6). In launch mode the URL is already correct because Chrome and the daemon share loopback.
5. Daemon writes its sidecar state file `{pid, host, port, attached}` at `ipc.DefaultStatePath()` on startup, regardless of mode — `attached: true` for the attach path, `attached: false` for the launch path. Removed in the same defer as the PID file. `webctl stop --force` reads it to decide whether to kill the browser on the configured port (Issue 5).
6. Rest of `Run` continues unchanged: `Target.setDiscoverTargets`, IPC server, REPL.

Command syntax (full matrix in Issue 10):

```
webctl start --attach                          # Loopback, auto-detect 9222-9229
webctl start --attach 9222                     # Loopback, port 9222
webctl start --attach :9222                    # Loopback, port 9222
webctl start --attach localhost:9222           # Loopback, port 9222
webctl start --attach somehost                 # Remote, auto-detect 9222-9229
webctl start --attach somehost:9222            # Remote, port 9222
webctl start --attach [::1]:9222               # IPv6 loopback, port 9222
webctl start --attach --port 9222              # Loopback, port 9222 (--port composes)
webctl start --attach somehost --port 9222     # Remote, port 9222
```

`--attach=VALUE` is equivalent to `--attach VALUE` in every form (Issue 10's PreRunE merge). `--headless` combined with `--attach` is rejected (Issue 7).

Error handling:

- Connection refused on explicit endpoint: "no CDP endpoint at HOST:PORT" with the address quoted.
- Auto-detect found nothing: "no Chrome with CDP enabled on HOST ports 9222-9229; launch Chrome with --remote-debugging-port".
- Invalid value form (e.g. `--attach=http://...`, `--attach=foo:bar:baz`): flag-validation error with the rejected value quoted.
- Port conflict between `--attach` value and `--port`: "port specified in both --attach and --port".

## Related Work

- Builds on the existing browser launch and daemon/IPC infrastructure
- Enables manual browser setup workflows
