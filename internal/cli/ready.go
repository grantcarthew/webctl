package cli

import (
	"encoding/json"
	"os"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "Wait for page to finish loading",
	Long: `Waits for the page to finish loading before continuing.

The command checks document.readyState first - if already "complete", returns
immediately. Otherwise, waits for the browser's load event to fire.

This is useful for synchronisation after navigation or page transitions,
ensuring all resources (images, scripts, stylesheets) have fully loaded.

Timeout:
  --timeout duration    Maximum time to wait (default 30s)
                        Accepts Go duration format: 10s, 1m, 500ms

Examples:
  # Basic usage - wait for page load
  ready

  # With custom timeout
  ready --timeout 10s           # Wait up to 10 seconds
  ready --timeout 1m            # Wait up to 1 minute
  ready --timeout 5s            # Quick timeout for fast pages

Common patterns:
  # Navigate and wait for load
  navigate example.com
  ready

  # Navigation with explicit wait (equivalent to navigate --wait)
  navigate example.com && ready

  # Form submission with page reload
  click "#submit"
  ready

  # Multiple page workflow
  navigate example.com/login
  ready
  type "#email" "user@example.com"
  type "#password" "secret"
  click "#login-button"
  ready                         # Wait for redirect after login

  # SPA navigation (may need eval instead for client-side routing)
  click ".nav-link"
  ready                         # Works if link causes full page load

When to use ready vs --wait flag:
  - ready: Explicit synchronisation point, composable in scripts
  - --wait: Inline with navigation command, single operation

For SPAs with client-side routing (React Router, Vue Router, etc.),
the page load event may not fire. Consider using eval to check
for specific elements or application state instead:
  eval "document.querySelector('.dashboard') !== null"

Error cases:
  - "timeout waiting for page load" - page didn't load within timeout
  - "no active session" - no browser page is open`,
	Args: cobra.NoArgs,
	RunE: runReady,
}

func init() {
	readyCmd.Flags().Duration("timeout", 30*time.Second, "Maximum time to wait for page load")
	rootCmd.AddCommand(readyCmd)
}

func runReady(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Read flags from command
	timeout, _ := cmd.Flags().GetDuration("timeout")

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	params, err := json.Marshal(ipc.ReadyParams{
		Timeout: int(timeout.Milliseconds()),
	})
	if err != nil {
		return outputError(err.Error())
	}

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "ready",
		Params: params,
	})
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	result := map[string]any{
		"ok": true,
	}
	return outputJSON(os.Stdout, result)
}
