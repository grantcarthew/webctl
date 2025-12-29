package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/grantcarthew/webctl/internal/cli/format"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var consoleCmd = &cobra.Command{
	Use:   "console",
	Short: "Extract console logs from current page (default: save to temp)",
	Long: `Extracts console logs from the current page with flexible output modes.

Default behavior (no subcommand):
  Saves console logs to /tmp/webctl-console/ with auto-generated filename
  Returns JSON with file path

Subcommands:
  show              Output console logs to stdout
  save <path>       Save console logs to custom path

Universal flags (work with default/show/save modes):
  --find, -f        Search for text within log messages
  --raw             Skip formatting (return raw JSON)
  --json            Output in JSON format (global flag)

Console-specific filter flags:
  --type TYPE       Filter by log type (log, warn, error, debug, info)
  --head N          Return first N entries
  --tail N          Return last N entries
  --range N-M       Return entries N through M

Examples:

Default mode (save to temp):
  console                                  # All logs to temp
  console --type error                     # Only errors to temp
  console --find "undefined"               # Search and save matches

Show mode (stdout):
  console show                             # All logs to stdout
  console show --type error,warn           # Only errors/warnings
  console show --find "TypeError"          # Search and show matches
  console show --tail 20                   # Last 20 entries

Save mode (custom path):
  console save ./logs/debug.json           # Save to file
  console save ./output/                   # Save to dir (auto-filename)
  console save ./errors.json --type error --tail 50

Response formats:
  Default/Save: {"ok": true, "path": "/tmp/webctl-console/25-12-28-143052-console.json"}
  Show:         [15:04:05] ERROR TypeError: undefined (to stdout)

Error cases:
  - "no matches found for 'text'" - find text not in logs
  - "daemon not running" - start daemon first with: webctl start`,
	RunE: runConsoleDefault,
}

var consoleShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Output console logs to stdout",
	Long: `Outputs console logs to stdout for real-time monitoring and piping.

Examples:
  console show                             # All logs
  console show --type error                # Only errors
  console show --find "undefined"          # Search within logs
  console show --tail 20                   # Last 20 entries`,
	RunE: runConsoleShow,
}

var consoleSaveCmd = &cobra.Command{
	Use:   "save <path>",
	Short: "Save console logs to custom path",
	Long: `Saves console logs to a custom file path.

If path is a directory, auto-generates filename.
If path is a file, uses exact path.

Examples:
  console save ./logs/debug.json           # Save to file
  console save ./output/                   # Save to dir
  console save ./errors.json --type error --find "fetch"`,
	Args: cobra.ExactArgs(1),
	RunE: runConsoleSave,
}

func init() {
	// Universal flags on root command (inherited by default/show/save subcommands)
	consoleCmd.PersistentFlags().StringP("find", "f", "", "Search for text within log messages")
	consoleCmd.PersistentFlags().Bool("raw", false, "Skip formatting (return raw JSON)")

	// Console-specific filter flags
	consoleCmd.PersistentFlags().StringSlice("type", nil, "Filter by entry type (repeatable, CSV-supported)")
	consoleCmd.PersistentFlags().Int("head", 0, "Return first N entries")
	consoleCmd.PersistentFlags().Int("tail", 0, "Return last N entries")
	consoleCmd.PersistentFlags().String("range", "", "Return entries in range (format: START-END)")
	consoleCmd.MarkFlagsMutuallyExclusive("head", "tail", "range")

	// Add all subcommands
	consoleCmd.AddCommand(consoleShowCmd, consoleSaveCmd)

	rootCmd.AddCommand(consoleCmd)
}

// runConsoleDefault handles default behavior: save to temp directory
func runConsoleDefault(cmd *cobra.Command, args []string) error {
	// Validate that no arguments were provided (catches unknown subcommands)
	if len(args) > 0 {
		return outputError(fmt.Sprintf("unknown command %q for \"webctl console\"", args[0]))
	}

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Get console logs from daemon
	entries, err := getConsoleFromDaemon(cmd)
	if err != nil {
		return outputError(err.Error())
	}

	// Generate filename in temp directory
	outputPath, err := generateConsolePath()
	if err != nil {
		return outputError(err.Error())
	}

	// Write console logs to file
	if err := writeConsoleToFile(outputPath, entries); err != nil {
		return outputError(err.Error())
	}

	// Return JSON response
	if JSONOutput {
		return outputJSON(os.Stdout, map[string]any{
			"ok":   true,
			"path": outputPath,
		})
	}

	return format.FilePath(os.Stdout, outputPath)
}

// runConsoleShow handles show subcommand: output to stdout
func runConsoleShow(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Get console logs from daemon
	entries, err := getConsoleFromDaemon(cmd)
	if err != nil {
		return outputError(err.Error())
	}

	// JSON mode: output JSON
	if JSONOutput {
		result := map[string]any{
			"ok":      true,
			"logs":    entries,
			"count":   len(entries),
		}
		return outputJSON(os.Stdout, result)
	}

	// Check --raw flag
	raw, _ := cmd.Flags().GetBool("raw")
	if !raw && cmd.Parent() != nil {
		raw, _ = cmd.Parent().PersistentFlags().GetBool("raw")
	}

	if raw {
		// Raw mode: output as JSON array
		result := map[string]any{
			"ok":      true,
			"logs":    entries,
			"count":   len(entries),
		}
		return outputJSON(os.Stdout, result)
	}

	// Text mode: use text formatter
	return format.Console(os.Stdout, entries, format.NewOutputOptions(JSONOutput, NoColor))
}

// runConsoleSave handles save subcommand: save to custom path
func runConsoleSave(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	path := args[0]

	// Get console logs from daemon
	entries, err := getConsoleFromDaemon(cmd)
	if err != nil {
		return outputError(err.Error())
	}

	// Handle directory vs file path
	fileInfo, err := os.Stat(path)
	if err == nil && fileInfo.IsDir() {
		// Path is a directory - auto-generate filename
		filename := generateConsoleFilename()
		path = filepath.Join(path, filename)
	}

	// Write console logs to file
	if err := writeConsoleToFile(path, entries); err != nil {
		return outputError(err.Error())
	}

	// Return JSON response
	if JSONOutput {
		return outputJSON(os.Stdout, map[string]any{
			"ok":   true,
			"path": path,
		})
	}

	return format.FilePath(os.Stdout, path)
}

// getConsoleFromDaemon fetches console logs from daemon, applying filters
func getConsoleFromDaemon(cmd *cobra.Command) ([]ipc.ConsoleEntry, error) {
	// Try to get flags from command, falling back to parent for persistent flags
	find, _ := cmd.Flags().GetString("find")
	if find == "" && cmd.Parent() != nil {
		find, _ = cmd.Parent().PersistentFlags().GetString("find")
	}

	types, _ := cmd.Flags().GetStringSlice("type")
	if len(types) == 0 && cmd.Parent() != nil {
		types, _ = cmd.Parent().PersistentFlags().GetStringSlice("type")
	}

	head, _ := cmd.Flags().GetInt("head")
	if head == 0 && cmd.Parent() != nil {
		head, _ = cmd.Parent().PersistentFlags().GetInt("head")
	}

	tail, _ := cmd.Flags().GetInt("tail")
	if tail == 0 && cmd.Parent() != nil {
		tail, _ = cmd.Parent().PersistentFlags().GetInt("tail")
	}

	rangeStr, _ := cmd.Flags().GetString("range")
	if rangeStr == "" && cmd.Parent() != nil {
		rangeStr, _ = cmd.Parent().PersistentFlags().GetString("range")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return nil, err
	}
	defer exec.Close()

	// Execute console request
	resp, err := exec.Execute(ipc.Request{Cmd: "console"})
	if err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	// Parse console data
	var data ipc.ConsoleData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}

	entries := data.Entries

	// Apply type filter
	if len(types) > 0 {
		entries = filterConsoleByType(entries, types)
	}

	// Apply --find filter if specified
	if find != "" {
		entries = filterConsoleByText(entries, find)
		if len(entries) == 0 {
			return nil, fmt.Errorf("no matches found for '%s'", find)
		}
	}

	// Apply limiting (head/tail/range)
	entries, err = applyConsoleLimiting(entries, head, tail, rangeStr)
	if err != nil {
		return nil, err
	}

	return entries, nil
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

// filterConsoleByText filters entries to only include those containing the search text
func filterConsoleByText(entries []ipc.ConsoleEntry, searchText string) []ipc.ConsoleEntry {
	var matchedEntries []ipc.ConsoleEntry
	searchLower := strings.ToLower(searchText)

	for _, entry := range entries {
		if strings.Contains(strings.ToLower(entry.Text), searchLower) {
			matchedEntries = append(matchedEntries, entry)
		}
	}

	return matchedEntries
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

// writeConsoleToFile writes console entries to a file in JSON format, creating directories if needed
func writeConsoleToFile(path string, entries []ipc.ConsoleEntry) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Marshal entries to JSON
	data := map[string]any{
		"ok":      true,
		"logs":    entries,
		"count":   len(entries),
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal console logs: %v", err)
	}

	// Write to file
	if err := os.WriteFile(path, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write console logs: %v", err)
	}

	return nil
}

// generateConsolePath generates a full path in /tmp/webctl-console/
// using the pattern: YY-MM-DD-HHMMSS-console.json
func generateConsolePath() (string, error) {
	filename := generateConsoleFilename()
	return filepath.Join("/tmp/webctl-console", filename), nil
}

// generateConsoleFilename generates a filename using the pattern:
// YY-MM-DD-HHMMSS-console.json
func generateConsoleFilename() string {
	// Generate timestamp: YY-MM-DD-HHMMSS
	now := time.Now()
	timestamp := now.Format("06-01-02-150405")

	// Generate filename with fixed identifier "console"
	return fmt.Sprintf("%s-console.json", timestamp)
}
