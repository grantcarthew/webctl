# P-008: Navigation & Interaction Commands

- Status: Proposed
- Started: -

## Overview

Implement commands for navigating the browser and interacting with page elements: navigate, reload, back, forward, click, type, select, scroll.

## Goals

1. Navigate to URLs and through history
2. Click elements by CSS selector
3. Type text into input fields
4. Select dropdown options
5. Scroll to elements or positions

## Scope

In Scope:

- `navigate` command
- `reload` command
- `back` command
- `forward` command
- `click` command
- `type` command
- `select` command
- `scroll` command

Out of Scope:

- Wait-for commands (P-009)
- Complex interactions (drag-drop, hover)
- File upload

## Success Criteria

- [ ] `webctl navigate https://example.com` loads page
- [ ] `webctl reload` refreshes page
- [ ] `webctl back` goes to previous page
- [ ] `webctl forward` goes to next page
- [ ] `webctl click ".button"` clicks element
- [ ] `webctl type "#input" "hello"` types text
- [ ] `webctl select "#dropdown" "option1"` selects option
- [ ] `webctl scroll ".element"` scrolls element into view

## Deliverables

- `cmd/webctl/navigate.go`
- `cmd/webctl/reload.go`
- `cmd/webctl/back.go`
- `cmd/webctl/forward.go`
- `cmd/webctl/click.go`
- `cmd/webctl/type.go`
- `cmd/webctl/select.go`
- `cmd/webctl/scroll.go`
- Daemon-side handlers

## Technical Design

### Navigate Command

```bash
webctl navigate https://example.com
```

Output:
```json
{"ok": true, "url": "https://example.com", "title": "Example"}
```

CDP: `Page.navigate`
- Wait for `Page.loadEventFired` before returning
- Return final URL (may differ due to redirects)

### Reload Command

```bash
webctl reload
webctl reload --ignore-cache
```

CDP: `Page.reload`
- `ignoreCache` parameter for hard reload

### Back/Forward Commands

```bash
webctl back
webctl forward
```

CDP:
1. `Page.getNavigationHistory` - get history entries and currentIndex
2. Calculate target index (current ± 1)
3. `Page.navigateToHistoryEntry` with target entry ID

Error if at beginning/end of history.

### Click Command

```bash
webctl click ".submit-button"
webctl click "#login"
```

CDP sequence:
1. `DOM.getDocument` - get document root
2. `DOM.querySelector` - find element by selector
3. `DOM.getBoxModel` - get element coordinates
4. Calculate center point of element
5. `Input.dispatchMouseEvent` type=mousePressed
6. `Input.dispatchMouseEvent` type=mouseReleased

Error if element not found.

### Type Command

```bash
webctl type "#username" "admin"
webctl type ".search-box" "search query"
```

CDP sequence:
1. Find element (same as click)
2. `DOM.focus` - focus the element
3. `Input.insertText` - insert text directly

For special keys (Enter, Tab, etc.):
```bash
webctl type "#input" --key Enter
```
Uses `Input.dispatchKeyEvent` instead.

### Select Command

```bash
webctl select "#country" "Australia"
```

Implementation via JS (simplest):
```javascript
const el = document.querySelector("#country");
el.value = "Australia";
el.dispatchEvent(new Event('change', {bubbles: true}));
```

CDP: `Runtime.evaluate`

Note: Only works for native `<select>` elements. Custom dropdowns need click-based interaction.

### Scroll Command

```bash
webctl scroll ".footer"           # Scroll element into view
webctl scroll --to "0,1000"       # Scroll to position
webctl scroll --by "0,500"        # Scroll by offset
```

Implementation via JS:
```javascript
document.querySelector(".footer").scrollIntoView({behavior: "smooth"});
// or
window.scrollTo(0, 1000);
// or
window.scrollBy(0, 500);
```

CDP: `Runtime.evaluate`

## CDP Methods Used

| Command | CDP Methods |
|---------|-------------|
| navigate | `Page.navigate`, `Page.loadEventFired` (event) |
| reload | `Page.reload` |
| back/forward | `Page.getNavigationHistory`, `Page.navigateToHistoryEntry` |
| click | `DOM.getDocument`, `DOM.querySelector`, `DOM.getBoxModel`, `Input.dispatchMouseEvent` |
| type | `DOM.getDocument`, `DOM.querySelector`, `DOM.focus`, `Input.insertText` |
| select | `Runtime.evaluate` |
| scroll | `Runtime.evaluate` |

## Error Handling

Common errors:
- Element not found: `{"ok": false, "error": "element not found: .missing"}`
- No history: `{"ok": false, "error": "no previous page in history"}`
- Navigation failed: `{"ok": false, "error": "navigation failed: net::ERR_NAME_NOT_RESOLVED"}`

## Dependencies

- P-007 (Observation Commands) - shares DOM query patterns

## Testing Strategy

1. **Integration tests** - Real browser, test page with forms/buttons

## Notes

Click is the most complex due to coordinate calculation. Reference rod/chromedp implementations for edge cases (elements in iframes, scrolled out of view, etc.).

For v1, assume elements are in main frame and visible. Document limitations.
