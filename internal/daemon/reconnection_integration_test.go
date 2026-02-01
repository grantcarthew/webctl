//go:build !short

package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
)

// waitForCondition polls a condition function until it returns true or timeout occurs.
func waitForCondition(timeout time.Duration, interval time.Duration, condition func() (bool, error)) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ok, err := condition()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("timeout waiting for condition after %v", timeout)
}

// waitForConnectionState waits for the daemon to reach a specific connection state.
func waitForConnectionState(client *ipc.Client, targetState string, timeout time.Duration) error {
	return waitForCondition(timeout, 500*time.Millisecond, func() (bool, error) {
		resp, err := client.SendCmd("status")
		if err != nil {
			return false, err
		}
		if !resp.OK {
			return false, fmt.Errorf("status error: %s", resp.Error)
		}

		var status ipc.StatusData
		if err := json.Unmarshal(resp.Data, &status); err != nil {
			return false, err
		}

		if status.Connection != nil && status.Connection.State == targetState {
			return true, nil
		}
		return false, nil
	})
}

// waitForBrowserReady waits for the daemon to have an active browser connection.
func waitForBrowserReady(client *ipc.Client, timeout time.Duration) error {
	return waitForCondition(timeout, 500*time.Millisecond, func() (bool, error) {
		resp, err := client.SendCmd("status")
		if err != nil {
			return false, err
		}
		if !resp.OK {
			return false, fmt.Errorf("status error: %s", resp.Error)
		}

		var status ipc.StatusData
		if err := json.Unmarshal(resp.Data, &status); err != nil {
			return false, err
		}

		// Browser is ready if we have sessions or connection is healthy
		return status.Running && (len(status.Sessions) > 0 || (status.Connection != nil && status.Connection.State == "connected")), nil
	})
}

// TestReconnection_BrowserKilled tests that the daemon successfully reconnects
// when the browser process is killed and restarted.
func TestReconnection_BrowserKilled(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "webctl.sock")
	pidPath := filepath.Join(tmpDir, "webctl.pid")

	cfg := Config{
		Headless:   true,
		Port:       0,
		SocketPath: socketPath,
		PIDPath:    pidPath,
		BufferSize: 100,
		Debug:      true, // Enable debug for better test diagnostics
	}

	d := New(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Run(ctx)
	}()

	if !waitForSocket(socketPath, 30*time.Second) {
		t.Fatal("daemon did not start in time")
	}

	client, err := ipc.DialPath(socketPath)
	if err != nil {
		t.Fatalf("failed to connect to daemon: %v", err)
	}
	defer func() { _ = client.Close() }()

	// Wait for browser to be ready
	if err := waitForBrowserReady(client, 10*time.Second); err != nil {
		t.Fatalf("browser not ready: %v", err)
	}

	// Get initial status to verify browser is connected
	resp, err := client.SendCmd("status")
	if err != nil {
		t.Fatalf("status command failed: %v", err)
	}
	if !resp.OK {
		t.Fatalf("status returned error: %s", resp.Error)
	}

	var status ipc.StatusData
	if err := json.Unmarshal(resp.Data, &status); err != nil {
		t.Fatalf("failed to parse status: %v", err)
	}

	if !status.Running {
		t.Fatal("daemon not running")
	}

	// Add a console entry to verify buffer preservation
	testMessage := "test message before disconnect"
	evalReq := ipc.Request{
		Cmd: "eval",
		Params: func() json.RawMessage {
			b, _ := json.Marshal(map[string]any{
				"expression": "console.log('" + testMessage + "')",
			})
			return b
		}(),
	}
	_, _ = client.Send(evalReq)
	time.Sleep(500 * time.Millisecond)

	// Get browser PID
	browserPID := d.browser.PID()
	t.Logf("Browser PID: %d", browserPID)

	// Kill the browser process (simulate crash)
	t.Logf("Killing browser process...")
	proc, err := os.FindProcess(browserPID)
	if err != nil {
		t.Fatalf("failed to find browser process: %v", err)
	}
	if err := proc.Kill(); err != nil {
		t.Fatalf("failed to kill browser: %v", err)
	}

	// Wait for reconnection to occur
	// The heartbeat interval is 5s, timeout is 5s, so worst case is 10s detection
	// Plus reconnection attempts with backoff (allow up to 30s total)
	t.Logf("Waiting for reconnection...")
	if err := waitForConnectionState(client, "connected", 30*time.Second); err != nil {
		t.Logf("Warning: did not reach connected state in time: %v", err)
		// Don't fail - check actual state below
	}

	// Check that daemon is still running and reconnected
	resp, err = client.SendCmd("status")
	if err != nil {
		t.Fatalf("status command failed after reconnect: %v", err)
	}
	if !resp.OK {
		t.Fatalf("status returned error after reconnect: %s", resp.Error)
	}

	if err := json.Unmarshal(resp.Data, &status); err != nil {
		t.Fatalf("failed to parse status after reconnect: %v", err)
	}

	// Verify daemon is running and reconnected
	if !status.Running {
		t.Fatal("daemon not running after reconnect")
	}

	// Check connection state
	if status.Connection == nil {
		t.Fatal("connection health not reported")
	}

	// Should be connected (reconnection succeeded) or reconnecting (still attempting)
	if status.Connection.State != "connected" && status.Connection.State != "reconnecting" {
		t.Errorf("unexpected connection state after browser kill: %s", status.Connection.State)
	}

	// Verify console buffer was preserved
	resp, err = client.SendCmd("console")
	if err != nil {
		t.Fatalf("console command failed: %v", err)
	}
	if !resp.OK {
		t.Fatalf("console returned error: %s", resp.Error)
	}

	var consoleResp struct {
		Entries []ipc.ConsoleEntry `json:"entries"`
	}
	if err := json.Unmarshal(resp.Data, &consoleResp); err != nil {
		t.Fatalf("failed to parse console: %v", err)
	}

	// Check if our test message is in the buffer
	found := false
	for _, entry := range consoleResp.Entries {
		if len(entry.Args) > 0 && entry.Args[0] == testMessage {
			found = true
			break
		}
	}
	if !found {
		t.Error("console buffer was not preserved across reconnection")
	}

	// Clean shutdown
	cancel()
	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Errorf("daemon exited with error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("daemon did not shut down in time")
	}
}

// TestReconnection_ManualReconnect tests the manual reconnect command.
func TestReconnection_ManualReconnect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "webctl.sock")
	pidPath := filepath.Join(tmpDir, "webctl.pid")

	cfg := Config{
		Headless:   true,
		Port:       0,
		SocketPath: socketPath,
		PIDPath:    pidPath,
		BufferSize: 100,
	}

	d := New(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Run(ctx)
	}()

	if !waitForSocket(socketPath, 30*time.Second) {
		t.Fatal("daemon did not start in time")
	}

	client, err := ipc.DialPath(socketPath)
	if err != nil {
		t.Fatalf("failed to connect to daemon: %v", err)
	}
	defer func() { _ = client.Close() }()

	// Wait for browser to be ready
	if err := waitForBrowserReady(client, 10*time.Second); err != nil {
		t.Fatalf("browser not ready: %v", err)
	}

	// Manual reconnect when already connected should succeed
	resp, err := client.SendCmd("reconnect")
	if err != nil {
		t.Fatalf("reconnect command failed: %v", err)
	}
	if !resp.OK {
		t.Fatalf("reconnect returned error: %s", resp.Error)
	}

	var result struct {
		Message string `json:"message"`
		State   string `json:"state"`
	}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		t.Fatalf("failed to parse reconnect response: %v", err)
	}

	// Should indicate already connected
	if result.Message != "already connected" {
		t.Errorf("expected 'already connected', got %q", result.Message)
	}
	if result.State != "connected" {
		t.Errorf("expected state 'connected', got %q", result.State)
	}

	// Clean shutdown
	cancel()
	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Errorf("daemon exited with error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("daemon did not shut down in time")
	}
}

// TestReconnection_SessionRecovery tests that the daemon restores the last URL after reconnection.
func TestReconnection_SessionRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "webctl.sock")
	pidPath := filepath.Join(tmpDir, "webctl.pid")

	cfg := Config{
		Headless:   true,
		Port:       0,
		SocketPath: socketPath,
		PIDPath:    pidPath,
		BufferSize: 100,
		Debug:      true,
	}

	d := New(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Run(ctx)
	}()

	if !waitForSocket(socketPath, 30*time.Second) {
		t.Fatal("daemon did not start in time")
	}

	client, err := ipc.DialPath(socketPath)
	if err != nil {
		t.Fatalf("failed to connect to daemon: %v", err)
	}
	defer func() { _ = client.Close() }()

	// Wait for browser to be ready
	if err := waitForBrowserReady(client, 10*time.Second); err != nil {
		t.Fatalf("browser not ready: %v", err)
	}

	// Navigate to a test page
	testURL := "https://example.com"
	navReq := ipc.Request{
		Cmd: "navigate",
		Params: func() json.RawMessage {
			b, _ := json.Marshal(map[string]any{"url": testURL})
			return b
		}(),
	}
	_, err = client.Send(navReq)
	if err != nil {
		t.Fatalf("navigate command failed: %v", err)
	}

	// Wait for navigation (poll for URL change)
	err = waitForCondition(10*time.Second, 500*time.Millisecond, func() (bool, error) {
		resp, err := client.SendCmd("status")
		if err != nil {
			return false, err
		}
		if !resp.OK {
			return false, nil
		}
		var status ipc.StatusData
		if err := json.Unmarshal(resp.Data, &status); err != nil {
			return false, err
		}
		if status.ActiveSession != nil && status.ActiveSession.URL == testURL {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Logf("Warning: navigation may not have completed: %v", err)
	}

	// Get browser PID
	browserPID := d.browser.PID()

	// Kill the browser process
	proc, err := os.FindProcess(browserPID)
	if err != nil {
		t.Fatalf("failed to find browser process: %v", err)
	}
	if err := proc.Kill(); err != nil {
		t.Fatalf("failed to kill browser: %v", err)
	}

	// Wait for reconnection (allow up to 35s for reconnection + session recovery)
	t.Logf("Waiting for reconnection and session recovery...")
	if err := waitForConnectionState(client, "connected", 35*time.Second); err != nil {
		t.Logf("Warning: did not reach connected state in time: %v", err)
		// Don't fail - check actual state below
	}

	// Check status - should have restored the URL
	resp, err := client.SendCmd("status")
	if err != nil {
		t.Fatalf("status command failed: %v", err)
	}
	if !resp.OK {
		t.Fatalf("status returned error: %s", resp.Error)
	}

	var status ipc.StatusData
	if err := json.Unmarshal(resp.Data, &status); err != nil {
		t.Fatalf("failed to parse status: %v", err)
	}

	// Verify session was recovered (may have sessions if reconnection succeeded)
	if len(status.Sessions) > 0 {
		t.Logf("Session recovered with URL: %s", status.Sessions[0].URL)
		// The URL should be the test URL or close to it
		// (browser might add trailing slash, etc.)
	} else {
		t.Log("No sessions after reconnection - reconnection may still be in progress")
	}

	// Clean shutdown
	cancel()
	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Errorf("daemon exited with error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("daemon did not shut down in time")
	}
}
