# DR-003: CDP Eager Data Capture

- Date: 2025-12-11
- Status: Accepted
- Category: Architecture

## Problem

CDP events contain data that may require additional CDP calls to fully capture:

1. Network response bodies: Available via `Network.getResponseBody` after `loadingFinished` event, but the body may become unavailable if the browser navigates or clears its cache.

2. Console object arguments: `Runtime.consoleAPICalled` events return object references (objectId) rather than values. Full object contents require calling `Runtime.getProperties`.

The daemon must decide when to fetch this additional data: eagerly (at event time) or lazily (when user requests it).

## Decision

Eager capture for both network bodies and console objects.

Network Response Bodies:
- Fetch body immediately when `Network.loadingFinished` fires
- Store complete body in the network event buffer
- Body is available instantly when `webctl network` is called

Console Object Arguments:
- Resolve object references immediately when `Runtime.consoleAPICalled` fires
- Call `Runtime.getProperties` for each argument with an objectId
- Store resolved values in the console event buffer

## Why

Predictable behaviour:
- Users always get complete data
- No "body not available" errors due to navigation timing
- No difference between immediate and delayed queries

Simplicity:
- Single code path for data retrieval
- No lazy-loading state management
- No retry logic or error handling for stale references

PoC validation:
- Tested during P-002: bodies available immediately after `loadingFinished`
- No timing issues observed
- Pattern works reliably

## Trade-offs

Accept:
- Higher memory usage (bodies and resolved objects stored in buffer)
- More CDP calls at event time (slight latency during page activity)
- Large response bodies consume buffer space quickly

Gain:
- Complete data always available
- Instant response to `webctl network` and `webctl console`
- No "data expired" failure modes
- Simpler implementation

## Alternatives

Lazy fetching (fetch on query):

- Pro: Lower memory usage
- Pro: Fewer CDP calls during page activity
- Con: Bodies may be unavailable after navigation
- Con: Object references become stale when execution context clears
- Con: Query latency as bodies are fetched
- Con: Complex error handling for unavailable data
- Rejected: Reliability more important than memory savings

Hybrid (eager for small, skip large):

- Pro: Limits memory for large responses
- Con: Unpredictable behaviour (some bodies available, some not)
- Con: Configurable threshold adds complexity
- Con: Users confused when large bodies missing
- Rejected: Inconsistent behaviour is worse than memory cost

## Implementation Notes

Memory management via existing buffer limits:
- 10,000 entries per buffer (console, network)
- Ring buffer with oldest-first eviction
- Large bodies evicted naturally as buffer fills

Future options if memory becomes an issue:
- Add `--max-body-size` flag to skip bodies over threshold
- Add `--buffer-size` flag to adjust entry count
- These would be explicit user choices, not hidden magic
