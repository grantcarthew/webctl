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
	Short: "Capture screenshot of current page (default: save to temp)",
	Long: `Captures a PNG screenshot of the current page and saves it to a file.

By default captures the current viewport and saves to temp directory.
Use --full-page to capture the entire scrollable page content.

Default behavior (no subcommand):
  Saves screenshot to /tmp/webctl-screenshots/ with auto-generated filename

Subcommands:
  save [path]       Save screenshot to file (temp dir if no path given)

Flags:
  --full-page       Capture entire scrollable page instead of viewport only

File location:
  Default: /tmp/webctl-screenshots/YY-MM-DD-HHMMSS-{title}.png
  Custom:  Specified path with save subcommand

The filename includes a timestamp and normalised page title for easy
identification when browsing the temp directory.

Examples:

Default mode (save to temp):
  screenshot                            # Current visible area to temp
  screenshot --full-page                # Entire scrollable content to temp

Save mode (custom path):
  screenshot save                       # Same as default (to temp)
  screenshot save ./page.png            # Save to specific location
  screenshot save ./full.png --full-page  # Full page to specific path

Common patterns:
  # Capture after navigation
  navigate example.com --wait
  screenshot

  # Before/after comparison
  screenshot save ./before.png
  click "#toggle-dark-mode"
  screenshot save ./after.png

  # Document visual state
  screenshot save ./docs/homepage-full.png --full-page

  # Debug layout issue
  navigate localhost:3000 --wait
  screenshot --full-page
  # Examine the screenshot file for layout issues

  # CI/CD test artifacts
  screenshot save ./test-results/homepage-${BUILD_ID}.png --full-page

  # Multi-tab capture
  target "Admin Panel"
  screenshot save ./admin.png
  target "Dashboard"
  screenshot save ./dashboard.png

Response:
  /tmp/webctl-screenshots/24-12-24-143052-example-domain.png

Error cases:
  - "failed to capture screenshot" - CDP capture failed
  - "failed to write screenshot: permission denied" - cannot write to path
  - "no active session" - no browser page open
  - "daemon not running" - start daemon first with: webctl start

Note: Screenshots are PNG format (lossless) for accurate debugging. Large
full-page screenshots of complex pages may take a moment to capture.`,
	RunE: runScreenshotDefault,
}

var screenshotSaveCmd = &cobra.Command{
	Use:   "save [path]",
	Short: "Save screenshot to file",
	Long: `Saves screenshot to a file.

Path conventions:
  (no path)         Save to /tmp/webctl-screenshots/ with auto-generated filename
  ./page.png        Save to exact file path
  ./output/         Save to directory with auto-generated filename (trailing slash required)

Examples:
  screenshot save                       # Save to temp dir
  screenshot save ./page.png            # Save to file
  screenshot save ./output/             # Save to dir (creates if needed)
  screenshot save ./full.png --full-page`,
	Args: cobra.MaximumNArgs(1),
	RunE: runScreenshotSave,
}

func init() {
	screenshotCmd.PersistentFlags().Bool("full-page", false, "Capture entire scrollable page instead of viewport")

	screenshotCmd.AddCommand(screenshotSaveCmd)
	rootCmd.AddCommand(screenshotCmd)
}

// runScreenshotDefault handles default behavior: save to temp directory
func runScreenshotDefault(cmd *cobra.Command, args []string) error {
	// Validate that no arguments were provided (catches unknown subcommands)
	if len(args) > 0 {
		return outputError(fmt.Sprintf("unknown command %q for \"webctl screenshot\"", args[0]))
	}

	return captureAndSaveScreenshot(cmd, "")
}

// runScreenshotSave handles save subcommand: save to file
func runScreenshotSave(cmd *cobra.Command, args []string) error {
	path := ""
	if len(args) > 0 {
		path = args[0]
	}
	return captureAndSaveScreenshot(cmd, path)
}

// captureAndSaveScreenshot captures a screenshot and saves it to the specified path
// If path is empty, saves to temp directory with auto-generated filename
func captureAndSaveScreenshot(cmd *cobra.Command, path string) error {
	t := startTimer("screenshot")
	defer t.log()

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Read flags from command, falling back to parent for persistent flags
	fullPage, _ := cmd.Flags().GetBool("full-page")
	if !fullPage && cmd.Parent() != nil {
		fullPage, _ = cmd.Parent().PersistentFlags().GetBool("full-page")
	}

	debugParam("fullPage=%v path=%q", fullPage, path)

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

	debugRequest("screenshot", fmt.Sprintf("fullPage=%v", fullPage))
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "screenshot",
		Params: params,
	})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

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
	if path == "" {
		// No path provided - save to temp directory
		outputPath, err = generateScreenshotPath(exec)
		if err != nil {
			return outputError(err.Error())
		}
	} else {
		// Path provided - check if ends with separator (directory convention)
		if strings.HasSuffix(path, string(os.PathSeparator)) || strings.HasSuffix(path, "/") {
			// Path ends with separator - treat as directory, auto-generate filename
			filename, err := generateScreenshotFilename(exec)
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

	// Ensure parent directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return outputError(fmt.Sprintf("failed to create directory: %v", err))
	}

	// Write PNG data to file
	if err := os.WriteFile(outputPath, data.Data, 0644); err != nil {
		return outputError(fmt.Sprintf("failed to write screenshot: %v", err))
	}

	debugFile("wrote", outputPath, len(data.Data))

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
	filename, err := generateScreenshotFilename(exec)
	if err != nil {
		return "", err
	}
	return filepath.Join("/tmp/webctl-screenshots", filename), nil
}

// generateScreenshotFilename generates a filename using the pattern:
// YY-MM-DD-HHMMSS-{normalized-title}.png
func generateScreenshotFilename(exec executor.Executor) (string, error) {
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
	return fmt.Sprintf("%s-%s.png", timestamp, title), nil
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
