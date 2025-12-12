package cli

import (
	"encoding/json"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

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
	if !ipc.IsDaemonRunning() {
		return outputSuccess(map[string]any{
			"running": false,
		})
	}

	client, err := ipc.Dial()
	if err != nil {
		return outputError(err.Error())
	}
	defer client.Close()

	resp, err := client.SendCmd("status")
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
