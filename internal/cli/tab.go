package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/grantcarthew/webctl/internal/cli/format"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var tabCmd = &cobra.Command{
	Use:   "tab",
	Short: "List, switch, create, or close browser tabs",
	Long: `Manage browser tabs.

Without a subcommand, lists all open tabs with their IDs, titles, and URLs.

Subcommands:
  switch <query>   Switch active tab and foreground it in the browser
  new [url]        Open a new tab (defaults to about:blank) and make it active
  close [query]    Close a tab (the active tab if no query)

Query matching (used by switch and close):
  - Session ID prefix (case-sensitive)
  - Title substring (case-insensitive)

Examples:
  webctl tab                    # List all tabs
  webctl tab switch 9A3E        # Switch by session ID prefix
  webctl tab switch example     # Switch by title substring
  webctl tab new                # Open about:blank
  webctl tab new example.com    # Open https://example.com
  webctl tab new localhost:3000 # Open http://localhost:3000
  webctl tab close              # Close the active tab
  webctl tab close example      # Close a tab matching the query`,
	Args: cobra.NoArgs,
	RunE: runTabList,
}

var tabSwitchCmd = &cobra.Command{
	Use:   "switch <query>",
	Short: "Switch to a tab and foreground it",
	Long: `Switch the active session to the matching tab and foreground it in the browser.

Query matching:
  - Session ID prefix (case-sensitive)
  - Title substring (case-insensitive)`,
	Args: cobra.ExactArgs(1),
	RunE: runTabSwitch,
}

var tabNewCmd = &cobra.Command{
	Use:   "new [url]",
	Short: "Open a new tab",
	Long: `Open a new tab and make it the active session.

URL protocol auto-detection (same rules as 'navigate'):
  - URLs without protocol get https:// added automatically
  - localhost, 127.0.0.1, 0.0.0.0 get http://
  - Explicit protocols (http://, https://, file://, about:, data:, ...) are preserved

If no URL is provided, opens about:blank.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTabNew,
}

var tabCloseCmd = &cobra.Command{
	Use:   "close [query]",
	Short: "Close a tab",
	Long: `Close a tab. If no query is given, closes the currently active tab.

If the closed tab was active, the most-recently-opened remaining tab becomes
active and is foregrounded.

Refuses to close the last remaining tab.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTabClose,
}

func init() {
	tabCmd.AddCommand(tabSwitchCmd, tabNewCmd, tabCloseCmd)
	rootCmd.AddCommand(tabCmd)
}

func runTabList(cmd *cobra.Command, args []string) error {
	t := startTimer("tab")
	defer t.log()

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer func() { _ = exec.Close() }()

	params, err := json.Marshal(ipc.TabParams{Action: "list"})
	if err != nil {
		return outputError(err.Error())
	}

	debugRequest("tab", "action=list")
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{Cmd: "tab", Params: params})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		return outputError(err.Error())
	}
	if !resp.OK {
		return outputError(resp.Error)
	}

	var data ipc.TabData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return outputError(err.Error())
	}

	if JSONOutput {
		return outputTabListJSON(data)
	}
	return format.Tab(os.Stdout, data, format.NewOutputOptions(JSONOutput, NoColor))
}

func runTabSwitch(cmd *cobra.Command, args []string) error {
	t := startTimer("tab switch")
	defer t.log()

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer func() { _ = exec.Close() }()

	query := args[0]
	params, err := json.Marshal(ipc.TabParams{Action: "switch", Query: query})
	if err != nil {
		return outputError(err.Error())
	}

	debugRequest("tab", "action=switch query="+query)
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{Cmd: "tab", Params: params})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		return outputError(err.Error())
	}
	if !resp.OK {
		return outputTabError(resp)
	}

	if JSONOutput {
		var data ipc.TabData
		_ = json.Unmarshal(resp.Data, &data)
		return outputJSON(os.Stdout, map[string]any{
			"ok":            true,
			"activeSession": data.ActiveSession,
		})
	}
	return outputSuccess(nil)
}

func runTabNew(cmd *cobra.Command, args []string) error {
	t := startTimer("tab new")
	defer t.log()

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer func() { _ = exec.Close() }()

	url := ""
	if len(args) == 1 {
		url = normalizeURL(args[0])
	}

	params, err := json.Marshal(ipc.TabParams{Action: "new", URL: url})
	if err != nil {
		return outputError(err.Error())
	}

	debugRequest("tab", "action=new url="+url)
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{Cmd: "tab", Params: params})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		return outputError(err.Error())
	}
	if !resp.OK {
		return outputError(resp.Error)
	}

	var data ipc.NewTabData
	_ = json.Unmarshal(resp.Data, &data)

	if JSONOutput {
		return outputJSON(os.Stdout, map[string]any{
			"ok":    true,
			"id":    data.ID,
			"url":   data.URL,
			"title": data.Title,
		})
	}
	return outputSuccess(nil)
}

func runTabClose(cmd *cobra.Command, args []string) error {
	t := startTimer("tab close")
	defer t.log()

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer func() { _ = exec.Close() }()

	query := ""
	if len(args) == 1 {
		query = args[0]
	}

	params, err := json.Marshal(ipc.TabParams{Action: "close", Query: query})
	if err != nil {
		return outputError(err.Error())
	}

	debugRequest("tab", "action=close query="+query)
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{Cmd: "tab", Params: params})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		return outputError(err.Error())
	}
	if !resp.OK {
		return outputTabError(resp)
	}

	if JSONOutput {
		var data ipc.TabData
		_ = json.Unmarshal(resp.Data, &data)
		return outputJSON(os.Stdout, map[string]any{
			"ok":            true,
			"activeSession": data.ActiveSession,
		})
	}
	return outputSuccess(nil)
}

// outputTabError handles error responses for switch/close, which may include
// candidate matches when the query is ambiguous.
func outputTabError(resp ipc.Response) error {
	var errData struct {
		Error    string            `json:"error,omitempty"`
		Sessions []ipc.PageSession `json:"sessions,omitempty"`
		Matches  []ipc.PageSession `json:"matches,omitempty"`
	}
	if len(resp.Data) > 0 {
		_ = json.Unmarshal(resp.Data, &errData)
	}

	if JSONOutput {
		out := map[string]any{
			"ok":    false,
			"error": resp.Error,
		}
		if len(errData.Matches) > 0 {
			out["matches"] = errData.Matches
		}
		if len(errData.Sessions) > 0 {
			out["sessions"] = errData.Sessions
		}
		_ = outputJSON(os.Stderr, out)
		return printedError{err: fmt.Errorf("%s", resp.Error)}
	}

	_ = format.TabError(os.Stderr, resp.Error, errData.Sessions, errData.Matches, format.NewOutputOptions(JSONOutput, NoColor))
	return printedError{err: fmt.Errorf("%s", resp.Error)}
}

// outputTabListJSON emits the tab list as JSON with full session IDs and titles,
// so JSON consumers can round-trip ids back into `tab switch` and friends. The
// text formatter (format.Tab) keeps its own truncation for display.
func outputTabListJSON(data ipc.TabData) error {
	sessions := make([]map[string]any, len(data.Sessions))
	for i, s := range data.Sessions {
		sessions[i] = map[string]any{
			"id":     s.ID,
			"title":  s.Title,
			"url":    s.URL,
			"active": s.ID == data.ActiveSession,
		}
	}
	return outputJSON(os.Stdout, map[string]any{
		"ok":            true,
		"activeSession": data.ActiveSession,
		"sessions":      sessions,
	})
}
