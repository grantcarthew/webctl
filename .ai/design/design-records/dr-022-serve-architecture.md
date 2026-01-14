# DR-022: Serve Command Architecture

- Date: 2025-12-26
- Status: Accepted
- Category: CLI
- Updated: 2025-12-30

## Problem

Developers need a way to serve web applications locally during development with hot reload capabilities while having access to webctl's browser automation and debugging features (console, network, screenshots, etc.). Current external dev servers don't integrate with webctl's CDP infrastructure.

The serve command needs to handle two distinct use cases:
1. Static file development - serving HTML/CSS/JS files with hot reload
2. Dynamic application development - proxying to a backend server while adding debugging capabilities

## Decision

Implement a daemon-managed development server with two separate modes (static and proxy) that integrates with webctl's existing CDP infrastructure for hot reload and provides automatic browser management.

Command Interface:

```bash
# Static mode (directory defaults to current directory)
webctl serve [directory] [--port <n>] [--host <ip>] [--watch <pattern>] [--ignore <pattern>]

# Proxy mode
webctl serve --proxy <url> [--port <n>] [--host <ip>] [--watch <pattern>]

# Examples
webctl serve                  # Serve current directory (new default)
webctl serve ./public
webctl serve ./dist --port 3000 --host 0.0.0.0
webctl serve --proxy http://localhost:3000
webctl serve --proxy http://localhost:3000 --watch "../backend/**/*.go"
```

Key Behaviors:
- Auto-starts daemon and browser if not running (one-command operation)
- Defaults to serving current directory (`.`) if no directory specified
- Auto-detects available CDP port (9222, 9223, ...) to avoid conflicts
- Auto-detects available web server port (3000, 8080, 8000, 5000, 4000, ...)
- Tries common index files for directory requests (index.html, index.htm, default.html, home.html)

No hybrid mode - either static OR proxy, not both.

## Why

Auto-Start Daemon and Browser:
- One-command workflow: `webctl serve` does everything
- Eliminates manual `webctl start` step
- Better developer experience for new users
- Consistent with user expectations (like `python -m http.server`)
- Auto-detects CDP port to avoid conflicts with existing browsers

Default to Current Directory:
- Most common use case (serve what you're working on)
- Matches standard dev server behavior (like `python -m http.server`)
- Reduces typing and cognitive load
- Explicit directory still supported when needed

Index File Fallback:
- Supports multiple common naming conventions (index.html, index.htm, default.html, home.html)
- Handles legacy projects and different frameworks
- Predictable priority order (index.html first)
- Better compatibility across different project types

Daemon Management:
- Consistent with how browser is managed
- Allows background operation
- Single stop command for everything
- Server lifecycle tied to daemon lifecycle

CDP-Based Hot Reload:
- No need for WebSocket server
- No script injection into HTML
- Leverages existing CDP infrastructure
- Simple: file change → CDP Page.reload() → browser reloads
- Works in both static and proxy modes

Separate Modes (No Hybrid):
- Simpler implementation and mental model
- No complex routing logic needed
- Clear separation of concerns
- If users need both, backend can serve its own static files

Auto-Start Browser and Navigate:
- Matches existing `webctl start` behavior
- Ready to use immediately
- Consistent user experience

Port Auto-Detection:
- Convenient default behavior
- Avoids common "port in use" errors
- Manual override available when needed

Glob Patterns for File Watching:
- Standard for file matching (like .gitignore)
- More intuitive than regex for paths
- Matches user expectations from other tools

## Trade-offs

Accept:

- No hybrid static+proxy mode (users must choose one)
- No WebSocket proxying in MVP (can add later)
- No smart reload (CSS-only) - always full page reload
- Server tied to daemon lifecycle (can't run standalone)
- No HTTPS support (localhost development only)
- No build process integration (pure serving/proxying)

Gain:

- Simple, clear architecture
- Leverages existing CDP infrastructure
- Minimal new code (no WebSocket, no injection)
- Consistent with webctl patterns
- Full debugging access while developing
- Easy to understand and use

## Alternatives

Traditional WebSocket-Based Hot Reload:

- Pro: Standard approach used by most dev servers
- Pro: Works without CDP
- Con: Requires WebSocket server
- Con: Requires injecting script into HTML responses
- Con: More complex in proxy mode (response transformation)
- Con: Doesn't leverage webctl's existing capabilities
- Rejected: CDP-based approach is simpler and more aligned with webctl's architecture

Hybrid Static+Proxy Mode:

- Pro: Serve static assets + proxy API routes in one server
- Pro: Common pattern (like webpack-dev-server)
- Con: Complex routing logic needed
- Con: Ambiguity in what gets served vs proxied
- Con: More configuration required
- Rejected: Keeping modes separate is simpler and clearer

Foreground Blocking Server:

- Pro: Traditional server behavior (like python -m http.server)
- Pro: Simple Ctrl+C to stop
- Con: Inconsistent with browser management
- Con: Blocks terminal
- Con: Separate lifecycle from daemon
- Rejected: Daemon-managed is more consistent with webctl patterns

Regex Pattern Matching:

- Pro: More powerful than glob
- Con: More complex for users
- Con: Less intuitive for file paths
- Con: Not standard for file matching
- Rejected: Glob is the standard for file watching

## Structure

Server Modes:

Static Mode:
- Serves files from specified directory (defaults to current directory)
- MIME type detection based on file extension
- Index file fallback for directory requests (tries in order):
  1. index.html
  2. index.htm
  3. default.html
  4. home.html
- Watches directory for file changes
- Auto-ignores: hidden files (.*), node_modules/, vendor/, __pycache__/
- Default watch pattern: entire served directory

Proxy Mode:
- Forwards all requests to specified backend URL
- Preserves headers, cookies, method, body
- Optional file watching (for backend file changes)
- No default watch pattern in proxy mode

## Hot Reload Mechanism

File Watching:
1. Use `fsnotify/fsnotify` for file system events
2. Filter events based on glob patterns
3. Debounce changes (100ms window)
4. Batch multiple file changes into single reload

Reload Trigger:
1. File watcher detects change
2. Call existing CDP reload function (same as `webctl reload` command)
3. Browser receives Page.reload() via CDP
4. Page reloads

No injection, no WebSocket, no client-side code needed.

## Process Lifecycle

Server Start:
1. User runs `webctl serve` (defaults to current directory)
2. If daemon not running:
   - Starts daemon in-process
   - Auto-detects available CDP port (tries 9222, 9223, 9224, ...)
   - Launches browser on detected CDP port
3. Daemon starts HTTP server
4. Server auto-detects available port (tries 3000, 8080, 8000, 5000, 4000, then OS-assigned) or uses --port
5. Server binds to localhost (127.0.0.1) or --host if specified
6. Browser navigates to server URL
7. File watcher starts (if applicable)
8. Command blocks until Ctrl+C (daemon runs in foreground)

Output on Start:
```
Starting daemon and server...
Server started: http://localhost:3000
Mode: static
Directory: .
Port: 3000

Watching for file changes (hot reload enabled)

Press Ctrl+C to stop the server and daemon
```

File Change Event:
```
[18:32:15] index.html changed - reloaded
```

Server Stop:
- User runs `stop`, `exit`, or `quit` in REPL
- Or runs `webctl stop` command
- Daemon shuts down (stops browser and server together)

## Port Selection

Web Server Port (Default Behavior):
- Try common dev ports in order: 3000, 8080, 8000, 5000, 4000
- If all in use, request OS-assigned port (random available port)
- Show which port was selected in output

CDP Port (Auto-Start Only):
- Try default port 9222
- If in use, try 9223, 9224, ... up to 9322 (100 ports)
- Displays message: "Port 9222 in use, using port 9223 instead"

Manual Override:
```bash
webctl serve ./public --port 3000
```
- Try specified port only
- Fail with clear error if port in use

## Network Binding

Default (Localhost Only):
```bash
webctl serve ./public
# Binds to 127.0.0.1 - only accessible from this machine
```

Network Access:
```bash
webctl serve ./public --host 0.0.0.0
# Binds to all interfaces - accessible from network
```

Specific Interface:
```bash
webctl serve ./public --host 192.168.1.5
# Binds to specific IP
```

## File Watching

Default Patterns:

Static Mode:
- Watch: `**/*.{html,css,js,json,svg,png,jpg,gif,webp,ico}`
- Ignore: `.git/**`, `node_modules/**`, `**/*.test.js`, `**/*_test.go`

Proxy Mode:
- No default watching
- User must specify `--watch` if desired

Custom Patterns:
```bash
# Watch only HTML and CSS
webctl serve ./public --watch "**/*.{html,css}"

# Watch backend Go files in proxy mode
webctl serve --proxy http://localhost:3000 --watch "../backend/**/*.go"

# Multiple ignore patterns
webctl serve ./public --ignore "dist/**" --ignore "build/**"
```

Debouncing:
- Wait 300ms after last file change before triggering reload
- Prevents rapid-fire reloads during multiple file saves
- Balances responsiveness with avoiding excessive reloads

## Integration with Existing Commands

All existing webctl commands work with served content:

```bash
# Start serving
webctl serve ./myapp

# In REPL or separate terminal, use all webctl features:
webctl console           # See console logs from your app
webctl network          # Debug API calls
webctl screenshot       # Visual testing
webctl eval 'window.state'  # Inspect application state
webctl find "error"     # Verify error messages
webctl click "#submit"  # Test interactions
```

The key value is using webctl's debugging capabilities while developing.

## Security Considerations

Directory Traversal Protection:
- Validate and clean file paths
- Reject requests with `..` that escape serve directory
- Return 404 for invalid paths

Proxy Target Validation:
- For MVP: warn if proxy target is not localhost
- Future: flag to allow remote proxy targets

Network Binding Warning:
- When binding to 0.0.0.0, warn about network exposure
- Remind that this is a development server, not production

CORS:
- For MVP: no special CORS handling
- Future: add CORS headers if needed

## Implementation Notes

Go Libraries (Implemented):
- `net/http` - HTTP server and handlers
- `net/http/httputil` - ReverseProxy for proxy mode
- `fsnotify/fsnotify` v1.9.0 - File system watching
- `path/filepath` - Path manipulation and glob matching
- Standard library only (no external glob library needed)

Server Package Structure (Implemented):
```
internal/server/
  server.go       - HTTP server setup and lifecycle
  static.go       - Static file handler with index fallback
  proxy.go        - Reverse proxy handler
  watcher.go      - File watching with debouncing
  server_test.go  - Unit tests for server lifecycle
  watcher_test.go - Unit tests for file watching
```

Daemon Integration:
- Add server instance to Daemon struct
- Add serve command handler in IPC
- Add server status to status response
- Hook server shutdown into daemon shutdown

## Updates

### 2025-12-30: Implementation Completed

Status changed from Proposed to Accepted. Implementation completed with the following enhancements:

Auto-Start Daemon (UX Enhancement):
- `webctl serve` now auto-starts daemon if not running
- Eliminates need for manual `webctl start` command
- One-command workflow for developers
- Uses DirectExecutorFactory for in-process daemon communication (avoids IPC race condition)

Default Directory Behavior:
- Directory argument now optional, defaults to current directory (`.`)
- Matches user expectations from other dev servers
- Reduces typing and cognitive load
- Use: `webctl serve` instead of `webctl serve .`

CDP Port Auto-Detection:
- When auto-starting daemon, uses port 0 to auto-detect available CDP port
- Browser package tries 9222, 9223, 9224, ... up to 9322
- Prevents "port already in use" errors
- Gracefully handles multiple browser instances

Index File Fallback (Robustness):
- Static handler tries multiple common index files in order:
  1. index.html (highest priority)
  2. index.htm
  3. default.html
  4. home.html
- Better compatibility with different project conventions
- Cleaner implementation (removed duplicate logic)

Implementation Details:
- External dependency: fsnotify/fsnotify v1.9.0 (approved)
- Debouncing: 300ms (balances responsiveness vs excessive reloads)
- Auto-ignore: hidden files (.*), node_modules/, vendor/, __pycache__/
- Web server port priority: 3000, 8080, 8000, 5000, 4000, then OS-assigned
- Security: Directory traversal protection, no directory listing

Test Coverage:
- Server lifecycle (start/stop)
- Index file fallback (all 4 file types + precedence)
- Port auto-detection
- File watcher events
- Ignore pattern matching
- Debouncer functionality
- All tests passing ✓

Files Added:
- internal/server/server.go (core server)
- internal/server/static.go (static file handler)
- internal/server/proxy.go (reverse proxy)
- internal/server/watcher.go (file watching with debouncing)
- internal/server/server_test.go (unit tests)
- internal/server/watcher_test.go (unit tests)
- internal/daemon/handlers_serve.go (daemon integration)
- internal/cli/serve.go (CLI command)
- internal/cli/serve_test.go (CLI tests)
- docs/cli/serve.md (user documentation)

Files Modified:
- internal/ipc/protocol.go (ServeParams, ServeData types)
- internal/daemon/daemon.go (devServer field, shutdown hook)
- AGENTS.md (active project status)
- .ai/projects/README.md (project status)
- go.mod (fsnotify dependency)

Project Status: P-016 CLI Serve Command - Completed 2025-12-30

Key Achievement: True one-command development server with hot reload, fully integrated with webctl's browser automation and debugging capabilities.
