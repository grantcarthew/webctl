package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/grantcarthew/webctl/internal/cli/format"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var networkCmd = &cobra.Command{
	Use:   "network",
	Short: "Extract network requests from current page (default: save to temp)",
	Long: `Extracts network requests from the current page with flexible output modes.

Default behavior (no subcommand):
  Saves network requests to /tmp/webctl-network/ with auto-generated filename
  Returns JSON with file path

Subcommands:
  show              Output network requests to stdout
  save <path>       Save network requests to custom path

Universal flags (work with default/show/save modes):
  --find, -f        Search for text within URLs and bodies
  --raw             Skip formatting (return raw JSON)
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

Default mode (save to temp):
  network                                  # All requests to temp
  network --status 4xx                     # Only 4xx to temp
  network --find "api"                     # Search and save matches

Show mode (stdout):
  network show                             # All requests to stdout
  network show --status 4xx,5xx            # Only errors
  network show --find "fetch"              # Search and show matches
  network show --tail 20                   # Last 20 entries

Save mode (custom path):
  network save ./logs/requests.json        # Save to file
  network save ./output/                   # Save to dir (auto-filename)
  network save ./errors.json --status 5xx --tail 50

Response formats:
  Default/Save: {"ok": true, "path": "/tmp/webctl-network/25-12-28-143052-network.json"}
  Show:         GET https://example.com 200 45ms (to stdout)

Error cases:
  - "no matches found for 'text'" - find text not in requests
  - "daemon not running" - start daemon first with: webctl start`,
	RunE: runNetworkDefault,
}

var networkShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Output network requests to stdout",
	Long: `Outputs network requests to stdout for real-time monitoring and piping.

Examples:
  network show                             # All requests
  network show --status 4xx                # Only client errors
  network show --find "api"                # Search within URLs/bodies
  network show --tail 20                   # Last 20 entries`,
	RunE: runNetworkShow,
}

var networkSaveCmd = &cobra.Command{
	Use:   "save <path>",
	Short: "Save network requests to custom path",
	Long: `Saves network requests to a custom file path.

If path is a directory, auto-generates filename.
If path is a file, uses exact path.

Examples:
  network save ./logs/requests.json        # Save to file
  network save ./output/                   # Save to dir
  network save ./errors.json --status 5xx --method POST`,
	Args: cobra.ExactArgs(1),
	RunE: runNetworkSave,
}

func init() {
	// Universal flags on root command (inherited by default/show/save subcommands)
	networkCmd.PersistentFlags().StringP("find", "f", "", "Search for text within URLs and bodies")
	networkCmd.PersistentFlags().Bool("raw", false, "Skip formatting (return raw JSON)")

	// Network-specific filter flags
	networkCmd.PersistentFlags().StringSlice("type", nil, "Filter by CDP resource type (repeatable, CSV-supported)")
	networkCmd.PersistentFlags().StringSlice("method", nil, "Filter by HTTP method (repeatable, CSV-supported)")
	networkCmd.PersistentFlags().StringSlice("status", nil, "Filter by status code or range (repeatable, CSV-supported)")
	networkCmd.PersistentFlags().String("url", "", "Filter by URL regex pattern")
	networkCmd.PersistentFlags().StringSlice("mime", nil, "Filter by MIME type (repeatable, CSV-supported)")
	networkCmd.PersistentFlags().Duration("min-duration", 0, "Filter by minimum request duration")
	networkCmd.PersistentFlags().Int64("min-size", 0, "Filter by minimum response size in bytes")
	networkCmd.PersistentFlags().Bool("failed", false, "Show only failed requests")
	networkCmd.PersistentFlags().Int("max-body-size", 102400, "Maximum body size in bytes before truncation (default 100KB)")
	networkCmd.PersistentFlags().Int("head", 0, "Return first N entries")
	networkCmd.PersistentFlags().Int("tail", 0, "Return last N entries")
	networkCmd.PersistentFlags().String("range", "", "Return entries in range (format: START-END)")
	networkCmd.MarkFlagsMutuallyExclusive("head", "tail", "range")

	// Add all subcommands
	networkCmd.AddCommand(networkShowCmd, networkSaveCmd)

	rootCmd.AddCommand(networkCmd)
}

// runNetworkDefault handles default behavior: save to temp directory
func runNetworkDefault(cmd *cobra.Command, args []string) error {
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
		return outputError(err.Error())
	}

	// Generate filename in temp directory
	outputPath, err := generateNetworkPath()
	if err != nil {
		return outputError(err.Error())
	}

	// Get max-body-size for JSON output
	maxBodySize, _ := cmd.Flags().GetInt("max-body-size")
	if maxBodySize == 0 && cmd.Parent() != nil {
		maxBodySize, _ = cmd.Parent().PersistentFlags().GetInt("max-body-size")
	}
	if maxBodySize == 0 {
		maxBodySize = 102400 // Default
	}

	// Write network requests to file
	if err := writeNetworkToFile(outputPath, entries, maxBodySize); err != nil {
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

// runNetworkShow handles show subcommand: output to stdout
func runNetworkShow(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Get network entries from daemon
	entries, err := getNetworkFromDaemon(cmd)
	if err != nil {
		return outputError(err.Error())
	}

	// JSON mode: output JSON
	if JSONOutput {
		maxBodySize, _ := cmd.Flags().GetInt("max-body-size")
		if maxBodySize == 0 && cmd.Parent() != nil {
			maxBodySize, _ = cmd.Parent().PersistentFlags().GetInt("max-body-size")
		}
		if maxBodySize == 0 {
			maxBodySize = 102400
		}
		return outputNetworkJSON(entries, maxBodySize)
	}

	// Check --raw flag
	raw, _ := cmd.Flags().GetBool("raw")
	if !raw && cmd.Parent() != nil {
		raw, _ = cmd.Parent().PersistentFlags().GetBool("raw")
	}

	if raw {
		// Raw mode: output as JSON
		maxBodySize, _ := cmd.Flags().GetInt("max-body-size")
		if maxBodySize == 0 && cmd.Parent() != nil {
			maxBodySize, _ = cmd.Parent().PersistentFlags().GetInt("max-body-size")
		}
		if maxBodySize == 0 {
			maxBodySize = 102400
		}
		return outputNetworkJSON(entries, maxBodySize)
	}

	// Text mode: use text formatter
	return format.Network(os.Stdout, entries, format.NewOutputOptions(JSONOutput, NoColor))
}

// runNetworkSave handles save subcommand: save to custom path
func runNetworkSave(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	path := args[0]

	// Get network entries from daemon
	entries, err := getNetworkFromDaemon(cmd)
	if err != nil {
		return outputError(err.Error())
	}

	// Handle directory vs file path
	fileInfo, err := os.Stat(path)
	if err == nil && fileInfo.IsDir() {
		// Path is a directory - auto-generate filename
		filename := generateNetworkFilename()
		path = filepath.Join(path, filename)
	}

	// Get max-body-size for JSON output
	maxBodySize, _ := cmd.Flags().GetInt("max-body-size")
	if maxBodySize == 0 && cmd.Parent() != nil {
		maxBodySize, _ = cmd.Parent().PersistentFlags().GetInt("max-body-size")
	}
	if maxBodySize == 0 {
		maxBodySize = 102400
	}

	// Write network requests to file
	if err := writeNetworkToFile(path, entries, maxBodySize); err != nil {
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

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return nil, err
	}
	defer exec.Close()

	// Execute network request
	resp, err := exec.Execute(ipc.Request{Cmd: "network"})
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
	entries = filterNetworkEntries(entries, urlRegex, statusMatchers, filterOpts)

	// Apply --find filter if specified
	if find != "" {
		entries = filterNetworkByText(entries, find)
		if len(entries) == 0 {
			return nil, fmt.Errorf("no matches found for '%s'", find)
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
		// Search in body
		if strings.Contains(strings.ToLower(entry.Body), searchLower) {
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

// outputNetworkJSON outputs entries in JSON format.
func outputNetworkJSON(entries []ipc.NetworkEntry, maxBodySize int) error {
	// Apply body truncation based on max-body-size flag
	for i := range entries {
		if len(entries[i].Body) > maxBodySize {
			entries[i].Body = entries[i].Body[:maxBodySize]
			entries[i].BodyTruncated = true
		}
	}

	result := map[string]any{
		"ok":      true,
		"entries": entries,
		"count":   len(entries),
	}
	return outputJSON(os.Stdout, result)
}

// writeNetworkToFile writes network entries to a file in JSON format, creating directories if needed
func writeNetworkToFile(path string, entries []ipc.NetworkEntry, maxBodySize int) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Apply body truncation
	for i := range entries {
		if len(entries[i].Body) > maxBodySize {
			entries[i].Body = entries[i].Body[:maxBodySize]
			entries[i].BodyTruncated = true
		}
	}

	// Marshal entries to JSON
	data := map[string]any{
		"ok":      true,
		"entries": entries,
		"count":   len(entries),
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal network entries: %v", err)
	}

	// Write to file
	if err := os.WriteFile(path, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write network entries: %v", err)
	}

	return nil
}

// generateNetworkPath generates a full path in /tmp/webctl-network/
// using the pattern: YY-MM-DD-HHMMSS-network.json
func generateNetworkPath() (string, error) {
	filename := generateNetworkFilename()
	return filepath.Join("/tmp/webctl-network", filename), nil
}

// generateNetworkFilename generates a filename using the pattern:
// YY-MM-DD-HHMMSS-network.json
func generateNetworkFilename() string {
	// Generate timestamp: YY-MM-DD-HHMMSS
	now := time.Now()
	timestamp := now.Format("06-01-02-150405")

	// Generate filename with fixed identifier "network"
	return fmt.Sprintf("%s-network.json", timestamp)
}
