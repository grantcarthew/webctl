package cli

import (
	"encoding/json"
	"errors"
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
	Short: "Extract console logs from current page (default: stdout)",
	Long: `Extracts console logs from the current page with flexible output modes.

Default behavior (no subcommand):
  Outputs console logs to stdout for piping or inspection

Subcommands:
  save [path]       Save console logs to file (temp dir if no path given)

Universal flags (work with all modes):
  --find, -f        Search for text within log messages
  --raw             Skip formatting (return raw JSON)
  --json            Output in JSON format (global flag)

Console-specific filter flags:
  --type TYPE       Filter by log type (log, warn, error, debug, info)
  --head N          Return first N entries
  --tail N          Return last N entries
  --range N-M       Return entries N through M (1-indexed, inclusive)

Examples:

Default mode (stdout):
  console                                  # All logs to stdout
  console --type error                     # Only errors to stdout
  console --find "undefined"               # Search and show matches
  console --tail 20                        # Last 20 entries

Save mode (file):
  console save                             # Save to temp with auto-filename
  console save ./logs/debug.json           # Save to custom file
  console save ./output/                   # Save to dir (auto-filename)
  console save --type error --tail 50

Response formats:
  Default:  [15:04:05] ERROR TypeError: undefined (to stdout)
  Save:     /tmp/webctl-console/25-12-28-143052-console.json

Error cases:
  - "No matches found" - find text not in logs
  - "daemon not running" - start daemon first with: webctl start`,
	RunE: runConsoleDefault,
}

var consoleSaveCmd = &cobra.Command{
	Use:   "save [path]",
	Short: "Save console logs to file",
	Long: `Saves console logs to a file.

Path conventions:
  (no path)         Save to /tmp/webctl-console/ with auto-generated filename
  ./logs.json       Save to exact file path
  ./output/         Save to directory with auto-generated filename (trailing slash required)

Examples:
  console save                             # Save to temp dir
  console save ./logs/debug.json           # Save to file
  console save ./output/                   # Save to dir (creates if needed)
  console save --type error --find "fetch"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runConsoleSave,
}

func init() {
	// Universal flags on root command (inherited by default/save subcommands)
	consoleCmd.PersistentFlags().StringP("find", "f", "", "Search for text within log messages")
	consoleCmd.PersistentFlags().Bool("raw", false, "Skip formatting (return raw JSON)")

	// Console-specific filter flags
	consoleCmd.PersistentFlags().StringSlice("type", nil, "Filter by entry type (repeatable, CSV-supported)")
	consoleCmd.PersistentFlags().Int("head", 0, "Return first N entries")
	consoleCmd.PersistentFlags().Int("tail", 0, "Return last N entries")
	consoleCmd.PersistentFlags().String("range", "", "Return entries N through M (1-indexed, inclusive)")
	// Note: MarkFlagsMutuallyExclusive doesn't work with PersistentFlags,
	// so we validate manually in getConsoleFromDaemon

	// Add all subcommands
	consoleCmd.AddCommand(consoleSaveCmd)

	rootCmd.AddCommand(consoleCmd)
}

// runConsoleDefault handles default behavior: output to stdout
func runConsoleDefault(cmd *cobra.Command, args []string) error {
	t := startTimer("console")
	defer t.log()

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
		if errors.Is(err, ErrNoMatches) {
			return outputNotice("No matches found")
		}
		if errors.Is(err, ErrNoEntriesInRange) {
			return outputNotice("No entries in range")
		}
		return outputError(err.Error())
	}

	// JSON mode: output JSON
	if JSONOutput {
		result := map[string]any{
			"ok":    true,
			"logs":  entries,
			"count": len(entries),
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
			"ok":    true,
			"logs":  entries,
			"count": len(entries),
		}
		return outputJSON(os.Stdout, result)
	}

	// Text mode: use text formatter
	return format.Console(os.Stdout, entries, format.NewOutputOptions(JSONOutput, NoColor))
}

// runConsoleSave handles save subcommand: save to file
func runConsoleSave(cmd *cobra.Command, args []string) error {
	t := startTimer("console save")
	defer t.log()

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Get console logs from daemon
	entries, err := getConsoleFromDaemon(cmd)
	if err != nil {
		if errors.Is(err, ErrNoMatches) {
			return outputNotice("No matches found")
		}
		if errors.Is(err, ErrNoEntriesInRange) {
			return outputNotice("No entries in range")
		}
		return outputError(err.Error())
	}

	var outputPath string

	if len(args) == 0 {
		// No path provided - save to temp directory
		outputPath, err = generateConsolePath()
		if err != nil {
			return outputError(err.Error())
		}
	} else {
		// Path provided
		path := args[0]

		// Check if path ends with separator (directory convention)
		if strings.HasSuffix(path, string(os.PathSeparator)) || strings.HasSuffix(path, "/") {
			// Path ends with separator - treat as directory, auto-generate filename
			filename := generateConsoleFilename()

			// Ensure directory exists
			if err := os.MkdirAll(path, 0755); err != nil {
				return outputError(fmt.Sprintf("failed to create directory: %v", err))
			}

			outputPath = filepath.Join(path, filename)
		} else {
			// No trailing slash - treat as file path
			outputPath = path
		}
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

// getConsoleFromDaemon fetches console logs from daemon, applying filters
func getConsoleFromDaemon(cmd *cobra.Command) ([]ipc.ConsoleEntry, error) {
	// Try to get flags from command's merged flags, then persistent flags,
	// then parent's persistent flags (for subcommands)
	find, _ := cmd.Flags().GetString("find")
	if find == "" {
		find, _ = cmd.PersistentFlags().GetString("find")
	}
	if find == "" && cmd.Parent() != nil {
		find, _ = cmd.Parent().PersistentFlags().GetString("find")
	}

	types, _ := cmd.Flags().GetStringSlice("type")
	if len(types) == 0 {
		types, _ = cmd.PersistentFlags().GetStringSlice("type")
	}
	if len(types) == 0 && cmd.Parent() != nil {
		types, _ = cmd.Parent().PersistentFlags().GetStringSlice("type")
	}

	head, _ := cmd.Flags().GetInt("head")
	if head == 0 {
		head, _ = cmd.PersistentFlags().GetInt("head")
	}
	if head == 0 && cmd.Parent() != nil {
		head, _ = cmd.Parent().PersistentFlags().GetInt("head")
	}

	tail, _ := cmd.Flags().GetInt("tail")
	if tail == 0 {
		tail, _ = cmd.PersistentFlags().GetInt("tail")
	}
	if tail == 0 && cmd.Parent() != nil {
		tail, _ = cmd.Parent().PersistentFlags().GetInt("tail")
	}

	rangeStr, _ := cmd.Flags().GetString("range")
	if rangeStr == "" {
		rangeStr, _ = cmd.PersistentFlags().GetString("range")
	}
	if rangeStr == "" && cmd.Parent() != nil {
		rangeStr, _ = cmd.Parent().PersistentFlags().GetString("range")
	}

	// Validate mutual exclusivity of head, tail, and range
	limitFlags := 0
	if head > 0 {
		limitFlags++
	}
	if tail > 0 {
		limitFlags++
	}
	if rangeStr != "" {
		limitFlags++
	}
	if limitFlags > 1 {
		return nil, fmt.Errorf("--head, --tail, and --range are mutually exclusive")
	}

	debugParam("find=%q types=%v head=%d tail=%d range=%q", find, types, head, tail, rangeStr)

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	debugRequest("console", "")
	ipcStart := time.Now()

	// Execute console request
	resp, err := exec.Execute(ipc.Request{Cmd: "console"})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

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
		beforeCount := len(entries)
		entries = filterConsoleByType(entries, types)
		debugFilter(fmt.Sprintf("--type %v", types), beforeCount, len(entries))
	}

	// Apply --find filter if specified
	if find != "" {
		beforeCount := len(entries)
		entries = filterConsoleByText(entries, find)
		debugFilter(fmt.Sprintf("--find %q", find), beforeCount, len(entries))
		if len(entries) == 0 {
			return nil, ErrNoMatches
		}
	}

	// Apply limiting (head/tail/range)
	entries, err = applyConsoleLimiting(entries, head, tail, rangeStr)
	if err != nil {
		return nil, err
	}

	// Check for empty range results
	if rangeStr != "" && len(entries) == 0 {
		return nil, ErrNoEntriesInRange
	}

	return entries, nil
}

// filterConsoleByType filters entries to only include those with matching types.
func filterConsoleByType(entries []ipc.ConsoleEntry, types []string) []ipc.ConsoleEntry {
	typeSet := make(map[string]bool)
	for _, t := range types {
		typeSet[ipc.NormalizeConsoleType(t)] = true
	}

	var filtered []ipc.ConsoleEntry
	for _, e := range entries {
		if typeSet[ipc.NormalizeConsoleType(e.Type)] {
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
			return nil, fmt.Errorf("invalid range format: use START-END (e.g., 1-10)")
		}
		start, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid range format: use START-END (e.g., 1-10)")
		}
		end, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid range format: use START-END (e.g., 1-10)")
		}

		// Convert from 1-indexed user input to 0-indexed slice indices.
		// User range is inclusive on both ends: --range 2-4 means entries 2, 3, 4.
		startIdx := start - 1
		endIdx := end

		// Clamp to valid bounds to avoid out-of-range errors
		if startIdx < 0 {
			startIdx = 0
		}
		if endIdx > len(entries) {
			endIdx = len(entries)
		}
		if startIdx >= endIdx || startIdx >= len(entries) {
			return []ipc.ConsoleEntry{}, nil
		}
		return entries[startIdx:endIdx], nil
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
		"ok":    true,
		"logs":  entries,
		"count": len(entries),
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal console logs: %v", err)
	}

	// Write to file
	if err := os.WriteFile(path, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write console logs: %v", err)
	}

	debugFile("wrote", path, len(jsonBytes))
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
