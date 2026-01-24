package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var keyCmd = &cobra.Command{
	Use:   "key <key>",
	Short: "Send a keyboard key",
	Long: `Sends a keyboard key to the focused element.

Supported special keys:
  Navigation:    Enter, Tab, Escape, Space
  Editing:       Backspace, Delete
  Arrows:        ArrowUp, ArrowDown, ArrowLeft, ArrowRight
  Page:          Home, End, PageUp, PageDown

Single character keys (a-z, A-Z, 0-9, punctuation) can be used directly.

Modifier flags (can be combined):
  --ctrl   Hold Ctrl modifier (Linux)
  --meta   Hold Meta/Cmd modifier (macOS)
  --alt    Hold Alt/Option modifier
  --shift  Hold Shift modifier

Examples:
  # Basic keys
  key Enter                    # Submit form / confirm
  key Tab                      # Move to next field
  key Escape                   # Close modal / cancel
  key Space                    # Toggle checkbox / click button

  # Text editing
  key Backspace                # Delete character before cursor
  key Delete                   # Delete character after cursor
  key a --ctrl                 # Select all (Linux)
  key a --meta                 # Select all (macOS)
  key z --ctrl                 # Undo (Linux)
  key z --meta                 # Undo (macOS)
  key z --ctrl --shift         # Redo (Linux)
  key z --meta --shift         # Redo (macOS)

  # Clipboard (requires browser permissions)
  key c --ctrl                 # Copy (Linux)
  key c --meta                 # Copy (macOS)
  key v --ctrl                 # Paste (Linux)
  key v --meta                 # Paste (macOS)
  key x --ctrl                 # Cut (Linux)
  key x --meta                 # Cut (macOS)

  # Navigation
  key ArrowDown                # Move down in list/menu
  key ArrowUp                  # Move up in list/menu
  key ArrowDown --shift        # Extend selection down
  key Home                     # Go to start of line/document
  key End                      # Go to end of line/document
  key PageDown                 # Scroll down one page
  key PageUp                   # Scroll up one page

  # Browser shortcuts
  key l --ctrl                 # Focus address bar (Linux)
  key l --meta                 # Focus address bar (macOS)
  key f --ctrl                 # Find in page (Linux)
  key f --meta                 # Find in page (macOS)`,
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
	t := startTimer("key")
	defer t.log()

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Read flags from command
	ctrl, _ := cmd.Flags().GetBool("ctrl")
	alt, _ := cmd.Flags().GetBool("alt")
	shift, _ := cmd.Flags().GetBool("shift")
	meta, _ := cmd.Flags().GetBool("meta")
	key := args[0]

	debugParam("key=%q ctrl=%v alt=%v shift=%v meta=%v", key, ctrl, alt, shift, meta)

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer func() { _ = exec.Close() }()

	params, err := json.Marshal(ipc.KeyParams{
		Key:   key,
		Ctrl:  ctrl,
		Alt:   alt,
		Shift: shift,
		Meta:  meta,
	})
	if err != nil {
		return outputError(err.Error())
	}

	debugRequest("key", fmt.Sprintf("key=%q ctrl=%v alt=%v shift=%v meta=%v", key, ctrl, alt, shift, meta))
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "key",
		Params: params,
	})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
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
