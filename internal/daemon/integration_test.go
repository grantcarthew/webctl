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

// TestDaemon_Integration runs a full daemon lifecycle test with a real browser.
// Uses testing.Short() rather than build tags so tests are still compiled and
// syntax-checked in normal builds. Run with: go test -run Integration ./...
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

	// Test that session URL updates after cross-origin navigation
	// This verifies Target.setDiscoverTargets is enabled for targetInfoChanged events
	t.Run("session_url_updates_after_navigation", func(t *testing.T) {
		// Navigate to a real external URL (cross-origin from about:blank)
		params, _ := json.Marshal(map[string]any{
			"url": "https://example.com",
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

		// Wait for navigation and target info update
		time.Sleep(2 * time.Second)

		// Verify status shows updated URL
		resp, err = client.SendCmd("status")
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

		// Check active session URL updated
		if status.ActiveSession == nil {
			t.Fatal("expected active session")
		}
		if status.ActiveSession.URL != "https://example.com/" {
			t.Errorf("expected session URL 'https://example.com/', got %q", status.ActiveSession.URL)
		}

		// Also verify the deprecated URL field for backwards compatibility
		if status.URL != "https://example.com/" {
			t.Errorf("expected status.URL 'https://example.com/', got %q", status.URL)
		}
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

// TestNetwork_Integration tests network event capture with a real browser.
// Run with: go test -run Integration ./...
func TestNetwork_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Use temp directory for socket and PID
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

	// Start daemon in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Run(ctx)
	}()

	// Wait for daemon to be ready
	if !waitForSocket(socketPath, 30*time.Second) {
		t.Fatal("daemon did not start in time")
	}

	// Connect client
	client, err := ipc.DialPath(socketPath)
	if err != nil {
		t.Fatalf("failed to connect to daemon: %v", err)
	}
	defer client.Close()

	// Test: Network entries populated after navigating to a real page
	// Note: Cross-origin navigation from about:blank creates a new session.
	// The initial Document request may not be captured because it happens
	// before the new session is attached. We verify by reloading the page
	// (same-origin) to capture the full request cycle.
	t.Run("network_entries_populated", func(t *testing.T) {
		// First navigate to example.com to establish the session
		params, _ := json.Marshal(map[string]any{
			"url": "https://example.com",
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

		// Wait for page load and session to be established
		time.Sleep(3 * time.Second)

		// Clear network buffer and reload to capture Document request
		client.Send(ipc.Request{Cmd: "clear", Target: "network"})

		// Reload page (same-origin navigation captures Document request)
		resp, err = client.Send(ipc.Request{
			Cmd:    "cdp",
			Target: "Page.reload",
		})
		if err != nil {
			t.Fatalf("reload failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("reload returned error: %s", resp.Error)
		}

		// Wait for page reload
		time.Sleep(3 * time.Second)

		// Query network entries
		resp, err = client.SendCmd("network")
		if err != nil {
			t.Fatalf("network command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("network returned error: %s", resp.Error)
		}

		var data ipc.NetworkData
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			t.Fatalf("failed to parse network data: %v", err)
		}

		if data.Count == 0 {
			t.Fatal("expected network entries after reload")
		}

		// Verify the document request to example.com exists
		var docEntry *ipc.NetworkEntry
		for i := range data.Entries {
			e := &data.Entries[i]
			if e.Type == "Document" && (e.URL == "https://example.com/" || e.URL == "https://example.com") {
				docEntry = e
				break
			}
		}

		if docEntry == nil {
			// Log what we found for debugging
			t.Logf("found %d entries:", len(data.Entries))
			for i, e := range data.Entries {
				t.Logf("  [%d] type=%s url=%s status=%d", i, e.Type, e.URL, e.Status)
			}
			t.Fatal("expected Document entry for https://example.com/")
		}

		// Verify required fields are populated
		if docEntry.Method != "GET" {
			t.Errorf("expected method GET, got %s", docEntry.Method)
		}
		if docEntry.Status != 200 {
			t.Errorf("expected status 200, got %d", docEntry.Status)
		}
		if docEntry.RequestTime == 0 {
			t.Error("expected RequestTime to be set")
		}
		if docEntry.MimeType == "" {
			t.Error("expected MimeType to be set")
		}
		if docEntry.SessionID == "" {
			t.Error("expected SessionID to be set")
		}
	})

	// Test: Fetch request captures XHR/fetch type and body
	// Uses httpbin.org directly as the page to avoid CORS issues
	t.Run("fetch_request_capture", func(t *testing.T) {
		// Clear network buffer
		client.Send(ipc.Request{Cmd: "clear", Target: "network"})

		// Navigate directly to httpbin.org/json to capture the JSON response
		params, _ := json.Marshal(map[string]any{
			"url": "https://httpbin.org/json",
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

		// Wait for request to complete
		time.Sleep(3 * time.Second)

		// Query network entries
		resp, err = client.SendCmd("network")
		if err != nil {
			t.Fatalf("network command failed: %v", err)
		}

		var data ipc.NetworkData
		json.Unmarshal(resp.Data, &data)

		// Find the document request to httpbin
		var jsonEntry *ipc.NetworkEntry
		for i := range data.Entries {
			e := &data.Entries[i]
			if e.URL == "https://httpbin.org/json" && e.Type == "Document" {
				jsonEntry = e
				break
			}
		}

		if jsonEntry == nil {
			t.Logf("entries: %+v", data.Entries)
			t.Skip("httpbin entry not found (may be blocked by network)")
		}

		// Verify entry fields
		if jsonEntry.Status != 200 {
			t.Errorf("expected status 200, got %d", jsonEntry.Status)
		}
		if jsonEntry.MimeType != "application/json" {
			t.Errorf("expected mimeType application/json, got %s", jsonEntry.MimeType)
		}
		// Body should be captured for JSON responses
		if jsonEntry.Body == "" {
			t.Log("body was not captured (may be timing issue)")
		} else {
			t.Logf("body length: %d bytes", len(jsonEntry.Body))
		}
	})

	// Test: Failed request capture
	// Navigate to a page that loads a non-existent resource
	t.Run("failed_request_capture", func(t *testing.T) {
		// Clear network buffer
		client.Send(ipc.Request{Cmd: "clear", Target: "network"})

		// Navigate to a data URL that tries to load a non-existent image
		// The image request will fail with net::ERR_NAME_NOT_RESOLVED
		dataURL := `data:text/html,<html><body><img src="https://nonexistent-domain-12345.invalid/image.png"></body></html>`
		params, _ := json.Marshal(map[string]any{
			"url": dataURL,
		})
		client.Send(ipc.Request{
			Cmd:    "cdp",
			Target: "Page.navigate",
			Params: params,
		})

		// Wait for page load and failed request - DNS failure may take time
		time.Sleep(10 * time.Second)

		// Query network entries
		resp, _ := client.SendCmd("network")
		var data ipc.NetworkData
		json.Unmarshal(resp.Data, &data)

		// Find the failed image request
		// Look for either Failed=true or Status=0 (indicates no response received)
		var failedEntry *ipc.NetworkEntry
		for i := range data.Entries {
			e := &data.Entries[i]
			if e.Failed {
				failedEntry = e
				break
			}
		}

		// If no explicit Failed entry, look for request with Status=0
		// which indicates the request never got a response
		if failedEntry == nil {
			for i := range data.Entries {
				e := &data.Entries[i]
				if e.Status == 0 && e.Type == "Image" {
					failedEntry = e
					break
				}
			}
		}

		if failedEntry == nil {
			t.Logf("entries count: %d", len(data.Entries))
			for i, e := range data.Entries {
				t.Logf("  [%d] type=%s url=%s status=%d failed=%v error=%s",
					i, e.Type, e.URL, e.Status, e.Failed, e.Error)
			}
			t.Skip("failed entry not found (DNS may have returned a result or request not captured)")
		}

		// Log what we found
		t.Logf("found failed entry: url=%s status=%d failed=%v error=%s",
			failedEntry.URL, failedEntry.Status, failedEntry.Failed, failedEntry.Error)

		// Verify entry indicates failure (either Failed=true or Status=0)
		if !failedEntry.Failed && failedEntry.Status != 0 {
			t.Errorf("expected Failed=true or Status=0, got Failed=%v Status=%d",
				failedEntry.Failed, failedEntry.Status)
		}
	})

	// Test: Empty buffer after clear
	t.Run("clear_network_buffer", func(t *testing.T) {
		// Clear network buffer
		resp, err := client.Send(ipc.Request{Cmd: "clear", Target: "network"})
		if err != nil {
			t.Fatalf("clear failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("clear returned error: %s", resp.Error)
		}

		// Query network entries
		resp, _ = client.SendCmd("network")
		var data ipc.NetworkData
		json.Unmarshal(resp.Data, &data)

		if data.Count != 0 {
			t.Errorf("expected 0 entries after clear, got %d", data.Count)
		}
	})

	// Test: Response headers captured
	t.Run("response_headers_captured", func(t *testing.T) {
		// Clear network buffer
		client.Send(ipc.Request{Cmd: "clear", Target: "network"})

		// Navigate to example.com
		params, _ := json.Marshal(map[string]any{
			"url": "https://example.com",
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
		time.Sleep(3 * time.Second)

		// Query network entries
		resp, err = client.SendCmd("network")
		if err != nil {
			t.Fatalf("network command failed: %v", err)
		}

		var data ipc.NetworkData
		json.Unmarshal(resp.Data, &data)

		t.Logf("network entries count: %d", data.Count)

		// Find document entry with URL containing example.com
		var docEntry *ipc.NetworkEntry
		for i := range data.Entries {
			e := &data.Entries[i]
			t.Logf("entry: type=%s url=%s status=%d headers=%d", e.Type, e.URL, e.Status, len(e.ResponseHeaders))
			if e.Type == "Document" && e.Status == 200 {
				docEntry = e
				break
			}
		}

		if docEntry == nil {
			t.Skip("document entry not found")
		}

		// Verify response headers are captured
		if len(docEntry.ResponseHeaders) == 0 {
			t.Error("expected response headers to be captured")
		} else {
			t.Logf("response headers count: %d", len(docEntry.ResponseHeaders))
		}

		// Check for common headers (case-insensitive check)
		found := false
		for k := range docEntry.ResponseHeaders {
			if k == "Content-Type" || k == "content-type" {
				found = true
				break
			}
		}
		if !found {
			t.Log("Content-Type header not found in response headers")
		}
	})

	// Cleanup
	client.Close()
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
