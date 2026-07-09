package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
  Lists console entries to stdout: one indexed summary line per entry. Each line
  is prefixed with the entry's seq (its drill-down address). Use "console <n>"
  to see one entry's full stack, arguments, and exception or Log-domain detail.

Subcommands:
  save [path]       Save console logs to file (temp dir if no path given)

Universal flags:
  --find, -f        Search for text within log messages (narrows the list)
  --json            Output in JSON format, always full fidelity (global flag)

Console-specific filter flags (list and save; ignored by drill-down):
  --type TYPE       Filter by log type (log, warn, error, debug, info)
  --head N          Return first N entries (count over the seq-ordered list)
  --tail N          Return last N entries (count over the seq-ordered list)
  --range START-END Keep entries whose seq is in [START, END] inclusive

Drill-down:
  console <n>       Show the single entry with seq n, rendered in full: the
                    complete stack, all arguments, and any exception or
                    Log-domain detail. Ignores the filter and range flags.

Examples:

Default mode (stdout):
  console                                  # Indexed list, one line per entry
  console --type error                     # Only errors to stdout
  console --find "undefined"               # Search and show matches
  console --tail 20                        # Last 20 entries
  console --range 318-425                  # Entries with seq in [318, 425]

Drill-down mode (stdout):
  console 42                               # Entry 42, rendered in full

Save mode (file):
  console save                             # Save to temp with auto-filename
  console save ./logs/debug.json           # Save to custom file
  console save ./output/                   # Save to dir (auto-filename)
  console save --type error --tail 50

Response formats:
  List:     03 [15:04:05] ERROR app.js:42:10 TypeError: undefined (to stdout)
  Drill:    the single entry with its full stack and arguments
  Save:     /tmp/webctl-console/25-12-28-143052-123-console.json

Error cases:
  - "No matches found" - find text not in logs
  - "entry <n> not in buffer" - drill-down seq the active session does not hold
  - "daemon not running" - start daemon first with: webctl start`,
	// At most one positional argument: the bare-integer drill-down address. A
	// stray extra token is a usage error rather than a silently discarded arg.
	// `save` dispatches as a subcommand before this constraint applies.
	Args: cobra.MaximumNArgs(1),
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

	// Console-specific filter flags
	consoleCmd.PersistentFlags().StringSlice("type", nil, "Filter by entry type (repeatable, CSV-supported)")
	consoleCmd.PersistentFlags().Int("head", 0, "Return first N entries (count over the seq-ordered list)")
	consoleCmd.PersistentFlags().Int("tail", 0, "Return last N entries (count over the seq-ordered list)")
	consoleCmd.PersistentFlags().String("range", "", "Keep entries whose seq is in [START, END] inclusive (format: START-END)")
	// Note: MarkFlagsMutuallyExclusive doesn't work with PersistentFlags,
	// so we validate manually in getConsoleFromDaemon

	// Add all subcommands
	consoleCmd.AddCommand(consoleSaveCmd)

	rootCmd.AddCommand(consoleCmd)
}

// runConsoleDefault handles the default command: list the active session's
// entries as an indexed summary, or drill into a single entry by seq
// (console <n>).
func runConsoleDefault(cmd *cobra.Command, args []string) error {
	t := startTimer("console")
	defer t.log()

	// A bare integer positional argument is a drill-down address. A non-integer
	// argument keeps the unknown-command error; `save` is dispatched by cobra as a
	// subcommand and never reaches here. Presence is tracked separately from the
	// value so a negative address is not mistaken for "no argument".
	var drillSeq int
	hasDrill := false
	if len(args) > 0 {
		n, err := strconv.Atoi(args[0])
		if err != nil {
			return outputError(fmt.Sprintf("unknown command %q for \"webctl console\"", args[0]))
		}
		drillSeq = n
		hasDrill = true
	}

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	if hasDrill {
		return runConsoleDrilldown(drillSeq)
	}

	// List mode. Fetch, filter, and limit the active session's entries.
	entries, err := getConsoleFromDaemon(cmd)
	if err != nil {
		if errors.Is(err, ErrNoMatches) {
			return outputNotice("No matches found")
		}
		return outputError(err.Error())
	}

	if JSONOutput {
		return outputConsoleJSON(entries)
	}

	return format.Console(os.Stdout, entries, format.NewOutputOptions(JSONOutput, NoColor))
}

// runConsoleDrilldown resolves a single entry by exact seq membership over the
// active session's full unfiltered set and renders it. It ignores the filter and
// head/tail/range flags so a live entry is never hidden by a narrowing flag, and
// derives its miss-error bounds from that same set.
func runConsoleDrilldown(n int) error {
	entries, err := fetchConsoleEntries()
	if err != nil {
		return outputError(err.Error())
	}

	entry, found := findConsoleEntryBySeq(entries, n)
	if !found {
		return outputError(consoleDrilldownMissMessage(n, entries))
	}

	if JSONOutput {
		return outputConsoleJSON([]ipc.ConsoleEntry{*entry})
	}

	return format.ConsoleDetail(os.Stdout, *entry, format.NewOutputOptions(JSONOutput, NoColor))
}

// consoleEntriesOrEmpty returns entries, or a non-nil empty slice when entries
// is nil, so JSON encodes "entries":[] rather than null for every empty path
// (type/find filters, empty buffer, empty range).
func consoleEntriesOrEmpty(entries []ipc.ConsoleEntry) []ipc.ConsoleEntry {
	if entries == nil {
		return []ipc.ConsoleEntry{}
	}
	return entries
}

// outputConsoleJSON writes entries in the standard console JSON envelope: an
// "entries" array with a "count", matching the network command and the
// underlying ConsoleData shape. Drill-down passes a single-element slice.
func outputConsoleJSON(entries []ipc.ConsoleEntry) error {
	entries = consoleEntriesOrEmpty(entries)
	return outputJSON(os.Stdout, map[string]any{
		"ok":      true,
		"entries": entries,
		"count":   len(entries),
	})
}

// findConsoleEntryBySeq returns the entry whose seq exactly equals n. The held
// seqs may be sparse, so this is a membership test, not a range check: n falling
// between the lowest and highest held seq does not imply it is present.
func findConsoleEntryBySeq(entries []ipc.ConsoleEntry, n int) (*ipc.ConsoleEntry, bool) {
	if n < 0 {
		return nil, false
	}
	target := uint64(n)
	for i := range entries {
		if entries[i].Seq == target {
			return &entries[i], true
		}
	}
	return nil, false
}

// consoleSeqBounds returns the lowest and highest seq held in entries. ok is
// false when entries is empty, meaning there is no bound to name.
func consoleSeqBounds(entries []ipc.ConsoleEntry) (lo, hi uint64, ok bool) {
	if len(entries) == 0 {
		return 0, 0, false
	}
	lo, hi = entries[0].Seq, entries[0].Seq
	for _, e := range entries {
		if e.Seq < lo {
			lo = e.Seq
		}
		if e.Seq > hi {
			hi = e.Seq
		}
	}
	return lo, hi, true
}

// consoleDrilldownMissMessage builds the eviction-aware error for a drill-down to
// a seq the active session does not hold. The named bounds are orientation only;
// they do not promise every value between them is present.
func consoleDrilldownMissMessage(n int, entries []ipc.ConsoleEntry) string {
	lo, hi, ok := consoleSeqBounds(entries)
	if !ok {
		return fmt.Sprintf("entry %d not in buffer (buffer empty)", n)
	}
	return fmt.Sprintf("entry %d not in buffer (holds seq %d-%d; run console to list)", n, lo, hi)
}

// runConsoleSave handles save subcommand: save to file
func runConsoleSave(cmd *cobra.Command, args []string) error {
	return runSave(cmd, args, saveSpec{
		timerLabel: "console save",
		tempDir:    "/tmp/webctl-console",
		ext:        "json",
		produce:    consoleSaveContent,
		identifier: fixedIdentifier("console"),
	})
}

// consoleSaveContent produces the console save-file payload: the JSON envelope
// written to disk, identical in shape to the console JSON output.
func consoleSaveContent(cmd *cobra.Command) (string, error) {
	entries, err := getConsoleFromDaemon(cmd)
	if err != nil {
		return "", err
	}
	entries = consoleEntriesOrEmpty(entries)
	return marshalSaveEnvelope(map[string]any{
		"ok":      true,
		"entries": entries,
		"count":   len(entries),
	})
}

// fetchConsoleEntries returns the active session's full unfiltered entry set from
// the daemon, in buffer order. Both the filtered list path and the unfiltered
// drill-down path build on it, so drill-down addresses the same scope the list
// derives its bounds from.
func fetchConsoleEntries() ([]ipc.ConsoleEntry, error) {
	exec, err := execFactory.NewExecutor()
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	debugRequest("console", "")
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{Cmd: "console"})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	var data ipc.ConsoleData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}
	return data.Entries, nil
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

	entries, err := fetchConsoleEntries()
	if err != nil {
		return nil, err
	}

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

	// Apply limiting (head/tail/range). An empty seq range is a routine result,
	// not an error: it returns an empty list with exit 0, matching network.
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
		start, end, err := parseSeqRange(rangeStr)
		if err != nil {
			return nil, err
		}
		// Inclusive seq membership, matching the network command. The held seqs are
		// sparse, so the endpoints need not be present; return whatever held seqs
		// fall inside [start, end], empty when none do. Use a non-nil empty slice so
		// JSON encodes "entries":[] rather than null when nothing matches.
		matched := make([]ipc.ConsoleEntry, 0)
		for _, e := range entries {
			if e.Seq >= start && e.Seq <= end {
				matched = append(matched, e)
			}
		}
		return matched, nil
	}

	return entries, nil
}
