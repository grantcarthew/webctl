# Agent Workflow Patterns

Common automation patterns for webctl.

## Lifecycle

```
webctl start
webctl status
webctl stop
```

## Form Interaction

```
webctl navigate https://example.com/login
webctl ready
webctl type "#username" "user@example.com"
webctl type "#password" "secret"
webctl click "button[type=submit]"
webctl ready --network-idle
webctl html --find "Dashboard"
```

## Data Extraction

```
webctl navigate https://example.com
webctl ready
webctl html --select "#main"
webctl console --type error
webctl network --status 4xx
webctl cookies --domain example.com
```

## Error Checking

```
webctl console --type error
webctl network --status 4xx
webctl network --status 5xx
webctl network --failed
```

## Visual Inspection

```
webctl screenshot save
webctl css --select ".header"
webctl css computed ".button"
webctl css get ".button" "background-color"
```
