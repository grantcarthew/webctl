package cli

import (
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
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
	t := startTimer("stop")
	defer t.log()

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	debugRequest("shutdown", "")
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{Cmd: "shutdown"})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	// JSON mode: include message
	if JSONOutput {
		return outputSuccess(map[string]string{
			"message": "daemon stopped",
		})
	}

	// Text mode: just output OK
	return outputSuccess(nil)
}
