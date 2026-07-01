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
  Outputs network requests to stdout for piping or inspection

Subcommands:
  save [path]       Save network requests to file (temp dir if no path given)

Universal flags (work with all modes):
  --find, -f        Search for text within URLs and bodies
  --headers         Show request and response headers (text mode)
  --json            Output in JSON format (global flag)

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
  --head N          Return first N entries
  --tail N          Return last N entries
  --range N-M       Return entries N through M

All filters are AND-combined. StringSlice flags support CSV (--status 4xx,5xx)
and repeatable (--status 4xx --status 5xx) syntax.

Examples:

Default mode (stdout):
  network                                  # All requests to stdout
  network --status 4xx                     # Only 4xx to stdout
  network --find "api"                     # Search and show matches
  network --tail 20                        # Last 20 entries

Save mode (file):
  network save                             # Save to temp with auto-filename
  network save ./logs/requests.json        # Save to custom file
  network save ./output/                   # Save to dir (auto-filename)
  network save --status 5xx --tail 50

Response formats:
  Default:  GET https://example.com 200 45ms xhr 3.4KB (to stdout)
            resource type and size append when captured; a failed request shows
            FAILED plus its reason instead of a status; --headers adds headers
  Save:     /tmp/webctl-network/25-12-28-143052-123-network.json

Error cases:
  - "No matches found" - find text not in requests
  - "daemon not running" - start daemon first with: webctl start`,
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
	networkCmd.PersistentFlags().Bool("headers", false, "Show request and response headers (text mode)")
	networkCmd.PersistentFlags().Int("max-body-size", ipc.DefaultMaxBodySize, "Maximum body size in bytes before truncation (default 100KB; 0 suppresses all body content)")
	networkCmd.PersistentFlags().Int("head", 0, "Return first N entries")
	networkCmd.PersistentFlags().Int("tail", 0, "Return last N entries")
	networkCmd.PersistentFlags().String("range", "", "Return entries in range (format: START-END)")
	networkCmd.MarkFlagsMutuallyExclusive("head", "tail", "range")

	// Add all subcommands
	networkCmd.AddCommand(networkSaveCmd)

	rootCmd.AddCommand(networkCmd)
}

// runNetworkDefault handles default behavior: output to stdout
func runNetworkDefault(cmd *cobra.Command, args []string) error {
	t := startTimer("network")
	defer t.log()

	// Validate that no arguments were provided (catches unknown subcommands)
	if len(args) > 0 {
		return outputError(fmt.Sprintf("unknown command %q for \"webctl network\"", args[0]))
	}

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Get network entries from daemon
	entries, err := getNetworkFromDaemon(cmd)
	if err != nil {
		if errors.Is(err, ErrNoMatches) {
			return outputNotice("No matches found")
		}
		return outputError(err.Error())
	}

	// JSON mode: output JSON
	if JSONOutput {
		return outputNetworkJSON(entries, resolveMaxBodySize(cmd))
	}

	// Text mode: bound bodies to --max-body-size before rendering, then format.
	// The formatter prints the already-bounded text and does not truncate itself.
	applyBodyTruncation(entries, resolveMaxBodySize(cmd))
	opts := format.NewOutputOptions(JSONOutput, NoColor)
	opts.ShowHeaders = resolveHeadersFlag(cmd)
	return format.Network(os.Stdout, entries, opts)
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

	applyBodyTruncation(entries, resolveMaxBodySize(cmd))

	return marshalSaveEnvelope(map[string]any{
		"ok":      true,
		"entries": entries,
		"count":   len(entries),
	})
}

// resolveMaxBodySize reads the --max-body-size flag, falling back to the parent
// command's persistent flag and finally the default. It distinguishes an unset
// flag from an explicit value via Changed, so a deliberate --max-body-size 0
// (suppress all body content) is honoured rather than coalesced to the default.
func resolveMaxBodySize(cmd *cobra.Command) int {
	if cmd.Flags().Changed("max-body-size") {
		maxBodySize, _ := cmd.Flags().GetInt("max-body-size")
		return maxBodySize
	}
	if cmd.Parent() != nil && cmd.Parent().PersistentFlags().Changed("max-body-size") {
		maxBodySize, _ := cmd.Parent().PersistentFlags().GetInt("max-body-size")
		return maxBodySize
	}
	return ipc.DefaultMaxBodySize
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

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	debugRequest("network", "")
	ipcStart := time.Now()

	// Execute network request
	resp, err := exec.Execute(ipc.Request{Cmd: "network"})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	// Parse network data
	var data ipc.NetworkData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}

	entries := data.Entries

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
			return []ipc.NetworkEntry{}, nil
		}
		return entries[start:end], nil
	}

	return entries, nil
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
// save, and text output paths so the byte-budget logic lives in one place.
func applyBodyTruncation(entries []ipc.NetworkEntry, maxBodySize int) {
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
