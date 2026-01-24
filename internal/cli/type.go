package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var typeCmd = &cobra.Command{
	Use:   "type [selector] <text>",
	Short: "Type text into an element",
	Long: `Types text into an element using CDP keyboard input simulation.

With one argument: types into the currently focused element.
With two arguments: focuses the element matching the selector, then types.

Flags:
  --key <key>     Send a key after typing (e.g., Enter, Tab)
  --clear         Clear existing content before typing (select all + delete)

The --clear flag is OS-aware:
  - macOS: Uses Cmd+A (Meta+A) to select all
  - Linux: Uses Ctrl+A to select all

Selector examples:
  type "#username" "john_doe"           # Type into element by ID
  type ".search-input" "query"          # Type into element by class
  type "input[name=email]" "a@b.com"    # Type into element by name
  type "[data-testid=search]" "test"    # Type into element by test ID

Without selector (types into focused element):
  focus "#input"
  type "hello world"                    # Types into already-focused element

With --key flag (send key after typing):
  type "#search" "query" --key Enter    # Type and submit
  type "#field1" "value" --key Tab      # Type and move to next field

With --clear flag (replace existing content):
  type "#email" "new@email.com" --clear # Clear first, then type

Combined flags:
  type "#search" "new query" --clear --key Enter

Given this HTML:
  <form id="login">
    <input type="text" id="username" value="old_user">
    <input type="password" id="password">
    <button type="submit">Login</button>
  </form>

Login form automation:
  type "#username" "new_user" --clear   # Replace existing username
  type "#password" "secret123"
  click "button[type=submit]"

Or with --key for keyboard submission:
  type "#username" "new_user" --clear
  type "#password" "secret123" --key Enter

Search form patterns:
  type "#search" "my query" --key Enter
  type ".search-box" "term" --clear --key Enter

Multi-field form with Tab navigation:
  type "#field1" "value1" --key Tab
  type "value2" --key Tab               # No selector, uses focused element
  type "value3" --key Enter             # Submit on last field

Error cases:
  - "element not found: .missing" - selector doesn't match any element
  - "element is not focusable" - cannot focus the target element
  - "daemon not running" - start daemon first with: webctl start`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runType,
}

func init() {
	typeCmd.Flags().String("key", "", "Key to send after typing (e.g., Enter)")
	typeCmd.Flags().Bool("clear", false, "Clear existing content before typing")
	rootCmd.AddCommand(typeCmd)
}

func runType(cmd *cobra.Command, args []string) error {
	t := startTimer("type")
	defer t.log()

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Read flags from command
	key, _ := cmd.Flags().GetString("key")
	clear, _ := cmd.Flags().GetBool("clear")

	var selector, text string
	if len(args) == 1 {
		text = args[0]
	} else {
		selector = args[0]
		text = args[1]
	}

	// Note: don't log text content for security reasons
	debugParam("selector=%q key=%q clear=%v textLen=%d", selector, key, clear, len(text))

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer func() { _ = exec.Close() }()

	params, err := json.Marshal(ipc.TypeParams{
		Selector: selector,
		Text:     text,
		Key:      key,
		Clear:    clear,
	})
	if err != nil {
		return outputError(err.Error())
	}

	debugRequest("type", fmt.Sprintf("selector=%q key=%q clear=%v", selector, key, clear))
	ipcStart := time.Now()

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "type",
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
