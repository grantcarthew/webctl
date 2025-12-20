# P-011: CDP Navigation & Page Load Debugging

- Status: In Progress
- Started: 2025-12-19

## Overview

Comprehensive debugging project to investigate and fix issues with CDP page load detection, navigation timing, and DOM operations. Multiple bugs have been identified where CDP operations fail or timeout during/after page navigation.

This project focuses on understanding the root causes through research and experimentation before implementing fixes. The issues are interconnected and require a systematic approach.

## IMPORTANT: Research Before Fixing

Before implementing any fixes in this project:

1. Reproduce the issue consistently
2. Add diagnostic logging to understand the sequence of events
3. Study how reference implementations handle the same scenario (Rod, Puppeteer, Playwright)
4. Understand the CDP event flow and timing
5. Document findings before writing fix code
6. Test fixes in isolation before integrating

Rushing to fix without understanding will create new bugs. Take time to understand the problem fully.

## IMPORTANT: Autonomous Debugging

Work autonomously without asking the user to run commands or test. The agent should:

1. Start the daemon using `./webctl start` or `go run ./cmd/webctl start`
2. Run navigation and observation commands directly
3. Add diagnostic logging and rebuild as needed
4. Test various scenarios and edge cases
5. Fix issues and verify fixes work

Only involve the user for final validation once the bug is confirmed fixed.

## Navigation & Interaction Commands Reference

These commands were implemented in P-008. Use them for testing:

Navigation:
```bash
webctl navigate <url>     # Navigate to URL, wait for frameNavigated
webctl reload             # Reload page (--ignore-cache for hard reload)
webctl back               # Go to previous page in history
webctl forward            # Go to next page in history
webctl ready              # Wait for page load (checks readyState first)
```

Interaction:
```bash
webctl click <selector>                    # Click element
webctl focus <selector>                    # Focus element
webctl type [selector] <text>              # Type text (--key Enter, --clear)
webctl key <key>                           # Send key (--ctrl, --alt, --shift, --meta)
webctl select <selector> <value>           # Select dropdown option
webctl scroll <selector>                   # Scroll element into view
webctl scroll --to x,y                     # Scroll to position
webctl scroll --by x,y                     # Scroll by offset
```

Observation:
```bash
webctl status             # Show daemon status and active session
webctl console            # Show console logs
webctl network            # Show network requests
webctl screenshot         # Capture screenshot
webctl html               # Get full page HTML
webctl html <selector>    # Get element HTML
webctl target             # List browser sessions
webctl target <id>        # Switch to session
```

Test Sequences to Debug:
```bash
# Sequence 1: Basic navigation + html
navigate https://example.com
ready
html

# Sequence 2: Navigation without ready
navigate https://example.com
html                      # Should this wait? Currently may timeout

# Sequence 3: Rapid navigation
navigate https://example.com
navigate https://google.com
html

# Sequence 4: Back/forward
navigate https://example.com
navigate https://google.com
back
html
forward
html
```

## Goals

1. Understand CDP page lifecycle events and their timing guarantees
2. Fix BUG-003: HTML command slow/timeout during navigation
3. Ensure all observation commands work reliably after navigation
4. Document CDP event patterns for future reference
5. Establish reliable page-ready detection pattern

## Known Issues

### BUG-003: HTML command extremely slow and times out in REPL

Symptom: When running `webctl html` in REPL after navigating to a page, DOM.getDocument takes ~10-12 seconds or times out entirely.

Root Cause (Identified): `DOM.getDocument` blocks until the page's DOM is ready. If called during navigation (before page load completes), Chrome blocks the call. If navigation happens DURING the call, Chrome never responds.

Approaches Already Tried:
1. `Page.loadEventFired` / `Page.domContentEventFired` events - didn't fire reliably for some pages
2. `Page.setLifecycleEventsEnabled` + `Page.lifecycleEvent` - works but some navigations still timeout
3. Navigation-aware context cancellation - helps but doesn't solve core timing issue
4. Polling `document.readyState` - fails with "client closed" during navigation
5. JavaScript Promise-based wait (Rod's approach) - currently being tested

Key Learnings:
- CDP lifecycle events require `Page.setLifecycleEventsEnabled` to be called first
- `DOM.getDocument` blocks until DOM is ready - it's not instantaneous
- Puppeteer uses `LifecycleWatcher` with frame-level tracking
- Rod uses JS Promise approach with `awaitPromise: true`
- "client closed" error often means CDP call sent to invalid/stale session

### Related: Navigation Event Timing

The `navigate`, `reload`, `back`, `forward` commands wait for `Page.frameNavigated`. The `ready` command waits for `Page.loadEventFired` (with readyState check first). These work but may have edge cases during rapid navigation.

## Scope

In Scope:
- Debugging and fixing BUG-003 (HTML command)
- Verifying all navigation commands work correctly
- Verifying all observation commands work after navigation
- Adding diagnostic logging as needed
- Documenting CDP event patterns
- Creating test cases for edge conditions

Out of Scope:
- New feature development
- Performance optimization (beyond fixing timeouts)
- Iframe support (documented v1 limitation)

## Research Resources

Local documentation in `./context/` directory:

| Directory | Description |
|-----------|-------------|
| `context/rod/` | Go library for CDP browser automation - reference implementation |
| `context/puppeteer/` | Node.js browser automation - reference implementation |
| `context/devtools-protocol/` | Chrome DevTools Protocol specification and JSON schemas |
| `context/docs/` | Fetched CDP domain documentation (Runtime, Network, Page, DOM, Input) |

Key files to study:
- `context/rod/page_navigate.go` - How Rod handles navigation
- `context/rod/element.go` - How Rod waits for elements
- `context/puppeteer/src/cdp/LifecycleWatcher.ts` - Puppeteer's lifecycle handling
- `context/devtools-protocol/pdl/domains/Page.pdl` - Page domain events and methods
- `context/devtools-protocol/pdl/domains/DOM.pdl` - DOM domain methods

## Success Criteria

- [ ] Implement `Target.setDiscoverTargets{Discover: true}` in daemon init
- [ ] Implement `Target.attachToTarget{Flatten: true}` for session management
- [ ] BUG-003 fixed: `html` command returns in <1 second (not 10+ seconds)
- [ ] `navigate` → `html` works instantly without waiting for `networkIdle`
- [ ] `navigate` → `ready` → `html` sequence works consistently
- [ ] Rapid `navigate` → `navigate` doesn't cause crashes or hangs
- [ ] All commands return sensible errors during navigation (not timeouts)
- [ ] Documented the CDP session management patterns
- [ ] Final validation with user confirms all fixes work correctly

## Deliverables

- Fixed `handleHTML` in `internal/daemon/daemon.go`
- Potentially updated navigation event handling
- Documentation of CDP event patterns (in this file or separate doc)
- Test cases for navigation edge conditions

## Technical Approach

Phase 1: Diagnosis
1. Add verbose diagnostic logging to track CDP event flow
2. Reproduce issues consistently with specific test URLs
3. Identify exact sequence of events leading to failure
4. Map out the timing relationships between events

Phase 2: Research
1. Study Rod's implementation of page navigation waiting
2. Study Puppeteer's LifecycleWatcher implementation
3. Identify patterns that work reliably across implementations
4. Document the differences and trade-offs

Phase 3: Implementation
1. Implement fix based on research findings
2. Test with various page types (static, SPA, slow-loading)
3. Test edge cases (navigation during operation, rapid navigation)
4. Verify all observation commands work

Phase 4: Documentation
1. Document the fix and why it works
2. Update DR-013 if navigation behavior changes
3. Create test cases for regression prevention
4. Close BUG-003 in P-007

Phase 5: Final User Validation
1. Notify user that fixes are ready for testing
2. User performs manual testing of all navigation sequences
3. User confirms fixes work in real-world usage
4. Mark project complete after user sign-off

## Files to Investigate

- `internal/daemon/daemon.go` - handleHTML, handleFrameNavigated, handleLoadEventFired
- `internal/daemon/session.go` - Session state tracking (loaded, loadCh, navCancel)
- `internal/cdp/client.go` - CDP event handling

## Notes

The core challenge is that CDP operations can fail in subtle ways during page transitions:
- Session IDs can become invalid
- Execution contexts get destroyed
- DOM nodes get invalidated
- Events may or may not fire depending on navigation type

Understanding these failure modes is key to building robust solutions.

## Updates

- 2025-12-19: Project created to systematically debug CDP navigation issues
- 2025-12-19: Session progress:
  - **Reproduced BUG-003**: `html` command times out (30s) when called immediately after `navigate` to complex pages like google.com
  - **Root cause confirmed**: `Runtime.evaluate` blocks during page load, regardless of JavaScript content
  - **Researched Rod's approach**: Uses Promise-based JavaScript with `awaitPromise: true` in `Runtime.evaluate`. Key function is `WaitLoad` which returns a Promise that resolves when `document.readyState === 'complete'` or on `window.onload` event
  - **Implemented fix attempt**: Modified `handleHTML` in `daemon.go:992-1153` to use Promise-based JavaScript with `awaitPromise: true`. The Promise waits for readyState complete or load event before extracting HTML
  - **Fix status**: Initial test showed 0.55s (success), but subsequent test showed 30s timeout. The fix may work for cached pages but not for fresh navigations. **Further investigation needed.**
  - **Key insight from testing**: The Promise-based approach may still block if `Runtime.evaluate` itself is blocked by Chrome during early navigation phase. May need to wait for navigation to complete at the daemon level before calling any Runtime methods.
  - **Next steps**:
    1. Add diagnostic logging to see when Runtime.evaluate is called vs when frameNavigated fires
    2. Consider waiting for Page.loadEventFired before allowing html command
    3. Or add retry logic similar to Rod's `Evaluate()` which retries on `ErrCtxNotFound`
- 2025-12-19: Continued debugging session:
  - **Implemented navigation state tracking**: Added `navigating sync.Map` to daemon to track sessions in navigation
  - **Navigation flow**: `handleNavigate/handleReload/navigateHistory` set navigating channel, `handleLoadEventFired` closes it
  - **handleHTML now waits for load**: Before calling `Runtime.evaluate`, checks if navigation is in progress and waits for `Page.loadEventFired`
  - **Key finding**: `Runtime.evaluate` ALWAYS blocks until `Page.loadEventFired` fires, regardless of JavaScript content (even simple `document.documentElement.outerHTML`)
  - **Fix verified**: Once `Page.loadEventFired` fires, JavaScript execution is instant (1ms)
  - **Environment issue identified**: On test machine, `Page.loadEventFired` takes ~15 seconds for simple data URLs - this is abnormal browser behavior, not a webctl issue
  - **Files changed**: `internal/daemon/daemon.go` - added `navigating` sync.Map, updated `handleNavigate`, `handleReload`, `navigateHistory`, `handleLoadEventFired`, `handleHTML`
  - **Status**: Fix implemented and tested. BUG-003 is addressed - html command now properly waits for page load before extracting HTML
- 2025-12-20: Deep investigation into Runtime.evaluate blocking behavior:
  - **CRITICAL DISCOVERY**: Previous fix was incorrect - `Runtime.evaluate` does NOT wait for `loadEventFired`, it waits for `Page.lifecycleEvent: name=networkIdle`
  - **Timeline analysis with timestamps**:
    - `DOMContentLoaded` fires when DOM is parsed and ready (~4s after navigation)
    - `loadEventFired` fires when page resources loaded (~4s after navigation)
    - `networkIdle` fires when no network activity for ~500ms (~14s after navigation due to slow favicon)
    - `Runtime.evaluate` completes at EXACT same millisecond as `networkIdle` (not `loadEventFired`)
  - **Test 1 - Runtime.evaluate timing**:
    - Called `Runtime.evaluate` before `DOMContentLoaded`
    - Waited 10 seconds after `loadEventFired` for `networkIdle`
    - Total time: 14-18 seconds for simple example.com page
    - Favicon 404 request takes 14-15 seconds, blocking `networkIdle`
  - **Test 2 - Calling Runtime.evaluate AFTER DOMContentLoaded**:
    - Modified code to wait for `DOMContentLoaded` before calling `Runtime.evaluate`
    - `Runtime.evaluate` STILL blocked 10 seconds until `networkIdle`
    - **Conclusion**: Chrome enforces `networkIdle` wait regardless of when you call `Runtime.evaluate`
  - **Test 3 - DOM.getDocument approach**:
    - Tried using `DOM.getDocument` + `DOM.getOuterHTML` instead of `Runtime.evaluate`
    - `DOM.getDocument` ALSO waits for `networkIdle` (10+ seconds)
    - `DOM.getOuterHTML` is instant (2ms) once you have nodeId
    - **Conclusion**: `DOM.getDocument` has same blocking behavior as `Runtime.evaluate`
  - **Test 4 - Direct DOM.getOuterHTML with nodeId=1**:
    - Attempted to skip `DOM.getDocument` and call `DOM.getOuterHTML(nodeId: 1)` directly
    - STILL waited 10 seconds for `networkIdle`, then returned error "Could not find node with given id"
    - **Conclusion**: ALL CDP calls block until `networkIdle`, even failing calls
  - **Test 5 - Rod comparison**:
    - Created test program using Rod library to navigate to example.com and get HTML
    - **Rod retrieves HTML in 73 milliseconds** (not 10+ seconds!)
    - **CRITICAL FINDING**: Rod does NOT experience the `networkIdle` blocking delay
    - Rod uses `DOM.getOuterHTML{ObjectID: ...}` with ObjectID from Runtime.RemoteObject (not nodeId)
    - **Question**: How does Rod avoid the `networkIdle` wait that we're seeing?
  - **Key discoveries**:
    1. `Runtime.evaluate` blocks until `Page.lifecycleEvent: name=networkIdle` (NOT `loadEventFired`)
    2. `DOM.getDocument` also blocks until `networkIdle`
    3. ALL CDP method calls block until `networkIdle`, regardless of method or parameters
    4. Slow network resources (like favicon 404s) delay `networkIdle` by 10+ seconds
    5. Rod successfully extracts HTML in <100ms without this blocking behavior
  - **Added comprehensive debug logging**:
    - All debug messages now include timestamps
    - Added logging for all Page lifecycle events (frameStartedLoading, frameStoppedLoading, lifecycleEvent)
    - Added logging for Runtime execution context events (contextCreated, contextDestroyed, contextsCleared)
    - Added logging for DOM events (documentUpdated)
    - Added logging for all Network events (requestWillBeSent, responseReceived, loadingFinished, loadingFailed)
  - **Status**: Root cause identified but solution unclear. Need to understand how Rod avoids `networkIdle` blocking.
- 2025-12-20: ROOT CAUSE IDENTIFIED - CDP Session Management Difference:
  - **Investigated Rod's source code** (local copy in `./context/rod/`)
  - **Test 6 - Rod timing breakdown**:
    - Rod's `MustNavigate()`: 18ms (returns immediately after navigation starts)
    - Rod's `MustHTML()` called immediately after navigate: 25ms (gets HTML instantly!)
    - Total time: ~100ms vs our 15+ seconds
  - **Test 7 - Tried Rod's exact ObjectID approach**:
    - Modified our code to use `Runtime.evaluate` for `document.documentElement` to get ObjectID
    - Then call `DOM.getOuterHTML{objectId: ...}` with that ObjectID
    - STILL blocked for 10+ seconds waiting for `networkIdle`
    - **Conclusion**: Using ObjectID vs nodeId is NOT the difference
  - **ROOT CAUSE DISCOVERED** - CDP session setup is fundamentally different:
    - **Rod's approach** (`context/rod/browser.go:273-276`):
      ```go
      session, err := proto.TargetAttachToTarget{
          TargetID: targetID,
          Flatten:  true, // if it's not set no response will return
      }.Call(b)
      ```
    - **Our approach**:
      - We DON'T call `Target.attachToTarget` ourselves
      - We DON'T use `Flatten: true`
      - We passively receive `Target.attachedToTarget` events from browser
      - We use sessionID from those events
  - **Key differences identified**:
    1. **Rod calls `Target.setDiscoverTargets{Discover: true}`** on browser connect (line 174)
    2. **Rod explicitly attaches to targets with `Flatten: true`** (line 273-276)
    3. Rod's comment: "if it's not set no response will return" - suggests `Flatten: true` is critical
    4. **We don't call either of these methods**
  - **Hypothesis**: Without `Flatten: true`, Chrome handles sessions differently and queues CDP responses until page reaches stable state (`networkIdle`). With `Flatten: true`, responses return immediately regardless of page state.
  - **Implementation plan**:
    1. Add `Target.setDiscoverTargets{Discover: true}` call in daemon initialization
    2. Refactor session attachment to actively call `Target.attachToTarget{Flatten: true}`
    3. Update session tracking to use the returned sessionID
    4. Test if this eliminates the `networkIdle` blocking behavior
  - **Files to modify**:
    - `internal/cdp/client.go` - May need to add Target domain methods
    - `internal/daemon/daemon.go` - Session initialization and attachment logic
    - `internal/daemon/session.go` - Session state management
  - **Status**: Root cause identified! Need to implement Rod's session attachment pattern with `Flatten: true`.

## Next Session Implementation Plan

**Goal**: Implement Rod's CDP session management pattern to eliminate `networkIdle` blocking.

**Step 1: Add Target.setDiscoverTargets**
- Location: `internal/daemon/daemon.go` in `New()` function after CDP client creation
- Add call: `Target.setDiscoverTargets{Discover: true}`
- This enables target discovery events from browser

**Step 2: Implement Active Target Attachment**
- Location: `internal/daemon/daemon.go` in target attachment handling
- Currently: We passively receive `Target.attachedToTarget` events
- Change to: Actively call `Target.attachToTarget{TargetID: ..., Flatten: true}`
- Use the returned sessionID for all operations on that target

**Step 3: Update Session Tracking**
- Location: `internal/daemon/session.go`
- Ensure sessions use the sessionID from `Target.attachToTarget` response
- May need to refactor how we track and manage sessions

**Step 4: Test**
- Build and test `navigate` → `html` sequence
- Verify HTML retrieval is <1 second (not 10+ seconds)
- Check debug logs show no `networkIdle` waiting

**Step 5: Verify Fix**
- Test all navigation sequences from project goals
- Test edge cases (rapid navigation, navigation during operations)
- Confirm all observation commands work reliably

**Reference Implementation** (from `./context/rod/`):
- `browser.go:174` - `proto.TargetSetDiscoverTargets{Discover: true}.Call(b)` in Connect()
- `browser.go:273-276` - `proto.TargetAttachToTarget{TargetID: targetID, Flatten: true}.Call(b)` in PageFromTarget()
- `browser.go:275` - Comment: "if it's not set no response will return" (critical!)
- `browser.go:313` - `page.EnableDomain(&proto.PageEnable{})` after attachment
- `states.go:59-65` - EnableDomain() implementation pattern
- `element.go:~2000` - HTML() method using `DOM.getOuterHTML{ObjectID: ...}`

**Key Rod differences**:
1. Active target attachment (line 273) vs our passive event listening
2. Flatten: true parameter (line 275) vs our missing parameter
3. SetDiscoverTargets on connect (line 174) vs our missing call
4. EnableDomain pattern (line 313) for Page domain
