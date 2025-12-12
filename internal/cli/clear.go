package cli

import (
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var clearCmd = &cobra.Command{
	Use:   "clear [console|network]",
	Short: "Clear event buffers",
	Long:  "Clears the console and/or network event buffers. Specify 'console' or 'network' to clear only that buffer, or omit to clear all.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runClear,
}

func init() {
	rootCmd.AddCommand(clearCmd)
}

func runClear(cmd *cobra.Command, args []string) error {
	client, err := ipc.Dial()
	if err != nil {
		if err == ipc.ErrDaemonNotRunning {
			return outputError("daemon is not running")
		}
		return outputError(err.Error())
	}
	defer client.Close()

	target := ""
	if len(args) > 0 {
		target = args[0]
		// Validate target
		if target != "console" && target != "network" {
			return outputError("invalid target: must be 'console' or 'network'")
		}
	}

	resp, err := client.Send(ipc.Request{
		Cmd:    "clear",
		Target: target,
	})
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	msg := "all buffers cleared"
	if target != "" {
		msg = target + " buffer cleared"
	}

	return outputSuccess(map[string]string{
		"message": msg,
	})
}
