package cli

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var evalCmd = &cobra.Command{
	Use:   "eval <expression>",
	Short: "Evaluate JavaScript in the browser",
	Long:  "Evaluates a JavaScript expression in the current page context and returns the result.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runEval,
}

func init() {
	evalCmd.Flags().DurationP("timeout", "t", 30*time.Second, "Timeout for async expressions")
	rootCmd.AddCommand(evalCmd)
}

func runEval(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Read flags from command
	timeout, _ := cmd.Flags().GetDuration("timeout")

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	// Join all args to form the expression (allows shell-friendly use without quotes)
	expression := strings.Join(args, " ")

	params, err := json.Marshal(ipc.EvalParams{
		Expression: expression,
		Timeout:    int(timeout.Milliseconds()),
	})
	if err != nil {
		return outputError(err.Error())
	}

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "eval",
		Params: params,
	})
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	// Parse the response data
	var data ipc.EvalData
	if len(resp.Data) > 0 {
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return outputError(err.Error())
		}
	}

	// Build result - only include value if it was present (not undefined)
	result := map[string]any{
		"ok": true,
	}
	if data.HasValue {
		result["value"] = data.Value
	}

	return outputJSON(os.Stdout, result)
}
