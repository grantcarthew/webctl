package cli

import (
	"encoding/json"
	"os"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var typeCmd = &cobra.Command{
	Use:   "type [selector] <text>",
	Short: "Type text into an element",
	Long: `Types text into an element. If selector is provided, focuses the element first.

With one argument: types into the currently focused element.
With two arguments: focuses the element matching the selector, then types.

Use --key to send a key after typing (e.g., Enter to submit a form).
Use --clear to clear existing content before typing.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runType,
}

var (
	typeKey   string
	typeClear bool
)

func init() {
	typeCmd.Flags().StringVar(&typeKey, "key", "", "Key to send after typing (e.g., Enter)")
	typeCmd.Flags().BoolVar(&typeClear, "clear", false, "Clear existing content before typing")
	rootCmd.AddCommand(typeCmd)
}

func runType(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	var selector, text string
	if len(args) == 1 {
		text = args[0]
	} else {
		selector = args[0]
		text = args[1]
	}

	params, err := json.Marshal(ipc.TypeParams{
		Selector: selector,
		Text:     text,
		Key:      typeKey,
		Clear:    typeClear,
	})
	if err != nil {
		return outputError(err.Error())
	}

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "type",
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
