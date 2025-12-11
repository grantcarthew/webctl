# P-005: Daemon & IPC

- Status: Proposed
- Started: -

## Overview

Build the persistent daemon that runs the browser, subscribes to CDP events, and buffers them. Implement Unix socket IPC for CLI communication.

## Goals

1. Daemon process that persists independently of CLI
2. Event buffering (console logs, network requests)
3. Unix socket server for IPC
4. JSON request/response protocol

## Scope

In Scope:

- Daemon process management (foreground and background modes)
- PID file for daemon detection
- Unix socket server
- JSON protocol for commands
- Ring buffers for console and network events
- CDP event subscription and buffering
- Clean shutdown handling

Out of Scope:

- TCP remote access (can add later)
- Token authentication (can add later)
- CLI implementation (P-006)
- Specific command implementations beyond basic lifecycle

## Success Criteria

- [ ] Daemon starts and runs independently
- [ ] PID file created at XDG-compliant location
- [ ] Unix socket accepts connections
- [ ] Can receive JSON commands and send responses
- [ ] Buffers CDP console events (up to 10,000)
- [ ] Buffers CDP network events (up to 10,000)
- [ ] Clean shutdown on SIGTERM/SIGINT
- [ ] Cleans up socket and PID file on exit

## Deliverables

- `internal/daemon/daemon.go` - main daemon logic
- `internal/daemon/buffer.go` - ring buffer implementation
- `internal/ipc/server.go` - Unix socket server
- `internal/ipc/protocol.go` - JSON message types
- `internal/ipc/client.go` - client for CLI to use
- Tests

## Technical Design

### Package Structure

```
internal/daemon/
├── daemon.go    # Daemon struct, Run, Shutdown
└── buffer.go    # RingBuffer for events

internal/ipc/
├── server.go    # UnixServer, Accept, Handle
├── client.go    # UnixClient for CLI
└── protocol.go  # Request, Response types
```

### XDG Paths (from DR-001)

```
Socket: $XDG_RUNTIME_DIR/webctl/webctl.sock
        Fallback: /tmp/webctl-<uid>/webctl.sock

PID:    $XDG_RUNTIME_DIR/webctl/webctl.pid
```

### Ring Buffer

```go
type RingBuffer[T any] struct {
    items []T
    head  int
    tail  int
    size  int
    mu    sync.RWMutex
}

func NewRingBuffer[T any](capacity int) *RingBuffer[T]
func (b *RingBuffer[T]) Push(item T)
func (b *RingBuffer[T]) All() []T
func (b *RingBuffer[T]) Clear()
```

### IPC Protocol

Request:
```json
{"cmd": "status"}
{"cmd": "console"}
{"cmd": "clear", "target": "console"}
```

Response:
```json
{"ok": true, "data": {...}}
{"ok": false, "error": "message"}
```

### Daemon Lifecycle

```go
type Daemon struct {
    browser     *browser.Browser
    cdp         *cdp.Client
    consoleBuf  *RingBuffer[ConsoleEntry]
    networkBuf  *RingBuffer[NetworkEntry]
    server      *ipc.Server
}

func (d *Daemon) Run(ctx context.Context) error
func (d *Daemon) Shutdown() error
```

### Event Subscription

On startup, daemon:
1. Launches browser (P-004)
2. Connects via CDP (P-003)
3. Enables domains: Runtime, Network, Page
4. Subscribes to events:
   - `Runtime.consoleAPICalled` → consoleBuf
   - `Runtime.exceptionThrown` → consoleBuf
   - `Network.requestWillBeSent` → networkBuf
   - `Network.responseReceived` → networkBuf (update entry)
   - `Network.loadingFinished` → fetch body, update entry

### Command Handlers

Initial commands (lifecycle only):
- `status` - return daemon status, current URL, title
- `clear` - clear buffers

Other commands added in P-007, P-008.

### Signal Handling

```go
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
<-sigCh
d.Shutdown()
```

## Dependencies

- P-003 (CDP Core Library)
- P-004 (Browser Launch)

## Testing Strategy

1. **Unit tests** - Ring buffer, protocol encoding
2. **Integration tests** - Start daemon, connect via socket, send commands

## Notes

The daemon is the heart of webctl. It must be rock-solid for reliability.

Consider: Should daemon run in foreground by default (like `webctl start` blocks) or background? DR-001 examples show `&` for background, suggesting foreground default. Decide in implementation.
