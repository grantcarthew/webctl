package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/grantcarthew/webctl/internal/cli/format"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var consoleCmd = &cobra.Command{
	Use:   "console",
	Short: "Show console log entries",
	Long:  "Returns buffered console log entries including logs, warnings, errors, and exceptions.",
	RunE:  runConsole,
}

func init() {
	consoleCmd.Flags().StringSlice("type", nil, "Filter by entry type (repeatable, CSV-supported)")
	consoleCmd.Flags().Int("head", 0, "Return first N entries")
	consoleCmd.Flags().Int("tail", 0, "Return last N entries")
	consoleCmd.Flags().String("range", "", "Return entries in range (format: START-END)")
	consoleCmd.MarkFlagsMutuallyExclusive("head", "tail", "range")
	rootCmd.AddCommand(consoleCmd)
}

func runConsole(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Read flags from command
	types, _ := cmd.Flags().GetStringSlice("type")
	head, _ := cmd.Flags().GetInt("head")
	tail, _ := cmd.Flags().GetInt("tail")
	rangeStr, _ := cmd.Flags().GetString("range")

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	resp, err := exec.Execute(ipc.Request{Cmd: "console"})
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	var data ipc.ConsoleData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return outputError(err.Error())
	}

	entries := data.Entries

	// Apply type filter
	if len(types) > 0 {
		entries = filterConsoleByType(entries, types)
	}

	// Apply limiting (head/tail/range)
	entries, err = applyConsoleLimiting(entries, head, tail, rangeStr)
	if err != nil {
		return outputError(err.Error())
	}

	// JSON mode: output JSON
	if JSONOutput {
		return outputConsoleJSON(entries)
	}

	// Text mode: use text formatter
	return format.Console(os.Stdout, entries, format.DefaultOptions())
}

// filterConsoleByType filters entries to only include those with matching types.
func filterConsoleByType(entries []ipc.ConsoleEntry, types []string) []ipc.ConsoleEntry {
	typeSet := make(map[string]bool)
	for _, t := range types {
		typeSet[strings.ToLower(t)] = true
	}

	var filtered []ipc.ConsoleEntry
	for _, e := range entries {
		if typeSet[strings.ToLower(e.Type)] {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// applyConsoleLimiting applies head, tail, or range limiting to entries.
func applyConsoleLimiting(entries []ipc.ConsoleEntry, head, tail int, rangeStr string) ([]ipc.ConsoleEntry, error) {
	if head > 0 {
		if head > len(entries) {
			head = len(entries)
		}
		return entries[:head], nil
	}

	if tail > 0 {
		if tail > len(entries) {
			tail = len(entries)
		}
		return entries[len(entries)-tail:], nil
	}

	if rangeStr != "" {
		parts := strings.Split(rangeStr, "-")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid range format: use START-END (e.g., 100-200)")
		}
		start, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid range format: use START-END (e.g., 100-200)")
		}
		end, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid range format: use START-END (e.g., 100-200)")
		}
		if start < 0 {
			start = 0
		}
		if end > len(entries) {
			end = len(entries)
		}
		if start >= end {
			return []ipc.ConsoleEntry{}, nil
		}
		return entries[start:end], nil
	}

	return entries, nil
}

// outputConsoleJSON outputs entries in JSON format.
func outputConsoleJSON(entries []ipc.ConsoleEntry) error {
	resp := map[string]any{
		"ok":      true,
		"entries": entries,
		"count":   len(entries),
	}
	return outputJSON(os.Stdout, resp)
}
