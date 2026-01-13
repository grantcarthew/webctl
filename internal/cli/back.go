package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var backCmd = &cobra.Command{
	Use:   "back",
	Short: "Navigate to previous page",
	Long:  "Navigates to the previous page in browser history. Returns immediately unless --wait is specified. Returns an error if there is no previous page.",
	Args:  cobra.NoArgs,
	RunE:  runBack,
}

func init() {
	backCmd.Flags().Bool("wait", false, "Wait for page load completion")
	backCmd.Flags().Int("timeout", 60, "Timeout in seconds (used with --wait)")
	rootCmd.AddCommand(backCmd)
}

func runBack(cmd *cobra.Command, args []string) error {
	t := startTimer("back")
	defer t.log()

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Read flags
	wait, _ := cmd.Flags().GetBool("wait")
	timeout, _ := cmd.Flags().GetInt("timeout")

	debugParam("wait=%v timeout=%d", wait, timeout)

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	params, err := json.Marshal(ipc.HistoryParams{
		Wait:    wait,
		Timeout: timeout,
	})
	if err != nil {
		return outputError(err.Error())
	}

	debugRequest("back", fmt.Sprintf("wait=%v timeout=%d", wait, timeout))
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "back",
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
