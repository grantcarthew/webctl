package cli

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var targetCmd = &cobra.Command{
	Use:   "target [query]",
	Short: "List or switch page sessions",
	Long: `List all page sessions or switch to a specific session.

Without arguments, lists all active sessions with their IDs, titles, and URLs.
With a query argument, switches to the matching session.

Query matching:
  - Session ID prefix (case-sensitive)
  - Title substring (case-insensitive)

Examples:
  webctl target           # List all sessions
  webctl target 9A3E      # Switch by session ID prefix
  webctl target example   # Switch by title substring`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTarget,
}

func init() {
	rootCmd.AddCommand(targetCmd)
}

func runTarget(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	query := ""
	if len(args) > 0 {
		query = args[0]
	}

	resp, err := exec.Execute(ipc.Request{Cmd: "target", Target: query})
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		// Check if error includes matches/sessions data
		if resp.Data != nil {
			// Output the full response with error and data
			return outputTargetError(resp)
		}
		return outputError(resp.Error)
	}

	var data ipc.TargetData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return outputError(err.Error())
	}

	return outputTargetJSON(data)
}

// outputTargetError outputs an error response that includes session data.
func outputTargetError(resp ipc.Response) error {
	// Parse the error response to get structured output
	var errData struct {
		Error    string            `json:"error,omitempty"`
		Sessions []ipc.PageSession `json:"sessions,omitempty"`
		Matches  []ipc.PageSession `json:"matches,omitempty"`
	}

	if resp.Data != nil {
		_ = json.Unmarshal(resp.Data, &errData)
	}

	output := map[string]any{
		"ok":    false,
		"error": resp.Error,
	}

	if len(errData.Sessions) > 0 {
		output["sessions"] = errData.Sessions
	}
	if len(errData.Matches) > 0 {
		output["matches"] = errData.Matches
	}

	return outputJSON(os.Stdout, output)
}

// outputTargetJSON outputs target data in JSON format.
func outputTargetJSON(data ipc.TargetData) error {
	output := map[string]any{
		"ok":            true,
		"activeSession": data.ActiveSession,
		"sessions":      formatSessions(data.Sessions, data.ActiveSession),
	}
	return outputJSON(os.Stdout, output)
}

// formatSessions formats sessions with truncated IDs for display.
func formatSessions(sessions []ipc.PageSession, activeID string) []map[string]any {
	result := make([]map[string]any, len(sessions))
	for i, s := range sessions {
		result[i] = map[string]any{
			"id":     truncateID(s.ID, 8),
			"title":  truncateTitle(s.Title, 40),
			"url":    s.URL,
			"active": s.ID == activeID,
		}
	}
	return result
}

// truncateID returns first n characters of an ID.
func truncateID(id string, n int) string {
	if len(id) <= n {
		return id
	}
	return id[:n] + "..."
}

// truncateTitle truncates a title to max length.
func truncateTitle(title string, max int) string {
	title = strings.TrimSpace(title)
	if len(title) <= max {
		return title
	}
	return title[:max-3] + "..."
}
