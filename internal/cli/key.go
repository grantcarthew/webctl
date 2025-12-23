package cli

import (
	"encoding/json"
	"os"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var keyCmd = &cobra.Command{
	Use:   "key <key>",
	Short: "Send a keyboard key",
	Long: `Sends a keyboard key to the focused element.

Common keys: Enter, Tab, Escape, Backspace, Delete, ArrowUp, ArrowDown, ArrowLeft, ArrowRight, Home, End, PageUp, PageDown, Space

Single character keys can be used directly (e.g., "a", "A", "1").`,
	Args: cobra.ExactArgs(1),
	RunE: runKey,
}

func init() {
	keyCmd.Flags().Bool("ctrl", false, "Hold Ctrl modifier")
	keyCmd.Flags().Bool("alt", false, "Hold Alt modifier")
	keyCmd.Flags().Bool("shift", false, "Hold Shift modifier")
	keyCmd.Flags().Bool("meta", false, "Hold Meta/Command modifier")
	rootCmd.AddCommand(keyCmd)
}

func runKey(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Read flags from command
	ctrl, _ := cmd.Flags().GetBool("ctrl")
	alt, _ := cmd.Flags().GetBool("alt")
	shift, _ := cmd.Flags().GetBool("shift")
	meta, _ := cmd.Flags().GetBool("meta")

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	params, err := json.Marshal(ipc.KeyParams{
		Key:   args[0],
		Ctrl:  ctrl,
		Alt:   alt,
		Shift: shift,
		Meta:  meta,
	})
	if err != nil {
		return outputError(err.Error())
	}

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "key",
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
