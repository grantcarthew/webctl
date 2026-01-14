# P-035: Debug Output

- Status: Completed
- Started: 2026-01-13
- Completed: 2026-01-13

## Overview

Add comprehensive debug output throughout the CLI codebase. Currently, only 2 debug calls exist in the CLI (in `html.go` and `css.go` for formatting failures). The `--debug` flag should provide useful diagnostic information for troubleshooting and development.

## Goals

1. Add consistent debug output to all command execution paths
2. Log IPC requests and responses
3. Include timing information for operations
4. Show filter/selector parameters being applied
5. Log file operations (paths, sizes)
6. Establish a consistent debug message format aligned with daemon output

## Scope

**In Scope:**

- All 25 CLI command files in `internal/cli/`
- IPC request/response logging
- Filter and selector parameter logging
- File I/O operations
- Timing for key operations
- Update `debugf()` to include timestamps

**Out of Scope:**

- Daemon-side debug output (already comprehensive)
- Log levels beyond debug (info, warn, error hierarchy)
- Log file output (debug goes to stderr only)

## Success Criteria

- [x] Every command produces debug output when `--debug` is set
- [x] IPC requests show command and parameters
- [x] IPC responses show status and data size
- [x] Filter operations log what was filtered and result counts
- [x] File saves log path and bytes written
- [x] Debug format is consistent across all commands
- [x] Debug output does not appear without `--debug` flag
- [x] Timestamps included in all debug messages

## Existing Infrastructure

The following debug infrastructure already exists:

| Component | Location | Description |
|-----------|----------|-------------|
| `Debug` variable | `root.go:32` | Global `var Debug bool` |
| `--debug` flag | `root.go:50` | Registered as persistent flag on root command |
| `debugf()` helper | `root.go:59-64` | Writes to stderr with `[DEBUG]` prefix |
| `Request.Debug` | `protocol.go:18` | IPC field for per-request debug |
| Debug propagation | `executor/ipc.go:40-42` | Automatically sets `Request.Debug` from CLI flag |

The daemon has a more sophisticated `debugf()` with timestamps (`daemon.go:96-100`). The CLI version should be updated to match.

## Debug Message Format

### Current Format (CLI)

```
[DEBUG] message
```

### Target Format (matches daemon)

```
[DEBUG] [HH:MM:SS.mmm] [CATEGORY] message
```

### Categories

| Category | Purpose | Example |
|----------|---------|---------|
| `REQUEST` | IPC request sent | `[REQUEST] cmd=html selector="#main"` |
| `RESPONSE` | IPC response received | `[RESPONSE] ok=true size=4523 bytes` |
| `FILTER` | Filter/selector applied | `[FILTER] --find "login": 312 -> 5 lines` |
| `FILE` | File I/O operation | `[FILE] wrote 4523 bytes to /tmp/webctl-html/...` |
| `TIMING` | Operation duration | `[TIMING] total: 35ms` |
| `PARAM` | Parameter resolution | `[PARAM] selector="#main" find="" raw=false` |

### Example Debug Session

```
$ webctl html --select "#main" --find "login" --debug
[DEBUG] [14:32:05.100] [PARAM] selector="#main" find="login" raw=false
[DEBUG] [14:32:05.101] [REQUEST] cmd=html params={"selector":"#main"}
[DEBUG] [14:32:05.145] [RESPONSE] ok=true size=4523 bytes (44ms)
[DEBUG] [14:32:05.146] [FILTER] --find "login": 312 -> 5 lines
[DEBUG] [14:32:05.146] [TIMING] total: 46ms
<div id="main">
  <form class="login-form">...
</form>
</div>
```

## Technical Approach

### Phase 1: Update Debug Infrastructure

Update `debugf()` in `root.go` to include timestamps:

```go
// debugf logs a debug message if debug mode is enabled.
// Format: [DEBUG] [HH:MM:SS.mmm] [CATEGORY] message
func debugf(category, format string, args ...any) {
    if Debug {
        timestamp := time.Now().Format("15:04:05.000")
        fmt.Fprintf(os.Stderr, "[DEBUG] [%s] [%s] "+format+"\n",
            append([]any{timestamp, category}, args...)...)
    }
}
```

Add category-specific helpers:

```go
func debugRequest(cmd string, params any) {
    if Debug {
        // Format params as key=value pairs, not full JSON
        debugf("REQUEST", "cmd=%s %s", cmd, formatParamsSummary(params))
    }
}

func debugResponse(ok bool, dataSize int, duration time.Duration) {
    if Debug {
        debugf("RESPONSE", "ok=%v size=%d bytes (%dms)", ok, dataSize, duration.Milliseconds())
    }
}

func debugFilter(name string, before, after int) {
    if Debug {
        debugf("FILTER", "%s: %d -> %d", name, before, after)
    }
}

func debugFile(operation, path string, size int) {
    if Debug {
        debugf("FILE", "%s %d bytes to %s", operation, size, path)
    }
}

func debugTiming(operation string, duration time.Duration) {
    if Debug {
        debugf("TIMING", "%s: %dms", operation, duration.Milliseconds())
    }
}

func debugParam(format string, args ...any) {
    if Debug {
        debugf("PARAM", format, args...)
    }
}
```

### Phase 2: Instrument Commands

Each command file needs debug calls at these points:

1. **After flag resolution** - Log resolved parameter values
2. **Before IPC request** - Log command and params summary
3. **After IPC response** - Log ok status, data size, duration
4. **After filtering** - Log filter criteria and before/after counts
5. **After file write** - Log path and bytes written
6. **At end of command** - Log total timing

### Command Files to Modify

| File | Debug Points | Notes |
|------|--------------|-------|
| `back.go` | IPC, timing | Simple navigation |
| `clear.go` | IPC, timing | Simple action |
| `click.go` | IPC, params, timing | Interaction |
| `console.go` | IPC, filter, file, timing | Observation with filters |
| `cookies.go` | IPC, filter, file, timing | Observation with filters |
| `css.go` | IPC, filter, file, timing | Observation with filters |
| `eval.go` | IPC, params, timing | Expression param |
| `find.go` | IPC, params, timing | Search params |
| `focus.go` | IPC, params, timing | Interaction |
| `forward.go` | IPC, timing | Simple navigation |
| `html.go` | IPC, filter, file, timing | Observation with filters |
| `key.go` | IPC, params, timing | Key modifiers |
| `navigate.go` | IPC, params, timing | URL param |
| `network.go` | IPC, filter, file, timing | Observation with filters |
| `ready.go` | IPC, params, timing | Wait conditions |
| `reload.go` | IPC, timing | Simple navigation |
| `screenshot.go` | IPC, file, timing | Binary file output |
| `scroll.go` | IPC, params, timing | Position params |
| `selectcmd.go` | IPC, params, timing | Interaction |
| `serve.go` | IPC, params, timing | Server params |
| `start.go` | Params, timing | No IPC (daemon startup) |
| `status.go` | IPC, timing | Simple query |
| `stop.go` | IPC, timing | Simple action |
| `target.go` | IPC, params, timing | Session selection |
| `type.go` | IPC, params, timing | Text input |

### Phase 3: Add Timing Support

Create a timing helper:

```go
type timer struct {
    start time.Time
    name  string
}

func startTimer(name string) *timer {
    return &timer{start: time.Now(), name: name}
}

func (t *timer) stop() time.Duration {
    return time.Since(t.start)
}

func (t *timer) log() {
    debugTiming(t.name, t.stop())
}
```

Usage pattern:

```go
func runNavigate(cmd *cobra.Command, args []string) error {
    t := startTimer("navigate")
    defer t.log()

    // ... command implementation
}
```

## Implementation Pattern

### Example: Instrumenting a Simple Command

**Before (click.go):**

```go
func runClick(cmd *cobra.Command, args []string) error {
    if !execFactory.IsDaemonRunning() {
        return outputError("daemon not running. Start with: webctl start")
    }

    exec, err := execFactory.NewExecutor()
    if err != nil {
        return outputError(err.Error())
    }
    defer exec.Close()

    params, err := json.Marshal(ipc.ClickParams{
        Selector: args[0],
    })
    if err != nil {
        return outputError(err.Error())
    }

    resp, err := exec.Execute(ipc.Request{
        Cmd:    "click",
        Params: params,
    })
    // ...
}
```

**After:**

```go
func runClick(cmd *cobra.Command, args []string) error {
    t := startTimer("click")
    defer t.log()

    if !execFactory.IsDaemonRunning() {
        return outputError("daemon not running. Start with: webctl start")
    }

    selector := args[0]
    debugParam("selector=%q", selector)

    exec, err := execFactory.NewExecutor()
    if err != nil {
        return outputError(err.Error())
    }
    defer exec.Close()

    params, err := json.Marshal(ipc.ClickParams{
        Selector: selector,
    })
    if err != nil {
        return outputError(err.Error())
    }

    debugRequest("click", fmt.Sprintf("selector=%q", selector))
    ipcStart := time.Now()

    resp, err := exec.Execute(ipc.Request{
        Cmd:    "click",
        Params: params,
    })

    debugResponse(resp.OK, len(resp.Data), time.Since(ipcStart))
    // ...
}
```

### Example: Instrumenting an Observation Command with Filters

**Pattern for html.go, css.go, console.go, network.go, cookies.go:**

```go
func getHTMLFromDaemon(cmd *cobra.Command) (string, error) {
    t := startTimer("getHTMLFromDaemon")
    defer t.log()

    // Flag resolution
    selector, _ := cmd.Flags().GetString("select")
    find, _ := cmd.Flags().GetString("find")
    raw, _ := cmd.Flags().GetBool("raw")

    debugParam("selector=%q find=%q raw=%v", selector, find, raw)

    exec, err := execFactory.NewExecutor()
    if err != nil {
        return "", err
    }
    defer exec.Close()

    params, err := json.Marshal(ipc.HTMLParams{Selector: selector})
    if err != nil {
        return "", err
    }

    debugRequest("html", fmt.Sprintf("selector=%q", selector))
    ipcStart := time.Now()

    resp, err := exec.Execute(ipc.Request{Cmd: "html", Params: params})

    debugResponse(resp.OK, len(resp.Data), time.Since(ipcStart))

    // ... parse response ...

    // Filter logging
    if find != "" {
        beforeCount := strings.Count(html, "\n") + 1
        html, err = filterHTMLByText(html, find, before, after)
        afterCount := strings.Count(html, "\n") + 1
        debugFilter(fmt.Sprintf("--find %q", find), beforeCount, afterCount)
    }

    return html, nil
}
```

### Example: Instrumenting File Save

```go
func writeHTMLToFile(path, html string) error {
    // ... write logic ...

    if err := os.WriteFile(path, []byte(html), 0644); err != nil {
        return fmt.Errorf("failed to write HTML: %v", err)
    }

    debugFile("wrote", path, len(html))
    return nil
}
```

## Security Considerations

**Never log sensitive data:**

- Cookie values (log names only)
- Form input from `type` command (log selector only)
- Eval expressions (could contain secrets)
- Full request/response JSON (could contain tokens)

**Safe to log:**

- Command names
- CSS selectors
- URLs (but truncate query strings if long)
- Data sizes
- Timing information
- Filter criteria

## Testing

Add tests to verify debug output appears correctly:

```go
func TestDebugOutput(t *testing.T) {
    // Capture stderr
    old := os.Stderr
    r, w, _ := os.Pipe()
    os.Stderr = w

    Debug = true
    debugf("REQUEST", "test message")
    Debug = false

    w.Close()
    os.Stderr = old

    var buf bytes.Buffer
    io.Copy(&buf, r)

    output := buf.String()
    if !strings.Contains(output, "[DEBUG]") {
        t.Error("expected [DEBUG] prefix")
    }
    if !strings.Contains(output, "[REQUEST]") {
        t.Error("expected [REQUEST] category")
    }
}
```

## Deliverables

1. Updated `debugf()` function with timestamp and category support
2. New debug helper functions (`debugRequest`, `debugResponse`, etc.)
3. Timer utility for operation timing
4. All 25 command files instrumented with debug calls
5. Tests for debug output format
6. Code comments documenting debug message format

## Implementation Order

1. Update `root.go` with new debug infrastructure
2. Add timer utility
3. Instrument simple commands first (status, stop, back, forward, reload)
4. Instrument navigation commands (navigate)
5. Instrument interaction commands (click, type, key, select, scroll, focus)
6. Instrument observation commands (html, css, console, network, cookies, screenshot)
7. Instrument remaining commands (eval, ready, find, clear, target, serve, start)
8. Add tests
9. Manual testing with `--debug` flag

## Verification

After implementation, verify with:

```bash
# Simple command
webctl status --debug

# Navigation with params
webctl navigate example.com --wait --debug

# Observation with filters
webctl html --select "body" --find "test" --debug

# Save operation
webctl screenshot save ./test.png --debug

# Should see no debug output without flag
webctl status
```

## Notes

- The daemon already has comprehensive debug output via its own `debugf()`
- The `Request.Debug` field propagates CLI debug flag to daemon
- When `--debug` is used, both CLI and daemon debug output will appear
- Debug output goes to stderr, command output goes to stdout
