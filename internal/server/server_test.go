package server

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid static config",
			config: Config{
				Mode:      ModeStatic,
				Directory: "/tmp",
			},
			wantErr: false,
		},
		{
			name: "valid proxy config",
			config: Config{
				Mode:     ModeProxy,
				ProxyURL: "http://localhost:8080",
			},
			wantErr: false,
		},
		{
			name: "invalid mode",
			config: Config{
				Mode: "invalid",
			},
			wantErr: true,
		},
		{
			name: "static mode missing directory",
			config: Config{
				Mode: ModeStatic,
			},
			wantErr: true,
		},
		{
			name: "proxy mode missing URL",
			config: Config{
				Mode: ModeProxy,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerStartStop(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a test index.html
	indexHTML := filepath.Join(tmpDir, "index.html")
	if err := os.WriteFile(indexHTML, []byte("<html><body>Hello from index.html</body></html>"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create server
	srv, err := New(Config{
		Mode:      ModeStatic,
		Directory: tmpDir,
		Port:      0, // Auto-select port
		Host:      "localhost",
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	ctx := context.Background()
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Check that server is running
	if !srv.IsRunning() {
		t.Error("Server should be running")
	}

	// Check that we have a valid URL
	url := srv.URL()
	if url == "" {
		t.Error("Server URL should not be empty")
	}

	// Check that we have a valid port
	port := srv.Port()
	if port == 0 {
		t.Error("Server port should not be 0")
	}

	// Test HTTP request
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	expected := "<html><body>Hello from index.html</body></html>"
	if string(body) != expected {
		t.Errorf("Expected body %q, got %q", expected, string(body))
	}

	// Stop server
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Stop(stopCtx); err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}

	// Check that server is not running
	if srv.IsRunning() {
		t.Error("Server should not be running")
	}
}

func TestIndexFileFallback(t *testing.T) {
	tests := []struct {
		name         string
		files        map[string]string
		expectedBody string
	}{
		{
			name: "index.html",
			files: map[string]string{
				"index.html": "<html><body>index.html</body></html>",
			},
			expectedBody: "<html><body>index.html</body></html>",
		},
		{
			name: "index.htm fallback",
			files: map[string]string{
				"index.htm": "<html><body>index.htm</body></html>",
			},
			expectedBody: "<html><body>index.htm</body></html>",
		},
		{
			name: "default.html fallback",
			files: map[string]string{
				"default.html": "<html><body>default.html</body></html>",
			},
			expectedBody: "<html><body>default.html</body></html>",
		},
		{
			name: "home.html fallback",
			files: map[string]string{
				"home.html": "<html><body>home.html</body></html>",
			},
			expectedBody: "<html><body>home.html</body></html>",
		},
		{
			name: "index.html takes precedence",
			files: map[string]string{
				"index.html":   "<html><body>index.html</body></html>",
				"index.htm":    "<html><body>index.htm</body></html>",
				"default.html": "<html><body>default.html</body></html>",
			},
			expectedBody: "<html><body>index.html</body></html>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create test files
			for filename, content := range tt.files {
				filePath := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			// Create and start server
			srv, err := New(Config{
				Mode:      ModeStatic,
				Directory: tmpDir,
				Port:      0,
			})
			if err != nil {
				t.Fatalf("Failed to create server: %v", err)
			}

			ctx := context.Background()
			if err := srv.Start(ctx); err != nil {
				t.Fatalf("Failed to start server: %v", err)
			}
			defer func() { _ = srv.Stop(ctx) }()

			// Test root path
			resp, err := http.Get(srv.URL() + "/")
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read body: %v", err)
			}

			if string(body) != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, string(body))
			}
		})
	}
}

func TestPortAutoDetection(t *testing.T) {
	tests := []struct {
		name string
		host string
	}{
		{"localhost", "localhost"},
		{"127.0.0.1", "127.0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, err := findAvailablePort(tt.host)
			if err != nil {
				t.Fatalf("findAvailablePort() error = %v", err)
			}

			if port == 0 {
				t.Error("Port should not be 0")
			}

			// Verify port is available
			if !isPortAvailable(tt.host, port) {
				t.Errorf("Port %d should be available", port)
			}
		})
	}
}
