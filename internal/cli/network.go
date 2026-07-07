package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/grantcarthew/webctl/internal/cli/format"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var networkCmd = &cobra.Command{
	Use:   "network",
	Short: "Extract network requests from current page (default: stdout)",
	Long: `Extracts network requests from the current page with flexible output modes.

Default behavior (no subcommand):
  Lists network requests to stdout: one indexed line per entry plus the transport
  detail block, no bodies. Each line is prefixed with the entry's seq (its
  drill-down address). Use "network <n>" to see one entry's bodies.

Subcommands:
  save [path]       Save network requests to file (temp dir if no path given)

Drill-down:
  network <n>       Show the single entry with seq n, rendered with its bodies
                    (unbounded by default). Ignores the filter and range flags.
  network <n> --schema
                    Preview entry n's JSON response body as a key skeleton with
                    type-name leaves, without pulling the full body.

Detail dial (text only; ignored in JSON and by --schema):
  --detail summary  One line per entry (main line only).
  --detail standard Main line plus the transport detail block (remote, timing,
                    initiator). No bodies. This is the default.
  --detail full     Standard plus request and response bodies, bounded by
                    --max-body-size (default 102400 at this level).

Universal flags:
  --find, -f        Search for text within URLs and bodies (narrows the list;
                    the matched body is seen by drilling into the entry)
  --headers         Show request and response headers (standard and full levels)
  --json            Output in JSON format, always full fidelity (global flag)

Network-specific filter flags:
  --type            CDP resource type: xhr, fetch, document, script, stylesheet, image,
                    font, websocket, media, manifest, texttrack, eventsource, prefetch, other
  --method          HTTP method: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS
  --status          Status code or range: 200, 4xx, 5xx, 200-299
  --url             URL regex pattern (Go regexp syntax)
  --mime            MIME type: application/json, text/html, image/png
  --min-duration    Minimum request duration: 1s, 500ms, 100ms
  --min-size        Minimum response size in bytes
  --failed          Show only failed requests (network errors, CORS, etc.)
  --head N          Return first N entries (count over the seq-ordered list)
  --tail N          Return last N entries (count over the seq-ordered list)
  --range START-END Keep entries whose seq is in [START, END] inclusive

All filters are AND-combined. StringSlice flags support CSV (--status 4xx,5xx)
and repeatable (--status 4xx --status 5xx) syntax.

Examples:

List mode (stdout):
  network                                  # Indexed list, transport block, no bodies
  network --detail summary                 # One line per entry
  network --detail full                    # List with bodies
  network --status 4xx                     # Only 4xx
  network --find "api"                     # Narrow to entries matching "api"
  network --tail 20                        # Last 20 entries
  network --range 318-425                  # Entries with seq in [318, 425]

Drill-down mode (stdout):
  network 42                               # Entry 42 with its bodies
  network 42 --schema                      # JSON shape of entry 42's response body

Save mode (file):
  network save                             # Save to temp with auto-filename
  network save ./logs/requests.json        # Save to custom file
  network save ./output/                   # Save to dir (auto-filename)
  network save --status 5xx --tail 50

Response formats:
  List:     01 GET https://example.com 200 45ms xhr 3.4KB (to stdout)
            seq prefix, then the main line; resource type and size append when
            captured; a failed request shows FAILED plus its reason; the
            transport block follows on indented lines
  Drill:    the single entry with its request and response bodies
  Save:     /tmp/webctl-network/25-12-28-143052-123-network.json

Error cases:
  - "No matches found" - find text not in requests
  - "entry <n> not in buffer" - drill-down seq the active session does not hold
  - "daemon not running" - start daemon first with: webctl start`,
	// At most one positional argument: the bare-integer drill-down address. A
	// stray extra token is a usage error rather than a silently discarded arg.
	// `save` dispatches as a subcommand before this constraint applies.
	Args: cobra.MaximumNArgs(1),
	RunE: runNetworkDefault,
}

var networkSaveCmd = &cobra.Command{
	Use:   "save [path]",
	Short: "Save network requests to file",
	Long: `Saves network requests to a file.

If no path is provided, saves to temp directory with auto-generated filename.
If path is a directory, auto-generates filename.
If path is a file, uses exact path.

Examples:
  network save                             # Save to temp dir
  network save ./logs/requests.json        # Save to file
  network save ./output/                   # Save to dir
  network save --status 5xx --method POST`,
	Args: cobra.MaximumNArgs(1),
	RunE: runNetworkSave,
}

func init() {
	// Universal flags on root command (inherited by default/save subcommands)
	networkCmd.PersistentFlags().StringP("find", "f", "", "Search for text within URLs and bodies")

	// Network-specific filter flags
	networkCmd.PersistentFlags().StringSlice("type", nil, "Filter by CDP resource type (repeatable, CSV-supported)")
	networkCmd.PersistentFlags().StringSlice("method", nil, "Filter by HTTP method (repeatable, CSV-supported)")
	networkCmd.PersistentFlags().StringSlice("status", nil, "Filter by status code or range (repeatable, CSV-supported)")
	networkCmd.PersistentFlags().String("url", "", "Filter by URL regex pattern")
	networkCmd.PersistentFlags().StringSlice("mime", nil, "Filter by MIME type (repeatable, CSV-supported)")
	networkCmd.PersistentFlags().Duration("min-duration", 0, "Filter by minimum request duration")
	networkCmd.PersistentFlags().Int64("min-size", 0, "Filter by minimum response size in bytes")
	networkCmd.PersistentFlags().Bool("failed", false, "Show only failed requests")
	networkCmd.PersistentFlags().Bool("headers", false, "Show request and response headers (standard and full detail levels)")
	// Registered default is 0 so pflag omits a misleading "(default N)": the real
	// unset default is mode-dependent and resolved via Changed, not this value.
	networkCmd.PersistentFlags().Int("max-body-size", 0, "Max body size in bytes: 102400 for the --detail full text list, unlimited for JSON/drill-down/save, 0 suppresses, -1 unlimited")
	networkCmd.PersistentFlags().Int("head", 0, "Return first N entries (count over the seq-ordered list)")
	networkCmd.PersistentFlags().Int("tail", 0, "Return last N entries (count over the seq-ordered list)")
	networkCmd.PersistentFlags().String("range", "", "Keep entries whose seq is in [START, END] inclusive (format: START-END)")
	networkCmd.MarkFlagsMutuallyExclusive("head", "tail", "range")

	// Text-only flags for the default (list/drill-down) command. Local rather than
	// persistent so `save` (a full-fidelity JSON archive) does not inherit them.
	networkCmd.Flags().String("detail", "standard", "Text detail level: summary, standard, or full")
	networkCmd.Flags().Bool("schema", false, "Preview an entry's JSON response body as a key skeleton (requires an entry index)")

	// Add all subcommands
	networkCmd.AddCommand(networkSaveCmd)

	rootCmd.AddCommand(networkCmd)
}

// runNetworkDefault handles the default command: list all entries, drill into a
// single entry by seq (network <n>), or preview an entry's response-body schema
// (network <n> --schema).
func runNetworkDefault(cmd *cobra.Command, args []string) error {
	t := startTimer("network")
	defer t.log()

	schema, _ := cmd.Flags().GetBool("schema")

	// A bare integer positional argument is a drill-down address. A non-integer
	// argument keeps the unknown-command error; `save` is dispatched by cobra as a
	// subcommand and never reaches here. Presence is tracked separately from the
	// value so a negative address is not mistaken for "no argument".
	var drillSeq int
	hasDrill := false
	if len(args) > 0 {
		n, err := strconv.Atoi(args[0])
		if err != nil {
			return outputError(fmt.Sprintf("unknown command %q for \"webctl network\"", args[0]))
		}
		drillSeq = n
		hasDrill = true
	}

	// --schema is a drill-down preview; it requires an entry index.
	if schema && !hasDrill {
		return outputError("network --schema requires an entry index (for example: network 42 --schema)")
	}

	// Validate --detail up front so a malformed value is a deterministic usage
	// error in every mode, before any daemon round-trip. The resolved level only
	// shapes the text list below: drill-down forces full and JSON ignores it, but
	// a bad value is still rejected there rather than silently accepted.
	detail, err := resolveDetailLevel(cmd)
	if err != nil {
		return outputError(err.Error())
	}

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	if hasDrill {
		return runNetworkDrilldown(cmd, drillSeq, schema)
	}

	// List mode. Fetch, filter, and limit the active session's entries.
	entries, err := getNetworkFromDaemon(cmd)
	if err != nil {
		if errors.Is(err, ErrNoMatches) {
			return outputNotice("No matches found")
		}
		return outputError(err.Error())
	}

	// JSON is always full fidelity: unlimited bodies unless --max-body-size is set.
	if JSONOutput {
		return outputNetworkJSON(entries, resolveMaxBodySize(cmd, ipc.MaxBodySizeUnlimited))
	}

	// Bodies render only at the full level, where the unset default is 102400.
	// The formatter prints the already-bounded text and does not truncate itself.
	if detail == format.DetailFull {
		applyBodyTruncation(entries, resolveMaxBodySize(cmd, ipc.DefaultMaxBodySize))
	}

	opts := format.NewOutputOptions(JSONOutput, NoColor)
	opts.ShowHeaders = resolveHeadersFlag(cmd)
	opts.Detail = detail
	return format.Network(os.Stdout, entries, opts)
}

// runNetworkDrilldown resolves a single entry by exact seq membership over the
// active session's full unfiltered set and renders it (or its schema). It ignores
// the filter and head/tail/range flags so a live entry is never hidden by a
// narrowing flag, and derives its miss-error bounds from that same set.
func runNetworkDrilldown(cmd *cobra.Command, n int, schema bool) error {
	entries, err := fetchNetworkEntries()
	if err != nil {
		return outputError(err.Error())
	}

	entry, found := findNetworkEntryBySeq(entries, n)
	if !found {
		return outputError(networkDrilldownMissMessage(n, entries))
	}

	if schema {
		return outputNetworkSchema(*entry)
	}

	// The payload view is complete by default: bodies are unbounded unless the
	// caller sets an explicit --max-body-size cap.
	maxBodySize := resolveMaxBodySize(cmd, ipc.MaxBodySizeUnlimited)
	single := []ipc.NetworkEntry{*entry}

	if JSONOutput {
		return outputNetworkJSON(single, maxBodySize)
	}

	// Drilling in is the explicit request to see the payload, so bodies render
	// regardless of --detail.
	applyBodyTruncation(single, maxBodySize)
	opts := format.NewOutputOptions(JSONOutput, NoColor)
	opts.ShowHeaders = resolveHeadersFlag(cmd)
	opts.Detail = format.DetailFull
	return format.Network(os.Stdout, single, opts)
}

// findNetworkEntryBySeq returns the entry whose seq exactly equals n. The held
// seqs may be sparse, so this is a membership test, not a range check: n falling
// between the lowest and highest held seq does not imply it is present.
func findNetworkEntryBySeq(entries []ipc.NetworkEntry, n int) (*ipc.NetworkEntry, bool) {
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

// networkSeqBounds returns the lowest and highest seq held in entries. ok is
// false when entries is empty, meaning there is no bound to name.
func networkSeqBounds(entries []ipc.NetworkEntry) (lo, hi uint64, ok bool) {
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

// networkDrilldownMissMessage builds the eviction-aware error for a drill-down
// to a seq the active session does not hold. The named bounds are orientation
// only; they do not promise every value between them is present.
func networkDrilldownMissMessage(n int, entries []ipc.NetworkEntry) string {
	lo, hi, ok := networkSeqBounds(entries)
	if !ok {
		return fmt.Sprintf("entry %d not in buffer (buffer empty)", n)
	}
	return fmt.Sprintf("entry %d not in buffer (holds seq %d-%d; run network to list)", n, lo, hi)
}

// resolveDetailLevel reads and validates the --detail flag. It is a local flag on
// the default command, so no parent fallback is needed.
func resolveDetailLevel(cmd *cobra.Command) (format.DetailLevel, error) {
	detail, _ := cmd.Flags().GetString("detail")
	switch detail {
	case "summary":
		return format.DetailSummary, nil
	case "standard", "":
		return format.DetailStandard, nil
	case "full":
		return format.DetailFull, nil
	default:
		return format.DetailStandard, fmt.Errorf("invalid --detail %q: use summary, standard, or full", detail)
	}
}

// runNetworkSave handles save subcommand: save to file
func runNetworkSave(cmd *cobra.Command, args []string) error {
	return runSave(cmd, args, saveSpec{
		timerLabel: "network save",
		tempDir:    "/tmp/webctl-network",
		ext:        "json",
		produce:    networkSaveContent,
		identifier: fixedIdentifier("network"),
	})
}

// networkSaveContent produces the network save-file payload: the JSON envelope
// with per-entry body truncation applied, matching the network JSON output.
func networkSaveContent(cmd *cobra.Command) (string, error) {
	entries, err := getNetworkFromDaemon(cmd)
	if err != nil {
		return "", err
	}

	// A saved file is a full-fidelity archive, so bodies are unbounded unless the
	// caller sets an explicit --max-body-size cap.
	applyBodyTruncation(entries, resolveMaxBodySize(cmd, ipc.MaxBodySizeUnlimited))

	return marshalSaveEnvelope(map[string]any{
		"ok":      true,
		"entries": entries,
		"count":   len(entries),
	})
}

// resolveMaxBodySize reads the --max-body-size flag, falling back to the parent
// command's persistent flag and finally defaultWhenUnset. It distinguishes an
// unset flag from an explicit value via Changed, so a deliberate --max-body-size 0
// (suppress) or -1 (unlimited) is honoured rather than coalesced to the default.
// The unset default is mode-dependent (102400 for the --detail full text list,
// unlimited for JSON, drill-down, and save), so the caller supplies it.
func resolveMaxBodySize(cmd *cobra.Command, defaultWhenUnset int) int {
	if cmd.Flags().Changed("max-body-size") {
		maxBodySize, _ := cmd.Flags().GetInt("max-body-size")
		return maxBodySize
	}
	if cmd.Parent() != nil && cmd.Parent().PersistentFlags().Changed("max-body-size") {
		maxBodySize, _ := cmd.Parent().PersistentFlags().GetInt("max-body-size")
		return maxBodySize
	}
	return defaultWhenUnset
}

// resolveHeadersFlag reads the --headers flag, falling back to the parent
// command's persistent flag so the default and save subcommands agree.
func resolveHeadersFlag(cmd *cobra.Command) bool {
	headers, _ := cmd.Flags().GetBool("headers")
	if !headers && cmd.Parent() != nil {
		headers, _ = cmd.Parent().PersistentFlags().GetBool("headers")
	}
	return headers
}

// fetchNetworkEntries returns the active session's full unfiltered entry set from
// the daemon, in buffer order. Both the filtered list path and the unfiltered
// drill-down path build on it, so drill-down addresses the same scope the list
// derives its bounds from.
func fetchNetworkEntries() ([]ipc.NetworkEntry, error) {
	exec, err := execFactory.NewExecutor()
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	debugRequest("network", "")
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{Cmd: "network"})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	var data ipc.NetworkData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}
	return data.Entries, nil
}

// getNetworkFromDaemon fetches network entries from daemon, applying filters
func getNetworkFromDaemon(cmd *cobra.Command) ([]ipc.NetworkEntry, error) {
	// Try to get flags from command, falling back to parent for persistent flags
	find, _ := cmd.Flags().GetString("find")
	if find == "" && cmd.Parent() != nil {
		find, _ = cmd.Parent().PersistentFlags().GetString("find")
	}

	types, _ := cmd.Flags().GetStringSlice("type")
	if len(types) == 0 && cmd.Parent() != nil {
		types, _ = cmd.Parent().PersistentFlags().GetStringSlice("type")
	}

	methods, _ := cmd.Flags().GetStringSlice("method")
	if len(methods) == 0 && cmd.Parent() != nil {
		methods, _ = cmd.Parent().PersistentFlags().GetStringSlice("method")
	}

	statuses, _ := cmd.Flags().GetStringSlice("status")
	if len(statuses) == 0 && cmd.Parent() != nil {
		statuses, _ = cmd.Parent().PersistentFlags().GetStringSlice("status")
	}

	urlPattern, _ := cmd.Flags().GetString("url")
	if urlPattern == "" && cmd.Parent() != nil {
		urlPattern, _ = cmd.Parent().PersistentFlags().GetString("url")
	}

	mimes, _ := cmd.Flags().GetStringSlice("mime")
	if len(mimes) == 0 && cmd.Parent() != nil {
		mimes, _ = cmd.Parent().PersistentFlags().GetStringSlice("mime")
	}

	minDuration, _ := cmd.Flags().GetDuration("min-duration")
	if minDuration == 0 && cmd.Parent() != nil {
		minDuration, _ = cmd.Parent().PersistentFlags().GetDuration("min-duration")
	}

	minSize, _ := cmd.Flags().GetInt64("min-size")
	if minSize == 0 && cmd.Parent() != nil {
		minSize, _ = cmd.Parent().PersistentFlags().GetInt64("min-size")
	}

	failed, _ := cmd.Flags().GetBool("failed")
	if !failed && cmd.Parent() != nil {
		failed, _ = cmd.Parent().PersistentFlags().GetBool("failed")
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

	// Validate URL regex if provided
	var urlRegex *regexp.Regexp
	if urlPattern != "" {
		var err error
		urlRegex, err = regexp.Compile(urlPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid URL pattern: %v", err)
		}
	}

	// Parse status patterns
	statusMatchers, err := parseStatusPatterns(statuses)
	if err != nil {
		return nil, err
	}

	debugParam("find=%q types=%v methods=%v statuses=%v urlPattern=%q failed=%v", find, types, methods, statuses, urlPattern, failed)

	entries, err := fetchNetworkEntries()
	if err != nil {
		return nil, err
	}

	// Build filter options
	filterOpts := networkFilterOptions{
		types:       types,
		methods:     methods,
		mimes:       mimes,
		minDuration: minDuration,
		minSize:     minSize,
		failed:      failed,
	}

	// Apply filters
	beforeCount := len(entries)
	entries = filterNetworkEntries(entries, urlRegex, statusMatchers, filterOpts)
	if len(entries) != beforeCount {
		debugFilter("network filters", beforeCount, len(entries))
	}

	// Apply --find filter if specified
	if find != "" {
		beforeCount := len(entries)
		entries = filterNetworkByText(entries, find)
		debugFilter(fmt.Sprintf("--find %q", find), beforeCount, len(entries))
		if len(entries) == 0 {
			return nil, ErrNoMatches
		}
	}

	// Apply limiting (head/tail/range)
	entries, err = applyNetworkLimiting(entries, head, tail, rangeStr)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// filterNetworkByText filters entries to only include those containing the search text in URL or body
func filterNetworkByText(entries []ipc.NetworkEntry, searchText string) []ipc.NetworkEntry {
	var matchedEntries []ipc.NetworkEntry
	searchLower := strings.ToLower(searchText)

	for _, entry := range entries {
		// Search in URL
		if strings.Contains(strings.ToLower(entry.URL), searchLower) {
			matchedEntries = append(matchedEntries, entry)
			continue
		}
		// Search in request body
		if strings.Contains(strings.ToLower(entry.RequestBody), searchLower) {
			matchedEntries = append(matchedEntries, entry)
			continue
		}
		// Search in response body
		if strings.Contains(strings.ToLower(entry.ResponseBody), searchLower) {
			matchedEntries = append(matchedEntries, entry)
			continue
		}
	}

	return matchedEntries
}

// statusMatcher represents a parsed status pattern.
type statusMatcher struct {
	exact      int  // Exact match (e.g., 200)
	rangeStart int  // Range start (e.g., 200 for 200-299)
	rangeEnd   int  // Range end (e.g., 299 for 200-299)
	isRange    bool // True if this is a range pattern
	isWildcard bool // True if this is a wildcard (e.g., 4xx)
}

// matches returns true if the given status code matches this pattern.
func (m statusMatcher) matches(status int) bool {
	if m.isRange {
		return status >= m.rangeStart && status <= m.rangeEnd
	}
	if m.isWildcard {
		return status >= m.rangeStart && status <= m.rangeEnd
	}
	return status == m.exact
}

// parseStatusPatterns parses status filter patterns into matchers.
func parseStatusPatterns(patterns []string) ([]statusMatcher, error) {
	var matchers []statusMatcher
	for _, p := range patterns {
		p = strings.TrimSpace(strings.ToLower(p))
		if p == "" {
			continue
		}

		// Check for wildcard pattern (e.g., 4xx, 5xx)
		if len(p) == 3 && p[1] == 'x' && p[2] == 'x' {
			digit, err := strconv.Atoi(string(p[0]))
			if err != nil || digit < 1 || digit > 5 {
				return nil, fmt.Errorf("invalid status pattern: %s", p)
			}
			matchers = append(matchers, statusMatcher{
				rangeStart: digit * 100,
				rangeEnd:   digit*100 + 99,
				isWildcard: true,
			})
			continue
		}

		// Check for range pattern (e.g., 200-299)
		if strings.Contains(p, "-") {
			parts := strings.Split(p, "-")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid status pattern: %s", p)
			}
			start, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid status pattern: %s", p)
			}
			end, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid status pattern: %s", p)
			}
			matchers = append(matchers, statusMatcher{
				rangeStart: start,
				rangeEnd:   end,
				isRange:    true,
			})
			continue
		}

		// Exact match
		exact, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid status pattern: %s", p)
		}
		matchers = append(matchers, statusMatcher{exact: exact})
	}
	return matchers, nil
}

// networkFilterOptions holds filter parameters for network entries.
type networkFilterOptions struct {
	types       []string
	methods     []string
	mimes       []string
	minDuration time.Duration
	minSize     int64
	failed      bool
}

// filterNetworkEntries applies all network filters.
func filterNetworkEntries(entries []ipc.NetworkEntry, urlRegex *regexp.Regexp, statusMatchers []statusMatcher, opts networkFilterOptions) []ipc.NetworkEntry {
	if len(opts.types) == 0 && len(opts.methods) == 0 && len(statusMatchers) == 0 &&
		urlRegex == nil && len(opts.mimes) == 0 && opts.minDuration == 0 &&
		opts.minSize == 0 && !opts.failed {
		return entries
	}

	var filtered []ipc.NetworkEntry
	for _, e := range entries {
		if !matchesNetworkFilters(e, urlRegex, statusMatchers, opts) {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}

// matchesNetworkFilters returns true if entry matches all specified filters.
func matchesNetworkFilters(e ipc.NetworkEntry, urlRegex *regexp.Regexp, statusMatchers []statusMatcher, opts networkFilterOptions) bool {
	// Type filter
	if len(opts.types) > 0 && !matchesStringSlice(e.Type, opts.types) {
		return false
	}

	// Method filter
	if len(opts.methods) > 0 && !matchesStringSlice(e.Method, opts.methods) {
		return false
	}

	// Status filter
	if len(statusMatchers) > 0 {
		matched := false
		for _, m := range statusMatchers {
			if m.matches(e.Status) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// URL regex filter
	if urlRegex != nil && !urlRegex.MatchString(e.URL) {
		return false
	}

	// MIME filter
	if len(opts.mimes) > 0 && !matchesStringSlice(e.MimeType, opts.mimes) {
		return false
	}

	// Min duration filter
	if opts.minDuration > 0 && e.Duration < opts.minDuration.Seconds() {
		return false
	}

	// Min size filter
	if opts.minSize > 0 && e.Size < opts.minSize {
		return false
	}

	// Failed filter
	if opts.failed && !e.Failed {
		return false
	}

	return true
}

// matchesStringSlice returns true if value matches any item in slice (case-insensitive).
func matchesStringSlice(value string, slice []string) bool {
	valueLower := strings.ToLower(value)
	for _, s := range slice {
		if strings.ToLower(s) == valueLower {
			return true
		}
	}
	return false
}

// applyNetworkLimiting applies head, tail, or range limiting to entries.
func applyNetworkLimiting(entries []ipc.NetworkEntry, head, tail int, rangeStr string) ([]ipc.NetworkEntry, error) {
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
		// Inclusive seq membership, AND-combined with the other filters. The held
		// seqs are sparse, so the endpoints need not be present; return whatever
		// held seqs fall inside [start, end], empty when none do.
		var matched []ipc.NetworkEntry
		for _, e := range entries {
			if e.Seq >= start && e.Seq <= end {
				matched = append(matched, e)
			}
		}
		return matched, nil
	}

	return entries, nil
}

// parseSeqRange parses a "START-END" range into inclusive seq bounds.
func parseSeqRange(rangeStr string) (start, end uint64, err error) {
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range format: use START-END (e.g., 318-425)")
	}
	start, err = strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid range format: use START-END (e.g., 318-425)")
	}
	end, err = strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid range format: use START-END (e.g., 318-425)")
	}
	return start, end, nil
}

// truncateBody cuts body to at most maxBytes bytes on a UTF-8 rune boundary
// (the longest valid prefix not exceeding maxBytes, so a multibyte rune is never
// split) and reports whether it truncated.
func truncateBody(body string, maxBytes int) (string, bool) {
	if maxBytes < 0 {
		maxBytes = 0
	}
	if len(body) <= maxBytes {
		return body, false
	}
	end := maxBytes
	for end > 0 && !utf8.RuneStart(body[end]) {
		end--
	}
	return body[:end], true
}

// applyBodyTruncation bounds the request and response body of every entry to
// maxBodySize, setting each entry's own truncation flag. Shared by the JSON,
// save, and text output paths so the byte-budget logic lives in one place. A
// negative maxBodySize (the unlimited sentinel) skips truncation entirely,
// leaving bodies at full fidelity; 0 suppresses all body content.
func applyBodyTruncation(entries []ipc.NetworkEntry, maxBodySize int) {
	if maxBodySize < 0 {
		return
	}
	for i := range entries {
		if truncated, did := truncateBody(entries[i].RequestBody, maxBodySize); did {
			entries[i].RequestBody = truncated
			entries[i].RequestBodyTruncated = true
		}
		if truncated, did := truncateBody(entries[i].ResponseBody, maxBodySize); did {
			entries[i].ResponseBody = truncated
			entries[i].ResponseBodyTruncated = true
		}
	}
}

// outputNetworkJSON outputs entries in JSON format.
func outputNetworkJSON(entries []ipc.NetworkEntry, maxBodySize int) error {
	applyBodyTruncation(entries, maxBodySize)

	result := map[string]any{
		"ok":      true,
		"entries": entries,
		"count":   len(entries),
	}
	return outputJSON(os.Stdout, result)
}

// outputNetworkSchema emits a token-efficient key skeleton of an entry's JSON
// response body. It reads the full stored body (a truncated body is not
// parseable JSON) and wraps the result in the standard envelope on stdout with
// exit 0. Both a parsed body and a non-JSON body share one envelope, one stream,
// and one exit code so a single parser branches on the schema field: a null
// schema means see the notice.
func outputNetworkSchema(entry ipc.NetworkEntry) error {
	var parsed any
	if strings.TrimSpace(entry.ResponseBody) == "" || json.Unmarshal([]byte(entry.ResponseBody), &parsed) != nil {
		notice := "response body is not JSON"
		if entry.MimeType != "" {
			notice = fmt.Sprintf("response body is not JSON (%s)", entry.MimeType)
		}
		return outputJSON(os.Stdout, map[string]any{
			"ok":     true,
			"schema": nil,
			"notice": notice,
		})
	}

	return outputJSON(os.Stdout, map[string]any{
		"ok":     true,
		"schema": buildSchema(parsed),
	})
}

// buildSchema mirrors a parsed JSON value's structure, replacing each leaf with
// its JSON type name. Objects keep their keys; arrays collapse to a single
// representative element that unions the shapes of every element, so a
// heterogeneous array does not hide fields.
func buildSchema(v any) any {
	switch t := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(t))
		for k, val := range t {
			m[k] = buildSchema(val)
		}
		return m
	case []any:
		if len(t) == 0 {
			return []any{}
		}
		schemas := make([]any, len(t))
		for i, el := range t {
			schemas[i] = buildSchema(el)
		}
		return []any{mergeSchemas(schemas)}
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

// mergeSchemas unions a list of schema values into one representative. Object
// schemas merge key by key (recursively), array schemas merge their inner
// representatives, and a scalar falls through. Composite shapes win over scalars
// when a heterogeneous array mixes them, so no object key is lost.
func mergeSchemas(schemas []any) any {
	var maps []map[string]any
	var arrays [][]any
	var scalar any
	for _, s := range schemas {
		switch t := s.(type) {
		case map[string]any:
			maps = append(maps, t)
		case []any:
			arrays = append(arrays, t)
		default:
			if scalar == nil {
				scalar = t
			}
		}
	}

	if len(maps) > 0 {
		merged := make(map[string]any)
		for _, m := range maps {
			for k, v := range m {
				if existing, ok := merged[k]; ok {
					merged[k] = mergeSchemas([]any{existing, v})
				} else {
					merged[k] = v
				}
			}
		}
		return merged
	}

	if len(arrays) > 0 {
		var inner []any
		for _, a := range arrays {
			inner = append(inner, a...)
		}
		if len(inner) == 0 {
			return []any{}
		}
		return []any{mergeSchemas(inner)}
	}

	return scalar
}
