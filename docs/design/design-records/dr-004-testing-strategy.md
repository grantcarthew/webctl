# DR-004: Testing Strategy

- Date: 2025-12-12
- Status: Accepted
- Category: Testing

## Problem

The CDP client library is the foundation for all webctl functionality. It involves:

- Concurrent operations (read loop goroutine + write operations)
- External message parsing (JSON from browser)
- Resource management (goroutines, channels, WebSocket connections)

Without a deliberate testing strategy, bugs in this layer propagate to all dependent code. CDP clients are particularly prone to:

- Race conditions from concurrent access
- Goroutine leaks from improper cleanup
- Panics from malformed external input
- Subtle correlation bugs in async request/response matching

## Decision

Implement a four-layer testing strategy focused on correctness before optimisation:

1. Unit tests with interface-based mocks
2. Race detection on all test runs
3. Integration tests with real WebSocket (no browser)
4. Goroutine leak detection

Defer benchmarks, golden files, and E2E tests until later projects.

## Why

Unit tests with mocks:

- Test logic in isolation without external dependencies
- Fast feedback loop during development
- Table-driven tests cover many cases efficiently
- Interface-based design enables testing without mocking frameworks

Race detection:

- CDP client is inherently concurrent (read loop + writes)
- Race conditions are easy to introduce, hard to debug
- Go's race detector catches real bugs with zero effort
- Run with every test, not as separate step

Integration tests:

- Validate real WebSocket message flow
- Use httptest server with WebSocket upgrader
- Catch serialisation and protocol issues mocks might miss
- Still fast, no browser required

Goroutine leak detection:

- CDP clients spawn goroutines for read loops
- Leaks accumulate silently, cause resource exhaustion
- goleak catches leaks at test exit
- Low overhead, high value

Fuzz testing for message parsing:

- CDP messages originate from external source (browser)
- Malformed JSON must not panic
- Go's native fuzzing is low-effort
- Catches edge cases humans miss

## Structure

Test file organisation:

```
internal/cdp/
├── conn.go              # Conn interface definition
├── client.go            # Client implementation
├── client_test.go       # Unit tests (run with -race)
├── message.go           # Message types
├── message_test.go      # Encoding tests + fuzz tests
├── integration_test.go  # //go:build integration
└── testdata/            # Mock CDP responses
```

Build tags:

- Default: Unit tests only (fast, no external dependencies)
- integration: Adds real WebSocket tests
- e2e: Future, adds Chrome-based tests (P-006+)

Test commands:

```bash
go test -race ./...                     # Unit + race detection (default)
go test -race -tags=integration ./...   # Add integration tests
go test -fuzz=FuzzParseMessage ./...    # Fuzz message parsing
```

## Interface Design for Testability

Define minimal interface for WebSocket connection:

```
Conn interface:
  ReadMessage() (messageType int, p []byte, err error)
  WriteMessage(messageType int, data []byte) error
  Close() error
```

Client accepts interface, not concrete type:

```
NewClient(conn Conn) *Client    # Accepts interface
Dial(wsURL string) (*Client, error)  # Convenience, returns concrete
```

Tests provide mock implementation:

```
mockConn struct:
  readMessages [][]byte   # Queued responses
  written      [][]byte   # Captured writes
  readErr      error      # Inject read errors
  writeErr     error      # Inject write errors
```

## Test Categories

Unit tests (client_test.go):

- TestClient_Send_CorrelatesResponseByID
- TestClient_Send_ReturnsErrorOnCDPError
- TestClient_Send_TimeoutWaitingForResponse
- TestClient_Subscribe_DispatchesToHandler
- TestClient_Subscribe_MultipleHandlers
- TestClient_Close_CleansUpResources
- TestClient_ConcurrentSends (run with -race)

Error injection tests:

- TestClient_Send_ConnectionClosedMidRequest
- TestClient_ReadLoop_MalformedJSON
- TestClient_ReadLoop_UnknownMessageID

Integration tests (integration_test.go):

- TestClient_RealWebSocket_RoundTrip
- TestClient_RealWebSocket_EventDispatch
- TestClient_RealWebSocket_Reconnect

Fuzz tests (message_test.go):

- FuzzParseMessage

## Dependencies

Required:

- go.uber.org/goleak - Goroutine leak detection

No mocking frameworks. Standard Go interfaces and structs suffice.

## Leak Detection Setup

Add to test main:

```
// internal/cdp/main_test.go
package cdp

import (
    "testing"
    "go.uber.org/goleak"
)

func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m)
}
```

This fails the test suite if any goroutines leak.

## CI Pipeline

Stages:

1. test-unit: `go test -race ./...`
2. test-integration: `go test -race -tags=integration ./...`
3. test-fuzz: `go test -fuzz=. -fuzztime=30s ./...` (optional, longer runs in scheduled jobs)

All stages run race detection. Fuzz testing can run with short fuzztime in CI, longer in scheduled jobs.

## Coverage Targets

| Package | Target | Rationale |
| ------- | ------ | --------- |
| internal/cdp | 90%+ | Foundation layer, must be solid |
| cmd/ | 80%+ | Logic functions high, glue code lower |

Coverage is a guide, not a goal. 90% coverage with poor tests is worse than 70% with good tests.

## Trade-offs

Accept:

- Additional test code to write and maintain
- goleak dependency
- Slower test runs with -race flag (~2-10x)
- Build tags add complexity

Gain:

- High confidence in concurrent code correctness
- Early detection of goroutine leaks
- Robust handling of malformed input
- Fast feedback without browser dependencies

## Alternatives

No interface, test against real WebSocket only:

- Pro: Tests real code path
- Pro: No mock maintenance
- Con: Requires WebSocket server for every test
- Con: Cannot inject errors easily
- Con: Slower test runs
- Rejected: Mock-based unit tests are faster and more flexible

Use mocking framework (gomock, testify/mock):

- Pro: Auto-generates mocks from interfaces
- Pro: Built-in assertion helpers
- Con: Additional dependency
- Con: Generated code can be verbose
- Con: Learning curve for framework
- Rejected: Hand-written mocks are simpler for small interfaces

Skip race detection in CI:

- Pro: Faster CI runs
- Con: Race conditions slip through
- Rejected: Race bugs are too costly; detection overhead is acceptable

## Deferred Items

Items intentionally deferred to later projects:

- E2E tests with real Chrome (P-006+, when CLI exists)
- Benchmark tests (optimise after correctness established)
- Golden file tests (for CLI output stability, P-006+)
- Black-box package tests (internal package, white-box is fine)

## Implementation Notes

CLI testability (future, P-006):

Follow the logic/output/command separation pattern:

- Logic functions return data structs, no I/O
- Output functions accept io.Writer and data
- Command functions wire Cobra to logic and output

This enables testing logic without mocking stdout.
