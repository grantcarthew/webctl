# P-029: Testing serve Command

- Status: In Progress
- Started: 2025-12-31

## Overview

Test the webctl serve command which starts a development server with hot reload capabilities. This command has two modes: static file serving and proxy mode, and auto-starts the daemon if not running.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-serve.sh
```

## Code References

- internal/cli/serve.go

## Command Signature

```
webctl serve [directory] [--proxy url]
```

Flags:
- --proxy <url>: Proxy mode (proxy requests to backend server)
- --port <number>: Server port (auto-detect if not specified)
- --host <address>: Network binding (default: localhost)
- --watch <paths>: Custom watch paths for hot reload
- --ignore <patterns>: Ignore patterns for file watching
- --headless: Browser headless mode (from auto-start)
- --debug: Enable debug output (global flag)
- --json: Output in JSON format (global flag)
- --no-color: Disable colored output (global flag)

## Test Checklist

Static mode - Basic functionality:
- [ ] Serve current directory (default)
- [ ] Serve specified directory
- [ ] Verify auto-start of daemon and browser
- [ ] Verify browser navigates to served URL
- [ ] Verify file serving works

Static mode - File watching and hot reload:
- [ ] Modify HTML file, verify auto-reload
- [ ] Modify CSS file, verify auto-reload
- [ ] Modify JS file, verify auto-reload
- [ ] Test custom watch paths (--watch)
- [ ] Test ignore patterns (--ignore)

Proxy mode:
- [ ] Proxy to localhost backend (--proxy localhost:3000)
- [ ] Proxy to full URL (--proxy http://api.example.com)
- [ ] Verify requests forwarded correctly
- [ ] Verify daemon auto-starts in proxy mode

Port and host options:
- [ ] Auto-detect available port (default)
- [ ] Custom port (--port 3000)
- [ ] Network binding (--host 0.0.0.0)
- [ ] Verify localhost vs network access

Auto-start behavior:
- [ ] Serve when daemon not running (auto-start)
- [ ] Serve when daemon already running (use existing)
- [ ] Headless mode (--headless flag)

Server lifecycle:
- [ ] Start server
- [ ] Stop with Ctrl+C
- [ ] Stop with webctl stop command
- [ ] Verify clean shutdown

Integration with webctl commands:
- [ ] Use console command while serving
- [ ] Use network command while serving
- [ ] Use html command while serving
- [ ] Verify all commands work during serve

Output formats:
- [ ] Default text output
- [ ] JSON output (--json)
- [ ] No color output (--no-color)
- [ ] Debug output (--debug)

Error cases:
- [ ] Serve non-existent directory
- [ ] Serve with port in use
- [ ] Proxy to unreachable backend
- [ ] Invalid --watch or --ignore patterns

CLI vs REPL:
- [ ] CLI: webctl serve ./public
- [ ] REPL: serve ./public (from already running daemon)

## Notes

- Serve command is unique with auto-start functionality
- Can serve static files or proxy to backend
- Includes file watching and hot reload for static mode
- Integrates with all other webctl commands during operation
- Primary use case for development workflow

## Issues Discovered

(Issues will be documented here during testing)
