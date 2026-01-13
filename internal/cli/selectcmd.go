package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

// Named selectCmd_ to avoid collision with Go's select keyword
var selectCmd_ = &cobra.Command{
	Use:   "select <selector> <value>",
	Short: "Select a dropdown option",
	Long: `Selects an option in a native HTML <select> dropdown element.

The selector identifies the <select> element using CSS selector syntax.
The value must match the option's value attribute (not the display text).

Only works with native HTML <select> elements. For custom JavaScript dropdowns
(like React Select, Material UI, etc.), use click and type commands instead.

The command dispatches a 'change' event after selection, triggering any
form validation or event handlers attached to the element.

Selector examples:
  select "#country" "AU"                    # By ID
  select "select[name=language]" "en"       # By name attribute
  select ".size-picker" "large"             # By class
  select "form#checkout select" "express"   # Nested in form
  select "[data-testid=region]" "asia"      # By test ID

Given this HTML (country selector):
  <select id="country" name="country">
    <option value="">Choose country...</option>
    <option value="US">United States</option>
    <option value="AU">Australia</option>
    <option value="UK">United Kingdom</option>
    <option value="NZ">New Zealand</option>
  </select>

Use: select "#country" "AU"
Note: Use "AU" (the value attribute), not "Australia" (the display text)

Given this HTML (size selector):
  <select class="product-size" name="size">
    <option value="xs">Extra Small</option>
    <option value="s">Small</option>
    <option value="m">Medium</option>
    <option value="l">Large</option>
    <option value="xl">Extra Large</option>
  </select>

Use: select ".product-size" "m"

Given this HTML (multi-select form):
  <form id="order">
    <select name="shipping">
      <option value="standard">Standard (5-7 days)</option>
      <option value="express">Express (2-3 days)</option>
      <option value="overnight">Overnight</option>
    </select>
    <select name="payment">
      <option value="credit">Credit Card</option>
      <option value="paypal">PayPal</option>
      <option value="bank">Bank Transfer</option>
    </select>
  </form>

Use:
  select "form#order select[name=shipping]" "express"
  select "form#order select[name=payment]" "credit"

Common form automation pattern:
  type "#email" "user@example.com"
  type "#name" "John Smith"
  select "#country" "AU"
  select "#state" "NSW"
  click "#submit"

For custom dropdowns (React, Vue, Material UI):
  click ".custom-dropdown"           # Open dropdown
  click ".option[data-value=AU]"     # Click option
  # Or:
  click ".custom-dropdown"
  type "Australia"                   # Type to filter
  key Enter                          # Select highlighted

Error cases:
  - "element not found" - selector doesn't match any element
  - "element is not a select" - matched element is not a <select>`,
	Args: cobra.ExactArgs(2),
	RunE: runSelect,
}

func init() {
	rootCmd.AddCommand(selectCmd_)
}

func runSelect(cmd *cobra.Command, args []string) error {
	t := startTimer("select")
	defer t.log()

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	selector := args[0]
	value := args[1]
	debugParam("selector=%q value=%q", selector, value)

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	params, err := json.Marshal(ipc.SelectParams{
		Selector: selector,
		Value:    value,
	})
	if err != nil {
		return outputError(err.Error())
	}

	debugRequest("select", fmt.Sprintf("selector=%q value=%q", selector, value))
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "select",
		Params: params,
	})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		if isNoElementsError(resp.Error) {
			return outputNotice("No elements found")
		}
		return outputError(resp.Error)
	}

	// JSON mode: output JSON
	if JSONOutput {
		result := map[string]any{
			"ok": true,
		}
		return outputJSON(os.Stdout, result)
	}

	// Text mode: just output OK
	return outputSuccess(nil)
}
