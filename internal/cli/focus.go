package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var focusCmd = &cobra.Command{
	Use:   "focus <selector>",
	Short: "Focus an element",
	Long:  "Focuses an element matching the CSS selector.",
	Args:  cobra.ExactArgs(1),
	RunE:  runFocus,
}

func init() {
	rootCmd.AddCommand(focusCmd)
}

func runFocus(cmd *cobra.Command, args []string) error {
	t := startTimer("focus")
	defer t.log()

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	selector := args[0]
	debugParam("selector=%q", selector)

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer func() { _ = exec.Close() }()

	params, err := json.Marshal(ipc.FocusParams{
		Selector: selector,
	})
	if err != nil {
		return outputError(err.Error())
	}

	debugRequest("focus", fmt.Sprintf("selector=%q", selector))
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "focus",
		Params: params,
	})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		if isNoElementsError(resp.Error) {
			return outputNotice("No elements found")
		}
		return outputError(resp.Error)
	}

	// JSON mode: output JSON
	if JSONOutput {
		result := map[string]any{
			"ok": true,
		}
		return outputJSON(os.Stdout, result)
	}

	// Text mode: just output OK
	return outputSuccess(nil)
}
