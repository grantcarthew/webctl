# DR-013: Navigation & Interaction Commands

- Date: 2025-12-19
- Status: Accepted
- Category: CLI

## Problem

AI agents automating web applications need to navigate between pages and interact with page elements. While observation commands (console, network, screenshot, html) allow agents to inspect page state, agents cannot currently:

- Navigate to URLs or through browser history
- Click buttons and links
- Type text into input fields
- Select dropdown options
- Scroll to elements or positions
- Wait for pages to fully load

Requirements:

- Navigate to URLs with redirect handling
- Navigate browser history (back/forward)
- Reload pages with cache control
- Click elements by CSS selector
- Type text into focused or selected elements
- Send keyboard keys with modifiers
- Focus elements programmatically
- Select dropdown options
- Scroll to elements or positions
- Wait for page load completion
- All commands must update REPL prompt with current page title

## Decision

Implement 11 commands for navigation and interaction:

Navigation commands:

| Command | Syntax | Description |
|---------|--------|-------------|
| navigate | `navigate <url>` | Navigate to URL |
| reload | `reload [--ignore-cache]` | Reload current page |
| back | `back` | Navigate to previous history entry |
| forward | `forward` | Navigate to next history entry |

Interaction commands:

| Command | Syntax | Description |
|---------|--------|-------------|
| click | `click <selector>` | Click element by CSS selector |
| type | `type [selector] <text> [--key <key>] [--clear]` | Type text into element |
| focus | `focus <selector>` | Focus element by CSS selector |
| key | `key <key> [--ctrl] [--alt] [--shift] [--meta]` | Send keyboard key |
| select | `select <selector> <value>` | Select dropdown option |
| scroll | `scroll <selector>\|--to x,y\|--by x,y` | Scroll element into view or to position |

Utility commands:

| Command | Syntax | Description |
|---------|--------|-------------|
| ready | `ready [--timeout 30s]` | Wait for page load completion |

All navigation commands (navigate, reload, back, forward) wait for `Page.frameNavigated` event before returning, ensuring the REPL prompt displays the correct page title.

## Why

Wait for frameNavigated on navigation:

The REPL prompt displays the current page title. Without waiting for `frameNavigated`, a race condition occurs:

1. Navigate command returns immediately
2. REPL displays prompt with old title
3. `frameNavigated` event arrives after prompt is shown
4. Title only updates on next command

Waiting for `frameNavigated` (which fires when browser commits to navigation, before resources load) ensures the title is correct with negligible delay (~10-100ms).

CDP mouse events for click:

Two approaches exist for clicking elements:

1. CDP mouse events: `DOM.getBoxModel` + `Input.dispatchMouseEvent`
2. JavaScript: `element.click()`

CDP mouse events are preferred because:

- True mouse simulation triggers full event chain (mouseenter, mouseover, mousedown, mouseup, click)
- Matches rod/puppeteer/playwright implementations
- More reliable for elements with complex event handlers
- Simulates real user interaction

Composable type/focus/key commands:

Breaking text input into three commands provides flexibility:

- `focus` - focus any element
- `type` - insert text (optionally focus first)
- `key` - send any key with modifiers

This enables composable workflows:

```bash
webctl focus "#input"
webctl type "hello"
webctl key Enter
```

Or convenient one-liners:

```bash
webctl type "#input" "hello" --key Enter
```

The `--clear` flag on type uses select-all + delete internally for clearing existing content.

Instant scroll:

For automation, scroll must complete before command returns. Smooth scrolling introduces timing uncertainty ("when is it done?"). Instant scroll (`behavior: "instant"`) ensures deterministic behavior.

Separate ready command:

The `ready` command waits for `loadEventFired`, indicating the page's `load` event has fired. This is separate from navigation commands because:

- Navigation commands wait for `frameNavigated` (fast, title available)
- `ready` waits for `loadEventFired` (slower, page fully loaded)
- Users compose as needed: `navigate url && ready && html`

This avoids blocking all navigations on slow-loading pages while providing explicit load waiting when needed.

## Trade-offs

Accept:

- Navigation commands block briefly for frameNavigated (~10-100ms)
- Click requires visible elements in main frame (v1 limitation)
- Type requires element to be focusable
- Select only works with native `<select>` elements
- Scroll is instant only (no smooth option)
- Ready may timeout on very slow pages

Gain:

- REPL prompt always shows correct title after navigation
- Reliable click via true mouse simulation
- Composable input commands (focus/type/key)
- Convenient one-liner with type --key flag
- Explicit page load waiting with ready command
- Consistent error handling across all commands

## Alternatives

Fire-and-forget navigation:

Return immediately after sending `Page.navigate`, don't wait for any event.

- Pro: Fastest possible return
- Pro: Simpler implementation
- Con: REPL shows stale title until next command
- Con: Poor user experience
- Rejected: Correct REPL prompt worth the minimal delay

Wait for loadEventFired on all navigation:

Wait for full page load on every navigation.

- Pro: Page always ready after navigation
- Pro: No need for separate ready command
- Con: Much slower (100ms → 1-5s on typical pages)
- Con: Blocks on slow pages unnecessarily
- Con: Heavy pages may never complete loading
- Rejected: frameNavigated is sufficient for title, ready command handles load waiting

JavaScript click:

Use `Runtime.evaluate` with `element.click()`.

- Pro: Simpler, single CDP call
- Pro: Works even if element has no dimensions
- Con: Bypasses native event flow
- Con: May not trigger all event handlers
- Con: No coordinate simulation
- Rejected: CDP mouse events more reliable for automation

Combined type command with special key syntax:

Use inline syntax like `webctl type "#input" "text{Enter}"`.

- Pro: Single command for text + key
- Pro: Familiar from testing frameworks
- Con: More complex parsing
- Con: Escaping issues with literal braces
- Con: Less composable
- Rejected: Separate key command simpler and more flexible

## Command Specifications

### navigate

Navigate to URL and wait for page to commit.

```bash
webctl navigate <url>
```

CDP sequence:

1. `Page.navigate` with URL
2. Wait for `Page.frameNavigated` event
3. Return URL and title from frame info

Success response:

```json
{"ok": true, "url": "https://example.com/", "title": "Example Domain"}
```

Error response (navigation failed):

```json
{"ok": false, "error": "net::ERR_NAME_NOT_RESOLVED"}
```

The errorText from `Page.navigate` response indicates navigation failure.

### reload

Reload current page.

```bash
webctl reload [--ignore-cache]
```

Flags:

| Flag | Description |
|------|-------------|
| --ignore-cache | Hard reload, bypass browser cache |

CDP: `Page.reload` with `ignoreCache` parameter, then wait for `Page.frameNavigated`.

### back / forward

Navigate browser history.

```bash
webctl back
webctl forward
```

CDP sequence:

1. `Page.getNavigationHistory` - get entries and currentIndex
2. Calculate target index (current ± 1)
3. Validate target exists (error if at beginning/end)
4. `Page.navigateToHistoryEntry` with target entry ID
5. Wait for `Page.frameNavigated`

Error response (no history):

```json
{"ok": false, "error": "no previous page in history"}
```

### click

Click element by CSS selector.

```bash
webctl click <selector>
```

CDP sequence:

1. `DOM.getDocument` - get document root
2. `DOM.querySelector` - find element by selector
3. `DOM.getBoxModel` - get element coordinates
4. Calculate center point: `(left + right) / 2, (top + bottom) / 2`
5. `Input.dispatchMouseEvent` type=mousePressed, button=left, clickCount=1
6. `Input.dispatchMouseEvent` type=mouseReleased, button=left, clickCount=1

Error response (element not found):

```json
{"ok": false, "error": "element not found: .missing-button"}
```

v1 limitations (documented):

- Element must be in main frame (no iframe support)
- Element must be visible (not scrolled out of view)

### type

Type text into element.

```bash
webctl type [selector] <text> [--key <key>] [--clear]
```

Arguments:

- selector (optional): CSS selector to focus before typing
- text: Text to insert

Flags:

| Flag | Description |
|------|-------------|
| --key | Send key after text (e.g., Enter, Tab) |
| --clear | Clear existing content before typing |

CDP sequence:

1. If selector provided: focus element (see focus command)
2. If --clear: send Ctrl+A then Backspace
3. `Input.insertText` with text
4. If --key: send key event (see key command)

Argument parsing:

- One argument = text only (type into currently focused element)
- Two arguments = selector + text

### focus

Focus element by CSS selector.

```bash
webctl focus <selector>
```

CDP sequence:

1. `DOM.getDocument` - get document root
2. `DOM.querySelector` - find element
3. `DOM.focus` - focus the element

### key

Send keyboard key to focused element.

```bash
webctl key <key> [--ctrl] [--alt] [--shift] [--meta]
```

Arguments:

- key: Key name (Enter, Tab, Escape, Backspace, ArrowUp, ArrowDown, etc.)

Flags:

| Flag | Description |
|------|-------------|
| --Ctrl | Hold Ctrl modifier |
| --alt | Hold Alt modifier |
| --Shift | Hold Shift modifier |
| --meta | Hold Meta/Command modifier |

CDP: `Input.dispatchKeyEvent` with type=keyDown then keyUp.

Modifier bitmap: Alt=1, Ctrl=2, Meta=4, Shift=8

Common keys and their codes:

| Key | code | key |
|-----|------|-----|
| Enter | Enter | Enter |
| Tab | Tab | Tab |
| Escape | Escape | Escape |
| Backspace | Backspace | Backspace |
| ArrowUp | ArrowUp | ArrowUp |
| ArrowDown | ArrowDown | ArrowDown |
| ArrowLeft | ArrowLeft | ArrowLeft |
| ArrowRight | ArrowRight | ArrowRight |

### select

Select dropdown option.

```bash
webctl select <selector> <value>
```

Implementation via JavaScript (simplest for native select elements):

```javascript
const el = document.querySelector(selector);
el.value = value;
el.dispatchEvent(new Event('change', {bubbles: true}));
```

CDP: `Runtime.evaluate` with the above script.

Limitation: Only works for native `<select>` elements. Custom dropdowns need click-based interaction.

### scroll

Scroll element into view or to position.

```bash
webctl scroll <selector>           # Scroll element into view
webctl scroll --to x,y             # Scroll to absolute position
webctl scroll --by x,y             # Scroll by offset
```

Flags:

| Flag | Description |
|------|-------------|
| --to | Scroll to absolute position (x,y in pixels) |
| --by | Scroll by offset (x,y in pixels) |

Implementation via JavaScript:

Element into view:

```javascript
document.querySelector(selector).scrollIntoView({block: "center", behavior: "instant"});
```

Scroll to position:

```javascript
window.scrollTo({left: x, top: y, behavior: "instant"});
```

Scroll by offset:

```javascript
window.scrollBy({left: x, top: y, behavior: "instant"});
```

CDP: `Runtime.evaluate` with appropriate script.

### ready

Wait for page load completion.

```bash
webctl ready [--timeout 30s]
```

Flags:

| Flag | Default | Description |
|------|---------|-------------|
| --timeout | 30s | Maximum wait time |

Implementation sequence:

1. Check `document.readyState` via `Runtime.evaluate`
2. If already "complete", return immediately
3. Otherwise, wait for `Page.loadEventFired` event

This approach handles the case where the page is already loaded - calling `ready` multiple times will succeed immediately rather than timing out.

Success response:

```json
{"ok": true}
```

Timeout response:

```json
{"ok": false, "error": "timeout waiting for page load"}
```

## Error Handling

Common error responses:

Daemon not running:

```json
{"ok": false, "error": "daemon not running. Start with: webctl start"}
```

No active session:

```json
{"ok": false, "error": "no active session - use 'webctl target <id>' to select"}
```

Element not found:

```json
{"ok": false, "error": "element not found: .missing-selector"}
```

Navigation failed:

```json
{"ok": false, "error": "net::ERR_NAME_NOT_RESOLVED"}
```

No history:

```json
{"ok": false, "error": "no previous page in history"}
```

Timeout:

```json
{"ok": false, "error": "timeout waiting for page load"}
```

## Implementation Notes

CLI structure:

Each command follows existing patterns:

- Cobra command in `internal/cli/`
- IPC request to daemon
- Daemon handler in `handleRequest` switch
- CDP calls via `d.cdp.SendToSession()`

Flag resets:

Add new command flags to `resetCommandFlags()` in root.go for REPL support.

Waiting for events:

Navigate, reload, back, forward, and ready commands need to wait for CDP events. Implementation options:

1. Subscribe to event before sending command, wait with timeout
2. Use a channel to signal event receipt
3. Daemon maintains pending navigation state

Coordinate calculation for click:

The `DOM.getBoxModel` returns content, padding, border, and margin quads. Use the content quad to calculate center point. Coordinates are relative to viewport.

## Testing Strategy

Integration tests:

- Navigate to URL, verify title in response
- Navigate to invalid URL, verify error
- Reload page, verify frameNavigated received
- Test back/forward with history
- Test back at history start (error case)
- Click button, verify event triggered
- Type into input, verify value changed
- Test type with --key Enter
- Test type with --clear
- Focus element, verify document.activeElement
- Key command with modifiers
- Select dropdown, verify value
- Scroll to element, verify visibility
- Ready after navigation, verify load complete
- Ready timeout on slow page

Test page:

Create test HTML page with:

- Form inputs for type testing
- Buttons for click testing
- Select dropdowns for select testing
- Links for navigation testing
- Long content for scroll testing

## Future Enhancements

Deferred from initial implementation:

Hover command:

```bash
webctl hover <selector>
```

Mouse hover without click. Useful for tooltips and menus.

Double-click:

```bash
webctl click <selector> --double
```

Double-click for specific interactions.

Right-click:

```bash
webctl click <selector> --right
```

Context menu interactions.

Drag and drop:

```bash
webctl drag <from-selector> <to-selector>
```

Complex drag interactions.

File upload:

```bash
webctl upload <selector> <file-path>
```

File input handling.

Wait for selector:

```bash
webctl wait-for <selector> [--timeout 30s]
```

Wait for element to appear (P-009 scope).

Wait for network idle:

```bash
webctl wait-for --network-idle
```

Wait for network activity to settle (P-009 scope).

## Updates

- 2025-12-19: Initial version
- 2025-12-19: Fixed `ready` command to check document.readyState before waiting for loadEventFired (prevents timeout when page already loaded)
