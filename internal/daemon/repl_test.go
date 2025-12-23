package daemon

import (
	"testing"

	"github.com/grantcarthew/webctl/internal/ipc"
)

func TestREPL_handleSpecialCommand(t *testing.T) {
	shutdownCalled := false
	r := NewREPL(func(req ipc.Request) ipc.Response {
		return ipc.SuccessResponse(nil)
	}, nil, func() { shutdownCalled = true })

	tests := []struct {
		name         string
		line         string
		wantHandled  bool
		wantShutdown bool
	}{
		{"exit", "exit", true, true},
		{"quit", "quit", true, true},
		{"stop", "stop", true, true},
		{"help", "help", true, false},
		{"question mark", "?", true, false},
		{"history", "history", true, false},
		{"regular command", "status", false, false},
		{"clear command", "clear console", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shutdownCalled = false
			handled := r.handleSpecialCommand(tt.line)

			if handled != tt.wantHandled {
				t.Errorf("handleSpecialCommand() = %v, want %v", handled, tt.wantHandled)
			}

			if tt.wantShutdown && !shutdownCalled {
				t.Error("expected shutdown to be called")
			}
		})
	}
}

func TestNewREPL(t *testing.T) {
	handlerCalled := false
	handler := func(req ipc.Request) ipc.Response {
		handlerCalled = true
		return ipc.SuccessResponse(nil)
	}

	shutdownCalled := false
	shutdown := func() {
		shutdownCalled = true
	}

	cmdExecCalled := false
	cmdExec := func(args []string) (bool, error) {
		cmdExecCalled = true
		return true, nil
	}

	r := NewREPL(handler, cmdExec, shutdown)

	if r == nil {
		t.Fatal("NewREPL() returned nil")
	}

	// Verify shutdown callback
	r.shutdown()
	if !shutdownCalled {
		t.Error("shutdown callback was not called")
	}

	// Verify command executor is set
	if r.cmdExec == nil {
		t.Error("cmdExec should not be nil")
	}

	// Call cmdExec to verify it works
	r.cmdExec([]string{"test"})
	if !cmdExecCalled {
		t.Error("cmdExec was not called")
	}

	// Verify handler is set (test basic fallback)
	r2 := NewREPL(handler, nil, shutdown)
	r2.executeBasic([]string{"status"})
	if !handlerCalled {
		t.Error("handler was not called through executeBasic")
	}
}

func TestREPL_parseBasicCommand(t *testing.T) {
	r := NewREPL(func(req ipc.Request) ipc.Response {
		return ipc.SuccessResponse(nil)
	}, nil, func() {})

	tests := []struct {
		name       string
		cmd        string
		args       []string
		wantCmd    string
		wantTarget string
		wantNil    bool
	}{
		{
			name:    "status",
			cmd:     "status",
			args:    nil,
			wantCmd: "status",
		},
		{
			name:    "console",
			cmd:     "console",
			args:    nil,
			wantCmd: "console",
		},
		{
			name:    "network",
			cmd:     "network",
			args:    nil,
			wantCmd: "network",
		},
		{
			name:       "clear without target",
			cmd:        "clear",
			args:       nil,
			wantCmd:    "clear",
			wantTarget: "",
		},
		{
			name:       "clear console",
			cmd:        "clear",
			args:       []string{"console"},
			wantCmd:    "clear",
			wantTarget: "console",
		},
		{
			name:       "clear network",
			cmd:        "clear",
			args:       []string{"network"},
			wantCmd:    "clear",
			wantTarget: "network",
		},
		{
			name:    "unknown command",
			cmd:     "unknown",
			args:    nil,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := r.parseBasicCommand(tt.cmd, tt.args)

			if tt.wantNil {
				if req != nil {
					t.Errorf("parseBasicCommand() = %v, want nil", req)
				}
				return
			}

			if req == nil {
				t.Fatal("parseBasicCommand() = nil, want non-nil")
			}
			if req.Cmd != tt.wantCmd {
				t.Errorf("parseBasicCommand().Cmd = %q, want %q", req.Cmd, tt.wantCmd)
			}
			if req.Target != tt.wantTarget {
				t.Errorf("parseBasicCommand().Target = %q, want %q", req.Target, tt.wantTarget)
			}
		})
	}
}

func TestREPL_executeCommand_withCommandExecutor(t *testing.T) {
	executedArgs := []string{}
	cmdExec := func(args []string) (bool, error) {
		executedArgs = args
		return true, nil
	}

	r := NewREPL(func(req ipc.Request) ipc.Response {
		return ipc.SuccessResponse(nil)
	}, cmdExec, func() {})

	// Test that command executor is called with correct args
	r.executeCommand("console --tail 5")

	if len(executedArgs) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(executedArgs), executedArgs)
	}
	if executedArgs[0] != "console" {
		t.Errorf("expected first arg 'console', got %q", executedArgs[0])
	}
	if executedArgs[1] != "--tail" {
		t.Errorf("expected second arg '--tail', got %q", executedArgs[1])
	}
	if executedArgs[2] != "5" {
		t.Errorf("expected third arg '5', got %q", executedArgs[2])
	}
}

func TestREPL_executeCommand_fallbackToBasic(t *testing.T) {
	handlerCalled := false
	receivedCmd := ""

	handler := func(req ipc.Request) ipc.Response {
		handlerCalled = true
		receivedCmd = req.Cmd
		return ipc.SuccessResponse(nil)
	}

	// No command executor - should fall back to basic
	r := NewREPL(handler, nil, func() {})

	r.executeCommand("status")

	if !handlerCalled {
		t.Error("handler was not called in fallback mode")
	}
	if receivedCmd != "status" {
		t.Errorf("expected cmd 'status', got %q", receivedCmd)
	}
}

func TestIsStdinTTY(t *testing.T) {
	// In test environment, stdin is typically not a TTY
	// Just verify the function runs without panic
	result := IsStdinTTY()
	t.Logf("IsStdinTTY() = %v (expected false in test environment)", result)
}

func TestExpandAbbreviation(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		commands []string
		want     string
		wantOK   bool
	}{
		// Webctl commands - single character (unique)
		{"h -> html", "h", webctlCommands, "html", true},
		{"k -> key", "k", webctlCommands, "key", true},

		// Single character (ambiguous)
		{"s ambiguous", "s", webctlCommands, "", false}, // status, screenshot, select, scroll
		{"c ambiguous", "c", webctlCommands, "", false}, // clear, click, console, cookies
		{"n ambiguous", "n", webctlCommands, "", false}, // navigate, network
		{"t ambiguous", "t", webctlCommands, "", false}, // target, type
		{"f ambiguous", "f", webctlCommands, "", false}, // focus, forward
		{"r ambiguous", "r", webctlCommands, "", false}, // ready, reload

		// Two character (unique)
		{"ba -> back", "ba", webctlCommands, "back", true},
		{"na -> navigate", "na", webctlCommands, "navigate", true},
		{"ne -> network", "ne", webctlCommands, "network", true},
		{"st -> status", "st", webctlCommands, "status", true},
		{"se -> select", "se", webctlCommands, "select", true},
		{"ta -> target", "ta", webctlCommands, "target", true},
		{"ty -> type", "ty", webctlCommands, "type", true},
		{"ev -> eval", "ev", webctlCommands, "eval", true},

		// Two character (ambiguous)
		{"sc ambiguous", "sc", webctlCommands, "", false}, // screenshot, scroll
		{"co ambiguous", "co", webctlCommands, "", false}, // console, cookies
		{"cl ambiguous", "cl", webctlCommands, "", false}, // clear, click
		{"fo ambiguous", "fo", webctlCommands, "", false}, // focus, forward
		{"re ambiguous", "re", webctlCommands, "", false}, // ready, reload

		// Three character (unique)
		{"con -> console", "con", webctlCommands, "console", true},
		{"coo -> cookies", "coo", webctlCommands, "cookies", true},
		{"cle -> clear", "cle", webctlCommands, "clear", true},
		{"cli -> click", "cli", webctlCommands, "click", true},
		{"foc -> focus", "foc", webctlCommands, "focus", true},
		{"for -> forward", "for", webctlCommands, "forward", true},
		{"rea -> ready", "rea", webctlCommands, "ready", true},
		{"rel -> reload", "rel", webctlCommands, "reload", true},

		// Four character (resolves screenshot/scroll ambiguity)
		{"scre -> screenshot", "scre", webctlCommands, "screenshot", true},
		{"scro -> scroll", "scro", webctlCommands, "scroll", true},

		// Full command names
		{"full status", "status", webctlCommands, "status", true},
		{"full navigate", "navigate", webctlCommands, "navigate", true},
		{"unknown", "xyz", webctlCommands, "", false},

		// REPL commands
		{"e -> exit", "e", replCommands, "exit", true},
		{"q -> quit", "q", replCommands, "quit", true},
		{"he -> help", "he", replCommands, "help", true},
		{"hi -> history", "hi", replCommands, "history", true},
		{"h ambiguous in repl", "h", replCommands, "", false}, // help and history
		{"sto -> stop", "sto", replCommands, "stop", true},

		// Case insensitivity
		{"uppercase S ambiguous", "S", webctlCommands, "", false},
		{"mixed case Sta", "Sta", webctlCommands, "status", true},
		{"mixed case Nav", "Nav", webctlCommands, "navigate", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := expandAbbreviation(tt.prefix, tt.commands)
			if ok != tt.wantOK {
				t.Errorf("expandAbbreviation(%q) ok = %v, want %v", tt.prefix, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("expandAbbreviation(%q) = %q, want %q", tt.prefix, got, tt.want)
			}
		})
	}
}

func TestREPL_handleSpecialCommand_abbreviations(t *testing.T) {
	shutdownCalled := false
	r := NewREPL(func(req ipc.Request) ipc.Response {
		return ipc.SuccessResponse(nil)
	}, nil, func() { shutdownCalled = true })

	tests := []struct {
		name         string
		line         string
		wantHandled  bool
		wantShutdown bool
	}{
		{"e -> exit", "e", true, true},
		{"q -> quit", "q", true, true},
		{"sto -> stop", "sto", true, true},
		{"he -> help", "he", true, false},
		{"hi -> history", "hi", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shutdownCalled = false
			handled := r.handleSpecialCommand(tt.line)

			if handled != tt.wantHandled {
				t.Errorf("handleSpecialCommand(%q) = %v, want %v", tt.line, handled, tt.wantHandled)
			}

			if tt.wantShutdown && !shutdownCalled {
				t.Errorf("expected shutdown to be called for %q", tt.line)
			}
		})
	}
}

func TestREPL_executeCommand_abbreviations(t *testing.T) {
	executedArgs := []string{}
	cmdExec := func(args []string) (bool, error) {
		executedArgs = args
		return true, nil
	}

	r := NewREPL(func(req ipc.Request) ipc.Response {
		return ipc.SuccessResponse(nil)
	}, cmdExec, func() {})

	tests := []struct {
		name        string
		line        string
		wantCommand string
	}{
		// Single character (unique)
		{"h -> html", "h", "html"},
		{"k -> key", "k", "key"},

		// Two character abbreviations
		{"st -> status", "st", "status"},
		{"ba -> back", "ba", "back"},
		{"na -> navigate", "na", "navigate"},
		{"ne -> network", "ne", "network"},
		{"se -> select", "se", "select"},
		{"ta -> target", "ta", "target"},
		{"ty -> type", "ty", "type"},
		{"ev -> eval", "ev", "eval"},

		// Three character abbreviations
		{"con -> console", "con", "console"},
		{"coo -> cookies", "coo", "cookies"},
		{"cle -> clear", "cle", "clear"},
		{"cli -> click", "cli", "click"},
		{"foc -> focus", "foc", "focus"},
		{"for -> forward", "for", "forward"},
		{"rea -> ready", "rea", "ready"},
		{"rel -> reload", "rel", "reload"},

		// Four character abbreviations
		{"scre -> screenshot", "scre", "screenshot"},
		{"scro -> scroll", "scro", "scroll"},

		// With arguments
		{"ne with args", "ne --head 5", "network"},
		{"na with args", "na https://example.com", "navigate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executedArgs = nil
			r.executeCommand(tt.line)

			if len(executedArgs) == 0 {
				t.Fatal("expected args to be set")
			}
			if executedArgs[0] != tt.wantCommand {
				t.Errorf("expected command %q, got %q", tt.wantCommand, executedArgs[0])
			}
		})
	}
}
