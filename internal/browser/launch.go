package browser

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// LaunchOptions configures browser launch behavior.
type LaunchOptions struct {
	// Headless runs the browser without a visible window.
	Headless bool

	// Port for CDP remote debugging. If 0, uses default 9222.
	Port int

	// UserDataDir specifies the browser profile directory.
	// Special values:
	//   - Empty string: create a temporary directory (default)
	//   - "default": use the user's default Chrome profile
	//   - Any path: use that directory
	UserDataDir string
}

// DefaultPort is the default CDP debugging port.
const DefaultPort = 9222

// UserDataDirDefault is the special value that means "use the user's Chrome profile".
const UserDataDirDefault = "default"

// buildArgs constructs the Chrome command line arguments.
func buildArgs(opts LaunchOptions) []string {
	port := opts.Port
	if port == 0 {
		port = DefaultPort
	}

	args := []string{
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-networking",
		"--disable-sync",
		"--disable-popup-blocking",
	}

	// Platform-specific flags to avoid system dialogs
	switch runtime.GOOS {
	case "darwin":
		args = append(args, "--use-mock-keychain")
	case "linux":
		args = append(args, "--password-store=basic")
	}

	if opts.Headless {
		args = append(args, "--headless")
	}

	// Handle user data directory:
	// - Empty or "default": no flag (use user's Chrome profile)
	// - Any path: use that directory
	if opts.UserDataDir != "" && opts.UserDataDir != UserDataDirDefault {
		args = append(args, fmt.Sprintf("--user-data-dir=%s", opts.UserDataDir))
	}

	// Open about:blank to avoid any default page loading
	args = append(args, "about:blank")

	return args
}

// createTempDataDir creates a temporary directory for browser profile data.
func createTempDataDir() (string, error) {
	return os.MkdirTemp("", "webctl-chrome-*")
}

// spawnProcess starts the browser process with the given binary and options.
// It does not wait for the process to exit.
// Returns the command, the data directory (empty if using default profile), and any error.
func spawnProcess(binPath string, opts LaunchOptions) (*exec.Cmd, string, error) {
	var dataDir string
	var createdTempDir bool

	switch opts.UserDataDir {
	case "":
		// Empty: create a temporary directory
		var err error
		dataDir, err = createTempDataDir()
		if err != nil {
			return nil, "", fmt.Errorf("create temp dir: %w", err)
		}
		opts.UserDataDir = dataDir
		createdTempDir = true
	case UserDataDirDefault:
		// "default": use user's Chrome profile, no temp dir
		dataDir = ""
	default:
		// Custom path: use as-is
		dataDir = opts.UserDataDir
	}

	args := buildArgs(opts)
	cmd := exec.Command(binPath, args...)

	// Detach from controlling terminal
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		// Clean up temp dir on failure
		if createdTempDir && dataDir != "" {
			os.RemoveAll(dataDir)
		}
		return nil, "", fmt.Errorf("start browser: %w", err)
	}

	return cmd, dataDir, nil
}
