package cli

import (
	"encoding/json"
	"os"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload current page",
	Long:  "Reloads the current page in the active browser session (hard reload, ignores cache). Returns immediately unless --wait is specified.",
	Args:  cobra.NoArgs,
	RunE:  runReload,
}

func init() {
	reloadCmd.Flags().Bool("wait", false, "Wait for page load completion")
	reloadCmd.Flags().Int("timeout", 30000, "Timeout in milliseconds (used with --wait)")
	rootCmd.AddCommand(reloadCmd)
}

func runReload(cmd *cobra.Command, args []string) error {
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

	// Always do hard reload (ignore cache)
	params, err := json.Marshal(ipc.ReloadParams{
		IgnoreCache: true,
		Wait:        wait,
		Timeout:     timeout,
	})
	if err != nil {
		return outputError(err.Error())
	}

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "reload",
		Params: params,
	})
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

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
