# Signal daemon readiness before reporting start success

Source: pre-commit review on 2026-06-15
Severity: info
Category: Observability / Correctness
Location: internal/cli/start.go:102-119, internal/daemon/daemon.go:212-308

## Goal

Make `webctl start` report success only once the daemon is actually operational, so a failed start never emits a success line ahead of its error. This keeps the JSON output contract trustworthy for AI agents, which are the primary consumers of webctl output.

## Scope

In scope: the startup success/failure reporting in `runStart` and the readiness signal it depends on from `daemon.Run`. The daemon keeps running in the foreground and blocking until shutdown.

Out of scope: changing the daemon lifecycle model (foreground blocking, background-and-poll automation), the profile-resolution logic, the browser launch path, and the fail-fast `--system-profile` error itself.

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

The first moment every subsystem is live is immediately after the IPC server starts serving. That is the natural readiness point. `daemon.Config` already carries a CLI-supplied callback (`CommandExecutor`), so adding a readiness callback follows an established pattern and keeps the daemon decoupled from the CLI output helpers.

## Requirements

1. Start success (text `OK`, JSON `ok:true`) is reported only after the daemon has reached operational readiness — browser launched, CDP connected, IPC server serving.
2. When startup fails before readiness, the agent-facing output contains only the failure: no success line precedes the error.
3. In JSON mode, a failed start yields exactly one machine-readable result an agent can act on (`ok:false`), and a successful start yields `ok:true` including the port actually bound.
4. Readiness reporting behaves identically whether or not a REPL is attached (TTY and non-TTY stdin).
5. The daemon continues to run in the foreground and block until shutdown; existing background-start-and-poll automation is unaffected.

## Constraints

1. Pure Go, standard library plus the existing dependency set in go.mod. No cgo.
2. Preserve the output-helper contracts in internal/cli/root.go: `outputSuccess`/`outputError`/`outputHint` and the `printedError` no-double-print behaviour.
3. Do not change the foreground-blocking daemon model that shell automation relies on (`webctl start &` then poll `webctl status`).
4. Backwards compatible: text mode still prints `OK` on a successful start, and existing tests that poll `is_daemon_running` continue to pass.

## Implementation Plan

1. Add a readiness signal to `daemon.Run`. Prefer a callback on `daemon.Config` (mirroring `CommandExecutor`) invoked once, from inside `Run`, immediately after the IPC server begins serving and before the blocking `select`. A buffered channel is an acceptable alternative if it reads more cleanly.
2. Move the success output in `runStart` out of the pre-`Run` position and into the readiness path, so it fires when the daemon signals ready. Because `Run` blocks on success, the signal must originate from within `Run` while it continues to block.
3. On the failure path, ensure `Run` returning an error before readiness produces only the error and hint — no success line. Reconcile this with the daemon-already-running early return, which must keep its current behaviour.
4. Keep the JSON-versus-text branch in the readiness output, including the bound port (use the actual port after auto-selection, which `Run` already resolves via `b.Port()`).

## Acceptance Criteria

1. A fast-failing start (for example `--system-profile` while another Chrome holds the default profile) prints no success line: in JSON mode stdout contains no `ok:true` object, and the only machine-readable result is the `ok:false` error.
2. A successful start prints `OK` (text) or `ok:true` (JSON) only after the daemon is serving IPC, and the JSON result includes the port actually bound.
3. Background-start-and-poll automation (`webctl start &` then poll `webctl status` / `is_daemon_running`) continues to succeed unchanged.
