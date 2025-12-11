# P-003: CDP Core Library

- Status: Proposed
- Started: -

## Overview

Build the minimal CDP client library that all other components depend on. This is the foundation layer - WebSocket connection, message encoding, command/response correlation, and event dispatch.

## Goals

1. Establish WebSocket connection to CDP endpoint
2. Send commands and receive responses with ID correlation
3. Subscribe to and dispatch CDP events
4. Handle errors and disconnections gracefully

## Scope

In Scope:

- WebSocket connection management
- Thread-safe write operations (mutex or channel-based)
- JSON message encoding/decoding for CDP protocol
- Command ID generation and response matching
- Event listener registration and dispatch
- Connection lifecycle (connect, disconnect, reconnect detection)

Out of Scope:

- Browser launch/detection (P-004)
- Specific CDP domain implementations (later projects)
- Daemon architecture (P-005)
- CLI interface (P-006)

## Success Criteria

- [ ] Can connect to a running CDP endpoint via WebSocket
- [ ] Can send commands and receive correlated responses
- [ ] Can subscribe to events and receive them asynchronously
- [ ] Thread-safe for concurrent command sends
- [ ] Handles connection errors gracefully
- [ ] Unit tests with mock WebSocket

## Deliverables

- `internal/cdp/client.go` - main CDP client
- `internal/cdp/message.go` - message types and encoding
- `internal/cdp/client_test.go` - unit tests

## Technical Design

### Package Structure

```
internal/cdp/
├── client.go      # CDPClient struct, Connect, Send, Subscribe
├── message.go     # Request, Response, Event types
└── client_test.go # Tests with mock WebSocket
```

### Core Types

```go
type Client struct {
    conn      *websocket.Conn
    writeMu   sync.Mutex
    msgID     atomic.Int64
    pending   sync.Map          // id -> chan Response
    listeners sync.Map          // method -> []func(Event)
}

type Request struct {
    ID     int         `json:"id"`
    Method string      `json:"method"`
    Params interface{} `json:"params,omitempty"`
}

type Response struct {
    ID     int             `json:"id"`
    Result json.RawMessage `json:"result,omitempty"`
    Error  *Error          `json:"error,omitempty"`
}

type Event struct {
    Method string          `json:"method"`
    Params json.RawMessage `json:"params"`
}

type Error struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}
```

### Key Methods

```go
func Dial(wsURL string) (*Client, error)
func (c *Client) Send(method string, params interface{}) (json.RawMessage, error)
func (c *Client) Subscribe(method string, handler func(Event))
func (c *Client) Close() error
```

### Thread Safety

WebSocket writes require synchronisation. Options:

1. **Mutex** - Simple, wrap each `WriteMessage` in lock
2. **Channel** - Send messages to a dedicated writer goroutine

Start with mutex for simplicity. Refactor to channel if performance requires.

### Event Loop

Dedicated goroutine reads messages and dispatches:

```go
func (c *Client) readLoop() {
    for {
        _, msg, err := c.conn.ReadMessage()
        if err != nil {
            // Handle disconnect
            return
        }
        // Dispatch to pending commands or event listeners
    }
}
```

## Dependencies

- `github.com/gorilla/websocket` - WebSocket client

## Testing Strategy

1. **Unit tests** - Mock WebSocket connection, verify message encoding
2. **Integration tests** - Connect to real Chrome instance (manual, not CI)

## Notes

This is the most critical package. Take time to get the API right - all other code depends on it.
