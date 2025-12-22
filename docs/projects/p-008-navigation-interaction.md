# P-008: Navigation & Interaction Commands

- Status: In Progress
- Started: 2025-12-19
- Design Record: DR-013

## Overview

Implement commands for navigating the browser and interacting with page elements. Includes navigation (navigate, reload, back, forward), interaction (click, type, focus, key, select, scroll), and utility (ready) commands.

## Goals

1. Navigate to URLs and through history
2. Click elements by CSS selector
3. Type text into input fields with optional key press
4. Focus elements and send keyboard keys
5. Select dropdown options
6. Scroll to elements or positions
7. Wait for page load completion

## Scope

In Scope (11 commands):

Navigation:

- `navigate` - Navigate to URL, wait for frameNavigated
- `reload` - Reload page, optional cache bypass
- `back` - Navigate to previous history entry
- `forward` - Navigate to next history entry

Interaction:

- `click` - Click element by CSS selector (CDP mouse events)
- `type` - Type text, optional selector/key/clear flags
- `focus` - Focus element by CSS selector
- `key` - Send keyboard key with modifier flags
- `select` - Select dropdown option (JS-based)
- `scroll` - Scroll element into view or to/by position

Utility:

- `ready` - Wait for page load (loadEventFired)

Out of Scope:

- Complex wait conditions (P-009: wait-for selector, network idle)
- Complex interactions (drag-drop, hover, double-click, right-click)
- File upload
- Iframe support (v1 limitation)

## Success Criteria

- [ ] `webctl navigate https://example.com` loads page, returns title
- [ ] `webctl reload` refreshes page
- [ ] `webctl reload --ignore-cache` hard refresh
- [ ] `webctl back` goes to previous page (error if none)
- [ ] `webctl forward` goes to next page (error if none)
- [ ] `webctl click ".button"` clicks element
- [ ] `webctl type "#input" "hello"` types text
- [ ] `webctl type "hello"` types into focused element
- [ ] `webctl type "#input" "hello" --key Enter` types and sends Enter
- [ ] `webctl type "#input" "new" --clear` clears then types
- [ ] `webctl focus "#input"` focuses element
- [ ] `webctl key Enter` sends Enter key
- [ ] `webctl key a --ctrl` sends Ctrl+A
- [ ] `webctl select "#dropdown" "option1"` selects option
- [ ] `webctl scroll ".element"` scrolls element into view
- [ ] `webctl scroll --to 0,1000` scrolls to position
- [ ] `webctl scroll --by 0,500` scrolls by offset
- [ ] `webctl ready` waits for page load
- [ ] `webctl ready --timeout 10s` with custom timeout
- [ ] REPL prompt shows correct title after navigation commands

## Deliverables

CLI commands (internal/cli/):

- `navigate.go`
- `reload.go`
- `back.go`
- `forward.go`
- `click.go`
- `type.go`
- `focus.go`
- `key.go`
- `select.go`
- `scroll.go`
- `ready.go`

Daemon handlers:

- Add cases to `handleRequest()` switch in daemon.go
- Implement CDP sequences for each command

IPC types (internal/ipc/):

- Request/response types for new commands

## Technical Design

See DR-013 for full design details. Key decisions:

Navigation wait behavior:

- All navigation commands wait for `Page.frameNavigated` (not loadEventFired)
- Ensures REPL prompt displays correct title
- `ready` command waits for `Page.loadEventFired` when full load needed

Click implementation:

- CDP mouse events (not JS click)
- `DOM.getBoxModel` for coordinates
- `Input.dispatchMouseEvent` mousePressed + mouseReleased

Type command:

- Optional selector (if omitted, types to focused element)
- `--key` flag sends key after text
- `--clear` flag clears content first (Ctrl+A + Backspace)

Scroll behavior:

- Instant only (no smooth animation)
- `scrollIntoView({block: "center", behavior: "instant"})`

## CDP Methods

| Command | CDP Methods |
|---------|-------------|
| navigate | `Page.navigate`, `Page.frameNavigated` (event) |
| reload | `Page.reload`, `Page.frameNavigated` (event) |
| back/forward | `Page.getNavigationHistory`, `Page.navigateToHistoryEntry`, `Page.frameNavigated` (event) |
| click | `DOM.getDocument`, `DOM.querySelector`, `DOM.getBoxModel`, `Input.dispatchMouseEvent` |
| type | `DOM.querySelector`, `DOM.focus`, `Input.insertText`, `Input.dispatchKeyEvent` |
| focus | `DOM.getDocument`, `DOM.querySelector`, `DOM.focus` |
| key | `Input.dispatchKeyEvent` |
| select | `Runtime.evaluate` |
| scroll | `Runtime.evaluate` |
| ready | `Page.loadEventFired` (event) |

## Dependencies

- P-007 (Observation Commands) - shares DOM query patterns
- Needed to fix bug in P-007

## Testing Strategy

Integration tests with real browser and test HTML page containing:

- Form inputs for type/focus testing
- Buttons for click testing
- Select dropdowns for select testing
- Links for navigation testing
- Long content for scroll testing
- Multiple pages for back/forward testing

## Notes

v1 limitations (documented):

- Elements must be in main frame (no iframe support)
- Elements must be visible for click
- Select only works with native `<select>` elements

Reference rod/chromedp implementations for edge cases.
