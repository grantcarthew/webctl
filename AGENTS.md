# webctl

CLI tool for browser automation and debugging, designed for AI agents. Captures DevTools data (console logs, network requests, JS errors) via a persistent daemon that buffers CDP events.

See <https://agents.md/> for the full AGENTS.md specification as this project matures.

## Status

Under active development.

## Active Project

Projects are stored in `.ai/projects/`. Update this section when starting a new project.

- Active Project: .ai/projects/p-054-force-stop-cleanup.md
- Design Record: .ai/design/design-records/dr-031-force-stop-cleanup.md

When projects are completed, move them to `.ai/projects/completed/`, update `.ai/projects/README.md`, and update the active project above to the next project.

## Completed Projects

Completed projects are in `.ai/projects/completed/`
- p-036: Testing console Command (2026-01-14)
- p-035: Testing css Command (2026-01-13)
- p-034: Testing html Command (2026-01-12)
- p-052: CSS Command Redesign (2026-01-12)
- p-051: Observation Commands Output Refactor (2026-01-09)
- p-033: Testing forward Command (2026-01-07)
- p-032: Testing back Command (2026-01-07)
- p-031: Testing reload Command (2026-01-07)
- p-030: Testing status Command (2026-01-07)
- p-029: Testing serve Command (2026-01-06)
- p-028: Testing navigate Command (2026-01-06)
- p-027: Testing stop Command (2026-01-06)
- p-026: Testing start Command (2026-01-04)
- p-025: Interactive Test Suite (2026-01-03)
- p-016: CLI Serve Command (2025-12-30)
- p-024: Cookies Command Implementation (2025-12-30)
- p-023: Network Command Implementation (2025-12-30)
- p-022: Console Command Implementation (2025-12-29)
- p-021: CSS Command Implementation (2025-12-28)
- p-020: HTML Command Implementation (2025-12-28)
- p-019: Observation Commands Interface Redesign (2025-12-28)
- p-017: CSS Commands (2025-12-28)
- p-018: Browser Connection Failure Handling (2025-12-27)
- p-015: HTML Formatting for Find and HTML Commands (2025-12-26)
- p-013: Find Command (2025-12-25)
- p-010: Ready Command Extensions (2025-12-25)
- p-014: Terminal Colors (2025-12-24)
- p-012: Text Output Format (2025-12-24)
- p-011: CDP Navigation Debugging (2025-12-23)
- p-009: Design Review & Validation of p-008 Commands (2025-12-24)
- p-008: Navigation & Interaction Commands (2025-12-23)
- p-007: Observation Commands (2025-12-23)
- p-006: CLI Framework (2025-12-15)
- p-005: Daemon & IPC (2025-12-15)
- p-004: Browser Launch (2025-12-15)
- p-003: CDP Core Library (2025-12-15)
- p-002: Project Definition (2025-12-11)
- p-001: Project Initialization (2025-12-11)

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
# Note: Use trailing slash for directories (./dir/) vs files (./file.ext)
webctl html                         # Output to stdout
webctl html save                    # Save to temp
webctl html save <path>             # Save to file or dir/ (trailing slash)
webctl html --select <selector> --find <text>  # Filter and search

webctl css                          # Output stylesheets to stdout
webctl css save                     # Save to temp
webctl css save <path>              # Save to custom path
webctl css --select <selector>      # Filter rules by selector pattern
webctl css --find <text>            # Search for text within CSS
webctl css computed <selector>      # Computed styles for element(s)
webctl css get <selector> <property> # Single property value
webctl css inline <selector>        # Inline style attributes
webctl css matched <selector>       # Matched CSS rules for element

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

- Read `.ai/workflow.md` for feature development process
- Read `.ai/projects/p-writing-guide.md` for project documentation
- Read `.ai/design/dr-writing-guide.md` for design record format
- Design records are in `.ai/design/design-records/`
