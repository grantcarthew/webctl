package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/grantcarthew/webctl/internal/browser"
	"github.com/grantcarthew/webctl/internal/daemon"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start daemon and browser",
	Long: `Starts the webctl daemon which launches a browser and begins capturing CDP events.

Profile modes (mutually exclusive; default is the persistent profile):
  (default)            Persistent profile at $XDG_DATA_HOME/webctl/profile
                       (falls back to ~/.local/share/webctl/profile). Logins,
                       cookies, and site state persist across restarts.
  --temp-profile       Throwaway profile created on start and deleted on stop.
  --user-data-dir DIR  Use DIR as the profile. webctl never deletes it.
  --system-profile     Use your real Chrome profile. Requires that no other
                       Chrome instance is running on the default profile, or the
                       launch forwards to it and webctl cannot attach.`,
	RunE: runStart,
}

var (
	startHeadless      bool
	startPort          int
	startTempProfile   bool
	startUserDataDir   string
	startSystemProfile bool
)

func init() {
	startCmd.Flags().BoolVar(&startHeadless, "headless", false, "Run browser in headless mode")
	startCmd.Flags().IntVar(&startPort, "port", 9222, "CDP port for browser")
	startCmd.Flags().BoolVar(&startTempProfile, "temp-profile", false, "Use a throwaway profile, deleted on stop")
	startCmd.Flags().StringVar(&startUserDataDir, "user-data-dir", "", "Use an explicit profile directory, never deleted by webctl")
	startCmd.Flags().BoolVar(&startSystemProfile, "system-profile", false, "Use the real Chrome profile (no other Chrome may run on it)")
	rootCmd.AddCommand(startCmd)
}

// startupErrorHint returns the guidance line for a daemon startup failure, or
// "" when no hint applies. A held system profile is matched by its typed error
// so it does not fall through to the orphan-reaping hint, which would be wrong
// advice for an externally launched Chrome.
func startupErrorHint(err error) string {
	switch {
	case errors.Is(err, browser.ErrSystemProfileInUse):
		return "close the running Chrome on the default profile, or start with the persistent default profile or --temp-profile"
	case errors.Is(err, browser.ErrPortInUse):
		return "use 'webctl stop --force' to kill orphaned processes"
	}
	return ""
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

	userDataDir, err := resolveProfile(startTempProfile, startUserDataDir, cmd.Flags().Changed("user-data-dir"), startSystemProfile)
	if err != nil {
		return outputError(err.Error())
	}
	debugParam("profile=%q", userDataDir)

	cfg := daemon.DefaultConfig()
	cfg.Headless = startHeadless
	cfg.Port = startPort
	cfg.UserDataDir = userDataDir
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

	// Report success only once the daemon is serving IPC, so a start that fails
	// before readiness emits its error without a preceding success line. Run
	// invokes this from within its blocking call, before any terminal-mode change.
	cfg.ReadyCallback = func(port int) {
		if JSONOutput {
			_ = outputSuccess(map[string]any{
				"message": "daemon ready",
				"port":    port,
			})
		} else {
			// Text mode: just output OK
			_ = outputSuccess(nil)
		}
	}

	d = daemon.New(cfg)

	// Run daemon (blocks until shutdown)
	if err := d.Run(context.Background()); err != nil {
		outErr := outputError(err.Error())
		if hint := startupErrorHint(err); hint != "" {
			outputHint(hint)
		}
		return outErr
	}

	return nil
}
