# P-016: Serve Command

- Status: Completed
- Started: 2025-12-30
- Completed: 2025-12-30

## Overview

Add a development web server to webctl that serves static files or proxies to a backend application while providing hot reload capabilities and full access to webctl's browser automation and debugging features. This enables developers to use webctl's console, network, and automation commands while developing their own web applications.

## Goals

1. Implement static file server with file watching and hot reload
2. Implement proxy mode to sit between backend applications and browser
3. Integrate server lifecycle with daemon management (like browser)
4. Provide automatic browser navigation and CDP-based reload mechanism
5. Support flexible file watching with glob patterns
6. Enable network access for mobile/remote testing

## Scope

In Scope:

- Static file serving from directory with MIME type detection
- Reverse proxy mode forwarding requests to backend server
- File watching with configurable glob patterns
- Hot reload via CDP (no script injection required)
- Auto-detect available ports with manual override
- Network binding options (localhost vs network accessible)
- Daemon-managed server lifecycle
- Automatic browser start and navigation
- Timestamped file change logging
- Integration with existing stop/exit/quit commands

Out of Scope:

- Hybrid mode (static + proxy combined) - keep modes separate
- Build process integration (no compilation, bundling, transpiling)
- HTTPS/TLS support (localhost development only)
- WebSocket proxying (initial version)
- Smart reload (CSS-only updates) - full page reload only
- Multiple simultaneous servers
- Request/response logging UI
- Mock API responses

## Success Criteria

- [ ] Can serve static files from a directory
- [ ] Can proxy requests to a backend server
- [ ] File changes trigger automatic page reload via CDP
- [ ] Server starts and stops via daemon like browser does
- [ ] Browser auto-starts and navigates to served URL
- [ ] Port auto-detection works, manual override available
- [ ] File watching supports glob patterns
- [ ] Can bind to network interface for remote access
- [ ] All webctl commands (console, network, etc.) work with served content
- [ ] File change events logged with timestamps
- [ ] stop/exit/quit commands stop server along with daemon

## Deliverables

- `internal/server/` - New server package
  - Static file server implementation
  - Reverse proxy implementation
  - File watcher with glob pattern support
  - Server lifecycle management
- `internal/cli/serve.go` - Serve command implementation
- `docs/cli/serve.md` - User documentation for serve command
- DR-022: Serve Command Architecture and Hot Reload Strategy
- Updated `internal/daemon/daemon.go` - Server lifecycle integration
- Updated AGENTS.md - Mark P-016 as active, then completed

## Technical Approach

High-level implementation strategy:

1. Server Package
   - HTTP server using `net/http`
   - Static file handler with MIME type detection
   - Reverse proxy using `net/http/httputil.ReverseProxy`
   - File watcher using `fsnotify/fsnotify`
   - Glob pattern matching for watch filters
   - Debouncing for file change events

2. Daemon Integration
   - Add server management to daemon (similar to browser)
   - Server starts via IPC request
   - Server stops with daemon shutdown
   - Server status in daemon status response

3. Hot Reload Mechanism
   - File watcher detects changes
   - Calls CDP Page.reload() via existing client
   - No WebSocket server or script injection needed
   - Leverage existing CDP infrastructure

4. Command Interface
   - `webctl serve <directory>` - Static mode
   - `webctl serve --proxy <url>` - Proxy mode
   - `--port <n>` - Manual port specification
   - `--host <ip>` - Network binding
   - `--watch <pattern>` - Custom watch patterns
   - `--ignore <pattern>` - Ignore patterns

## Questions & Uncertainties

- Should we support serving index.html for directory requests?
- How to handle 404s in static mode - show error page or just 404?
- Should directory listing be enabled or disabled?
- WebSocket proxying - add later or include in MVP?
- Should we validate proxy targets (localhost only vs any URL)?
- Compression support (gzip) - needed for dev server?
- Caching headers - set or omit for development?

## Testing Strategy

- Unit tests for file watcher and pattern matching
- Integration tests for static file serving
- Integration tests for proxy mode
- Test file change detection and reload triggering
- Test port auto-detection and conflicts
- Test daemon lifecycle integration
- Manual testing with real web applications

## Notes

Key insight: webctl already has all the debugging infrastructure via CDP. This feature lets developers use those capabilities (console, network, automation) while developing their own apps, not just testing external sites.

The CDP-based reload is simpler than traditional hot reload (no WebSocket, no injection) and leverages existing infrastructure.

Keeping static and proxy modes separate (no hybrid) reduces complexity and keeps the implementation focused.
