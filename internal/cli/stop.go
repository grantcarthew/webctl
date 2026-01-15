package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the daemon",
	Long: `Sends a shutdown command to the running daemon, which cleanly closes the browser and exits.

Use --force to forcefully terminate processes and clean up stale files when
the daemon is unresponsive or processes are orphaned.

Force cleanup sequence:
  1. Attempt graceful shutdown via IPC
  2. Kill daemon process from PID file
  3. Kill browser process on CDP port
  4. Remove stale socket and PID files`,
	RunE: runStop,
}

var (
	stopForce bool
	stopPort  int
)

func init() {
	stopCmd.Flags().BoolVar(&stopForce, "force", false, "Force kill processes and clean up stale files")
	stopCmd.Flags().IntVar(&stopPort, "port", 9222, "CDP port for browser process discovery (used with --force)")
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	t := startTimer("stop")
	defer t.log()

	debugParam("force=%v port=%d", stopForce, stopPort)

	// Try graceful shutdown first
	gracefulOK := tryGracefulShutdown()

	// If graceful shutdown worked and not forcing, we're done
	if gracefulOK && !stopForce {
		if JSONOutput {
			return outputSuccess(map[string]string{
				"message": "daemon stopped",
			})
		}
		return outputSuccess(nil)
	}

	// If not forcing, report graceful shutdown failure
	if !stopForce {
		return outputError("daemon not running or not responding")
	}

	// Force mode: clean up everything
	return forceCleanup()
}

// tryGracefulShutdown attempts to stop the daemon via IPC.
// Returns true if successful, false otherwise.
func tryGracefulShutdown() bool {
	exec, err := execFactory.NewExecutor()
	if err != nil {
		debugf("STOP", "failed to create executor: %v", err)
		return false
	}
	defer exec.Close()

	debugRequest("shutdown", "")
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{Cmd: "shutdown"})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		debugf("STOP", "IPC error: %v", err)
		return false
	}

	if !resp.OK {
		debugf("STOP", "shutdown failed: %s", resp.Error)
		return false
	}

	return true
}

// forceCleanup performs forceful cleanup of daemon, browser, and stale files.
func forceCleanup() error {
	var cleaned []string
	var errors []string

	// 1. Kill daemon from PID file
	pidPath := ipc.DefaultPIDPath()
	if pid, err := readPIDFile(pidPath); err == nil {
		if killProcess(pid) {
			cleaned = append(cleaned, fmt.Sprintf("killed daemon (PID %d)", pid))
			debugf("STOP", "killed daemon PID %d", pid)
		} else {
			debugf("STOP", "daemon PID %d not running", pid)
		}
	} else {
		debugf("STOP", "no PID file or error: %v", err)
	}

	// 2. Kill browser process on CDP port
	if browserPID := findBrowserOnPort(stopPort); browserPID > 0 {
		if killProcess(browserPID) {
			cleaned = append(cleaned, fmt.Sprintf("killed browser (PID %d) on port %d", browserPID, stopPort))
			debugf("STOP", "killed browser PID %d on port %d", browserPID, stopPort)
		} else {
			debugf("STOP", "browser PID %d not running or permission denied", browserPID)
		}
	} else {
		debugf("STOP", "no browser found on port %d", stopPort)
	}

	// 3. Remove stale socket file
	socketPath := ipc.DefaultSocketPath()
	if err := os.Remove(socketPath); err == nil {
		cleaned = append(cleaned, "removed socket file")
		debugf("STOP", "removed socket: %s", socketPath)
	} else if !os.IsNotExist(err) {
		errors = append(errors, fmt.Sprintf("failed to remove socket: %v", err))
	}

	// 4. Remove stale PID file
	if err := os.Remove(pidPath); err == nil {
		cleaned = append(cleaned, "removed PID file")
		debugf("STOP", "removed PID file: %s", pidPath)
	} else if !os.IsNotExist(err) {
		errors = append(errors, fmt.Sprintf("failed to remove PID file: %v", err))
	}

	// Report results
	if len(errors) > 0 {
		return outputError(strings.Join(errors, "; "))
	}

	if len(cleaned) == 0 {
		if JSONOutput {
			return outputSuccess(map[string]string{
				"message": "nothing to clean up",
			})
		}
		fmt.Fprintln(os.Stdout, "Nothing to clean up")
		return nil
	}

	if JSONOutput {
		return outputSuccess(map[string]any{
			"message": "force cleanup complete",
			"actions": cleaned,
		})
	}

	// Text mode: output each action
	for _, action := range cleaned {
		fmt.Fprintln(os.Stdout, action)
	}
	return nil
}

// readPIDFile reads the PID from the given file path.
func readPIDFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}
	return pid, nil
}

// killProcess sends SIGKILL to the given PID.
// Returns true if the process was killed, false if it wasn't running or permission denied.
func killProcess(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Kill the process directly - Kill() returns an error if process doesn't exist
	if err := process.Kill(); err != nil {
		return false
	}
	return true
}

// findBrowserOnPort uses lsof to find a Chrome/Chromium process listening on the given port.
// Returns the PID if found, 0 otherwise.
func findBrowserOnPort(port int) int {
	// Use lsof to find process listening on the port
	cmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port), "-sTCP:LISTEN", "-t")
	output, err := cmd.Output()
	if err != nil {
		debugf("STOP", "lsof error: %v", err)
		return 0
	}

	// Parse PIDs from output (one per line)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		pidStr := strings.TrimSpace(scanner.Text())
		if pid, err := strconv.Atoi(pidStr); err == nil {
			// Verify it's a Chrome/Chromium process
			if isBrowserProcess(pid) {
				return pid
			}
		}
	}

	return 0
}

// isBrowserProcess checks if the given PID is a Chrome/Chromium process.
func isBrowserProcess(pid int) bool {
	// Read the process command from /proc or use ps on macOS
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	comm := strings.ToLower(strings.TrimSpace(string(output)))
	return strings.Contains(comm, "chrome") || strings.Contains(comm, "chromium")
}
