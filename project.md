# Tab Command

## Goal

Replace the `target` command with a `tab` command family covering list, switch, new, and close. The user-facing concept is "tab", not CDP's internal "target". Every operation that changes the active session also foregrounds the resulting tab in the browser, so what webctl operates on matches what the user sees.

## Scope

In scope:

- Rename the `target` CLI command and IPC dispatch entry to `tab`.
- Bare `webctl tab` lists tabs.
- `webctl tab switch <query>` sets the webctl active session and foregrounds the tab.
- `webctl tab new [url]` creates a new tab and makes it active.
- `webctl tab close [query]` closes a tab; closes the active tab when no query is given.
- Bash integration tests under `scripts/test/cli/`.
- Go tests under `internal/cli/` and `internal/daemon/`.
- AGENTS.md updates: Quick Reference and Active Project pointer.
- Removal of the old `target` command, its IPC handler, and any text-formatter helpers that exist only to serve it. No deprecation alias.

Out of scope:

- Window management (new windows vs tabs).
- Tab positioning, ordering, groups.
- Tab duplication.
- A `--wait` flag for `tab new`. Use the existing `webctl ready` for load waiting.
- A `--no-activate` or `--background` flag for `tab new`.
- Renaming internal CDP-aligned types (`SessionManager`, `PageSession`, `TargetData`).
- Broader AGENTS.md cleanup of the `.ai/projects/` workflow text.

## Status

In progress. Implementation landed; one outstanding test investigation remains.

Completed:

- IPC types: `TabParams`, `TabData`, `NewTabData` in `internal/ipc/protocol.go`. `TargetData` removed.
- `SessionManager` extended with `TargetID(sessionID)` and `GetByTargetID(targetID)` accessors (`internal/daemon/session.go`).
- Daemon waiter fields `tabAttachWaiters` (targetID-keyed) and `tabDetachWaiters` (sessionID-keyed) on `Daemon`. `handleTargetAttached` and `handleTargetDetached` non-blocking-send on the matching waiter (`internal/daemon/events.go`).
- `handleTab` dispatch and four sub-action helpers in `internal/daemon/handlers_tab.go`: list, switch (calls `Target.activateTarget`), new (`Target.createTarget` + attach-waiter + sets active), close (last-tab guard, `Target.closeTarget` + detach-waiter + foregrounds promoted active).
- IPC dispatch entry `case "tab"` in `handleRequest`. Old `case "target"` removed.
- `internal/daemon/handlers_session.go`: `handleTarget` removed, `noActiveSessionError` message updated to reference `webctl tab switch <query>`.
- CLI command tree in `internal/cli/tab.go`: `tab` (list), `tab switch`, `tab new` (with `normalizeURL`), `tab close`. Old `internal/cli/target.go` deleted.
- Text formatters updated in `internal/cli/format/text.go`: `Tab` and `TabError` (replacing `Target`/`TargetError`).
- AGENTS.md Quick Reference now lists the four `tab` forms. Active Project line already pointed at this project.
- Tests:
  - `internal/cli/cli_test.go`: tab tests (list, switch, ambiguous, new with/without URL/localhost, close no-query, last-tab error). Old `runTarget` tests replaced.
  - `internal/cli/format/format_test.go`: `TestTab`, `TestTabError_AmbiguousMatches` (replaced `TestTarget`).
  - `internal/daemon/handlers_tab_test.go`: unit tests for `SessionManager.TargetID`, `GetByTargetID`, `ambiguousTabError`, and the no-CDP error paths (no-match switch, ambiguous switch, no-active close, last-tab guard, list).
  - `internal/daemon/integration_test.go`: `TestTab_Integration` covering list / new (about:blank and explicit URL) / switch by id prefix / no-match / close with promotion against a real headless browser.
  - `scripts/test/cli/test-tab.sh`: end-to-end bash coverage of all four sub-actions including last-tab guard, no-match, localhost auto-detection, and active-tab close promotion.

Test results:

- `go test ./...`: passes (daemon integration suite ~41s, all green).
- `bash scripts/test/cli/test-tab.sh`: 30/30 pass.
- `bash scripts/test/cli/test-start-stop.sh`: 53/53 pass.
- `bash scripts/test/cli/test-navigation.sh`: 104/105 pass (see Remaining Issues).

## Remaining Issues

1. `test-navigation.sh` has one failing assertion: `Browser navigated to navigation.html: 'navigation.html' not in captured URL`. The failure occurred during a re-run after the tab-command landing; subsequent runs of `test-tab.sh` and `test-start-stop.sh` are clean, and the unit + integration Go suites are green. Likely a pre-existing flake (timing-sensitive `capture_page_state` after the very first navigation in the script), but it should be re-run in isolation and triaged before declaring this project done. Reproduction: `bash scripts/test/cli/test-navigation.sh` from a clean state with `force_stop_daemon` first. If reproducible, capture the failing assertion's context (the test name / URL it was navigating to) and confirm whether it relates to the `tab` work (it should not — `test-navigation.sh` does not call `tab`).

2. `test-observation.sh` and `test-interaction.sh` have not been re-run yet. They do not exercise `target`/`tab` directly but should be confirmed clean before the project is closed.

3. The CDP warning observed during `test-tab.sh` ("warning: failed to enable domains for session: failed to enable Page.enable: cdp error -32001: Session with given id not found.") fires when a tab is closed before the asynchronous `enableDomainsForSession` goroutine runs. It is benign — the session is already gone — but the noise on stderr is worth tracking as a follow-up. A fix would either ignore `-32001` in `enableDomainsForSession` or check `sessions.Get(sessionID) != nil` before issuing the enables.

## Current State

CLI surface:

- `internal/cli/target.go` — current `target` command: bare lists, positional query switches.
- `internal/cli/navigate.go` — navigation pattern with optional `--json`, optional `--wait`/`--timeout`, and `normalizeURL` for protocol auto-detection.
- `internal/cli/cookies.go`, `css.go`, `clear.go` — examples of subcommand families with text and JSON output.
- The global `--json` flag is wired through `outputJSON`/`outputSuccess`/`outputError` helpers used by every command.

Daemon and IPC:

- `internal/daemon/daemon.go` — `handleRequest` is a switch on `req.Cmd`. Adding a new command requires a new case.
- `internal/daemon/handlers_session.go` — current `handleTarget` and the `noActiveSessionError` helper that returns the available-sessions list with the error.
- `internal/daemon/events.go` — `handleTargetCreated` auto-attaches new page targets via `Target.attachToTarget` with `flatten:true`. `handleTargetAttached` registers sessions in `SessionManager` and asynchronously enables CDP domains. `handleTargetDetached` removes sessions and purges buffer entries.
- `internal/daemon/session.go` — `SessionManager`. `Remove()` already promotes the most-recently-attached remaining session to active when the active one is removed. `FindByQuery` matches ID prefix (case-sensitive) then title substring (case-insensitive). `order` slice tracks attachment order, newest last.
- `internal/ipc/protocol.go` — `Request`, `Response`, `TargetData`, `PageSession`. New params and data types for tab operations go here.

CDP integration:

- The daemon connects browser-level via `cdp.Dial` and calls `Target.setDiscoverTargets` at startup. Manual `Target.attachToTarget` with `flatten:true` is used per-target. `setAutoAttach` is intentionally not used (causes networkIdle blocking).
- `Target.createTarget`, `Target.activateTarget`, and `Target.closeTarget` are not currently called anywhere in the daemon.

Tests:

- `scripts/test/cli/test-*.sh` — bash integration tests against a real browser. Existing scripts cover navigation, interaction, observation, start/stop. There is no `tests/` directory and no bats infrastructure.
- `internal/cli/cli_test.go` and `internal/daemon/integration_test.go` — Go test coverage for handlers and CLI plumbing.

Documentation:

- AGENTS.md Quick Reference does not list any tab management command; `target` is absent and the four `tab` forms need to be added.
- AGENTS.md Active Project pointer is stale (still references the completed heartbeat project).
- Project documents now live at the repo root as `project.md`. The `.ai/projects/` directory referenced elsewhere in AGENTS.md does not exist.

Text formatters:

- `internal/cli/format/text.go` defines `Target` (list output) and `TargetError` (multi-match / no-match formatting). These are the only target-only formatter helpers and will be replaced by `Tab` / `TabError` (or equivalent) and removed.

## Requirements

1. CLI command `webctl tab` with these forms:
   - Bare `webctl tab` lists all tabs (text by default, `--json` for structured).
   - `webctl tab switch <query>` sets the webctl active session and foregrounds the tab in the browser.
   - `webctl tab new [url]` creates a new tab. The optional URL is normalised the same way as `webctl navigate` (`https://` default; `http://` for `localhost`, `127.0.0.1`, `0.0.0.0`; explicit protocols preserved). With no URL, opens `about:blank`. The new tab becomes the active session.
   - `webctl tab close [query]` closes a tab. With no query, closes the active tab. If the closed tab was active, the most-recently-opened remaining tab becomes active and is foregrounded.

2. Query semantics for `switch` and `close`:
   - Match against session ID prefix first (case-sensitive), then fall back to title substring (case-insensitive).
   - Zero matches: error `no tab matches query: <query>`.
   - Multiple matches: error `ambiguous query '<query>'` with the candidate list returned in the JSON `matches` field and printed by the text formatter.
   - Single match: proceed.

3. Output:
   - All forms support text and JSON via the existing `--json` global flag.
   - Bare `tab` output matches today's `target` listing layout (id, title, url, active marker).
   - `switch`, `new`, `close` return `OK` in text mode; `{ok, ...}` in JSON. `new` includes the new session id and url; `switch` and `close` include the resulting active session id.

4. Error handling:
   - `tab close` on the last remaining tab: refuse with `cannot close the last tab; use 'webctl stop' to shut down the browser`. Check before sending the CDP call.
   - `tab close` with no query and no active tab: error `no active tab`.
   - `tab new` with an invalid URL: surface the CDP error text from the `Target.createTarget` response.
   - `tab switch` and `tab close` with zero or multiple matches: as in requirement 2.

5. CDP wiring:
   - `tab` (list) reads from `SessionManager`. No CDP calls.
   - `tab switch` calls `SessionManager.SetActive` then `Target.activateTarget`.
   - `tab new` calls `Target.createTarget` with the URL (or `about:blank`) and `newWindow:false`. The handler waits for the corresponding session to be registered by the existing `Target.targetCreated` → `attachToTarget` → `handleTargetAttached` flow before returning. The new session becomes active. CDP foregrounds the new tab by default; no explicit `activateTarget` is required.
   - `tab close` calls `Target.closeTarget`, then waits for the matching session to be removed from `SessionManager` (via the existing `Target.detachedFromTarget` flow) before returning. If the closed tab was active, the handler then reads the new `SessionManager.ActiveID()` and calls `Target.activateTarget` on it.

6. IPC dispatch: a single `tab` command. Sub-action is carried in the request payload, consistent with how `cookies` carries `Action: "list"|"set"|"delete"` in `CookiesParams`.

7. Tests:
   - Bash integration script `scripts/test/cli/test-tab.sh` exercising list, switch (zero/one/multiple matches), new (with and without URL, with localhost auto-detection), close (with and without query, last-tab refusal, active-tab close promoting next active).
   - Go test coverage for the handler logic in `internal/daemon/` covering each sub-action and each error path, plus CLI plumbing in `internal/cli/cli_test.go`.

8. Documentation:
   - Add the four `tab` forms to AGENTS.md Quick Reference (no `target` entry exists today, so this is an addition rather than a rename).
   - Update the AGENTS.md Active Project line to reference this project.
   - Update inline command help and any error messages that mention `target` (notably `noActiveSessionError`'s text, which currently says `use 'webctl target <id>' to select`).

9. Removal: delete the `target` cobra command, the `target` case in `handleRequest`, `handleTarget`, and any text-formatter functions that exist only to serve `target`. No deprecation alias.

## Implementation Plan

1. Add tab IPC types in `internal/ipc/protocol.go`. A single `TabParams` carrying `Action` (`"list"|"switch"|"new"|"close"`), optional `Query`, optional `URL`. Reuse `PageSession` for list output. Add response data shapes for `new` (id, url, title) and `switch`/`close` (resulting active id).

2. Extend `SessionManager` (`internal/daemon/session.go`) with two accessors needed by the tab handlers: `TargetID(sessionID string) string` returning the targetID for a given sessionID (empty if not found), and `GetByTargetID(targetID string) *PageSession` returning the session for a given targetID (nil if not found). Both wrap reads of the existing internal `session` struct under `mu.RLock()`. `PageSession` itself stays unchanged.

3. Add a `tab` case to `handleRequest` in `internal/daemon/daemon.go`. Implement `handleTab` dispatching on `Action`. Reuse the `requireBrowser` guard, `noActiveSessionError` helper, and `FindByQuery` pattern from the current `handleTarget`.

4. Implement the four sub-actions:
   - List: wraps the existing list logic from `handleTarget`.
   - Switch: `FindByQuery`, validate single match, `SessionManager.SetActive`, then `Target.activateTarget` using the targetID from `SessionManager.TargetID(sessionID)`.
   - New: `Target.createTarget`, wait until the new session appears in `SessionManager` (bounded timeout), recover the sessionID via `SessionManager.GetByTargetID(targetID)`, `SessionManager.SetActive`, return `{id, url, title}`.
   - Close: resolve query (or use the active session when empty), apply the last-tab guard, `Target.closeTarget` using the targetID from `SessionManager.TargetID(sessionID)`, check the response `success` flag (treat `false` as an error and skip the wait), wait for the session to be removed from `SessionManager` (bounded timeout), then if the closed session was active, foreground the new active by reading the new `SessionManager.ActiveID()` and calling `Target.activateTarget` on its targetID.

5. Add the `tab` cobra command and its subcommands in a new `internal/cli/tab.go`. Reuse `normalizeURL` from `navigate.go` for `tab new`.

6. Update text formatting under `internal/cli/format/` for tab list and tab error responses (multiple matches, last-tab, no match).

7. Delete `internal/cli/target.go`, `handleTarget`, the `target` case in `handleRequest`, and any target-only formatter helpers. Update `noActiveSessionError`'s message text.

8. Add `scripts/test/cli/test-tab.sh` covering all sub-actions and error paths.

9. Add Go test coverage in `internal/daemon/` and `internal/cli/cli_test.go`.

10. Update AGENTS.md Quick Reference and Active Project line.

11. Run the bash test suite and `go test ./...`. Fix any regressions in tests that exercised `target` semantics.

## Constraints

- Pure Go, no cgo.
- Follow existing patterns: cobra for CLI, IPC `Request`/`Response`, `requireBrowser` guard at handler entry, the global `--json` flag.
- Browser-level CDP calls via `d.cdp.Send`. Session-scoped calls via `d.sendToSession` for connection-error detection.
- Continue using `Target.setDiscoverTargets` plus manual `Target.attachToTarget` with `flatten:true`. Do not introduce `setAutoAttach`.

## Implementation Guidance

- The `tab new` handler coordinates with the asynchronous attach flow via a target-id-keyed waiter `sync.Map` on the `Daemon`. Sequence: send `Target.createTarget`, read the returned targetID, register a buffered (capacity 1) one-shot channel keyed by that targetID, then check `SessionManager.GetByTargetID(targetID)` — if the session is already present, skip the wait; otherwise wait on the channel with a bounded timeout (a few seconds is sufficient under normal load). The check-then-wait order matters because the daemon read loop can process the `Target.attachedToTarget` event before this handler regains control after `Send` returns; the buffered channel plus the post-register check closes the race in both directions. `handleTargetAttached` does a non-blocking send on the channel after `SessionManager.Add` (silently no-ops if no waiter exists, which is fine — the handler will see the session via `GetByTargetID`). The handler must `defer` waiter cleanup so the entry is removed on every exit path (timeout, CDP error, success). This matches the existing waiter pattern used by `navWaiters`, `loadWaiters`, `navigating`, `attachedTargets`, and `networkEnabled`. Do not poll `SessionManager`.

- The `tab close` handler uses the same buffered-channel pattern, keyed by session id. Sequence: register the waiter, send `Target.closeTarget`, check the response `success` flag (treat `false` as an error — return an error, skip the wait), then check `SessionManager.Get(sessionID)` — if the session is already gone, skip the wait; otherwise wait on the channel with a bounded timeout. `handleTargetDetached` non-blocking-sends on the channel after `SessionManager.Remove`. The handler `defer`s waiter cleanup on all exit paths (timeout, CDP error, success). After the waiter resolves, `SessionManager` already reflects the close, so reading `ActiveID()` returns the correctly promoted next active. This avoids a race where a follow-up `tab list` would otherwise show the just-closed session as still present.

- `tab new` returns once the session is registered in `SessionManager`. Domain initialisation (`Page.enable`, `Runtime.enable`, `DOM.enable`, `Network.enable`, `Page.setLifecycleEventsEnabled`) happens asynchronously inside `handleTargetAttached`'s goroutine and completes shortly after. This matches the lifecycle of any newly attached tab today. Callers that immediately need page-load completion on the new tab should chain `webctl ready`; this project does not add a `--wait` flag to `tab new`.

- `handleTab`'s switch path calls `d.repl.refreshPrompt()` after `SessionManager.SetActive`, so the REPL prompt updates to reflect the new active session. Today's `handleTarget` omits this and the prompt only refreshes when the next CDP event fires; the new implementation fixes that small UX gap. `tab new` and `tab close` already get prompt refreshes for free via the existing `handleTargetAttached`/`handleTargetDetached` calls.

- The active-tab principle is one mental model: webctl's active tab is the tab the browser shows. Every action that changes the active session must also foreground the resulting tab, except where CDP already does it (`Target.createTarget` foregrounds by default).

- The last-tab guard runs before the `Target.closeTarget` call. The check is `SessionManager.Count() == 1` plus the resolved match being the only session. Returning the guard error before any CDP call avoids a half-completed shutdown if `requireBrowser` then trips on the empty session list.

- Internal types (`SessionManager`, `PageSession`, `TargetData`) stay CDP-aligned. Only the user-facing surface and IPC `Cmd` strings are renamed. If the existing `TargetData` is reused for tab list responses, keep the type name; consumers in test code and formatters remain stable.

- New daemon handler code lives in a new `internal/daemon/handlers_tab.go`, holding `handleTab` and its sub-action helpers. `noActiveSessionError` stays in `handlers_session.go` because it is shared by handlers across multiple files (`handleNavigate`, `handleHTML`, and others), not specific to the tab family.

## Acceptance Criteria

- `webctl tab` lists all open tabs in text and JSON modes; output matches the format produced by today's `webctl target`.
- `webctl tab switch <query>` sets the active session and the named tab is visibly foregrounded in the browser.
- `webctl tab new` opens an `about:blank` tab and makes it the active session.
- `webctl tab new <url>` opens a tab navigated to the URL with protocol auto-detection (`example.com` → `https://example.com`; `localhost:3000` → `http://localhost:3000`) and makes it the active session.
- `webctl tab close` with no query closes the currently active tab; the most-recently-opened remaining tab becomes active and is foregrounded.
- `webctl tab close <query>` closes the matched tab; if it was active, the most-recently-opened remaining tab becomes active and is foregrounded.
- `webctl tab close` against a single-tab session returns `cannot close the last tab; use 'webctl stop' to shut down the browser` and does not call `Target.closeTarget`.
- A `webctl tab close <query>` immediately followed by `webctl tab` does not list the just-closed tab; the close handler does not return until `SessionManager` reflects the removal.
- `webctl tab switch <query>` and `webctl tab close <query>` return `no tab matches query` for zero matches and `ambiguous query` with a candidate list for multiple matches.
- The `target` command and any internal references to it no longer exist in the binary, command help text, or AGENTS.md Quick Reference.
- `scripts/test/cli/test-tab.sh` passes.
- `go test ./...` passes.
