# webctl

CLI tool for browser automation and debugging, designed for AI agents. Captures DevTools data (console logs, network requests, JS errors) via a persistent daemon that buffers CDP events.

See <https://agents.md/> for the full AGENTS.md specification as this project matures.

## Status

Under active development.

## Active Project

Projects are stored in the docs/projects/ directory. Update this when starting a new project.

- Active Project: docs/projects/p-051-observation-output-refactor.md
- Design Record: N/A

When projects are completed, move them to docs/projects/completed/, update docs/projects/README.md, and update the active project above to the next project.

## Completed Projects

Completed projects are in docs/projects/completed/
- P-033: Testing forward Command (2026-01-07)
- P-032: Testing back Command (2026-01-07)
- P-031: Testing reload Command (2026-01-07)
- P-030: Testing status Command (2026-01-07)
- P-029: Testing serve Command (2026-01-06)
- P-028: Testing navigate Command (2026-01-06)
- P-027: Testing stop Command (2026-01-06)
- P-026: Testing start Command (2026-01-04)
- P-025: Interactive Test Suite (2026-01-03)
- P-016: CLI Serve Command (2025-12-30)
- P-024: Cookies Command Implementation (2025-12-30)
- P-023: Network Command Implementation (2025-12-30)
- P-022: Console Command Implementation (2025-12-29)
- P-021: CSS Command Implementation (2025-12-28)
- P-020: HTML Command Implementation (2025-12-28)
- P-019: Observation Commands Interface Redesign (2025-12-28)
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

webctl serve <directory>                # Static file server
webctl serve --proxy <url>              # Reverse proxy server

webctl navigate <url>
webctl reload
webctl back
webctl forward

# Observation commands (default: stdout, save for files)
webctl html                         # Output to stdout
webctl html save                    # Save to temp
webctl html save <path>             # Save to custom path
webctl html --select <selector> --find <text>  # Filter and search

webctl css                          # Output stylesheets to stdout
webctl css save                     # Save to temp
webctl css save <path>              # Save to custom path
webctl css --select <selector> --find <text>
webctl css computed <selector>      # Computed styles (stdout)
webctl css get <selector> <property> # Single property (stdout)

webctl console                      # Output logs to stdout
webctl console save                 # Save to temp
webctl console save <path>          # Save to custom path
webctl console --type <type> --find <text>

webctl network                      # Output requests to stdout
webctl network save                 # Save to temp
webctl network save <path>          # Save to custom path
webctl network --status <code> --method <method> --find <text>

webctl cookies                      # Output cookies to stdout
webctl cookies save                 # Save to temp
webctl cookies save <path>          # Save to custom path
webctl cookies --domain <domain> --find <text>
webctl cookies set <name> <value>
webctl cookies delete <name>

webctl screenshot                   # Save to temp (binary output)
webctl screenshot save <path>       # Save to custom path

webctl eval <js-expression>

# Interaction commands
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
