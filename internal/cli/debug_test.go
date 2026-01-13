package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

// captureStderr captures stderr output during test execution.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

// enableDebug enables debug mode for the duration of the test.
func enableDebug(t *testing.T) {
	old := Debug
	Debug = true
	t.Cleanup(func() { Debug = old })
}

func TestDebugfFormat(t *testing.T) {
	enableDebug(t)

	output := captureStderr(t, func() {
		debugf("TEST", "hello %s", "world")
	})

	// Format should be: [DEBUG] [HH:MM:SS.mmm] [CATEGORY] message
	if !strings.Contains(output, "[DEBUG]") {
		t.Errorf("expected [DEBUG] prefix, got: %s", output)
	}
	if !strings.Contains(output, "[TEST]") {
		t.Errorf("expected [TEST] category, got: %s", output)
	}
	if !strings.Contains(output, "hello world") {
		t.Errorf("expected 'hello world' message, got: %s", output)
	}
	// Check timestamp format pattern [HH:MM:SS.mmm]
	if !strings.Contains(output, ":") || !strings.Contains(output, ".") {
		t.Errorf("expected timestamp with : and ., got: %s", output)
	}
}

func TestDebugfNoOutputWhenDisabled(t *testing.T) {
	// Ensure debug is off
	old := Debug
	Debug = false
	defer func() { Debug = old }()

	output := captureStderr(t, func() {
		debugf("TEST", "should not appear")
	})

	if output != "" {
		t.Errorf("expected no output when debug disabled, got: %s", output)
	}
}

func TestDebugRequest(t *testing.T) {
	enableDebug(t)

	output := captureStderr(t, func() {
		debugRequest("navigate", "url=example.com")
	})

	if !strings.Contains(output, "[REQUEST]") {
		t.Errorf("expected [REQUEST] category, got: %s", output)
	}
	if !strings.Contains(output, "cmd=navigate") {
		t.Errorf("expected cmd=navigate, got: %s", output)
	}
	if !strings.Contains(output, "url=example.com") {
		t.Errorf("expected url=example.com, got: %s", output)
	}
}

func TestDebugResponse(t *testing.T) {
	enableDebug(t)

	output := captureStderr(t, func() {
		debugResponse(true, 1024, 50*time.Millisecond)
	})

	if !strings.Contains(output, "[RESPONSE]") {
		t.Errorf("expected [RESPONSE] category, got: %s", output)
	}
	if !strings.Contains(output, "ok=true") {
		t.Errorf("expected ok=true, got: %s", output)
	}
	if !strings.Contains(output, "size=1024 bytes") {
		t.Errorf("expected size=1024 bytes, got: %s", output)
	}
	if !strings.Contains(output, "50ms") {
		t.Errorf("expected 50ms duration, got: %s", output)
	}
}

func TestDebugFilter(t *testing.T) {
	enableDebug(t)

	output := captureStderr(t, func() {
		debugFilter("--find text", 100, 25)
	})

	if !strings.Contains(output, "[FILTER]") {
		t.Errorf("expected [FILTER] category, got: %s", output)
	}
	if !strings.Contains(output, "--find text: 100 -> 25") {
		t.Errorf("expected filter details, got: %s", output)
	}
}

func TestDebugFile(t *testing.T) {
	enableDebug(t)

	output := captureStderr(t, func() {
		debugFile("wrote", "/tmp/test.html", 4096)
	})

	if !strings.Contains(output, "[FILE]") {
		t.Errorf("expected [FILE] category, got: %s", output)
	}
	if !strings.Contains(output, "wrote 4096 bytes to /tmp/test.html") {
		t.Errorf("expected file details, got: %s", output)
	}
}

func TestDebugTiming(t *testing.T) {
	enableDebug(t)

	output := captureStderr(t, func() {
		debugTiming("navigate", 150*time.Millisecond)
	})

	if !strings.Contains(output, "[TIMING]") {
		t.Errorf("expected [TIMING] category, got: %s", output)
	}
	if !strings.Contains(output, "navigate: 150ms") {
		t.Errorf("expected timing details, got: %s", output)
	}
}

func TestDebugParam(t *testing.T) {
	enableDebug(t)

	output := captureStderr(t, func() {
		debugParam("url=%q wait=%v", "http://example.com", true)
	})

	if !strings.Contains(output, "[PARAM]") {
		t.Errorf("expected [PARAM] category, got: %s", output)
	}
	if !strings.Contains(output, "url=\"http://example.com\"") {
		t.Errorf("expected url param, got: %s", output)
	}
	if !strings.Contains(output, "wait=true") {
		t.Errorf("expected wait param, got: %s", output)
	}
}

func TestStartTimer(t *testing.T) {
	timer := startTimer("test-op")

	// Timer should have a name
	if timer.name != "test-op" {
		t.Errorf("expected name 'test-op', got: %s", timer.name)
	}

	// Wait a short time
	time.Sleep(10 * time.Millisecond)

	// Duration should be at least 10ms
	d := timer.stop()
	if d < 10*time.Millisecond {
		t.Errorf("expected duration >= 10ms, got: %v", d)
	}
}

func TestTimerLog(t *testing.T) {
	enableDebug(t)

	output := captureStderr(t, func() {
		timer := startTimer("test-op")
		time.Sleep(5 * time.Millisecond)
		timer.log()
	})

	if !strings.Contains(output, "[TIMING]") {
		t.Errorf("expected [TIMING] category, got: %s", output)
	}
	if !strings.Contains(output, "test-op:") {
		t.Errorf("expected 'test-op:' in output, got: %s", output)
	}
}

func TestDebugHelpersNoOutputWhenDisabled(t *testing.T) {
	// Ensure debug is off
	old := Debug
	Debug = false
	defer func() { Debug = old }()

	tests := []struct {
		name string
		fn   func()
	}{
		{"debugRequest", func() { debugRequest("cmd", "params") }},
		{"debugResponse", func() { debugResponse(true, 100, time.Millisecond) }},
		{"debugFilter", func() { debugFilter("name", 10, 5) }},
		{"debugFile", func() { debugFile("op", "path", 100) }},
		{"debugTiming", func() { debugTiming("op", time.Second) }},
		{"debugParam", func() { debugParam("key=%v", "value") }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output := captureStderr(t, tc.fn)
			if output != "" {
				t.Errorf("expected no output when debug disabled, got: %s", output)
			}
		})
	}
}
