package daemon

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/peterh/liner"
	"golang.org/x/term"
)

// SessionProvider returns the active session info and total count.
type SessionProvider func() (active *ipc.PageSession, count int)

// REPL provides an interactive command interface for the daemon.
type REPL struct {
	handler     ipc.Handler
	cmdExec     ipc.CommandExecutor
	sessionProv SessionProvider
	liner       *liner.State
	history     []string
	shutdown    func()
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

// IsStdinTTY returns true if stdin is a terminal.
func IsStdinTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// Run starts the REPL loop. Blocks until exit command or EOF.
func (r *REPL) Run() error {
	r.liner = liner.NewLiner()
	defer r.liner.Close()

	r.liner.SetCtrlCAborts(true)

	for {
		line, err := r.liner.Prompt(r.prompt())
		if err != nil {
			if err == liner.ErrPromptAborted || err == io.EOF {
				return nil
			}
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		r.liner.AppendHistory(line)
		r.history = append(r.history, line)

		if r.handleSpecialCommand(line) {
			continue
		}

		r.executeCommand(line)
	}
}

// prompt generates the REPL prompt with session context.
func (r *REPL) prompt() string {
	if r.sessionProv == nil {
		return "webctl> "
	}

	active, count := r.sessionProv()
	if active == nil {
		return "webctl> "
	}

	// Truncate title to 30 chars
	title := active.Title
	if len(title) > 30 {
		title = title[:27] + "..."
	}

	if count > 1 {
		return fmt.Sprintf("webctl [%s](%d)> ", title, count)
	}
	return fmt.Sprintf("webctl [%s]> ", title)
}

// replCommands lists REPL-specific commands for abbreviation matching.
var replCommands = []string{"exit", "quit", "help", "history", "stop"}

// webctlCommands lists webctl commands for abbreviation matching.
var webctlCommands = []string{"status", "console", "network", "screenshot", "html", "target", "clear"}

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
// Returns true if the command was handled, false otherwise.
func (r *REPL) handleSpecialCommand(line string) bool {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return false
	}
	cmd := strings.ToLower(parts[0])

	// Try to expand abbreviation
	if expanded, ok := expandAbbreviation(cmd, replCommands); ok {
		cmd = expanded
	}

	switch cmd {
	case "exit", "quit":
		r.shutdown()
		return true

	case "help", "?":
		r.printHelp()
		return true

	case "history":
		r.printHistory()
		return true

	case "stop":
		// Map "stop" to shutdown (REPL-specific behavior)
		r.shutdown()
		return true
	}

	return false
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

// outputError writes an error response as JSON to stdout.
func outputError(msg string) {
	outputJSON(map[string]any{
		"ok":    false,
		"error": msg,
	})
}

// outputResponse writes the response as JSON to stdout.
func (r *REPL) outputResponse(resp ipc.Response) {
	outputJSON(resp)
}

// printHelp displays available commands.
func (r *REPL) printHelp() {
	help := `
Commands (unique prefixes accepted: st=status, sc=screenshot, h=html, n=network, t=target, co=console, cl=clear):
  status              Show daemon status
  console [flags]     Show console log entries
    --format text|json  Output format (default: json)
    --type <type>       Filter by entry type (repeatable)
    --head <n>          Return first N entries
    --tail <n>          Return last N entries
    --range <start-end> Return entries in range
  network             Show network requests
  screenshot          Capture screenshot of current page
  html [selector]     Extract HTML from current page
  target [query]      List sessions or switch to a session
  clear [target]      Clear event buffers (console, network, or all)

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
