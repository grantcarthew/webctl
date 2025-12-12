package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/grantcarthew/webctl/internal/ipc"
)

// mockClient implements IPCClient for testing.
type mockClient struct {
	sendFunc    func(req ipc.Request) (ipc.Response, error)
	sendCmdFunc func(cmd string) (ipc.Response, error)
	closed      bool
}

func (m *mockClient) Send(req ipc.Request) (ipc.Response, error) {
	if m.sendFunc != nil {
		return m.sendFunc(req)
	}
	return ipc.Response{OK: true}, nil
}

func (m *mockClient) SendCmd(cmd string) (ipc.Response, error) {
	if m.sendCmdFunc != nil {
		return m.sendCmdFunc(cmd)
	}
	return ipc.Response{OK: true}, nil
}

func (m *mockClient) Close() error {
	m.closed = true
	return nil
}

// mockDialer implements Dialer for testing.
type mockDialer struct {
	client          IPCClient
	dialErr         error
	daemonRunning   bool
}

func (m *mockDialer) Dial() (IPCClient, error) {
	if m.dialErr != nil {
		return nil, m.dialErr
	}
	return m.client, nil
}

func (m *mockDialer) IsDaemonRunning() bool {
	return m.daemonRunning
}

// setMockDialer replaces the package dialer and returns a restore function.
func setMockDialer(d Dialer) func() {
	old := dialer
	dialer = d
	return func() { dialer = old }
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

	restore := setMockDialer(&mockDialer{daemonRunning: false})
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

	client := &mockClient{
		sendCmdFunc: func(cmd string) (ipc.Response, error) {
			if cmd != "status" {
				t.Errorf("expected cmd=status, got %s", cmd)
			}
			return ipc.Response{OK: true, Data: statusJSON}, nil
		},
	}

	restore := setMockDialer(&mockDialer{
		daemonRunning: true,
		client:        client,
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

	if !client.closed {
		t.Error("expected client to be closed")
	}
}

func TestRunStop_Success(t *testing.T) {

	client := &mockClient{
		sendCmdFunc: func(cmd string) (ipc.Response, error) {
			if cmd != "shutdown" {
				t.Errorf("expected cmd=shutdown, got %s", cmd)
			}
			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockDialer(&mockDialer{
		daemonRunning: true,
		client:        client,
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

	if !client.closed {
		t.Error("expected client to be closed")
	}
}

func TestRunStop_DialError(t *testing.T) {

	restore := setMockDialer(&mockDialer{
		dialErr: errors.New("daemon is not running"),
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

	client := &mockClient{
		sendFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "clear" {
				t.Errorf("expected cmd=clear, got %s", req.Cmd)
			}
			if req.Target != "" {
				t.Errorf("expected target='', got %s", req.Target)
			}
			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockDialer(&mockDialer{
		daemonRunning: true,
		client:        client,
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

	client := &mockClient{
		sendFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Target != "console" {
				t.Errorf("expected target=console, got %s", req.Target)
			}
			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockDialer(&mockDialer{
		daemonRunning: true,
		client:        client,
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

	client := &mockClient{}

	restore := setMockDialer(&mockDialer{
		daemonRunning: true,
		client:        client,
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

	restore := setMockDialer(&mockDialer{daemonRunning: true})
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
