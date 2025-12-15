package daemon

import (
	"encoding/json"
	"testing"
	"time"

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
	monotonic := 12345.678                  // Some arbitrary monotonic time

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
		{"IMAGE/PNG", ".png"},                 // case insensitive
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
