package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/grantcarthew/webctl/internal/cdp"
	"github.com/grantcarthew/webctl/internal/ipc"
)

func TestDaemon_parseRequestEvent_usesWallTime(t *testing.T) {
	// Create a daemon for testing
	d := New(DefaultConfig())

	// Create a mock Network.requestWillBeSent event with both timestamp and wallTime
	// wallTime should be used (Unix epoch in seconds)
	// timestamp is monotonic and should be ignored
	wallTime := float64(time.Now().Unix()) // Unix epoch seconds
	monotonic := 12345.678                 // Some arbitrary monotonic time

	params := map[string]any{
		"requestId": "test-123",
		"timestamp": monotonic, // Monotonic - should be ignored
		"wallTime":  wallTime,  // Unix epoch - should be used
		"request": map[string]any{
			"url":     "https://example.com/api",
			"method":  "GET",
			"headers": map[string]string{"Accept": "application/json"},
		},
		"type": "XHR",
	}
	paramsJSON, _ := json.Marshal(params)

	evt := cdp.Event{
		Method: "Network.requestWillBeSent",
		Params: json.RawMessage(paramsJSON),
	}

	entry, ok := d.parseRequestEvent(evt)
	if !ok {
		t.Fatal("parseRequestEvent returned false")
	}

	// Verify the entry uses wallTime (in milliseconds)
	expectedTime := int64(wallTime * 1000)
	if entry.RequestTime != expectedTime {
		t.Errorf("RequestTime = %d, want %d (based on wallTime)", entry.RequestTime, expectedTime)
	}

	// Verify it's NOT using monotonic timestamp
	monotonicMs := int64(monotonic * 1000)
	if entry.RequestTime == monotonicMs {
		t.Error("RequestTime incorrectly uses monotonic timestamp instead of wallTime")
	}

	// Verify the timestamp is a reasonable Unix time (after year 2020)
	year2020 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	if entry.RequestTime < year2020 {
		t.Errorf("RequestTime %d appears to be before 2020, suggesting monotonic time was used", entry.RequestTime)
	}
}

func TestDaemon_parseConsoleEvent(t *testing.T) {
	d := New(DefaultConfig())

	// Console events use timestamp which is Unix epoch milliseconds
	timestamp := float64(time.Now().UnixMilli())

	params := map[string]any{
		"type":      "log",
		"timestamp": timestamp,
		"args": []map[string]any{
			{"type": "string", "value": "Hello, World!"},
		},
	}
	paramsJSON, _ := json.Marshal(params)

	evt := cdp.Event{
		Method: "Runtime.consoleAPICalled",
		Params: json.RawMessage(paramsJSON),
	}

	entry, ok := d.parseConsoleEvent(evt)
	if !ok {
		t.Fatal("parseConsoleEvent returned false")
	}

	expectedTime := int64(timestamp)
	if entry.Timestamp != expectedTime {
		t.Errorf("Timestamp = %d, want %d", entry.Timestamp, expectedTime)
	}

	if entry.Text != "Hello, World!" {
		t.Errorf("Text = %q, want %q", entry.Text, "Hello, World!")
	}
}

func TestDaemon_Handler(t *testing.T) {
	d := New(DefaultConfig())

	handler := d.Handler()
	if handler == nil {
		t.Fatal("Handler() returned nil")
	}

	// Test that handler works (clear command should succeed even without buffers)
	resp := handler(ipc.Request{Cmd: "clear"})
	if !resp.OK {
		t.Errorf("handler returned OK=false for clear command: %s", resp.Error)
	}
}

func TestIsBinaryMimeType(t *testing.T) {
	tests := []struct {
		mimeType string
		want     bool
	}{
		{"image/png", true},
		{"image/jpeg", true},
		{"image/gif", true},
		{"image/webp", true},
		{"image/svg+xml", true},
		{"audio/mpeg", true},
		{"audio/ogg", true},
		{"video/mp4", true},
		{"video/webm", true},
		{"font/woff", true},
		{"font/woff2", true},
		{"application/octet-stream", true},
		{"application/pdf", true},
		{"application/zip", true},
		{"application/wasm", true},
		{"text/html", false},
		{"text/plain", false},
		{"text/css", false},
		{"text/javascript", false},
		{"application/json", false},
		{"application/javascript", false},
		{"application/xml", false},
		{"", false},
		{"IMAGE/PNG", true}, // case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			if got := isBinaryMimeType(tt.mimeType); got != tt.want {
				t.Errorf("isBinaryMimeType(%q) = %v, want %v", tt.mimeType, got, tt.want)
			}
		})
	}
}

func TestExtensionFromMimeType(t *testing.T) {
	tests := []struct {
		mimeType string
		want     string
	}{
		{"image/png", ".png"},
		{"image/jpeg", ".jpg"},
		{"image/gif", ".gif"},
		{"image/webp", ".webp"},
		{"image/svg+xml", ".svg"},
		{"font/woff", ".woff"},
		{"font/woff2", ".woff2"},
		{"audio/mpeg", ".mp3"},
		{"video/mp4", ".mp4"},
		{"application/pdf", ".pdf"},
		{"application/zip", ".zip"},
		{"text/html", ""},
		{"application/json", ""},
		{"unknown/type", ""},
		{"image/png; charset=utf-8", ".png"}, // handles parameters
		{"IMAGE/PNG", ".png"},                // case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			if got := extensionFromMimeType(tt.mimeType); got != tt.want {
				t.Errorf("extensionFromMimeType(%q) = %q, want %q", tt.mimeType, got, tt.want)
			}
		})
	}
}

func TestGetBodiesDir(t *testing.T) {
	dir := getBodiesDir()
	if dir == "" {
		t.Error("getBodiesDir() returned empty string")
	}
	// Should end with webctl/bodies
	if !contains(dir, "webctl") || !contains(dir, "bodies") {
		t.Errorf("getBodiesDir() = %q, expected to contain 'webctl' and 'bodies'", dir)
	}
}

func TestDaemon_parseExceptionEvent(t *testing.T) {
	d := New(DefaultConfig())

	t.Run("with exception description", func(t *testing.T) {
		timestamp := float64(time.Now().UnixMilli())
		params := map[string]any{
			"timestamp": timestamp,
			"exceptionDetails": map[string]any{
				"text":         "Uncaught Error",
				"url":          "https://example.com/script.js",
				"lineNumber":   42,
				"columnNumber": 10,
				"exception": map[string]any{
					"description": "Error: Something went wrong\n    at foo (script.js:42:10)",
				},
			},
		}
		paramsJSON, _ := json.Marshal(params)

		evt := cdp.Event{
			Method: "Runtime.exceptionThrown",
			Params: json.RawMessage(paramsJSON),
		}

		entry, ok := d.parseExceptionEvent(evt)
		if !ok {
			t.Fatal("parseExceptionEvent returned false")
		}

		if entry.Type != "error" {
			t.Errorf("Type = %q, want 'error'", entry.Type)
		}
		// Should prefer exception.description over exceptionDetails.text
		if entry.Text != "Error: Something went wrong\n    at foo (script.js:42:10)" {
			t.Errorf("Text = %q, want exception description", entry.Text)
		}
		if entry.URL != "https://example.com/script.js" {
			t.Errorf("URL = %q, want 'https://example.com/script.js'", entry.URL)
		}
		if entry.Line != 42 {
			t.Errorf("Line = %d, want 42", entry.Line)
		}
		if entry.Column != 10 {
			t.Errorf("Column = %d, want 10", entry.Column)
		}
		if entry.Timestamp != int64(timestamp) {
			t.Errorf("Timestamp = %d, want %d", entry.Timestamp, int64(timestamp))
		}
	})

	t.Run("without exception object", func(t *testing.T) {
		params := map[string]any{
			"timestamp": float64(1000),
			"exceptionDetails": map[string]any{
				"text":         "Script error.",
				"url":          "",
				"lineNumber":   0,
				"columnNumber": 0,
			},
		}
		paramsJSON, _ := json.Marshal(params)

		evt := cdp.Event{
			Method: "Runtime.exceptionThrown",
			Params: json.RawMessage(paramsJSON),
		}

		entry, ok := d.parseExceptionEvent(evt)
		if !ok {
			t.Fatal("parseExceptionEvent returned false")
		}

		// Should use exceptionDetails.text when no exception object
		if entry.Text != "Script error." {
			t.Errorf("Text = %q, want 'Script error.'", entry.Text)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		evt := cdp.Event{
			Method: "Runtime.exceptionThrown",
			Params: json.RawMessage(`{invalid json}`),
		}

		_, ok := d.parseExceptionEvent(evt)
		if ok {
			t.Error("parseExceptionEvent should return false for invalid JSON")
		}
	})
}

func TestDaemon_updateResponseEvent(t *testing.T) {
	d := New(DefaultConfig())

	// First, add a request to the network buffer
	requestEntry := ipc.NetworkEntry{
		RequestID:   "req-123",
		URL:         "https://example.com/api",
		Method:      "GET",
		RequestTime: time.Now().Add(-100 * time.Millisecond).UnixMilli(),
	}
	d.networkBuf.Push(requestEntry)

	// Now simulate a response event
	params := map[string]any{
		"requestId": "req-123",
		"response": map[string]any{
			"status":     200,
			"statusText": "OK",
			"mimeType":   "application/json",
			"headers": map[string]string{
				"Content-Type": "application/json",
			},
		},
	}
	paramsJSON, _ := json.Marshal(params)

	evt := cdp.Event{
		Method: "Network.responseReceived",
		Params: json.RawMessage(paramsJSON),
	}

	d.updateResponseEvent(evt)

	// Verify the entry was updated
	entries := d.networkBuf.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Status != 200 {
		t.Errorf("Status = %d, want 200", entry.Status)
	}
	if entry.StatusText != "OK" {
		t.Errorf("StatusText = %q, want 'OK'", entry.StatusText)
	}
	if entry.MimeType != "application/json" {
		t.Errorf("MimeType = %q, want 'application/json'", entry.MimeType)
	}
	if entry.ResponseHeaders["Content-Type"] != "application/json" {
		t.Errorf("ResponseHeaders[Content-Type] = %q, want 'application/json'", entry.ResponseHeaders["Content-Type"])
	}
	if entry.ResponseTime == 0 {
		t.Error("ResponseTime should be set")
	}
	if entry.Duration <= 0 {
		t.Error("Duration should be positive")
	}
}

func TestDaemon_handleLoadingFailed(t *testing.T) {
	t.Run("network error", func(t *testing.T) {
		d := New(DefaultConfig())

		// Add a request to the buffer
		requestEntry := ipc.NetworkEntry{
			RequestID:   "req-456",
			URL:         "https://example.com/missing",
			Method:      "GET",
			RequestTime: time.Now().Add(-50 * time.Millisecond).UnixMilli(),
		}
		d.networkBuf.Push(requestEntry)

		// Simulate a loading failed event
		params := map[string]any{
			"requestId": "req-456",
			"errorText": "net::ERR_CONNECTION_REFUSED",
			"canceled":  false,
		}
		paramsJSON, _ := json.Marshal(params)

		evt := cdp.Event{
			Method: "Network.loadingFailed",
			Params: json.RawMessage(paramsJSON),
		}

		d.handleLoadingFailed(evt)

		entries := d.networkBuf.All()
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}

		entry := entries[0]
		if !entry.Failed {
			t.Error("Failed should be true")
		}
		if entry.Error != "net::ERR_CONNECTION_REFUSED" {
			t.Errorf("Error = %q, want 'net::ERR_CONNECTION_REFUSED'", entry.Error)
		}
		if entry.ResponseTime == 0 {
			t.Error("ResponseTime should be set")
		}
		if entry.Duration <= 0 {
			t.Error("Duration should be positive")
		}
	})

	t.Run("canceled request", func(t *testing.T) {
		d := New(DefaultConfig())

		requestEntry := ipc.NetworkEntry{
			RequestID:   "req-789",
			URL:         "https://example.com/slow",
			Method:      "GET",
			RequestTime: time.Now().Add(-50 * time.Millisecond).UnixMilli(),
		}
		d.networkBuf.Push(requestEntry)

		params := map[string]any{
			"requestId": "req-789",
			"errorText": "",
			"canceled":  true,
		}
		paramsJSON, _ := json.Marshal(params)

		evt := cdp.Event{
			Method: "Network.loadingFailed",
			Params: json.RawMessage(paramsJSON),
		}

		d.handleLoadingFailed(evt)

		entries := d.networkBuf.All()
		entry := entries[0]
		if !entry.Failed {
			t.Error("Failed should be true")
		}
		if entry.Error != "canceled" {
			t.Errorf("Error = %q, want 'canceled'", entry.Error)
		}
	})

	t.Run("no matching request", func(t *testing.T) {
		d := New(DefaultConfig())

		// Don't add any request - should not panic
		params := map[string]any{
			"requestId": "nonexistent",
			"errorText": "error",
			"canceled":  false,
		}
		paramsJSON, _ := json.Marshal(params)

		evt := cdp.Event{
			Method: "Network.loadingFailed",
			Params: json.RawMessage(paramsJSON),
		}

		// Should not panic
		d.handleLoadingFailed(evt)
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestDaemon_handleLoadingFinished_UsesSessionID tests that handleLoadingFinished
// calls Network.getResponseBody with the correct session ID from the event.
// This test was added to catch a bug where SendContext (browser-level, no session ID)
// was used instead of SendToSession (with the event's session ID).
func TestDaemon_handleLoadingFinished_UsesSessionID(t *testing.T) {
	d := New(DefaultConfig())

	// Add a request to the buffer that will match the loading finished event
	requestEntry := ipc.NetworkEntry{
		RequestID:   "req-789",
		URL:         "https://example.com/api/data",
		Method:      "GET",
		MimeType:    "application/json",
		RequestTime: time.Now().Add(-100 * time.Millisecond).UnixMilli(),
	}
	d.networkBuf.Push(requestEntry)

	// Create a mock CDP connection that captures requests
	mockConn := newSessionCapturingMockConn()
	d.cdp = cdp.NewClient(mockConn)

	// Simulate a loading finished event WITH a session ID
	eventSessionID := "session-abc-123"
	params := map[string]any{
		"requestId":         "req-789",
		"encodedDataLength": int64(1234),
	}
	paramsJSON, _ := json.Marshal(params)

	evt := cdp.Event{
		Method:    "Network.loadingFinished",
		Params:    json.RawMessage(paramsJSON),
		SessionID: eventSessionID, // This session ID should be used for the CDP call
	}

	// Call handleLoadingFinished - this should trigger a Network.getResponseBody call
	d.handleLoadingFinished(evt)

	// Wait briefly for the async CDP call
	time.Sleep(50 * time.Millisecond)

	// Verify that the CDP request used the correct session ID
	requests := mockConn.getCapturedRequests()
	if len(requests) == 0 {
		t.Fatal("expected at least one CDP request to be sent")
	}

	// Find the Network.getResponseBody request
	var found bool
	for _, req := range requests {
		if req.Method == "Network.getResponseBody" {
			found = true
			if req.SessionID != eventSessionID {
				t.Errorf("Network.getResponseBody was called with sessionId=%q, want %q",
					req.SessionID, eventSessionID)
			}
			break
		}
	}

	if !found {
		t.Error("Network.getResponseBody was not called")
	}

	d.cdp.Close()
}

// sessionCapturingMockConn is a mock CDP connection that captures all requests
// and their session IDs for verification in tests.
type sessionCapturingMockConn struct {
	mu        sync.Mutex
	requests  []cdp.Request
	responses chan []byte
	closed    bool
	closeCh   chan struct{}
}

func newSessionCapturingMockConn() *sessionCapturingMockConn {
	return &sessionCapturingMockConn{
		responses: make(chan []byte, 100),
		closeCh:   make(chan struct{}),
	}
}

func (m *sessionCapturingMockConn) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	select {
	case resp := <-m.responses:
		return websocket.MessageText, resp, nil
	case <-m.closeCh:
		return 0, nil, errors.New("connection closed")
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	}
}

func (m *sessionCapturingMockConn) Write(ctx context.Context, typ websocket.MessageType, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return errors.New("connection closed")
	}

	// Parse and capture the request
	var req cdp.Request
	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}
	m.requests = append(m.requests, req)

	// Send back a success response
	resp := map[string]any{
		"id":     req.ID,
		"result": map[string]any{"body": "test body", "base64Encoded": false},
	}
	respData, _ := json.Marshal(resp)
	m.responses <- respData

	return nil
}

func (m *sessionCapturingMockConn) Close(code websocket.StatusCode, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		close(m.closeCh)
	}
	return nil
}

func (m *sessionCapturingMockConn) getCapturedRequests() []cdp.Request {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]cdp.Request, len(m.requests))
	copy(result, m.requests)
	return result
}
