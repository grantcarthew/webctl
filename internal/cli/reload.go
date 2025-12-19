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
	Long:  "Reloads the current page in the active browser session. Waits for navigation to commit before returning.",
	Args:  cobra.NoArgs,
	RunE:  runReload,
}

var reloadIgnoreCache bool

func init() {
	reloadCmd.Flags().BoolVar(&reloadIgnoreCache, "ignore-cache", false, "Bypass browser cache (hard reload)")
	rootCmd.AddCommand(reloadCmd)
}

func runReload(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	params, err := json.Marshal(ipc.ReloadParams{
		IgnoreCache: reloadIgnoreCache,
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
