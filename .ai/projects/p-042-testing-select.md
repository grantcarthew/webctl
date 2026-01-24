# p-042: Testing select Command

- Status: Superseded
- Started: 2025-12-31
- Superseded: 2026-01-24
- Superseded By: p-062 CLI Interaction Tests (automated test framework)

## Overview

Test the webctl select command which selects an option in a native HTML select dropdown element. Only works with native select elements (not custom JavaScript dropdowns). Dispatches a change event after selection.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-select.sh
```

## Code References

- internal/cli/selectcmd.go

## Command Signature

```
webctl select <selector> <value>
```

Arguments:
- selector: CSS selector to identify the select element
- value: Value attribute of the option to select (not display text)

No flags for this command.

## Test Checklist

Basic selection:
- [ ] select "#country" "AU" (select by ID)
- [ ] select "select[name=language]" "en" (select by name attribute)
- [ ] select ".size-picker" "large" (select by class)
- [ ] select "form#checkout select" "express" (nested in form)
- [ ] select "[data-testid=region]" "asia" (select by test ID)
- [ ] Verify option selected
- [ ] Verify selected value updated

Select by value attribute:
- [ ] Select using value attribute (not display text)
- [ ] Verify value="AU" selects "Australia" option
- [ ] Verify value="US" selects "United States" option
- [ ] Verify value must match exactly

Different select scenarios:
- [ ] Country selector (select country)
- [ ] State/region selector (select state)
- [ ] Size selector (select product size)
- [ ] Shipping method selector (select shipping option)
- [ ] Payment method selector (select payment type)
- [ ] Language selector (select language)
- [ ] Timezone selector (select timezone)

Change event:
- [ ] Verify change event fires after selection
- [ ] Verify change event handlers execute
- [ ] Verify form validation triggered by change event
- [ ] Verify dependent fields updated by change event

Multi-select forms:
- [ ] Select shipping method in one select
- [ ] Select payment method in another select
- [ ] Verify both selections work correctly
- [ ] Verify selections independent

Form workflows:
- [ ] Type name, type email, select country, select state, click submit
- [ ] Verify complete form submission with select elements
- [ ] Multiple selects in sequence

Select with default option:
- [ ] Select from dropdown with "Choose..." placeholder
- [ ] Select from dropdown with pre-selected value
- [ ] Verify selection changes from default

Verify selection:
- [ ] Use eval to check selected value after selection
- [ ] Use eval to verify selectedIndex changed
- [ ] Use css to check selected option styling

Error cases:
- [ ] Select non-existent selector (error: element not found)
- [ ] Select non-select element (error: element is not a select)
- [ ] Select with empty selector
- [ ] Select with invalid CSS selector
- [ ] Select with non-existent value (may succeed with no change or error)
- [ ] Select with empty value string
- [ ] Daemon not running (error message)

Element type validation:
- [ ] Select on div element (error: not a select)
- [ ] Select on input element (error: not a select)
- [ ] Select on button element (error: not a select)
- [ ] Select on textarea element (error: not a select)
- [ ] Select on native select element (success)

Custom dropdowns (should fail):
- [ ] Attempt select on React Select component (error)
- [ ] Attempt select on Material UI dropdown (error)
- [ ] Attempt select on custom div-based dropdown (error)
- [ ] Verify error message explains native select requirement

Output formats:
- [ ] Default output: {"ok": true}
- [ ] --json output format
- [ ] --no-color output
- [ ] --debug verbose output
- [ ] Error output format

CLI vs REPL:
- [ ] CLI: webctl select "#country" "AU"
- [ ] CLI: webctl select "select[name=shipping]" "express"
- [ ] CLI: webctl select ".size-picker" "large"
- [ ] REPL: select "#country" "AU"
- [ ] REPL: select "select[name=shipping]" "express"
- [ ] REPL: select ".size-picker" "large"

## Notes

- Only works with native HTML select elements
- Value must match option's value attribute, not display text
- Dispatches change event after selection
- Change event triggers form validation and event handlers
- For custom JavaScript dropdowns (React Select, Material UI, etc.), use click and type commands instead
- Cannot be used with multi-select elements (multiple attribute)
- Selecting non-existent value may succeed with no change depending on browser
- Useful for form automation with native dropdowns

## Issues Discovered

(Issues will be documented here during testing)
