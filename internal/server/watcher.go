package server

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileEvent represents a file system event.
type FileEvent struct {
	Path string
	Op   string
	Time time.Time
}

// WatcherConfig holds file watcher configuration.
type WatcherConfig struct {
	Paths   []string                 // Paths to watch (files or directories)
	Ignore  []string                 // Glob patterns to ignore
	OnEvent func(event FileEvent)   // Callback for file events
	Debug   bool                     // Enable debug logging
}

// Watcher watches files for changes and triggers callbacks.
type Watcher struct {
	config     WatcherConfig
	fsWatcher  *fsnotify.Watcher
	debouncer  *debouncer
	mu         sync.Mutex
	running    bool
	done       chan struct{}
	debugLog   func(format string, args ...any)
}

// NewWatcher creates a new file watcher.
func NewWatcher(cfg WatcherConfig) (*Watcher, error) {
	if len(cfg.Paths) == 0 {
		return nil, fmt.Errorf("at least one path is required")
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	w := &Watcher{
		config:    cfg,
		fsWatcher: fsWatcher,
		done:      make(chan struct{}),
	}

	// Setup debug logger
	if cfg.Debug {
		w.debugLog = func(format string, args ...any) {
			log.Printf("[WATCHER] "+format, args...)
		}
	} else {
		w.debugLog = func(format string, args ...any) {}
	}

	// Create debouncer with 300ms delay (balances responsiveness with avoiding duplicate reloads)
	w.debouncer = newDebouncer(300*time.Millisecond, func() {
		if cfg.OnEvent != nil {
			cfg.OnEvent(FileEvent{
				Path: "debounced",
				Op:   "change",
				Time: time.Now(),
			})
		}
	})

	return w, nil
}

// Start starts watching the configured paths.
func (w *Watcher) Start() error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("watcher already running")
	}
	w.running = true
	w.mu.Unlock()

	// Add watch paths
	for _, path := range w.config.Paths {
		if err := w.addPath(path); err != nil {
			return fmt.Errorf("failed to add watch path %s: %w", path, err)
		}
	}

	// Start event loop
	go w.eventLoop()

	return nil
}

// Stop stops the file watcher.
func (w *Watcher) Stop() error {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = false
	w.mu.Unlock()

	// Stop debouncer
	if w.debouncer != nil {
		w.debouncer.stop()
	}

	// Close fsnotify watcher
	if w.fsWatcher != nil {
		if err := w.fsWatcher.Close(); err != nil {
			return err
		}
	}

	// Wait for event loop to finish
	<-w.done

	return nil
}

// addPath adds a path (file or directory) to the watcher.
// For directories, recursively adds all subdirectories.
func (w *Watcher) addPath(path string) error {
	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}

	if info.IsDir() {
		// Walk directory tree and add all directories
		return filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip ignored paths
			if w.shouldIgnore(path) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Only watch directories (fsnotify watches files in those directories)
			if info.IsDir() {
				if err := w.fsWatcher.Add(path); err != nil {
					return err
				}
				w.debugLog("Watching directory: %s", path)
			}

			return nil
		})
	}

	// Single file - watch its directory
	dir := filepath.Dir(absPath)
	if err := w.fsWatcher.Add(dir); err != nil {
		return err
	}
	w.debugLog("Watching file: %s (via directory: %s)", absPath, dir)

	return nil
}

// shouldIgnore checks if a path should be ignored based on glob patterns.
func (w *Watcher) shouldIgnore(path string) bool {
	// Always ignore hidden files (files starting with .)
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".") {
		return true
	}

	// Check if path contains common directories to ignore
	pathParts := strings.Split(filepath.Clean(path), string(filepath.Separator))
	for _, part := range pathParts {
		if part == "node_modules" || part == "vendor" || part == "__pycache__" {
			return true
		}
	}

	// Check user-defined ignore patterns
	for _, pattern := range w.config.Ignore {
		matched, err := filepath.Match(pattern, base)
		if err == nil && matched {
			return true
		}

		// Also try matching the full path
		matched, err = filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
	}

	return false
}

// eventLoop processes file system events.
func (w *Watcher) eventLoop() {
	defer close(w.done)

	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			// Skip ignored paths
			if w.shouldIgnore(event.Name) {
				continue
			}

			// Log event
			w.debugLog("Event: %s %s", event.Op, event.Name)

			// Handle directory creation (add new directories to watch)
			if event.Op&fsnotify.Create == fsnotify.Create {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if err := w.addPath(event.Name); err != nil {
						w.debugLog("Failed to add new directory: %v", err)
					}
				}
			}

			// Trigger debounced reload
			w.debouncer.trigger()

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			w.debugLog("Watcher error: %v", err)
		}
	}
}

// debouncer debounces events to prevent excessive triggers.
type debouncer struct {
	delay    time.Duration
	callback func()
	timer    *time.Timer
	mu       sync.Mutex
}

// newDebouncer creates a new debouncer.
func newDebouncer(delay time.Duration, callback func()) *debouncer {
	return &debouncer{
		delay:    delay,
		callback: callback,
	}
}

// trigger triggers the debouncer, resetting the delay timer.
func (d *debouncer) trigger() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.delay, func() {
		d.callback()
	})
}

// stop stops the debouncer.
func (d *debouncer) stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
}
