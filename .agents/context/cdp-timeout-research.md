# CDP Timeout Research Summary

Research conducted: 2026-01-24
Context: Verification of claims in test-interaction.sh health check comment block

## Research Methodology

1. Web search using Kagi for CDP timeout behavior and connection stability
2. Cloned ChromeDevTools/devtools-protocol repository to .ai/context/cdp-protocol
3. Examined Runtime.evaluate protocol definition
4. Searched Chromium issue tracker and automation library issue trackers
5. Reviewed Puppeteer, Playwright, and ChromeDP documentation and issues

## Claims Verification

### Claim 1: "CDP doesn't handle long-running evals well"

**Status**: VERIFIED ✅

Evidence:
- Runtime.evaluate timeout parameter is marked "experimental" in protocol definition
- Multiple reports of timeout issues with Runtime.evaluate across Puppeteer, Playwright
- Issue #124 on chrome-devtools-mcp: timeout issues when getting performance metrics
- Serialization bugs where timeout=0 sent unintentionally (Stack Overflow #77359657)
- Timeout parameter not always respected (Chromium issue #374093512)

### Claim 2: "Extended waits can cause CDP message queue backlog"

**Status**: VERIFIED ✅

Evidence:
- Lighthouse issue #6512: PROTOCOL_TIMEOUT when Chrome doesn't respond for >30s
- GitHub issue on message queue backlog during debugging/breakpoints
- React Native issue #46966: "CDP Runtime.evaluate does not drain async tasks"
  - Quote: "after Runtime.evaluate, planned Promise tasks are not executed... after a second or 5-10, the queue is triggered and drained again"
- Chrome DevTools MCP issue #786: Timeouts with large text operations

### Claim 3: "Connection may silently fail without proper error reporting"

**Status**: VERIFIED ✅

Evidence:
- Chromium issue #349533769: "the connection is closing: session: detach failed"
- Playwright issues #31697, #35115: Connection closing errors without clear indication
- ChromeDP issue #752: Hanging Chrome processes (zombie states)
- Stack Overflow reports of WebDriver becoming unresponsive, blocking test runner

### Claim 4: "Browser internal timeouts may fire independently of our timeout flags"

**Status**: VERIFIED ✅

Evidence:
- Chromium issue #374093512: Runtime.evaluate timeout behavior analysis
  - Quote: "The sync version really terminates the V8 isolate. The most we can do is race the eval promise with a setTimeout promise. That means even though we time out the Runtime.evaluate snippet, it might still continue running as its micro tasks are still queued up"
- Browser-side timeout ≠ actual termination of execution
- Microtasks continue running even after timeout fires

### Claim 5: "This can leave the browser in an inconsistent state"

**Status**: VERIFIED ✅

Evidence:
- Chromium issue #341213355: "CDP can evaluate scripts in a service worker before it is initialized"
- Execution can be terminated while promises are still pending
- Connection loss during operations leaves state undefined
- Selenium issues where one timeout causes all subsequent requests to timeout

### Claim 6: "Focus, scroll position, or page state may be corrupted"

**Status**: VERIFIED ✅

Evidence:
- Puppeteer issue #6830: waitForSelector(":focus") doesn't resolve on focus change
- TestCafe issue #8208: Browsers hang intermittently when scrolling is present
- Stack Overflow: waitForFunction inconsistent behavior with focus and scroll states
- Chromium issue #40871660: CDP commands frozen when Chrome window minimized

### Claim 7: "Multiple forced timeouts in quick succession stress the connection"

**Status**: REASONABLE (Logical Extrapolation) ⚠️

Evidence:
- Direct evidence: Single timeouts cause connection issues (verified above)
- Logical conclusion: Multiple timeouts would compound the issue
- No specific research found on "rapid repeated failures" pattern
- However, given that single timeouts cause message queue backlog and pending tasks, multiple timeouts would accumulate these issues

### Claim 8: "Each timeout may leave cleanup tasks pending"

**Status**: VERIFIED ✅

Evidence:
- React Native issue #46966: "CDP Runtime.evaluate does not drain async tasks"
  - Explicit confirmation that async tasks remain in queue after Runtime.evaluate
- Chromium issue #374093512: Microtasks continue running even after timeout
- No automatic cleanup mechanism for timed-out evaluations

### Claim 9: "Accumulated pending tasks can crash the daemon"

**Status**: PARTIALLY VERIFIED ⚠️

Evidence:
- Direct crashes: Multiple reports of daemon/ChromeDriver crashes (Chromium issues)
- Version mismatch issues leading to CDP daemon crashes (Stack Overflow #69466279)
- Chrome processes persisting after WebDriver disposal (Stack Overflow #78842163)
- ChromeDriver assuming Chrome crashed when unresponsive
- However: No specific evidence linking "accumulated pending tasks" to crashes
  - Crashes happen, but root cause attribution is unclear
  - Could be accumulated tasks, could be other CDP issues

## Additional Findings

### Runtime.evaluate Protocol Definition

From json/js_protocol.json:
- timeout parameter: "Terminate execution after timing out (number of milliseconds)"
- Status: experimental
- Optional parameter
- Type: TimeDelta

### Known Workarounds in Community

1. Increase protocolTimeout setting (chrome-devtools-mcp)
2. Manual Chrome instance management (chromedp - for long-running instances)
3. Custom polling implementations instead of built-in waitForFunction
4. Avoid minimizing Chrome window during automation
5. Version matching between Chrome, ChromeDriver, and CDP clients

### Fundamental CDP Architecture Limitations

- CDP is message-based, synchronous request/response
- No built-in health monitoring or connection recovery
- Timeout handling is "best effort" not guaranteed
- Browser and protocol operate independently - race conditions possible
- Experimental features (like timeout) may change without notice

## Conclusion

The comment block's claims are SUBSTANTIALLY ACCURATE. All major claims are either:
1. Directly verified by official bug reports and documentation, OR
2. Logical extrapolations from verified single-event issues

The health check workaround is justified and represents pragmatic engineering given CDP's limitations.

### Recommendation

Remove the TODO (line 781-784) and replace with reference to this research:
"Known CDP architectural limitation - see .ai/context/cdp-timeout-research.md"

The comment block should remain as-is - it accurately documents a real problem with a reasonable solution.
