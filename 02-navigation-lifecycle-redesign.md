# Navigation Lifecycle Rendezvous Redesign (R2)

## Goal

Replace the seven ad-hoc sync.Map fields on the Daemon type that coordinate navigation, page load, target attachment, network enablement, and tab attach/detach with a principled model. A navigation becomes a first-class value with milestone broadcasts owned by a per-session tracker; per-session state moves onto the session; tab and attach concerns move to their rightful owners. This resolves architecture review finding H2 (project finding R2) and is the lead extraction for R1.

## Scope

In scope:
- Introduce a Navigation value type with milestone broadcasts and a per-session tracker that owns navigation/load/frameNavigated rendezvous.
- Move the per-session networkEnabled flag onto the session inside SessionManager.
- Move tab attach/detach rendezvous onto the owner of session lifecycle (SessionManager) rather than bare maps on Daemon.
- Move target-attach deduplication (attachedTargets) onto a small dedicated set owned by the attach logic.
- Remove all seven sync.Map fields from the Daemon struct and rewire every call site to the new owners.
- Stop performing a CDP request inside the read loop when handling Page.frameNavigated; resolve the page title in the consumer goroutine instead.

Out of scope:
- R1's remaining decompositions (dev-server control, buffer/observation state). This project performs only the navigation/lifecycle extraction. Do not refactor those concerns here, though the owners introduced here are intended for R1 to build on.
- Findings R3 through R11.
- Any change to IPC wire types, JSON tags, or command semantics. This is internal daemon refactoring only.
- The Executor / ExecutorFactory write-once command design. Preserve it.
- Changing the ready selector/eval/network-idle polling modes. Only the default page-load mode interacts with the navigation tracker.

## Current State

webctl is a browser-automation CLI plus a persistent daemon that holds a Chrome DevTools Protocol (CDP) WebSocket. A single CDP read-loop goroutine in internal/cdp demultiplexes events and invokes subscribed handlers. The daemon's event handlers (producers) run on that read loop; IPC request handlers (consumers) run on their own goroutines and await lifecycle events that will arrive later through the read loop. The coordination between them currently lives in seven sync.Map fields on the Daemon struct.

### The seven maps

Declared in internal/daemon/daemon.go:88-109.

| Field | Key to value | Purpose |
|-------|--------------|---------|
| navWaiters | sessionID to chan *frameNavigatedInfo | Await main-frame Page.frameNavigated (history navigation) |
| loadWaiters | sessionID to chan struct{} | Await Page.loadEventFired (full load) |
| navigating | sessionID to chan struct{} | Presence marks a session as in-flight; cleared on first of domContentEventFired or loadEventFired |
| attachedTargets | targetID to bool | Deduplicate Target.attachToTarget calls |
| networkEnabled | sessionID to bool | Track whether Network.enable was sent for a session |
| tabAttachWaiters | targetID to chan struct{} | Await a new target's session attaching (tab new) |
| tabDetachWaiters | sessionID to chan struct{} | Await a session detaching (tab close) |

### Call sites

| Map | Producer (read loop) | Consumer / writer (request goroutine) |
|-----|----------------------|----------------------------------------|
| navWaiters | events.go handleFrameNavigated: LoadAndDelete, fetches title via CDP, non-blocking send | handlers_navigation.go navigateHistory: Store before send, Delete on error, defer Delete, select with timeout |
| loadWaiters | events.go handleLoadEventFired: LoadAndDelete, non-blocking send | handlers_navigation.go handleNavigate and handleReload: Store before send, Delete on error, defer Delete, select; waitForLoadEvent: Store, defer Delete, double-check |
| navigating | events.go handleLoadEventFired and handleDOMContentEventFired: LoadAndDelete then close | handlers_navigation.go handleNavigate, handleReload, navigateHistory: close old then store new; waitForLoadEvent: Load to test presence |
| attachedTargets | events.go handleTargetCreated: LoadOrStore, Delete on attach failure | daemon.go enableAutoAttach: LoadOrStore, Delete on attach failure |
| networkEnabled | n/a | daemon.go enableDomainsForSession: Store; handlers_navigation.go ensureNetworkEnabled: Load then Store; handlers_observation.go handleNetwork: LoadOrStore |
| tabAttachWaiters | events.go handleTargetAttached: Load, non-blocking send | handlers_tab.go handleTabNew: Store before checking GetByTargetID, defer Delete, select with timeout |
| tabDetachWaiters | events.go handleTargetDetached: Load, non-blocking send | handlers_tab.go handleTabClose: Store before send, defer Delete, select with timeout |

The frameNavigatedInfo type (URL and Title) is declared in daemon.go and carries the frameNavigated result to the waiter.

SessionManager (internal/daemon/session.go) already owns session identity: it maps sessionID to an internal session struct holding sessionID, targetID, url, title, plus the active session and attachment order. Add and Remove are the natural attach/detach points; they are called from events.go handleTargetAttached and handleTargetDetached.

### Implicit invariants and known hazards

The current correctness depends on rules documented only in comments, spread across two files:
- Register the waiter before sending the CDP command, so a fast event (especially a cached load) is not missed.
- waitForLoadEvent re-checks navigating after registering its waiter (double-check) to cover the race where the load completes during registration.
- frameNavigated is used for history navigation because Chrome's BFCache suppresses Page.loadEventFired for cached pages.
- navigating presence is cleared on the first of domContentEventFired or loadEventFired, so the ready default mode can return at DOM-ready without waiting for all resources.
- handleFrameNavigated resolves the page title with a synchronous CDP call on the read-loop goroutine. This is a latent read-loop stall (the architecture review's L1 concern surfacing inside the navigation path).
- The review notes a possible use-after-close on the navigating channel under rapid navigation, where a superseding navigation closes a channel another goroutine may still reference.

### Behavioral contract that must be preserved

These observable behaviors must survive the refactor unchanged. They are the acceptance contract; map the new milestones to them exactly.

1. navigate --wait and reload --wait block until Page.loadEventFired (full load), then return URL and title.
2. The ready command default mode (page load) waits only if a navigation this daemon initiated is in flight, and then only until DOM-ready (first of domContentEventFired or loadEventFired). If no navigation is in flight, it returns immediately. It retains the document.readyState == "complete" fast path that returns before consulting the tracker.
3. back --wait and forward --wait block until a main-frame Page.frameNavigated, returning the target URL and the resolved title.
4. tab new blocks until the new target's session is registered; tab close blocks until the session is removed. Both first check current SessionManager state to handle an event that already fired, then wait with a bounded timeout.
5. Network.enable is sent at most once per session, whether triggered at startup (enableDomainsForSession), lazily by network (handleNetwork), or by ready --network-idle (ensureNetworkEnabled).
6. Target.attachToTarget is issued at most once per targetID; a failed attach clears the mark so a retry can occur.

## References

- .start/reviews/2026-06-22-architecture-01.md: source architecture review. Finding H2 describes the overlapping concurrent maps and the implicit protocol; the Assessment section names a per-session navigation/lifecycle coordinator as the highest-leverage extraction.
- 01-architecture-review.md: the remediation project. Finding R2 is this work; this project supersedes R2's implementation direction with the design below. Constraints and Progress Tracking there apply.
- AGENTS.md: repository conventions for build, test, daemon/IPC contracts, and code style.

## Requirements

1. Navigation value type. Introduce a Navigation type (new file internal/daemon, for example coordinator.go or navigation.go) representing one in-flight or just-completed navigation for a session. It exposes named milestones as broadcasts, each a channel closed exactly once when the milestone is reached, so that awaiting an already-reached milestone returns immediately and multiple waiters are supported. The milestones are:
   - DOM-ready: reached on Page.domContentEventFired, and also on Page.loadEventFired (load implies DOM-ready). Closing it is idempotent.
   - Loaded: reached on Page.loadEventFired.
   - FrameNavigated: reached on a main-frame Page.frameNavigated; carries the navigated URL, readable after the milestone closes.
   - Cancelled: reached when this navigation is superseded by a new navigation for the same session, or when the session detaches.
   The producer-side mark operations and the close of each channel must be safe under concurrent calls (guarded or once-style), eliminating the use-after-close hazard by construction.

2. Per-session navigation tracker. Introduce a type that owns the mapping from sessionID to the current Navigation, with named operations: begin a navigation (atomically cancel and replace any prior navigation for that session), read the current navigation, and clear on detach. This single type replaces navWaiters, loadWaiters, and navigating. The read-loop producers reach the current navigation through it and mark milestones; the request-goroutine consumers begin a navigation or read the current one and await a milestone. No ordering comment should be required to explain correctness: registering interest is creating or reading the Navigation, and completion is a closed channel, so the register-before-fire race and the double-check disappear.

3. Consumer mapping. Wire the navigation handlers to milestones exactly per the behavioral contract:
   - handleNavigate (wait) and handleReload (wait) await Loaded.
   - waitForLoadEvent (ready default mode) awaits DOM-ready, and only when a navigation is currently in flight for the session.
   - navigateHistory (wait) awaits FrameNavigated.
   Each waiting consumer also selects on Cancelled and on its timeout, so a superseded navigation wakes the waiter with a clear superseded outcome rather than waiting out the timeout or receiving another navigation's event.

4. Title resolution off the read loop. handleFrameNavigated must not issue a CDP request on the read-loop goroutine. The producer records the navigated URL on the Navigation and closes the FrameNavigated milestone. The consumer, after waking, resolves the page title via the existing getPageTitle path on its own goroutine.

5. Network enablement as session state. Remove the networkEnabled map. Represent network enablement as a per-session flag on the SessionManager session, with operations to test and set it under the manager's existing lock. Route enableDomainsForSession, ensureNetworkEnabled, and handleNetwork through these operations, preserving the at-most-once guarantee.

6. Tab attach/detach via session lifecycle. Remove tabAttachWaiters and tabDetachWaiters from Daemon. Coordinate the new-tab and close-tab rendezvous through the owner of session lifecycle (SessionManager), keyed by targetID for attach and sessionID for detach. Add and Remove signal any registered waiter; the consumer first checks current state (GetByTargetID for attach, Get for detach) to cover an event that already fired, then awaits with the existing bounded timeout. Do not leave a standalone waiter map on Daemon.

7. Target-attach dedup as a set. Remove the attachedTargets map. Represent in-flight or completed attaches as a small dedicated set keyed by targetID, owned by the attach logic, with operations to mark on first attach (reporting whether this is the first) and to clear on failure. Route enableAutoAttach and handleTargetCreated through it.

8. Daemon struct cleanup. After the above, the Daemon struct holds references to the new owners (the navigation tracker, the attach set, and the existing SessionManager) and no longer declares any of the seven sync.Map fields. The frameNavigatedInfo type moves to wherever the Navigation type lives, or is replaced by fields on Navigation.

9. Tests. Add unit tests for the new types that do not require Chrome (gated consistent with the existing -short convention): Navigation milestone semantics including load-implies-DOM-ready and idempotent closes; tracker supersession waking a waiter via Cancelled; SessionManager network flag at-most-once; SessionManager attach/detach signalling including the already-fired fast path. A targeted test should demonstrate that awaiting a milestone reached before the await still returns promptly (the race that previously needed the double-check).

## Constraints

- Pure Go, standard library and existing go.mod dependencies only. No cgo, no new dependencies.
- No change to any IPC wire type, JSON tag, or command behavior. Verify the navigation, reload, history, tab, and ready command outputs are unchanged.
- Maintain the acyclic package graph. The new types live in internal/daemon. Do not add imports that create cycles.
- Use the existing debug helper (debugf) for any retained logging. Do not write to stdout/stderr directly or introduce a new logging style. The new value types should not depend on the daemon's logger; keep them pure and log at the call sites where it adds diagnostic value.
- Format with gofmt; pass go vet and staticcheck (./test-runner lint).
- Scoped commit messages (scope: description). No Co-Authored-By trailers.

## Implementation Plan

1. Add the Navigation type and its per-session tracker in a new file under internal/daemon. Define the four milestones as broadcasts with idempotent, concurrency-safe mark operations, the URL field for FrameNavigated, and tracker operations begin (cancel-and-replace), current, and clear. Encode load-implies-DOM-ready inside the mark operation. Write the unit tests for milestone semantics and supersession first or alongside.

2. Rewire the producers in events.go: handleLoadEventFired marks Loaded (which also closes DOM-ready) on the current navigation; handleDOMContentEventFired marks DOM-ready; handleFrameNavigated records the URL and closes FrameNavigated with no CDP call. Drop the frameNavigatedInfo title fetch from the read loop.

3. Rewire the consumers in handlers_navigation.go: handleNavigate and handleReload begin a navigation then await Loaded or Cancelled or timeout; navigateHistory begins a navigation then awaits FrameNavigated or Cancelled or timeout and resolves the title afterward; waitForLoadEvent reads the current navigation and awaits DOM-ready, returning immediately when none is in flight. Remove the manual map Store/Delete/close and the double-check comment.

4. Move networkEnabled onto SessionManager as a per-session flag with test/set operations. Update enableDomainsForSession, ensureNetworkEnabled, and handleNetwork. Remove the map.

5. Move tab attach/detach rendezvous onto SessionManager so Add and Remove signal registered waiters. Update handleTabNew and handleTabClose to register through SessionManager and keep the current-state fast path. Remove both maps.

6. Replace attachedTargets with the dedicated dedup set. Update enableAutoAttach and handleTargetCreated. Remove the map.

7. Delete the seven sync.Map fields and frameNavigatedInfo from the Daemon struct and initialize the new owners in New. Confirm the struct now exposes only coordinator references for this concern.

8. Run gofmt, go vet, staticcheck, and ./test-runner go unit and go race. Run the CLI bash suites that exercise navigation, reload, history, tab, and ready behavior, and the daemon integration tests where Chrome is available. Resolve anything they surface at the root, not by weakening a test.

9. In the same commit as the completed fix, set the R2 row in 01-architecture-review.md Progress Tracking to Done with the short commit SHA, and add a one-line note that R2 was implemented via this redesign (Navigation milestones plus per-session tracker; network and tab concerns rehomed to SessionManager).

Steps 1 through 3 are the core and must land together to keep the package compiling. Steps 4 through 6 are independent of each other and can land in any order after step 1. Step 7 follows the others.

## Implementation Guidance

- Model completion as a closed channel (broadcast), never as a one-shot send to a pre-registered waiter. This is the change that removes the register-before-send ordering rule and the double-check: receiving from a closed channel returns immediately, so "the event already happened" and "I am waiting for it" become one path.
- Separate state from rendezvous. networkEnabled is a fact about a session and belongs on the session. attachedTargets is attach-process bookkeeping and belongs to the attach logic. Only the genuine cross-goroutine awaiting belongs in the navigation tracker.
- Return milestone channels as receive-only from the Navigation type so consumers can await but cannot manage channel lifetime.
- Keep the new types free of the daemon's logger and of CDP access. The producer marks milestones with data already in the event; any follow-up CDP call (title) happens in the consumer. This keeps the read loop non-blocking and the new types unit-testable without a browser.
- Treat the title fetch moving off the read loop as a correctness improvement, not a behavior change: the navigate --wait path already resolves the title in the consumer, so history navigation should match it.
- If preserving the ready default-mode behavior (wait only when in flight, return at DOM-ready) reveals that the old code returned earlier or later than intended, preserve the observable behavior the bash tests assert and note any latent discrepancy rather than silently changing it.

## Acceptance Criteria

1. The Daemon struct declares none of navWaiters, loadWaiters, navigating, attachedTargets, networkEnabled, tabAttachWaiters, or tabDetachWaiters, and no replacement bare sync.Map for these concerns. Navigation rendezvous is owned by a single tracker type with named operations.
2. events.go and handlers_navigation.go interact with navigation, load, and frame-navigated rendezvous only through the Navigation tracker's named operations. No channel create/store/close/delete coordination for these remains inline in the handlers.
3. handleFrameNavigated issues no CDP request on the read-loop goroutine; the page title for history navigation is resolved in the consumer.
4. networkEnabled is a per-session property of SessionManager; tab attach/detach are coordinated through SessionManager; target-attach dedup is a dedicated set. None remain as fields on Daemon.
5. The six behavioral contract items hold, verified by the navigation, reload, history, tab, and ready CLI bash suites and, where Chrome is available, the daemon integration tests.
6. A unit test demonstrates that a milestone reached before a consumer awaits it still returns promptly, and that a superseded navigation wakes its waiter via Cancelled.
7. gofmt, go vet, staticcheck, ./test-runner go unit, and ./test-runner go race pass; ./test-runner ci passes.
8. The R2 row in 01-architecture-review.md Progress Tracking is set to Done with the commit SHA in the same commit as the fix.
