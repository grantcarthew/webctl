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

var (
	networkFormat      string
	networkTypes       []string
	networkMethods     []string
	networkStatuses    []string
	networkURL         string
	networkMimes       []string
	networkMinDuration time.Duration
	networkMinSize     int64
	networkFailed      bool
	networkMaxBodySize int
	networkHead        int
	networkTail        int
	networkRange       string
)

func init() {
	networkCmd.Flags().StringVar(&networkFormat, "format", "", "Output format: json or text (auto-detect by default)")
	networkCmd.Flags().StringSliceVar(&networkTypes, "type", nil, "Filter by CDP resource type (repeatable, CSV-supported)")
	networkCmd.Flags().StringSliceVar(&networkMethods, "method", nil, "Filter by HTTP method (repeatable, CSV-supported)")
	networkCmd.Flags().StringSliceVar(&networkStatuses, "status", nil, "Filter by status code or range (repeatable, CSV-supported)")
	networkCmd.Flags().StringVar(&networkURL, "url", "", "Filter by URL regex pattern")
	networkCmd.Flags().StringSliceVar(&networkMimes, "mime", nil, "Filter by MIME type (repeatable, CSV-supported)")
	networkCmd.Flags().DurationVar(&networkMinDuration, "min-duration", 0, "Filter by minimum request duration")
	networkCmd.Flags().Int64Var(&networkMinSize, "min-size", 0, "Filter by minimum response size in bytes")
	networkCmd.Flags().BoolVar(&networkFailed, "failed", false, "Show only failed requests")
	networkCmd.Flags().IntVar(&networkMaxBodySize, "max-body-size", 102400, "Maximum body size in bytes before truncation (default 100KB)")
	networkCmd.Flags().IntVar(&networkHead, "head", 0, "Return first N entries")
	networkCmd.Flags().IntVar(&networkTail, "tail", 0, "Return last N entries")
	networkCmd.Flags().StringVar(&networkRange, "range", "", "Return entries in range (format: START-END)")
	networkCmd.MarkFlagsMutuallyExclusive("head", "tail", "range")
	rootCmd.AddCommand(networkCmd)
}

func runNetwork(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Validate URL regex if provided
	var urlRegex *regexp.Regexp
	if networkURL != "" {
		var err error
		urlRegex, err = regexp.Compile(networkURL)
		if err != nil {
			return outputError(fmt.Sprintf("invalid URL pattern: %v", err))
		}
	}

	// Parse status patterns
	statusMatchers, err := parseStatusPatterns(networkStatuses)
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

	// Apply filters
	entries = filterNetworkEntries(entries, urlRegex, statusMatchers)

	// Apply limiting (head/tail/range)
	entries, err = applyNetworkLimiting(entries, networkHead, networkTail, networkRange)
	if err != nil {
		return outputError(err.Error())
	}

	// Determine output format
	format := networkFormat
	if format == "" {
		format = "json"
	}

	if format == "text" {
		return outputNetworkText(entries)
	}
	return outputNetworkJSON(entries)
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

// filterNetworkEntries applies all network filters.
func filterNetworkEntries(entries []ipc.NetworkEntry, urlRegex *regexp.Regexp, statusMatchers []statusMatcher) []ipc.NetworkEntry {
	if len(networkTypes) == 0 && len(networkMethods) == 0 && len(statusMatchers) == 0 &&
		urlRegex == nil && len(networkMimes) == 0 && networkMinDuration == 0 &&
		networkMinSize == 0 && !networkFailed {
		return entries
	}

	var filtered []ipc.NetworkEntry
	for _, e := range entries {
		if !matchesNetworkFilters(e, urlRegex, statusMatchers) {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}

// matchesNetworkFilters returns true if entry matches all specified filters.
func matchesNetworkFilters(e ipc.NetworkEntry, urlRegex *regexp.Regexp, statusMatchers []statusMatcher) bool {
	// Type filter
	if len(networkTypes) > 0 && !matchesStringSlice(e.Type, networkTypes) {
		return false
	}

	// Method filter
	if len(networkMethods) > 0 && !matchesStringSlice(e.Method, networkMethods) {
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
	if len(networkMimes) > 0 && !matchesStringSlice(e.MimeType, networkMimes) {
		return false
	}

	// Min duration filter
	if networkMinDuration > 0 && e.Duration < networkMinDuration.Seconds() {
		return false
	}

	// Min size filter
	if networkMinSize > 0 && e.Size < networkMinSize {
		return false
	}

	// Failed filter
	if networkFailed && !e.Failed {
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

// outputNetworkText outputs entries in human-readable text format.
func outputNetworkText(entries []ipc.NetworkEntry) error {
	for _, e := range entries {
		ts := time.UnixMilli(e.RequestTime).Local()
		timestamp := ts.Format("2006-01-02 15:04:05.000")

		// Format duration
		var durationStr string
		if e.Duration > 0 {
			durationStr = fmt.Sprintf("%.0fms", e.Duration*1000)
		} else {
			durationStr = "0ms"
		}

		// Format status
		var statusStr string
		if e.Failed {
			statusStr = "ERR"
		} else if e.Status > 0 {
			statusStr = strconv.Itoa(e.Status)
		} else {
			statusStr = "---"
		}

		// Format MIME type (use "-" if empty)
		mimeType := e.MimeType
		if mimeType == "" {
			mimeType = "-"
		}

		// Base output line
		line := fmt.Sprintf("[%s] %s %s %s %s %s",
			timestamp, e.Method, statusStr, durationStr, mimeType, e.URL)

		// Add error info for failed requests
		if e.Failed && e.Error != "" {
			line += fmt.Sprintf(" (%s)", e.Error)
		}

		fmt.Println(line)
	}
	return nil
}

// outputNetworkJSON outputs entries in JSON format.
func outputNetworkJSON(entries []ipc.NetworkEntry) error {
	// Apply body truncation based on max-body-size flag
	for i := range entries {
		if len(entries[i].Body) > networkMaxBodySize {
			entries[i].Body = entries[i].Body[:networkMaxBodySize]
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
