# P-040: Testing click Command

- Status: In Progress
- Started: 2025-12-31

## Overview

Test the webctl click command which clicks an element matching a CSS selector. Uses CDP mouse events for true click simulation, triggering the full event chain (mouseenter → mouseover → mousedown → mouseup → click). Elements are automatically scrolled into view before clicking.

## Code References

- internal/cli/click.go

## Command Signature

```
webctl click <selector>
```

Arguments:
- selector: CSS selector to identify the element to click

No flags for this command.

## Test Checklist

Basic element clicking:
- [ ] click "#submit" (button by ID)
- [ ] click ".btn-primary" (button by class)
- [ ] click "button[type=submit]" (button by attribute)
- [ ] click "a[href='/page']" (link by attribute)
- [ ] click "[data-testid=login-btn]" (by test ID)
- [ ] Verify element receives click
- [ ] Verify click action executes (form submit, navigation, etc.)

Different element types:
- [ ] Click button element
- [ ] Click link element
- [ ] Click div with click handler
- [ ] Click checkbox input
- [ ] Click radio input
- [ ] Click image element
- [ ] Click span element

Complex selectors:
- [ ] click "form#login button" (nested selector)
- [ ] click "nav a:first-child" (pseudo-selector)
- [ ] click ".container > .button" (direct child)
- [ ] click "div.class1.class2" (multiple classes)
- [ ] click "button:not(.disabled)" (negation)

Auto-scrolling behavior:
- [ ] Click element below viewport (verify scroll)
- [ ] Click element above viewport (verify scroll)
- [ ] Click element already in view (no scroll needed)
- [ ] Verify element centered in viewport before click

Form interactions:
- [ ] Type into email field, type into password field, click submit button
- [ ] Click submit button on form
- [ ] Click checkbox to toggle
- [ ] Click radio button to select
- [ ] Click cancel/reset button

Navigation clicks:
- [ ] Click link to navigate to new page
- [ ] Click hash link (same page navigation)
- [ ] Click external link
- [ ] Click link that opens in new tab (target=_blank)

Button interactions:
- [ ] Click normal button
- [ ] Click button that shows modal
- [ ] Click button that hides element
- [ ] Click toggle button
- [ ] Click disabled button (may fail or succeed depending on implementation)

Event verification:
- [ ] Verify mouseenter event fires
- [ ] Verify mouseover event fires
- [ ] Verify mousedown event fires
- [ ] Verify mouseup event fires
- [ ] Verify click event fires
- [ ] Verify event handlers execute

Overlapping elements:
- [ ] Click element covered by another element (warning expected)
- [ ] Verify warning message returned
- [ ] Verify click still proceeds despite warning

Error cases:
- [ ] Click non-existent selector (error: element not found)
- [ ] Click empty selector string
- [ ] Click invalid CSS selector
- [ ] Daemon not running (error message)

Output formats:
- [ ] Default output: {"ok": true}
- [ ] Output with warning: {"ok": true, "warning": "..."}
- [ ] --json output format
- [ ] --no-color output
- [ ] --debug verbose output
- [ ] Error output format

CLI vs REPL:
- [ ] CLI: webctl click "#submit"
- [ ] CLI: webctl click ".btn-primary"
- [ ] CLI: webctl click "a[href='/page']"
- [ ] REPL: click "#submit"
- [ ] REPL: click ".btn-primary"
- [ ] REPL: click "a[href='/page']"

## Notes

- Uses CDP mouse events for true click simulation
- Full event chain: mouseenter → mouseover → mousedown → mouseup → click
- Elements automatically scrolled into view (centered in viewport)
- Warning returned if element may be covered by another element
- Click still proceeds even if element is covered
- No iframe support yet (main frame only)
- For native select dropdowns, use select command instead
- Click triggers all event handlers just like real user interaction

## Issues Discovered

(Issues will be documented here during testing)
