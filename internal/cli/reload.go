package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

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
	reloadCmd.Flags().Int("timeout", 60, "Timeout in seconds (used with --wait)")
	rootCmd.AddCommand(reloadCmd)
}

func runReload(cmd *cobra.Command, args []string) error {
	t := startTimer("reload")
	defer t.log()

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Read flags
	wait, _ := cmd.Flags().GetBool("wait")
	timeout, _ := cmd.Flags().GetInt("timeout")

	debugParam("wait=%v timeout=%d ignoreCache=true", wait, timeout)

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

	debugRequest("reload", fmt.Sprintf("wait=%v timeout=%d ignoreCache=true", wait, timeout))
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "reload",
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
