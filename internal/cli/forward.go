package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var forwardCmd = &cobra.Command{
	Use:   "forward",
	Short: "Navigate to next page",
	Long:  "Navigates to the next page in browser history. Returns immediately unless --wait is specified. Returns an error if there is no next page.",
	Args:  cobra.NoArgs,
	RunE:  runForward,
}

func init() {
	forwardCmd.Flags().Bool("wait", false, "Wait for page load completion")
	forwardCmd.Flags().Int("timeout", 60, "Timeout in seconds (used with --wait)")
	rootCmd.AddCommand(forwardCmd)
}

func runForward(cmd *cobra.Command, args []string) error {
	t := startTimer("forward")
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
	defer func() { _ = exec.Close() }()

	params, err := json.Marshal(ipc.HistoryParams{
		Wait:    wait,
		Timeout: timeout,
	})
	if err != nil {
		return outputError(err.Error())
	}

	debugRequest("forward", fmt.Sprintf("wait=%v timeout=%d", wait, timeout))
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "forward",
		Params: params,
	})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		if isNoHistoryError(resp.Error) {
			return outputNotice("No next page")
		}
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
