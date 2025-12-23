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
- All commands must update REPL prompt with current page URL

## Decision

Implement 11 commands for navigation and interaction:

Navigation commands:

| Command | Syntax | Description |
|---------|--------|-------------|
| navigate | `navigate <url> [--wait] [--timeout <ms>]` | Navigate to URL |
| reload | `reload [--wait] [--timeout <ms>]` | Reload current page (hard reload) |
| back | `back [--wait] [--timeout <ms>]` | Navigate to previous history entry |
| forward | `forward [--wait] [--timeout <ms>]` | Navigate to next history entry |

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

All navigation commands return immediately by default for fast feedback. Use `--wait` flag to wait for page load completion (`loadEventFired`). The REPL prompt displays the current page URL (not title), which is available immediately.

## Why

Immediate return for navigation commands:

Navigation commands return immediately for fast feedback and better automation performance:

1. No blocking on page load - commands complete in <100ms
2. Users compose wait behavior explicitly: `navigate url --wait` or `navigate url && ready`
3. REPL prompt shows URL (not title), which is known immediately
4. Title-based prompts caused blocking and Chrome issues

URL in REPL prompt:

The REPL prompt displays the current page URL (protocol and trailing slash stripped):
- `https://example.com/` → `webctl [example.com]>`
- `http://localhost:3000/api` → `webctl [localhost:3000/api]>`

URLs are available immediately when navigating, unlike titles which require waiting for page load. For automation, the URL is more useful than the title.

Auto-detect URL protocol:

The navigate command automatically adds protocol if missing:
- `example.com` → `https://example.com`
- `localhost:3000` → `http://localhost:3000`
- `127.0.0.1` → `http://127.0.0.1`

Defaults to https for security, http for local development domains.

Hard reload by default:

The reload command always performs a hard reload (ignores cache) because automation and testing scenarios almost always want fresh content, not cached responses.

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

Separate ready command and --wait flag:

The `ready` command and `--wait` flag both wait for `loadEventFired`, indicating the page's `load` event has fired. Two ways to wait:

1. Inline with navigation: `navigate url --wait`
2. Separate command: `navigate url && ready`

Navigation commands return immediately by default, avoiding blocking on slow pages. Users explicitly request waiting when needed for commands that require page content (html, screenshot, etc).

## Trade-offs

Accept:

- No protocol in URL requires auto-detection heuristic
- Reload always hard - no soft reload option (can add later if needed)
- URL in prompt less "friendly" than title (but more useful for automation)
- Users must explicitly wait when needed (--wait flag or ready command)
- Click requires visible elements in main frame (v1 limitation)
- Type requires element to be focusable
- Select only works with native `<select>` elements
- Scroll is instant only (no smooth option)

Gain:

- Fast navigation commands (<100ms return time)
- No blocking on slow page loads
- URL available immediately in REPL prompt
- Clear, composable wait behavior
- Users type less (example.com vs https://example.com/)
- Hard reload default matches automation use case
- Reliable click via true mouse simulation
- Composable input commands (focus/type/key)
- Convenient one-liner with type --key flag or --wait flag
- Consistent error handling across all commands

## Alternatives

Wait for frameNavigated on all navigation:

Wait for `frameNavigated` event before returning, which provides page title.

- Pro: Title available for REPL prompt
- Pro: Slightly more "complete" feeling (~10-100ms wait)
- Con: Blocks navigation commands even when title not needed
- Con: Caused Chrome internal blocking issues in testing
- Con: Title less useful than URL for automation
- Rejected: Immediate return with URL prompt is faster and more useful

Always wait for loadEventFired (no --wait flag):

Make all navigation commands wait for full page load by default.

- Pro: Page always ready after navigation
- Pro: Simpler for beginners - no need to understand wait
- Con: Much slower (100ms → 1-5s on typical pages)
- Con: Blocks on slow/heavy pages
- Con: Forces waiting even when not needed
- Rejected: Optional --wait flag gives users control

Title in REPL prompt:

Display page title instead of URL in REPL prompt.

- Pro: More "friendly" and human-readable
- Con: Requires waiting for page load to get title
- Con: Title may change or be empty
- Con: URL is more useful for automation (shows exact location)
- Rejected: URL is available immediately and more useful

Soft reload by default:

Make reload use cache by default, add --ignore-cache for hard reload.

- Pro: Matches browser behavior
- Pro: Faster reload
- Con: Automation/testing almost always wants fresh content
- Con: Users would constantly use --ignore-cache flag
- Rejected: Hard reload better matches automation use case

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
- 2025-12-23: Major revision based on P-009 design review:
  - Navigation commands now return immediately (no frameNavigated wait)
  - Added --wait and --timeout flags to all navigation commands for optional page load waiting
  - Changed reload to always hard reload (removed --ignore-cache flag)
  - Added URL protocol auto-detection to navigate command
  - Changed REPL prompt to show URL instead of title (URL available immediately)
