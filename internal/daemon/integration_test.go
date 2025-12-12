package daemon

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
)

func TestDaemon_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Use temp directory for socket and PID
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "webctl.sock")
	pidPath := filepath.Join(tmpDir, "webctl.pid")

	cfg := Config{
		Headless:   true,
		Port:       0, // Use default
		SocketPath: socketPath,
		PIDPath:    pidPath,
		BufferSize: 100,
	}

	d := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start daemon in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Run(ctx)
	}()

	// Wait for daemon to be ready (socket exists)
	if !waitForSocket(socketPath, 30*time.Second) {
		t.Fatal("daemon did not start in time")
	}

	// Connect client
	client, err := ipc.DialPath(socketPath)
	if err != nil {
		t.Fatalf("failed to connect to daemon: %v", err)
	}
	defer client.Close()

	// Test status command
	t.Run("status", func(t *testing.T) {
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
			t.Error("expected running=true")
		}
		if status.PID == 0 {
			t.Error("expected non-zero PID")
		}
	})

	// Test CDP passthrough command
	t.Run("cdp_command", func(t *testing.T) {
		// Use Runtime.evaluate to run console.log
		params, _ := json.Marshal(map[string]any{
			"expression": `console.log("integration-test-message")`,
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "cdp",
			Target: "Runtime.evaluate",
			Params: params,
		})
		if err != nil {
			t.Fatalf("cdp command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("cdp returned error: %s", resp.Error)
		}
	})

	// Give time for events to be processed
	time.Sleep(200 * time.Millisecond)

	// Verify console event was captured
	t.Run("console_event_captured", func(t *testing.T) {
		resp, err := client.SendCmd("console")
		if err != nil {
			t.Fatalf("console command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("console returned error: %s", resp.Error)
		}

		var data ipc.ConsoleData
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			t.Fatalf("failed to parse console data: %v", err)
		}

		if data.Count == 0 {
			t.Fatal("expected at least one console entry")
		}

		// Verify our message is in the buffer
		found := false
		for _, entry := range data.Entries {
			if entry.Text == "integration-test-message" {
				found = true
				if entry.Type != "log" {
					t.Errorf("expected type 'log', got %q", entry.Type)
				}
				break
			}
		}
		if !found {
			t.Errorf("expected to find 'integration-test-message' in console entries, got: %+v", data.Entries)
		}
	})

	// Navigate to trigger network events
	t.Run("navigate_triggers_network", func(t *testing.T) {
		// Clear network buffer first
		client.Send(ipc.Request{Cmd: "clear", Target: "network"})

		// Navigate to a data URL
		params, _ := json.Marshal(map[string]any{
			"url": "data:text/html,<h1>Test</h1>",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "cdp",
			Target: "Page.navigate",
			Params: params,
		})
		if err != nil {
			t.Fatalf("navigate failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("navigate returned error: %s", resp.Error)
		}

		// Wait for page load
		time.Sleep(500 * time.Millisecond)

		// Check network entries
		resp, err = client.SendCmd("network")
		if err != nil {
			t.Fatalf("network command failed: %v", err)
		}

		var data ipc.NetworkData
		json.Unmarshal(resp.Data, &data)
		t.Logf("network entries after navigate: %d", data.Count)

		// Note: data: URLs may not generate network events in all Chrome versions
		// so we just log rather than fail
	})

	// Test clear command
	t.Run("clear", func(t *testing.T) {
		// First add a console entry
		params, _ := json.Marshal(map[string]any{
			"expression": `console.log("before-clear")`,
		})
		client.Send(ipc.Request{
			Cmd:    "cdp",
			Target: "Runtime.evaluate",
			Params: params,
		})
		time.Sleep(100 * time.Millisecond)

		// Clear console
		resp, err := client.Send(ipc.Request{Cmd: "clear", Target: "console"})
		if err != nil {
			t.Fatalf("clear command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("clear returned error: %s", resp.Error)
		}

		// Verify console is empty
		resp, _ = client.SendCmd("console")
		var data ipc.ConsoleData
		json.Unmarshal(resp.Data, &data)
		if data.Count != 0 {
			t.Errorf("expected 0 entries after clear, got %d", data.Count)
		}
	})

	// Test clear all
	t.Run("clear_all", func(t *testing.T) {
		// Add entries
		params, _ := json.Marshal(map[string]any{
			"expression": `console.log("test")`,
		})
		client.Send(ipc.Request{
			Cmd:    "cdp",
			Target: "Runtime.evaluate",
			Params: params,
		})
		time.Sleep(100 * time.Millisecond)

		// Clear all
		resp, _ := client.Send(ipc.Request{Cmd: "clear"})
		if !resp.OK {
			t.Fatalf("clear all returned error: %s", resp.Error)
		}

		// Verify both are empty
		resp, _ = client.SendCmd("console")
		var consoleData ipc.ConsoleData
		json.Unmarshal(resp.Data, &consoleData)

		resp, _ = client.SendCmd("network")
		var networkData ipc.NetworkData
		json.Unmarshal(resp.Data, &networkData)

		if consoleData.Count != 0 || networkData.Count != 0 {
			t.Errorf("expected 0 entries after clear all, got console=%d network=%d",
				consoleData.Count, networkData.Count)
		}
	})

	// Test unknown command
	t.Run("unknown_command", func(t *testing.T) {
		resp, err := client.SendCmd("nonexistent")
		if err != nil {
			t.Fatalf("send failed: %v", err)
		}
		if resp.OK {
			t.Error("expected error for unknown command")
		}
		if resp.Error == "" {
			t.Error("expected error message")
		}
	})

	// Test cdp command error handling
	t.Run("cdp_missing_target", func(t *testing.T) {
		resp, _ := client.Send(ipc.Request{Cmd: "cdp"})
		if resp.OK {
			t.Error("expected error for cdp without target")
		}
	})

	// Verify PID file exists
	t.Run("pid_file", func(t *testing.T) {
		data, err := os.ReadFile(pidPath)
		if err != nil {
			t.Fatalf("failed to read PID file: %v", err)
		}
		if len(data) == 0 {
			t.Error("PID file is empty")
		}
	})

	// Close client before shutting down daemon to avoid deadlock
	// (server.Close waits for all connections to finish)
	client.Close()

	// Shutdown
	cancel()

	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Errorf("daemon exited with error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("daemon did not shut down in time")
	}

	// Verify cleanup
	t.Run("cleanup", func(t *testing.T) {
		// Give a moment for cleanup
		time.Sleep(100 * time.Millisecond)

		if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
			t.Error("socket file should be removed after shutdown")
		}
		if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
			t.Error("PID file should be removed after shutdown")
		}
	})
}

func waitForSocket(path string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			// Socket exists, try to connect
			if ipc.IsDaemonRunningAt(path) {
				return true
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
