package cli

import (
	"encoding/json"
	"os"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var clickCmd = &cobra.Command{
	Use:   "click <selector>",
	Short: "Click an element",
	Long:  "Clicks an element matching the CSS selector. Uses CDP mouse events for true click simulation.",
	Args:  cobra.ExactArgs(1),
	RunE:  runClick,
}

func init() {
	rootCmd.AddCommand(clickCmd)
}

func runClick(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	params, err := json.Marshal(ipc.ClickParams{
		Selector: args[0],
	})
	if err != nil {
		return outputError(err.Error())
	}

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "click",
		Params: params,
	})
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	result := map[string]any{
		"ok": true,
	}
	return outputJSON(os.Stdout, result)
}
