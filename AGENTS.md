# webctl

CLI tool for browser automation and debugging, designed for AI agents. Captures DevTools data (console logs, network requests, JS errors) via a persistent daemon that buffers CDP events.

See <https://agents.md/> for the full AGENTS.md specification as this project matures.

## Status

Under active development.

## Active Project

Projects are stored in the docs/projects/ directory. Update this when starting a new project.

Active Project: docs/projects/p-007-observation-commands.md

Completed projects are in docs/projects/completed/

## Quick Reference

```bash
webctl browser [--headless] [--port 9222]
webctl start [--headless] [--attach :9222] [--listen :9444]
webctl stop
webctl status

webctl navigate <url>
webctl reload
webctl back
webctl forward

webctl console
webctl network
webctl screenshot
webctl html [selector]
webctl eval <js-expression>
webctl cookies

webctl click <selector>
webctl type <selector> <text>
webctl select <selector> <value>
webctl scroll <selector|position>

webctl wait-for <selector|condition>

webctl clear [console|network]
```

## Tech Stack

- Language: Go
- Browser control: CDP (Chrome DevTools Protocol)
- IPC: Unix socket (local), TCP (remote)
- Output: JSON

---

## Documentation Driven Development (DDD)

This project uses Documentation Driven Development. Design decisions are documented in Design Records (DRs) before or during implementation.

For complete DR writing guidelines: See [docs/design/dr-writing-guide.md](docs/design/dr-writing-guide.md)

Location: `docs/design/design-records/`
