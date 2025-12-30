package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/grantcarthew/webctl/internal/executor"
	"github.com/grantcarthew/webctl/internal/ipc"
)

func init() {
	// Disable colors in tests to avoid ANSI codes in output assertions
	color.NoColor = true
}

// enableJSONOutput sets JSONOutput to true for the duration of the test.
func enableJSONOutput(t *testing.T) {
	old := JSONOutput
	JSONOutput = true
	t.Cleanup(func() { JSONOutput = old })
}

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
	executeFunc   func(req ipc.Request) (ipc.Response, error) // Function to create executors
	newErr        error
	daemonRunning bool
}

func (m *mockFactory) NewExecutor() (executor.Executor, error) {
	if m.newErr != nil {
		return nil, m.newErr
	}
	// If executor is set, return it (for backward compatibility)
	if m.executor != nil {
		return m.executor, nil
	}
	// If executeFunc is set, create a new executor with it
	if m.executeFunc != nil {
		return &mockExecutor{executeFunc: m.executeFunc}, nil
	}
	// Default: return a basic executor
	return &mockExecutor{}, nil
}

func (m *mockFactory) IsDaemonRunning() bool {
	return m.daemonRunning
}

// setMockFactory replaces the package execFactory and returns a restore function.
func setMockFactory(f ExecutorFactory) func() {
	old := execFactory
	execFactory = f
	return func() {
		execFactory = old
		// Also ensure global flags are reset
		Debug = false
		JSONOutput = false
		NoColor = false
	}
}

func TestOutputSuccess(t *testing.T) {
	enableJSONOutput(t)

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
	enableJSONOutput(t)

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
	enableJSONOutput(t)
	// Set JSON output mode for test
	oldJSON := JSONOutput
	JSONOutput = true
	defer func() { JSONOutput = oldJSON }()

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
	enableJSONOutput(t)

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
	enableJSONOutput(t)

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
	enableJSONOutput(t)

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
	enableJSONOutput(t)

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
	enableJSONOutput(t)

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
	enableJSONOutput(t)

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
	enableJSONOutput(t)

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
	enableJSONOutput(t)

	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	// Capture stderr
	old := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	err := runConsoleDefault(nil, nil)

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
	enableJSONOutput(t)

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

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runConsoleShow(consoleShowCmd, nil)

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

	logs, ok := result["logs"].([]any)
	if !ok {
		t.Fatalf("expected logs to be array, got %T", result["logs"])
	}

	if len(logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(logs))
	}

	if !exec.closed {
		t.Error("expected executor to be closed")
	}
}

func TestRunConsole_EmptyBuffer(t *testing.T) {
	enableJSONOutput(t)

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

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runConsoleShow(consoleShowCmd, nil)

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

func TestFilterConsoleByText(t *testing.T) {
	entries := []ipc.ConsoleEntry{
		{Type: "log", Text: "User logged in successfully"},
		{Type: "error", Text: "TypeError: Cannot read property 'name'"},
		{Type: "warn", Text: "Deprecated API usage"},
		{Type: "error", Text: "ReferenceError: foo is not defined"},
		{Type: "log", Text: "Application log entry"},
	}

	tests := []struct {
		name     string
		search   string
		expected int
	}{
		{"exact match", "TypeError", 1},
		{"case insensitive", "typeerror", 1},
		{"partial match", "error", 2},
		{"multiple matches", "log", 2},
		{"no match", "xyz123", 0},
		{"common word", "API", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterConsoleByText(entries, tt.search)
			if len(filtered) != tt.expected {
				t.Errorf("expected %d entries, got %d", tt.expected, len(filtered))
			}
		})
	}
}

func TestRunConsoleDefault_Success(t *testing.T) {
	enableJSONOutput(t)

	consoleData := ipc.ConsoleData{
		Entries: []ipc.ConsoleEntry{
			{Type: "log", Text: "test message", Timestamp: 1702000000000},
		},
		Count: 1,
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

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runConsoleDefault(consoleCmd, nil)

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

	path, ok := result["path"].(string)
	if !ok {
		t.Fatalf("expected path to be string, got %T", result["path"])
	}

	// Verify path is in temp directory
	if !strings.HasPrefix(path, "/tmp/webctl-console/") {
		t.Errorf("expected path to start with /tmp/webctl-console/, got %s", path)
	}

	// Verify filename format: YY-MM-DD-HHMMSS-console.json
	if !strings.HasSuffix(path, "-console.json") {
		t.Errorf("expected path to end with -console.json, got %s", path)
	}

	if !exec.closed {
		t.Error("expected executor to be closed")
	}
}

func TestRunConsoleDefault_UnknownSubcommand(t *testing.T) {
	enableJSONOutput(t)

	restore := setMockFactory(&mockFactory{daemonRunning: true})
	defer restore()

	err := runConsoleDefault(consoleCmd, []string{"invalid"})

	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}

	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected 'unknown command' error, got: %v", err)
	}
}

func TestRunConsoleSave_CustomFilePath(t *testing.T) {
	enableJSONOutput(t)

	consoleData := ipc.ConsoleData{
		Entries: []ipc.ConsoleEntry{
			{Type: "log", Text: "test", Timestamp: 1702000000000},
		},
		Count: 1,
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

	// Create temp file for testing
	tmpFile, err := os.CreateTemp("", "console-test-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = runConsoleSave(consoleSaveCmd, []string{tmpPath})

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

	if result["path"] != tmpPath {
		t.Errorf("expected path=%s, got %v", tmpPath, result["path"])
	}

	// Verify file was written
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	var savedData map[string]any
	if err := json.Unmarshal(data, &savedData); err != nil {
		t.Fatalf("saved file is not valid JSON: %v", err)
	}

	if savedData["ok"] != true {
		t.Error("saved file should contain ok=true")
	}

	logs, ok := savedData["logs"].([]any)
	if !ok {
		t.Fatalf("saved file should contain logs array, got %T", savedData["logs"])
	}

	if len(logs) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(logs))
	}

	if !exec.closed {
		t.Error("expected executor to be closed")
	}
}

func TestRunConsoleSave_DirectoryPath(t *testing.T) {
	enableJSONOutput(t)

	consoleData := ipc.ConsoleData{
		Entries: []ipc.ConsoleEntry{
			{Type: "log", Text: "test", Timestamp: 1702000000000},
		},
		Count: 1,
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

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "console-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = runConsoleSave(consoleSaveCmd, []string{tmpDir})

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

	path, ok := result["path"].(string)
	if !ok {
		t.Fatalf("expected path to be string, got %T", result["path"])
	}

	// Verify path is in the specified directory
	if !strings.HasPrefix(path, tmpDir) {
		t.Errorf("expected path to start with %s, got %s", tmpDir, path)
	}

	// Verify auto-generated filename
	if !strings.HasSuffix(path, "-console.json") {
		t.Errorf("expected path to end with -console.json, got %s", path)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("file was not created")
	}

	if !exec.closed {
		t.Error("expected executor to be closed")
	}
}

func TestRunConsoleShow_RawFlag(t *testing.T) {
	enableJSONOutput(t)

	consoleData := ipc.ConsoleData{
		Entries: []ipc.ConsoleEntry{
			{Type: "log", Text: "test message", Timestamp: 1702000000000},
		},
		Count: 1,
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

	// Set --raw flag on parent command
	consoleCmd.PersistentFlags().Set("raw", "true")
	t.Cleanup(func() { consoleCmd.PersistentFlags().Set("raw", "false") })

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runConsoleShow(consoleShowCmd, nil)

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

	// Raw flag should output JSON
	logs, ok := result["logs"].([]any)
	if !ok {
		t.Fatalf("expected logs to be array, got %T", result["logs"])
	}

	if len(logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(logs))
	}

	if !exec.closed {
		t.Error("expected executor to be closed")
	}
}

func TestRunConsoleShow_CombinedFilters(t *testing.T) {
	enableJSONOutput(t)

	consoleData := ipc.ConsoleData{
		Entries: []ipc.ConsoleEntry{
			{Type: "log", Text: "Starting application", Timestamp: 1},
			{Type: "error", Text: "TypeError: Cannot read property", Timestamp: 2},
			{Type: "warn", Text: "Deprecated API", Timestamp: 3},
			{Type: "error", Text: "ReferenceError: undefined variable", Timestamp: 4},
			{Type: "error", Text: "TypeError: Invalid argument", Timestamp: 5},
			{Type: "log", Text: "Process complete", Timestamp: 6},
		},
		Count: 6,
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

	// Set combined filters on the parent command: --type error --find "TypeError" --tail 2
	consoleCmd.PersistentFlags().Set("type", "error")
	consoleCmd.PersistentFlags().Set("find", "TypeError")
	consoleCmd.PersistentFlags().Set("tail", "2")
	t.Cleanup(func() {
		consoleCmd.PersistentFlags().Set("type", "")
		consoleCmd.PersistentFlags().Set("find", "")
		consoleCmd.PersistentFlags().Set("tail", "0")
	})

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runConsoleShow(consoleShowCmd, nil)

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

	logs, ok := result["logs"].([]any)
	if !ok {
		t.Fatalf("expected logs to be array, got %T", result["logs"])
	}

	// Should get 2 TypeError entries (filtered by type=error, find=TypeError, tail=2)
	// But we only have 2 TypeError entries total, so should get both
	if len(logs) != 2 {
		t.Errorf("expected 2 logs after combined filters, got %d", len(logs))
	}

	if !exec.closed {
		t.Error("expected executor to be closed")
	}
}

func TestRunConsoleShow_FindNoMatches(t *testing.T) {
	enableJSONOutput(t)

	consoleData := ipc.ConsoleData{
		Entries: []ipc.ConsoleEntry{
			{Type: "log", Text: "test message", Timestamp: 1702000000000},
		},
		Count: 1,
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

	// Set --find with no matches on parent command
	consoleCmd.PersistentFlags().Set("find", "nonexistent-string-xyz")
	t.Cleanup(func() { consoleCmd.PersistentFlags().Set("find", "") })

	err := runConsoleShow(consoleShowCmd, nil)

	if err == nil {
		t.Fatal("expected error when no matches found")
	}

	if !strings.Contains(err.Error(), "no matches found") {
		t.Errorf("expected 'no matches found' error, got: %v", err)
	}

	if !exec.closed {
		t.Error("expected executor to be closed")
	}
}

func TestRunConsoleShow_TextOutput(t *testing.T) {
	// Disable JSON output for this test
	oldJSON := JSONOutput
	JSONOutput = false
	defer func() { JSONOutput = oldJSON }()

	consoleData := ipc.ConsoleData{
		Entries: []ipc.ConsoleEntry{
			{Type: "log", Text: "Application started", Timestamp: 1702000000000},
			{Type: "error", Text: "Fatal error occurred", Timestamp: 1702000001000, URL: "https://example.com/app.js", Line: 42},
		},
		Count: 2,
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

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runConsoleShow(consoleShowCmd, nil)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify text format output
	// Format is: [HH:MM:SS] LEVEL Message
	// Should contain formatted log entries with timestamps and messages
	if len(output) == 0 {
		t.Error("expected non-empty output")
	}

	// Check for timestamp format [HH:MM:SS]
	if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
		t.Errorf("expected output to contain timestamp format, got: %s", output)
	}

	// At least one message should be present
	hasLog := strings.Contains(output, "Application started")
	hasError := strings.Contains(output, "Fatal error occurred")
	if !hasLog && !hasError {
		t.Errorf("expected output to contain at least one log message, got: %s", output)
	}

	if !exec.closed {
		t.Error("expected executor to be closed")
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
	// TODO: This test is flaky when run with the full test suite due to global state sharing
	// in Cobra commands. It passes when run in isolation. Need to refactor to use isolated
	// command instances rather than global rootCmd.
	t.Skip("Skipping flaky test - global state isolation issue with Cobra commands")

	// This test verifies that flags are reset between REPL command executions

	// Ensure clean state at start
	Debug = false
	JSONOutput = false
	NoColor = false

	// Log the initial factory state
	t.Logf("Initial execFactory.IsDaemonRunning() = %v", execFactory.IsDaemonRunning())

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

	callCount := 0
	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			callCount++
			// Verify the data can be unmarshaled correctly
			var testData ipc.ConsoleData
			if err := json.Unmarshal(consoleJSON, &testData); err != nil {
				t.Logf("ERROR unmarshaling in test: %v", err)
			} else {
				t.Logf("ExecuteFunc call %d: cmd=%q, entries=%d",
					callCount, req.Cmd, len(testData.Entries))
			}
			return ipc.Response{OK: true, Data: consoleJSON}, nil
		},
	})
	defer restore()

	// First call with --tail 2 --json
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut1, wOut1, _ := os.Pipe()
	rErr1, wErr1, _ := os.Pipe()
	os.Stdout = wOut1
	os.Stderr = wErr1

	recognized, err := ExecuteArgs([]string{"console", "show", "--json", "--tail", "2"})
	if !recognized {
		t.Fatal("command not recognized")
	}
	if err != nil {
		t.Fatalf("ExecuteArgs returned error: %v", err)
	}

	wOut1.Close()
	wErr1.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut1, bufErr1 bytes.Buffer
	bufOut1.ReadFrom(rOut1)
	bufErr1.ReadFrom(rErr1)

	if bufErr1.Len() > 0 {
		t.Logf("First call stderr: %s", bufErr1.String())
	}

	var result1 map[string]any
	if err := json.Unmarshal(bufOut1.Bytes(), &result1); err != nil {
		t.Fatalf("failed to parse first call output: %v, output: %s", err, bufOut1.String())
	}

	count1, ok := result1["count"].(float64)
	if !ok {
		t.Fatalf("first call: count not found or not float64, result: %+v, raw output: %s", result1, bufOut1.String())
	}
	if count1 != 2 {
		t.Errorf("first call with --tail 2: count = %v, want 2, raw output: %s", count1, bufOut1.String())
	}

	// Second call without flags - should show all 5
	rOut2, wOut2, _ := os.Pipe()
	rErr2, wErr2, _ := os.Pipe()
	os.Stdout = wOut2
	os.Stderr = wErr2

	recognized, err = ExecuteArgs([]string{"console", "show", "--json"})
	if !recognized {
		t.Fatal("second command not recognized")
	}
	if err != nil {
		t.Fatalf("second ExecuteArgs returned error: %v", err)
	}

	wOut2.Close()
	wErr2.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut2, bufErr2 bytes.Buffer
	bufOut2.ReadFrom(rOut2)
	bufErr2.ReadFrom(rErr2)

	if bufErr2.Len() > 0 {
		t.Logf("Second call stderr: %s", bufErr2.String())
	}

	var result2 map[string]any
	if err := json.Unmarshal(bufOut2.Bytes(), &result2); err != nil {
		t.Fatalf("failed to parse second call output: %v, output: %s", err, bufOut2.String())
	}

	count2, ok := result2["count"].(float64)
	if !ok {
		t.Fatalf("second call: count not found or not float64, result: %+v", result2)
	}
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
	enableJSONOutput(t)
	restore := setMockFactory(&mockFactory{
		daemonRunning: false,
	})
	defer restore()

	// Capture stderr for error output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runNetworkDefault(networkCmd, []string{})

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
	enableJSONOutput(t)
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

	err := runNetworkDefault(networkCmd, []string{})

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
		t.Errorf("expected ok=true, got %v", result["ok"])
	}

	path, ok := result["path"].(string)
	if !ok {
		t.Fatalf("expected path to be string, got %T", result["path"])
	}

	// Verify path is in temp directory
	if !strings.HasPrefix(path, "/tmp/webctl-network/") {
		t.Errorf("expected path to start with /tmp/webctl-network/, got %s", path)
	}

	// Verify filename format: YY-MM-DD-HHMMSS-network.json
	if !strings.HasSuffix(path, "-network.json") {
		t.Errorf("expected path to end with -network.json, got %s", path)
	}
}

// Target command tests

func TestRunTarget_DaemonNotRunning(t *testing.T) {
	enableJSONOutput(t)
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
	enableJSONOutput(t)
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
	enableJSONOutput(t)
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
	enableJSONOutput(t)
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
	enableJSONOutput(t)
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
	enableJSONOutput(t)
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

	// Set custom output path in temp dir via flag
	screenshotCmd.Flags().Set("output", tmpDir+"/test-screenshot.png")
	defer screenshotCmd.Flags().Set("output", "")

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
	enableJSONOutput(t)
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

	screenshotCmd.Flags().Set("output", customPath)
	defer screenshotCmd.Flags().Set("output", "")

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
	enableJSONOutput(t)
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
	screenshotCmd.Flags().Set("output", tmpDir+"/test.png")
	defer screenshotCmd.Flags().Set("output", "")

	screenshotCmd.Flags().Set("full-page", "true")
	defer screenshotCmd.Flags().Set("full-page", "false")

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
	enableJSONOutput(t)
	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runHTMLDefault(htmlCmd, []string{})

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
	enableJSONOutput(t)
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

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runHTMLSave(htmlSaveCmd, []string{tmpDir + "/test.html"})

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

	// HTML should be formatted by default (not raw)
	expectedFormatted := `<!DOCTYPE html>
<html>
  <head>
    <title>
      Test
    </title>
  </head>
  <body>
    <h1>
      Hello
    </h1>
  </body>
</html>
`
	if string(data) != expectedFormatted {
		t.Errorf("HTML content mismatch: got %q, want %q", string(data), expectedFormatted)
	}
}

func TestRunHTML_WithSelector(t *testing.T) {
	enableJSONOutput(t)
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

	// Set persistent flag on parent command
	htmlCmd.PersistentFlags().Set("select", ".content")
	defer htmlCmd.PersistentFlags().Set("select", "")

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runHTMLSave(htmlSaveCmd, []string{tmpDir + "/test.html"})

	w.Close()
	os.Stdout = old

	if capturedSelector != ".content" {
		t.Errorf("expected selector='.content', got %q", capturedSelector)
	}
}

func TestRunHTML_CustomOutput(t *testing.T) {
	enableJSONOutput(t)
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

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runHTMLSave(htmlSaveCmd, []string{customPath})

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

func TestRunEval_DaemonNotRunning(t *testing.T) {
	enableJSONOutput(t)
	restore := setMockFactory(&mockFactory{
		daemonRunning: false,
	})
	defer restore()

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runEval(evalCmd, []string{"1+1"})

	w.Close()
	os.Stderr = old

	if err == nil {
		t.Fatal("expected error when daemon not running")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !bytes.Contains([]byte(output), []byte("daemon not running")) {
		t.Errorf("expected 'daemon not running' error, got: %s", output)
	}
}

func TestRunEval_BasicExpression(t *testing.T) {
	enableJSONOutput(t)
	evalData := ipc.EvalData{Value: float64(2), HasValue: true}
	evalJSON, _ := json.Marshal(evalData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd == "eval" {
				return ipc.Response{OK: true, Data: evalJSON}, nil
			}
			return ipc.Response{OK: false}, nil
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

	err := runEval(evalCmd, []string{"1+1"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	json.Unmarshal(buf.Bytes(), &result)

	if result["ok"] != true {
		t.Errorf("expected ok=true, got %v", result["ok"])
	}

	if result["value"] != float64(2) {
		t.Errorf("expected value=2, got %v", result["value"])
	}
}

func TestRunEval_Undefined(t *testing.T) {
	enableJSONOutput(t)
	evalData := ipc.EvalData{HasValue: false}
	evalJSON, _ := json.Marshal(evalData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd == "eval" {
				return ipc.Response{OK: true, Data: evalJSON}, nil
			}
			return ipc.Response{OK: false}, nil
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

	err := runEval(evalCmd, []string{"undefined"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	json.Unmarshal(buf.Bytes(), &result)

	if result["ok"] != true {
		t.Errorf("expected ok=true, got %v", result["ok"])
	}

	// Should not have value field for undefined
	if _, exists := result["value"]; exists {
		t.Error("expected no 'value' field for undefined result")
	}
}

func TestRunEval_MultipleArgs(t *testing.T) {
	enableJSONOutput(t)
	var capturedExpression string

	evalData := ipc.EvalData{Value: "Hello World", HasValue: true}
	evalJSON, _ := json.Marshal(evalData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd == "eval" {
				var params ipc.EvalParams
				json.Unmarshal(req.Params, &params)
				capturedExpression = params.Expression
				return ipc.Response{OK: true, Data: evalJSON}, nil
			}
			return ipc.Response{OK: false}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runEval(evalCmd, []string{"'Hello'", "+", "'World'"})

	w.Close()
	os.Stdout = old

	expected := "'Hello' + 'World'"
	if capturedExpression != expected {
		t.Errorf("expected expression=%q, got %q", expected, capturedExpression)
	}
}

func TestRunEval_Error(t *testing.T) {
	enableJSONOutput(t)
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd == "eval" {
				return ipc.Response{OK: false, Error: "ReferenceError: foo is not defined"}, nil
			}
			return ipc.Response{OK: false}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runEval(evalCmd, []string{"foo"})

	w.Close()
	os.Stderr = old

	if err == nil {
		t.Fatal("expected error for undefined variable")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !bytes.Contains([]byte(output), []byte("ReferenceError")) {
		t.Errorf("expected ReferenceError in output, got: %s", output)
	}
}

func TestRunEval_Timeout(t *testing.T) {
	enableJSONOutput(t)
	var capturedTimeout int

	evalData := ipc.EvalData{Value: float64(42), HasValue: true}
	evalJSON, _ := json.Marshal(evalData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd == "eval" {
				var params ipc.EvalParams
				json.Unmarshal(req.Params, &params)
				capturedTimeout = params.Timeout
				return ipc.Response{OK: true, Data: evalJSON}, nil
			}
			return ipc.Response{OK: false}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	evalCmd.Flags().Set("timeout", "5s")
	defer evalCmd.Flags().Set("timeout", "30s")

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runEval(evalCmd, []string{"42"})

	w.Close()
	os.Stdout = old

	if capturedTimeout != 5000 {
		t.Errorf("expected timeout=5000ms, got %d", capturedTimeout)
	}
}

func TestRunCookiesList_DaemonNotRunning(t *testing.T) {
	enableJSONOutput(t)
	restore := setMockFactory(&mockFactory{
		daemonRunning: false,
	})
	defer restore()

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runCookiesShow(cookiesShowCmd, []string{})

	w.Close()
	os.Stderr = old

	if err == nil {
		t.Fatal("expected error when daemon not running")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !bytes.Contains([]byte(output), []byte("daemon not running")) {
		t.Errorf("expected 'daemon not running' error, got: %s", output)
	}
}

func TestRunCookiesList_Success(t *testing.T) {
	enableJSONOutput(t)
	cookies := []ipc.Cookie{
		{Name: "session", Value: "abc123", Domain: "example.com", Path: "/"},
		{Name: "user", Value: "john", Domain: "example.com", Path: "/"},
	}
	cookiesData := ipc.CookiesData{Cookies: cookies, Count: 2}
	cookiesJSON, _ := json.Marshal(cookiesData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd == "cookies" {
				return ipc.Response{OK: true, Data: cookiesJSON}, nil
			}
			return ipc.Response{OK: false}, nil
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

	err := runCookiesShow(cookiesShowCmd, []string{})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	json.Unmarshal(buf.Bytes(), &result)

	if result["ok"] != true {
		t.Errorf("expected ok=true, got %v", result["ok"])
	}

	if result["count"] != float64(2) {
		t.Errorf("expected count=2, got %v", result["count"])
	}
}

func TestRunCookiesSet_Basic(t *testing.T) {
	enableJSONOutput(t)
	var capturedParams ipc.CookiesParams

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd == "cookies" {
				json.Unmarshal(req.Params, &capturedParams)
				return ipc.Response{OK: true}, nil
			}
			return ipc.Response{OK: false}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runCookiesSet(cookiesSetCmd, []string{"session", "xyz789"})

	w.Close()
	os.Stdout = old

	if capturedParams.Action != "set" {
		t.Errorf("expected action=set, got %s", capturedParams.Action)
	}

	if capturedParams.Name != "session" {
		t.Errorf("expected name=session, got %s", capturedParams.Name)
	}

	if capturedParams.Value != "xyz789" {
		t.Errorf("expected value=xyz789, got %s", capturedParams.Value)
	}
}

func TestRunCookiesSet_WithFlags(t *testing.T) {
	enableJSONOutput(t)
	var capturedParams ipc.CookiesParams

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd == "cookies" {
				json.Unmarshal(req.Params, &capturedParams)
				return ipc.Response{OK: true}, nil
			}
			return ipc.Response{OK: false}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	cookiesSetCmd.Flags().Set("domain", "example.com")
	cookiesSetCmd.Flags().Set("secure", "true")
	cookiesSetCmd.Flags().Set("httponly", "true")
	cookiesSetCmd.Flags().Set("max-age", "3600")
	cookiesSetCmd.Flags().Set("samesite", "Strict")
	defer func() {
		cookiesSetCmd.Flags().Set("domain", "")
		cookiesSetCmd.Flags().Set("secure", "false")
		cookiesSetCmd.Flags().Set("httponly", "false")
		cookiesSetCmd.Flags().Set("max-age", "0")
		cookiesSetCmd.Flags().Set("samesite", "")
	}()

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runCookiesSet(cookiesSetCmd, []string{"auth", "token123"})

	w.Close()
	os.Stdout = old

	if capturedParams.Domain != "example.com" {
		t.Errorf("expected domain=example.com, got %s", capturedParams.Domain)
	}

	if !capturedParams.Secure {
		t.Error("expected secure=true")
	}

	if !capturedParams.HTTPOnly {
		t.Error("expected httpOnly=true")
	}

	if capturedParams.MaxAge != 3600 {
		t.Errorf("expected maxAge=3600, got %d", capturedParams.MaxAge)
	}

	if capturedParams.SameSite != "Strict" {
		t.Errorf("expected sameSite=Strict, got %s", capturedParams.SameSite)
	}
}

func TestRunCookiesDelete_Success(t *testing.T) {
	enableJSONOutput(t)
	var capturedParams ipc.CookiesParams

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd == "cookies" {
				json.Unmarshal(req.Params, &capturedParams)
				return ipc.Response{OK: true}, nil
			}
			return ipc.Response{OK: false}, nil
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

	err := runCookiesDelete(cookiesDeleteCmd, []string{"session"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	json.Unmarshal(buf.Bytes(), &result)

	if result["ok"] != true {
		t.Errorf("expected ok=true, got %v", result["ok"])
	}

	if capturedParams.Action != "delete" {
		t.Errorf("expected action=delete, got %s", capturedParams.Action)
	}

	if capturedParams.Name != "session" {
		t.Errorf("expected name=session, got %s", capturedParams.Name)
	}
}

func TestRunCookiesDelete_AmbiguousError(t *testing.T) {
	enableJSONOutput(t)
	matches := []ipc.Cookie{
		{Name: "session", Domain: "example.com"},
		{Name: "session", Domain: "api.example.com"},
	}
	matchData := ipc.CookiesData{Matches: matches}
	matchJSON, _ := json.Marshal(matchData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd == "cookies" {
				return ipc.Response{
					OK:    false,
					Error: "multiple cookies named 'session' found",
					Data:  matchJSON,
				}, nil
			}
			return ipc.Response{OK: false}, nil
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

	oldErr := os.Stderr
	_, we, _ := os.Pipe()
	os.Stderr = we

	err := runCookiesDelete(cookiesDeleteCmd, []string{"session"})

	w.Close()
	os.Stdout = old

	we.Close()
	os.Stderr = oldErr

	if err == nil {
		t.Fatal("expected error for ambiguous delete")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	json.Unmarshal(buf.Bytes(), &result)

	if result["ok"] != false {
		t.Errorf("expected ok=false, got %v", result["ok"])
	}

	matchesResult, ok := result["matches"].([]any)
	if !ok {
		t.Fatal("expected matches array in result")
	}

	if len(matchesResult) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matchesResult))
	}
}

func TestRunCookiesDelete_WithDomain(t *testing.T) {
	enableJSONOutput(t)
	var capturedParams ipc.CookiesParams

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd == "cookies" {
				json.Unmarshal(req.Params, &capturedParams)
				return ipc.Response{OK: true}, nil
			}
			return ipc.Response{OK: false}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	cookiesDeleteCmd.Flags().Set("domain", "api.example.com")
	defer cookiesDeleteCmd.Flags().Set("domain", "")

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runCookiesDelete(cookiesDeleteCmd, []string{"session"})

	w.Close()
	os.Stdout = old

	if capturedParams.Domain != "api.example.com" {
		t.Errorf("expected domain=api.example.com, got %s", capturedParams.Domain)
	}
}

// Navigation command tests

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Already has protocol
		{"https already", "https://example.com", "https://example.com"},
		{"http already", "http://example.com", "http://example.com"},
		{"ftp already", "ftp://files.example.com", "ftp://files.example.com"},

		// Localhost variants - should use http
		{"localhost", "localhost", "http://localhost"},
		{"localhost with port", "localhost:3000", "http://localhost:3000"},
		{"localhost with path", "localhost:8080/api/v1", "http://localhost:8080/api/v1"},
		{"LOCALHOST uppercase", "LOCALHOST:3000", "http://LOCALHOST:3000"},

		// Local IPs - should use http
		{"127.0.0.1", "127.0.0.1", "http://127.0.0.1"},
		{"127.0.0.1 with port", "127.0.0.1:8080", "http://127.0.0.1:8080"},
		{"0.0.0.0", "0.0.0.0:3000", "http://0.0.0.0:3000"},

		// External domains - should use https
		{"simple domain", "example.com", "https://example.com"},
		{"domain with path", "example.com/path/to/page", "https://example.com/path/to/page"},
		{"domain with port", "example.com:8443", "https://example.com:8443"},
		{"subdomain", "api.example.com", "https://api.example.com"},
		{"complex url", "api.example.com:8080/v1/users?id=123", "https://api.example.com:8080/v1/users?id=123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeURL(tt.input)
			if got != tt.want {
				t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRunNavigate_DaemonNotRunning(t *testing.T) {
	enableJSONOutput(t)
	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runNavigate(navigateCmd, []string{"example.com"})

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

	if resp["error"] != "daemon not running. Start with: webctl start" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
}

func TestRunNavigate_Success(t *testing.T) {
	enableJSONOutput(t)
	navData := ipc.NavigateData{URL: "https://example.com", Title: "Example Domain"}
	navJSON, _ := json.Marshal(navData)

	var capturedParams ipc.NavigateParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "navigate" {
				t.Errorf("expected cmd=navigate, got %s", req.Cmd)
			}
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true, Data: navJSON}, nil
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

	err := runNavigate(navigateCmd, []string{"example.com"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify URL was normalized
	if capturedParams.URL != "https://example.com" {
		t.Errorf("expected URL=https://example.com, got %s", capturedParams.URL)
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
	if result["url"] != "https://example.com" {
		t.Errorf("expected url=https://example.com, got %v", result["url"])
	}
}

func TestRunNavigate_WithWaitFlag(t *testing.T) {
	enableJSONOutput(t)
	navData := ipc.NavigateData{URL: "https://example.com", Title: "Example Domain"}
	navJSON, _ := json.Marshal(navData)

	var capturedParams ipc.NavigateParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true, Data: navJSON}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	navigateCmd.Flags().Set("wait", "true")
	navigateCmd.Flags().Set("timeout", "5000")
	defer func() {
		navigateCmd.Flags().Set("wait", "false")
		navigateCmd.Flags().Set("timeout", "30000")
	}()

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runNavigate(navigateCmd, []string{"example.com"})

	w.Close()
	os.Stdout = old

	if !capturedParams.Wait {
		t.Error("expected Wait=true")
	}
	if capturedParams.Timeout != 5000 {
		t.Errorf("expected Timeout=5000, got %d", capturedParams.Timeout)
	}
}

func TestRunNavigate_LocalhostUsesHTTP(t *testing.T) {
	enableJSONOutput(t)
	navData := ipc.NavigateData{URL: "http://localhost:3000", Title: ""}
	navJSON, _ := json.Marshal(navData)

	var capturedParams ipc.NavigateParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true, Data: navJSON}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runNavigate(navigateCmd, []string{"localhost:3000"})

	w.Close()
	os.Stdout = old

	if capturedParams.URL != "http://localhost:3000" {
		t.Errorf("expected URL=http://localhost:3000, got %s", capturedParams.URL)
	}
}

func TestRunNavigate_Error(t *testing.T) {
	enableJSONOutput(t)
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: false, Error: "net::ERR_NAME_NOT_RESOLVED"}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runNavigate(navigateCmd, []string{"invalid.invalid"})

	w.Close()
	os.Stderr = oldStderr

	if err == nil {
		t.Error("expected error for failed navigation")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var resp map[string]any
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if resp["ok"] != false {
		t.Error("expected ok=false")
	}
	if resp["error"] != "net::ERR_NAME_NOT_RESOLVED" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
}

func TestRunReload_DaemonNotRunning(t *testing.T) {
	enableJSONOutput(t)
	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runReload(reloadCmd, []string{})

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

func TestRunReload_Success(t *testing.T) {
	enableJSONOutput(t)
	navData := ipc.NavigateData{URL: "https://example.com", Title: "Example Domain"}
	navJSON, _ := json.Marshal(navData)

	var capturedParams ipc.ReloadParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "reload" {
				t.Errorf("expected cmd=reload, got %s", req.Cmd)
			}
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true, Data: navJSON}, nil
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

	err := runReload(reloadCmd, []string{})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify hard reload (ignoreCache=true always)
	if !capturedParams.IgnoreCache {
		t.Error("expected IgnoreCache=true (hard reload)")
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
}

func TestRunReload_WithWaitFlag(t *testing.T) {
	enableJSONOutput(t)
	navData := ipc.NavigateData{URL: "https://example.com", Title: "Example Domain"}
	navJSON, _ := json.Marshal(navData)

	var capturedParams ipc.ReloadParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true, Data: navJSON}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	reloadCmd.Flags().Set("wait", "true")
	reloadCmd.Flags().Set("timeout", "10000")
	defer func() {
		reloadCmd.Flags().Set("wait", "false")
		reloadCmd.Flags().Set("timeout", "30000")
	}()

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runReload(reloadCmd, []string{})

	w.Close()
	os.Stdout = old

	if !capturedParams.Wait {
		t.Error("expected Wait=true")
	}
	if capturedParams.Timeout != 10000 {
		t.Errorf("expected Timeout=10000, got %d", capturedParams.Timeout)
	}
}

func TestRunBack_DaemonNotRunning(t *testing.T) {
	enableJSONOutput(t)
	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runBack(backCmd, []string{})

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

func TestRunBack_Success(t *testing.T) {
	enableJSONOutput(t)
	navData := ipc.NavigateData{URL: "https://previous.com", Title: "Previous Page"}
	navJSON, _ := json.Marshal(navData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "back" {
				t.Errorf("expected cmd=back, got %s", req.Cmd)
			}
			return ipc.Response{OK: true, Data: navJSON}, nil
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

	err := runBack(backCmd, []string{})

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
	if result["url"] != "https://previous.com" {
		t.Errorf("expected url=https://previous.com, got %v", result["url"])
	}
}

func TestRunBack_NoHistory(t *testing.T) {
	enableJSONOutput(t)
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: false, Error: "no previous page in history"}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runBack(backCmd, []string{})

	w.Close()
	os.Stderr = oldStderr

	if err == nil {
		t.Error("expected error when no history")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var resp map[string]any
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if resp["ok"] != false {
		t.Error("expected ok=false")
	}
	if resp["error"] != "no previous page in history" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
}

func TestRunBack_WithWaitFlag(t *testing.T) {
	enableJSONOutput(t)
	navData := ipc.NavigateData{URL: "https://previous.com", Title: "Previous"}
	navJSON, _ := json.Marshal(navData)

	var capturedParams ipc.HistoryParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true, Data: navJSON}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	backCmd.Flags().Set("wait", "true")
	backCmd.Flags().Set("timeout", "15000")
	defer func() {
		backCmd.Flags().Set("wait", "false")
		backCmd.Flags().Set("timeout", "30000")
	}()

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runBack(backCmd, []string{})

	w.Close()
	os.Stdout = old

	if !capturedParams.Wait {
		t.Error("expected Wait=true")
	}
	if capturedParams.Timeout != 15000 {
		t.Errorf("expected Timeout=15000, got %d", capturedParams.Timeout)
	}
}

func TestRunForward_DaemonNotRunning(t *testing.T) {
	enableJSONOutput(t)
	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runForward(forwardCmd, []string{})

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

func TestRunForward_Success(t *testing.T) {
	enableJSONOutput(t)
	navData := ipc.NavigateData{URL: "https://next.com", Title: "Next Page"}
	navJSON, _ := json.Marshal(navData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "forward" {
				t.Errorf("expected cmd=forward, got %s", req.Cmd)
			}
			return ipc.Response{OK: true, Data: navJSON}, nil
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

	err := runForward(forwardCmd, []string{})

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
	if result["url"] != "https://next.com" {
		t.Errorf("expected url=https://next.com, got %v", result["url"])
	}
}

func TestRunForward_NoHistory(t *testing.T) {
	enableJSONOutput(t)
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: false, Error: "no next page in history"}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runForward(forwardCmd, []string{})

	w.Close()
	os.Stderr = oldStderr

	if err == nil {
		t.Error("expected error when no forward history")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var resp map[string]any
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if resp["ok"] != false {
		t.Error("expected ok=false")
	}
	if resp["error"] != "no next page in history" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
}

func TestRunForward_WithWaitFlag(t *testing.T) {
	enableJSONOutput(t)
	navData := ipc.NavigateData{URL: "https://next.com", Title: "Next"}
	navJSON, _ := json.Marshal(navData)

	var capturedParams ipc.HistoryParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true, Data: navJSON}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	forwardCmd.Flags().Set("wait", "true")
	forwardCmd.Flags().Set("timeout", "20000")
	defer func() {
		forwardCmd.Flags().Set("wait", "false")
		forwardCmd.Flags().Set("timeout", "30000")
	}()

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runForward(forwardCmd, []string{})

	w.Close()
	os.Stdout = old

	if !capturedParams.Wait {
		t.Error("expected Wait=true")
	}
	if capturedParams.Timeout != 20000 {
		t.Errorf("expected Timeout=20000, got %d", capturedParams.Timeout)
	}
}

// Click command tests

func TestRunClick_DaemonNotRunning(t *testing.T) {
	enableJSONOutput(t)
	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runClick(clickCmd, []string{"#button"})

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

func TestRunClick_Success(t *testing.T) {
	enableJSONOutput(t)
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "click" {
				t.Errorf("expected cmd=click, got %s", req.Cmd)
			}
			var params ipc.ClickParams
			json.Unmarshal(req.Params, &params)
			if params.Selector != "#submit-btn" {
				t.Errorf("expected selector=#submit-btn, got %s", params.Selector)
			}
			return ipc.Response{OK: true}, nil
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

	err := runClick(clickCmd, []string{"#submit-btn"})

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
}

func TestRunClick_WithWarning(t *testing.T) {
	enableJSONOutput(t)
	warningData, _ := json.Marshal(map[string]any{
		"warning": "element may be covered by another element: #hidden-btn",
	})

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: true, Data: warningData}, nil
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

	err := runClick(clickCmd, []string{"#hidden-btn"})

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
	if result["warning"] == nil {
		t.Error("expected warning in response")
	}
}

func TestRunClick_ElementNotFound(t *testing.T) {
	enableJSONOutput(t)
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: false, Error: "element not found: #missing"}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runClick(clickCmd, []string{"#missing"})

	w.Close()
	os.Stderr = oldStderr

	if err == nil {
		t.Error("expected error for missing element")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var resp map[string]any
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if resp["ok"] != false {
		t.Error("expected ok=false")
	}
	if resp["error"] != "element not found: #missing" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
}

// Focus command tests

func TestRunFocus_DaemonNotRunning(t *testing.T) {
	enableJSONOutput(t)
	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runFocus(focusCmd, []string{"#input"})

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

func TestRunFocus_Success(t *testing.T) {
	enableJSONOutput(t)
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "focus" {
				t.Errorf("expected cmd=focus, got %s", req.Cmd)
			}
			var params ipc.FocusParams
			json.Unmarshal(req.Params, &params)
			if params.Selector != "#email-input" {
				t.Errorf("expected selector=#email-input, got %s", params.Selector)
			}
			return ipc.Response{OK: true}, nil
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

	err := runFocus(focusCmd, []string{"#email-input"})

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
}

func TestRunFocus_ElementNotFound(t *testing.T) {
	enableJSONOutput(t)
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: false, Error: "element not found: #missing"}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runFocus(focusCmd, []string{"#missing"})

	w.Close()
	os.Stderr = oldStderr

	if err == nil {
		t.Error("expected error for missing element")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var resp map[string]any
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if resp["ok"] != false {
		t.Error("expected ok=false")
	}
}

// Type command tests

func TestRunType_DaemonNotRunning(t *testing.T) {
	enableJSONOutput(t)
	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runType(typeCmd, []string{"hello"})

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

func TestRunType_TextOnly(t *testing.T) {
	enableJSONOutput(t)
	var capturedParams ipc.TypeParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "type" {
				t.Errorf("expected cmd=type, got %s", req.Cmd)
			}
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true}, nil
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

	err := runType(typeCmd, []string{"hello world"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With one arg, selector should be empty, text should be the arg
	if capturedParams.Selector != "" {
		t.Errorf("expected empty selector, got %s", capturedParams.Selector)
	}
	if capturedParams.Text != "hello world" {
		t.Errorf("expected text='hello world', got %s", capturedParams.Text)
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
}

func TestRunType_WithSelector(t *testing.T) {
	enableJSONOutput(t)
	var capturedParams ipc.TypeParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runType(typeCmd, []string{"#input", "test text"})

	w.Close()
	os.Stdout = old

	if capturedParams.Selector != "#input" {
		t.Errorf("expected selector=#input, got %s", capturedParams.Selector)
	}
	if capturedParams.Text != "test text" {
		t.Errorf("expected text='test text', got %s", capturedParams.Text)
	}
}

func TestRunType_WithKeyFlag(t *testing.T) {
	enableJSONOutput(t)
	var capturedParams ipc.TypeParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	typeCmd.Flags().Set("key", "Enter")
	defer typeCmd.Flags().Set("key", "")

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runType(typeCmd, []string{"#search", "query"})

	w.Close()
	os.Stdout = old

	if capturedParams.Key != "Enter" {
		t.Errorf("expected key=Enter, got %s", capturedParams.Key)
	}
}

func TestRunType_WithClearFlag(t *testing.T) {
	enableJSONOutput(t)
	var capturedParams ipc.TypeParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	typeCmd.Flags().Set("clear", "true")
	defer typeCmd.Flags().Set("clear", "false")

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runType(typeCmd, []string{"#input", "new value"})

	w.Close()
	os.Stdout = old

	if !capturedParams.Clear {
		t.Error("expected Clear=true")
	}
}

func TestRunType_AllFlags(t *testing.T) {
	enableJSONOutput(t)
	var capturedParams ipc.TypeParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	typeCmd.Flags().Set("key", "Tab")
	typeCmd.Flags().Set("clear", "true")
	defer func() {
		typeCmd.Flags().Set("key", "")
		typeCmd.Flags().Set("clear", "false")
	}()

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runType(typeCmd, []string{"#form-field", "updated"})

	w.Close()
	os.Stdout = old

	if capturedParams.Selector != "#form-field" {
		t.Errorf("expected selector=#form-field, got %s", capturedParams.Selector)
	}
	if capturedParams.Text != "updated" {
		t.Errorf("expected text='updated', got %s", capturedParams.Text)
	}
	if capturedParams.Key != "Tab" {
		t.Errorf("expected key=Tab, got %s", capturedParams.Key)
	}
	if !capturedParams.Clear {
		t.Error("expected Clear=true")
	}
}

// Key command tests

func TestRunKey_DaemonNotRunning(t *testing.T) {
	enableJSONOutput(t)
	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runKey(keyCmd, []string{"Enter"})

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

func TestRunKey_Success(t *testing.T) {
	enableJSONOutput(t)
	var capturedParams ipc.KeyParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "key" {
				t.Errorf("expected cmd=key, got %s", req.Cmd)
			}
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true}, nil
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

	err := runKey(keyCmd, []string{"Enter"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedParams.Key != "Enter" {
		t.Errorf("expected key=Enter, got %s", capturedParams.Key)
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
}

func TestRunKey_WithCtrlModifier(t *testing.T) {
	enableJSONOutput(t)
	var capturedParams ipc.KeyParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	keyCmd.Flags().Set("ctrl", "true")
	defer keyCmd.Flags().Set("ctrl", "false")

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runKey(keyCmd, []string{"a"})

	w.Close()
	os.Stdout = old

	if capturedParams.Key != "a" {
		t.Errorf("expected key=a, got %s", capturedParams.Key)
	}
	if !capturedParams.Ctrl {
		t.Error("expected Ctrl=true")
	}
}

func TestRunKey_WithMetaModifier(t *testing.T) {
	enableJSONOutput(t)
	var capturedParams ipc.KeyParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	keyCmd.Flags().Set("meta", "true")
	defer keyCmd.Flags().Set("meta", "false")

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runKey(keyCmd, []string{"c"})

	w.Close()
	os.Stdout = old

	if capturedParams.Key != "c" {
		t.Errorf("expected key=c, got %s", capturedParams.Key)
	}
	if !capturedParams.Meta {
		t.Error("expected Meta=true")
	}
}

func TestRunKey_AllModifiers(t *testing.T) {
	enableJSONOutput(t)
	var capturedParams ipc.KeyParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	keyCmd.Flags().Set("ctrl", "true")
	keyCmd.Flags().Set("alt", "true")
	keyCmd.Flags().Set("shift", "true")
	keyCmd.Flags().Set("meta", "true")
	defer func() {
		keyCmd.Flags().Set("ctrl", "false")
		keyCmd.Flags().Set("alt", "false")
		keyCmd.Flags().Set("shift", "false")
		keyCmd.Flags().Set("meta", "false")
	}()

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runKey(keyCmd, []string{"Delete"})

	w.Close()
	os.Stdout = old

	if capturedParams.Key != "Delete" {
		t.Errorf("expected key=Delete, got %s", capturedParams.Key)
	}
	if !capturedParams.Ctrl {
		t.Error("expected Ctrl=true")
	}
	if !capturedParams.Alt {
		t.Error("expected Alt=true")
	}
	if !capturedParams.Shift {
		t.Error("expected Shift=true")
	}
	if !capturedParams.Meta {
		t.Error("expected Meta=true")
	}
}

// Select command tests

func TestRunSelect_DaemonNotRunning(t *testing.T) {
	enableJSONOutput(t)
	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runSelect(selectCmd_, []string{"#dropdown", "value1"})

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

func TestRunSelect_Success(t *testing.T) {
	enableJSONOutput(t)
	var capturedParams ipc.SelectParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "select" {
				t.Errorf("expected cmd=select, got %s", req.Cmd)
			}
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true}, nil
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

	err := runSelect(selectCmd_, []string{"#country", "AU"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedParams.Selector != "#country" {
		t.Errorf("expected selector=#country, got %s", capturedParams.Selector)
	}
	if capturedParams.Value != "AU" {
		t.Errorf("expected value=AU, got %s", capturedParams.Value)
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
}

func TestRunSelect_ElementNotFound(t *testing.T) {
	enableJSONOutput(t)
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: false, Error: "element not found: #missing"}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runSelect(selectCmd_, []string{"#missing", "value"})

	w.Close()
	os.Stderr = oldStderr

	if err == nil {
		t.Error("expected error for missing element")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var resp map[string]any
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if resp["ok"] != false {
		t.Error("expected ok=false")
	}
	if resp["error"] != "element not found: #missing" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
}

func TestRunSelect_NotASelectElement(t *testing.T) {
	enableJSONOutput(t)
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: false, Error: "element is not a select: #div"}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runSelect(selectCmd_, []string{"#div", "value"})

	w.Close()
	os.Stderr = oldStderr

	if err == nil {
		t.Error("expected error for non-select element")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var resp map[string]any
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if resp["ok"] != false {
		t.Error("expected ok=false")
	}
	if resp["error"] != "element is not a select: #div" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
}

// Scroll command tests

func TestParseCoords(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantX   int
		wantY   int
		wantErr bool
	}{
		{"basic", "100,200", 100, 200, false},
		{"zeros", "0,0", 0, 0, false},
		{"negative", "-100,-200", -100, -200, false},
		{"mixed", "50,-100", 50, -100, false},
		{"with spaces", " 100 , 200 ", 100, 200, false},
		{"large numbers", "10000,20000", 10000, 20000, false},
		{"missing comma", "100200", 0, 0, true},
		{"too many parts", "100,200,300", 0, 0, true},
		{"invalid x", "abc,200", 0, 0, true},
		{"invalid y", "100,xyz", 0, 0, true},
		{"empty", "", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y, err := parseCoords(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseCoords(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseCoords(%q) unexpected error: %v", tt.input, err)
				return
			}
			if x != tt.wantX || y != tt.wantY {
				t.Errorf("parseCoords(%q) = (%d, %d), want (%d, %d)", tt.input, x, y, tt.wantX, tt.wantY)
			}
		})
	}
}

func TestRunScroll_DaemonNotRunning(t *testing.T) {
	enableJSONOutput(t)
	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runScroll(scrollCmd, []string{"#element"})

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

func TestRunScroll_ElementMode(t *testing.T) {
	enableJSONOutput(t)
	var capturedParams ipc.ScrollParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "scroll" {
				t.Errorf("expected cmd=scroll, got %s", req.Cmd)
			}
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true}, nil
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

	err := runScroll(scrollCmd, []string{"#footer"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedParams.Mode != "element" {
		t.Errorf("expected mode=element, got %s", capturedParams.Mode)
	}
	if capturedParams.Selector != "#footer" {
		t.Errorf("expected selector=#footer, got %s", capturedParams.Selector)
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
}

func TestRunScroll_ToMode(t *testing.T) {
	enableJSONOutput(t)
	var capturedParams ipc.ScrollParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	scrollCmd.Flags().Set("to", "100,500")
	defer scrollCmd.Flags().Set("to", "")

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runScroll(scrollCmd, []string{})

	w.Close()
	os.Stdout = old

	if capturedParams.Mode != "to" {
		t.Errorf("expected mode=to, got %s", capturedParams.Mode)
	}
	if capturedParams.ToX != 100 {
		t.Errorf("expected ToX=100, got %d", capturedParams.ToX)
	}
	if capturedParams.ToY != 500 {
		t.Errorf("expected ToY=500, got %d", capturedParams.ToY)
	}
}

func TestRunScroll_ByMode(t *testing.T) {
	enableJSONOutput(t)
	var capturedParams ipc.ScrollParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	scrollCmd.Flags().Set("by", "0,-100")
	defer scrollCmd.Flags().Set("by", "")

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runScroll(scrollCmd, []string{})

	w.Close()
	os.Stdout = old

	if capturedParams.Mode != "by" {
		t.Errorf("expected mode=by, got %s", capturedParams.Mode)
	}
	if capturedParams.ByX != 0 {
		t.Errorf("expected ByX=0, got %d", capturedParams.ByX)
	}
	if capturedParams.ByY != -100 {
		t.Errorf("expected ByY=-100, got %d", capturedParams.ByY)
	}
}

func TestRunScroll_InvalidToCoords(t *testing.T) {
	enableJSONOutput(t)
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	scrollCmd.Flags().Set("to", "invalid")
	defer scrollCmd.Flags().Set("to", "")

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runScroll(scrollCmd, []string{})

	w.Close()
	os.Stderr = oldStderr

	if err == nil {
		t.Error("expected error for invalid coordinates")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var resp map[string]any
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if resp["ok"] != false {
		t.Error("expected ok=false")
	}
}

func TestRunScroll_NoModeSpecified(t *testing.T) {
	enableJSONOutput(t)
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runScroll(scrollCmd, []string{})

	w.Close()
	os.Stderr = oldStderr

	if err == nil {
		t.Error("expected error when no mode specified")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var resp map[string]any
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if resp["ok"] != false {
		t.Error("expected ok=false")
	}
	if resp["error"] != "provide a selector, --to x,y, or --by x,y" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
}

func TestRunScroll_ElementNotFound(t *testing.T) {
	enableJSONOutput(t)
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: false, Error: "element not found: #missing"}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runScroll(scrollCmd, []string{"#missing"})

	w.Close()
	os.Stderr = oldStderr

	if err == nil {
		t.Error("expected error for missing element")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var resp map[string]any
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if resp["ok"] != false {
		t.Error("expected ok=false")
	}
	if resp["error"] != "element not found: #missing" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
}

// Ready command tests

func TestRunReady_DaemonNotRunning(t *testing.T) {
	enableJSONOutput(t)
	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runReady(readyCmd, []string{})

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

func TestRunReady_Success(t *testing.T) {
	enableJSONOutput(t)
	var capturedParams ipc.ReadyParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "ready" {
				t.Errorf("expected cmd=ready, got %s", req.Cmd)
			}
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true}, nil
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

	err := runReady(readyCmd, []string{})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Default timeout should be 60 seconds = 60000ms
	if capturedParams.Timeout != 60000 {
		t.Errorf("expected Timeout=60000, got %d", capturedParams.Timeout)
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
}

func TestRunReady_WithCustomTimeout(t *testing.T) {
	enableJSONOutput(t)
	var capturedParams ipc.ReadyParams
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			json.Unmarshal(req.Params, &capturedParams)
			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	readyCmd.Flags().Set("timeout", "10s")
	defer readyCmd.Flags().Set("timeout", "30s")

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	runReady(readyCmd, []string{})

	w.Close()
	os.Stdout = old

	// 10 seconds = 10000ms
	if capturedParams.Timeout != 10000 {
		t.Errorf("expected Timeout=10000, got %d", capturedParams.Timeout)
	}
}

func TestRunReady_Timeout(t *testing.T) {
	enableJSONOutput(t)
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: false, Error: "timeout waiting for page load"}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runReady(readyCmd, []string{})

	w.Close()
	os.Stderr = oldStderr

	if err == nil {
		t.Error("expected error on timeout")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var resp map[string]any
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if resp["ok"] != false {
		t.Error("expected ok=false")
	}
	if resp["error"] != "timeout waiting for page load" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
}

func TestRunReady_NoActiveSession(t *testing.T) {
	enableJSONOutput(t)
	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: false, Error: "no active session"}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runReady(readyCmd, []string{})

	w.Close()
	os.Stderr = oldStderr

	if err == nil {
		t.Error("expected error when no active session")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var resp map[string]any
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if resp["ok"] != false {
		t.Error("expected ok=false")
	}
	if resp["error"] != "no active session" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
}
