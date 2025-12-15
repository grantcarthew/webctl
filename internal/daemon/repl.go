package daemon

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/grantcarthew/webctl/internal/executor"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/peterh/liner"
	"golang.org/x/term"
)

// REPL provides an interactive command interface for the daemon.
type REPL struct {
	executor executor.Executor
	liner    *liner.State
	history  []string
	shutdown func()
}

// NewREPL creates a new REPL with the given handler and shutdown callback.
func NewREPL(handler ipc.Handler, shutdown func()) *REPL {
	return &REPL{
		executor: executor.NewDirectExecutor(handler),
		shutdown: shutdown,
	}
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
		line, err := r.liner.Prompt("webctl> ")
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

// handleSpecialCommand handles REPL-specific commands.
// Returns true if the command was handled, false otherwise.
func (r *REPL) handleSpecialCommand(line string) bool {
	cmd := strings.ToLower(strings.Fields(line)[0])

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
	}

	return false
}

// executeCommand parses and executes a webctl command.
func (r *REPL) executeCommand(line string) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return
	}

	cmd := parts[0]
	req := r.parseCommand(cmd, parts[1:])
	if req == nil {
		fmt.Printf("{\"ok\":false,\"error\":\"unknown command: %s\"}\n", cmd)
		return
	}

	resp, err := r.executor.Execute(*req)
	if err != nil {
		fmt.Printf("{\"ok\":false,\"error\":\"%s\"}\n", err.Error())
		return
	}

	r.outputResponse(resp)
}

// parseCommand converts command and args to an IPC request.
func (r *REPL) parseCommand(cmd string, args []string) *ipc.Request {
	switch cmd {
	case "status":
		return &ipc.Request{Cmd: "status"}

	case "console":
		return &ipc.Request{Cmd: "console"}

	case "network":
		return &ipc.Request{Cmd: "network"}

	case "clear":
		target := ""
		if len(args) > 0 {
			target = args[0]
		}
		return &ipc.Request{Cmd: "clear", Target: target}

	case "stop":
		return &ipc.Request{Cmd: "shutdown"}

	default:
		return nil
	}
}

// outputResponse writes the response as JSON to stdout.
func (r *REPL) outputResponse(resp ipc.Response) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(resp)
}

// printHelp displays available commands.
func (r *REPL) printHelp() {
	help := `
Commands:
  status      Show daemon status
  console     Show console log entries
  network     Show network requests
  clear       Clear event buffers

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
