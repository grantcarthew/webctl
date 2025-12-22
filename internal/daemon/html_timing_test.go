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

// TestHTMLTiming_NetworkIdleBlocking tests the networkIdle blocking issue (BUG-003).
// This test reproduces the issue where Runtime.evaluate blocks until networkIdle fires,
// causing HTML extraction to take 10-17+ seconds instead of <1 second.
//
// Run with: go test -run TestHTMLTiming -v ./internal/daemon/
func TestHTMLTiming_NetworkIdleBlocking(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing test in short mode")
	}

	// Use /tmp directly instead of t.TempDir() because Unix socket paths
	// have a maximum length of ~108 chars and Go's TempDir creates long paths
	tmpDir, err := os.MkdirTemp("/tmp", "webctl-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	socketPath := filepath.Join(tmpDir, "webctl.sock")
	pidPath := filepath.Join(tmpDir, "webctl.pid")

	cfg := Config{
		Headless:   true,
		Port:       0,
		SocketPath: socketPath,
		PIDPath:    pidPath,
		BufferSize: 100,
		Debug:      false, // Set to true to see CDP timing details
	}

	d := New(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Run(ctx)
	}()

	// Use waitForSocketWithDiag to debug why socket isn't ready
	if !waitForSocketWithDiag(t, socketPath, 30*time.Second, errCh) {
		t.Fatal("daemon did not start in time")
	}

	client, err := ipc.DialPath(socketPath)
	if err != nil {
		t.Fatalf("failed to connect to daemon: %v", err)
	}
	defer client.Close()

	// Test 1: Navigate to example.com and immediately request HTML
	// This should trigger the networkIdle blocking issue
	t.Run("navigate_then_html_timing", func(t *testing.T) {
		// Navigate to example.com
		navParams, _ := json.Marshal(ipc.NavigateParams{URL: "https://example.com"})
		navStart := time.Now()
		resp, err := client.Send(ipc.Request{
			Cmd:    "navigate",
			Params: navParams,
		})
		navDuration := time.Since(navStart)
		if err != nil {
			t.Fatalf("navigate failed: %v", err)
		}
		if !resp.OK {
			t.Fatalf("navigate returned error: %s", resp.Error)
		}
		t.Logf("navigate completed in %v", navDuration)

		// Immediately request HTML without waiting
		htmlParams, _ := json.Marshal(ipc.HTMLParams{})
		htmlStart := time.Now()
		resp, err = client.Send(ipc.Request{
			Cmd:    "html",
			Params: htmlParams,
		})
		htmlDuration := time.Since(htmlStart)

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

		t.Logf("HTML extraction took: %v", htmlDuration)
		t.Logf("HTML length: %d bytes", len(data.HTML))

		// The bug: HTML extraction takes 10-17+ seconds due to networkIdle blocking
		// The fix: Should complete in <1 second
		if htmlDuration > 2*time.Second {
			t.Errorf("BUG-003 REPRODUCED: HTML extraction took %v (expected <2s)", htmlDuration)
			t.Log("This indicates Runtime.evaluate is blocking until networkIdle")
		} else {
			t.Logf("SUCCESS: HTML extraction completed in %v (within acceptable range)", htmlDuration)
		}
	})

	// Test 2: Multiple navigations to see if timing is consistent
	t.Run("multiple_navigation_timing", func(t *testing.T) {
		urls := []string{
			"https://example.com",
			"https://example.org",
			"data:text/html,<html><body><h1>Test</h1></body></html>",
		}

		for _, url := range urls {
			// Navigate
			navParams, _ := json.Marshal(ipc.NavigateParams{URL: url})
			resp, err := client.Send(ipc.Request{
				Cmd:    "navigate",
				Params: navParams,
			})
			if err != nil || !resp.OK {
				t.Logf("navigate to %s failed, skipping", url)
				continue
			}

			// Get HTML immediately
			htmlParams, _ := json.Marshal(ipc.HTMLParams{})
			htmlStart := time.Now()
			resp, err = client.Send(ipc.Request{
				Cmd:    "html",
				Params: htmlParams,
			})
			htmlDuration := time.Since(htmlStart)

			if err != nil || !resp.OK {
				t.Logf("html for %s failed: %v", url, resp.Error)
				continue
			}

			t.Logf("URL: %s -> HTML extraction: %v", url, htmlDuration)

			if htmlDuration > 2*time.Second {
				t.Errorf("BUG-003: %s took %v", url, htmlDuration)
			}
		}
	})

	// Test 3: Test with data URL (should be instant, no network)
	t.Run("data_url_timing", func(t *testing.T) {
		// Data URLs should not have networkIdle delays
		dataURL := "data:text/html,<html><head><title>Test</title></head><body><h1>Hello</h1></body></html>"

		navParams, _ := json.Marshal(ipc.NavigateParams{URL: dataURL})
		resp, err := client.Send(ipc.Request{
			Cmd:    "navigate",
			Params: navParams,
		})
		if err != nil || !resp.OK {
			t.Fatalf("navigate to data URL failed: %v", resp.Error)
		}

		htmlParams, _ := json.Marshal(ipc.HTMLParams{})
		htmlStart := time.Now()
		resp, err = client.Send(ipc.Request{
			Cmd:    "html",
			Params: htmlParams,
		})
		htmlDuration := time.Since(htmlStart)

		if err != nil || !resp.OK {
			t.Fatalf("html command failed: %v", resp.Error)
		}

		t.Logf("Data URL HTML extraction took: %v", htmlDuration)

		// Data URLs should definitely be fast
		if htmlDuration > 500*time.Millisecond {
			t.Errorf("Data URL HTML took %v (expected <500ms)", htmlDuration)
		}
	})

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

// TestHTMLTiming_CompareWithRod compares webctl's HTML extraction timing with Rod.
// This test helps identify what Rod does differently.
//
// Run with: go test -run TestHTMLTiming_CompareWithRod -v ./internal/daemon/
func TestHTMLTiming_CompareWithRod(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Rod comparison test in short mode")
	}

	// This test requires Rod to be importable
	// For now, we'll just document what we know from the research:
	t.Log("Rod comparison baseline (from project research):")
	t.Log("  - Rod MustNavigate(): ~18ms")
	t.Log("  - Rod MustHTML(): ~25ms")
	t.Log("  - Rod Total: ~100ms")
	t.Log("")
	t.Log("webctl with BUG-003:")
	t.Log("  - navigate: variable")
	t.Log("  - html: 10-17+ seconds (blocks on networkIdle)")
	t.Log("")
	t.Log("Key differences identified:")
	t.Log("  1. Rod uses Target.setDiscoverTargets + Target.attachToTarget(flatten:true)")
	t.Log("  2. Rod uses DOM.getOuterHTML with ObjectID (not nodeId)")
	t.Log("  3. Rod's Element() uses querySelector to get element first")
	t.Log("  4. Rod doesn't wait for networkIdle before executing JavaScript")
}

// BenchmarkHTMLExtraction benchmarks HTML extraction to quantify the issue.
//
// Run with: go test -bench=BenchmarkHTMLExtraction -benchtime=5x ./internal/daemon/
func BenchmarkHTMLExtraction(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	tmpDir := b.TempDir()
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

	go d.Run(ctx)

	// Wait for daemon
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if ipc.IsDaemonRunningAt(socketPath) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	client, err := ipc.DialPath(socketPath)
	if err != nil {
		b.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Navigate once before benchmarking
	navParams, _ := json.Marshal(ipc.NavigateParams{URL: "https://example.com"})
	resp, _ := client.Send(ipc.Request{Cmd: "navigate", Params: navParams})
	if !resp.OK {
		b.Fatalf("initial navigate failed: %s", resp.Error)
	}

	// Wait for page to fully load
	time.Sleep(2 * time.Second)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		htmlParams, _ := json.Marshal(ipc.HTMLParams{})
		resp, err := client.Send(ipc.Request{
			Cmd:    "html",
			Params: htmlParams,
		})
		if err != nil {
			b.Fatalf("html failed: %v", err)
		}
		if !resp.OK {
			b.Fatalf("html error: %s", resp.Error)
		}
	}
}

// TestHTMLTiming_RodDirectComparison uses Rod directly to establish baseline timing.
// This requires Rod to be a dependency - add if needed for debugging.
//
// To enable this test:
// 1. go get github.com/go-rod/rod
// 2. Uncomment the test below
// 3. Run: go test -run TestHTMLTiming_RodDirectComparison -v ./internal/daemon/
func TestHTMLTiming_RodDirectComparison(t *testing.T) {
	t.Skip("Rod not imported - uncomment and add dependency to enable")

	// Uncomment when Rod is available:
	/*
		browser := rod.New().MustConnect()
		defer browser.MustClose()

		// Test 1: Navigate and get HTML timing
		start := time.Now()
		page := browser.MustPage("https://example.com")
		navTime := time.Since(start)

		start = time.Now()
		html := page.MustHTML()
		htmlTime := time.Since(start)

		t.Logf("Rod navigate: %v", navTime)
		t.Logf("Rod HTML: %v", htmlTime)
		t.Logf("Rod HTML length: %d", len(html))

		// Rod should complete HTML in <100ms
		if htmlTime > 500*time.Millisecond {
			t.Errorf("Rod HTML took longer than expected: %v", htmlTime)
		}
	*/
}

// printTimingReport prints a summary of timing measurements.
func printTimingReport(t *testing.T, measurements map[string]time.Duration) {
	t.Log("\n=== Timing Report ===")
	for name, duration := range measurements {
		status := "OK"
		if duration > 2*time.Second {
			status = "SLOW (BUG-003)"
		}
		t.Logf("  %s: %v [%s]", name, duration, status)
	}
}

// Helper to format duration for comparison
func formatDuration(d time.Duration) string {
	if d > time.Second {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	return fmt.Sprintf("%dms", d.Milliseconds())
}

// waitForSocketWithDiag waits for socket with diagnostic output
func waitForSocketWithDiag(t *testing.T, path string, timeout time.Duration, errCh chan error) bool {
	deadline := time.Now().Add(timeout)
	lastLog := time.Now()
	for time.Now().Before(deadline) {
		// Check if daemon exited with error
		select {
		case err := <-errCh:
			t.Logf("DIAG: daemon exited with error: %v", err)
			return false
		default:
		}

		// Check socket
		if ipc.IsDaemonRunningAt(path) {
			t.Logf("DIAG: socket ready at %s", path)
			return true
		}

		// Log progress every 5 seconds
		if time.Since(lastLog) > 5*time.Second {
			t.Logf("DIAG: still waiting for socket at %s...", path)
			lastLog = time.Now()
		}

		time.Sleep(100 * time.Millisecond)
	}
	t.Logf("DIAG: timeout waiting for socket")
	return false
}
