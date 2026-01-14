# p-045: Testing key Command

- Status: In Progress
- Started: 2025-12-31

## Overview

Test the webctl key command which sends a keyboard key to the focused element. Supports special keys (Enter, Tab, Escape, etc.) and modifier flags (ctrl, alt, shift, meta). Useful for keyboard shortcuts, navigation, and testing keyboard interactions.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-key.sh
```

## Code References

- internal/cli/key.go

## Command Signature

```
webctl key <key> [--ctrl] [--alt] [--shift] [--meta]
```

Arguments:
- key: The key to send (special keys or single characters)

Flags:
- --ctrl: Hold Ctrl modifier (Linux/Windows)
- --alt: Hold Alt/Option modifier
- --shift: Hold Shift modifier
- --meta: Hold Meta/Cmd modifier (macOS)

Supported special keys:
- Navigation: Enter, Tab, Escape, Space
- Editing: Backspace, Delete
- Arrows: ArrowUp, ArrowDown, ArrowLeft, ArrowRight
- Page: Home, End, PageUp, PageDown

## Test Checklist

Basic keys (no modifiers):
- [ ] key Enter (submit form / confirm)
- [ ] key Tab (move to next field)
- [ ] key Escape (close modal / cancel)
- [ ] key Space (toggle checkbox / click button)
- [ ] key Backspace (delete character before cursor)
- [ ] key Delete (delete character after cursor)
- [ ] Verify each key action executes

Arrow keys:
- [ ] key ArrowDown (move down in list)
- [ ] key ArrowUp (move up in list)
- [ ] key ArrowLeft (move left / cursor back)
- [ ] key ArrowRight (move right / cursor forward)
- [ ] Verify navigation with arrow keys

Page navigation keys:
- [ ] key Home (go to start of line/document)
- [ ] key End (go to end of line/document)
- [ ] key PageDown (scroll down one page)
- [ ] key PageUp (scroll up one page)
- [ ] Verify page navigation

Single character keys:
- [ ] key a (type 'a')
- [ ] key z (type 'z')
- [ ] key A (type 'A')
- [ ] key 5 (type '5')
- [ ] Verify single characters typed

Text editing shortcuts (Linux):
- [ ] key a --ctrl (select all)
- [ ] key c --ctrl (copy)
- [ ] key v --ctrl (paste)
- [ ] key x --ctrl (cut)
- [ ] key z --ctrl (undo)
- [ ] key z --ctrl --shift (redo)
- [ ] Verify Linux text editing shortcuts

Text editing shortcuts (macOS):
- [ ] key a --meta (select all)
- [ ] key c --meta (copy)
- [ ] key v --meta (paste)
- [ ] key x --meta (cut)
- [ ] key z --meta (undo)
- [ ] key z --meta --shift (redo)
- [ ] Verify macOS text editing shortcuts

Browser shortcuts (Linux):
- [ ] key l --ctrl (focus address bar)
- [ ] key f --ctrl (find in page)
- [ ] key t --ctrl (new tab)
- [ ] key w --ctrl (close tab)
- [ ] Verify browser shortcuts (may not work in headless)

Browser shortcuts (macOS):
- [ ] key l --meta (focus address bar)
- [ ] key f --meta (find in page)
- [ ] key t --meta (new tab)
- [ ] key w --meta (close tab)
- [ ] Verify browser shortcuts (may not work in headless)

Shift modifier:
- [ ] key ArrowDown --shift (extend selection down)
- [ ] key ArrowUp --shift (extend selection up)
- [ ] key Home --shift (select to start of line)
- [ ] key End --shift (select to end of line)
- [ ] Verify shift selection

Alt modifier:
- [ ] key a --alt (alt+a)
- [ ] key Tab --alt (switch windows/apps)
- [ ] Verify alt combinations

Multiple modifiers:
- [ ] key a --ctrl --shift (ctrl+shift+a)
- [ ] key z --meta --shift (cmd+shift+z, redo on macOS)
- [ ] key Delete --ctrl (delete word)
- [ ] Verify multiple modifier combinations

Form navigation:
- [ ] Focus first field, key Tab (move to next)
- [ ] key Tab multiple times (navigate through form)
- [ ] key Tab --shift (move to previous field)
- [ ] key Enter on submit button (submit form)

Dropdown/list navigation:
- [ ] Focus dropdown/list
- [ ] key ArrowDown (move to next item)
- [ ] key ArrowUp (move to previous item)
- [ ] key Enter (select current item)
- [ ] key Escape (close dropdown)

Modal interactions:
- [ ] Open modal, key Escape (close modal)
- [ ] key Tab through modal elements
- [ ] Verify modal keyboard interactions

Textarea editing:
- [ ] Focus textarea
- [ ] key Home (start of line)
- [ ] key End (end of line)
- [ ] key Backspace (delete character)
- [ ] key Enter (new line)

Error cases:
- [ ] key with invalid key name (error)
- [ ] key with empty key string (error)
- [ ] key without focused element (may succeed or fail)
- [ ] key with unsupported special key (error)
- [ ] Daemon not running (error message)

Key event verification:
- [ ] Verify keydown event fires
- [ ] Verify keyup event fires
- [ ] Verify keypress event fires (deprecated)
- [ ] Verify event handlers execute

Output formats:
- [ ] Default output: {"ok": true}
- [ ] --json output format
- [ ] --no-color output
- [ ] --debug verbose output
- [ ] Error output format

CLI vs REPL:
- [ ] CLI: webctl key Enter
- [ ] CLI: webctl key Tab
- [ ] CLI: webctl key a --ctrl
- [ ] CLI: webctl key z --meta --shift
- [ ] CLI: webctl key ArrowDown
- [ ] REPL: key Enter
- [ ] REPL: key Tab
- [ ] REPL: key a --ctrl
- [ ] REPL: key ArrowDown

## Notes

- Sends keyboard key to focused element
- Supports special keys: Enter, Tab, Escape, Space, Backspace, Delete, Arrows, Home, End, PageUp, PageDown
- Single character keys (a-z, A-Z, 0-9, punctuation) supported
- Modifier flags: --ctrl (Linux), --meta (macOS), --alt, --shift
- Multiple modifiers can be combined
- Useful for keyboard shortcuts, navigation, accessibility testing
- Browser shortcuts may not work in headless mode
- Clipboard operations (copy/paste) require browser permissions
- Different OS conventions: Ctrl on Linux/Windows, Cmd (Meta) on macOS
- Can be used after focus or type commands

## Issues Discovered

(Issues will be documented here during testing)
