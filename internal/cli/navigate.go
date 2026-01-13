package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var navigateCmd = &cobra.Command{
	Use:   "navigate <url>",
	Short: "Navigate to URL",
	Long: `Navigates the active browser session to the specified URL.

Returns immediately by default for fast feedback (<100ms). The page continues
loading in the background. Use --wait to block until the page fully loads.

URL protocol auto-detection:
  - URLs without protocol get https:// added automatically
  - localhost, 127.0.0.1, and 0.0.0.0 get http:// (local development)
  - Explicit protocols (http://, https://, file://) are preserved

Flags:
  --wait              Wait for page load completion (load event fired)
  --timeout <seconds> Timeout in seconds when using --wait (default 60)

Examples:
  # Basic navigation (fast return, page loads in background)
  navigate example.com                    # https://example.com
  navigate www.google.com/search?q=test   # https://www.google.com/search?q=test

  # Local development (auto-detects http://)
  navigate localhost:3000                 # http://localhost:3000
  navigate 127.0.0.1:8080/api             # http://127.0.0.1:8080/api
  navigate 0.0.0.0:5000                   # http://0.0.0.0:5000

  # Explicit protocol
  navigate http://insecure-site.com       # Preserves http://
  navigate file:///tmp/test.html          # Local file

  # Wait for page load (blocks until load event)
  navigate example.com --wait
  navigate slow-site.com --wait --timeout 60

  # Common workflow patterns
  navigate example.com && ready           # Equivalent to --wait
  navigate example.com && screenshot      # Capture after navigation
  navigate example.com --wait && html     # Get HTML after full load

Response:
  {"ok": true, "url": "https://example.com/", "title": "Example Domain"}

Error cases:
  - "net::ERR_NAME_NOT_RESOLVED" - domain does not exist
  - "net::ERR_CONNECTION_REFUSED" - server not responding
  - "timeout waiting for page load" - page didn't load within timeout (--wait)
  - "daemon not running" - start daemon first with: webctl start`,
	Args: cobra.ExactArgs(1),
	RunE: runNavigate,
}

func init() {
	navigateCmd.Flags().Bool("wait", false, "Wait for page load completion")
	navigateCmd.Flags().Int("timeout", 60, "Timeout in seconds (used with --wait)")
	rootCmd.AddCommand(navigateCmd)
}

// normalizeURL adds protocol to URL if missing.
// Uses http:// for localhost/127.0.0.1/0.0.0.0, https:// otherwise.
func normalizeURL(url string) string {
	// Already has protocol
	if strings.Contains(url, "://") {
		return url
	}

	// Check for localhost or local IPs
	lower := strings.ToLower(url)
	if strings.HasPrefix(lower, "localhost") ||
		strings.HasPrefix(lower, "127.0.0.1") ||
		strings.HasPrefix(lower, "0.0.0.0") {
		return "http://" + url
	}

	// Default to https
	return "https://" + url
}

func runNavigate(cmd *cobra.Command, args []string) error {
	t := startTimer("navigate")
	defer t.log()

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Read flags
	wait, _ := cmd.Flags().GetBool("wait")
	timeout, _ := cmd.Flags().GetInt("timeout")

	// Normalize URL (add protocol if missing)
	url := normalizeURL(args[0])

	debugParam("url=%q wait=%v timeout=%d", url, wait, timeout)

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	// Send navigate request
	params, err := json.Marshal(ipc.NavigateParams{
		URL:     url,
		Wait:    wait,
		Timeout: timeout,
	})
	if err != nil {
		return outputError(err.Error())
	}

	debugRequest("navigate", fmt.Sprintf("url=%q wait=%v timeout=%d", url, wait, timeout))
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "navigate",
		Params: params,
	})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	// JSON mode: include URL and title
	if JSONOutput {
		var data ipc.NavigateData
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return outputError(err.Error())
		}

		result := map[string]any{
			"ok":    true,
			"url":   data.URL,
			"title": data.Title,
		}
		return outputJSON(os.Stdout, result)
	}

	// Text mode: just output OK
	return outputSuccess(nil)
}
