package cli

import (
	"encoding/json"
	"os"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var backCmd = &cobra.Command{
	Use:   "back",
	Short: "Navigate to previous page",
	Long:  "Navigates to the previous page in browser history. Returns an error if there is no previous page.",
	Args:  cobra.NoArgs,
	RunE:  runBack,
}

func init() {
	rootCmd.AddCommand(backCmd)
}

func runBack(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	resp, err := exec.Execute(ipc.Request{
		Cmd: "back",
	})
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	var data ipc.NavigateData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return outputError(err.Error())
	}

	result := map[string]any{
		"ok":    true,
		"url":   data.URL,
		"title": data.Title,
	}
	return outputJSON(os.Stdout, result)
}
