package cli

import (
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

// Note: ipc import kept for ipc.Request type

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
	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	target := ""
	if len(args) > 0 {
		target = args[0]
		// Validate target
		if target != "console" && target != "network" {
			return outputError("invalid target: must be 'console' or 'network'")
		}
	}

	resp, err := exec.Execute(ipc.Request{
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
