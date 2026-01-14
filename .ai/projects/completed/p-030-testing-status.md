# P-030: Testing status Command

- Status: Complete
- Started: 2025-12-31
- Completed: 2026-01-07

## Overview

Test the webctl status command which reports daemon state including whether it's running, current URL, and page title.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-status.sh
```

## Code References

- internal/cli/status.go

## Command Signature

```
webctl status
```

No command-specific flags. Global flags apply:
- --debug: Enable debug output (global flag) - Note: no extra output for status
- --json: Output in JSON format (global flag)
- --no-color: Disable colored output (global flag)

## Test Checklist

When daemon running:
- [x] Check status shows "running: true"
- [x] Verify current URL displayed
- [x] Verify page title displayed
- [x] Test after navigation to different page

When daemon not running:
- [x] Check status shows "running: false"
- [x] Verify appropriate message/output

Output formats:
- [x] Default text format
- [x] JSON format with --json flag
- [x] No color format with --no-color flag

Various states:
- [x] Status immediately after start
- [x] Status during page navigation
- [x] Status with loaded page
- [x] Status with error page (404, etc.)

CLI vs REPL:
- [x] CLI: webctl status
- [x] REPL: status

## Notes

- Status can be checked from any terminal (not just daemon terminal)
- Should work in both CLI and REPL modes
- Provides quick overview of daemon state
- Useful for scripting and automation
- --debug flag shows no extra output (status is a simple IPC query)

## Enhancements Made During Testing

### 1. Improved "Not running" message
Changed from uninformative "Not running" to helpful "Not running (start with: webctl start)".

Files: `internal/cli/format/text.go`

### 2. REPL prompt refresh on session attach
Fixed intermittent issue where REPL prompt didn't show URL after daemon start. Added `refreshPrompt()` call in `handleTargetAttached()`.

Files: `internal/daemon/events.go`

### 3. HTTP status code in session list
Added HTTP status display for sessions (e.g., `* https://example.com/page (404)`). Looks up status from network buffer at query time.

Files:
- `internal/ipc/protocol.go` - Added Status field to PageSession
- `internal/daemon/handlers_observation.go` - Added enrichSessionsWithHTTPStatus()
- `internal/cli/format/text.go` - Added formatHTTPStatus() with color coding

Note: HTTP status only available if network entry is still in buffer.
