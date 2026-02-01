package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var reconnectCmd = &cobra.Command{
	Use:   "reconnect",
	Short: "Reconnect to the browser",
	Long:  "Manually reconnect to the browser after a connection loss. Use when automatic reconnection fails or is disabled.",
	RunE:  runReconnect,
}

func init() {
	rootCmd.AddCommand(reconnectCmd)
}

func runReconnect(cmd *cobra.Command, args []string) error {
	t := startTimer("reconnect")
	defer t.log()

	// Check if daemon is running
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running (start with: webctl start)")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer func() { _ = exec.Close() }()

	debugRequest("reconnect", "")
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{Cmd: "reconnect"})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	// Parse response
	var result struct {
		Message        string `json:"message"`
		State          string `json:"state"`
		URL            string `json:"url,omitempty"`
		ReconnectCount int    `json:"reconnectCount,omitempty"`
	}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return outputError(err.Error())
	}

	// JSON mode: output full JSON
	if JSONOutput {
		return outputSuccess(result)
	}

	// Text mode: output message
	fmt.Fprintln(os.Stdout, result.Message)
	if result.URL != "" {
		fmt.Fprintf(os.Stdout, "url: %s\n", result.URL)
	}

	return nil
}
