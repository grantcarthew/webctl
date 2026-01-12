package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

// ErrNoMatches indicates a search found no matches (informational, not an error).
var ErrNoMatches = errors.New("no matches found")

// ErrNoElements indicates a selector matched no elements (informational, not an error).
var ErrNoElements = errors.New("no elements found")

// isNoElementsError checks if an error message indicates no elements were found.
func isNoElementsError(msg string) bool {
	return strings.Contains(msg, "matched no elements")
}

// Version is set at build time.
var Version = "dev"

// Debug enables verbose debug output.
var Debug bool

// JSONOutput enables JSON output format (default is text).
var JSONOutput bool

// NoColor disables color output.
var NoColor bool

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
	rootCmd.PersistentFlags().BoolVar(&JSONOutput, "json", false, "Output in JSON format (default is text)")
	rootCmd.PersistentFlags().BoolVar(&NoColor, "no-color", false, "Disable color output")
	rootCmd.SetVersionTemplate(`webctl version {{.Version}}
Repository: https://github.com/grantcarthew/webctl
Report issues: https://github.com/grantcarthew/webctl/issues/new
`)
}

// debugf logs a debug message if debug mode is enabled.
func debugf(format string, args ...any) {
	if Debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// Execute runs the root command.
// Supports command abbreviation via unique prefix matching.
func Execute() error {
	// Try abbreviation expansion for CLI commands
	args := os.Args[1:]
	if len(args) > 0 {
		if expanded := tryExpandCommand(args[0]); expanded != "" {
			args[0] = expanded
			rootCmd.SetArgs(args)
		}
	}
	return rootCmd.Execute()
}

// tryExpandCommand attempts to expand a command abbreviation.
// Returns the expanded command if exactly one match is found, empty string otherwise.
func tryExpandCommand(prefix string) string {
	// Get all subcommands of root
	var commands []string
	for _, cmd := range rootCmd.Commands() {
		commands = append(commands, cmd.Name())
	}

	// Try to expand
	var matches []string
	for _, cmd := range commands {
		if cmd == prefix {
			// Exact match, no expansion needed
			return ""
		}
		if len(prefix) < len(cmd) && cmd[:len(prefix)] == prefix {
			matches = append(matches, cmd)
		}
	}

	// Return expanded command only if exactly one match
	if len(matches) == 1 {
		return matches[0]
	}
	return ""
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
	resetFlags := func(flags *pflag.FlagSet) {
		flags.VisitAll(func(f *pflag.Flag) {
			// For slice types with DefValue "[]", use empty string to properly reset.
			// Using Set("[]") would incorrectly create a slice containing "[]" as a literal.
			defVal := f.DefValue
			if defVal == "[]" {
				defVal = ""
			}
			_ = f.Value.Set(defVal)
			f.Changed = false
		})
	}

	// Reset both local flags and persistent flags
	resetFlags(cmd.Flags())
	resetFlags(cmd.PersistentFlags())

	// Also reset persistent flags from parent commands
	for parent := cmd.Parent(); parent != nil; parent = parent.Parent() {
		resetFlags(parent.PersistentFlags())
	}

	// Reset global flag variables to their defaults
	// (BoolVar bindings should update automatically via Set(), but we ensure it here)
	Debug = false
	JSONOutput = false
	NoColor = false

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

// outputSuccess writes a successful response to stdout.
// Uses text format by default, JSON if --json flag is set.
// For action commands (no data), outputs "OK" in text mode.
func outputSuccess(data any) error {
	if JSONOutput {
		resp := map[string]any{
			"ok": true,
		}
		if data != nil {
			resp["data"] = data
		}
		return outputJSON(os.Stdout, resp)
	}

	// Text mode: just "OK" for action commands (no data)
	if data == nil {
		if shouldUseColor() {
			color.New(color.FgGreen).Fprintln(os.Stdout, "OK")
		} else {
			fmt.Fprintln(os.Stdout, "OK")
		}
		return nil
	}

	// For commands with data, they should use their own formatters
	// This fallback shouldn't be hit in normal usage
	_, err := fmt.Fprintf(os.Stdout, "%v\n", data)
	return err
}

// outputError writes an error response to stderr and returns an error.
// Uses text format by default, JSON if --json flag is set.
func outputError(msg string) error {
	if JSONOutput {
		resp := map[string]any{
			"ok":    false,
			"error": msg,
		}
		outputJSON(os.Stderr, resp)
	} else {
		// Apply color to error prefix if colors are enabled
		if shouldUseColor() {
			color.New(color.FgRed).Fprint(os.Stderr, "Error:")
			fmt.Fprintf(os.Stderr, " %s\n", msg)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
		}
	}
	return fmt.Errorf("%s", msg)
}

// outputNotice writes a notice message to stderr without "Error:" prefix.
// Used for informational messages that still result in non-zero exit code.
func outputNotice(msg string) error {
	if JSONOutput {
		resp := map[string]any{
			"ok":      false,
			"message": msg,
		}
		outputJSON(os.Stderr, resp)
	} else {
		fmt.Fprintln(os.Stderr, msg)
	}
	return errors.New(msg)
}

// shouldUseColor determines if color output should be used based on flags and environment.
func shouldUseColor() bool {
	if JSONOutput {
		return false
	}
	if NoColor {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stderr.Fd()))
}
