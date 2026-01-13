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
	"github.com/grantcarthew/webctl/internal/cssformat"
	"github.com/grantcarthew/webctl/internal/executor"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var cssCmd = &cobra.Command{
	Use:   "css",
	Short: "Extract CSS from current page (default: stdout)",
	Long: `Extracts CSS from the current page with flexible output modes.

Default behavior (no subcommand):
  Outputs CSS stylesheets to stdout for piping or inspection

Subcommands:
  save [path]       Save CSS to file (temp dir if no path given)
  computed <sel>    Get computed styles for element(s)
  get <sel> <prop>  Get single CSS property value
  inline <sel>      Get inline style attributes
  matched <sel>     Get matched CSS rules from stylesheets

Universal flags (work with default/save modes):
  --select, -s      Filter CSS rules by selector pattern
  --find, -f        Search for text within CSS
  --raw             Skip CSS formatting (return as-is from browser)
  --json            Output in JSON format (global flag)

Examples:

Default mode (stdout):
  css                                  # All stylesheets to stdout
  css --select "h1"                    # CSS rules with h1 selector
  css --find "background"              # Search and show matches

Save mode (file):
  css save                             # Save to temp with auto-filename
  css save ./styles.css                # Save to custom file
  css save ./output/                   # Save to dir (auto-filename)
  css save --select "button" --find "color"

Element-specific operations:
  css computed "h1"                    # Computed styles (all h1 elements)
  css get "#header" background-color   # Single property value
  css inline "[style]"                 # Inline style attributes
  css matched "#main"                  # Matched CSS rules for element

Response formats:
  Default:  body { margin: 0; ... } (to stdout)
  Save:     /tmp/webctl-css/25-12-28-143052-example.css
  Computed: property: value (multiple elements with -- separators)
  Get:      rgb(0,0,0) (to stdout)
  Inline:   style attribute content (multiple with -- separators)
  Matched:  /* selector */ property: value; (with -- separators)

Error cases:
  - "selector matched no elements" - nothing matches selector
  - "no CSS rules match selector" - no rules match --select pattern
  - "daemon not running" - start daemon first with: webctl start`,
	RunE: runCSSDefault,
}

var cssSaveCmd = &cobra.Command{
	Use:   "save [path]",
	Short: "Save CSS to file",
	Long: `Saves CSS to a file.

If no path is provided, saves to temp directory with auto-generated filename.
If path is a directory, auto-generates filename.
If path is a file, uses exact path.

Examples:
  css save                             # Save to temp dir
  css save ./styles.css                # Save to file
  css save ./output/                   # Save to dir
  css save --select "#app" --find "color"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCSSSave,
}

var cssComputedCmd = &cobra.Command{
	Use:   "computed <selector>",
	Short: "Get computed styles for element(s)",
	Long: `Gets all computed CSS styles for matching elements and outputs to stdout.

Returns all CSS properties computed by the browser for each matched element.
Multiple elements are separated by -- markers.

Flags:
  --json            Output in JSON format

Text format output (single element):
  display: flex
  background-color: rgb(255, 255, 255)
  width: 1200px
  margin: 0px

Text format output (multiple elements):
  display: block
  color: rgb(0, 0, 0)
  --
  display: inline
  color: rgb(255, 0, 0)

JSON format output:
  {
    "ok": true,
    "styles": [
      {"display": "flex", "background-color": "rgb(255, 255, 255)"},
      {"display": "block", "background-color": "rgb(0, 0, 0)"}
    ]
  }

Examples:
  css computed "#header"
  css computed ".button"
  css computed "h1"              # All h1 elements
  css computed "nav > ul" --json

Common patterns:
  # Debug element styles
  css computed "#main"

  # Get styles for all matching elements
  css computed "p"

  # Check computed values
  css computed ".button" --json | jq '.styles[0].display'`,
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

var cssInlineCmd = &cobra.Command{
	Use:   "inline <selector>",
	Short: "Get inline style attributes to stdout",
	Long: `Gets inline style attributes from matching elements and outputs to stdout.

Returns the raw style attribute content from each matching element.
Multiple elements are separated by -- markers.

Examples:
  css inline "div"
  css inline "[style]"
  css inline "#header"

Output (single element):
  color: red; font-size: 16px;

Output (multiple elements):
  color: red;
  --
  background: blue;
  --
  margin: 10px;

Common patterns:
  # Find all inline styles on page
  css inline "[style]"

  # Check specific element's inline styles
  css inline "#main"`,
	Args: cobra.ExactArgs(1),
	RunE: runCSSInline,
}

var cssMatchedCmd = &cobra.Command{
	Use:   "matched <selector>",
	Short: "Get matched CSS rules for element",
	Long: `Gets CSS rules from stylesheets that apply to the matched element.

Shows rules in specificity order, including the selector and properties.
Uses CDP CSS.getMatchedStylesForNode to get actual applied rules.

Examples:
  css matched "#header"
  css matched ".button"
  css matched "nav > ul"

Output:
  /* (inline) */
  color: red;
  font-weight: bold;
  --
  /* .header */
  background-color: white;
  padding: 10px;
  --
  /* body .header */
  margin: 0;

Common patterns:
  # Debug why styles are applied
  css matched "#main"

  # See all rules affecting an element
  css matched ".button" | grep background`,
	Args: cobra.ExactArgs(1),
	RunE: runCSSMatched,
}

func init() {
	// Universal flags on root command (inherited by default/save subcommands)
	cssCmd.PersistentFlags().StringP("select", "s", "", "Filter CSS rules by selector pattern")
	cssCmd.PersistentFlags().StringP("find", "f", "", "Search for text within CSS")
	cssCmd.PersistentFlags().IntP("before", "B", 0, "Show N lines before each match (requires --find)")
	cssCmd.PersistentFlags().IntP("after", "A", 0, "Show N lines after each match (requires --find)")
	cssCmd.PersistentFlags().IntP("context", "C", 0, "Show N lines before and after each match (requires --find)")
	cssCmd.PersistentFlags().Bool("raw", false, "Skip CSS formatting")

	// Add all subcommands
	cssCmd.AddCommand(cssSaveCmd, cssComputedCmd, cssGetCmd, cssInlineCmd, cssMatchedCmd)

	rootCmd.AddCommand(cssCmd)
}

// runCSSDefault handles default behavior: output to stdout
func runCSSDefault(cmd *cobra.Command, args []string) error {
	t := startTimer("css")
	defer t.log()

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
		if errors.Is(err, ErrNoMatches) {
			return outputNotice("No matches found")
		}
		if errors.Is(err, ErrNoElements) {
			return outputNotice("No elements found")
		}
		if errors.Is(err, ErrNoRules) {
			return outputNotice("No rules found")
		}
		return outputError(err.Error())
	}

	// JSON mode: output JSON
	if JSONOutput {
		result := map[string]any{
			"ok":  true,
			"css": css,
		}
		return outputJSON(os.Stdout, result)
	}

	// Output to stdout
	fmt.Println(css)
	return nil
}

// runCSSSave handles save subcommand: save to file
func runCSSSave(cmd *cobra.Command, args []string) error {
	t := startTimer("css save")
	defer t.log()

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Get CSS from daemon
	css, err := getCSSFromDaemon(cmd)
	if err != nil {
		if errors.Is(err, ErrNoMatches) {
			return outputNotice("No matches found")
		}
		if errors.Is(err, ErrNoElements) {
			return outputNotice("No elements found")
		}
		if errors.Is(err, ErrNoRules) {
			return outputNotice("No rules found")
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

		outputPath, err = generateCSSPath(exec, selector)
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

			filename, err := generateCSSFilename(exec, selector)
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

func runCSSComputed(cmd *cobra.Command, args []string) error {
	t := startTimer("css computed")
	defer t.log()

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
		if isNoElementsError(resp.Error) {
			return outputNotice("No elements found")
		}
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
			"styles": data.ComputedMulti,
		}
		return outputJSON(os.Stdout, result)
	}

	// Text mode: use multi-element formatter with -- separators
	return format.ComputedStylesMulti(os.Stdout, data.ComputedMulti)
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
		if isNoElementsError(resp.Error) {
			return outputNotice("No elements found")
		}
		if resp.Error == "property not found" {
			return outputNotice("Property not found")
		}
		if resp.Error == "no value" {
			return outputNotice("No value")
		}
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

func runCSSInline(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	params, err := json.Marshal(ipc.CSSParams{
		Action:   "inline",
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
		if isNoElementsError(resp.Error) {
			return outputNotice("No elements found")
		}
		return outputError(resp.Error)
	}

	// Parse CSS data
	var data ipc.CSSData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return outputError(err.Error())
	}

	// Check if all inline styles are empty
	allEmpty := true
	for _, style := range data.Inline {
		if style != "" {
			allEmpty = false
			break
		}
	}

	// JSON mode: output JSON
	if JSONOutput {
		result := map[string]any{
			"ok":     true,
			"inline": data.Inline,
		}
		return outputJSON(os.Stdout, result)
	}

	// If all inline styles are empty, show notice
	if allEmpty {
		return outputNotice("No inline styles")
	}

	// Text mode: output inline styles with -- separators
	return format.InlineStyles(os.Stdout, data.Inline)
}

func runCSSMatched(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	params, err := json.Marshal(ipc.CSSParams{
		Action:   "matched",
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
		if isNoElementsError(resp.Error) {
			return outputNotice("No elements found")
		}
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
			"ok":      true,
			"matched": data.Matched,
		}
		return outputJSON(os.Stdout, result)
	}

	// Check if no rules matched (element exists but only has user-agent styles)
	if len(data.Matched) == 0 {
		return outputNotice("No rules found")
	}

	// Text mode: output matched rules
	return format.MatchedRules(os.Stdout, data.Matched)
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
	params, err := json.Marshal(ipc.CSSParams{
		Action:   "save",
		Selector: selector,
	})
	if err != nil {
		return "", err
	}

	debugRequest("css", fmt.Sprintf("action=save selector=%q", selector))
	ipcStart := time.Now()

	// Execute CSS request
	resp, err := exec.Execute(ipc.Request{
		Cmd:    "css",
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

	// Parse CSS data
	var data ipc.CSSData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return "", err
	}

	css := data.CSS

	// Apply --select filter to CSS rules by selector
	if selector != "" {
		css = cssformat.FilterRulesBySelector(css, selector)
		if css == "" {
			return "", ErrNoRules
		}
	}

	// Format CSS unless --raw flag is set
	if !raw && selector == "" {
		// Only format full stylesheets when no selector filtering applied
		formatted, err := cssformat.Format(css)
		if err != nil {
			// If formatting fails, fall back to raw CSS
			debugf("FORMAT", "CSS formatting failed: %v", err)
		} else {
			css = formatted
		}
	}

	// Apply --find filter if specified (after formatting so line-based search works)
	if find != "" {
		beforeCount := strings.Count(css, "\n") + 1
		css, err = filterCSSByText(css, find, before, after)
		if err != nil {
			return "", err
		}
		afterCount := strings.Count(css, "\n") + 1
		debugFilter(fmt.Sprintf("--find %q", find), beforeCount, afterCount)
	}

	return css, nil
}

// filterCSSByText filters CSS to only include lines containing the search text
// with optional context lines before and after each match
func filterCSSByText(css, searchText string, before, after int) (string, error) {
	lines := strings.Split(css, "\n")
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
