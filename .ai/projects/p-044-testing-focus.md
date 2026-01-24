# p-044: Testing focus Command

- Status: Superseded
- Started: 2025-12-31
- Superseded: 2026-01-24
- Superseded By: p-062 CLI Interaction Tests (automated test framework)

## Overview

Test the webctl focus command which focuses an element matching a CSS selector. Simple command with no flags, used primarily for focusing input fields before typing or for testing focus-based interactions.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-focus.sh
```

## Code References

- internal/cli/focus.go

## Command Signature

```
webctl focus <selector>
```

Arguments:
- selector: CSS selector to identify the element to focus

No flags for this command.

## Test Checklist

Basic focusing:
- [ ] focus "#username" (focus input by ID)
- [ ] focus ".search-input" (focus input by class)
- [ ] focus "input[name=email]" (focus input by name)
- [ ] focus "[data-testid=search]" (focus by test ID)
- [ ] Verify element receives focus
- [ ] Verify focus styles applied (outline, border, etc.)

Different input types:
- [ ] Focus text input
- [ ] Focus password input
- [ ] Focus email input
- [ ] Focus search input
- [ ] Focus number input
- [ ] Focus tel input
- [ ] Focus url input
- [ ] Focus textarea
- [ ] Focus contenteditable div

Focusable elements:
- [ ] Focus button element
- [ ] Focus link (a) element
- [ ] Focus select element
- [ ] Focus checkbox input
- [ ] Focus radio input
- [ ] Focus element with tabindex="0"
- [ ] Focus element with tabindex="-1"

Complex selectors:
- [ ] focus "form#login input[type=text]" (nested selector)
- [ ] focus "div.container > input" (direct child)
- [ ] focus "input:first-of-type" (pseudo-selector)
- [ ] focus "input:not([disabled])" (negation)

Focus followed by typing:
- [ ] focus "#username" then type "john_doe"
- [ ] focus "#password" then type "secret123"
- [ ] focus "#search" then type "query" --key Enter
- [ ] Verify focus + type workflow

Focus for accessibility testing:
- [ ] Focus through form fields with Tab
- [ ] Verify focus order (tabindex)
- [ ] Verify focus visible (outline/ring)
- [ ] Verify skip links work when focused

Focus events:
- [ ] Verify focus event fires
- [ ] Verify focusin event fires
- [ ] Verify blur event fires on previously focused element
- [ ] Verify focus event handlers execute

Focus state verification:
- [ ] Use eval to check document.activeElement after focus
- [ ] Verify focused element matches selector
- [ ] Verify focus styles applied (CSS)

Error cases:
- [ ] Focus non-existent selector (error: element not found)
- [ ] Focus empty selector string
- [ ] Focus invalid CSS selector
- [ ] Focus non-focusable element (error: element is not focusable)
- [ ] Focus disabled input (error or no focus)
- [ ] Focus hidden element (error or no focus)
- [ ] Focus display:none element (error or no focus)
- [ ] Daemon not running (error message)

Non-focusable elements:
- [ ] Focus div without tabindex (should error)
- [ ] Focus span element (should error)
- [ ] Focus p element (should error)
- [ ] Focus img element (should error)
- [ ] Verify error message explains focusability

Modal and overlay interactions:
- [ ] Open modal, focus first input
- [ ] Focus element in overlay
- [ ] Verify focus trapped in modal (if applicable)

Output formats:
- [ ] Default output: {"ok": true}
- [ ] --json output format
- [ ] --no-color output
- [ ] --debug verbose output
- [ ] Error output format

CLI vs REPL:
- [ ] CLI: webctl focus "#username"
- [ ] CLI: webctl focus ".search-input"
- [ ] CLI: webctl focus "input[name=email]"
- [ ] REPL: focus "#username"
- [ ] REPL: focus ".search-input"
- [ ] REPL: focus "input[name=email]"

## Notes

- Simple command with no flags
- Focuses element matching CSS selector
- Element must be focusable (input, button, link, select, textarea, or element with tabindex)
- Non-focusable elements (div, span, p, etc.) will error unless they have tabindex
- Useful before typing without selector: focus then type
- Used for testing focus order and accessibility
- Focus event fires when element receives focus
- Blur event fires on previously focused element
- Can be verified with eval: document.activeElement

## Issues Discovered

(Issues will be documented here during testing)
