package cli

import (
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the daemon",
	Long:  "Sends a shutdown command to the running daemon, which cleanly closes the browser and exits.",
	RunE:  runStop,
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	client, err := dialer.Dial()
	if err != nil {
		return outputError(err.Error())
	}
	defer client.Close()

	resp, err := client.SendCmd("shutdown")
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	return outputSuccess(map[string]string{
		"message": "daemon stopped",
	})
}
