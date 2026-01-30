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

Under active development. See `docs/design/design-records/` for design decisions.

### Implemented

- Daemon with CDP event buffering (console, network)
- IPC via Unix socket
- CLI framework (Cobra)
- Lifecycle commands: `start`, `stop`, `status`, `clear`

### In Progress

- Observation commands (P-007): `console`, `network`, `screenshot`, etc.

## Commands

| Category | Commands |
|----------|----------|
| Observation | console, network, screenshot, html, eval, cookies |
| Navigation | navigate, reload, back, forward |
| Interaction | click, type, select, scroll |
| Synchronisation | wait-for |
| Lifecycle | start, stop, status, clear |

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

## Documentation

- Design Records: `docs/design/design-records/`
- Projects: `docs/projects/`

## Companion Packages

The following software packages and systems work well when used side by side with webctl:

- [Snag](https://github.com/grantcarthew/snag): Snag web pages via Chromium/Chrome to markdown
- [Kagi](https://github.com/grantcarthew/kagi): CLI access to the Kagi FastGPT internet search
- [agentation](https://agentation.dev/): Point at problems, not code

## License

MPL-2.0
