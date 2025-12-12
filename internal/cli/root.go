package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:           "webctl",
	Short:         "Browser automation CLI for AI agents",
	Long:          "webctl captures DevTools data (console logs, network requests, JS errors) via a persistent daemon that buffers CDP events.",
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// outputJSON writes a JSON response to the given writer.
func outputJSON(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
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
