package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/grantcarthew/webctl/internal/cli/format"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var findCmd = &cobra.Command{
	Use:   "find <text>",
	Short: "Search HTML content for text patterns",
	Long: `Search raw HTML content for text patterns and show context around matches.

Without flags, performs case-insensitive text search.
Use -E for regex patterns, -c for case-sensitive search.

Examples:
  webctl find "login"                    # Find "login" (case-insensitive)
  webctl find -c "Login"                 # Case-sensitive search
  webctl find -E "sign\s*up|register"    # Regex pattern
  webctl find --limit 5 "button"         # First 5 matches only

Output shows one line before and after each match, with the matching line
prefixed with ">". Matched text is highlighted in yellow.

JSON output includes CSS selector and XPath for each match, enabling
piping to click/type commands:

  webctl find --json "submit" | jq -r '.matches[0].selector' | xargs webctl click

Requirements:
  - Query must be at least 3 characters
  - Daemon must be running (webctl start)
  - Active browser session required`,
	Args: cobra.ExactArgs(1),
	RunE: runFind,
}

func init() {
	findCmd.Flags().BoolP("regex", "E", false, "Treat query as regex pattern")
	findCmd.Flags().BoolP("case-sensitive", "c", false, "Case-sensitive search (plain text only)")
	findCmd.Flags().IntP("limit", "l", 0, "Limit number of matches (default: all)")
	rootCmd.AddCommand(findCmd)
}

func runFind(cmd *cobra.Command, args []string) error {
	t := startTimer("find")
	defer t.log()

	// Get query from args
	query := args[0]

	// Validate minimum query length (before daemon check, per DR-017)
	if len(query) < 3 {
		return outputError("query must be at least 3 characters")
	}

	// Read flags from command
	isRegex, _ := cmd.Flags().GetBool("regex")
	caseSensitive, _ := cmd.Flags().GetBool("case-sensitive")
	limit, _ := cmd.Flags().GetInt("limit")

	// Validate case-sensitive flag only applies to plain text
	if caseSensitive && isRegex {
		return outputError("--case-sensitive flag cannot be used with --regex (use regex flags like (?i) instead)")
	}

	// Validate regex compilation if regex mode (before daemon check)
	if isRegex {
		if _, err := regexp.Compile(query); err != nil {
			return outputError(fmt.Sprintf("invalid regex pattern: %v", err))
		}
	}

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	debugParam("query=%q regex=%v caseSensitive=%v limit=%d", query, isRegex, caseSensitive, limit)

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	// Build request
	params, err := json.Marshal(ipc.FindParams{
		Query:         query,
		Regex:         isRegex,
		CaseSensitive: caseSensitive,
		Limit:         limit,
	})
	if err != nil {
		return outputError(err.Error())
	}

	debugRequest("find", fmt.Sprintf("query=%q regex=%v limit=%d", query, isRegex, limit))
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "find",
		Params: params,
	})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	// Parse find data
	var data ipc.FindData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return outputError(err.Error())
	}

	// JSON mode: output JSON
	if JSONOutput {
		return outputFindJSON(data)
	}

	// Text mode: use text formatter
	return format.Find(os.Stdout, data, format.NewOutputOptions(JSONOutput, NoColor))
}

// outputFindJSON outputs find results in JSON format.
func outputFindJSON(data ipc.FindData) error {
	resp := map[string]any{
		"ok":      true,
		"query":   data.Query,
		"total":   data.Total,
		"matches": data.Matches,
	}
	return outputJSON(os.Stdout, resp)
}
