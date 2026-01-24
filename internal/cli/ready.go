package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var readyCmd = &cobra.Command{
	Use:   "ready [selector]",
	Short: "Wait for page or application to be ready",
	Long: `Waits for the page or application to be ready before continuing.

Supports multiple synchronization modes for different use cases:
- Page load (default): waits for browser load event
- Element presence: waits for CSS selector to match
- Network idle: waits for all network requests to complete
- JS condition: waits for custom JavaScript expression to be true

Only works with a single mode at a time. For complex conditions,
chain multiple ready commands or use custom JavaScript.

Page load mode (default):
  Checks document.readyState first - if already "complete", returns
  immediately. Otherwise, waits for the browser's load event to fire.

  This is useful after navigation to ensure all resources (images,
  scripts, stylesheets) have fully loaded.

Selector mode:
  Waits for an element matching the CSS selector to appear in the DOM.
  Only checks presence, not visibility or interactivity (use eval for that).

  Useful for dynamic content loading, SPAs, or lazy-loaded components.

Network idle mode:
  Waits for all pending network requests to complete and the network
  to be quiet for 500ms. Useful after form submissions, AJAX requests,
  or API calls.

Eval mode:
  Waits for a custom JavaScript expression to evaluate to a truthy value.
  Most flexible option for application-specific ready states.

Timeout:
  --timeout duration    Maximum time to wait (default 60s)
                        Accepts Go duration format: 10s, 1m, 500ms

Examples:
  # Page load mode - wait for full page load
  ready
  ready --timeout 10s

  # Selector mode - wait for element to appear
  ready ".content-loaded"              # Wait for element with class
  ready "#dashboard"                   # Wait for element with ID
  ready "[data-loaded=true]"           # Wait for attribute
  ready "button.submit:enabled"        # Wait for enabled button

  # Network idle mode - wait for requests to complete
  ready --network-idle                 # Default 60s timeout
  ready --network-idle --timeout 120s  # Longer timeout for slow APIs

  # Eval mode - wait for custom condition
  ready --eval "document.readyState === 'complete'"
  ready --eval "window.appReady === true"
  ready --eval "document.querySelector('.error') === null"

Common patterns:
  # Navigate and wait for page load
  navigate example.com
  ready

  # SPA navigation - wait for route content
  click ".nav-dashboard"
  ready "#dashboard-content"

  # Form submission - wait for success message
  click "#submit"
  ready ".success-message"

  # API call - wait for network idle
  click "#load-data"
  ready --network-idle

  # Complex initialization - wait for app state
  navigate app.example.com
  ready --eval "window.app && window.app.initialized"

  # Dynamic content loading
  scroll "#load-more"
  ready --network-idle
  ready ".new-items"

  # Chaining multiple conditions
  ready                               # Page load
  ready --network-idle                # Then network idle
  ready --eval "window.dataLoaded"    # Then custom state

When to use each mode:
  - Page load: Full page navigation, browser reload
  - Selector: Dynamic content, SPA routes, lazy loading
  - Network idle: Form submissions, AJAX calls, API requests
  - Eval: Custom app states, complex conditions, visibility checks

For SPAs with client-side routing (React Router, Vue Router, etc.),
the page load event may not fire. Use selector or eval modes instead.

Error cases:
  - "timeout waiting for: <condition>" - condition not met within timeout
  - "no active session" - no browser page is open`,
	Args: cobra.MaximumNArgs(1),
	RunE: runReady,
}

func init() {
	readyCmd.Flags().Duration("timeout", 60*time.Second, "Maximum time to wait")
	readyCmd.Flags().Bool("network-idle", false, "Wait for network to be idle (500ms of no activity)")
	readyCmd.Flags().String("eval", "", "JavaScript expression to evaluate")
	rootCmd.AddCommand(readyCmd)
}

func runReady(cmd *cobra.Command, args []string) error {
	t := startTimer("ready")
	defer t.log()

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Read flags from command
	timeout, _ := cmd.Flags().GetDuration("timeout")
	networkIdle, _ := cmd.Flags().GetBool("network-idle")
	evalExpr, _ := cmd.Flags().GetString("eval")

	// Get selector from args if provided
	var selector string
	if len(args) > 0 {
		selector = args[0]
	}

	debugParam("timeout=%v selector=%q networkIdle=%v eval=%q", timeout, selector, networkIdle, evalExpr)

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer func() { _ = exec.Close() }()

	params, err := json.Marshal(ipc.ReadyParams{
		Timeout:     int(timeout.Seconds()),
		Selector:    selector,
		NetworkIdle: networkIdle,
		Eval:        evalExpr,
	})
	if err != nil {
		return outputError(err.Error())
	}

	debugRequest("ready", fmt.Sprintf("timeout=%v selector=%q networkIdle=%v", timeout, selector, networkIdle))
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "ready",
		Params: params,
	})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
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
