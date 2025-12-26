# DR-022: Serve Command Architecture

- Date: 2025-12-26
- Status: Proposed
- Category: CLI

## Problem

Developers need a way to serve web applications locally during development with hot reload capabilities while having access to webctl's browser automation and debugging features (console, network, screenshots, etc.). Current external dev servers don't integrate with webctl's CDP infrastructure.

The serve command needs to handle two distinct use cases:
1. Static file development - serving HTML/CSS/JS files with hot reload
2. Dynamic application development - proxying to a backend server while adding debugging capabilities

## Decision

Implement a daemon-managed development server with two separate modes (static and proxy) that integrates with webctl's existing CDP infrastructure for hot reload and provides automatic browser management.

Command Interface:

```bash
# Static mode
webctl serve <directory> [--port <n>] [--host <ip>] [--watch <pattern>] [--ignore <pattern>]

# Proxy mode
webctl serve --proxy <url> [--port <n>] [--host <ip>] [--watch <pattern>]

# Examples
webctl serve ./public
webctl serve ./dist --port 3000 --host 0.0.0.0
webctl serve --proxy http://localhost:3000
webctl serve --proxy http://localhost:3000 --watch "../backend/**/*.go"
```

No hybrid mode - either static OR proxy, not both.

## Why

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
- Serves files from specified directory
- MIME type detection based on file extension
- Watches directory for file changes
- Default watch pattern: `**/*.{html,css,js,json,svg,png,jpg,gif,webp,ico}`

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
1. User runs `webctl serve <dir>` or `webctl serve --proxy <url>`
2. Command starts daemon if not running
3. Daemon starts browser if not running
4. Daemon starts HTTP server
5. Server auto-detects available port (starting at 8080) or uses --port
6. Server binds to localhost (127.0.0.1) or --host if specified
7. Browser navigates to server URL
8. File watcher starts (if applicable)

Output on Start:
```
Server started on http://127.0.0.1:8080
Serving: /home/user/myapp
Watching: **/*.{html,css,js,json,svg,png,jpg,gif,webp,ico}
Network: http://192.168.1.5:8080 (run with --host 0.0.0.0)
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

Default Behavior:
- Try port 8080
- If in use, try 8081, 8082, etc.
- Show which port was selected

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
- Wait 100ms after last file change before triggering reload
- Prevents rapid-fire reloads during multiple file saves

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

Go Libraries:
- `net/http` - HTTP server and handlers
- `net/http/httputil` - ReverseProxy for proxy mode
- `fsnotify/fsnotify` - File system watching
- `path/filepath` - Path manipulation and matching
- Glob matching library (research needed - possibly `github.com/gobwas/glob`)

Server Package Structure:
```
internal/server/
  server.go       - HTTP server setup and lifecycle
  static.go       - Static file handler
  proxy.go        - Reverse proxy handler
  watcher.go      - File watching and reload triggering
  patterns.go     - Glob pattern matching
```

Daemon Integration:
- Add server instance to Daemon struct
- Add serve command handler in IPC
- Add server status to status response
- Hook server shutdown into daemon shutdown

## Updates

None yet.
