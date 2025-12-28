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
	Short: "CSS extraction, inspection, and manipulation",
	Long: `Extract, inspect, and manipulate CSS in the browser.

Subcommands:
  css save [selector]           Extract and save CSS to file
  css computed <selector>       Get computed styles to stdout
  css get <selector> <property> Get single CSS property to stdout
  css inject <css>              Inject CSS into page

Extract all stylesheets:
  css save
  css save -o ./styles.css
  css save --raw                # Unformatted

Extract computed styles for element:
  css save "#header"
  css save ".button" -o ./button-styles.css

Get all computed styles:
  css computed "#main"
  css computed ".button" --json

Get single property:
  css get "#header" background-color
  css get ".button" display

Inject CSS:
  css inject "body { background: red; }"
  css inject --file ./custom.css

Response:
  save:     {"ok": true, "path": "/tmp/webctl-css/..."}
  computed: Styles output to stdout
  get:      Property value to stdout
  inject:   {"ok": true}

Error cases:
  - "selector matched no elements" - nothing matches selector
  - "property does not exist" - invalid CSS property
  - "daemon not running" - start daemon first with: webctl start`,
}

var cssSaveCmd = &cobra.Command{
	Use:   "save [selector]",
	Short: "Extract and save CSS to file",
	Long: `Extracts CSS and saves it to a file.

Without a selector, returns all stylesheets (inline, style tags, linked).
With a selector, returns computed styles for the matched element.

Flags:
  --output, -o      Save to specified path instead of temp directory
  --raw             Save unformatted CSS (default: formatted for readability)

File location:
  Default: /tmp/webctl-css/YY-MM-DD-HHMMSS-{title}.css
  Custom:  Specified path with --output flag

Extract all stylesheets:
  css save                                  # All CSS from page
  css save -o ./debug/styles.css            # Save to specific location
  css save --raw                            # Unformatted CSS

Extract computed styles for element:
  css save "#main"                          # Element by ID
  css save ".content"                       # Element by class
  css save "nav > ul"                       # Nested selector

Common patterns:
  # Debug element styling
  navigate example.com --wait
  css save "#header" -o ./header.css

  # Extract all page styles
  css save -o ./site-styles.css

  # Compare styles before/after
  css save ".button" -o ./before.css
  click "#toggle-theme"
  ready
  css save ".button" -o ./after.css

Response:
  {"ok": true, "path": "/tmp/webctl-css/24-12-28-143052-example-domain.css"}

Error cases:
  - "selector matched no elements" - nothing matches
  - "failed to write CSS: permission denied" - cannot write to path`,
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

var cssInjectCmd = &cobra.Command{
	Use:   "inject <css>",
	Short: "Inject CSS into page",
	Long: `Injects CSS into the current page.

The CSS is added as a style tag in the document head and persists
until the page is reloaded. Useful for testing and visual debugging.

Flags:
  --file, -f        Inject CSS from file instead of inline

Inline CSS:
  css inject "body { background: red; }"
  css inject ".ads { display: none !important; }"

CSS from file:
  css inject --file ./custom.css
  css inject -f ./dark-mode.css

Common patterns:
  # Hide elements for screenshots
  css inject ".ads { display: none !important; }"
  screenshot -o ./clean.png

  # Test responsive styles
  css inject ".container { max-width: 768px; }"
  screenshot

  # Apply dark mode for testing
  css inject --file ./dark-theme.css
  screenshot

  # Override vendor styles
  css inject "button { border-radius: 0 !important; }"

Response:
  {"ok": true}

Note: Injected CSS is temporary and removed on page reload.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCSSInject,
}

func init() {
	// Flags for save subcommand
	cssSaveCmd.Flags().StringP("output", "o", "", "Save to specified path instead of temp directory")
	cssSaveCmd.Flags().Bool("raw", false, "Save unformatted CSS (default: formatted for readability)")

	// Flags for inject subcommand
	cssInjectCmd.Flags().StringP("file", "f", "", "Inject CSS from file")

	cssCmd.AddCommand(cssSaveCmd)
	cssCmd.AddCommand(cssComputedCmd)
	cssCmd.AddCommand(cssGetCmd)
	cssCmd.AddCommand(cssInjectCmd)
	rootCmd.AddCommand(cssCmd)
}

func runCSSSave(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Read flags
	output, _ := cmd.Flags().GetString("output")
	rawOutput, _ := cmd.Flags().GetBool("raw")

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

	params, err := json.Marshal(ipc.CSSParams{
		Action:   "save",
		Selector: selector,
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

	// Format CSS unless --raw flag is set
	var cssOutput string
	if selector == "" {
		// All stylesheets - data.CSS contains the CSS
		cssOutput = data.CSS
		if !rawOutput {
			formatted, err := cssformat.Format(data.CSS)
			if err != nil {
				// If formatting fails, fall back to raw CSS
				debugf("CSS formatting failed: %v", err)
			} else {
				cssOutput = formatted
			}
		}
	} else {
		// Computed styles - data.Styles contains the map
		cssOutput = cssformat.FormatComputedStyles(data.Styles)
	}

	// Determine output path
	var outputPath string
	if output != "" {
		outputPath = output
	} else {
		// Generate filename in temp directory
		outputPath, err = generateCSSPath(exec, selector)
		if err != nil {
			return outputError(err.Error())
		}
	}

	// Ensure parent directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return outputError(fmt.Sprintf("failed to create directory: %v", err))
	}

	// Write CSS to file
	if err := os.WriteFile(outputPath, []byte(cssOutput), 0644); err != nil {
		return outputError(fmt.Sprintf("failed to write CSS: %v", err))
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

func runCSSInject(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	// Read flags
	filePath, _ := cmd.Flags().GetString("file")

	// Either file or inline CSS must be provided
	if filePath == "" && len(args) == 0 {
		return outputError("either provide CSS inline or use --file flag")
	}

	var cssContent string
	if len(args) > 0 {
		cssContent = args[0]
	}

	params, err := json.Marshal(ipc.CSSParams{
		Action: "inject",
		CSS:    cssContent,
		File:   filePath,
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

	// JSON mode: output JSON
	if JSONOutput {
		return outputJSON(os.Stdout, map[string]any{"ok": true})
	}

	// Text mode: just output OK
	return outputSuccess(nil)
}

// generateCSSPath generates a filename in /tmp/webctl-css/
// using the pattern: YY-MM-DD-HHMMSS-{normalized-title}.css
// For selectors, uses normalized selector as identifier
func generateCSSPath(exec executor.Executor, selector string) (string, error) {
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

	// Generate identifier
	var identifier string
	if selector != "" {
		// Use normalized selector
		identifier = normalizeSelector(selector)
	} else {
		// Use page title
		identifier = "untitled"
		if status.ActiveSession != nil && status.ActiveSession.Title != "" {
			identifier = normalizeTitle(status.ActiveSession.Title)
		}
	}

	// Generate timestamp: YY-MM-DD-HHMMSS
	now := time.Now()
	timestamp := now.Format("06-01-02-150405")

	// Generate filename
	filename := fmt.Sprintf("%s-%s.css", timestamp, identifier)

	// Return path in /tmp/webctl-css/
	return filepath.Join("/tmp/webctl-css", filename), nil
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
