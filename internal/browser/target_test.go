package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchTargets_ParsesResponse(t *testing.T) {
	t.Parallel()

	targets := []Target{
		{
			ID:           "ABC123",
			Type:         "page",
			Title:        "Test Page",
			URL:          "https://example.com",
			WebSocketURL: "ws://127.0.0.1:9222/devtools/page/ABC123",
		},
		{
			ID:   "DEF456",
			Type: "background_page",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(targets)
	}))
	defer server.Close()

	// Extract host and port from server URL
	addr := strings.TrimPrefix(server.URL, "http://")
	parts := strings.Split(addr, ":")
	host := parts[0]
	var port int
	if len(parts) > 1 {
		_, _ = fmt.Sscanf(parts[1], "%d", &port)
	}

	result, err := FetchTargets(context.Background(), host, port)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 targets, got %d", len(result))
	}

	if result[0].ID != "ABC123" {
		t.Errorf("expected ID ABC123, got %s", result[0].ID)
	}

	if result[0].WebSocketURL != "ws://127.0.0.1:9222/devtools/page/ABC123" {
		t.Errorf("unexpected WebSocket URL: %s", result[0].WebSocketURL)
	}
}

func TestFetchTargets_HandlesError(t *testing.T) {
	t.Parallel()

	_, err := FetchTargets(context.Background(), "127.0.0.1", 59999)
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
}

func TestFetchVersion_ParsesResponse(t *testing.T) {
	t.Parallel()

	info := VersionInfo{
		Browser:      "Chrome/120.0.0.0",
		ProtocolVer:  "1.3",
		UserAgent:    "Mozilla/5.0",
		WebSocketURL: "ws://127.0.0.1:9222/devtools/browser/abc",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/version" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(info)
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	parts := strings.Split(addr, ":")
	host := parts[0]
	var port int
	if len(parts) > 1 {
		_, _ = fmt.Sscanf(parts[1], "%d", &port)
	}

	result, err := FetchVersion(context.Background(), host, port)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Browser != "Chrome/120.0.0.0" {
		t.Errorf("expected Chrome/120.0.0.0, got %s", result.Browser)
	}
}

func TestFindPageTarget_ReturnsFirstPage(t *testing.T) {
	t.Parallel()

	targets := []Target{
		{ID: "1", Type: "background_page"},
		{ID: "2", Type: "page", Title: "First Page"},
		{ID: "3", Type: "page", Title: "Second Page"},
	}

	target := FindPageTarget(targets)
	if target == nil {
		t.Fatal("expected to find a page target")
	}

	if target.ID != "2" {
		t.Errorf("expected ID 2, got %s", target.ID)
	}
}

func TestFindPageTarget_ReturnsNilWhenNoPage(t *testing.T) {
	t.Parallel()

	targets := []Target{
		{ID: "1", Type: "background_page"},
		{ID: "2", Type: "service_worker"},
	}

	target := FindPageTarget(targets)
	if target != nil {
		t.Error("expected nil for no page targets")
	}
}

func TestFindPageTarget_EmptyList(t *testing.T) {
	t.Parallel()

	target := FindPageTarget(nil)
	if target != nil {
		t.Error("expected nil for empty list")
	}
}
