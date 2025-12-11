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

## License

MPL-2.0
