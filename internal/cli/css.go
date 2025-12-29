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
	"github.com/grantcarthew/webctl/internal/cssformat"
	"github.com/grantcarthew/webctl/internal/executor"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var cssCmd = &cobra.Command{
	Use:   "css",
	Short: "Extract CSS from current page (default: save to temp)",
	Long: `Extracts CSS from the current page with flexible output modes.

Default behavior (no subcommand):
  Saves CSS to /tmp/webctl-css/ with auto-generated filename
  Returns JSON with file path

Subcommands:
  show              Output CSS to stdout
  save <path>       Save CSS to custom path
  computed <sel>    Get computed styles to stdout
  get <sel> <prop>  Get single CSS property to stdout

Universal flags (work with default/show/save modes):
  --select, -s      Filter to element's computed styles
  --find, -f        Search for text within CSS
  --raw             Skip CSS formatting (return as-is from browser)
  --json            Output in JSON format (global flag)

Examples:

Default mode (save to temp):
  css                                  # All stylesheets to temp
  css --select "#header"               # Computed styles to temp
  css --find "background"              # Search and save matches

Show mode (stdout):
  css show                             # All stylesheets to stdout
  css show --select ".button"          # Computed styles to stdout
  css show --find "color"              # Search and show matches

Save mode (custom path):
  css save ./styles.css                # Save to file
  css save ./output/                   # Save to dir (auto-filename)
  css save ./debug.css --select "form" --find "border"

CSS-specific operations:
  css computed "#main"                 # All computed styles
  css get "#header" background-color   # Single property

Response formats:
  Default/Save: {"ok": true, "path": "/tmp/webctl-css/25-12-28-143052-example.css"}
  Show:         body { margin: 0; ... } (to stdout)
  Computed:     display: flex\ncolor: rgb(0,0,0) (to stdout)
  Get:          rgb(0,0,0) (to stdout)

Error cases:
  - "selector matched no elements" - nothing matches selector
  - "property does not exist" - invalid CSS property
  - "daemon not running" - start daemon first with: webctl start`,
	RunE: runCSSDefault,
}

var cssShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Output CSS to stdout",
	Long: `Outputs CSS to stdout for piping or inspection.

Examples:
  css show                             # All stylesheets
  css show --select "#main"            # Computed styles
  css show --find "background"         # Search within CSS
  css show --raw                       # Unformatted output`,
	RunE: runCSSShow,
}

var cssSaveCmd = &cobra.Command{
	Use:   "save <path>",
	Short: "Save CSS to custom path",
	Long: `Saves CSS to a custom file path.

If path is a directory, auto-generates filename.
If path is a file, uses exact path.

Examples:
  css save ./styles.css                # Save to file
  css save ./output/                   # Save to dir
  css save ./debug.css --select "#app" --find "color"`,
	Args: cobra.ExactArgs(1),
	RunE: runCSSSave,
}

var cssComputedCmd = &cobra.Command{
	Use:   "computed <selector>",
	Short: "Get computed styles to stdout",
	Long: `Gets all computed CSS styles for a selector and outputs to stdout.

Returns all CSS properties computed by the browser for the matched element.

Flags:
  --json            Output in JSON format

Text format output:
  display: flex
  background-color: rgb(255, 255, 255)
  width: 1200px
  margin: 0px

JSON format output:
  {
    "ok": true,
    "styles": {
      "display": "flex",
      "background-color": "rgb(255, 255, 255)",
      "width": "1200px",
      "margin": "0px"
    }
  }

Examples:
  css computed "#header"
  css computed ".button"
  css computed "nav > ul" --json

Common patterns:
  # Debug element styles
  css computed "#main"

  # Verify responsive styles
  navigate example.com --wait
  css computed ".hero" | grep width

  # Check computed values
  css computed ".button" --json | jq '.styles.display'`,
	Args: cobra.ExactArgs(1),
	RunE: runCSSComputed,
}

var cssGetCmd = &cobra.Command{
	Use:   "get <selector> <property>",
	Short: "Get single CSS property to stdout",
	Long: `Gets a single computed CSS property value and outputs to stdout.

Returns just the property value - perfect for scripting and automation.

Examples:
  css get "#header" background-color
  css get ".button" display
  css get "body" font-size

Output:
  rgb(0, 0, 0)
  flex
  16px

Common patterns:
  # Check if element is visible
  if [ "$(css get '.modal' display)" = "none" ]; then
    echo "Modal is hidden"
  fi

  # Verify color
  css get "#header" background-color

  # Get width for calculations
  WIDTH=$(css get ".container" width)
  echo "Container width: $WIDTH"`,
	Args: cobra.ExactArgs(2),
	RunE: runCSSGet,
}


func init() {
	// Universal flags on root command (inherited by default/show/save subcommands)
	cssCmd.PersistentFlags().StringP("select", "s", "", "Filter to element's computed styles")
	cssCmd.PersistentFlags().StringP("find", "f", "", "Search for text within CSS")
	cssCmd.PersistentFlags().Bool("raw", false, "Skip CSS formatting")

	// Add all subcommands
	cssCmd.AddCommand(cssShowCmd, cssSaveCmd, cssComputedCmd, cssGetCmd)

	rootCmd.AddCommand(cssCmd)
}

// runCSSDefault handles default behavior: save to temp directory
func runCSSDefault(cmd *cobra.Command, args []string) error {
	// Validate that no arguments were provided (catches unknown subcommands)
	if len(args) > 0 {
		return outputError(fmt.Sprintf("unknown command %q for \"webctl css\"", args[0]))
	}

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Get CSS from daemon
	css, err := getCSSFromDaemon(cmd)
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

	outputPath, err := generateCSSPath(exec, selector)
	if err != nil {
		return outputError(err.Error())
	}

	// Write CSS to file
	if err := writeCSSToFile(outputPath, css); err != nil {
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

// runCSSShow handles show subcommand: output to stdout
func runCSSShow(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Get CSS from daemon
	css, err := getCSSFromDaemon(cmd)
	if err != nil {
		return outputError(err.Error())
	}

	// Output to stdout
	fmt.Println(css)
	return nil
}

// runCSSSave handles save subcommand: save to custom path
func runCSSSave(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	path := args[0]

	// Get CSS from daemon
	css, err := getCSSFromDaemon(cmd)
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
		filename, err := generateCSSFilename(exec, selector)
		if err != nil {
			return outputError(err.Error())
		}
		path = filepath.Join(path, filename)
	}

	// Write CSS to file
	if err := writeCSSToFile(path, css); err != nil {
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

func runCSSComputed(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	params, err := json.Marshal(ipc.CSSParams{
		Action:   "computed",
		Selector: args[0],
	})
	if err != nil {
		return outputError(err.Error())
	}

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "css",
		Params: params,
	})
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	// Parse CSS data
	var data ipc.CSSData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return outputError(err.Error())
	}

	// JSON mode: output JSON
	if JSONOutput {
		result := map[string]any{
			"ok":     true,
			"styles": data.Styles,
		}
		return outputJSON(os.Stdout, result)
	}

	// Text mode: use text formatter
	return format.ComputedStyles(os.Stdout, data.Styles)
}

func runCSSGet(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	params, err := json.Marshal(ipc.CSSParams{
		Action:   "get",
		Selector: args[0],
		Property: args[1],
	})
	if err != nil {
		return outputError(err.Error())
	}

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "css",
		Params: params,
	})
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	// Parse CSS data
	var data ipc.CSSData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return outputError(err.Error())
	}

	// JSON mode: output JSON
	if JSONOutput {
		result := map[string]any{
			"ok":    true,
			"value": data.Value,
		}
		return outputJSON(os.Stdout, result)
	}

	// Text mode: just output the value
	return format.PropertyValue(os.Stdout, data.Value)
}


// getCSSFromDaemon fetches CSS from daemon, applying filters and formatting
func getCSSFromDaemon(cmd *cobra.Command) (string, error) {
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
	params, err := json.Marshal(ipc.CSSParams{
		Action:   "save",
		Selector: selector,
	})
	if err != nil {
		return "", err
	}

	// Execute CSS request
	resp, err := exec.Execute(ipc.Request{
		Cmd:    "css",
		Params: params,
	})
	if err != nil {
		return "", err
	}

	if !resp.OK {
		return "", fmt.Errorf("%s", resp.Error)
	}

	// Parse CSS data
	var data ipc.CSSData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return "", err
	}

	var css string
	if selector == "" {
		// All stylesheets - data.CSS contains the CSS
		css = data.CSS
	} else {
		// Computed styles - data.Styles contains the map
		css = cssformat.FormatComputedStyles(data.Styles)
	}

	// Apply --find filter if specified
	if find != "" {
		css, err = filterCSSByText(css, find)
		if err != nil {
			return "", err
		}
	}

	// Format CSS unless --raw flag is set
	if !raw && selector == "" {
		// Only format full stylesheets, not computed styles
		formatted, err := cssformat.Format(css)
		if err != nil {
			// If formatting fails, fall back to raw CSS
			debugf("CSS formatting failed: %v", err)
		} else {
			css = formatted
		}
	}

	return css, nil
}

// filterCSSByText filters CSS to only include lines containing the search text
func filterCSSByText(css, searchText string) (string, error) {
	lines := strings.Split(css, "\n")
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

// writeCSSToFile writes CSS content to a file, creating directories if needed
func writeCSSToFile(path, css string) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Write CSS to file
	if err := os.WriteFile(path, []byte(css), 0644); err != nil {
		return fmt.Errorf("failed to write CSS: %v", err)
	}

	return nil
}

// generateCSSPath generates a full path in /tmp/webctl-css/
// using the pattern: YY-MM-DD-HHMMSS-{identifier}.css
func generateCSSPath(exec executor.Executor, selector string) (string, error) {
	filename, err := generateCSSFilename(exec, selector)
	if err != nil {
		return "", err
	}

	return filepath.Join("/tmp/webctl-css", filename), nil
}

// generateCSSFilename generates a filename using the pattern:
// YY-MM-DD-HHMMSS-{identifier}.css
// Identifier is based on selector (if provided) or page title
func generateCSSFilename(exec executor.Executor, selector string) (string, error) {
	// Generate timestamp: YY-MM-DD-HHMMSS
	now := time.Now()
	timestamp := now.Format("06-01-02-150405")

	// Get identifier (selector or page title)
	identifier := "untitled"
	if selector != "" {
		// Use normalized selector
		identifier = normalizeSelector(selector)
	} else {
		// Get page title
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

		if status.ActiveSession != nil && status.ActiveSession.Title != "" {
			identifier = normalizeTitle(status.ActiveSession.Title)
		}
	}

	// Generate filename
	return fmt.Sprintf("%s-%s.css", timestamp, identifier), nil
}

// normalizeSelector normalizes a CSS selector for use in a filename.
// Removes special characters and converts to lowercase.
func normalizeSelector(selector string) string {
	// Remove leading/trailing whitespace
	selector = strings.TrimSpace(selector)

	// Limit to 30 characters
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

	// Fallback to "element" if empty after normalization
	if selector == "" {
		selector = "element"
	}

	return selector
}
