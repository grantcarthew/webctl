package cli

import (
	"context"

	"github.com/grantcarthew/webctl/internal/daemon"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start daemon and browser",
	Long:  "Starts the webctl daemon which launches a browser and begins capturing CDP events.",
	RunE:  runStart,
}

var (
	startHeadless bool
	startPort     int
)

func init() {
	startCmd.Flags().BoolVar(&startHeadless, "headless", false, "Run browser in headless mode")
	startCmd.Flags().IntVar(&startPort, "port", 9222, "CDP port for browser")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	// Check if daemon is already running
	if dialer.IsDaemonRunning() {
		return outputError("daemon is already running")
	}

	cfg := daemon.DefaultConfig()
	cfg.Headless = startHeadless
	cfg.Port = startPort

	d := daemon.New(cfg)

	// Output startup message
	outputSuccess(map[string]any{
		"message": "daemon starting",
		"port":    startPort,
	})

	// Run daemon (blocks until shutdown)
	if err := d.Run(context.Background()); err != nil {
		return outputError(err.Error())
	}

	return nil
}
