package ipc

import (
	"encoding/json"
	"testing"
)

func TestSuccessResponse(t *testing.T) {
	data := StatusData{
		Running: true,
		URL:     "https://example.com",
		Title:   "Example",
		PID:     1234,
	}

	resp := SuccessResponse(data)

	if !resp.OK {
		t.Error("expected OK to be true")
	}
	if resp.Error != "" {
		t.Errorf("expected no error, got %q", resp.Error)
	}
	if resp.Data == nil {
		t.Error("expected data to be set")
	}

	// Verify data can be unmarshaled
	var status StatusData
	if err := json.Unmarshal(resp.Data, &status); err != nil {
		t.Fatalf("failed to unmarshal data: %v", err)
	}
	if status.Running != true || status.URL != "https://example.com" {
		t.Error("data mismatch")
	}
}

func TestSuccessResponseNilData(t *testing.T) {
	resp := SuccessResponse(nil)

	if !resp.OK {
		t.Error("expected OK to be true")
	}
	if resp.Data != nil {
		t.Errorf("expected nil data, got %v", resp.Data)
	}
}

func TestErrorResponse(t *testing.T) {
	resp := ErrorResponse("something went wrong")

	if resp.OK {
		t.Error("expected OK to be false")
	}
	if resp.Error != "something went wrong" {
		t.Errorf("expected error message, got %q", resp.Error)
	}
	if resp.Data != nil {
		t.Error("expected nil data for error response")
	}
}

func TestRequest_JSON(t *testing.T) {
	tests := []struct {
		name string
		req  Request
		want string
	}{
		{
			name: "simple command",
			req:  Request{Cmd: "status"},
			want: `{"cmd":"status"}`,
		},
		{
			name: "command with target",
			req:  Request{Cmd: "clear", Target: "console"},
			want: `{"cmd":"clear","target":"console"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var got Request
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if got.Cmd != tt.req.Cmd || got.Target != tt.req.Target {
				t.Errorf("round-trip mismatch: got %+v, want %+v", got, tt.req)
			}
		})
	}
}

func TestConsoleEntry_JSON(t *testing.T) {
	entry := ConsoleEntry{
		Type:      "log",
		Text:      "hello world",
		Args:      []string{"hello", "world"},
		Timestamp: 1234567890,
		URL:       "https://example.com/script.js",
		Line:      42,
		Column:    10,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var got ConsoleEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if got.Type != entry.Type || got.Text != entry.Text || got.Line != entry.Line {
		t.Errorf("round-trip mismatch: got %+v", got)
	}
}

func TestNetworkEntry_JSON(t *testing.T) {
	entry := NetworkEntry{
		RequestID:    "req-123",
		URL:          "https://api.example.com/data",
		Method:       "GET",
		Status:       200,
		StatusText:   "OK",
		Type:         "XHR",
		MimeType:     "application/json",
		RequestTime:  1000,
		ResponseTime: 1500,
		Duration:     0.5,
		Size:         1024,
		RequestHeaders: map[string]string{"Content-Type": "application/json"},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var got NetworkEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if got.RequestID != entry.RequestID || got.Status != entry.Status || got.Duration != entry.Duration {
		t.Errorf("round-trip mismatch: got %+v", got)
	}
}
