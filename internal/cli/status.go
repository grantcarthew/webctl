package cli

import (
	"encoding/json"
	"os"
	"time"

	"github.com/grantcarthew/webctl/internal/cli/format"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

// Note: ipc import kept for ipc.StatusData type

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	Long:  "Returns the current daemon status including whether it's running, the current URL, and page title.",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	t := startTimer("status")
	defer t.log()

	// Check if daemon is running
	if !execFactory.IsDaemonRunning() {
		debugf("PARAM", "daemon not running, returning offline status")
		status := ipc.StatusData{Running: false}

		if JSONOutput {
			return outputSuccess(map[string]any{
				"running": false,
			})
		}
		return format.Status(os.Stdout, status, format.NewOutputOptions(JSONOutput, NoColor))
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer func() { _ = exec.Close() }()

	debugRequest("status", "")
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{Cmd: "status"})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	// Parse status data
	var status ipc.StatusData
	if err := json.Unmarshal(resp.Data, &status); err != nil {
		return outputError(err.Error())
	}

	// JSON mode: output full JSON
	if JSONOutput {
		return outputSuccess(status)
	}

	// Text mode: use text formatter
	return format.Status(os.Stdout, status, format.NewOutputOptions(JSONOutput, NoColor))
}
