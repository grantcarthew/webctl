package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/grantcarthew/webctl/internal/executor"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var htmlCmd = &cobra.Command{
	Use:   "html [selector]",
	Short: "Extract HTML from current page",
	Long:  "Extracts HTML from the current page. Without selector, returns full page HTML. With selector, returns matching element(s) HTML.",
	RunE:  runHTML,
}

var htmlOutput string

func init() {
	htmlCmd.Flags().StringVarP(&htmlOutput, "output", "o", "", "Save to specified path instead of temp directory")
	rootCmd.AddCommand(htmlCmd)
}

func runHTML(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

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

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "html",
		Params: params,
	})
	if err != nil {
		return outputError(err.Error())
	}

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
	if htmlOutput != "" {
		outputPath = htmlOutput
	} else {
		// Generate filename in temp directory
		outputPath, err = generateHTMLPath(exec)
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

	// Return JSON with file path
	result := map[string]any{
		"ok":   true,
		"path": outputPath,
	}
	return outputJSON(os.Stdout, result)
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
