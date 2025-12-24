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
	Short: "Show network request entries",
	Long: `Returns buffered network request entries including URLs, methods, status codes,
headers, and response bodies.

Filter Flags:
  --type        CDP resource type: xhr, fetch, document, script, stylesheet, image,
                font, websocket, media, manifest, texttrack, eventsource, prefetch, other
  --method      HTTP method: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS
  --status      Status code or range: 200, 4xx, 5xx, 200-299
  --url         URL regex pattern (Go regexp syntax)
  --mime        MIME type: application/json, text/html, image/png
  --min-duration Minimum request duration: 1s, 500ms, 100ms
  --min-size    Minimum response size in bytes
  --failed      Show only failed requests (network errors, CORS, etc.)

All filters are AND-combined. StringSlice flags support CSV (--status 4xx,5xx)
and repeatable (--status 4xx --status 5xx) syntax.`,
	RunE: runNetwork,
}

func init() {
	networkCmd.Flags().StringSlice("type", nil, "Filter by CDP resource type (repeatable, CSV-supported)")
	networkCmd.Flags().StringSlice("method", nil, "Filter by HTTP method (repeatable, CSV-supported)")
	networkCmd.Flags().StringSlice("status", nil, "Filter by status code or range (repeatable, CSV-supported)")
	networkCmd.Flags().String("url", "", "Filter by URL regex pattern")
	networkCmd.Flags().StringSlice("mime", nil, "Filter by MIME type (repeatable, CSV-supported)")
	networkCmd.Flags().Duration("min-duration", 0, "Filter by minimum request duration")
	networkCmd.Flags().Int64("min-size", 0, "Filter by minimum response size in bytes")
	networkCmd.Flags().Bool("failed", false, "Show only failed requests")
	networkCmd.Flags().Int("max-body-size", 102400, "Maximum body size in bytes before truncation (default 100KB)")
	networkCmd.Flags().Int("head", 0, "Return first N entries")
	networkCmd.Flags().Int("tail", 0, "Return last N entries")
	networkCmd.Flags().String("range", "", "Return entries in range (format: START-END)")
	networkCmd.MarkFlagsMutuallyExclusive("head", "tail", "range")
	rootCmd.AddCommand(networkCmd)
}

func runNetwork(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Read flags from command
	types, _ := cmd.Flags().GetStringSlice("type")
	methods, _ := cmd.Flags().GetStringSlice("method")
	statuses, _ := cmd.Flags().GetStringSlice("status")
	urlPattern, _ := cmd.Flags().GetString("url")
	mimes, _ := cmd.Flags().GetStringSlice("mime")
	minDuration, _ := cmd.Flags().GetDuration("min-duration")
	minSize, _ := cmd.Flags().GetInt64("min-size")
	failed, _ := cmd.Flags().GetBool("failed")
	maxBodySize, _ := cmd.Flags().GetInt("max-body-size")
	head, _ := cmd.Flags().GetInt("head")
	tail, _ := cmd.Flags().GetInt("tail")
	rangeStr, _ := cmd.Flags().GetString("range")

	// Validate URL regex if provided
	var urlRegex *regexp.Regexp
	if urlPattern != "" {
		var err error
		urlRegex, err = regexp.Compile(urlPattern)
		if err != nil {
			return outputError(fmt.Sprintf("invalid URL pattern: %v", err))
		}
	}

	// Parse status patterns
	statusMatchers, err := parseStatusPatterns(statuses)
	if err != nil {
		return outputError(err.Error())
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	resp, err := exec.Execute(ipc.Request{Cmd: "network"})
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	var data ipc.NetworkData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return outputError(err.Error())
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

	// Apply limiting (head/tail/range)
	entries, err = applyNetworkLimiting(entries, head, tail, rangeStr)
	if err != nil {
		return outputError(err.Error())
	}

	// JSON mode: output JSON
	if JSONOutput {
		return outputNetworkJSON(entries, maxBodySize)
	}

	// Text mode: use text formatter
	return format.Network(os.Stdout, entries, format.NewOutputOptions(JSONOutput, NoColor))
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

	resp := map[string]any{
		"ok":      true,
		"entries": entries,
		"count":   len(entries),
	}
	return outputJSON(os.Stdout, resp)
}

// getBodiesDir returns the path to the bodies storage directory.
func getBodiesDir() string {
	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		home, _ := os.UserHomeDir()
		stateHome = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(stateHome, "webctl", "bodies")
}
