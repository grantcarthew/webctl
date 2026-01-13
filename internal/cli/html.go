package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/grantcarthew/webctl/internal/cli/format"
	"github.com/grantcarthew/webctl/internal/executor"
	"github.com/grantcarthew/webctl/internal/htmlformat"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var htmlCmd = &cobra.Command{
	Use:   "html",
	Short: "Extract HTML from current page (default: stdout)",
	Long: `Extracts HTML from the current page with flexible output modes.

Default behavior (no subcommand):
  Outputs HTML to stdout for piping or inspection

Subcommands:
  save [path]       Save HTML to file (temp dir if no path given)

Universal flags (work with all modes):
  --select, -s      Filter to element(s) matching CSS selector
  --find, -f        Search for text within HTML
  --raw             Skip HTML formatting (return as-is from browser)
  --json            Output in JSON format (global flag)

Examples:

Default mode (stdout):
  html                                  # Full page to stdout
  html --select "#main"                 # Element to stdout
  html --find "login"                   # Search and show matches

Save mode (file):
  html save                             # Save to temp with auto-filename
  html save ./page.html                 # Save to custom file
  html save ./output/                   # Save to dir (auto-filename)
  html save --select "form" --find "password"

Response formats:
  Default:  <html>...</html> (to stdout)
  Save:     /tmp/webctl-html/25-12-28-143052-example.html

Error cases:
  - "selector '.missing' matched no elements" - nothing matches
  - "No matches found" - find text not in HTML
  - "daemon not running" - start daemon first with: webctl start`,
	RunE: runHTMLDefault,
}

var htmlSaveCmd = &cobra.Command{
	Use:   "save [path]",
	Short: "Save HTML to file",
	Long: `Saves HTML to a file.

Path conventions:
  (no path)         Save to /tmp/webctl-html/ with auto-generated filename
  ./page.html       Save to exact file path
  ./output/         Save to directory with auto-generated filename (trailing slash required)

Examples:
  html save                             # Save to temp dir
  html save ./page.html                 # Save to file
  html save ./output/                   # Save to dir (creates if needed)
  html save --select "#app" --find "error"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runHTMLSave,
}

func init() {
	// Universal flags on root command (inherited by subcommands)
	htmlCmd.PersistentFlags().StringP("select", "s", "", "Filter to element(s) matching CSS selector")
	htmlCmd.PersistentFlags().StringP("find", "f", "", "Search for text within HTML")
	htmlCmd.PersistentFlags().IntP("before", "B", 0, "Show N lines before each match (requires --find)")
	htmlCmd.PersistentFlags().IntP("after", "A", 0, "Show N lines after each match (requires --find)")
	htmlCmd.PersistentFlags().IntP("context", "C", 0, "Show N lines before and after each match (requires --find)")
	htmlCmd.PersistentFlags().Bool("raw", false, "Skip HTML formatting")

	// Add subcommands
	htmlCmd.AddCommand(htmlSaveCmd)

	rootCmd.AddCommand(htmlCmd)
}

// runHTMLDefault handles default behavior: output to stdout
func runHTMLDefault(cmd *cobra.Command, args []string) error {
	t := startTimer("html")
	defer t.log()

	// Validate that no arguments were provided (catches unknown subcommands)
	if len(args) > 0 {
		return outputError(fmt.Sprintf("unknown command %q for \"webctl html\"", args[0]))
	}

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Get HTML from daemon
	html, err := getHTMLFromDaemon(cmd)
	if err != nil {
		if errors.Is(err, ErrNoMatches) {
			return outputNotice("No matches found")
		}
		if errors.Is(err, ErrNoElements) {
			return outputNotice("No elements found")
		}
		return outputError(err.Error())
	}

	// JSON mode: output JSON
	if JSONOutput {
		result := map[string]any{
			"ok":   true,
			"html": html,
		}
		return outputJSON(os.Stdout, result)
	}

	// Output to stdout
	fmt.Println(html)
	return nil
}

// runHTMLSave handles save subcommand: save to file
func runHTMLSave(cmd *cobra.Command, args []string) error {
	t := startTimer("html save")
	defer t.log()

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Get HTML from daemon
	html, err := getHTMLFromDaemon(cmd)
	if err != nil {
		if errors.Is(err, ErrNoMatches) {
			return outputNotice("No matches found")
		}
		if errors.Is(err, ErrNoElements) {
			return outputNotice("No elements found")
		}
		return outputError(err.Error())
	}

	// Get selector for filename generation
	selector, _ := cmd.Flags().GetString("select")
	if selector == "" && cmd.Parent() != nil {
		selector, _ = cmd.Parent().PersistentFlags().GetString("select")
	}

	var outputPath string

	if len(args) == 0 {
		// No path provided - save to temp directory
		exec, err := execFactory.NewExecutor()
		if err != nil {
			return outputError(err.Error())
		}
		defer exec.Close()

		outputPath, err = generateHTMLPath(exec, selector)
		if err != nil {
			return outputError(err.Error())
		}
	} else {
		// Path provided
		path := args[0]

		// Check if path ends with separator (directory convention)
		if strings.HasSuffix(path, string(os.PathSeparator)) || strings.HasSuffix(path, "/") {
			// Path ends with separator - treat as directory, auto-generate filename
			exec, err := execFactory.NewExecutor()
			if err != nil {
				return outputError(err.Error())
			}
			defer exec.Close()

			filename, err := generateHTMLFilename(exec, selector)
			if err != nil {
				return outputError(err.Error())
			}

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

	// Write HTML to file
	if err := writeHTMLToFile(outputPath, html); err != nil {
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

// getHTMLFromDaemon fetches HTML from daemon, applying filters and formatting
func getHTMLFromDaemon(cmd *cobra.Command) (string, error) {
	// Try to get flags from command, falling back to parent for persistent flags
	selector, _ := cmd.Flags().GetString("select")
	if selector == "" && cmd.Parent() != nil {
		selector, _ = cmd.Parent().PersistentFlags().GetString("select")
	}

	find, _ := cmd.Flags().GetString("find")
	if find == "" && cmd.Parent() != nil {
		find, _ = cmd.Parent().PersistentFlags().GetString("find")
	}

	raw, _ := cmd.Flags().GetBool("raw")
	if !raw && cmd.Parent() != nil {
		raw, _ = cmd.Parent().PersistentFlags().GetBool("raw")
	}

	before, _ := cmd.Flags().GetInt("before")
	if before == 0 && cmd.Parent() != nil {
		before, _ = cmd.Parent().PersistentFlags().GetInt("before")
	}

	after, _ := cmd.Flags().GetInt("after")
	if after == 0 && cmd.Parent() != nil {
		after, _ = cmd.Parent().PersistentFlags().GetInt("after")
	}

	context, _ := cmd.Flags().GetInt("context")
	if context == 0 && cmd.Parent() != nil {
		context, _ = cmd.Parent().PersistentFlags().GetInt("context")
	}

	// -C is shorthand for -B N -A N
	if context > 0 {
		before = context
		after = context
	}

	debugParam("selector=%q find=%q raw=%v before=%d after=%d", selector, find, raw, before, after)

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return "", err
	}
	defer exec.Close()

	// Build request with selector
	params, err := json.Marshal(ipc.HTMLParams{
		Selector: selector,
	})
	if err != nil {
		return "", err
	}

	debugRequest("html", fmt.Sprintf("selector=%q", selector))
	ipcStart := time.Now()

	// Execute HTML request
	resp, err := exec.Execute(ipc.Request{
		Cmd:    "html",
		Params: params,
	})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		return "", err
	}

	if !resp.OK {
		if isNoElementsError(resp.Error) {
			return "", ErrNoElements
		}
		return "", fmt.Errorf("%s", resp.Error)
	}

	// Parse HTML data
	var data ipc.HTMLData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return "", err
	}

	html := data.HTML

	// Format HTML unless --raw flag is set
	if !raw {
		formatted, err := htmlformat.Format(html)
		if err != nil {
			// If formatting fails, fall back to raw HTML
			debugf("FORMAT", "HTML formatting failed: %v", err)
		} else {
			html = formatted
		}
	}

	// Apply --find filter if specified (after formatting so line-based search works)
	if find != "" {
		beforeCount := strings.Count(html, "\n") + 1
		html, err = filterHTMLByText(html, find, before, after)
		if err != nil {
			return "", err
		}
		afterCount := strings.Count(html, "\n") + 1
		debugFilter(fmt.Sprintf("--find %q", find), beforeCount, afterCount)
	}

	return html, nil
}

// filterHTMLByText filters HTML to only include lines containing the search text
// with optional context lines before and after each match
func filterHTMLByText(html, searchText string, before, after int) (string, error) {
	lines := strings.Split(html, "\n")
	searchLower := strings.ToLower(searchText)

	// Find all matching line indices
	var matchIndices []int
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), searchLower) {
			matchIndices = append(matchIndices, i)
		}
	}

	if len(matchIndices) == 0 {
		return "", ErrNoMatches
	}

	// If no context requested, return matching lines with separators between non-adjacent matches
	if before == 0 && after == 0 {
		var result []string
		for i, idx := range matchIndices {
			// Add separator if this match is not adjacent to the previous one
			if i > 0 && idx > matchIndices[i-1]+1 {
				result = append(result, "--")
			}
			result = append(result, lines[idx])
		}
		return strings.Join(result, "\n"), nil
	}

	// Build ranges with context, merging overlapping regions
	type lineRange struct {
		start, end int
	}
	var ranges []lineRange

	for _, idx := range matchIndices {
		start := idx - before
		if start < 0 {
			start = 0
		}
		end := idx + after
		if end >= len(lines) {
			end = len(lines) - 1
		}

		// Merge with previous range if overlapping or adjacent
		if len(ranges) > 0 && start <= ranges[len(ranges)-1].end+1 {
			ranges[len(ranges)-1].end = end
		} else {
			ranges = append(ranges, lineRange{start, end})
		}
	}

	// Build output with separators between non-contiguous ranges
	var result []string
	for i, r := range ranges {
		if i > 0 {
			result = append(result, "--")
		}
		for j := r.start; j <= r.end; j++ {
			result = append(result, lines[j])
		}
	}

	return strings.Join(result, "\n"), nil
}

// writeHTMLToFile writes HTML content to a file, creating directories if needed
func writeHTMLToFile(path, html string) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Write HTML to file
	if err := os.WriteFile(path, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to write HTML: %v", err)
	}

	debugFile("wrote", path, len(html))
	return nil
}

// generateHTMLPath generates a full path in /tmp/webctl-html/
// using the pattern: YY-MM-DD-HHMMSS-{identifier}.html
func generateHTMLPath(exec executor.Executor, selector string) (string, error) {
	filename, err := generateHTMLFilename(exec, selector)
	if err != nil {
		return "", err
	}

	return filepath.Join("/tmp/webctl-html", filename), nil
}

// generateHTMLFilename generates a filename using the pattern:
// YY-MM-DD-HHMMSS-{identifier}.html
// Identifier is based on selector (if provided) or page title
func generateHTMLFilename(exec executor.Executor, selector string) (string, error) {
	// Generate timestamp: YY-MM-DD-HHMMSS
	now := time.Now()
	timestamp := now.Format("06-01-02-150405")

	// Determine identifier
	identifier := "page"
	if selector != "" {
		identifier = sanitizeSelector(selector)
	} else {
		// Get page title for identifier
		resp, err := exec.Execute(ipc.Request{Cmd: "status"})
		if err == nil && resp.OK {
			var status ipc.StatusData
			if err := json.Unmarshal(resp.Data, &status); err == nil {
				if status.ActiveSession != nil && status.ActiveSession.Title != "" {
					identifier = normalizeTitle(status.ActiveSession.Title)
				}
			}
		}
	}

	// Generate filename
	return fmt.Sprintf("%s-%s.html", timestamp, identifier), nil
}

// sanitizeSelector converts a CSS selector to a safe filename component
func sanitizeSelector(selector string) string {
	// Remove leading # or .
	selector = strings.TrimPrefix(selector, "#")
	selector = strings.TrimPrefix(selector, ".")

	// Limit length
	if len(selector) > 30 {
		selector = selector[:30]
	}

	// Convert non-alphanumeric to hyphens
	reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	selector = reg.ReplaceAllString(selector, "-")

	// Replace multiple consecutive hyphens with single hyphen
	reg = regexp.MustCompile(`-+`)
	selector = reg.ReplaceAllString(selector, "-")

	// Remove leading/trailing hyphens
	selector = strings.Trim(selector, "-")

	// Convert to lowercase
	selector = strings.ToLower(selector)

	// Fallback if empty
	if selector == "" {
		selector = "element"
	}

	return selector
}
