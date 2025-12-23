package cli

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var navigateCmd = &cobra.Command{
	Use:   "navigate <url>",
	Short: "Navigate to URL",
	Long:  "Navigates the active browser session to the specified URL. Returns immediately unless --wait is specified.",
	Args:  cobra.ExactArgs(1),
	RunE:  runNavigate,
}

func init() {
	navigateCmd.Flags().Bool("wait", false, "Wait for page load completion")
	navigateCmd.Flags().Int("timeout", 30000, "Timeout in milliseconds (used with --wait)")
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
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Read flags
	wait, _ := cmd.Flags().GetBool("wait")
	timeout, _ := cmd.Flags().GetInt("timeout")

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	// Normalize URL (add protocol if missing)
	url := normalizeURL(args[0])

	// Send navigate request
	params, err := json.Marshal(ipc.NavigateParams{
		URL:     url,
		Wait:    wait,
		Timeout: timeout,
	})
	if err != nil {
		return outputError(err.Error())
	}

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "navigate",
		Params: params,
	})
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	// Parse and output the navigation result
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
