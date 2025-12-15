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
