package cli

import (
	"encoding/json"
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
	Short: "Extract HTML from current page (default: save to temp)",
	Long: `Extracts HTML from the current page with flexible output modes.

Default behavior (no subcommand):
  Saves HTML to /tmp/webctl-html/ with auto-generated filename
  Returns JSON with file path

Subcommands:
  show              Output HTML to stdout
  save <path>       Save HTML to custom path

Universal flags (work with all modes):
  --select, -s      Filter to element(s) matching CSS selector
  --find, -f        Search for text within HTML
  --raw             Skip HTML formatting (return as-is from browser)
  --json            Output in JSON format (global flag)

Examples:

Default mode (save to temp):
  html                                  # Full page to temp
  html --select "#main"                 # Element to temp
  html --find "login"                   # Search and save matches

Show mode (stdout):
  html show                             # Full page to stdout
  html show --select ".content"         # Element to stdout
  html show --find "error"              # Search and show matches

Save mode (custom path):
  html save ./page.html                 # Save to file
  html save ./output/                   # Save to dir (auto-filename)
  html save ./debug.html --select "form" --find "password"

Response formats:
  Default/Save: {"ok": true, "path": "/tmp/webctl-html/25-12-28-143052-example.html"}
  Show:         <html>...</html> (to stdout)

Error cases:
  - "selector '.missing' matched no elements" - nothing matches
  - "no matches found for 'text'" - find text not in HTML
  - "daemon not running" - start daemon first with: webctl start`,
	RunE: runHTMLDefault,
}

var htmlShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Output HTML to stdout",
	Long: `Outputs HTML to stdout for piping or inspection.

Examples:
  html show                             # Full page
  html show --select "#main"            # Specific element
  html show --find "login"              # Search within HTML
  html show --raw                       # Unformatted output`,
	RunE: runHTMLShow,
}

var htmlSaveCmd = &cobra.Command{
	Use:   "save <path>",
	Short: "Save HTML to custom path",
	Long: `Saves HTML to a custom file path.

If path is a directory, auto-generates filename.
If path is a file, uses exact path.

Examples:
  html save ./page.html                 # Save to file
  html save ./output/                   # Save to dir
  html save ./debug.html --select "#app" --find "error"`,
	Args: cobra.ExactArgs(1),
	RunE: runHTMLSave,
}

func init() {
	// Universal flags on root command (inherited by subcommands)
	htmlCmd.PersistentFlags().StringP("select", "s", "", "Filter to element(s) matching CSS selector")
	htmlCmd.PersistentFlags().StringP("find", "f", "", "Search for text within HTML")
	htmlCmd.PersistentFlags().Bool("raw", false, "Skip HTML formatting")

	// Add subcommands
	htmlCmd.AddCommand(htmlShowCmd, htmlSaveCmd)

	rootCmd.AddCommand(htmlCmd)
}

// runHTMLDefault handles default behavior: save to temp directory
func runHTMLDefault(cmd *cobra.Command, args []string) error {
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
		return outputError(err.Error())
	}

	// Get selector for filename generation
	selector, _ := cmd.Flags().GetString("select")

	// Generate filename in temp directory
	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	outputPath, err := generateHTMLPath(exec, selector)
	if err != nil {
		return outputError(err.Error())
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

// runHTMLShow handles show subcommand: output to stdout
func runHTMLShow(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Get HTML from daemon
	html, err := getHTMLFromDaemon(cmd)
	if err != nil {
		return outputError(err.Error())
	}

	// Output to stdout
	fmt.Println(html)
	return nil
}

// runHTMLSave handles save subcommand: save to custom path
func runHTMLSave(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	path := args[0]

	// Get HTML from daemon
	html, err := getHTMLFromDaemon(cmd)
	if err != nil {
		return outputError(err.Error())
	}

	// Handle directory vs file path
	fileInfo, err := os.Stat(path)
	if err == nil && fileInfo.IsDir() {
		// Path is a directory - auto-generate filename
		exec, err := execFactory.NewExecutor()
		if err != nil {
			return outputError(err.Error())
		}
		defer exec.Close()

		selector, _ := cmd.Flags().GetString("select")
		filename, err := generateHTMLFilename(exec, selector)
		if err != nil {
			return outputError(err.Error())
		}
		path = filepath.Join(path, filename)
	}

	// Write HTML to file
	if err := writeHTMLToFile(path, html); err != nil {
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

	// Execute HTML request
	resp, err := exec.Execute(ipc.Request{
		Cmd:    "html",
		Params: params,
	})
	if err != nil {
		return "", err
	}

	if !resp.OK {
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
			debugf("HTML formatting failed: %v", err)
		} else {
			html = formatted
		}
	}

	// Apply --find filter if specified (after formatting so line-based search works)
	if find != "" {
		html, err = filterHTMLByText(html, find)
		if err != nil {
			return "", err
		}
	}

	return html, nil
}

// filterHTMLByText filters HTML to only include lines containing the search text
func filterHTMLByText(html, searchText string) (string, error) {
	lines := strings.Split(html, "\n")
	var matchedLines []string

	searchLower := strings.ToLower(searchText)

	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), searchLower) {
			matchedLines = append(matchedLines, line)
		}
	}

	if len(matchedLines) == 0 {
		return "", fmt.Errorf("no matches found for '%s'", searchText)
	}

	return strings.Join(matchedLines, "\n"), nil
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
