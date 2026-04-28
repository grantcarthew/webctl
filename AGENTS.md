# webctl

CLI tool for browser automation and debugging, designed for AI agents. Captures DevTools data (console logs, network requests, JS errors) via a persistent daemon that buffers CDP events.

See <https://agents.md/> for the full AGENTS.md specification as this project matures.

## Status

Under active development.

## Project Workflow

When completing a project and starting the next:

1. Mark project status as Done, set completion date
2. Replace `project.md` content with the next project (drawn from the roadmap or backlog)
3. Update Active Project below to reference the new project

## Active Project

The active project lives in `project.md` at the repository root.

- Active Project: Start Attach Mode (see project.md)

## Quick Reference

```bash
webctl start [--headless] [--port 9222]
webctl stop
webctl status

webctl serve <directory>                # Static file server
webctl serve --proxy <url>              # Reverse proxy server

webctl navigate <url>
webctl reload
webctl back
webctl forward

# Tab management
webctl tab                          # List open tabs
webctl tab switch <query>           # Switch active tab and foreground it
webctl tab new [url]                # Open a new tab (about:blank if no url)
webctl tab close [query]            # Close a tab (active tab if no query)

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

## Workflow

- Read `.ai/workflow.md` for the feature development process
- Read `.ai/docs/project-writing-guide.md` for project documentation
