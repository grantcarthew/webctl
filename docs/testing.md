# Testing Guide

## Current Coverage

As of 2025-12-17, overall test coverage is **65.6%**.

| Package | Coverage | Notes |
|---------|----------|-------|
| `internal/cdp` | 82.2% | Well tested CDP client |
| `internal/ipc` | 70.7% | Good IPC layer coverage |
| `internal/daemon` | 69.2% | Core event handling tested |
| `internal/cli` | 67.6% | Command handlers tested |
| `internal/browser` | 35.6% | Launch/close paths need work |
| `internal/executor` | 23.1% | Thin wrappers, low priority |

## Running Tests

```bash
# Run all tests
go test ./internal/...

# Run with coverage
go test -coverprofile=coverage.out ./internal/...
go tool cover -func=coverage.out

# Run integration tests (requires Chrome)
go test -run Integration ./internal/...

# Skip integration tests
go test -short ./internal/...
```

## Remaining Coverage Gaps

### Priority 1: High Value

#### `internal/browser/browser.go` (0% on critical functions)

The browser package has 0% coverage on launch and lifecycle functions:

| Function | Coverage | Difficulty |
|----------|----------|------------|
| `Start` | 0% | Medium - needs process mocking |
| `StartWithBinary` | 0% | Medium |
| `waitForCDP` | 0% | Low - can test timeout path |
| `Close` | 0% | Medium - needs process mocking |
| `Targets` | 0% | Low - HTTP mock |
| `PageTarget` | 0% | Low |
| `Version` | 0% | Low - HTTP mock |
| `WebSocketURL` | 0% | Low |

**Recommended approach**: Refactor `spawnProcess` to accept an interface, allowing injection of a mock process for testing. Alternatively, add integration tests that actually launch Chrome (already done in daemon integration test).

#### `internal/daemon/daemon.go:handleLoadingFinished` (0%)

This function fetches response bodies via CDP, which requires a connected CDP client.

**Recommended approach**:
1. Extract body fetching logic into a testable function
2. Mock the CDP client's `SendContext` method
3. Test both text and binary body handling paths

### Priority 2: Medium Value

#### `internal/cli/network.go` - Filtering functions

| Function | Coverage | Notes |
|----------|----------|-------|
| `filterNetworkEntries` | 25% | Need more filter combinations |
| `matchesNetworkFilters` | 0% | Core filtering logic |
| `outputNetworkText` | 0% | TTY output mode |

#### `internal/cli/console.go` - Text output

| Function | Coverage |
|----------|----------|
| `outputConsoleText` | 0% |

**Note**: Text output modes are only used when stdout is a TTY. These are lower priority since JSON output is the primary interface for AI agents.

### Priority 3: Low Value (Thin Wrappers)

#### `internal/executor/ipc.go` (0%)

Simple wrapper around `ipc.Client`. The underlying client is tested.

```go
func (e *IPCExecutor) Execute(req ipc.Request) (ipc.Response, error) {
    return e.client.Send(req)
}
```

**Recommendation**: Skip unless IPC executor is extended with additional logic.

#### `internal/ipc/client.go` - Convenience functions

| Function | Coverage | Notes |
|----------|----------|-------|
| `Dial` | 0% | Wrapper for `DialPath` |
| `IsDaemonRunning` | 0% | Wrapper for `IsDaemonRunningAt` |

## Test Patterns

### Mocking the Executor Factory

CLI commands use an executor factory for testability:

```go
exec := &mockExecutor{
    executeFunc: func(req ipc.Request) (ipc.Response, error) {
        // Verify request and return mock response
        return ipc.Response{OK: true, Data: mockJSON}, nil
    },
}

restore := setMockFactory(&mockFactory{
    daemonRunning: true,
    executor:      exec,
})
defer restore()
```

### Testing CDP Events

Daemon event handlers can be tested by creating mock CDP events:

```go
params := map[string]any{
    "requestId": "req-123",
    "errorText": "net::ERR_CONNECTION_REFUSED",
}
paramsJSON, _ := json.Marshal(params)

evt := cdp.Event{
    Method: "Network.loadingFailed",
    Params: json.RawMessage(paramsJSON),
}

d.handleLoadingFailed(evt)
```

### Table-Driven Tests

Use table-driven tests for comprehensive edge case coverage:

```go
tests := []struct {
    name     string
    input    string
    want     string
    wantErr  bool
}{
    {"valid input", "foo", "bar", false},
    {"empty input", "", "", true},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got, err := functionUnderTest(tt.input)
        if (err != nil) != tt.wantErr {
            t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
        }
        if got != tt.want {
            t.Errorf("got %q, want %q", got, tt.want)
        }
    })
}
```

## Integration Tests

Integration tests require Chrome and are skipped with `-short`:

```go
func TestDaemon_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    // ... test with real browser
}
```

Run integration tests explicitly:

```bash
go test -v -run Integration ./internal/daemon/
```
