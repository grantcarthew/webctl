package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var consoleCmd = &cobra.Command{
	Use:   "console",
	Short: "Show console log entries",
	Long:  "Returns buffered console log entries including logs, warnings, errors, and exceptions.",
	RunE:  runConsole,
}

var (
	consoleFormat string
	consoleTypes  []string
	consoleHead   int
	consoleTail   int
	consoleRange  string
)

func init() {
	consoleCmd.Flags().StringVar(&consoleFormat, "format", "", "Output format: json or text (auto-detect by default)")
	consoleCmd.Flags().StringSliceVar(&consoleTypes, "type", nil, "Filter by entry type (repeatable, CSV-supported)")
	consoleCmd.Flags().IntVar(&consoleHead, "head", 0, "Return first N entries")
	consoleCmd.Flags().IntVar(&consoleTail, "tail", 0, "Return last N entries")
	consoleCmd.Flags().StringVar(&consoleRange, "range", "", "Return entries in range (format: START-END)")
	consoleCmd.MarkFlagsMutuallyExclusive("head", "tail", "range")
	rootCmd.AddCommand(consoleCmd)
}

func runConsole(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

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
	if len(consoleTypes) > 0 {
		entries = filterConsoleByType(entries, consoleTypes)
	}

	// Apply limiting (head/tail/range)
	entries, err = applyConsoleLimiting(entries, consoleHead, consoleTail, consoleRange)
	if err != nil {
		return outputError(err.Error())
	}

	// Determine output format
	format := consoleFormat
	if format == "" {
		format = "json"
	}

	if format == "text" {
		return outputConsoleText(entries)
	}
	return outputConsoleJSON(entries, isStdoutTTY())
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

// isStdoutTTY returns true if stdout is a terminal.
func isStdoutTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// outputConsoleText outputs entries in human-readable text format.
func outputConsoleText(entries []ipc.ConsoleEntry) error {
	for _, e := range entries {
		ts := time.UnixMilli(e.Timestamp).Local()
		timestamp := ts.Format("2006-01-02 15:04:05.000")

		var source string
		if e.URL != "" {
			filename := filepath.Base(e.URL)
			if e.Line > 0 {
				source = fmt.Sprintf("%s:%d", filename, e.Line)
			} else {
				source = filename
			}
		}

		if source != "" {
			fmt.Printf("[%s] %s %s\n", timestamp, source, e.Text)
		} else {
			fmt.Printf("[%s] %s\n", timestamp, e.Text)
		}
	}
	return nil
}

// outputConsoleJSON outputs entries in JSON format.
func outputConsoleJSON(entries []ipc.ConsoleEntry, pretty bool) error {
	resp := map[string]any{
		"ok":      true,
		"entries": entries,
		"count":   len(entries),
	}

	enc := json.NewEncoder(os.Stdout)
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(resp)
}
