# webctl Agent Help

stdout is token efficient

Use --json for structured data selection

## Quick Start

```
webctl start &  # Start daemon in background (or run in separate shell), must stay running
webctl navigate https://example.com
webctl ready
webctl html --select "#main"
webctl css
webctl console
webctl network
webctl cookies
webctl screenshot save
webctl click "button.login"
webctl type "#username" "user@example.com"
webctl type "#password" "secret" --key Enter
webctl ready --network-idle
webctl console --type error
webctl html --find "Welcome"
webctl navigate https://example.com/dashboard
webctl ready
webctl eval "document.title"
webctl back
webctl forward
webctl reload
webctl target              # List tabs
webctl target "Dashboard"  # Switch to tab by title
webctl clear console
webctl stop
```

## Global Flags

```
--debug        Enable verbose debug output
--json         Output in JSON format
--no-color     Disable color output
```

## Command Flags

```
start:
  --headless         Run browser in headless mode
  --port <port>      CDP port (default: 9222)
navigate, reload, back, forward:
  --wait             Wait for page load completion
  --timeout <sec>    Timeout in seconds (default: 60)
html, css:
  --select, -s <sel> Filter to element(s)
  --find, -f <text>  Search for text
  --raw              Skip formatting
html with --find:
  --before, -B <n>   Show N lines before match
  --after, -A <n>    Show N lines after match
  --context, -C <n>  Show N lines before and after
console:
  --find, -f <text>  Search for text
  --raw              Skip formatting
  --type <type>      Filter by type (log, error, warn, info, debug)
  --head <n>         First N entries
  --tail <n>         Last N entries
  --range <n-m>      Entries N through M (1-indexed, inclusive)
network:
  --find, -f <text>  Search in URLs and bodies
  --raw              Skip formatting
  --type <type>      CDP resource type (xhr, fetch, document, script, etc)
  --method <method>  HTTP method (GET, POST, PUT, DELETE, etc)
  --status <code>    Status code or range (200, 4xx, 5xx, 200-299)
  --url <pattern>    URL regex pattern
  --mime <type>      MIME type (application/json, text/html, etc)
  --min-duration     Minimum duration (1s, 500ms, 100ms)
  --min-size <n>     Minimum size in bytes
  --failed           Only failed requests
  --max-body-size    Max body size before truncation (default: 100KB)
  --head <n>         First N entries
  --tail <n>         Last N entries
  --range <n-m>      Entries N through M
cookies:
  --find, -f <text>  Search in names and values
  --raw              Skip formatting
  --domain <domain>  Filter by domain
  --name <name>      Filter by exact name
type:
  --key <key>        Send key after text (Enter, Tab, Escape, etc)
  --clear            Clear existing content
key:
  --ctrl             Hold Ctrl modifier
  --alt              Hold Alt modifier
  --shift            Hold Shift modifier
  --meta             Hold Meta/Command modifier
ready:
  --timeout <dur>    Maximum wait time (default: 60s)
  --network-idle     Wait for network idle (500ms quiet)
  --eval <expr>      Wait for JS expression to be truthy
```

## Help Topics

Use webctl help agents <topic> for detailed guidance.

```
webctl help agents workflow
webctl help agents observe
webctl help agents interact
webctl help agents wait
webctl help agents errors
webctl help agents output
webctl help agents serve
```

All topics combined (use when you need complete reference):

```
webctl help agents all
```

