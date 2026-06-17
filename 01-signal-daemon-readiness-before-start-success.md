# Signal daemon readiness before reporting start success

Source: pre-commit review on 2026-06-15
Severity: info
Category: Observability / Correctness
Location: internal/cli/start.go:102-119, internal/cli/serve.go:114-136, internal/daemon/daemon.go:212-308

## Goal

Make `webctl start` report success only once the daemon is serving IPC — browser launched, CDP connected, IPC socket accepting — so a failed start never emits a success line ahead of its error. This keeps the JSON output contract trustworthy for AI agents, which are the primary consumers of webctl output.

The same readiness signal also replaces the fixed 500ms sleep that `webctl serve` uses to guess the daemon is up before it issues commands. Both are the same missing primitive; building it once and routing both callers through it is the principled fix (see Scope).

The readiness bar is IPC-serving, not full interactivity. At the moment success fires, the first page target may still be attaching (target attachment is asynchronous), so `sessions.Count()` can briefly be zero. The success line means the daemon is up and accepting commands, not that a browser command issued in the same instant will find an attached session. Waiting for the first session before signalling readiness is a separate concern and out of scope here (see Scope).

## Scope

In scope: a single daemon-readiness signal emitted from `daemon.Run` once IPC is serving, and both of its CLI consumers. `runStart` uses it to gate startup success/failure reporting. `runServeWithDaemon` uses it to gate command issuance, replacing the fixed `time.Sleep(500ms)` it currently uses to guess that the daemon is up. The daemon keeps running in the foreground and blocking until shutdown.

The two callers are the same bug with two symptoms: neither has a real readiness signal, so each invents an unreliable proxy (success printed before `Run` starts; a timed sleep before issuing commands). Introducing the signal once and routing both callers through it fixes the root cause rather than one symptom.

Out of scope: changing the daemon lifecycle model (foreground blocking, background-and-poll automation), the profile-resolution logic, the browser launch path, the fail-fast `--system-profile` error itself, and extending readiness to await the first attached page session (the bar is IPC-serving, not session-attached).

## Current State

`runStart` (internal/cli/start.go) prints the success message before it runs the daemon:

```
_ = outputSuccess(...)            // text: "OK"; JSON: {"ok":true,"data":{"message":"daemon starting","port":...}}
if err := d.Run(context.Background()); err != nil {
    outErr := outputError(err.Error())   // stderr: {"ok":false,"error":...}
    if hint := startupErrorHint(err); hint != "" { outputHint(hint) }
    return outErr
}
```

`daemon.Run` (internal/daemon/daemon.go) performs the real startup work after the success line has already printed: write PID file, `browser.Start`, fetch browser version, `cdp.Dial`, subscribe to events, enable auto-attach, start the heartbeat, then create the IPC server and launch it with `go func(){ errCh <- d.server.Serve(ctx) }()`. Only after that does it block on a `select` awaiting signals, REPL exit, or browser disconnect. Any failure in those steps returns an error from `Run` — after stdout already reported success.

Before this change set, the `--system-profile`-in-use case hung for the full 30s CDP timeout, so the contradictory "success then error" pair was rarely observed. The new fail-fast path (`browser.ErrSystemProfileInUse`, returned well inside the timeout) makes it common: stdout carries `ok:true` while stderr carries `ok:false` for the same failed start. An agent reading only stdout misreads the failure as success.

The first moment the daemon is accepting commands is immediately after the IPC server starts serving — the browser is launched, CDP is connected, and the socket is bound. That is the natural readiness point. Target attachment continues asynchronously past this point, so a freshly-signalled daemon may report zero sessions for a brief window; this is the existing behaviour and is not addressed here. `daemon.Config` already carries a CLI-supplied callback (`CommandExecutor`), so adding a readiness callback follows an established pattern and keeps the daemon decoupled from the CLI output helpers.

`runStart` is not the only caller of `daemon.Run`. `runServeWithDaemon` (internal/cli/serve.go) launches `d.Run` in a background goroutine, then sleeps a fixed 500ms and does a non-blocking check of the daemon error channel before creating a direct executor and issuing the `serve` command:

```
go func() { daemonErr <- d.Run(context.Background()) }()
time.Sleep(500 * time.Millisecond)
select {
case err := <-daemonErr: // early failure
    ...
default: // assume started
}
```

The 500ms is a guess. Browser launch plus `cdp.Dial` (30s CDP timeout) can exceed it on a cold or loaded machine, in which case `serve` proceeds to issue commands against a daemon whose IPC socket is not yet accepting. The same readiness signal removes this race: because the callback is nil-safe (mirroring `CommandExecutor`, which falls back when nil), the daemon emits readiness regardless of caller, and `serve` waits on it instead of sleeping.

## Requirements

1. Start success (text `OK`, JSON `ok:true`) is reported only after the daemon has reached operational readiness — browser launched, CDP connected, IPC server serving.
2. When startup fails before readiness, the agent-facing output contains only the failure: no success line precedes the error.
3. In JSON mode, a failed start yields exactly one machine-readable result an agent can act on (`ok:false`), and a successful start yields `ok:true` including the port actually bound. Because the success line now fires after the daemon is serving, its message reflects the achieved state (`daemon ready`) rather than the in-progress `daemon starting` it carried when it printed before `Run`.
4. Readiness reporting behaves identically whether or not a REPL is attached (TTY and non-TTY stdin).
5. The daemon continues to run in the foreground and block until shutdown; existing background-start-and-poll automation is unaffected.
6. `runServeWithDaemon` gates command issuance on the readiness signal rather than a fixed sleep: it proceeds the instant the daemon is serving IPC and fails fast the instant `Run` returns an error, with no fixed-duration wait. The readiness callback is nil-safe, so any caller that does not set one (now or later) is unaffected.

## Constraints

1. Pure Go, standard library plus the existing dependency set in go.mod. No cgo.
2. Preserve the output-helper contracts in internal/cli/root.go: `outputSuccess`/`outputError`/`outputHint` and the `printedError` no-double-print behaviour.
3. Do not change the foreground-blocking daemon model that shell automation relies on (`webctl start &` then poll `webctl status`).
4. Backwards compatible: text mode still prints `OK` on a successful start, and existing tests that poll `is_daemon_running` continue to pass. The one intentional output change is the JSON success message string (`daemon starting` to `daemon ready`); the `start --json` assertion in `scripts/test/cli/test-start-stop.sh` is updated in lockstep so the suite stays green.

## Implementation Plan

1. Add a readiness signal to `daemon.Run`. Prefer a callback on `daemon.Config` (mirroring `CommandExecutor`) invoked once, from inside `Run`, immediately after the `go d.server.Serve(ctx)` goroutine is launched and strictly before the TTY/REPL setup block (`internal/daemon/daemon.go:311`). The listener is already bound in `ipc.NewServer`, so readiness is genuine at that point. This placement is deliberate: the REPL puts the terminal into raw mode when its goroutine runs `readline.NewEx`, so firing the readiness output after the REPL starts would let the success line race raw-mode entry and render differently on a TTY than on non-TTY stdin. Invoking the callback before any terminal-mode change makes the readiness output identical in both cases by construction. A buffered channel is an acceptable alternative if it reads more cleanly, provided the same ordering holds. The callback must be nil-safe: guard the invocation so a caller that leaves it unset (the daemon's own tests, and any future caller) is unaffected, exactly as `CommandExecutor` falls back when nil. Pass the resolved port (`b.Port()`) to the callback so consumers report the actual bound port rather than the requested one.
2. Move the success output in `runStart` out of the pre-`Run` position and into the readiness path, so it fires when the daemon signals ready. Because `Run` blocks on success, the signal must originate from within `Run` while it continues to block.
3. On the failure path, ensure `Run` returning an error before readiness produces only the error and hint — no success line. Reconcile this with the daemon-already-running early return, which must keep its current behaviour.
4. Keep the JSON-versus-text branch in the readiness output, including the bound port (use the actual port after auto-selection, which `Run` already resolves via `b.Port()`). Change the JSON success message from `daemon starting` to `daemon ready`, since it now prints at the moment the daemon is serving rather than before `Run`. Update the existing CLI test that pins the old string: `scripts/test/cli/test-start-stop.sh` asserts `.data.message == "daemon starting"` (around line 306); change it to `daemon ready`. Its `.data.port == 9222` assertion remains valid because the default port binds as requested.
5. Route `runServeWithDaemon` (internal/cli/serve.go) through the same readiness callback. Set a readiness callback on its `daemon.Config` that closes a `ready` channel, then replace the `time.Sleep(500 * time.Millisecond)` and its non-blocking `select` with a blocking `select` on `{ready, daemonErr}`: a value on `ready` means proceed to create the direct executor and issue the `serve` command; a value on `daemonErr` means the daemon failed before serving, so report the error and hint as today. This removes the fixed-duration guess and the cold-start race without changing serve's output or the rest of its flow. Because `serve` uses `cfg.Port = 0` (auto-detected CDP port), the resolved-port argument from step 1 matters here even though it does not for the default `start` path.

## Acceptance Criteria

1. A fast-failing start (for example `--system-profile` while another Chrome holds the default profile) prints no success line: in JSON mode stdout contains no `ok:true` object, and the only machine-readable result is the `ok:false` error.
2. A successful start prints `OK` (text) or `ok:true` (JSON) only after the daemon is serving IPC, and the JSON result includes the port actually bound.
3. Background-start-and-poll automation (`webctl start &` then poll `webctl status` / `is_daemon_running`) continues to succeed unchanged.
4. A successful start in JSON mode reports `.data.message == "daemon ready"` (not `daemon starting`), and the updated `start --json` assertion in `scripts/test/cli/test-start-stop.sh` passes.
5. `webctl serve` (when it starts the daemon itself) issues its first command only after the daemon is serving IPC, with no fixed-duration sleep in the path: a daemon that takes longer than the old 500ms to bind IPC no longer causes serve to race the socket, and a daemon that fails before serving makes serve report the failure without waiting out a timer.
