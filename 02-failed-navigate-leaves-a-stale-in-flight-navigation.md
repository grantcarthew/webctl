# Failed navigate leaves a stale in-flight navigation

Source: pre-commit review on 2026-06-24
Severity: info (latent correctness; pre-existing, surfaced by the R2 navigation refactor)
Category: Error Handling
Location: internal/daemon/handlers_navigation.go (handleNavigate, handleReload, navigateHistory)

## Goal

Ensure a navigation command that fails to start does not leave a phantom in-flight navigation in the per-session tracker, so a later ready default-mode call does not block until its timeout waiting for a page load that will never happen.

## Scope

In scope:
- The three daemon navigation handlers that call navTracker.begin and can return an error before any lifecycle milestone is reached: handleNavigate, handleReload, navigateHistory.
- A navTracker (and, if needed, Navigation) operation that cleanly ends a navigation that never started, waking any already-blocked ready consumer with a truthful, non-error outcome.
- Unit tests, gated to the existing -short convention and requiring no Chrome.

Out of scope:
- The two deliberate behaviour changes from the R2 refactor: ready default mode returning at DOM-ready, and the superseded-navigation error for --wait commands. Preserve both.
- Any change to IPC wire types, JSON tags, or command output on success paths.
- The sessionID-keyed straggler-event limitation documented in the R2 design (milestone attribution by loaderId). Not this project.
- The network-enable and tab attach/detach rendezvous. Untouched.

## Current State

webctl runs a persistent daemon holding a CDP WebSocket. internal/daemon/navigation.go defines a Navigation value type whose milestones (DOMReady, Loaded, FrameNavigated, Cancelled) are broadcasts: a channel closed exactly once when reached. A navTracker maps sessionID to the current Navigation. cancelReason distinguishes why a navigation was cancelled: cancelSuperseded (a newer navigation replaced it) or cancelDetached (the session detached).

Consumers:
- waitForLoadEvent (ready default mode) reads current(sessionID); if nil it returns immediately ("nothing to wait for"), otherwise it awaits DOMReady, re-binding on a supersession and returning a session-closed error on a detach.
- The four --wait navigation commands await their milestone or Cancelled; on Cancelled they return errNavigationSuperseded, or a session-closed error when the reason is cancelDetached.

The defect. Each of handleNavigate, handleReload, and navigateHistory calls navTracker.begin(sessionID) unconditionally (before any --wait check, by design, so a later ready can detect an in-flight navigation). They then issue the CDP navigate/reload/history command and can return early on two failure conditions:
- the CDP send returns an error, or
- Chrome returns a non-empty errorText in the navigate response (handleNavigate).

On these returns the just-created Navigation is left as the session's current navigation. No lifecycle event will ever close its DOMReady milestone because no navigation occurred. A subsequent webctl ready in default mode reads that Navigation and blocks on DOMReady until the full timeout elapses, then returns "timeout waiting for page load" even though nothing is loading. The stale entry self-heals only when the next begin for that session supersedes it.

This is pre-existing behaviour. The legacy implementation left the navigating sync.Map entry in place on the same error paths, so the R2 refactor preserved it rather than introducing it. It is recorded here because the new tracker model makes a clean fix natural, and the constraint that R2 preserve command behaviour is why it was not fixed inside that change.

No clean primitive exists today to end a navigation that never started. cancelDetached would wake a blocked ready with a misleading "session closed" error; cancelSuperseded would wake a --wait consumer with the supersession error. Neither is truthful for "the navigation failed to start", so a dedicated path is required.

## Requirements

1. After a navigation command returns early because the start failed (CDP send error, or a non-empty errorText), the session must have no in-flight navigation that a later ready default-mode call would wait on. A ready issued afterward must return promptly with success, exactly as it does when no navigation is in flight.

2. Any ready default-mode consumer already blocked on the failed navigation when the start fails must wake promptly with a non-error outcome (the navigation never happened, so the page is in whatever state it already held). It must not receive the supersession message or the session-closed message.

3. The failure must not be misattributed. Introduce a distinct cancellation cause for "navigation aborted before it started" rather than reusing cancelSuperseded or cancelDetached. A woken consumer must be able to tell this cause apart from supersession and detach.

4. The fix applies to all three handlers (handleNavigate, handleReload, navigateHistory) on every early-return failure path that follows begin. A --wait command whose own start fails continues to return its existing start-failure error to the caller; the tracker cleanup is in addition to, not a replacement for, that error response.

5. The cleanup must be safe under concurrency: it runs on a request goroutine while read-loop producers may be marking milestones and other request goroutines may be calling begin or current. Reuse the existing mutex discipline; do not introduce a new lock-ordering path.

6. Unit tests, no Chrome, consistent with -short:
   - A failed-start path clears the session's navigation so current() returns nil and a following waitForLoadEvent returns immediately with success.
   - A ready consumer blocked on a navigation that is then aborted returns with success and not an error, and not the supersession or session-closed message.
   - The abort cause is distinct from cancelSuperseded and cancelDetached.

## Constraints

- Pure Go, standard library and existing go.mod dependencies only. No cgo, no new dependencies.
- Preserve all behaviour the R2 refactor established, including the two deliberate changes (ready returns at DOM-ready; --wait commands return the supersession error) and the at-most-once network-enable and attach-dedup guarantees.
- No change to any IPC wire type, JSON tag, or success-path command output.
- Keep the Navigation and navTracker types free of the daemon logger and of CDP access; log at call sites with the existing debugf helper.
- Maintain the acyclic package graph; the new operation lives in internal/daemon.
- Format with gofmt; pass go vet and staticcheck (./test-runner lint). Scoped commit messages; no Co-Authored-By trailers.

## Implementation Plan

1. Add an aborted cancellation cause to navigation.go and a navTracker operation that ends a session's current navigation as aborted: cancel it with the new cause so blocked waiters wake, and remove it from the map so current() returns nil afterward. Guard the map mutation and the cancel under the existing locks, matching the t.mu to nav.mu ordering already used by begin and clear. Make it a no-op when the session has no tracked navigation, or when the tracked navigation is not the one being aborted (a later begin may already have superseded it; do not clobber the newer navigation).

2. Map the new cause in the consumers:
   - waitForLoadEvent treats the aborted cause as success (return nil): the navigation did not happen, so there is nothing to wait for.
   - The --wait navigation commands map the aborted cause to a clear, non-supersession error if it can reach them; in normal flow the aborting handler is the same goroutine and has already returned its start-failure error, so this is a defensive mapping rather than a primary path.

3. Call the new operation on every post-begin early-return failure path in handleNavigate (send error and errorText), handleReload (send error), and navigateHistory (send error), passing the navigation begun in that handler so a superseding navigation is never aborted by accident.

4. Add the unit tests from Requirement 6 alongside the existing navigation_test.go and session_rendezvous_test.go suites.

5. Run gofmt, go vet, staticcheck, ./test-runner go unit and go race. Where Chrome is available, run the navigation, reload, and history CLI bash suites to confirm success paths and the existing timeout/superseded behaviours are unchanged.

## Acceptance Criteria

1. After a navigate, reload, or history command fails to start, current(sessionID) returns nil and a following ready default-mode call returns immediately with success rather than blocking until timeout.
2. A ready consumer blocked on a navigation that is then aborted returns with success, carrying neither the supersession nor the session-closed message.
3. The aborted cancellation cause is distinct from cancelSuperseded and cancelDetached, verified by a unit test.
4. Aborting a failed start never cancels or removes a newer navigation created by a later begin for the same session.
5. The two R2 behaviour changes and the existing timeout and superseded-navigation outcomes for --wait commands remain unchanged.
