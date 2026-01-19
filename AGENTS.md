# webctl

CLI tool for browser automation and debugging, designed for AI agents. Captures DevTools data (console logs, network requests, JS errors) via a persistent daemon that buffers CDP events.

See <https://agents.md/> for the full AGENTS.md specification as this project matures.

## Status

Under active development.

## Project Workflow

When completing a project and starting the next:

1. Mark project status as Done, set completion date
2. Move project file to `.ai/projects/completed/`
3. Create next project file (from design record or roadmap)
4. Update Active Project below to reference new project
5. Update `.ai/projects/README.md` table entry

## Active Project

Projects are stored in `.ai/projects/`.

- Active Project: None (p-061 completed)
- Next: p-062 CLI Interaction Tests (per DR-032) or other priorities
- Design Record: .ai/design/design-records/dr-032-test-framework-architecture.md

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
