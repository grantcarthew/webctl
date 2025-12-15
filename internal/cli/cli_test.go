package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/grantcarthew/webctl/internal/executor"
	"github.com/grantcarthew/webctl/internal/ipc"
)

// mockExecutor implements executor.Executor for testing.
type mockExecutor struct {
	executeFunc func(req ipc.Request) (ipc.Response, error)
	closed      bool
}

func (m *mockExecutor) Execute(req ipc.Request) (ipc.Response, error) {
	if m.executeFunc != nil {
		return m.executeFunc(req)
	}
	return ipc.Response{OK: true}, nil
}

func (m *mockExecutor) Close() error {
	m.closed = true
	return nil
}

// mockFactory implements ExecutorFactory for testing.
type mockFactory struct {
	executor      executor.Executor
	newErr        error
	daemonRunning bool
}

func (m *mockFactory) NewExecutor() (executor.Executor, error) {
	if m.newErr != nil {
		return nil, m.newErr
	}
	return m.executor, nil
}

func (m *mockFactory) IsDaemonRunning() bool {
	return m.daemonRunning
}

// setMockFactory replaces the package execFactory and returns a restore function.
func setMockFactory(f ExecutorFactory) func() {
	old := execFactory
	execFactory = f
	return func() { execFactory = old }
}

func TestOutputSuccess(t *testing.T) {

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := map[string]string{"message": "test"}
	err := outputSuccess(data)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if result["ok"] != true {
		t.Errorf("expected ok=true, got %v", result["ok"])
	}

	resultData, ok := result["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data to be map, got %T", result["data"])
	}

	if resultData["message"] != "test" {
		t.Errorf("expected message=test, got %v", resultData["message"])
	}
}

func TestOutputError(t *testing.T) {

	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := outputError("something went wrong")

	w.Close()
	os.Stderr = old

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.Error() != "something went wrong" {
		t.Errorf("expected error message 'something went wrong', got %v", err.Error())
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if result["ok"] != false {
		t.Errorf("expected ok=false, got %v", result["ok"])
	}

	if result["error"] != "something went wrong" {
		t.Errorf("expected error='something went wrong', got %v", result["error"])
	}
}

func TestRunStatus_DaemonNotRunning(t *testing.T) {

	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runStatus(nil, nil)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if result["ok"] != true {
		t.Errorf("expected ok=true, got %v", result["ok"])
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data to be map, got %T", result["data"])
	}

	if data["running"] != false {
		t.Errorf("expected running=false, got %v", data["running"])
	}
}

func TestRunStatus_DaemonRunning(t *testing.T) {

	statusData := ipc.StatusData{
		Running: true,
		URL:     "https://example.com",
		Title:   "Example",
		PID:     12345,
	}
	statusJSON, _ := json.Marshal(statusData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "status" {
				t.Errorf("expected cmd=status, got %s", req.Cmd)
			}
			return ipc.Response{OK: true, Data: statusJSON}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runStatus(nil, nil)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data to be map, got %T", result["data"])
	}

	if data["running"] != true {
		t.Errorf("expected running=true, got %v", data["running"])
	}
	if data["url"] != "https://example.com" {
		t.Errorf("expected url=https://example.com, got %v", data["url"])
	}

	if !exec.closed {
		t.Error("expected executor to be closed")
	}
}

func TestRunStop_Success(t *testing.T) {

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "shutdown" {
				t.Errorf("expected cmd=shutdown, got %s", req.Cmd)
			}
			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runStop(nil, nil)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if result["ok"] != true {
		t.Errorf("expected ok=true, got %v", result["ok"])
	}

	if !exec.closed {
		t.Error("expected executor to be closed")
	}
}

func TestRunStop_NewExecutorError(t *testing.T) {

	restore := setMockFactory(&mockFactory{
		newErr: errors.New("daemon is not running"),
	})
	defer restore()

	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runStop(nil, nil)

	w.Close()
	os.Stderr = old

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if result["ok"] != false {
		t.Errorf("expected ok=false, got %v", result["ok"])
	}
}

func TestRunClear_AllBuffers(t *testing.T) {

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "clear" {
				t.Errorf("expected cmd=clear, got %s", req.Cmd)
			}
			if req.Target != "" {
				t.Errorf("expected target='', got %s", req.Target)
			}
			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runClear(nil, []string{})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	data := result["data"].(map[string]any)
	if data["message"] != "all buffers cleared" {
		t.Errorf("expected 'all buffers cleared', got %v", data["message"])
	}
}

func TestRunClear_ConsoleOnly(t *testing.T) {

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Target != "console" {
				t.Errorf("expected target=console, got %s", req.Target)
			}
			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runClear(nil, []string{"console"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	data := result["data"].(map[string]any)
	if data["message"] != "console buffer cleared" {
		t.Errorf("expected 'console buffer cleared', got %v", data["message"])
	}
}

func TestRunClear_InvalidTarget(t *testing.T) {

	exec := &mockExecutor{}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	// Capture stderr
	old := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	err := runClear(nil, []string{"invalid"})

	w.Close()
	os.Stderr = old

	if err == nil {
		t.Fatal("expected error for invalid target")
	}

	if err.Error() != "invalid target: must be 'console' or 'network'" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunStart_DaemonAlreadyRunning(t *testing.T) {

	restore := setMockFactory(&mockFactory{daemonRunning: true})
	defer restore()

	// Capture stderr
	old := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	err := runStart(nil, nil)

	w.Close()
	os.Stderr = old

	if err == nil {
		t.Fatal("expected error when daemon already running")
	}

	if err.Error() != "daemon is already running" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunConsole_DaemonNotRunning(t *testing.T) {

	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	// Capture stderr
	old := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	err := runConsole(nil, nil)

	w.Close()
	os.Stderr = old

	if err == nil {
		t.Fatal("expected error when daemon not running")
	}

	if err.Error() != "daemon not running. Start with: webctl start" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunConsole_Success(t *testing.T) {

	consoleData := ipc.ConsoleData{
		Entries: []ipc.ConsoleEntry{
			{Type: "log", Text: "hello", Timestamp: 1702000000000},
			{Type: "error", Text: "oops", Timestamp: 1702000001000, URL: "https://example.com/app.js", Line: 42},
		},
		Count: 2,
	}
	consoleJSON, _ := json.Marshal(consoleData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "console" {
				t.Errorf("expected cmd=console, got %s", req.Cmd)
			}
			return ipc.Response{OK: true, Data: consoleJSON}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	// Reset flags to defaults
	consoleFormat = ""
	consoleTypes = nil
	consoleHead = 0
	consoleTail = 0
	consoleRange = ""

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runConsole(nil, nil)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if result["ok"] != true {
		t.Errorf("expected ok=true, got %v", result["ok"])
	}

	if result["count"] != float64(2) {
		t.Errorf("expected count=2, got %v", result["count"])
	}

	entries, ok := result["entries"].([]any)
	if !ok {
		t.Fatalf("expected entries to be array, got %T", result["entries"])
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}

	if !exec.closed {
		t.Error("expected executor to be closed")
	}
}

func TestRunConsole_EmptyBuffer(t *testing.T) {

	consoleData := ipc.ConsoleData{
		Entries: []ipc.ConsoleEntry{},
		Count:   0,
	}
	consoleJSON, _ := json.Marshal(consoleData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: true, Data: consoleJSON}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	// Reset flags
	consoleFormat = ""
	consoleTypes = nil
	consoleHead = 0
	consoleTail = 0
	consoleRange = ""

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runConsole(nil, nil)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if result["count"] != float64(0) {
		t.Errorf("expected count=0, got %v", result["count"])
	}
}

func TestFilterConsoleByType(t *testing.T) {
	entries := []ipc.ConsoleEntry{
		{Type: "log", Text: "log1"},
		{Type: "error", Text: "error1"},
		{Type: "warn", Text: "warn1"},
		{Type: "error", Text: "error2"},
		{Type: "log", Text: "log2"},
	}

	tests := []struct {
		name     string
		types    []string
		expected int
	}{
		{"single type", []string{"error"}, 2},
		{"multiple types", []string{"error", "warn"}, 3},
		{"case insensitive", []string{"ERROR"}, 2},
		{"no match", []string{"info"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterConsoleByType(entries, tt.types)
			if len(filtered) != tt.expected {
				t.Errorf("expected %d entries, got %d", tt.expected, len(filtered))
			}
		})
	}
}

func TestApplyConsoleLimiting(t *testing.T) {
	entries := []ipc.ConsoleEntry{
		{Type: "log", Text: "0"},
		{Type: "log", Text: "1"},
		{Type: "log", Text: "2"},
		{Type: "log", Text: "3"},
		{Type: "log", Text: "4"},
	}

	tests := []struct {
		name        string
		head        int
		tail        int
		rangeStr    string
		expected    int
		firstText   string
		lastText    string
		expectError bool
	}{
		{"no limit", 0, 0, "", 5, "0", "4", false},
		{"head 2", 2, 0, "", 2, "0", "1", false},
		{"head exceeds length", 10, 0, "", 5, "0", "4", false},
		{"tail 2", 0, 2, "", 2, "3", "4", false},
		{"tail exceeds length", 0, 10, "", 5, "0", "4", false},
		{"range 1-3", 0, 0, "1-3", 2, "1", "2", false},
		{"range 0-5", 0, 0, "0-5", 5, "0", "4", false},
		{"range start >= end", 0, 0, "3-2", 0, "", "", false},
		{"invalid range format", 0, 0, "abc", 0, "", "", true},
		{"invalid range no dash", 0, 0, "12", 0, "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := applyConsoleLimiting(entries, tt.head, tt.tail, tt.rangeStr)
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != tt.expected {
				t.Errorf("expected %d entries, got %d", tt.expected, len(result))
			}
			if tt.expected > 0 {
				if result[0].Text != tt.firstText {
					t.Errorf("expected first text=%s, got %s", tt.firstText, result[0].Text)
				}
				if result[len(result)-1].Text != tt.lastText {
					t.Errorf("expected last text=%s, got %s", tt.lastText, result[len(result)-1].Text)
				}
			}
		})
	}
}
