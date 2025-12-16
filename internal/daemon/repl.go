package daemon

import (
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

// handleSpecialCommand handles REPL-specific commands.
// Returns true if the command was handled, false otherwise.
func (r *REPL) handleSpecialCommand(line string) bool {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return false
	}
	cmd := strings.ToLower(parts[0])

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

	// Use command executor if available (provides full Cobra flag support)
	if r.cmdExec != nil {
		recognized, err := r.cmdExec(args)
		if !recognized {
			fmt.Printf("{\"ok\":false,\"error\":\"unknown command: %s\"}\n", args[0])
			return
		}
		// Errors are already output by the command, but Cobra may return an error
		// for flag parsing issues that aren't output
		if err != nil && !strings.Contains(err.Error(), "daemon") {
			fmt.Printf("{\"ok\":false,\"error\":\"%s\"}\n", err.Error())
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
		fmt.Printf("{\"ok\":false,\"error\":\"unknown command: %s\"}\n", cmd)
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

// outputResponse writes the response as JSON to stdout.
func (r *REPL) outputResponse(resp ipc.Response) {
	fmt.Printf("{\"ok\":%t", resp.OK)
	if resp.Error != "" {
		fmt.Printf(",\"error\":\"%s\"", resp.Error)
	}
	if resp.Data != nil {
		fmt.Printf(",\"data\":%s", string(resp.Data))
	}
	fmt.Println("}")
}

// printHelp displays available commands.
func (r *REPL) printHelp() {
	help := `
Commands:
  status              Show daemon status
  console [flags]     Show console log entries
    --format text|json  Output format (default: json)
    --type <type>       Filter by entry type (repeatable)
    --head <n>          Return first N entries
    --tail <n>          Return last N entries
    --range <start-end> Return entries in range
  network             Show network requests
  target [query]      List sessions or switch to a session
  clear [target]      Clear event buffers (console, network, or all)

REPL:
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
