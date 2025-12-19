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

- [ ] BUG-003 fixed: `html` command works reliably after navigation
- [ ] `navigate` → `ready` → `html` sequence works consistently
- [ ] `navigate` → `html` (without ready) gives clear error or waits appropriately
- [ ] Rapid `navigate` → `navigate` doesn't cause crashes or hangs
- [ ] All commands return sensible errors during navigation (not timeouts)
- [ ] Documented the CDP event patterns discovered
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
