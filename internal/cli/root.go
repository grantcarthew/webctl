package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Version is set at build time.
var Version = "dev"

// Debug enables verbose debug output.
var Debug bool

var rootCmd = &cobra.Command{
	Use:           "webctl",
	Short:         "Browser automation CLI for AI agents",
	Long:          "webctl captures DevTools data (console logs, network requests, JS errors) via a persistent daemon that buffers CDP events.",
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&Debug, "debug", false, "Enable verbose debug output")
}

// debugf logs a debug message if debug mode is enabled.
func debugf(format string, args ...any) {
	if Debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// ExecuteArgs runs a command with the given arguments.
// Used by the REPL to execute commands parsed from user input.
// Returns true if the command was recognized (even if it failed), false if unknown.
func ExecuteArgs(args []string) (recognized bool, err error) {
	if len(args) == 0 {
		return false, nil
	}

	// Check if command exists before executing
	cmd, _, findErr := rootCmd.Find(args)
	if findErr != nil || cmd == rootCmd {
		return false, nil
	}

	// Reset flags to defaults before each REPL command execution.
	// Cobra persists flag values between calls, so we must reset them.
	resetCommandFlags()

	rootCmd.SetArgs(args)
	err = rootCmd.Execute()
	return true, err
}

// resetCommandFlags resets all command flags to their default values.
// This is necessary for REPL usage where commands are executed repeatedly.
//
// TODO: Consider a registration pattern where each command registers its own
// reset function, rather than maintaining this central list.
//
// IMPORTANT: When adding new commands with flags, add their reset logic here.
func resetCommandFlags() {
	// Console command flags
	consoleFormat = ""
	consoleTypes = nil
	consoleHead = 0
	consoleTail = 0
	consoleRange = ""

	// Network command flags
	networkFormat = ""
	networkTypes = nil
	networkMethods = nil
	networkStatuses = nil
	networkURL = ""
	networkMimes = nil
	networkMinDuration = 0
	networkMinSize = 0
	networkFailed = false
	networkMaxBodySize = 102400
	networkHead = 0
	networkTail = 0
	networkRange = ""

	// Screenshot command flags
	screenshotFullPage = false
	screenshotOutput = ""

	// HTML command flags
	htmlOutput = ""

	// Reload command flags
	reloadIgnoreCache = false

	// Ready command flags
	readyTimeout = 30 * time.Second

	// Key command flags
	keyCtrl = false
	keyAlt = false
	keyShift = false
	keyMeta = false

	// Type command flags
	typeKey = ""
	typeClear = false

	// Scroll command flags
	scrollTo = ""
	scrollBy = ""

	// Eval command flags
	evalTimeout = 30 * time.Second
}

// isStdoutTTY returns true if stdout is a terminal.
func isStdoutTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// outputJSON writes a JSON response to the given writer.
// Pretty prints if stdout is a TTY, compact otherwise.
func outputJSON(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	if isStdoutTTY() {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(data)
}

// outputSuccess writes a successful JSON response to stdout.
func outputSuccess(data any) error {
	resp := map[string]any{
		"ok": true,
	}
	if data != nil {
		resp["data"] = data
	}
	return outputJSON(os.Stdout, resp)
}

// outputError writes an error JSON response to stderr and returns an error.
func outputError(msg string) error {
	resp := map[string]any{
		"ok":    false,
		"error": msg,
	}
	outputJSON(os.Stderr, resp)
	return fmt.Errorf("%s", msg)
}
