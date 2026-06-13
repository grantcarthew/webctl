# Project: Persistent Default Browser Profile

## Goal

Make `webctl start` reuse one persistent browser profile by default so that logins, cookies, and site state carry across daemon restarts. Today every launch gets a throwaway profile, which forces re-authentication and loses state on every restart. Disposable profiles become opt-in instead of the default.

## Scope

In scope:

- A persistent default profile stored under the user's XDG data directory, used when no profile flag is given.
- A `--temp-profile` flag that restores the current behavior: a throwaway profile created on start and deleted on stop.
- A `--user-data-dir <path>` flag that points the browser at an explicit profile directory, persistent and never deleted by webctl.
- A `--system-profile` flag that launches against the user's real Chrome profile.
- Verifying whether a stale Chrome singleton lock blocks relaunch of a persistent profile after an unclean exit, and clearing a confirmed-stale lock only if Chrome does not recover on its own.
- CLI-to-daemon plumbing for the profile selection, which does not exist today.
- Documentation for the new flags.

Out of scope:

- Multiple named profiles or profile switching beyond the modes above.
- Migration or import of an existing system Chrome profile.
- Changes to attach-mode (`webctl start --attach`), beyond not regressing it.

## Current State

Profile handling lives in `internal/browser` and is not exposed through the CLI.

- `internal/cli/start.go` defines `webctl start` with only `--headless` and `--port` flags. It builds `daemon.DefaultConfig()` and sets `Headless` and `Port`. There is no profile flag and no profile field is passed.
- `internal/daemon/daemon.go` `Config` has no profile field. `Run` calls `browser.Start(browser.LaunchOptions{Port, Headless})`, so `UserDataDir` is always the zero value.
- `internal/browser/launch.go` `LaunchOptions.UserDataDir` already supports three modes: empty string creates a temp dir via `os.MkdirTemp("", "webctl-chrome-*")`; the sentinel `"default"` (`UserDataDirDefault`) passes no `--user-data-dir` flag and uses the real system Chrome profile; any other value is used as a literal path. `buildArgs` and `spawnProcess` implement this switch.
- `internal/browser/browser.go` tracks ownership with `ownsData`, set to `opts.UserDataDir == ""`. `Browser.Close()` runs `os.RemoveAll(dataDir)` only when `ownsData` is true. This means an empty `UserDataDir` is the only case that gets deleted on stop.

Net effect: because the daemon always passes an empty `UserDataDir`, every launch creates and later deletes a fresh temp profile.

The existing ownership rule is convenient. If the CLI resolves the persistent default to a concrete path and passes that path, the browser layer treats it as a literal directory and `ownsData` is false, so it survives restarts with no change to the cleanup rule. Only the `--temp-profile` case needs to reach the browser as an empty `UserDataDir` so the existing temp-dir create-and-delete path runs unchanged.

There is no existing config or data directory for webctl beyond the runtime socket and PID files, which `internal/ipc/server.go` places under `$XDG_RUNTIME_DIR/webctl/` or `/tmp/webctl-<uid>/`. Those locations are ephemeral by design and are not appropriate for profile data.

## References

- XDG Base Directory Specification: `$XDG_DATA_HOME` defaults to `~/.local/share`. Persistent user data belongs here.
- Chrome writes `SingletonLock`, `SingletonSocket`, and `SingletonCookie` into a user-data-dir to enforce one instance per profile. A stale lock from an unclean exit can block a new launch on the same directory.

## Requirements

1. When `webctl start` is run with no profile flag, the browser uses a persistent profile directory at `$XDG_DATA_HOME/webctl/profile`, falling back to `~/.local/share/webctl/profile` when `XDG_DATA_HOME` is unset. The CLI resolver creates this directory with `os.MkdirAll(path, 0700)` when it resolves the persistent-default mode, rather than relying on Chrome to create it; this is deterministic across platforms and sets correct permissions on a directory that holds cookies and session state. Its contents persist across `webctl stop` and subsequent `webctl start`.

2. A `--temp-profile` boolean flag on `webctl start` creates a throwaway profile for that launch and removes it on stop. This reproduces the pre-change default behavior.

3. A `--user-data-dir <path>` string flag on `webctl start` launches the browser against the given directory. webctl never deletes it. The CLI rejects an empty value with a usage error and resolves the path with `filepath.Abs` before passing it down. This is required because `LaunchOptions.UserDataDir` overloads its value: an empty string means "temp profile, deleted on stop" and the literal `default` (`UserDataDirDefault`) means "system profile, no `--user-data-dir` flag." Without normalization, `--user-data-dir ""` would silently delete the named directory on stop and `--user-data-dir default` would launch the system profile instead of a directory called `default`. An absolute path can never alias either sentinel.

4. A `--system-profile` boolean flag on `webctl start` launches against the user's real Chrome profile. The CLI maps it to the existing `UserDataDirDefault` value internally so the browser passes no `--user-data-dir` flag. webctl never deletes this profile.

5. `--temp-profile`, `--user-data-dir`, and `--system-profile` are mutually exclusive. Supplying more than one is a usage error reported through the standard error helpers.

6. The daemon carries the resolved profile selection from the CLI through `daemon.Config` to `browser.LaunchOptions`. The browser layer continues to own temp-dir creation and deletion; persistent and explicit paths are never deleted by webctl.

7. A persistent profile must not be left unusable by a stale Chrome singleton lock after an unclean browser exit. First determine whether Chrome already recovers a stale lock on relaunch against the default profile. Add explicit lock clearing only for cases Chrome does not handle on its own, and only when the lock is confirmed stale: the recorded PID is not a live process and the host matches. Never remove a lock that may belong to a running browser. If explicit clearing is added, `webctl stop --force` removes the confirmed-stale lock state as part of reaping orphaned processes.

8. `docs/start.md` documents the profile modes and the default location. `webctl start --help` describes the new flags. The `--system-profile` documentation warns that it requires no other Chrome instance to be running on the default profile: a running instance was not started with remote debugging, the new launch forwards to it and exits, and webctl cannot attach. The persistent default avoids this by using its own dedicated directory.

9. `--system-profile` fails fast instead of hanging. When that mode is selected and the launched process exits within a short window without the CDP endpoint coming up, webctl reports a targeted error explaining the likely cause (an existing Chrome instance holds the default profile) rather than waiting out the full CDP start timeout.

## Constraints

- Pure Go, standard library only. No new dependencies. No cgo. Go 1.25.5 minimum, per `go.mod`.
- Use the existing CLI output helpers (`outputError`, `outputHint`, etc.) for usage errors. Do not write to stdout or stderr directly.
- Resolve XDG paths from the environment at runtime. Do not hardcode `/home` or a literal home path.
- Do not regress attach-mode. If the in-progress `attached` flag gates `Close()`, profile cleanup must remain correct for both attached and launched browsers.
- Follow the REPL flag-reset contract: if any new persistent global flag is added, reset it in `ExecuteArgs`. Per-command flags on `startCmd` are preferred and avoid this concern.
- Test isolation: the bash CLI suite currently launches via bare `webctl start`, which this change repurposes to the shared persistent profile. The suite must continue to run against throwaway profiles, and must never touch the developer's real profile; the test harness, not the new default, owns that isolation. This requires both a sandboxed `$XDG_DATA_HOME` for the whole suite and `--temp-profile` on every launch that should stay throwaway, including the direct `webctl start` sites in `test-start-stop.sh`, not only `start_daemon()`.

## Implementation Plan

1. Add an XDG-aware resolver for the default profile path (`$XDG_DATA_HOME/webctl/profile` with the `~/.local/share` fallback). The resolver creates the directory with `os.MkdirAll(path, 0700)` so the persistent default does not depend on Chrome creating it. This stays in the CLI layer; the browser layer still receives a concrete path. Applies only to the persistent default — `--user-data-dir` remains the user's responsibility, and the temp and system modes do not create anything here.

2. Add `--temp-profile`, `--user-data-dir`, and `--system-profile` flags to `startCmd`. Reject any combination of the three with a usage error.

3. Resolve the modes in the CLI into a single value passed to the daemon: temp profile resolves to an empty `UserDataDir` (so the browser creates and later deletes a temp dir); the default mode resolves to the persistent path; `--user-data-dir` rejects an empty value and passes its `filepath.Abs`-resolved path through, so the value can never alias the empty-string or `default` sentinels; `--system-profile` resolves to the `UserDataDirDefault` value.

4. Add a profile field to `daemon.Config`, set it in `runStart`, and pass it into `browser.LaunchOptions` from `daemon.Run`.

5. Verify singleton-lock behavior before building anything: force-kill the browser, then run `webctl start` against the default profile and observe whether Chrome recovers on its own. Add lock clearing only if it does not, gated on a confirmed-stale lock (recorded PID not alive, host matches), and extend `webctl stop --force` to remove that lock state. Do not delete a lock that may be live.

6. Update `docs/start.md` and the command help text, including the `--system-profile` warning that a running browser on the default profile blocks attachment.

   For `--system-profile`, detect the forward-and-quit case: if the launched process exits before the CDP endpoint responds, abort the wait early and return a targeted error pointing at an existing Chrome instance, rather than letting `waitForCDP` run its full timeout.

   Implement the exit detection with a single process-exit watcher on `Browser`, established at spawn: one goroutine calls `cmd.Wait()` exactly once and closes a `done` channel, capturing the exit error. `waitForCDP` selects on `done` alongside its ticker and context so a dead process aborts the wait early (with the `--system-profile`-specific message when that mode is selected), and `Browser.Close()` waits on the same `done` channel instead of calling `cmd.Wait()` itself. Do not add a second `cmd.Wait()` in the startup path: `Wait` is single-call, so a startup-side `Wait` would collide with the one in `Close()` and break its SIGKILL-fallback timing. The shared watcher is mandatory, not optional, because the fail-fast path and `Close()` would otherwise both reap the process.

7. Update the bash test harness for isolation, in two parts:
   - Export `XDG_DATA_HOME` to a per-run temporary directory for the whole CLI suite (set it in the harness setup, remove it on cleanup). This shields the developer's real `~/.local/share/webctl/profile`, isolates every `webctl start` path at once, and gives the step-8 default-profile test a known sandbox to write into and tear down.
   - Pass `--temp-profile` on the launches that should stay throwaway: `start_daemon()` in `scripts/bash_modules/setup.sh`, and the three direct `webctl start` invocations in `scripts/test/cli/test-start-stop.sh` that spawn a browser (initial start, `--port 9333`, `--json`). Patching only `start_daemon()` misses those direct sites. Without `--temp-profile`, every test shares `$XDG_DATA_HOME/webctl/profile`, where unclean `stop --force` teardowns leave a stale singleton lock that can block the next test's start, and profile state accumulates within a run.

8. Add or extend tests: argument building per mode in `internal/browser`, the mutually-exclusive flag error in `internal/cli`, and a CLI bash test that explicitly opts into the default persistent profile, writes a sentinel file into the resolved profile directory under the suite's sandboxed `$XDG_DATA_HOME`, and asserts it survives a default stop/start cycle but not a `--temp-profile` cycle.

## Implementation Guidance

- Keep the mode resolution in the CLI layer. The browser layer should stay a thin executor of a concrete `UserDataDir` value and should not learn about XDG or webctl's default location.
- Prefer mapping `--temp-profile` to an empty `UserDataDir` rather than adding a new browser-layer flag. This reuses the existing create-and-delete path and its ownership rule without inverting `ownsData`.
- Singleton-lock clearing is conditional on the verification in step 5; do not build it if Chrome already recovers. The lock encodes a host and PID; treat it as stale only when the host matches and the recorded PID is not a live process. Never delete a lock that may belong to a running browser.

## Acceptance Criteria

- With no flags, a sentinel file placed in the resolved profile directory under `$XDG_DATA_HOME/webctl/profile` survives a `webctl stop` and a subsequent `webctl start`, confirming the same directory is reused; the directory exists and is non-empty after stop.
- With `--temp-profile`, the profile directory does not persist after `webctl stop`, matching the previous default.
- With `--user-data-dir <path>`, the browser uses that directory and webctl does not delete it on stop.
- With `--system-profile`, the browser launches with no `--user-data-dir` flag and webctl does not delete any profile on stop.
- With `--system-profile` while another Chrome instance already holds the default profile, `webctl start` fails quickly with an error naming the existing-instance cause, well inside the CDP start timeout, rather than hanging for the full wait.
- Passing more than one of `--temp-profile`, `--user-data-dir`, or `--system-profile` produces a usage error and a non-zero exit.
- After a browser is force-killed, a subsequent `webctl start` against the default profile launches successfully without manual lock removal.
- `docs/start.md` and `webctl start --help` describe the profile modes and the default location.
