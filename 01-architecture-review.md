# Architecture Review Remediation

Work through the findings in this project using obo.

## Goal

Resolve the structural findings from the webctl architecture review one at a time, biasing each fix toward the principled long-term solution rather than the smallest diff. The aim is to reduce the concentration of state and behaviour in the Daemon type, tighten boundaries and contracts, and close observability gaps, without regressing the daemon-plus-stateless-command model that the system depends on.

## Scope

In scope: the eleven findings listed under Requirements, spanning internal/daemon, internal/cdp, internal/cli, internal/ipc, and internal/browser.

Out of scope:
- The single-binary client-plus-daemon model and the IPC-over-Unix-socket transport. These are sound and are not under review.
- The write-once command design (the Executor / ExecutorFactory abstraction that lets commands run both over IPC and inside the daemon REPL). Preserve it.
- internal/server decoupling from browser/cdp. It is already clean; do not entangle it.
- Adding a persistent datastore. In-memory ring buffers that reset on restart are a deliberate, correct choice.
- Finding L5 (CLI imports daemon/browser directly in start.go). Recorded as informational only; it is correct because the binary is both client and server. No change required. Listed here so a future reviewer does not re-raise it.

## Current State

webctl is a browser-automation CLI for AI agents, written in pure Go (module github.com/grantcarthew/webctl, Go 1.25.5 minimum, no cgo). A persistent daemon launches Chrome/Chromium, holds a Chrome DevTools Protocol (CDP) WebSocket, buffers ephemeral CDP events, and serves IPC over a Unix socket. All non-lifecycle commands are short-lived processes that talk to the daemon and exit.

Package layout and dependency flow (acyclic):
- cmd/webctl depends only on internal/cli.
- internal/cli depends on internal/executor, internal/ipc, the formatters, and (only in start.go) internal/daemon and internal/browser.
- internal/daemon depends on internal/browser, internal/cdp, internal/ipc, internal/server.
- internal/ipc is the shared contract leaf both the CLI and daemon converge on.
- internal/cdp and internal/browser are independent leaves.

Key files referenced by the findings:
- internal/daemon/daemon.go: Daemon struct (lines 67-110) and the request dispatch switch (handleRequest, lines 489-544).
- internal/daemon/events.go: CDP event ingestion; producer side of navigation rendezvous.
- internal/daemon/handlers_navigation.go: consumer side of navigation rendezvous.
- internal/daemon/handlers_*.go: 24 command handlers, all methods on Daemon, constructing raw CDP calls inline.
- internal/daemon/handlers_serve.go: dev-server lifecycle and the OnReload hot-reload callback.
- internal/cdp/client.go: CDP client, read loop, request/event demultiplexing.
- internal/cli/root.go: package-level Debug/JSONOutput/NoColor flags, commandGroups map, ExecuteArgs flag-reset logic.
- internal/cli/client.go: package-level execFactory and ExecutorFactory.
- internal/ipc/protocol.go: ~380-line wire contract with 30+ command DTOs.
- internal/browser/target.go: exported discovery helpers FetchTargets, FetchVersion, FindPageTarget.

The full review is at .start/reviews/2026-06-22-architecture-01.md.

Repository conventions that bound the work (from AGENTS.md):
- Pure Go, standard library and existing go.mod dependencies only. No cgo.
- Avoid speculative abstraction. Prefer a concrete handler or command over a new framework.
- Use the output helpers (outputSuccess/outputError/outputNotice/outputHint) rather than writing to stdout/stderr directly.
- Format with gofmt; pass go vet and staticcheck.
- Test via ./test-runner (go unit/race, lint, cli bash suite). Run ./test-runner quick before pushing and ./test-runner ci for anything touching command surface, daemon, or IPC.
- Do not add Co-Authored-By trailers.

## References

- .start/reviews/2026-06-22-architecture-01.md: the source architecture review this project remediates.
- AGENTS.md: repository conventions, build, testing, and CLI/daemon contracts.

## Requirements

Work the findings in severity order. Each is a standalone deliverable. Land at the root of each problem, not the symptom that surfaced it.

### R1 (High, H1): Decompose the Daemon god-object

Location: internal/daemon/daemon.go:67-110 and all internal/daemon/handlers_*.go.

Problem: a single Daemon struct owns the browser handle, CDP client, session manager, two ring buffers, the IPC server, the dev HTTP server, terminal state, the REPL pointer, shutdown coordination, and seven sync.Map fields. All 24 handlers are methods on it, so every handler has ambient access to every field. Change cost and reasoning load concentrate here.

Direction: extract cohesive sub-components that own their own state and expose a narrow interface, leaving Daemon as a thin coordinator. At minimum, pull out a navigation/lifecycle coordinator (consumed by R2), and consider separating dev-server control and buffer/observation state into their own owners. Handlers should depend on the sub-component they need, not on the whole Daemon. Do not introduce a speculative framework; introduce concrete types that each reduce total complexity now.

This is the largest item. It is acceptable to land it as a sequence of focused extractions rather than one change, provided each step compiles and passes tests. R2 is the natural first extraction.

### R2 (High, H2): Encapsulate navigation rendezvous behind one owner

Location: internal/daemon/daemon.go:88-109 (navWaiters, loadWaiters, navigating, attachedTargets, networkEnabled, tabAttachWaiters, tabDetachWaiters), internal/daemon/events.go (producer), internal/daemon/handlers_navigation.go (consumer).

Problem: navigation and target/session lifecycle correctness depends on an unwritten contract about the order in which channels are created, stored, closed, and deleted across two files and multiple goroutines (the CDP read loop plus request goroutines). Seven parallel maps track related lifecycle facts with no single owner, making the logic fragile to modify and easy to regress.

Direction: introduce an explicit owner (for example a per-session coordinator) that encapsulates these maps behind named operations with a documented lifecycle (begin navigation, await load, signal load complete, attach/detach, enable network). The ordering rules that are currently implicit comments become invariants enforced inside the type. The read-loop producer and the request-goroutine consumer interact only through that type's methods. Verify the previously documented race-avoidance ordering (registering a waiter before the awaited event can fire) is preserved by construction, not by comment.

### R3 (Medium, M1): Reduce CDP protocol coupling in the handler layer

Location: internal/daemon/handlers_navigation.go, handlers_css.go, handlers_interaction.go, handlers_observation.go, and peers.

Problem: handlers construct raw CDP method calls (Page.navigate, Runtime.evaluate, Input.dispatchMouseEvent, etc.) and unmarshal raw response JSON inline. CDP domain knowledge and cross-cutting policy (such as per-call timeouts) are spread across every handler file.

Direction: this is an intentional trade-off consistent with the repo's avoid-speculative-abstraction rule, so do not build a full page-operations layer on spec. Instead, factor out the repeated, decision-free mechanics that already recur: a single place for per-call timeout/context policy, and helpers for the most duplicated send-and-parse patterns. Keep handlers expressing intent; remove only the copy-paste that obscures it. If R1/R2 introduce a session/CDP sub-component, route handler CDP calls through it where natural.

### R4 (Medium, M2): Make the command registration coupling explicit

Location: internal/daemon/daemon.go:489-544 (dispatch switch), internal/cli command files, internal/ipc/protocol.go, internal/cli/root.go:194 (commandGroups).

Problem: adding a command touches four sites linked only by string command names, with no compile-time link between the CLI command, the dispatch case, and the DTO. Drift (a CLI command with no daemon case, or a mistyped command string) fails only at runtime as unknown command.

Direction: replace the hand-maintained string switch with a registration table keyed by a shared command constant, so the CLI command name, the dispatch entry, and the DTO reference the same symbol. The table is a concrete map of command constant to handler func, not a plugin framework. Keep dispatch behaviour identical; the goal is to remove silent string drift and give one source of truth for the command set.

### R5 (Medium, M3): Split the monolithic contract file by concern

Location: internal/ipc/protocol.go (~380 lines, 30+ DTOs).

Problem: every new command adds DTOs to one growing file that aggregates unrelated command schemas, hurting navigability.

Direction: keep all wire types in the internal/ipc package (it must remain the shared contract leaf), but split the file by concern: core protocol (Request, Response, constructors, shared helpers) plus per-group DTO files mirroring the command groups (navigation, observation, interaction, etc.). This is structural cohesion only; no type names, JSON tags, or wire semantics change.

### R6 (Medium, M4): Remove the global-flag reset footgun for REPL reuse

Location: internal/cli/root.go:69-76 (package-level Debug, JSONOutput, NoColor), internal/cli/root.go ExecuteArgs (manual flag-set walk and global reset), internal/cli/client.go:45 (package-level execFactory).

Problem: REPL command reuse depends on ExecuteArgs manually resetting package-level globals after each invocation. A newly added global persistent flag that is not added to the reset path leaks across REPL commands. The coupling between adding a flag and remembering the reset is a recurring, documented footgun.

Direction: make correct reset structural rather than remembered. Prefer reading flag values from the parsed cobra command at point of use and deriving per-invocation state from that, so there is no long-lived global to reset; or, if globals must remain, drive the reset from a single registered list of resettable flags so adding a flag cannot bypass it. The acceptance test is that introducing a new global persistent flag requires no edit to a separate reset routine to stay correct across REPL calls.

### R7 (Medium, M5): Surface dev-server hot-reload failures

Location: internal/daemon/handlers_serve.go (OnReload wired to handleServerReload, which runs CDP Page.reload in a goroutine; failures logged, not surfaced).

Problem: a reload triggered by a file change runs fire-and-forget. When it cannot succeed (no active session, browser disconnected), the failure is silent to the user watching for the reload.

Direction: keep the watch loop non-blocking, but make reload outcomes observable. Fail loud on the daemon side: emit a classified notice through the daemon's existing logging/notification path (the REPL notification channel already exists) so a failed reload is visible, and ensure a missing precondition (no session, browser gone) produces a clear message rather than a swallowed error. Do not block the file-watcher goroutine on the reload result.

### R8 (Low, L1): Isolate CDP event handlers from the read loop

Location: internal/cdp/client.go (event dispatch invokes subscribed handlers synchronously inside the single read-loop goroutine).

Problem: a slow or blocking event handler stalls all subsequent CDP response and event processing. Currently benign because daemon handlers push to buffers quickly, but it couples handler execution time to protocol throughput.

Direction: decouple handler execution from the read loop so a slow handler cannot stall response delivery. Preserve event ordering for a given subscription. Keep the change minimal and within internal/cdp; do not leak the concurrency model to callers.

### R9 (Low, L2): Do not silently drop CDP responses

Location: internal/cdp/client.go (response dispatch uses a non-blocking send with a default branch).

Problem: a response arriving after its caller has gone (timeout or cancel) is dropped with no log, removing observability when something times out unexpectedly.

Direction: when a response cannot be delivered to a waiter, record it (debug-level log or equivalent) rather than discarding it silently. Errors and edge cases get the same care as the happy path. Do not change the timeout/cancel semantics themselves.

### R10 (Low, L3): Tighten the browser discovery boundary

Location: internal/browser/target.go (exported FetchTargets, FetchVersion, FindPageTarget), called directly by internal/daemon in addition to the Browser methods that wrap them.

Problem: exporting the package-level HTTP discovery helpers dilutes the "Browser encapsulates discovery" boundary by exposing the discovery endpoints as standalone functions.

Direction: prefer routing daemon discovery through Browser methods and unexporting the helpers, so discovery details stay inside the package. If a genuine caller needs standalone discovery without a Browser instance, keep that one path exported and document why; unexport the rest. Confirm no import cycle is introduced.

### R11 (Low, L4): Add IPC protocol version safety

Location: internal/ipc/protocol.go (Request/Response carry no schema version).

Problem: because webctl is one binary, CLI and daemon normally share a build, but upgrading the binary while an old daemon keeps running can produce a schema mismatch with no guard. The protocol already shows compat care (omitempty throughout, explicitly deprecated fields), so the one realistic mismatch window is unguarded.

Direction: add a protocol version constant to the contract and include it on requests (or on the initial handshake). The daemon rejects or warns clearly on a mismatch, pointing the user to restart the daemon. Keep the check cheap and backward-tolerant; the goal is a clear message instead of a confusing decode failure when a stale daemon is hit after an upgrade.

## Constraints

- Pure Go, standard library and existing go.mod dependencies only. No cgo, no new heavy dependencies.
- Preserve all existing IPC wire semantics except where a finding explicitly changes them. R5 must not change any JSON tag or type shape. R11 adds a field but must remain tolerant of its absence.
- Preserve the Executor / ExecutorFactory write-once command design and the Direct-vs-IPC executor split.
- Keep internal/server free of browser, cdp, daemon, and ipc imports.
- Maintain the acyclic package graph. Any new sub-component types from R1/R2 live in internal/daemon (or a new internal subpackage) without creating cycles.
- Use the existing output and debug helpers; do not write to stdout/stderr directly or add ad-hoc logging styles.
- Format with gofmt; pass go vet and staticcheck (./test-runner lint).
- No Co-Authored-By trailers in commits. Use scoped commit messages (scope: description).

## Implementation Plan

1. R2 first as the lead extraction for R1: build the navigation/lifecycle coordinator that owns the seven sync.Maps with an explicit, documented lifecycle. Move the producer calls in events.go and consumer calls in handlers_navigation.go onto it. Confirm navigation, reload, back/forward, tab attach/detach, and ready behaviour via the relevant cli bash tests and daemon integration tests.
2. R1 continued: with rendezvous extracted, pull remaining cohesive concerns (dev-server control, buffer/observation state) off Daemon as warranted, leaving Daemon a coordinator. Land as focused steps, each compiling and green.
3. R4: replace the dispatch switch with a registration table keyed by shared command constants; align the CLI command names and commandGroups to the same constants.
4. R3: factor the decision-free CDP send/parse and timeout-policy duplication out of handlers, routing through the R1/R2 sub-component where natural.
5. R5: split protocol.go by concern within internal/ipc.
6. R6: remove the global-flag reset footgun (prefer deriving per-invocation state from the parsed cobra command).
7. R7: make dev-server reload outcomes observable through the existing notification path.
8. R8, R9: isolate CDP event-handler execution from the read loop and stop silently dropping undeliverable responses (both within internal/cdp).
9. R10: tighten the browser discovery boundary (unexport helpers, route through Browser methods).
10. R11: add the protocol version constant and stale-daemon mismatch handling.
11. After each finding, run the project checks for the touched packages (gofmt, go vet, ./test-runner go unit, and the relevant cli bash suite). Run ./test-runner ci after changes that touch command surface, daemon, or IPC.
12. After each finding passes its checks, mark it done in the Progress Tracking table below in the same commit as the fix.

Step 1 must precede step 2 (R2 is the first extraction of R1). Steps 3-10 are independent of each other and can proceed in any order after step 2, though the listed order minimises churn.

## Progress Tracking

Update this table as the single source of truth for remediation progress. After a finding passes its checks (plan step 11), set its Status to Done, fill the Commit column with the short SHA, and include this table edit in the same commit as the fix. Use Skipped (with a one-line reason in Notes) only if a finding is withdrawn after re-evaluation. Do not mark a finding Done until its acceptance criterion holds.

Status values: Todo, In progress, Done, Skipped.

| Finding | Title | Status | Commit | Notes |
|---------|-------|--------|--------|-------|
| R1 | Decompose the Daemon god-object | Todo | | |
| R2 | Encapsulate navigation rendezvous | Done | | Implemented here: Navigation milestones plus per-session tracker (internal/daemon/navigation.go); network and tab rendezvous rehomed to SessionManager. Deferred follow-up tracked in 02-failed-navigate-leaves-a-stale-in-flight-navigation.md. |
| R3 | Reduce CDP coupling in handlers | Todo | | |
| R4 | Explicit command registration | Todo | | |
| R5 | Split the contract file | Todo | | |
| R6 | Remove global-flag reset footgun | Todo | | |
| R7 | Surface hot-reload failures | Todo | | |
| R8 | Isolate CDP event handlers | Todo | | |
| R9 | Stop silent response drops | Todo | | |
| R10 | Tighten discovery boundary | Todo | | |
| R11 | IPC protocol version safety | Todo | | |
| R12 | Attribute lifecycle milestones by loaderId | Todo | | Deferred from 02. domContentEventFired/loadEventFired carry no loaderId, so under rapid back-to-back navigations on one session a straggler event can close the successor's milestone early. 02 bounds this (matches today's sessionID-keyed behaviour) but does not close it. Principled fix: correlate Page.lifecycleEvent loaderId with Page.navigate/frameNavigated. Low priority for sequential single-agent use. |

The project is complete when every row is Done or Skipped and the Acceptance Criteria below hold.

## Implementation Guidance

- Treat the seven navigation/lifecycle maps as one subsystem. The win in R2 is making the ordering invariants enforced by a type, not by comments; if a comment is still required to explain why an operation must precede another, the encapsulation is incomplete.
- For R4, the registration table is a concrete map plus shared constants, not a plugin system. Resist generalising beyond removing string drift.
- For R6, the strongest outcome is no long-lived global to reset at all. Only fall back to a registered resettable-flag list if removing the globals proves disproportionate.
- Keep comments to genuine why-not-what. Do not annotate the refactors with narrative or ticket references.
- Each finding is independently committable. Prefer a sequence of focused, scoped commits over one large change, especially for R1/R2.

## Acceptance Criteria

1. R1: the Daemon struct no longer owns the navigation rendezvous maps directly; handlers depend on extracted sub-components rather than reaching into unrelated Daemon fields. The daemon and CDP integration tests pass.
2. R2: navigation rendezvous state is owned by a single type with named lifecycle operations; events.go and handlers_navigation.go interact with it only through those operations, with no standalone sync.Map coordination left in Daemon for navigation/load/attach/network/tab waiters.
3. R3: the duplicated CDP send-parse-timeout mechanics are factored to a single place; handlers no longer each define their own context/timeout boilerplate for the common case.
4. R4: dispatch is driven by a registration table keyed by shared command constants; the CLI command name, dispatch entry, and group entry reference the same constants, and adding a command no longer relies on matching free-form strings across files.
5. R5: protocol.go is split into a core file plus per-group DTO files within internal/ipc, with no change to any JSON tag, type name, or wire shape.
6. R6: adding a new global persistent flag does not require editing a separate reset routine to remain correct across REPL invocations; REPL flag-bleed tests pass.
7. R7: a hot-reload that cannot succeed produces a visible, classified message through the daemon's notification path; the file-watcher goroutine is not blocked on the reload result.
8. R8: a deliberately slow event handler does not stall CDP response delivery in a targeted test; event ordering per subscription is preserved.
9. R9: an undeliverable response is logged rather than silently dropped.
10. R10: the browser discovery helpers are no longer exported except where a documented standalone caller requires it; the daemon obtains discovery data through Browser methods; the package graph remains acyclic.
11. R11: a request from a newer client to a stale daemon (or vice versa) yields a clear version-mismatch message advising a daemon restart, while a request carrying no version is still tolerated.
12. gofmt, go vet, staticcheck, the go unit suite, and the cli bash suite pass; ./test-runner ci passes after the command-surface/daemon/IPC changes.
