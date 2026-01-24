# p-041: Testing type Command

- Status: Superseded
- Started: 2025-12-31
- Superseded: 2026-01-24
- Superseded By: p-062 CLI Interaction Tests (automated test framework)

## Overview

Test the webctl type command which types text into an element using CDP keyboard input simulation. Supports typing with or without selector, clearing content before typing, and sending a key after typing.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-type.sh
```

## Code References

- internal/cli/type.go

## Command Signature

```
webctl type [selector] <text> [--key key] [--clear]
```

Arguments:
- selector (optional): CSS selector to focus before typing
- text: Text to type into the element

Flags:
- --key <key>: Send a key after typing (e.g., Enter, Tab)
- --clear: Clear existing content before typing (OS-aware: Cmd+A on macOS, Ctrl+A on Linux)

## Test Checklist

Basic typing (with selector):
- [ ] type "#username" "john_doe" (type into element by ID)
- [ ] type ".search-input" "query" (type into element by class)
- [ ] type "input[name=email]" "a@b.com" (type by name attribute)
- [ ] type "[data-testid=search]" "test" (type by test ID)
- [ ] Verify text appears in target element
- [ ] Verify cursor position after typing

Basic typing (without selector):
- [ ] focus "#input" then type "hello world" (type into focused element)
- [ ] Verify typing into already-focused element works
- [ ] Type without focusing first (should error or fail)

Clear flag:
- [ ] type "#email" "new@email.com" --clear (replace existing content)
- [ ] type "#input" "new text" --clear (clear then type)
- [ ] Verify existing content removed before typing
- [ ] Verify clear uses OS-appropriate shortcut (Cmd+A macOS, Ctrl+A Linux)
- [ ] Clear on empty field (should work without error)

Key flag (submit actions):
- [ ] type "#search" "query" --key Enter (type and submit)
- [ ] type "#field" "value" --key Enter (type and submit form)
- [ ] Verify Enter key triggers form submission
- [ ] Verify form submits after typing

Key flag (navigation):
- [ ] type "#field1" "value" --key Tab (type and move to next field)
- [ ] type "#field2" "value" --key Tab (continue to next field)
- [ ] Verify Tab moves focus to next focusable element
- [ ] Verify typed text retained after Tab

Key flag (special keys):
- [ ] type "#input" "text" --key Escape (type then escape)
- [ ] type "#input" "text" --key Space (type then space)
- [ ] type "#input" "text" --key Backspace (type then backspace)
- [ ] type "#input" "text" --key Delete (type then delete)
- [ ] Verify each key action executes correctly

Combined flags:
- [ ] type "#search" "new query" --clear --key Enter (clear, type, submit)
- [ ] type "#input" "text" --clear --key Tab (clear, type, tab)
- [ ] Verify both flags work together correctly

Form workflows:
- [ ] Login form: type "#username" "user" --clear, type "#password" "pass", click submit
- [ ] Search form: type "#search" "query" --key Enter
- [ ] Multi-field form: type "#field1" "val1" --key Tab, type "val2" --key Tab, type "val3" --key Enter
- [ ] Verify complete form workflows

Different input types:
- [ ] type into text input
- [ ] type into password input
- [ ] type into email input
- [ ] type into search input
- [ ] type into textarea
- [ ] type into contenteditable div
- [ ] type into number input (numeric text)

Special characters:
- [ ] Type text with spaces
- [ ] Type text with punctuation
- [ ] Type text with special characters (!@#$%^&*)
- [ ] Type text with quotes
- [ ] Type text with newlines (in textarea)
- [ ] Type emojis
- [ ] Type unicode characters

Error cases:
- [ ] Type with non-existent selector (error: element not found)
- [ ] Type with empty selector string
- [ ] Type with invalid CSS selector
- [ ] Type into non-focusable element (error: element is not focusable)
- [ ] Type with invalid --key value
- [ ] Type without text argument (error: missing argument)
- [ ] Daemon not running (error message)

Output formats:
- [ ] Default output: {"ok": true}
- [ ] --json output format
- [ ] --no-color output
- [ ] --debug verbose output
- [ ] Error output format

CLI vs REPL:
- [ ] CLI: webctl type "#username" "john_doe"
- [ ] CLI: webctl type "#search" "query" --key Enter
- [ ] CLI: webctl type "#email" "new@email.com" --clear
- [ ] CLI: webctl type "#field" "value" --clear --key Tab
- [ ] REPL: type "#username" "john_doe"
- [ ] REPL: type "#search" "query" --key Enter
- [ ] REPL: type "#email" "new@email.com" --clear
- [ ] REPL: type "focused text" (no selector)

## Notes

- With one argument: types into currently focused element
- With two arguments: focuses element matching selector, then types
- --clear flag uses OS-aware shortcuts (Cmd+A on macOS, Ctrl+A on Linux)
- --key flag sends key after typing completes
- Uses CDP keyboard input simulation for realistic typing
- Special keys supported: Enter, Tab, Escape, Space, Backspace, Delete, etc.
- Can be used without selector if element already focused
- Useful for multi-field forms with Tab navigation
- Works with contenteditable elements as well as inputs

## Issues Discovered

(Issues will be documented here during testing)
