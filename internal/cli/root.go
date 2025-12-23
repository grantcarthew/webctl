package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

	rootCmd.SetArgs(args)
	err = rootCmd.Execute()

	// Reset flags to defaults AFTER each REPL command execution.
	// Since we read flags from cmd.Flags() in RunE (which gets values from Cobra's parsing),
	// we reset AFTER execution so the next call starts fresh.
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// For slice types with DefValue "[]", use empty string to properly reset.
		// Using Set("[]") would incorrectly create a slice containing "[]" as a literal.
		defVal := f.DefValue
		if defVal == "[]" {
			defVal = ""
		}
		_ = f.Value.Set(defVal)
		f.Changed = false
	})

	return true, err
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
