package ipc

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSuccessResponse(t *testing.T) {
	data := StatusData{
		Running: true,
		PID:     1234,
		ActiveSession: &PageSession{
			URL:   "https://example.com",
			Title: "Example",
		},
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
	if status.Running != true || status.ActiveSession.URL != "https://example.com" {
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
		Type: "log",
		Text: "hello world",
		Args: []ConsoleArg{
			{Type: "string", Value: json.RawMessage(`"hello world"`)},
			{Type: "object", Subtype: "array", Description: "Array(2)", Preview: []ConsolePreviewProp{
				{Name: "0", Type: "number", Value: "1"},
				{Name: "1", Type: "number", Value: "2"},
			}},
		},
		Timestamp: 1234567890,
		URL:       "https://example.com/script.js",
		Line:      42,
		Column:    10,
		Stack: []ConsoleFrame{
			{Function: "foo", URL: "https://example.com/script.js", Line: 42, Column: 10},
			{Function: "onClick", URL: "https://example.com/app.js", Line: 7, Async: "Promise.then"},
		},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Arguments must serialize as structured values, not strings.
	if !strings.Contains(string(data), `"description":"Array(2)"`) {
		t.Errorf("expected structured object arg in JSON, got %s", data)
	}

	var got ConsoleEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if got.Type != entry.Type || got.Text != entry.Text || got.Line != entry.Line {
		t.Errorf("round-trip mismatch: got %+v", got)
	}
	if len(got.Args) != 2 || got.Args[1].Description != "Array(2)" || len(got.Args[1].Preview) != 2 {
		t.Errorf("args round-trip mismatch: got %+v", got.Args)
	}
	if len(got.Stack) != 2 || got.Stack[0].Function != "foo" || got.Stack[1].Async != "Promise.then" {
		t.Errorf("stack round-trip mismatch: got %+v", got.Stack)
	}
}

func TestNetworkEntry_JSON(t *testing.T) {
	entry := NetworkEntry{
		RequestID:      "req-123",
		URL:            "https://api.example.com/data",
		Method:         "GET",
		Status:         200,
		StatusText:     "OK",
		Type:           "XHR",
		MimeType:       "application/json",
		RequestTime:    1000,
		ResponseTime:   1500,
		Duration:       0.5,
		Size:           1024,
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

func TestNetworkEntry_TelemetryJSON(t *testing.T) {
	entry := NetworkEntry{
		RequestID:         "req-456",
		URL:               "https://example.com/app.js",
		Method:            "GET",
		RemoteIPAddress:   "93.184.216.34",
		RemotePort:        443,
		Protocol:          "h2",
		FromDiskCache:     true,
		FromServiceWorker: true,
		FromPrefetchCache: true,
		ConnectionID:      17,
		SecurityState:     "secure",
		Timing:            &NetworkTiming{DNSMs: 5, ConnectMs: 20, TLSMs: 15, SendMs: 1, WaitMs: 30},
		Initiator:         &NetworkInitiator{Type: "parser", URL: "https://example.com/", Line: 42},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var got NetworkEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if got.RemoteIPAddress != entry.RemoteIPAddress || got.RemotePort != entry.RemotePort {
		t.Errorf("remote endpoint round-trip mismatch: got IP=%q port=%d", got.RemoteIPAddress, got.RemotePort)
	}
	if got.Protocol != "h2" || got.ConnectionID != 17 || got.SecurityState != "secure" {
		t.Errorf("transport fields round-trip mismatch: got %+v", got)
	}
	if !got.FromDiskCache || !got.FromServiceWorker || !got.FromPrefetchCache {
		t.Errorf("cache-origin flags round-trip mismatch: got %+v", got)
	}
	if got.Timing == nil || *got.Timing != *entry.Timing {
		t.Errorf("timing round-trip mismatch: got %+v", got.Timing)
	}
	if got.Initiator == nil || *got.Initiator != *entry.Initiator {
		t.Errorf("initiator round-trip mismatch: got %+v", got.Initiator)
	}
}

func TestEntrySeq_AlwaysPresentInJSON(t *testing.T) {
	// seq is a primary identifier agents address entries by, so it must appear
	// in JSON even at the reserved zero value, unlike the omitempty telemetry.
	consoleData, err := json.Marshal(ConsoleEntry{Type: "log", Text: "hi"})
	if err != nil {
		t.Fatalf("failed to marshal ConsoleEntry: %v", err)
	}
	if !strings.Contains(string(consoleData), `"seq":0`) {
		t.Errorf("ConsoleEntry JSON must include seq even when zero, got %s", consoleData)
	}

	networkData, err := json.Marshal(NetworkEntry{RequestID: "req-1", URL: "https://example.com/"})
	if err != nil {
		t.Fatalf("failed to marshal NetworkEntry: %v", err)
	}
	if !strings.Contains(string(networkData), `"seq":0`) {
		t.Errorf("NetworkEntry JSON must include seq even when zero, got %s", networkData)
	}

	// A stamped value must round-trip.
	stamped, err := json.Marshal(ConsoleEntry{Seq: 42, Type: "log"})
	if err != nil {
		t.Fatalf("failed to marshal stamped ConsoleEntry: %v", err)
	}
	var got ConsoleEntry
	if err := json.Unmarshal(stamped, &got); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if got.Seq != 42 {
		t.Errorf("expected seq 42 after round-trip, got %d", got.Seq)
	}
}

func TestNetworkEntry_TelemetryOmitEmpty(t *testing.T) {
	// A bare entry (request seen, no response yet) must not emit any of the new
	// transport keys, keeping JSON output lean for the common in-flight case.
	data, err := json.Marshal(NetworkEntry{RequestID: "req-789", URL: "https://example.com/"})
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	for _, key := range []string{
		"remoteIPAddress", "remotePort", "protocol", "fromDiskCache",
		"fromServiceWorker", "fromPrefetchCache", "connectionId",
		"securityState", "timing", "initiator",
	} {
		if strings.Contains(string(data), key) {
			t.Errorf("empty entry JSON should omit %q, got %s", key, data)
		}
	}
}
