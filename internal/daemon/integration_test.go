package daemon

import (
	"bytes"
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
	defer func() { _ = client.Close() }()

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
		_, _ = client.Send(ipc.Request{Cmd: "clear", Target: "network"})

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
		_ = json.Unmarshal(resp.Data, &data)
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
	})

	// Test clear command
	t.Run("clear", func(t *testing.T) {
		// First add a console entry
		params, _ := json.Marshal(map[string]any{
			"expression": `console.log("before-clear")`,
		})
		_, _ = client.Send(ipc.Request{
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
		_ = json.Unmarshal(resp.Data, &data)
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
		_, _ = client.Send(ipc.Request{
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
		_ = json.Unmarshal(resp.Data, &consoleData)

		resp, _ = client.SendCmd("network")
		var networkData ipc.NetworkData
		_ = json.Unmarshal(resp.Data, &networkData)

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
	_ = client.Close()

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
	defer func() { _ = client.Close() }()

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

		// Enable Network domain by calling network command first (lazy enablement)
		// Then clear buffer and reload to capture Document request
		_, _ = client.SendCmd("network") // This enables Network.enable lazily
		_, _ = client.Send(ipc.Request{Cmd: "clear", Target: "network"})

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
		_, _ = client.Send(ipc.Request{Cmd: "clear", Target: "network"})

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
		_ = json.Unmarshal(resp.Data, &data)

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
		_, _ = client.Send(ipc.Request{Cmd: "clear", Target: "network"})

		// Navigate to a data URL that tries to load a non-existent image
		// The image request will fail with net::ERR_NAME_NOT_RESOLVED
		dataURL := `data:text/html,<html><body><img src="https://nonexistent-domain-12345.invalid/image.png"></body></html>`
		params, _ := json.Marshal(map[string]any{
			"url": dataURL,
		})
		_, _ = client.Send(ipc.Request{
			Cmd:    "cdp",
			Target: "Page.navigate",
			Params: params,
		})

		// Wait for page load and failed request - DNS failure may take time
		time.Sleep(10 * time.Second)

		// Query network entries
		resp, _ := client.SendCmd("network")
		var data ipc.NetworkData
		_ = json.Unmarshal(resp.Data, &data)

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
		_ = json.Unmarshal(resp.Data, &data)

		if data.Count != 0 {
			t.Errorf("expected 0 entries after clear, got %d", data.Count)
		}
	})

	// Test: Response headers captured
	t.Run("response_headers_captured", func(t *testing.T) {
		// Clear network buffer
		_, _ = client.Send(ipc.Request{Cmd: "clear", Target: "network"})

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
		_ = json.Unmarshal(resp.Data, &data)

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
	_ = client.Close()
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

// TestScreenshot_Integration tests screenshot capture with a real browser.
// Run with: go test -run Integration ./...
func TestScreenshot_Integration(t *testing.T) {
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

	// Navigate to a test page first
	params, _ := json.Marshal(map[string]any{
		"url": "data:text/html,<html><head><title>Screenshot Test</title></head><body><h1>Test Page</h1></body></html>",
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

	time.Sleep(500 * time.Millisecond)

	// Test: Basic viewport screenshot
	t.Run("basic_viewport_screenshot", func(t *testing.T) {
		params, _ := json.Marshal(ipc.ScreenshotParams{
			FullPage: false,
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "screenshot",
			Params: params,
		})
		if err != nil {
			t.Fatalf("screenshot command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("screenshot returned error: %s", resp.Error)
		}

		var data ipc.ScreenshotData
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			t.Fatalf("failed to parse screenshot data: %v", err)
		}

		if len(data.Data) == 0 {
			t.Fatal("expected screenshot data")
		}

		// Verify PNG header
		if len(data.Data) < 8 {
			t.Fatal("screenshot data too small")
		}
		pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
		if !bytes.Equal(data.Data[:8], pngHeader) {
			t.Errorf("invalid PNG header: got %x", data.Data[:8])
		}

		t.Logf("screenshot size: %d bytes", len(data.Data))
	})

	// Test: Full-page screenshot
	t.Run("full_page_screenshot", func(t *testing.T) {
		params, _ := json.Marshal(ipc.ScreenshotParams{
			FullPage: true,
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "screenshot",
			Params: params,
		})
		if err != nil {
			t.Fatalf("screenshot command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("screenshot returned error: %s", resp.Error)
		}

		var data ipc.ScreenshotData
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			t.Fatalf("failed to parse screenshot data: %v", err)
		}

		if len(data.Data) == 0 {
			t.Fatal("expected screenshot data")
		}

		// Verify PNG header
		pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
		if !bytes.Equal(data.Data[:8], pngHeader) {
			t.Errorf("invalid PNG header: got %x", data.Data[:8])
		}

		t.Logf("full-page screenshot size: %d bytes", len(data.Data))
	})

	// Test: Screenshot with no active session should fail gracefully
	t.Run("no_active_session", func(t *testing.T) {
		// This test would need session manipulation which isn't easy in current architecture
		// We'll skip for now but document the expected behavior
		t.Skip("session manipulation not easily testable in current architecture")
	})

	// Test: Screenshot after navigation updates session
	t.Run("screenshot_after_navigation", func(t *testing.T) {
		// Navigate to new page
		params, _ := json.Marshal(map[string]any{
			"url": "data:text/html,<html><head><title>Second Page</title></head><body><h1>Page 2</h1></body></html>",
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

		time.Sleep(500 * time.Millisecond)

		// Capture screenshot
		screenshotParams, _ := json.Marshal(ipc.ScreenshotParams{
			FullPage: false,
		})
		resp, err = client.Send(ipc.Request{
			Cmd:    "screenshot",
			Params: screenshotParams,
		})
		if err != nil {
			t.Fatalf("screenshot command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("screenshot returned error: %s", resp.Error)
		}

		var data ipc.ScreenshotData
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			t.Fatalf("failed to parse screenshot data: %v", err)
		}

		if len(data.Data) == 0 {
			t.Fatal("expected screenshot data after navigation")
		}

		t.Logf("screenshot after navigation size: %d bytes", len(data.Data))
	})

	// Test: Screenshot command validates fullPage parameter
	t.Run("parameter_validation", func(t *testing.T) {
		// Test with explicit false
		params, _ := json.Marshal(ipc.ScreenshotParams{
			FullPage: false,
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "screenshot",
			Params: params,
		})
		if err != nil {
			t.Fatalf("screenshot command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("screenshot returned error: %s", resp.Error)
		}
	})

	_ = client.Close()
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

// TestHTML_Integration tests HTML extraction with a real browser.
// Run with: go test -run Integration ./...
func TestHTML_Integration(t *testing.T) {
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

	// Navigate to a test page first
	testHTML := `<html><head><title>HTML Test</title></head><body><h1>Test Page</h1><div class="content">Content 1</div><div class="content">Content 2</div><p id="unique">Unique element</p></body></html>`
	params, _ := json.Marshal(map[string]any{
		"url": "data:text/html," + testHTML,
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

	time.Sleep(500 * time.Millisecond)

	// Test: Full page HTML
	t.Run("full_page_html", func(t *testing.T) {
		params, _ := json.Marshal(ipc.HTMLParams{})
		resp, err := client.Send(ipc.Request{
			Cmd:    "html",
			Params: params,
		})
		if err != nil {
			t.Fatalf("html command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("html returned error: %s", resp.Error)
		}

		var data ipc.HTMLData
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			t.Fatalf("failed to parse HTML data: %v", err)
		}

		if len(data.HTML) == 0 {
			t.Fatal("expected HTML data")
		}

		// Verify it contains expected elements
		if !bytes.Contains([]byte(data.HTML), []byte("HTML Test")) {
			t.Error("HTML should contain title")
		}
		if !bytes.Contains([]byte(data.HTML), []byte("Test Page")) {
			t.Error("HTML should contain h1 content")
		}

		t.Logf("full page HTML length: %d bytes", len(data.HTML))
	})

	// Test: Single element match
	t.Run("single_element_selector", func(t *testing.T) {
		params, _ := json.Marshal(ipc.HTMLParams{
			Selector: "#unique",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "html",
			Params: params,
		})
		if err != nil {
			t.Fatalf("html command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("html returned error: %s", resp.Error)
		}

		var data ipc.HTMLData
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			t.Fatalf("failed to parse HTML data: %v", err)
		}

		if !bytes.Contains([]byte(data.HTML), []byte("Unique element")) {
			t.Error("HTML should contain unique element content")
		}
		if !bytes.Contains([]byte(data.HTML), []byte(`id="unique"`)) {
			t.Error("HTML should include element attributes")
		}

		t.Logf("element HTML: %s", data.HTML)
	})

	// Test: Multiple element matches
	t.Run("multiple_element_matches", func(t *testing.T) {
		params, _ := json.Marshal(ipc.HTMLParams{
			Selector: ".content",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "html",
			Params: params,
		})
		if err != nil {
			t.Fatalf("html command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("html returned error: %s", resp.Error)
		}

		var data ipc.HTMLData
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			t.Fatalf("failed to parse HTML data: %v", err)
		}

		// Should contain both content elements
		if !bytes.Contains([]byte(data.HTML), []byte("Content 1")) {
			t.Error("HTML should contain first content element")
		}
		if !bytes.Contains([]byte(data.HTML), []byte("Content 2")) {
			t.Error("HTML should contain second content element")
		}

		// Should contain -- separator between elements (consistent with other observation commands)
		if !bytes.Contains([]byte(data.HTML), []byte("--")) {
			t.Error("HTML should contain element separator")
		}

		t.Logf("multiple elements HTML length: %d bytes", len(data.HTML))
	})

	// Test: Selector matches no elements
	t.Run("no_match_error", func(t *testing.T) {
		params, _ := json.Marshal(ipc.HTMLParams{
			Selector: ".nonexistent",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "html",
			Params: params,
		})
		if err != nil {
			t.Fatalf("html command failed: %v", err)
		}

		if resp.OK {
			t.Error("expected error for non-matching selector")
		}

		if !bytes.Contains([]byte(resp.Error), []byte("matched no elements")) {
			t.Errorf("error should mention no matches, got: %s", resp.Error)
		}
	})

	// Test: Complex selector
	t.Run("complex_selector", func(t *testing.T) {
		params, _ := json.Marshal(ipc.HTMLParams{
			Selector: "body > h1",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "html",
			Params: params,
		})
		if err != nil {
			t.Fatalf("html command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("html returned error: %s", resp.Error)
		}

		var data ipc.HTMLData
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			t.Fatalf("failed to parse HTML data: %v", err)
		}

		if !bytes.Contains([]byte(data.HTML), []byte("<h1>Test Page</h1>")) {
			t.Error("HTML should contain h1 element")
		}
	})

	_ = client.Close()
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

// TestDaemon_EvalCommand tests JavaScript evaluation functionality.
func TestDaemon_EvalCommand(t *testing.T) {
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

	// Navigate to a test page
	params, _ := json.Marshal(map[string]any{
		"url": "data:text/html,<html><head><title>Eval Test</title></head><body><h1>Test</h1></body></html>",
	})
	_, _ = client.Send(ipc.Request{
		Cmd:    "cdp",
		Target: "Page.navigate",
		Params: params,
	})
	time.Sleep(200 * time.Millisecond)

	// Test: Basic arithmetic expression
	t.Run("basic_arithmetic", func(t *testing.T) {
		params, _ := json.Marshal(ipc.EvalParams{
			Expression: "1 + 1",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "eval",
			Params: params,
		})
		if err != nil {
			t.Fatalf("eval command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("eval returned error: %s", resp.Error)
		}

		var data ipc.EvalData
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			t.Fatalf("failed to parse eval data: %v", err)
		}

		if !data.HasValue {
			t.Error("expected HasValue=true")
		}

		if data.Value != float64(2) {
			t.Errorf("expected value=2, got %v", data.Value)
		}
	})

	// Test: String expression
	t.Run("string_expression", func(t *testing.T) {
		params, _ := json.Marshal(ipc.EvalParams{
			Expression: "document.title",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "eval",
			Params: params,
		})
		if err != nil {
			t.Fatalf("eval command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("eval returned error: %s", resp.Error)
		}

		var data ipc.EvalData
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			t.Fatalf("failed to parse eval data: %v", err)
		}

		if !data.HasValue {
			t.Error("expected HasValue=true")
		}

		if data.Value != "Eval Test" {
			t.Errorf("expected value='Eval Test', got %v", data.Value)
		}
	})

	// Test: Undefined value
	t.Run("undefined_value", func(t *testing.T) {
		params, _ := json.Marshal(ipc.EvalParams{
			Expression: "undefined",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "eval",
			Params: params,
		})
		if err != nil {
			t.Fatalf("eval command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("eval returned error: %s", resp.Error)
		}

		var data ipc.EvalData
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			t.Fatalf("failed to parse eval data: %v", err)
		}

		if data.HasValue {
			t.Error("expected HasValue=false for undefined")
		}
	})

	// Test: Promise resolution
	t.Run("promise_resolution", func(t *testing.T) {
		params, _ := json.Marshal(ipc.EvalParams{
			Expression: "Promise.resolve(42)",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "eval",
			Params: params,
		})
		if err != nil {
			t.Fatalf("eval command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("eval returned error: %s", resp.Error)
		}

		var data ipc.EvalData
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			t.Fatalf("failed to parse eval data: %v", err)
		}

		if !data.HasValue {
			t.Error("expected HasValue=true")
		}

		if data.Value != float64(42) {
			t.Errorf("expected value=42, got %v", data.Value)
		}
	})

	// Test: JavaScript error
	t.Run("javascript_error", func(t *testing.T) {
		params, _ := json.Marshal(ipc.EvalParams{
			Expression: "nonexistent.property",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "eval",
			Params: params,
		})
		if err != nil {
			t.Fatalf("eval command failed: %v", err)
		}
		if resp.OK {
			t.Fatal("expected error for invalid expression")
		}

		if !bytes.Contains([]byte(resp.Error), []byte("not defined")) {
			t.Errorf("error should mention 'not defined', got: %s", resp.Error)
		}
	})

	// Test: Complex object return
	t.Run("complex_object", func(t *testing.T) {
		params, _ := json.Marshal(ipc.EvalParams{
			Expression: "({foo: 'bar', num: 123})",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "eval",
			Params: params,
		})
		if err != nil {
			t.Fatalf("eval command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("eval returned error: %s", resp.Error)
		}

		var data ipc.EvalData
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			t.Fatalf("failed to parse eval data: %v", err)
		}

		if !data.HasValue {
			t.Error("expected HasValue=true")
		}

		obj, ok := data.Value.(map[string]any)
		if !ok {
			t.Fatalf("expected object, got %T", data.Value)
		}

		if obj["foo"] != "bar" {
			t.Errorf("expected foo='bar', got %v", obj["foo"])
		}

		if obj["num"] != float64(123) {
			t.Errorf("expected num=123, got %v", obj["num"])
		}
	})

	// Test: Empty expression error
	t.Run("empty_expression", func(t *testing.T) {
		params, _ := json.Marshal(ipc.EvalParams{
			Expression: "",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "eval",
			Params: params,
		})
		if err != nil {
			t.Fatalf("eval command failed: %v", err)
		}
		if resp.OK {
			t.Fatal("expected error for empty expression")
		}

		if !bytes.Contains([]byte(resp.Error), []byte("expression is required")) {
			t.Errorf("error should mention required expression, got: %s", resp.Error)
		}
	})

	_ = client.Close()
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

// TestDaemon_CookiesCommand tests cookie management functionality.
func TestDaemon_CookiesCommand(t *testing.T) {
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

	// Navigate to a test page - use http://localhost for cookie support
	params, _ := json.Marshal(map[string]any{
		"url": "http://localhost/test",
	})
	_, _ = client.Send(ipc.Request{
		Cmd:    "cdp",
		Target: "Page.navigate",
		Params: params,
	})
	time.Sleep(200 * time.Millisecond)

	// Test: List cookies when empty
	t.Run("list_empty", func(t *testing.T) {
		params, _ := json.Marshal(ipc.CookiesParams{
			Action: "list",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "cookies",
			Params: params,
		})
		if err != nil {
			t.Fatalf("cookies command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("cookies returned error: %s", resp.Error)
		}

		var data ipc.CookiesData
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			t.Fatalf("failed to parse cookies data: %v", err)
		}

		if data.Count != 0 {
			t.Errorf("expected count=0, got %d", data.Count)
		}
	})

	// Test: Set a cookie
	t.Run("set_basic_cookie", func(t *testing.T) {
		params, _ := json.Marshal(ipc.CookiesParams{
			Action: "set",
			Name:   "test_session",
			Value:  "abc123",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "cookies",
			Params: params,
		})
		if err != nil {
			t.Fatalf("cookies set failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("cookies set returned error: %s", resp.Error)
		}
	})

	// Test: Verify cookie appears in list
	t.Run("list_after_set", func(t *testing.T) {
		params, _ := json.Marshal(ipc.CookiesParams{
			Action: "list",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "cookies",
			Params: params,
		})
		if err != nil {
			t.Fatalf("cookies command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("cookies returned error: %s", resp.Error)
		}

		var data ipc.CookiesData
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			t.Fatalf("failed to parse cookies data: %v", err)
		}

		if data.Count == 0 {
			t.Fatal("expected at least one cookie")
		}

		found := false
		for _, cookie := range data.Cookies {
			if cookie.Name == "test_session" && cookie.Value == "abc123" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected to find test_session cookie")
		}
	})

	// Test: Set cookie with flags
	t.Run("set_cookie_with_flags", func(t *testing.T) {
		params, _ := json.Marshal(ipc.CookiesParams{
			Action:   "set",
			Name:     "secure_token",
			Value:    "xyz789",
			Secure:   true,
			HTTPOnly: true,
			MaxAge:   3600,
			SameSite: "Strict",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "cookies",
			Params: params,
		})
		if err != nil {
			t.Fatalf("cookies set failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("cookies set returned error: %s", resp.Error)
		}

		// Verify it was set with correct attributes
		listParams, _ := json.Marshal(ipc.CookiesParams{Action: "list"})
		resp, err = client.Send(ipc.Request{
			Cmd:    "cookies",
			Params: listParams,
		})
		if err != nil {
			t.Fatalf("cookies list failed: %v", err)
		}

		var data ipc.CookiesData
		_ = json.Unmarshal(resp.Data, &data)

		found := false
		for _, cookie := range data.Cookies {
			if cookie.Name == "secure_token" {
				found = true
				if !cookie.Secure {
					t.Error("expected secure=true")
				}
				if !cookie.HTTPOnly {
					t.Error("expected httpOnly=true")
				}
				if cookie.SameSite != "Strict" {
					t.Errorf("expected sameSite=Strict, got %s", cookie.SameSite)
				}
				break
			}
		}
		if !found {
			t.Error("expected to find secure_token cookie")
		}
	})

	// Test: Delete existing cookie
	t.Run("delete_existing_cookie", func(t *testing.T) {
		params, _ := json.Marshal(ipc.CookiesParams{
			Action: "delete",
			Name:   "test_session",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "cookies",
			Params: params,
		})
		if err != nil {
			t.Fatalf("cookies delete failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("cookies delete returned error: %s", resp.Error)
		}

		// Verify it was deleted
		listParams, _ := json.Marshal(ipc.CookiesParams{Action: "list"})
		resp, err = client.Send(ipc.Request{
			Cmd:    "cookies",
			Params: listParams,
		})
		if err != nil {
			t.Fatalf("cookies list failed: %v", err)
		}

		var data ipc.CookiesData
		_ = json.Unmarshal(resp.Data, &data)

		for _, cookie := range data.Cookies {
			if cookie.Name == "test_session" {
				t.Error("cookie should have been deleted")
			}
		}
	})

	// Test: Delete non-existent cookie (idempotent)
	t.Run("delete_nonexistent_cookie", func(t *testing.T) {
		params, _ := json.Marshal(ipc.CookiesParams{
			Action: "delete",
			Name:   "nonexistent",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "cookies",
			Params: params,
		})
		if err != nil {
			t.Fatalf("cookies delete failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("cookies delete should succeed for nonexistent cookie, got error: %s", resp.Error)
		}
	})

	// Test: Empty cookie name error
	t.Run("empty_name_error", func(t *testing.T) {
		params, _ := json.Marshal(ipc.CookiesParams{
			Action: "set",
			Name:   "",
			Value:  "test",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "cookies",
			Params: params,
		})
		if err != nil {
			t.Fatalf("cookies command failed: %v", err)
		}
		if resp.OK {
			t.Fatal("expected error for empty cookie name")
		}

		if !bytes.Contains([]byte(resp.Error), []byte("name is required")) {
			t.Errorf("error should mention required name, got: %s", resp.Error)
		}
	})

	_ = client.Close()
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

// TestFind_Integration tests find command with minified HTML.
// Verifies that HTML formatting makes search results readable.
// Run with: go test -run Integration ./...
func TestFind_Integration(t *testing.T) {
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

	// Navigate to a test page with minified HTML
	// This simulates modern frameworks that output single-line HTML
	minifiedHTML := `<!DOCTYPE html><html><head><title>Minified Page</title></head><body><div class="container"><header><h1>Welcome</h1></header><main><article class="post"><h2>Article Title</h2><p>This is a test paragraph with searchable text.</p></article><aside><p>Sidebar content here.</p></aside></main><footer>Footer text</footer></div></body></html>`
	params, _ := json.Marshal(map[string]any{
		"url": "data:text/html," + minifiedHTML,
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

	time.Sleep(500 * time.Millisecond)

	// Test: Search for text in minified HTML
	// REMOVED: Find command removed per DR-030 (use html --find instead)
	t.Run("search_minified_html", func(t *testing.T) {
		t.Skip("find command removed per DR-030 - use html --find instead")
	})

	// Test: Regex search in minified HTML
	// REMOVED: Find command removed per DR-030 (use html --find instead)
	t.Run("regex_search", func(t *testing.T) {
		t.Skip("find command removed per DR-030 - use html --find instead")
	})

	// Test: No matches
	// REMOVED: Find command removed per DR-030 (use html --find instead)
	t.Run("no_matches", func(t *testing.T) {
		t.Skip("find command removed per DR-030 - use html --find instead")
	})

	_ = client.Close()
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

// TestType_Integration tests type and key commands trigger correct DOM events.
// Verifies that Enter key triggers keydown, keypress, and keyup events.
// Run with: go test -run Integration ./...
func TestType_Integration(t *testing.T) {
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

	// Navigate to a test page with an input field
	testHTML := `<!DOCTYPE html><html><body>
		<input type="text" id="input">
		<script>
			window._events = [];
			const input = document.getElementById('input');
			['keydown', 'keypress', 'keyup'].forEach(type => {
				input.addEventListener(type, e => {
					if (e.key === 'Enter') window._events.push(type);
				});
			});
		</script>
	</body></html>`
	params, _ := json.Marshal(map[string]any{
		"url": "data:text/html," + testHTML,
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

	time.Sleep(500 * time.Millisecond)

	// Test: Type with Enter key triggers all three DOM events
	t.Run("enter_triggers_all_events", func(t *testing.T) {
		// Type text with Enter key
		typeParams, _ := json.Marshal(ipc.TypeParams{
			Selector: "#input",
			Text:     "test",
			Key:      "Enter",
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "type",
			Params: typeParams,
		})
		if err != nil {
			t.Fatalf("type command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("type returned error: %s", resp.Error)
		}

		// Check which events were triggered
		evalParams, _ := json.Marshal(ipc.EvalParams{
			Expression: "JSON.stringify(window._events)",
		})
		resp, err = client.Send(ipc.Request{
			Cmd:    "eval",
			Params: evalParams,
		})
		if err != nil {
			t.Fatalf("eval command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("eval returned error: %s", resp.Error)
		}

		var evalData ipc.EvalData
		if err := json.Unmarshal(resp.Data, &evalData); err != nil {
			t.Fatalf("failed to parse eval data: %v", err)
		}

		eventsJSON, ok := evalData.Value.(string)
		if !ok {
			t.Fatalf("expected string, got %T", evalData.Value)
		}

		var events []string
		if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
			t.Fatalf("failed to parse events: %v", err)
		}

		// Verify all three events were triggered
		expected := []string{"keydown", "keypress", "keyup"}
		if len(events) != len(expected) {
			t.Errorf("expected %d events, got %d: %v", len(expected), len(events), events)
		}
		for i, exp := range expected {
			if i >= len(events) || events[i] != exp {
				t.Errorf("event[%d] = %q, want %q", i, events[i], exp)
			}
		}
	})

	// Test: Text is inserted into input
	t.Run("text_inserted", func(t *testing.T) {
		// Clear and type new text
		typeParams, _ := json.Marshal(ipc.TypeParams{
			Selector: "#input",
			Text:     "hello",
			Clear:    true,
		})
		resp, err := client.Send(ipc.Request{
			Cmd:    "type",
			Params: typeParams,
		})
		if err != nil {
			t.Fatalf("type command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("type returned error: %s", resp.Error)
		}

		// Verify input value
		evalParams, _ := json.Marshal(ipc.EvalParams{
			Expression: "document.getElementById('input').value",
		})
		resp, err = client.Send(ipc.Request{
			Cmd:    "eval",
			Params: evalParams,
		})
		if err != nil {
			t.Fatalf("eval command failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("eval returned error: %s", resp.Error)
		}

		var evalData ipc.EvalData
		_ = json.Unmarshal(resp.Data, &evalData)

		if evalData.Value != "hello" {
			t.Errorf("input value = %q, want %q", evalData.Value, "hello")
		}
	})

	_ = client.Close()
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
