package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/grantcarthew/webctl/internal/cli/format"
	"github.com/grantcarthew/webctl/internal/executor"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var htmlCmd = &cobra.Command{
	Use:   "html [selector]",
	Short: "Extract HTML from current page",
	Long: `Extracts HTML from the current page and saves it to a file.

Without a selector, returns the complete page HTML (document.documentElement).
With a selector, returns the outer HTML of matching element(s).

The HTML is saved to a file (not returned inline) because pages are typically
large. This allows incremental reading with offset/limit parameters.

Flags:
  --output, -o      Save to specified path instead of temp directory

File location:
  Default: /tmp/webctl-html/YY-MM-DD-HHMMSS-{title}.html
  Custom:  Specified path with --output flag

Full page extraction:
  html                                  # Entire page HTML
  html -o ./debug/page.html             # Save to specific location

Selector extraction (outer HTML):
  html "#main"                          # Element by ID
  html ".content"                       # Element by class
  html "article"                        # Element by tag
  html "nav > ul > li"                  # Nested selector
  html "[data-testid=results]"          # By test ID

Multiple matches:
When a selector matches multiple elements, all are included with comment
separators showing the match count:

  html "div.card"

  Output file contains:
  <!-- Element 1 of 3: div.card -->
  <div class="card">...</div>

  <!-- Element 2 of 3: div.card -->
  <div class="card">...</div>

  <!-- Element 3 of 3: div.card -->
  <div class="card">...</div>

Common patterns:
  # Debug page structure
  navigate example.com --wait
  html
  # Read the file to analyse structure

  # Extract specific section
  html "#main-content" -o ./content.html

  # Compare before/after
  html "#results" -o ./before.html
  click "#load-more"
  ready
  html "#results" -o ./after.html

  # Scrape multiple items
  html ".product-card"                  # All product cards in one file

Response:
  {"ok": true, "path": "/tmp/webctl-html/24-12-24-143052-example-domain.html"}

Error cases:
  - "selector '.missing' matched no elements" - nothing matches
  - "invalid CSS selector syntax" - malformed selector
  - "failed to write HTML: permission denied" - cannot write to path
  - "daemon not running" - start daemon first with: webctl start`,
	RunE: runHTML,
}

func init() {
	htmlCmd.Flags().StringP("output", "o", "", "Save to specified path instead of temp directory")
	rootCmd.AddCommand(htmlCmd)
}

func runHTML(cmd *cobra.Command, args []string) error {
	start := time.Now()
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Read flags from command
	output, _ := cmd.Flags().GetString("output")

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	// Build request with optional selector
	var selector string
	if len(args) > 0 {
		selector = args[0]
	}

	params, err := json.Marshal(ipc.HTMLParams{
		Selector: selector,
	})
	if err != nil {
		return outputError(err.Error())
	}

	t1 := time.Now()
	resp, err := exec.Execute(ipc.Request{
		Cmd:    "html",
		Params: params,
	})
	debugf("html IPC call took %v", time.Since(t1))
	if err != nil {
		return outputError(err.Error())
	}
	_ = start // used later

	if !resp.OK {
		return outputError(resp.Error)
	}

	// Parse HTML data
	var data ipc.HTMLData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return outputError(err.Error())
	}

	// Determine output path
	var outputPath string
	if output != "" {
		outputPath = output
	} else {
		// Generate filename in temp directory
		t2 := time.Now()
		outputPath, err = generateHTMLPath(exec)
		debugf("generateHTMLPath took %v", time.Since(t2))
		if err != nil {
			return outputError(err.Error())
		}
	}

	// Ensure parent directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return outputError(fmt.Sprintf("failed to create directory: %v", err))
	}

	// Write HTML to file
	if err := os.WriteFile(outputPath, []byte(data.HTML), 0644); err != nil {
		return outputError(fmt.Sprintf("failed to write HTML: %v", err))
	}

	debugf("runHTML total: %v", time.Since(start))

	// JSON mode: return JSON with file path
	if JSONOutput {
		result := map[string]any{
			"ok":   true,
			"path": outputPath,
		}
		return outputJSON(os.Stdout, result)
	}

	// Text mode: just output the file path
	return format.FilePath(os.Stdout, outputPath)
}

// generateHTMLPath generates a filename in /tmp/webctl-html/
// using the pattern: YY-MM-DD-HHMMSS-{normalized-title}.html
func generateHTMLPath(exec executor.Executor) (string, error) {
	// Get current session for title
	resp, err := exec.Execute(ipc.Request{Cmd: "status"})
	if err != nil {
		return "", err
	}

	if !resp.OK {
		return "", fmt.Errorf("%s", resp.Error)
	}

	var status ipc.StatusData
	if err := json.Unmarshal(resp.Data, &status); err != nil {
		return "", err
	}

	// Get title from active session (fallback to "untitled")
	title := "untitled"
	if status.ActiveSession != nil && status.ActiveSession.Title != "" {
		title = normalizeTitle(status.ActiveSession.Title)
	}

	// Generate timestamp: YY-MM-DD-HHMMSS
	now := time.Now()
	timestamp := now.Format("06-01-02-150405")

	// Generate filename
	filename := fmt.Sprintf("%s-%s.html", timestamp, title)

	// Return path in /tmp/webctl-html/
	return filepath.Join("/tmp/webctl-html", filename), nil
}
