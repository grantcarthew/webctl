package cli

import (
	"encoding/json"
	"os"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

// Named selectCmd_ to avoid collision with Go's select keyword
var selectCmd_ = &cobra.Command{
	Use:   "select <selector> <value>",
	Short: "Select a dropdown option",
	Long: `Selects an option in a native <select> dropdown element.

The value should match the option's value attribute.
Only works with native HTML <select> elements. For custom dropdowns, use click.`,
	Args: cobra.ExactArgs(2),
	RunE: runSelect,
}

func init() {
	rootCmd.AddCommand(selectCmd_)
}

func runSelect(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	params, err := json.Marshal(ipc.SelectParams{
		Selector: args[0],
		Value:    args[1],
	})
	if err != nil {
		return outputError(err.Error())
	}

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "select",
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
