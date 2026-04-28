# webctl serve

Start a development web server with automatic hot reload capabilities.

## Synopsis

```bash
webctl serve                          # Serve current directory (default)
webctl serve <directory>              # Static file server
webctl serve --proxy <url>            # Reverse proxy server
```

## Description

The `serve` command starts a local development web server that integrates with webctl's browser automation and debugging features. It supports two modes:

- **Static Mode**: Serve files from a directory with automatic hot reload
- **Proxy Mode**: Reverse proxy requests to a backend server

When files change (static mode), the page automatically reloads via CDP, eliminating the need for browser extensions or script injection.

## Static Mode

Serve static files from a directory:

```bash
webctl serve <directory>
```

### Examples

```bash
# Serve current directory (default)
webctl serve

# Serve current directory (explicit)
webctl serve .

# Serve specific directory
webctl serve ./public
webctl serve ./dist
webctl serve /var/www/html

# Custom port
webctl serve ./public --port 3000

# Network accessible (not just localhost)
webctl serve ./public --host 0.0.0.0

# Custom watch paths
webctl serve ./public --watch src/,assets/

# Ignore patterns
webctl serve ./public --ignore "*.tmp,*.log,*.swp"
```

### Static Mode Behavior

- Automatic index file detection for directory requests (tries in order):
  1. `index.html`
  2. `index.htm`
  3. `default.html`
  4. `home.html`
- Disables browser caching (development mode)
- Watches directory for changes (default)
- Triggers page reload on file changes
- MIME type detection automatic

## Proxy Mode

Reverse proxy requests to a backend server:

```bash
webctl serve --proxy <url>
```

### Examples

```bash
# Proxy to local backend
webctl serve --proxy localhost:8080
webctl serve --proxy http://127.0.0.1:3000

# Proxy to remote backend
webctl serve --proxy http://api.example.com
webctl serve --proxy https://staging.example.com

# Custom frontend port
webctl serve --proxy localhost:8080 --port 3001

# Network accessible proxy
webctl serve --proxy localhost:8080 --host 0.0.0.0
```

### Proxy Mode Behavior

- Forwards all requests to backend
- Preserves request headers
- Sets `X-Forwarded-Host` header
- No caching (development mode)
- No file watching (backend handles changes)

## Flags

### Mode Selection

- `[directory]` - Directory to serve (static mode, defaults to `.`)
- `--proxy <url>` - Backend URL to proxy (proxy mode)

### Optional

- `--port <n>` - Server port (default: auto-detect)
- `--host <ip>` - Bind host (default: `localhost`)
  - `localhost` - Local only
  - `0.0.0.0` - Network accessible
- `--watch <paths>` - Additional watch paths (comma-separated)
- `--ignore <patterns>` - Glob patterns to ignore (comma-separated)

### Global Flags

- `--json` - Output in JSON format
- `--debug` - Enable debug logging

## Auto-Detection

### Port Selection

When `--port` is not specified (or `--port 0`), webctl automatically selects an available port:

1. Tries common dev ports: 3000, 8080, 8000, 5000, 4000
2. Falls back to OS-assigned port if all are busy

### Browser Navigation

When the server starts, webctl automatically:

1. Navigates the browser to the server URL
2. Configures hot reload for file changes (static mode)

## Hot Reload

Hot reload is automatic in static mode and works by:

1. **File Watcher**: Monitors directory for changes using fsnotify
2. **Debouncing**: Groups rapid changes (300ms window)
3. **CDP Reload**: Calls `Page.reload()` via Chrome DevTools Protocol
4. **No Injection**: Pure CDP, no scripts injected into page

### Watched Files

By default, the served directory is watched. Customize with flags:

```bash
# Watch additional directories
webctl serve ./public --watch ../src/

# Watch multiple paths
webctl serve ./public --watch ../src/,../assets/
```

### Ignored Patterns

These are automatically ignored:

- Hidden files (`.git`, `.env`, etc.)
- `node_modules/`
- `vendor/`
- `__pycache__/`

Add custom patterns:

```bash
# Ignore specific patterns
webctl serve . --ignore "*.tmp,*.swp,*.bak"

# Ignore build artifacts
webctl serve ./public --ignore "*.map,*.min.js"
```

## Integration with webctl Commands

While the server is running, use all webctl debugging commands:

```bash
# Start server
webctl serve ./public

# In another terminal (or REPL)
webctl console                       # Monitor console logs
webctl network --status 4xx          # Monitor network errors
webctl html --select "#app"          # Inspect rendered HTML
webctl css computed ".button"        # Debug computed styles
webctl screenshot                    # Capture page state
```

## Auto-Start Behavior

`webctl serve` automatically starts the daemon if it's not already running:

```bash
# One command to start everything
webctl serve ./public

# Equivalent to:
# webctl start
# webctl serve ./public
```

The serve command will:
1. Start the daemon (if not running)
2. Launch the browser (auto-detects available CDP port)
3. Start the web server (auto-detects available port)
4. Navigate browser to server URL
5. Run until Ctrl+C

Note: If the default CDP port (9222) is in use, the browser will automatically use the next available port (9223, 9224, etc.)

## Typical Workflows

### Static Site Development

```bash
# Single command - starts daemon + server
webctl serve ./public

# Edit files in your editor
# Browser auto-reloads on save

# In another terminal: Debug as needed
webctl console
webctl network
```

### API Integration Development

```bash
# Terminal 1: Start backend
cd backend && npm start  # Running on localhost:3000

# Terminal 2: Start proxy
webctl serve --proxy localhost:3000 --port 8080

# Browser opens at http://localhost:8080
# Edit frontend code, hot reload
# Debug API calls with webctl network
```

### Network Testing

```bash
# Start server accessible from network
webctl serve ./public --host 0.0.0.0 --port 3000

# Access from mobile device
# Open http://<your-ip>:3000 in mobile browser

# Debug mobile issues
webctl console
webctl screenshot
```

## Output

### Text Mode (default)

```
Server started: http://localhost:3000
Mode: static
Directory: ./public
Port: 3000

Watching for file changes (hot reload enabled)

Press Ctrl+C or run 'webctl stop' to stop the server
```

### JSON Mode

```bash
webctl serve ./public --json
```

```json
{
  "ok": true,
  "mode": "static",
  "url": "http://localhost:3000",
  "port": 3000
}
```

## Stopping the Server

The server stops when:

1. **Ctrl+C** in the terminal running `webctl start`
2. **`webctl stop`** command
3. **Daemon shutdown** (any reason)

The server is tied to the daemon lifecycle.

## Error Cases

### Directory Not Found

```
Error: directory does not exist: ./missing
```

**Solution**: Verify the directory path:

```bash
ls -la ./missing
webctl serve ./correct-path
```

### Port Already in Use

When using `--port`, if the port is busy:

```
Error: failed to listen on localhost:3000: address already in use
```

**Solution**: Use a different port or let webctl auto-detect:

```bash
webctl serve ./public --port 3001
# or
webctl serve ./public  # auto-detect
```

### Server Already Running

```
Error: server already running
```

**Solution**: Stop the existing server first:

```bash
webctl stop
webctl start
webctl serve ./public
```

## Technical Details

### Server Stack

- HTTP server: Go `net/http`
- File watching: `fsnotify/fsnotify`
- Reverse proxy: `net/http/httputil.ReverseProxy`
- Hot reload: Chrome DevTools Protocol

### Performance

- **Startup**: < 100ms
- **Hot reload latency**: ~300-500ms (debounce + reload)
- **File watching**: Native OS events (inotify on Linux)
- **Memory**: Minimal (no buffering, streaming)

### Security

**Development only**: This server is designed for local development and should not be used in production:

- No HTTPS
- No authentication
- No rate limiting
- No input validation beyond path traversal prevention
- Disables browser caching

For production, use a proper web server (nginx, Apache, Caddy).

## Common Patterns

### React/Vue/Angular Development

```bash
# Terminal 1: Start dev server with hot module replacement
npm run dev  # Usually runs on localhost:3000

# Terminal 2: Proxy with webctl for debugging
webctl start
webctl serve --proxy localhost:3000 --port 8080

# Browser opens at http://localhost:8080
# Use webctl commands to debug
```

### Static Site with Build Process

```bash
# Build once
npm run build  # outputs to ./dist

# Serve built files
webctl serve ./dist

# In watch mode (external build tool)
# Terminal 1: Build watcher
npm run watch  # rebuilds to ./dist on changes

# Terminal 2: Serve
webctl serve ./dist  # auto-reloads when ./dist changes
```

### Multi-Page Application

```bash
# Serve root directory
webctl serve ./public

# All pages available
# http://localhost:3000/index.html
# http://localhost:3000/about.html
# http://localhost:3000/contact.html

# Subdirectories work too
# http://localhost:3000/blog/post1.html
```

## See Also

- [`webctl start`](./start.md) - Start daemon and browser
- [`webctl stop`](./stop.md) - Stop daemon, browser, and server
- [`webctl console`](./console.md) - Monitor console logs
- [`webctl network`](./network.md) - Monitor network requests
- [`webctl reload`](./reload.md) - Manually reload page
