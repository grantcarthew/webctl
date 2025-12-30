package server

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewWatcher(t *testing.T) {
	tests := []struct {
		name    string
		config  WatcherConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: WatcherConfig{
				Paths: []string{"/tmp"},
			},
			wantErr: false,
		},
		{
			name:    "empty paths",
			config:  WatcherConfig{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewWatcher(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWatcher() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWatcherFileEvents(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Track events
	var mu sync.Mutex
	events := 0

	// Create watcher
	w, err := NewWatcher(WatcherConfig{
		Paths: []string{tmpDir},
		OnEvent: func(event FileEvent) {
			mu.Lock()
			events++
			mu.Unlock()
		},
	})
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	// Start watcher
	if err := w.Start(); err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}
	defer w.Stop()

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Create a file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Wait for debounced event (300ms debounce + buffer)
	time.Sleep(500 * time.Millisecond)

	// Check that we got at least one event
	mu.Lock()
	gotEvents := events
	mu.Unlock()

	if gotEvents == 0 {
		t.Error("Expected at least one file event")
	}
}

func TestWatcherIgnorePatterns(t *testing.T) {
	w := &Watcher{
		config: WatcherConfig{
			Ignore: []string{"*.tmp", "*.log"},
		},
	}

	tests := []struct {
		path   string
		ignore bool
	}{
		{"test.txt", false},
		{"test.tmp", true},
		{"test.log", true},
		{".hidden", true},
		{"node_modules/pkg/file.js", true},
		{"vendor/pkg/file.go", true},
		{"src/main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := w.shouldIgnore(tt.path)
			if got != tt.ignore {
				t.Errorf("shouldIgnore(%q) = %v, want %v", tt.path, got, tt.ignore)
			}
		})
	}
}

func TestDebouncer(t *testing.T) {
	callCount := 0
	var mu sync.Mutex

	d := newDebouncer(50*time.Millisecond, func() {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	// Trigger multiple times rapidly
	for i := 0; i < 10; i++ {
		d.trigger()
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce delay + buffer
	time.Sleep(100 * time.Millisecond)

	// Should only be called once due to debouncing
	mu.Lock()
	got := callCount
	mu.Unlock()

	if got != 1 {
		t.Errorf("Expected 1 callback, got %d", got)
	}

	// Stop debouncer
	d.stop()
}
