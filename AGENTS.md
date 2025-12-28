# webctl

CLI tool for browser automation and debugging, designed for AI agents. Captures DevTools data (console logs, network requests, JS errors) via a persistent daemon that buffers CDP events.

See <https://agents.md/> for the full AGENTS.md specification as this project matures.

## Status

Under active development.

## Active Project

Projects are stored in the docs/projects/ directory. Update this when starting a new project.

- Active Project: None
- Design Record: None

## Completed Projects


Completed projects are in docs/projects/completed/
- P-017: CSS Commands (2025-12-28)
- P-018: Browser Connection Failure Handling (2025-12-27)
- P-015: HTML Formatting for Find and HTML Commands (2025-12-26)
- P-013: Find Command (2025-12-25)
- P-010: Ready Command Extensions (2025-12-25)
- P-014: Terminal Colors (2025-12-24)
- P-012: Text Output Format (2025-12-24)
- P-011: CDP Navigation Debugging (2025-12-23)
- P-009: Design Review & Validation of P-008 Commands (2025-12-24)
- P-008: Navigation & Interaction Commands (2025-12-23)
- P-007: Observation Commands (2025-12-23)
- P-006: CLI Framework (2025-12-15)
- P-005: Daemon & IPC (2025-12-15)
- P-004: Browser Launch (2025-12-15)
- P-003: CDP Core Library (2025-12-15)
- P-002: Project Definition (2025-12-11)
- P-001: Project Initialization (2025-12-11)

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
webctl css save [selector]
webctl css computed <selector>
webctl css get <selector> <property>
webctl css inject <css>
webctl eval <js-expression>
webctl cookies
webctl find <text> [-E] [-c] [--limit N]

webctl click <selector>
webctl type <selector> <text>
webctl select <selector> <value>
webctl scroll <selector|position>

webctl ready [selector] [--network-idle] [--eval "condition"]

webctl clear [console|network]
```

## Tech Stack

- Language: Go
- Browser control: CDP (Chrome DevTools Protocol)
- IPC: Unix socket (local), TCP (remote)
- Output: Text format (default), JSON (--json flag)

---

## Documentation Driven Development (DDD)

This project uses Documentation Driven Development. Design decisions are documented in Design Records (DRs) before or during implementation.

For complete DR writing guidelines: See [docs/design/dr-writing-guide.md](docs/design/dr-writing-guide.md)

Location: `docs/design/design-records/`
