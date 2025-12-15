package daemon

import (
	"testing"

	"github.com/grantcarthew/webctl/internal/ipc"
)

func TestREPL_parseCommand(t *testing.T) {
	shutdownCalled := false
	r := NewREPL(func(req ipc.Request) ipc.Response {
		return ipc.SuccessResponse(nil)
	}, func() { shutdownCalled = true })

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
			name:    "stop maps to shutdown",
			cmd:     "stop",
			args:    nil,
			wantCmd: "shutdown",
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
			req := r.parseCommand(tt.cmd, tt.args)

			if tt.wantNil {
				if req != nil {
					t.Errorf("parseCommand() = %v, want nil", req)
				}
				return
			}

			if req == nil {
				t.Fatal("parseCommand() = nil, want non-nil")
			}
			if req.Cmd != tt.wantCmd {
				t.Errorf("parseCommand().Cmd = %q, want %q", req.Cmd, tt.wantCmd)
			}
			if req.Target != tt.wantTarget {
				t.Errorf("parseCommand().Target = %q, want %q", req.Target, tt.wantTarget)
			}
		})
	}

	_ = shutdownCalled
}

func TestREPL_handleSpecialCommand(t *testing.T) {
	shutdownCalled := false
	r := NewREPL(func(req ipc.Request) ipc.Response {
		return ipc.SuccessResponse(nil)
	}, func() { shutdownCalled = true })

	tests := []struct {
		name            string
		line            string
		wantHandled     bool
		wantShutdown    bool
	}{
		{"exit", "exit", true, true},
		{"quit", "quit", true, true},
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

			// Only check shutdown for first occurrence (exit/quit reset state)
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

	r := NewREPL(handler, shutdown)

	if r == nil {
		t.Fatal("NewREPL() returned nil")
	}

	// Verify executor works
	r.executor.Execute(ipc.Request{Cmd: "test"})
	if !handlerCalled {
		t.Error("handler was not called through executor")
	}

	// Verify shutdown callback
	r.shutdown()
	if !shutdownCalled {
		t.Error("shutdown callback was not called")
	}
}

func TestIsStdinTTY(t *testing.T) {
	// In test environment, stdin is typically not a TTY
	// Just verify the function runs without panic
	result := IsStdinTTY()
	t.Logf("IsStdinTTY() = %v (expected false in test environment)", result)
}
