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

func TestExecuteArgs_recognizedCommand(t *testing.T) {
	// Set up mock factory with proper status response
	statusData := ipc.StatusData{
		Running: true,
		URL:     "https://example.com",
		Title:   "Test",
		PID:     12345,
	}
	statusJSON, _ := json.Marshal(statusData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
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
	_, w, _ := os.Pipe()
	os.Stdout = w

	recognized, err := ExecuteArgs([]string{"status"})

	w.Close()
	os.Stdout = old

	if !recognized {
		t.Error("ExecuteArgs should recognize 'status' command")
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExecuteArgs_unrecognizedCommand(t *testing.T) {
	recognized, err := ExecuteArgs([]string{"nonexistent-command"})

	if recognized {
		t.Error("ExecuteArgs should not recognize 'nonexistent-command'")
	}
	if err != nil {
		t.Errorf("unexpected error for unrecognized command: %v", err)
	}
}

func TestExecuteArgs_emptyArgs(t *testing.T) {
	recognized, err := ExecuteArgs([]string{})

	if recognized {
		t.Error("ExecuteArgs should not recognize empty args")
	}
	if err != nil {
		t.Errorf("unexpected error for empty args: %v", err)
	}
}

func TestDirectExecutorFactory(t *testing.T) {
	handlerCalled := false
	receivedCmd := ""

	handler := func(req ipc.Request) ipc.Response {
		handlerCalled = true
		receivedCmd = req.Cmd
		return ipc.SuccessResponse(map[string]string{"result": "ok"})
	}

	factory := NewDirectExecutorFactory(handler)

	// Test IsDaemonRunning always returns true
	if !factory.IsDaemonRunning() {
		t.Error("DirectExecutorFactory.IsDaemonRunning() should always return true")
	}

	// Test NewExecutor returns working executor
	exec, err := factory.NewExecutor()
	if err != nil {
		t.Fatalf("NewExecutor() error: %v", err)
	}

	resp, err := exec.Execute(ipc.Request{Cmd: "test"})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if !handlerCalled {
		t.Error("handler was not called")
	}
	if receivedCmd != "test" {
		t.Errorf("received cmd = %q, want %q", receivedCmd, "test")
	}
	if !resp.OK {
		t.Error("response.OK should be true")
	}

	// Test Close is a no-op
	if err := exec.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}
}

func TestExecuteArgs_resetsFlagsBetweenCalls(t *testing.T) {
	// This test verifies that flags are reset between REPL command executions
	consoleData := ipc.ConsoleData{
		Entries: []ipc.ConsoleEntry{
			{Type: "log", Text: "1", Timestamp: 1},
			{Type: "log", Text: "2", Timestamp: 2},
			{Type: "log", Text: "3", Timestamp: 3},
			{Type: "log", Text: "4", Timestamp: 4},
			{Type: "log", Text: "5", Timestamp: 5},
		},
		Count: 5,
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

	// First call with --tail 2
	old := os.Stdout
	r1, w1, _ := os.Pipe()
	os.Stdout = w1

	ExecuteArgs([]string{"console", "--tail", "2"})

	w1.Close()
	os.Stdout = old

	var buf1 bytes.Buffer
	buf1.ReadFrom(r1)

	var result1 map[string]any
	json.Unmarshal(buf1.Bytes(), &result1)

	count1 := result1["count"].(float64)
	if count1 != 2 {
		t.Errorf("first call with --tail 2: count = %v, want 2", count1)
	}

	// Second call without flags - should show all 5
	r2, w2, _ := os.Pipe()
	os.Stdout = w2

	ExecuteArgs([]string{"console"})

	w2.Close()
	os.Stdout = old

	var buf2 bytes.Buffer
	buf2.ReadFrom(r2)

	var result2 map[string]any
	json.Unmarshal(buf2.Bytes(), &result2)

	count2 := result2["count"].(float64)
	if count2 != 5 {
		t.Errorf("second call without flags: count = %v, want 5 (flags should be reset)", count2)
	}
}

// Network command tests

func TestParseStatusPatterns(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		status   int
		want     bool
		wantErr  bool
	}{
		{"exact match", []string{"200"}, 200, true, false},
		{"exact no match", []string{"200"}, 404, false, false},
		{"wildcard 4xx match", []string{"4xx"}, 404, true, false},
		{"wildcard 4xx no match", []string{"4xx"}, 500, false, false},
		{"wildcard 5xx match", []string{"5xx"}, 503, true, false},
		{"wildcard 2xx match", []string{"2xx"}, 201, true, false},
		{"range match", []string{"200-299"}, 250, true, false},
		{"range no match", []string{"200-299"}, 300, false, false},
		{"multiple patterns", []string{"4xx", "5xx"}, 500, true, false},
		{"invalid pattern", []string{"abc"}, 200, false, true},
		{"invalid wildcard", []string{"6xx"}, 200, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matchers, err := parseStatusPatterns(tt.patterns)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseStatusPatterns() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			matched := false
			for _, m := range matchers {
				if m.matches(tt.status) {
					matched = true
					break
				}
			}
			if matched != tt.want {
				t.Errorf("status %d match = %v, want %v", tt.status, matched, tt.want)
			}
		})
	}
}

func TestMatchesStringSlice(t *testing.T) {
	tests := []struct {
		name  string
		value string
		slice []string
		want  bool
	}{
		{"exact match", "GET", []string{"GET", "POST"}, true},
		{"case insensitive", "get", []string{"GET", "POST"}, true},
		{"no match", "DELETE", []string{"GET", "POST"}, false},
		{"empty slice", "GET", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesStringSlice(tt.value, tt.slice); got != tt.want {
				t.Errorf("matchesStringSlice(%q, %v) = %v, want %v", tt.value, tt.slice, got, tt.want)
			}
		})
	}
}

func TestApplyNetworkLimiting(t *testing.T) {
	entries := []ipc.NetworkEntry{
		{RequestID: "1", URL: "https://example.com/1"},
		{RequestID: "2", URL: "https://example.com/2"},
		{RequestID: "3", URL: "https://example.com/3"},
		{RequestID: "4", URL: "https://example.com/4"},
		{RequestID: "5", URL: "https://example.com/5"},
	}

	tests := []struct {
		name      string
		head      int
		tail      int
		rangeStr  string
		wantCount int
		wantFirst string
		wantLast  string
		wantErr   bool
	}{
		{"no limit", 0, 0, "", 5, "1", "5", false},
		{"head 2", 2, 0, "", 2, "1", "2", false},
		{"head exceeds length", 10, 0, "", 5, "1", "5", false},
		{"tail 2", 0, 2, "", 2, "4", "5", false},
		{"tail exceeds length", 0, 10, "", 5, "1", "5", false},
		{"range 1-3", 0, 0, "1-3", 2, "2", "3", false},
		{"range 0-5", 0, 0, "0-5", 5, "1", "5", false},
		{"range start >= end", 0, 0, "3-2", 0, "", "", false},
		{"invalid range format", 0, 0, "abc", 0, "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := applyNetworkLimiting(entries, tt.head, tt.tail, tt.rangeStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("applyNetworkLimiting() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(result) != tt.wantCount {
				t.Errorf("got %d entries, want %d", len(result), tt.wantCount)
				return
			}
			if tt.wantCount > 0 {
				if result[0].RequestID != tt.wantFirst {
					t.Errorf("first entry = %s, want %s", result[0].RequestID, tt.wantFirst)
				}
				if result[len(result)-1].RequestID != tt.wantLast {
					t.Errorf("last entry = %s, want %s", result[len(result)-1].RequestID, tt.wantLast)
				}
			}
		})
	}
}

func TestRunNetwork_DaemonNotRunning(t *testing.T) {
	restore := setMockFactory(&mockFactory{
		daemonRunning: false,
	})
	defer restore()

	// Capture stderr for error output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runNetwork(networkCmd, []string{})

	w.Close()
	os.Stderr = oldStderr

	if err == nil {
		t.Error("expected error when daemon not running")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	var resp map[string]any
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if resp["ok"] != false {
		t.Error("expected ok=false in error response")
	}
}

func TestRunNetwork_Success(t *testing.T) {
	networkData := ipc.NetworkData{
		Entries: []ipc.NetworkEntry{
			{
				RequestID:   "1",
				URL:         "https://api.example.com/users",
				Method:      "GET",
				Status:      200,
				MimeType:    "application/json",
				RequestTime: 1734151712450,
				Duration:    0.234,
			},
			{
				RequestID:   "2",
				URL:         "https://api.example.com/posts",
				Method:      "POST",
				Status:      201,
				MimeType:    "application/json",
				RequestTime: 1734151712789,
				Duration:    0.567,
			},
		},
		Count: 2,
	}
	networkJSON, _ := json.Marshal(networkData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "network" {
				t.Errorf("expected cmd=network, got %s", req.Cmd)
			}
			return ipc.Response{OK: true, Data: networkJSON}, nil
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

	err := runNetwork(networkCmd, []string{})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if result["ok"] != true {
		t.Error("expected ok=true")
	}
	if result["count"].(float64) != 2 {
		t.Errorf("expected count=2, got %v", result["count"])
	}
}

// Target command tests

func TestRunTarget_DaemonNotRunning(t *testing.T) {
	restore := setMockFactory(&mockFactory{
		daemonRunning: false,
	})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runTarget(targetCmd, []string{})

	w.Close()
	os.Stderr = oldStderr

	if err == nil {
		t.Error("expected error when daemon not running")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var resp map[string]any
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if resp["ok"] != false {
		t.Error("expected ok=false in error response")
	}
}

func TestRunTarget_ListSessions(t *testing.T) {
	targetData := ipc.TargetData{
		ActiveSession: "session-abc",
		Sessions: []ipc.PageSession{
			{ID: "session-abc", URL: "https://example.com", Title: "Example"},
			{ID: "session-def", URL: "https://test.com", Title: "Test Page"},
		},
	}
	targetJSON, _ := json.Marshal(targetData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "target" {
				t.Errorf("expected cmd=target, got %s", req.Cmd)
			}
			if req.Target != "" {
				t.Errorf("expected empty target for list, got %s", req.Target)
			}
			return ipc.Response{OK: true, Data: targetJSON}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runTarget(targetCmd, []string{})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if result["ok"] != true {
		t.Error("expected ok=true")
	}
	if result["activeSession"] != "session-abc" {
		t.Errorf("expected activeSession=session-abc, got %v", result["activeSession"])
	}

	sessions := result["sessions"].([]any)
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestRunTarget_SwitchSession(t *testing.T) {
	targetData := ipc.TargetData{
		ActiveSession: "session-def",
		Sessions: []ipc.PageSession{
			{ID: "session-abc", URL: "https://example.com", Title: "Example"},
			{ID: "session-def", URL: "https://test.com", Title: "Test Page"},
		},
	}
	targetJSON, _ := json.Marshal(targetData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "target" {
				t.Errorf("expected cmd=target, got %s", req.Cmd)
			}
			if req.Target != "test" {
				t.Errorf("expected target=test, got %s", req.Target)
			}
			return ipc.Response{OK: true, Data: targetJSON}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runTarget(targetCmd, []string{"test"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if result["ok"] != true {
		t.Error("expected ok=true")
	}
	if result["activeSession"] != "session-def" {
		t.Errorf("expected activeSession=session-def, got %v", result["activeSession"])
	}
}

func TestRunTarget_AmbiguousMatch(t *testing.T) {
	// Daemon returns error with multiple matches
	matchData := struct {
		Error   string            `json:"error"`
		Matches []ipc.PageSession `json:"matches"`
	}{
		Error: "ambiguous query 'test', matches multiple sessions",
		Matches: []ipc.PageSession{
			{ID: "session-abc", URL: "https://test1.com", Title: "Test 1"},
			{ID: "session-def", URL: "https://test2.com", Title: "Test 2"},
		},
	}
	matchJSON, _ := json.Marshal(matchData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: false, Error: matchData.Error, Data: matchJSON}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_ = runTarget(targetCmd, []string{"test"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if result["ok"] != false {
		t.Error("expected ok=false for ambiguous match")
	}
	if result["error"] == nil || result["error"] == "" {
		t.Error("expected error message")
	}
	if result["matches"] == nil {
		t.Error("expected matches in response")
	}
}

func TestTruncateID(t *testing.T) {
	tests := []struct {
		id   string
		n    int
		want string
	}{
		{"short", 8, "short"},
		{"exactly8", 8, "exactly8"},
		{"toolongid123456", 8, "toolongi..."},
		{"", 8, ""},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			if got := truncateID(tt.id, tt.n); got != tt.want {
				t.Errorf("truncateID(%q, %d) = %q, want %q", tt.id, tt.n, got, tt.want)
			}
		})
	}
}

func TestTruncateTitle(t *testing.T) {
	tests := []struct {
		title string
		max   int
		want  string
	}{
		{"Short title", 40, "Short title"},
		{"  Padded  ", 40, "Padded"},
		{"This is a very long title that exceeds the maximum length allowed", 40, "This is a very long title that exceed..."},
		{"", 40, ""},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			if got := truncateTitle(tt.title, tt.max); got != tt.want {
				t.Errorf("truncateTitle(%q, %d) = %q, want %q", tt.title, tt.max, got, tt.want)
			}
		})
	}
}

// Screenshot command tests

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		// Basic cases
		{"simple title", "Example Domain", "example-domain"},
		{"with punctuation", "React App - Development Server!", "react-app-development-server"},
		{"mixed case", "MyWebApp 2024", "mywebapp-2024"},

		// Truncation cases
		{"long title truncated", "JSONPlaceholder - Free Fake REST API for Testing", "jsonplaceholder-free-fake-re"},
		{"exactly 30 chars", "123456789012345678901234567890", "123456789012345678901234567890"},
		{"over 30 chars", "1234567890123456789012345678901", "123456789012345678901234567890"},
		{"truncate at special char", "abcdefghijklmnopqrstuvwxyz!@#$", "abcdefghijklmnopqrstuvwxyz"},
		{"truncate creates trailing hyphen", "abcdefghijklmnopqrstuvwxyz----extra", "abcdefghijklmnopqrstuvwxyz"},

		// Whitespace cases
		{"multiple spaces", "   Lots   of---Spaces!!!   ", "lots-of-spaces"},
		{"empty string", "", "untitled"},
		{"only whitespace", "   ", "untitled"},
		{"whitespace in middle", "foo   bar", "foo-bar"},
		{"tabs and newlines", "\n\t  Title  \n\t", "title"},

		// Special character cases
		{"only non-alphanumeric", "!@#$%^&*()", "untitled"},
		{"single hyphen", "-", "untitled"},
		{"multiple hyphens only", "-----", "untitled"},
		{"special unicode", "Café ☕ 日本", "caf"},
		{"long special chars only", "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!", "untitled"},

		// Hyphen handling
		{"trailing hyphens", "---Title---", "title"},
		{"leading hyphens only", "---title", "title"},
		{"multiple consecutive hyphens", "foo---bar___baz", "foo-bar-baz"},
		{"underscores to hyphens", "foo__bar__baz", "foo-bar-baz"},

		// Single/minimal cases
		{"single character", "A", "a"},
		{"single digit", "1", "1"},
		{"numbers only", "123456", "123456"},

		// Real-world examples
		{"github url style", "my-awesome-project", "my-awesome-project"},
		{"windows filename", "file:name*.txt", "file-name-txt"},
		{"path separators", "path/to/file", "path-to-file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeTitle(tt.title)
			if got != tt.want {
				t.Errorf("normalizeTitle(%q) = %q, want %q", tt.title, got, tt.want)
			}
		})
	}
}

func TestRunScreenshot_DaemonNotRunning(t *testing.T) {
	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runScreenshot(screenshotCmd, []string{})

	w.Close()
	os.Stderr = oldStderr

	if err == nil {
		t.Error("expected error when daemon not running")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var resp map[string]any
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if resp["ok"] != false {
		t.Error("expected ok=false in error response")
	}
}

func TestRunScreenshot_Success(t *testing.T) {
	// Create temp directory for screenshots
	tmpDir := t.TempDir()

	// Mock screenshot data (minimal valid PNG header)
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	screenshotData := ipc.ScreenshotData{
		Data: pngData,
	}
	screenshotJSON, _ := json.Marshal(screenshotData)

	statusData := ipc.StatusData{
		Running: true,
		ActiveSession: &ipc.PageSession{
			ID:    "session-123",
			URL:   "https://example.com",
			Title: "Example Domain",
		},
	}
	statusJSON, _ := json.Marshal(statusData)

	callCount := 0
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			callCount++
			if req.Cmd == "screenshot" {
				return ipc.Response{OK: true, Data: screenshotJSON}, nil
			}
			if req.Cmd == "status" {
				return ipc.Response{OK: true, Data: statusJSON}, nil
			}
			t.Errorf("unexpected command: %s", req.Cmd)
			return ipc.Response{OK: false}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	// Set custom output path in temp dir
	screenshotOutput = tmpDir + "/test-screenshot.png"
	defer func() { screenshotOutput = "" }()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runScreenshot(screenshotCmd, []string{})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if result["ok"] != true {
		t.Error("expected ok=true")
	}

	path, ok := result["path"].(string)
	if !ok {
		t.Fatal("expected path in response")
	}

	// Verify file was created
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("failed to read screenshot file: %v", err)
	}

	if !bytes.Equal(data, pngData) {
		t.Errorf("screenshot data mismatch")
	}
}

func TestRunScreenshot_CustomOutput(t *testing.T) {
	tmpDir := t.TempDir()
	customPath := tmpDir + "/custom/screenshot.png"

	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	screenshotData := ipc.ScreenshotData{Data: pngData}
	screenshotJSON, _ := json.Marshal(screenshotData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd == "screenshot" {
				var params ipc.ScreenshotParams
				json.Unmarshal(req.Params, &params)
				return ipc.Response{OK: true, Data: screenshotJSON}, nil
			}
			return ipc.Response{OK: false}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	screenshotOutput = customPath
	defer func() { screenshotOutput = "" }()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runScreenshot(screenshotCmd, []string{})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	json.Unmarshal(buf.Bytes(), &result)

	// Verify custom path is used
	if result["path"] != customPath {
		t.Errorf("expected path=%s, got %v", customPath, result["path"])
	}

	// Verify file exists at custom path
	if _, err := os.Stat(customPath); err != nil {
		t.Errorf("screenshot not created at custom path: %v", err)
	}
}

func TestRunScreenshot_FullPage(t *testing.T) {
	pngData := []byte{0x89, 0x50, 0x4E, 0x47}
	screenshotData := ipc.ScreenshotData{Data: pngData}
	screenshotJSON, _ := json.Marshal(screenshotData)

	var capturedFullPage bool
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd == "screenshot" {
				var params ipc.ScreenshotParams
				json.Unmarshal(req.Params, &params)
				capturedFullPage = params.FullPage
				return ipc.Response{OK: true, Data: screenshotJSON}, nil
			}
			return ipc.Response{OK: false}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	tmpDir := t.TempDir()
	screenshotOutput = tmpDir + "/test.png"
	defer func() { screenshotOutput = "" }()

	screenshotFullPage = true
	defer func() { screenshotFullPage = false }()

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runScreenshot(screenshotCmd, []string{})

	w.Close()
	os.Stdout = old

	if !capturedFullPage {
		t.Error("expected FullPage=true in screenshot params")
	}
}

// HTML command tests

func TestRunHTML_DaemonNotRunning(t *testing.T) {
	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runHTML(htmlCmd, []string{})

	w.Close()
	os.Stderr = oldStderr

	if err == nil {
		t.Error("expected error when daemon not running")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var resp map[string]any
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if resp["ok"] != false {
		t.Error("expected ok=false in error response")
	}
}

func TestRunHTML_FullPage(t *testing.T) {
	tmpDir := t.TempDir()

	htmlContent := "<!DOCTYPE html><html><head><title>Test</title></head><body><h1>Hello</h1></body></html>"
	htmlData := ipc.HTMLData{HTML: htmlContent}
	htmlJSON, _ := json.Marshal(htmlData)

	statusData := ipc.StatusData{
		Running: true,
		ActiveSession: &ipc.PageSession{
			ID:    "session-123",
			URL:   "https://example.com",
			Title: "Test Page",
		},
	}
	statusJSON, _ := json.Marshal(statusData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd == "html" {
				return ipc.Response{OK: true, Data: htmlJSON}, nil
			}
			if req.Cmd == "status" {
				return ipc.Response{OK: true, Data: statusJSON}, nil
			}
			t.Errorf("unexpected command: %s", req.Cmd)
			return ipc.Response{OK: false}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	htmlOutput = tmpDir + "/test.html"
	defer func() { htmlOutput = "" }()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runHTML(htmlCmd, []string{})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if result["ok"] != true {
		t.Error("expected ok=true")
	}

	path, ok := result["path"].(string)
	if !ok {
		t.Fatal("expected path in response")
	}

	// Verify file was created
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("failed to read HTML file: %v", err)
	}

	if string(data) != htmlContent {
		t.Errorf("HTML content mismatch: got %q, want %q", string(data), htmlContent)
	}
}

func TestRunHTML_WithSelector(t *testing.T) {
	tmpDir := t.TempDir()

	htmlContent := `<div class="content">Test Content</div>`
	htmlData := ipc.HTMLData{HTML: htmlContent}
	htmlJSON, _ := json.Marshal(htmlData)

	var capturedSelector string
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd == "html" {
				var params ipc.HTMLParams
				json.Unmarshal(req.Params, &params)
				capturedSelector = params.Selector
				return ipc.Response{OK: true, Data: htmlJSON}, nil
			}
			return ipc.Response{OK: false}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	htmlOutput = tmpDir + "/test.html"
	defer func() { htmlOutput = "" }()

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runHTML(htmlCmd, []string{".content"})

	w.Close()
	os.Stdout = old

	if capturedSelector != ".content" {
		t.Errorf("expected selector='.content', got %q", capturedSelector)
	}
}

func TestRunHTML_CustomOutput(t *testing.T) {
	tmpDir := t.TempDir()
	customPath := tmpDir + "/custom/page.html"

	htmlContent := "<!DOCTYPE html><html><body>Test</body></html>"
	htmlData := ipc.HTMLData{HTML: htmlContent}
	htmlJSON, _ := json.Marshal(htmlData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd == "html" {
				return ipc.Response{OK: true, Data: htmlJSON}, nil
			}
			return ipc.Response{OK: false}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	htmlOutput = customPath
	defer func() { htmlOutput = "" }()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runHTML(htmlCmd, []string{})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	json.Unmarshal(buf.Bytes(), &result)

	// Verify custom path is used
	if result["path"] != customPath {
		t.Errorf("expected path=%s, got %v", customPath, result["path"])
	}

	// Verify file exists at custom path
	if _, err := os.Stat(customPath); err != nil {
		t.Errorf("HTML not created at custom path: %v", err)
	}
}
