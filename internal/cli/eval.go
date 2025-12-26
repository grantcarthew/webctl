package cli

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/grantcarthew/webctl/internal/cli/format"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var evalCmd = &cobra.Command{
	Use:   "eval <expression>",
	Short: "Evaluate JavaScript in the browser",
	Long: `Evaluates a JavaScript expression in the current page context and returns the result.

Supports both synchronous and asynchronous (Promise-based) expressions. Results
are automatically serialized to JSON. Non-serializable values (DOM nodes,
functions, circular references) return undefined.

Flags:
  --timeout, -t     Timeout for async expressions (default 30s)
                    Accepts Go duration format: 10s, 1m, 500ms

Simple expressions:
  eval "1 + 1"                                  # {"ok": true, "value": 2}
  eval "document.title"                         # {"ok": true, "value": "Page Title"}
  eval "window.location.href"                   # {"ok": true, "value": "https://..."}
  eval "Date.now()"                             # {"ok": true, "value": 1703419200000}

Object and array results:
  eval "[1, 2, 3].map(x => x * 2)"              # {"ok": true, "value": [2, 4, 6]}
  eval "({name: 'test', count: 42})"            # {"ok": true, "value": {"name": "test", "count": 42}}
  eval "Array.from(document.querySelectorAll('a')).map(a => a.href)"

DOM inspection (values only, not nodes):
  eval "document.querySelectorAll('a').length"  # Count elements
  eval "document.querySelector('#main').textContent.trim()"
  eval "document.querySelector('input').value"  # Get input value
  eval "getComputedStyle(document.body).backgroundColor"

Async/Promise expressions (automatically awaited):
  eval "fetch('/api/data').then(r => r.json())"
  eval "new Promise(r => setTimeout(() => r('done'), 1000))"

Check element existence (useful for SPAs):
  eval "document.querySelector('.dashboard') !== null"
  eval "!!document.getElementById('loaded-indicator')"

Get application state:
  eval "window.__APP_STATE__"                   # React/Redux state
  eval "localStorage.getItem('user')"
  eval "sessionStorage.getItem('token')"

Multi-statement with IIFE:
  eval "(function() { const x = 1; const y = 2; return x + y; })()"
  eval "(() => { let sum = 0; for(let i=0; i<10; i++) sum += i; return sum; })()"

Modify page state (use with caution):
  eval "document.body.style.background = 'red'"
  eval "localStorage.setItem('debug', 'true')"

With custom timeout:
  eval --timeout 60s "slowAsyncOperation()"
  eval -t 5s "quickCheck()"

Response formats:
  {"ok": true, "value": 42}                     # With value
  {"ok": true}                                  # Expression returned undefined

Error cases:
  - "SyntaxError: Unexpected token" - invalid JavaScript syntax
  - "ReferenceError: x is not defined" - undefined variable
  - "evaluation timed out after 30s" - async operation took too long
  - "daemon not running" - start daemon first with: webctl start

Note: For complex scripts, consider using a file and piping:
  cat script.js | xargs -0 webctl eval`,
	Args: cobra.MinimumNArgs(1),
	RunE: runEval,
}

func init() {
	evalCmd.Flags().DurationP("timeout", "t", 60*time.Second, "Timeout for async expressions")
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

	// JSON mode: output JSON with value
	if JSONOutput {
		result := map[string]any{
			"ok": true,
		}
		if data.HasValue {
			result["value"] = data.Value
		}
		return outputJSON(os.Stdout, result)
	}

	// Text mode: use text formatter (outputs raw value)
	return format.EvalResult(os.Stdout, data)
}
