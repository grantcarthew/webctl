# P-004: Browser Launch & Target Management

- Status: Completed
- Started: 2025-12-12
- Completed: 2025-12-12

## Overview

Find Chrome on the system, launch it with CDP enabled, discover available targets, and attach to a page. This handles the "browser side" before the daemon architecture.

## Goals

1. Detect Chrome/Chromium binary across platforms
2. Launch browser with appropriate CDP flags
3. Discover available targets via CDP HTTP endpoint
4. Select and attach to page targets

## Scope

In Scope:

- Chrome binary detection (macOS, Linux)
- Process spawning with CDP flags
- Target discovery via `http://localhost:PORT/json`
- Page target selection
- Process lifecycle (launch, wait, kill)
- Headless mode support

Out of Scope:

- Daemon architecture (P-005)
- CLI interface (P-006)
- Multiple browser sessions

## Success Criteria

- [x] Finds Chrome on macOS (tested)
- [ ] Finds Chrome on Linux (tested)
- [x] Launches Chrome with `--remote-debugging-port`
- [x] Discovers page targets from `/json` endpoint
- [x] Returns WebSocket URL for CDP connection
- [x] Clean process termination on close

## Deliverables

- `internal/browser/detect.go` - find Chrome binary
- `internal/browser/launch.go` - spawn browser process
- `internal/browser/target.go` - target discovery
- `internal/browser/browser.go` - main Browser type
- Tests

## Technical Design

### Package Structure

```
internal/browser/
├── browser.go   # Browser struct, Launch, Close
├── detect.go    # FindChrome for each platform
├── launch.go    # Process spawning
└── target.go    # Target discovery types
```

### Chrome Detection Paths

**macOS:**

```
/Applications/Google Chrome.app/Contents/MacOS/Google Chrome
/Applications/Chromium.app/Contents/MacOS/Chromium
```

**Linux:**

```
/usr/bin/google-chrome
/usr/bin/google-chrome-stable
/usr/bin/chromium
/usr/bin/chromium-browser
/snap/bin/chromium
```

### Launch Flags

See DR-005 for full details.

Required:

```
--remote-debugging-port=PORT
--no-first-run
--no-default-browser-check
--disable-background-networking
--disable-sync
--disable-popup-blocking
```

Platform-specific:

```
--use-mock-keychain          # macOS
--password-store=basic       # Linux
```

Headless:

```
--headless
```

User data directory:

```
--user-data-dir=TEMP_DIR     # Default: temp directory
                              # "default": use user's Chrome profile
                              # Any path: use that directory
```

### Core Types

```go
type Browser struct {
    cmd       *exec.Cmd
    port      int
    targetURL string
    wsURL     string
    dataDir   string  // temp user data dir
}

type Target struct {
    ID                   string `json:"id"`
    Type                 string `json:"type"`
    Title                string `json:"title"`
    URL                  string `json:"url"`
    WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

type LaunchOptions struct {
    Headless    bool
    Port        int     // 0 = default 9222
    UserDataDir string  // empty = temp, "default" = user profile, path = use path
}
```

### Key Methods

```go
func FindChrome() (string, error)
func Launch(opts LaunchOptions) (*Browser, error)
func (b *Browser) Targets() ([]Target, error)
func (b *Browser) PageTarget() (*Target, error)  // First page target
func (b *Browser) Close() error
```

### Port Selection

- Default: 9222
- If specified port is in use, return error (don't auto-increment)
- User can specify `--port 0` for auto-selection in future

### User Data Directory

Three modes (see DR-005):

- Empty (default): Create temp directory, cleaned up on close
- `default`: Use user's Chrome profile
- Any path: Use that directory

Temp directory location: `os.MkdirTemp("", "webctl-chrome-*")`

### Startup Timeout

30 seconds to wait for CDP endpoint to respond.

### Graceful Shutdown

Send SIGINT first, fall back to SIGKILL if needed.

## Dependencies

- P-003 (CDP Core Library) - for eventual connection
- Standard library only for this package

## Testing Strategy

1. **Unit tests** - Mock exec.Command for process spawning
2. **Integration tests** - Actually launch Chrome (manual)

## Notes

Linux and macOS are the supported platforms. Windows is not planned.
