package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var clickCmd = &cobra.Command{
	Use:   "click <selector>",
	Short: "Click an element",
	Long: `Clicks an element matching the CSS selector.

Uses CDP mouse events for true click simulation, triggering the full event chain:
mouseenter → mouseover → mousedown → mouseup → click. This matches how real
users interact with elements and ensures all event handlers fire correctly.

Before clicking, the element is automatically scrolled into view (centered in
the viewport). If another element covers the target, a warning is returned but
the click still proceeds.

Selector examples:
  click "#submit"                       # By ID
  click ".btn-primary"                  # By class
  click "button[type=submit]"           # By attribute
  click "form#login button"             # Nested selector
  click "[data-testid=login-btn]"       # By test ID (recommended)
  click "nav a:first-child"             # First link in nav

Given this HTML:
  <form id="login">
    <input type="email" id="email">
    <input type="password" id="password">
    <button type="submit" class="btn">Login</button>
  </form>

Use:
  click "#login .btn"                   # Click the login button
  click "button[type=submit]"           # Same button, different selector

Common patterns:
  # Form submission
  type "#email" "user@example.com"
  type "#password" "secret"
  click "#login button"

  # Navigation via link
  click "nav a[href='/dashboard']"
  ready                                 # Wait for new page

  # Toggle/checkbox
  click "#dark-mode-toggle"
  click "input[type=checkbox]#agree"

  # Modal interaction
  click "#open-modal"
  click ".modal .close-button"

  # Dropdown menu (custom, not <select>)
  click ".dropdown-trigger"
  click ".dropdown-menu .option:first-child"

Response:
  {"ok": true}
  {"ok": true, "warning": "element may be covered by another element"}

Error cases:
  - "element not found: .missing" - selector doesn't match any element
  - "daemon not running" - start daemon first with: webctl start

Limitations:
  - Element must be in main frame (no iframe support yet)
  - For native <select> dropdowns, use the select command instead`,
	Args: cobra.ExactArgs(1),
	RunE: runClick,
}

func init() {
	rootCmd.AddCommand(clickCmd)
}

func runClick(cmd *cobra.Command, args []string) error {
	t := startTimer("click")
	defer t.log()

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	selector := args[0]
	debugParam("selector=%q", selector)

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	params, err := json.Marshal(ipc.ClickParams{
		Selector: selector,
	})
	if err != nil {
		return outputError(err.Error())
	}

	debugRequest("click", fmt.Sprintf("selector=%q", selector))
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "click",
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

	// JSON mode: include any warnings from response data
	if JSONOutput {
		result := map[string]any{
			"ok": true,
		}

		if len(resp.Data) > 0 {
			var data map[string]any
			if err := json.Unmarshal(resp.Data, &data); err == nil {
				if warning, ok := data["warning"].(string); ok {
					result["warning"] = warning
				}
			}
		}

		return outputJSON(os.Stdout, result)
	}

	// Text mode: just output OK (warnings ignored in text mode for simplicity)
	return outputSuccess(nil)
}
