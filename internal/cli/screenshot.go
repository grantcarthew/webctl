package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/grantcarthew/webctl/internal/executor"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var screenshotCmd = &cobra.Command{
	Use:   "screenshot",
	Short: "Capture screenshot of current page",
	Long:  "Captures a screenshot of the current page and saves it to a file. Returns JSON with the file path.",
	RunE:  runScreenshot,
}

func init() {
	screenshotCmd.Flags().Bool("full-page", false, "Capture entire scrollable page instead of viewport")
	screenshotCmd.Flags().StringP("output", "o", "", "Save to specified path instead of temp directory")
	rootCmd.AddCommand(screenshotCmd)
}

func runScreenshot(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Read flags from command
	fullPage, _ := cmd.Flags().GetBool("full-page")
	output, _ := cmd.Flags().GetString("output")

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	// Send screenshot request with fullPage parameter
	params, err := json.Marshal(ipc.ScreenshotParams{
		FullPage: fullPage,
	})
	if err != nil {
		return outputError(err.Error())
	}

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "screenshot",
		Params: params,
	})
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	// Parse screenshot data
	var data ipc.ScreenshotData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return outputError(err.Error())
	}

	// Determine output path
	var outputPath string
	if output != "" {
		outputPath = output
	} else {
		// Generate filename in temp directory
		outputPath, err = generateScreenshotPath(exec)
		if err != nil {
			return outputError(err.Error())
		}
	}

	// Ensure parent directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return outputError(fmt.Sprintf("failed to create directory: %v", err))
	}

	// Write PNG data to file
	if err := os.WriteFile(outputPath, data.Data, 0644); err != nil {
		return outputError(fmt.Sprintf("failed to write screenshot: %v", err))
	}

	// Return JSON with file path
	result := map[string]any{
		"ok":   true,
		"path": outputPath,
	}
	return outputJSON(os.Stdout, result)
}

// generateScreenshotPath generates a filename in /tmp/webctl-screenshots/
// using the pattern: YY-MM-DD-HHMMSS-{normalized-title}.png
func generateScreenshotPath(exec executor.Executor) (string, error) {
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
	filename := fmt.Sprintf("%s-%s.png", timestamp, title)

	// Return path in /tmp/webctl-screenshots/
	return filepath.Join("/tmp/webctl-screenshots", filename), nil
}

// normalizeTitle normalizes a page title for use in filenames.
// Algorithm:
// 1. Trim whitespace
// 2. Limit to 30 characters
// 3. Convert non-alphanumeric to hyphens
// 4. Replace multiple consecutive hyphens with single hyphen
// 5. Remove leading/trailing hyphens
// 6. Convert to lowercase
func normalizeTitle(title string) string {
	// Trim whitespace
	title = strings.TrimSpace(title)

	// Limit to 30 characters
	if len(title) > 30 {
		title = title[:30]
	}

	// Convert non-alphanumeric to hyphens
	reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	title = reg.ReplaceAllString(title, "-")

	// Replace multiple consecutive hyphens with single hyphen
	reg = regexp.MustCompile(`-+`)
	title = reg.ReplaceAllString(title, "-")

	// Remove leading/trailing hyphens
	title = strings.Trim(title, "-")

	// Convert to lowercase
	title = strings.ToLower(title)

	// Fallback to "untitled" if empty after normalization
	if title == "" {
		title = "untitled"
	}

	return title
}
