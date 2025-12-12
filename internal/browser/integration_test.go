//go:build integration

package browser

import (
	"context"
	"testing"
	"time"
)

func TestStart_LaunchesBrowser(t *testing.T) {
	b, err := Start(LaunchOptions{Headless: true})
	if err != nil {
		t.Fatalf("failed to start browser: %v", err)
	}
	defer b.Close()

	if b.Port() != DefaultPort {
		t.Errorf("expected port %d, got %d", DefaultPort, b.Port())
	}

	if b.PID() == 0 {
		t.Error("expected non-zero PID")
	}
}

func TestBrowser_Targets(t *testing.T) {
	b, err := Start(LaunchOptions{Headless: true})
	if err != nil {
		t.Fatalf("failed to start browser: %v", err)
	}
	defer b.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	targets, err := b.Targets(ctx)
	if err != nil {
		t.Fatalf("failed to get targets: %v", err)
	}

	if len(targets) == 0 {
		t.Error("expected at least one target")
	}

	t.Logf("Found %d targets", len(targets))
	for _, target := range targets {
		t.Logf("  %s: %s (%s)", target.Type, target.Title, target.URL)
	}
}

func TestBrowser_PageTarget(t *testing.T) {
	b, err := Start(LaunchOptions{Headless: true})
	if err != nil {
		t.Fatalf("failed to start browser: %v", err)
	}
	defer b.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	target, err := b.PageTarget(ctx)
	if err != nil {
		t.Fatalf("failed to get page target: %v", err)
	}

	if target.Type != "page" {
		t.Errorf("expected page type, got %s", target.Type)
	}

	if target.WebSocketURL == "" {
		t.Error("expected non-empty WebSocket URL")
	}

	t.Logf("Page target: %s", target.WebSocketURL)
}

func TestBrowser_Version(t *testing.T) {
	b, err := Start(LaunchOptions{Headless: true})
	if err != nil {
		t.Fatalf("failed to start browser: %v", err)
	}
	defer b.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := b.Version(ctx)
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}

	if info.Browser == "" {
		t.Error("expected non-empty browser string")
	}

	t.Logf("Browser: %s", info.Browser)
	t.Logf("Protocol: %s", info.ProtocolVer)
}

func TestBrowser_WebSocketURL(t *testing.T) {
	b, err := Start(LaunchOptions{Headless: true})
	if err != nil {
		t.Fatalf("failed to start browser: %v", err)
	}
	defer b.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL, err := b.WebSocketURL(ctx)
	if err != nil {
		t.Fatalf("failed to get WebSocket URL: %v", err)
	}

	if wsURL == "" {
		t.Error("expected non-empty WebSocket URL")
	}

	t.Logf("WebSocket URL: %s", wsURL)
}

func TestBrowser_CustomPort(t *testing.T) {
	b, err := Start(LaunchOptions{Headless: true, Port: 9333})
	if err != nil {
		t.Fatalf("failed to start browser: %v", err)
	}
	defer b.Close()

	if b.Port() != 9333 {
		t.Errorf("expected port 9333, got %d", b.Port())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = b.Version(ctx)
	if err != nil {
		t.Fatalf("failed to get version on custom port: %v", err)
	}
}

func TestBrowser_Close(t *testing.T) {
	b, err := Start(LaunchOptions{Headless: true})
	if err != nil {
		t.Fatalf("failed to start browser: %v", err)
	}

	pid := b.PID()
	if pid == 0 {
		t.Fatal("expected non-zero PID before close")
	}

	err = b.Close()
	if err != nil {
		t.Errorf("unexpected error on close: %v", err)
	}

	// Double close should be safe
	err = b.Close()
	if err != nil {
		t.Errorf("unexpected error on double close: %v", err)
	}
}
