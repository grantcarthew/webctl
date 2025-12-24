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
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var screenshotCmd = &cobra.Command{
	Use:   "screenshot",
	Short: "Capture screenshot of current page",
	Long: `Captures a PNG screenshot of the current page and saves it to a file.

By default captures the current viewport. Use --full-page to capture the
entire scrollable page content.

Flags:
  --full-page       Capture entire scrollable page instead of viewport only
  --output, -o      Save to specified path instead of temp directory

File location:
  Default: /tmp/webctl-screenshots/YY-MM-DD-HHMMSS-{title}.png
  Custom:  Specified path with --output flag

The filename includes a timestamp and normalised page title for easy
identification when browsing the temp directory.

Viewport screenshot (default):
  screenshot                            # Current visible area
  screenshot -o ./debug/page.png        # Save to specific location

Full-page screenshot:
  screenshot --full-page                # Entire scrollable content
  screenshot --full-page -o ./full.png  # Full page to specific path

Common patterns:
  # Capture after navigation
  navigate example.com --wait
  screenshot

  # Before/after comparison
  screenshot -o ./before.png
  click "#toggle-dark-mode"
  screenshot -o ./after.png

  # Document visual state
  screenshot --full-page -o ./docs/homepage-full.png

  # Debug layout issue
  navigate localhost:3000 --wait
  screenshot --full-page
  # Examine the screenshot file for layout issues

  # CI/CD test artifacts
  screenshot --full-page -o ./test-results/homepage-${BUILD_ID}.png

  # Multi-tab capture
  target "Admin Panel"
  screenshot -o ./admin.png
  target "Dashboard"
  screenshot -o ./dashboard.png

Response:
  {"ok": true, "path": "/tmp/webctl-screenshots/24-12-24-143052-example-domain.png"}

Error cases:
  - "failed to capture screenshot" - CDP capture failed
  - "failed to write screenshot: permission denied" - cannot write to path
  - "no active session" - no browser page open
  - "daemon not running" - start daemon first with: webctl start

Note: Screenshots are PNG format (lossless) for accurate debugging. Large
full-page screenshots of complex pages may take a moment to capture.`,
	RunE: runScreenshot,
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
