package cli

import (
	"context"
	"fmt"
	"strings"

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
	t := startTimer("start")
	defer t.log()

	// Check if daemon is already running
	if execFactory.IsDaemonRunning() {
		_ = outputError("daemon is already running")
		outputHint("use 'webctl stop' to stop the daemon, or 'webctl stop --force' to force cleanup")
		return printedError{err: fmt.Errorf("daemon is already running")}
	}

	debugParam("headless=%v port=%d", startHeadless, startPort)

	cfg := daemon.DefaultConfig()
	cfg.Headless = startHeadless
	cfg.Port = startPort
	cfg.Debug = Debug

	// Declare d first so the closure can capture it.
	// The closure is only called when REPL executes commands, by which time d is set.
	var d *daemon.Daemon

	// Create command executor for REPL that uses Cobra with direct execution.
	cfg.CommandExecutor = func(args []string) (bool, error) {
		factory := NewDirectExecutorFactory(d.Handler())
		SetExecutorFactory(factory)
		defer ResetExecutorFactory()
		return ExecuteArgs(args)
	}

	d = daemon.New(cfg)

	// Output startup message
	if JSONOutput {
		_ = outputSuccess(map[string]any{
			"message": "daemon starting",
			"port":    startPort,
		})
	} else {
		// Text mode: just output OK
		_ = outputSuccess(nil)
	}

	// Run daemon (blocks until shutdown)
	if err := d.Run(context.Background()); err != nil {
		outErr := outputError(err.Error())
		// Add hint for port-in-use errors
		if strings.Contains(err.Error(), "port") || strings.Contains(err.Error(), "in use") {
			outputHint("use 'webctl stop --force' to kill orphaned processes")
		}
		return outErr
	}

	return nil
}
