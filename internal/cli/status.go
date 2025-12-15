package cli

import (
	"encoding/json"

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
	// Check if daemon is running
	if !execFactory.IsDaemonRunning() {
		return outputSuccess(map[string]any{
			"running": false,
		})
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	resp, err := exec.Execute(ipc.Request{Cmd: "status"})
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	// Parse and output status data
	var status ipc.StatusData
	if err := json.Unmarshal(resp.Data, &status); err != nil {
		return outputError(err.Error())
	}

	return outputSuccess(status)
}
