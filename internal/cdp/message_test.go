package cdp

import (
	"encoding/json"
	"testing"
)

func TestParseMessage_Response(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantID     int64
		wantResult string
		wantErr    bool
	}{
		{
			name:       "successful response",
			input:      `{"id":1,"result":{"frameId":"ABC123"}}`,
			wantID:     1,
			wantResult: `{"frameId":"ABC123"}`,
			wantErr:    false,
		},
		{
			name:       "response with null result",
			input:      `{"id":42,"result":null}`,
			wantID:     42,
			wantResult: `null`,
			wantErr:    false,
		},
		{
			name:       "response with empty result",
			input:      `{"id":5,"result":{}}`,
			wantID:     5,
			wantResult: `{}`,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp, evt, err := parseMessage([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if evt != nil {
				t.Errorf("expected event to be nil, got %+v", evt)
			}
			if resp == nil {
				t.Fatal("expected response, got nil")
			}
			if resp.ID != tt.wantID {
				t.Errorf("expected ID %d, got %d", tt.wantID, resp.ID)
			}
			if string(resp.Result) != tt.wantResult {
				t.Errorf("expected result %s, got %s", tt.wantResult, string(resp.Result))
			}
		})
	}
}

func TestParseMessage_ResponseWithError(t *testing.T) {
	t.Parallel()

	input := `{"id":1,"error":{"code":-32000,"message":"Target closed","data":"extra info"}}`

	resp, evt, err := parseMessage([]byte(input))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if evt != nil {
		t.Errorf("expected event to be nil, got %+v", evt)
	}
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error == nil {
		t.Fatal("expected error in response, got nil")
	}
	if resp.Error.Code != -32000 {
		t.Errorf("expected error code -32000, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "Target closed" {
		t.Errorf("expected message 'Target closed', got %s", resp.Error.Message)
	}
	if resp.Error.Data != "extra info" {
		t.Errorf("expected data 'extra info', got %s", resp.Error.Data)
	}
}

func TestParseMessage_Event(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantMethod string
		wantParams string
	}{
		{
			name:       "simple event",
			input:      `{"method":"Page.loadEventFired","params":{"timestamp":123.456}}`,
			wantMethod: "Page.loadEventFired",
			wantParams: `{"timestamp":123.456}`,
		},
		{
			name:       "event with empty params",
			input:      `{"method":"Network.dataReceived","params":{}}`,
			wantMethod: "Network.dataReceived",
			wantParams: `{}`,
		},
		{
			name:       "event with complex params",
			input:      `{"method":"Runtime.consoleAPICalled","params":{"type":"log","args":[{"type":"string","value":"hello"}]}}`,
			wantMethod: "Runtime.consoleAPICalled",
			wantParams: `{"type":"log","args":[{"type":"string","value":"hello"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp, evt, err := parseMessage([]byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if resp != nil {
				t.Errorf("expected response to be nil, got %+v", resp)
			}
			if evt == nil {
				t.Fatal("expected event, got nil")
			}
			if evt.Method != tt.wantMethod {
				t.Errorf("expected method %s, got %s", tt.wantMethod, evt.Method)
			}
			if string(evt.Params) != tt.wantParams {
				t.Errorf("expected params %s, got %s", tt.wantParams, string(evt.Params))
			}
		})
	}
}

func TestParseMessage_InvalidJSON(t *testing.T) {
	t.Parallel()

	inputs := []string{
		`not json`,
		`{`,
		`{"id":}`,
		``,
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			_, _, err := parseMessage([]byte(input))
			if err == nil {
				t.Error("expected error for invalid JSON, got nil")
			}
		})
	}
}

func TestParseMessage_UnknownFormat(t *testing.T) {
	t.Parallel()

	// Message with neither ID nor method
	input := `{"foo":"bar"}`

	_, _, err := parseMessage([]byte(input))
	if err == nil {
		t.Error("expected error for unknown format, got nil")
	}
}

func TestError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      Error
		expected string
	}{
		{
			name:     "error without data",
			err:      Error{Code: -32000, Message: "Target closed"},
			expected: "cdp error -32000: Target closed",
		},
		{
			name:     "error with data",
			err:      Error{Code: -32602, Message: "Invalid params", Data: "missing 'url'"},
			expected: "cdp error -32602: Invalid params (missing 'url')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestRequest_Marshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		req      Request
		expected string
	}{
		{
			name:     "request without params",
			req:      Request{ID: 1, Method: "Page.enable"},
			expected: `{"id":1,"method":"Page.enable"}`,
		},
		{
			name:     "request with params",
			req:      Request{ID: 2, Method: "Page.navigate", Params: map[string]string{"url": "https://example.com"}},
			expected: `{"id":2,"method":"Page.navigate","params":{"url":"https://example.com"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(data))
			}
		})
	}
}

func FuzzParseMessage(f *testing.F) {
	// Seed with valid message formats
	f.Add([]byte(`{"id":1,"result":{}}`))
	f.Add([]byte(`{"id":1,"error":{"code":-1,"message":"error"}}`))
	f.Add([]byte(`{"method":"Page.loadEventFired","params":{}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"id":0}`))
	f.Add([]byte(`not json`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic regardless of input
		_, _, _ = parseMessage(data)
	})
}
