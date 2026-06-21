package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/grantcarthew/webctl/internal/markdownformat"
	"github.com/spf13/cobra"
)

var markdownCmd = &cobra.Command{
	Use:     "markdown",
	Aliases: []string{"md"},
	Short:   "Extract current page as Markdown (default: stdout)",
	Long: `Converts the current page HTML to Markdown for token-efficient reading.

Default behavior (no subcommand):
  Outputs Markdown to stdout for piping or inspection

Subcommands:
  save [path]       Save Markdown to file (temp dir if no path given)

Universal flags (work with all modes):
  --select, -s      Convert only element(s) matching CSS selector
  --find, -f        Search for text within the Markdown
  --json            Output in JSON format (global flag)

Examples:

Default mode (stdout):
  markdown                              # Full page as Markdown
  md                                    # Same, via alias
  markdown --select "#main"             # Convert only the selected subtree
  markdown --find "install"             # Search and show matches

Save mode (file):
  markdown save                         # Save to temp with auto-filename
  markdown save ./page.md               # Save to custom file
  markdown save ./output/               # Save to dir (auto-filename)
  markdown save --select "article"

Response formats:
  Default:  # Heading ... (to stdout)
  Save:     /tmp/webctl-markdown/25-12-28-143052-123-example.md

Error cases:
  - "selector '.missing' matched no elements" - nothing matches
  - "No matches found" - find text not in Markdown
  - "daemon not running" - start daemon first with: webctl start`,
	RunE: runMarkdownDefault,
}

var markdownSaveCmd = &cobra.Command{
	Use:   "save [path]",
	Short: "Save Markdown to file",
	Long: `Saves the current page as Markdown to a file.

Path conventions:
  (no path)         Save to /tmp/webctl-markdown/ with auto-generated filename
  ./page.md         Save to exact file path
  ./output/         Save to directory with auto-generated filename (trailing slash required)

Examples:
  markdown save                         # Save to temp dir
  markdown save ./page.md               # Save to file
  markdown save ./output/               # Save to dir (creates if needed)
  markdown save --select "#app" --find "error"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMarkdownSave,
}

func init() {
	// Universal flags on root command (inherited by subcommands)
	markdownCmd.PersistentFlags().StringP("select", "s", "", "Convert only element(s) matching CSS selector")
	markdownCmd.PersistentFlags().StringP("find", "f", "", "Search for text within the Markdown")
	markdownCmd.PersistentFlags().IntP("before", "B", 0, "Show N lines before each match (requires --find)")
	markdownCmd.PersistentFlags().IntP("after", "A", 0, "Show N lines after each match (requires --find)")
	markdownCmd.PersistentFlags().IntP("context", "C", 0, "Show N lines before and after each match (requires --find)")

	markdownCmd.AddCommand(markdownSaveCmd)

	rootCmd.AddCommand(markdownCmd)
}

// runMarkdownDefault handles default behavior: output Markdown to stdout.
func runMarkdownDefault(cmd *cobra.Command, args []string) error {
	t := startTimer("markdown")
	defer t.log()

	// Validate that no arguments were provided (catches unknown subcommands)
	if len(args) > 0 {
		return outputError(fmt.Sprintf("unknown command %q for \"webctl markdown\"", args[0]))
	}

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	md, err := getMarkdownFromDaemon(cmd)
	if err != nil {
		if notice, ok := saveSentinelNotice(err); ok {
			return notice
		}
		return outputError(err.Error())
	}

	if JSONOutput {
		return outputJSON(os.Stdout, map[string]any{
			"ok":       true,
			"markdown": md,
		})
	}

	fmt.Println(md)
	return nil
}

// runMarkdownSave handles save subcommand: save Markdown to file.
func runMarkdownSave(cmd *cobra.Command, args []string) error {
	return runSave(cmd, args, saveSpec{
		timerLabel: "markdown save",
		tempDir:    "/tmp/webctl-markdown",
		ext:        "md",
		produce:    getMarkdownFromDaemon,
		identifier: selectorOrTitleIdentifier,
	})
}

// getMarkdownFromDaemon fetches the raw page HTML via the existing html IPC
// path, converts it to Markdown, then applies the line-based --find filter.
//
// It deliberately uses the raw element outerHTML rather than the html command's
// presentation output: no per-element identifier headers, no "--" separators,
// and no htmlformat prettifying. For a multi-match selector the raw outerHTML of
// each matched element is concatenated directly before conversion.
func getMarkdownFromDaemon(cmd *cobra.Command) (string, error) {
	html, err := getRawHTMLFromDaemon(cmd)
	if err != nil {
		return "", err
	}

	md, err := markdownformat.Convert(html)
	if err != nil {
		return "", fmt.Errorf("markdown conversion failed: %v", err)
	}

	// Apply --find filter to the Markdown output, reusing html's line filter.
	find, _ := cmd.Flags().GetString("find")
	if find == "" && cmd.Parent() != nil {
		find, _ = cmd.Parent().PersistentFlags().GetString("find")
	}
	if find != "" {
		before, after := findContext(cmd)
		beforeCount := strings.Count(md, "\n") + 1
		md, err = filterHTMLByText(md, find, before, after)
		if err != nil {
			return "", err
		}
		afterCount := strings.Count(md, "\n") + 1
		debugFilter(fmt.Sprintf("--find %q", find), beforeCount, afterCount)
	}

	return md, nil
}

// getRawHTMLFromDaemon fetches the live page HTML over the html IPC request and
// returns the raw, unformatted outerHTML. For a multi-element selector match it
// concatenates each element's outerHTML directly, without headers or separators.
func getRawHTMLFromDaemon(cmd *cobra.Command) (string, error) {
	selector, _ := cmd.Flags().GetString("select")
	if selector == "" && cmd.Parent() != nil {
		selector, _ = cmd.Parent().PersistentFlags().GetString("select")
	}

	debugParam("selector=%q", selector)

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return "", err
	}
	defer func() { _ = exec.Close() }()

	params, err := json.Marshal(ipc.HTMLParams{Selector: selector})
	if err != nil {
		return "", err
	}

	debugRequest("html", fmt.Sprintf("selector=%q", selector))
	ipcStart := time.Now()

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

	var data ipc.HTMLData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return "", err
	}

	if len(data.HTMLMulti) > 0 {
		var b strings.Builder
		for _, elem := range data.HTMLMulti {
			b.WriteString(elem.HTML)
		}
		return b.String(), nil
	}

	return data.HTML, nil
}

// findContext resolves the -B/-A/-C context flags, with -C overriding -B/-A.
func findContext(cmd *cobra.Command) (before, after int) {
	before, _ = cmd.Flags().GetInt("before")
	if before == 0 && cmd.Parent() != nil {
		before, _ = cmd.Parent().PersistentFlags().GetInt("before")
	}

	after, _ = cmd.Flags().GetInt("after")
	if after == 0 && cmd.Parent() != nil {
		after, _ = cmd.Parent().PersistentFlags().GetInt("after")
	}

	context, _ := cmd.Flags().GetInt("context")
	if context == 0 && cmd.Parent() != nil {
		context, _ = cmd.Parent().PersistentFlags().GetInt("context")
	}
	if context > 0 {
		before = context
		after = context
	}
	return before, after
}
