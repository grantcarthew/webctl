package executor

import (
	"encoding/json"
	"testing"

	"github.com/grantcarthew/webctl/internal/ipc"
)

func TestDirectExecutor_Execute(t *testing.T) {
	tests := []struct {
		name     string
		request  ipc.Request
		response ipc.Response
	}{
		{
			name:     "simple command",
			request:  ipc.Request{Cmd: "status"},
			response: ipc.SuccessResponse(map[string]bool{"running": true}),
		},
		{
			name:     "command with target",
			request:  ipc.Request{Cmd: "clear", Target: "console"},
			response: ipc.SuccessResponse(nil),
		},
		{
			name:     "error response",
			request:  ipc.Request{Cmd: "unknown"},
			response: ipc.ErrorResponse("unknown command"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(req ipc.Request) ipc.Response {
				if req.Cmd != tt.request.Cmd {
					t.Errorf("handler received cmd %q, want %q", req.Cmd, tt.request.Cmd)
				}
				if req.Target != tt.request.Target {
					t.Errorf("handler received target %q, want %q", req.Target, tt.request.Target)
				}
				return tt.response
			}

			exec := NewDirectExecutor(handler)
			resp, err := exec.Execute(tt.request)

			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if resp.OK != tt.response.OK {
				t.Errorf("Execute() OK = %v, want %v", resp.OK, tt.response.OK)
			}
			if resp.Error != tt.response.Error {
				t.Errorf("Execute() Error = %q, want %q", resp.Error, tt.response.Error)
			}
		})
	}
}

func TestDirectExecutor_Close(t *testing.T) {
	exec := NewDirectExecutor(func(req ipc.Request) ipc.Response {
		return ipc.SuccessResponse(nil)
	})

	if err := exec.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestDirectExecutor_HandlerReceivesParams(t *testing.T) {
	params := map[string]string{"url": "https://example.com"}
	paramsJSON, _ := json.Marshal(params)

	var receivedParams json.RawMessage
	handler := func(req ipc.Request) ipc.Response {
		receivedParams = req.Params
		return ipc.SuccessResponse(nil)
	}

	exec := NewDirectExecutor(handler)
	req := ipc.Request{
		Cmd:    "navigate",
		Params: paramsJSON,
	}

	_, err := exec.Execute(req)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if string(receivedParams) != string(paramsJSON) {
		t.Errorf("handler received params %s, want %s", receivedParams, paramsJSON)
	}
}
