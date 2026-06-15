# webctl start

Start the webctl daemon, which launches a browser and begins capturing Chrome DevTools Protocol (CDP) events.

## Synopsis

```bash
webctl start                          # Persistent default profile
webctl start --temp-profile           # Throwaway profile, deleted on stop
webctl start --user-data-dir <path>   # Explicit profile directory
webctl start --system-profile         # Your real Chrome profile
```

## Description

The `start` command runs the persistent daemon. It launches Chrome/Chromium with CDP enabled, connects over a WebSocket, buffers events (console, network, and others), and serves IPC over a Unix socket so that later stateless commands can query them.

By default the daemon reuses a **persistent profile**, so logins, cookies, and site state carry across daemon restarts. The profile modes below are mutually exclusive — supplying more than one is a usage error.

## Profile Modes

### Persistent default (no flag)

```bash
webctl start
```

With no profile flag, the browser uses a persistent profile directory at:

```
$XDG_DATA_HOME/webctl/profile
```

falling back to `~/.local/share/webctl/profile` when `XDG_DATA_HOME` is unset. The directory is created automatically (mode `0700`) and its contents persist across `webctl stop` and subsequent `webctl start`. This is the recommended mode for day-to-day use because authenticated sessions survive restarts.

The persistent default uses its own dedicated directory, so it never conflicts with a running Chrome instance.

### Temporary profile

```bash
webctl start --temp-profile
```

Creates a throwaway profile for that launch and removes it on stop. This reproduces the previous default behavior: every launch starts from a clean slate with no carried-over state. Use it for isolated, reproducible sessions.

### Explicit profile directory

```bash
webctl start --user-data-dir /path/to/profile
```

Launches the browser against the given directory. webctl never deletes it. The path is resolved to an absolute path; an empty value is rejected with a usage error.

### System profile

```bash
webctl start --system-profile
```

Launches against your real Chrome profile (the one Chrome uses when you start it normally). webctl never deletes it.

> **Warning:** `--system-profile` requires that no other Chrome instance is already running on the default profile. If a normal Chrome window is open (it was not started with remote debugging), the new launch forwards to that instance and immediately exits, and webctl cannot attach. In that case `webctl start --system-profile` fails fast with a targeted error rather than hanging for the full CDP start timeout. Close the running Chrome first, or use the persistent default profile, which avoids this entirely by using its own dedicated directory.

## Flags

| Flag | Description |
|------|-------------|
| `--headless` | Run the browser without a visible window. |
| `--port <n>` | CDP port for the browser (default `9222`). |
| `--temp-profile` | Use a throwaway profile, deleted on stop. |
| `--user-data-dir <path>` | Use an explicit profile directory, never deleted by webctl. |
| `--system-profile` | Use the real Chrome profile (no other Chrome may run on it). |
| `--json` | Emit machine-readable JSON output. |

## Crash recovery

After an unclean exit (a crash, `webctl stop --force`, or a `SIGKILL`), Chrome records the previous session as crashed. On the next `webctl start` against a persistent or explicit profile, webctl suppresses the "Restore pages?" crash-restore bubble (via `--hide-crash-restore-bubble`) so it does not overlay the active page or interfere with screenshots and click coordinates. A stale Chrome singleton lock left by such an exit is recovered automatically by Chrome on relaunch.

## Behavior

- The command blocks while the daemon runs. In shell automation, run it in the background and poll `webctl status`.
- If a daemon is already running, `start` reports an error and hints at `webctl stop`.
- If the requested port is in use, the launch fails; use `webctl stop --force` to reap orphaned processes.

## See also

- `webctl stop` — stop the daemon and the browser it owns.
- `webctl status` — report daemon state.
