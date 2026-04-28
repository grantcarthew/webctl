# webctl

CLI tool for browser automation and debugging, designed for AI agents.

## Overview

webctl captures DevTools data (console logs, network requests, JS errors) that standard web fetching tools cannot access. CDP events are ephemeral - you must be listening when they occur. webctl solves this with a persistent daemon that buffers events for later query.

## Architecture

Daemon + stateless command model:

```bash
webctl start                    # Launch browser, buffer CDP events
webctl console                  # Query buffered console logs
webctl click ".selector"        # Send commands
webctl reload                   # Refresh current page
webctl stop                     # Clean shutdown
```

## Status

Under active development.

### Implemented

- Daemon with CDP event buffering (console, network)
- IPC via Unix socket
- CLI framework (Cobra) with abbreviation expansion and JSON output
- Lifecycle: `start`, `stop` (with `--force` reaper), `status`, `clear`
- Navigation: `navigate`, `reload`, `back`, `forward`
- Tabs: `tab` (list, switch, new, close)
- Observation: `html`, `css`, `console`, `network`, `cookies`, `screenshot`, `eval`
- Interaction: `click`, `type`, `select`, `scroll`, `focus`, `key`
- Synchronisation: `ready` (page load, selector, network idle, JS condition)
- Local server: `serve` (static files or reverse proxy with hot reload)

### In Progress

- `webctl start --attach`: connect to an existing CDP endpoint instead of launching a new browser

## Commands

| Category | Commands |
|----------|----------|
| Lifecycle | start, stop, status, clear |
| Navigation | navigate, reload, back, forward |
| Tabs | tab |
| Observation | html, css, console, network, cookies, screenshot, eval |
| Interaction | click, type, select, scroll, focus, key |
| Synchronisation | ready |
| Local server | serve |

## Agent Workflow

```bash
webctl start --headless &
webctl navigate https://localhost:3000
webctl console                             # Check for JS errors
# ... agent fixes code ...
webctl reload
webctl console                             # Verify fix
webctl stop
```

## Companion Packages

The following software packages and systems work well when used side by side with webctl:

- [Snag](https://github.com/grantcarthew/snag): Snag web pages via Chromium/Chrome to markdown
- [Kagi](https://github.com/grantcarthew/kagi): CLI access to the Kagi FastGPT internet search
- [agentation](https://agentation.dev/): Point at problems, not code

## License

MPL-2.0
