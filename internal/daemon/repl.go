package daemon

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/grantcarthew/webctl/internal/ipc"
	"golang.org/x/term"
)

// SessionProvider returns the active session info and total count.
type SessionProvider func() (active *ipc.PageSession, count int)

// REPL provides an interactive command interface for the daemon.
type REPL struct {
	handler     ipc.Handler
	cmdExec     ipc.CommandExecutor
	sessionProv SessionProvider
	readline    *readline.Instance
	history     []string
	shutdown    func()
	closeOnce   sync.Once
	closeErr    error
}

// NewREPL creates a new REPL with the given handler, command executor, and shutdown callback.
// The cmdExec function executes CLI commands with full flag support.
// If cmdExec is nil, REPL falls back to basic IPC-only command execution.
func NewREPL(handler ipc.Handler, cmdExec ipc.CommandExecutor, shutdown func()) *REPL {
	return &REPL{
		handler:  handler,
		cmdExec:  cmdExec,
		shutdown: shutdown,
	}
}

// SetSessionProvider sets the session provider for dynamic prompt generation.
func (r *REPL) SetSessionProvider(sp SessionProvider) {
	r.sessionProv = sp
}

// Close closes the readline instance if it exists.
// Safe to call multiple times (idempotent).
// Returns the error from the first close attempt on all subsequent calls.
func (r *REPL) Close() error {
	r.closeOnce.Do(func() {
		if r.readline != nil {
			r.closeErr = r.readline.Close()
		}
	})
	return r.closeErr
}

// IsStdinTTY returns true if stdin is a terminal.
func IsStdinTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// Run starts the REPL loop. Blocks until exit command or EOF.
func (r *REPL) Run() error {
	// Create readline instance with initial prompt
	cfg := &readline.Config{
		Prompt:          r.prompt(),
		InterruptPrompt: "^C",
		EOFPrompt:       "^D",
	}

	rl, err := readline.NewEx(cfg)
	if err != nil {
		return err
	}
	r.readline = rl
	defer r.Close()

	for {
		// Update prompt dynamically before each read
		r.readline.SetPrompt(r.prompt())

		line, err := r.readline.Readline()
		if err != nil {
			if err == readline.ErrInterrupt || err == io.EOF {
				return nil
			}
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		r.history = append(r.history, line)

		handled, err := r.handleSpecialCommand(line)
		if err != nil {
			// Clean exit requested (io.EOF from exit/quit/stop commands)
			return nil
		}
		if handled {
			continue
		}

		r.executeCommand(line)
	}
}

// cleanURLForDisplay removes protocol and trailing slash from URL for prompt display.
func cleanURLForDisplay(url string) string {
	// Remove https:// or http://
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Remove trailing slash only if no path
	if strings.HasSuffix(url, "/") && strings.Count(url, "/") == 1 {
		url = strings.TrimSuffix(url, "/")
	}

	return url
}

// prompt generates the REPL prompt with session context.
func (r *REPL) prompt() string {
	// Check if color should be enabled
	useColor := shouldUseREPLColor()

	if r.sessionProv == nil {
		if useColor {
			return coloredPrompt("", 0)
		}
		return "webctl> "
	}

	active, count := r.sessionProv()
	if active == nil {
		if useColor {
			return coloredPrompt("", 0)
		}
		return "webctl> "
	}

	// Use URL instead of title, clean it for display
	displayURL := cleanURLForDisplay(active.URL)
	if displayURL == "" {
		displayURL = "no-url"
	}

	// Truncate URL to 40 chars (URLs can be longer than titles)
	if len(displayURL) > 40 {
		displayURL = displayURL[:37] + "..."
	}

	if useColor {
		return coloredPrompt(displayURL, count)
	}

	if count > 1 {
		return fmt.Sprintf("webctl [%s](%d)> ", displayURL, count)
	}
	return fmt.Sprintf("webctl [%s]> ", displayURL)
}

// shouldUseREPLColor determines if the REPL should use colors.
// Respects NO_COLOR env var but always assumes TTY in interactive mode.
func shouldUseREPLColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	// REPL is always interactive, so we default to true
	return true
}

// coloredPrompt generates a colored REPL prompt.
// Format: webctl [url](count)>
// Colors: webctl=blue, []=default, url=cyan, count=default, >=bold white
func coloredPrompt(url string, count int) string {
	blue := color.New(color.FgBlue)
	cyan := color.New(color.FgCyan)
	boldWhite := color.New(color.FgWhite, color.Bold)

	if url == "" {
		// No session: webctl>
		return blue.Sprint("webctl") + boldWhite.Sprint("> ")
	}

	if count > 1 {
		// Multiple sessions: webctl [url](count)>
		return blue.Sprint("webctl") + " [" + cyan.Sprint(url) + fmt.Sprintf("](%d)", count) + boldWhite.Sprint("> ")
	}

	// Single session: webctl [url]>
	return blue.Sprint("webctl") + " [" + cyan.Sprint(url) + "]" + boldWhite.Sprint("> ")
}

// replCommands lists REPL-specific commands for abbreviation matching.
var replCommands = []string{"exit", "quit", "help", "history", "stop"}

// webctlCommands lists webctl commands for abbreviation matching.
var webctlCommands = []string{
	"back", "clear", "click", "console", "cookies", "eval", "find", "focus",
	"forward", "html", "key", "navigate", "network", "ready", "reload",
	"screenshot", "scroll", "select", "status", "target", "type",
}

// expandAbbreviation expands a command prefix to a full command name.
// Returns the expanded command and true if exactly one match found.
// Returns empty string and false if no matches or ambiguous.
func expandAbbreviation(prefix string, commands []string) (string, bool) {
	prefix = strings.ToLower(prefix)
	var matches []string
	for _, cmd := range commands {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}
	if len(matches) == 1 {
		return matches[0], true
	}
	return "", false
}

// handleSpecialCommand handles REPL-specific commands.
// Returns (handled, error). Returns io.EOF for clean exit commands.
func (r *REPL) handleSpecialCommand(line string) (bool, error) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return false, nil
	}
	cmd := strings.ToLower(parts[0])

	// Try to expand abbreviation
	if expanded, ok := expandAbbreviation(cmd, replCommands); ok {
		cmd = expanded
	}

	switch cmd {
	case "exit", "quit", "stop":
		// Return io.EOF to signal clean exit.
		// This allows deferred cleanup to run before daemon shutdown.
		return true, io.EOF

	case "help", "?":
		r.printHelp()
		return true, nil

	case "history":
		r.printHistory()
		return true, nil
	}

	return false, nil
}

// executeCommand parses and executes a webctl command.
func (r *REPL) executeCommand(line string) {
	args := strings.Fields(line)
	if len(args) == 0 {
		return
	}

	// Try to expand command abbreviation
	if expanded, ok := expandAbbreviation(args[0], webctlCommands); ok {
		args[0] = expanded
	}

	// Use command executor if available (provides full Cobra flag support)
	if r.cmdExec != nil {
		recognized, err := r.cmdExec(args)
		if !recognized {
			outputError(fmt.Sprintf("unknown command: %s", args[0]))
			return
		}
		// Errors are already output by the command, but Cobra may return an error
		// for flag parsing issues that aren't output
		if err != nil && !strings.Contains(err.Error(), "daemon") {
			outputError(err.Error())
		}
		return
	}

	// Fallback: basic IPC-only execution (no flag support)
	r.executeBasic(args)
}

// executeBasic provides basic command execution without Cobra flag support.
// This is a fallback when no CommandExecutor is provided.
func (r *REPL) executeBasic(args []string) {
	cmd := args[0]
	req := r.parseBasicCommand(cmd, args[1:])
	if req == nil {
		outputError(fmt.Sprintf("unknown command: %s", cmd))
		return
	}

	resp := r.handler(*req)
	r.outputResponse(resp)
}

// parseBasicCommand converts command and args to an IPC request (basic mode only).
func (r *REPL) parseBasicCommand(cmd string, args []string) *ipc.Request {
	switch cmd {
	case "status":
		return &ipc.Request{Cmd: "status"}
	case "console":
		return &ipc.Request{Cmd: "console"}
	case "network":
		return &ipc.Request{Cmd: "network"}
	case "target":
		query := ""
		if len(args) > 0 {
			query = args[0]
		}
		return &ipc.Request{Cmd: "target", Target: query}
	case "clear":
		target := ""
		if len(args) > 0 {
			target = args[0]
		}
		return &ipc.Request{Cmd: "clear", Target: target}
	default:
		return nil
	}
}

// isStdoutTTY returns true if stdout is a terminal.
func isStdoutTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// outputJSON writes data as JSON to stdout, pretty-printing if stdout is a TTY.
func outputJSON(data any) {
	enc := json.NewEncoder(os.Stdout)
	if isStdoutTTY() {
		enc.SetIndent("", "  ")
	}
	enc.Encode(data)
}

// outputError writes an error response in text format to stderr.
// The REPL uses text format for its own errors since it's interactive.
// Individual commands respect the --json flag separately.
func outputError(msg string) {
	if shouldUseREPLColor() {
		color.New(color.FgRed).Fprint(os.Stderr, "Error:")
		fmt.Fprintf(os.Stderr, " %s\n", msg)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	}
}

// outputResponse writes the response as JSON to stdout.
func (r *REPL) outputResponse(resp ipc.Response) {
	outputJSON(resp)
}

// printHelp displays available commands.
func (r *REPL) printHelp() {
	help := `
Commands (unique prefixes accepted: h=html, k=key, ba=back, na=navigate, ne=network, cli=click, foc=focus):
  Navigation:
    navigate <url>      Navigate to URL
    reload              Reload current page
    back                Go back in history
    forward             Go forward in history

  Interaction:
    click <selector>    Click element
    type <selector> <text>  Type text into element
    select <selector> <value>  Select dropdown option
    scroll <target>     Scroll to element or position
    focus <selector>    Focus element
    key <key>           Send keyboard key

  Observation:
    status              Show daemon status
    console [flags]     Show console log entries
      --type <type>       Filter by entry type (repeatable)
      --head <n>          Return first N entries
      --tail <n>          Return last N entries
      --range <start-end> Return entries in range
    network             Show network requests
    screenshot          Capture screenshot of current page
    html [selector]     Extract HTML from current page
    eval <expression>   Evaluate JavaScript expression
    cookies             Show cookies for current page

  Utility:
    target [query]      List sessions or switch to a session
    clear [target]      Clear event buffers (console, network, or all)
    ready               Wait for page load

REPL (unique prefixes accepted: he=help, hi=history, e=exit, q=quit):
  help, ?     Show this help
  history     Show command history
  exit, quit  Stop daemon and exit
`
	fmt.Println(help)
}

// printHistory displays command history.
func (r *REPL) printHistory() {
	for i, cmd := range r.history {
		fmt.Printf("  %d  %s\n", i+1, cmd)
	}
}

// refreshPrompt updates the REPL prompt to reflect current session state.
// Called when session info changes (e.g., from CDP events).
func (r *REPL) refreshPrompt() {
	if r.readline == nil {
		return
	}
	r.readline.SetPrompt(r.prompt())
	r.readline.Refresh()
}

// displayExternalCommand shows an external command notification in the REPL.
// Clears the current prompt line, prints the notification, and refreshes the prompt.
func (r *REPL) displayExternalCommand(summary string) {
	if r.readline == nil {
		return
	}

	// Clear current line and print notification in dim text
	fmt.Print("\r\033[K")
	if shouldUseREPLColor() {
		dim := color.New(color.Faint)
		dim.Printf("< %s\n", summary)
	} else {
		fmt.Printf("< %s\n", summary)
	}

	// Refresh prompt with updated state
	r.refreshPrompt()
}

// formatCommandSummary extracts the primary argument from a request for display.
func formatCommandSummary(req ipc.Request) string {
	switch req.Cmd {
	case "navigate":
		var params ipc.NavigateParams
		if json.Unmarshal(req.Params, &params) == nil && params.URL != "" {
			return "navigate " + params.URL
		}
	case "click":
		var params ipc.ClickParams
		if json.Unmarshal(req.Params, &params) == nil && params.Selector != "" {
			return "click " + params.Selector
		}
	case "focus":
		var params ipc.FocusParams
		if json.Unmarshal(req.Params, &params) == nil && params.Selector != "" {
			return "focus " + params.Selector
		}
	case "type":
		var params ipc.TypeParams
		if json.Unmarshal(req.Params, &params) == nil && params.Selector != "" {
			return "type " + params.Selector
		}
	case "select":
		var params ipc.SelectParams
		if json.Unmarshal(req.Params, &params) == nil && params.Selector != "" {
			return "select " + params.Selector
		}
	case "scroll":
		var params ipc.ScrollParams
		if json.Unmarshal(req.Params, &params) == nil {
			if params.Selector != "" {
				return "scroll " + params.Selector
			}
			if params.Mode == "to" {
				return fmt.Sprintf("scroll to %d,%d", params.ToX, params.ToY)
			}
			if params.Mode == "by" {
				return fmt.Sprintf("scroll by %d,%d", params.ByX, params.ByY)
			}
		}
	case "css":
		var params ipc.CSSParams
		if json.Unmarshal(req.Params, &params) == nil {
			if params.Action == "computed" && params.Selector != "" {
				return "css computed " + params.Selector
			}
			if params.Action == "get" && params.Selector != "" {
				return "css get " + params.Selector
			}
		}
	case "cookies":
		var params ipc.CookiesParams
		if json.Unmarshal(req.Params, &params) == nil {
			if params.Action == "set" && params.Name != "" {
				return "cookies set " + params.Name
			}
			if params.Action == "delete" && params.Name != "" {
				return "cookies delete " + params.Name
			}
		}
	case "clear":
		if req.Target != "" {
			return "clear " + req.Target
		}
	}

	// Default: just the command name
	return req.Cmd
}
