package ipc

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestServer_ClientCommunication(t *testing.T) {
	// Create temp directory for socket
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create handler that echoes command back
	handler := func(req Request) Response {
		switch req.Cmd {
		case "ping":
			return SuccessResponse(map[string]string{"reply": "pong"})
		case "echo":
			return SuccessResponse(map[string]string{"target": req.Target})
		default:
			return ErrorResponse("unknown command")
		}
	}

	// Start server
	server, err := NewServer(socketPath, handler)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = server.Serve(ctx) }()
	defer func() { _ = server.Close() }()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Connect client
	client, err := DialPath(socketPath)
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}
	defer client.Close()

	// Test ping command
	resp, err := client.SendCmd("ping")
	if err != nil {
		t.Fatalf("failed to send ping: %v", err)
	}
	if !resp.OK {
		t.Errorf("expected OK response, got error: %s", resp.Error)
	}

	// Test echo command with target
	resp, err = client.Send(Request{Cmd: "echo", Target: "test-target"})
	if err != nil {
		t.Fatalf("failed to send echo: %v", err)
	}
	if !resp.OK {
		t.Errorf("expected OK response, got error: %s", resp.Error)
	}

	// Test unknown command
	resp, err = client.SendCmd("unknown")
	if err != nil {
		t.Fatalf("failed to send unknown: %v", err)
	}
	if resp.OK {
		t.Error("expected error response for unknown command")
	}
	if resp.Error != "unknown command" {
		t.Errorf("unexpected error message: %s", resp.Error)
	}
}

func TestServer_MultipleClients(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	var counter int32
	handler := func(req Request) Response {
		count := atomic.AddInt32(&counter, 1)
		return SuccessResponse(map[string]int{"count": int(count)})
	}

	server, err := NewServer(socketPath, handler)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = server.Serve(ctx) }()
	defer func() { _ = server.Close() }()

	time.Sleep(50 * time.Millisecond)

	// Connect multiple clients
	client1, err := DialPath(socketPath)
	if err != nil {
		t.Fatalf("failed to connect client1: %v", err)
	}
	defer client1.Close()

	client2, err := DialPath(socketPath)
	if err != nil {
		t.Fatalf("failed to connect client2: %v", err)
	}
	defer client2.Close()

	// Both clients should be able to send commands
	_, err = client1.SendCmd("inc")
	if err != nil {
		t.Fatalf("client1 send failed: %v", err)
	}

	_, err = client2.SendCmd("inc")
	if err != nil {
		t.Fatalf("client2 send failed: %v", err)
	}

	if atomic.LoadInt32(&counter) != 2 {
		t.Errorf("expected counter=2, got %d", atomic.LoadInt32(&counter))
	}
}

func TestServer_CleanupOnClose(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	handler := func(req Request) Response {
		return SuccessResponse(nil)
	}

	server, err := NewServer(socketPath, handler)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = server.Serve(ctx) }()

	time.Sleep(50 * time.Millisecond)

	// Verify socket exists
	if _, err := os.Stat(socketPath); err != nil {
		t.Errorf("socket should exist: %v", err)
	}

	cancel()
	_ = server.Close()

	// Verify socket is cleaned up
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Error("socket should be removed after close")
	}
}

func TestDefaultPaths(t *testing.T) {
	// Just verify these don't panic and return non-empty strings
	socketPath := DefaultSocketPath()
	if socketPath == "" {
		t.Error("DefaultSocketPath returned empty string")
	}

	pidPath := DefaultPIDPath()
	if pidPath == "" {
		t.Error("DefaultPIDPath returned empty string")
	}
}

func TestIsDaemonRunning_NotRunning(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "nonexistent.sock")

	if IsDaemonRunningAt(socketPath) {
		t.Error("expected daemon to not be running")
	}
}

func TestClient_DaemonNotRunning(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "nonexistent.sock")

	_, err := DialPath(socketPath)
	if err != ErrDaemonNotRunning {
		t.Errorf("expected ErrDaemonNotRunning, got %v", err)
	}
}
