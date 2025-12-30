package server

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// newStaticHandler creates an HTTP handler for serving static files.
func newStaticHandler(directory string, debugLog func(format string, args ...any)) http.Handler {
	// Resolve absolute path
	absDir, err := filepath.Abs(directory)
	if err != nil {
		log.Printf("failed to resolve directory path: %v", err)
		absDir = directory
	}

	// Check if directory exists
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		log.Printf("warning: directory does not exist: %s", absDir)
	}

	handler := &staticHandler{
		root:     absDir,
		debugLog: debugLog,
	}

	return handler
}

// staticHandler serves static files from a directory.
type staticHandler struct {
	root     string
	debugLog func(format string, args ...any)
}

// ServeHTTP implements http.Handler for static file serving.
func (h *staticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Log request
	h.debugLog("%s %s", r.Method, r.URL.Path)

	// Only allow GET and HEAD
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Clean path to prevent directory traversal
	path := filepath.Clean(r.URL.Path)

	// Remove leading slash for filepath.Join
	path = strings.TrimPrefix(path, "/")

	// Build full file path
	fullPath := filepath.Join(h.root, path)

	// Security check: ensure path is within root
	if !strings.HasPrefix(fullPath, h.root) {
		h.debugLog("403 Forbidden: path outside root: %s", fullPath)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Check if file exists
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			h.debugLog("404 Not Found: %s", r.URL.Path)
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		h.debugLog("500 Internal Server Error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// If path is a directory, try common index files
	if fileInfo.IsDir() {
		// Try common index files in order
		indexFiles := []string{"index.html", "index.htm", "default.html", "home.html"}
		found := false

		for _, indexFile := range indexFiles {
			indexPath := filepath.Join(fullPath, indexFile)
			if _, err := os.Stat(indexPath); err == nil {
				fullPath = indexPath
				found = true
				h.debugLog("Serving directory index: %s", indexFile)
				break
			}
		}

		if !found {
			// Directory listing disabled - return 404
			h.debugLog("404 Not Found: directory without index file (tried: %v): %s", indexFiles, r.URL.Path)
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
	}

	// Disable caching for development
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Serve the file
	http.ServeFile(w, r, fullPath)

	// Log response
	duration := time.Since(start)
	h.debugLog("200 OK: %s (%v)", r.URL.Path, duration)
}
