# webctl Browser Automation CLI

- Use `webctl status` before `webctl start` to check whether a daemon is already running
- Use `webctl start &` to launch Chromium and start the daemon (or run in a separate shell); the daemon must stay running
- stdout is token-efficient; use `--json` only when output must be parsed programmatically

## Core Commands

```
# Lifecycle
webctl start [--headless] [--port <port>]
webctl status
webctl stop

# Navigation
webctl navigate <url> [--wait]
webctl reload [--wait]
webctl back [--wait]
webctl forward [--wait]

# Tabs
webctl tab
webctl tab switch <query>
webctl tab new [url]
webctl tab close [query]

# Observation
webctl html [save [path]]
webctl css [save [path]]
webctl css computed <selector>
webctl css get <selector> <property>
webctl css inline <selector>
webctl css matched <selector>
webctl console [save [path]]
webctl network [save [path]]
webctl cookies [save [path]]
webctl cookies set <name> <value>
webctl cookies delete <name>
webctl screenshot save [path] [--full-page]
webctl eval <js-expression>

# Interaction
webctl click <selector>
webctl type <selector> <text>
webctl select <selector> <value>
webctl scroll <selector|--to x,y|--by x,y>
webctl focus <selector>
webctl key <key>

# Synchronization
webctl ready [selector] [--network-idle] [--eval <js>]

# Buffers
webctl clear [console|network]

# Local Server
webctl serve [directory]
webctl serve --proxy <url>
```

For flag detail, use `webctl <command> --help`.

## Global Flags

```
--debug        Enable verbose debug output
--json         Output in JSON format
--no-color     Disable color output
```

## Help Topics

Use `webctl help <topic>` for detailed guidance.

```
webctl help workflow
webctl help observe
webctl help interact
webctl help wait
webctl help errors
webctl help output
webctl help serve
webctl help all
```
