package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// Mode represents the server mode (static or proxy).
type Mode string

const (
	ModeStatic Mode = "static"
	ModeProxy  Mode = "proxy"
)

// Config holds server configuration.
type Config struct {
	Mode        Mode     // Server mode: static or proxy
	Directory   string   // Directory to serve (static mode)
	ProxyURL    string   // Backend URL to proxy (proxy mode)
	Port        int      // Server port (0 = auto-detect)
	Host        string   // Bind host ("localhost" or "0.0.0.0")
	WatchPaths  []string // Paths to watch for changes
	IgnorePaths []string // Glob patterns to ignore
	OnReload    func()   // Callback when files change (triggers reload)
	Debug       bool     // Enable debug logging
}

// Server is a development web server with hot reload capabilities.
type Server struct {
	config   Config
	httpSrv  *http.Server
	watcher  *Watcher
	listener net.Listener
	mu       sync.RWMutex
	running  bool
	debugLog func(format string, args ...any)
}

// New creates a new server with the given configuration.
func New(cfg Config) (*Server, error) {
	// Validate config
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	// Set defaults
	if cfg.Host == "" {
		cfg.Host = "localhost"
	}
	if len(cfg.WatchPaths) == 0 && cfg.Mode == ModeStatic {
		cfg.WatchPaths = []string{cfg.Directory}
	}

	s := &Server{
		config: cfg,
	}

	// Setup debug logger
	if cfg.Debug {
		s.debugLog = func(format string, args ...any) {
			log.Printf("[SERVER] "+format, args...)
		}
	} else {
		s.debugLog = func(format string, args ...any) {}
	}

	return s, nil
}

// validateConfig validates the server configuration.
func validateConfig(cfg Config) error {
	switch cfg.Mode {
	case ModeStatic:
		if cfg.Directory == "" {
			return fmt.Errorf("directory is required for static mode")
		}
	case ModeProxy:
		if cfg.ProxyURL == "" {
			return fmt.Errorf("proxy URL is required for proxy mode")
		}
	default:
		return fmt.Errorf("invalid mode: %s (must be 'static' or 'proxy')", cfg.Mode)
	}
	return nil
}

// Start starts the server and file watcher.
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	s.running = true
	s.mu.Unlock()

	// Find available port if needed
	port := s.config.Port
	if port == 0 {
		var err error
		port, err = findAvailablePort(s.config.Host)
		if err != nil {
			return fmt.Errorf("failed to find available port: %w", err)
		}
		s.debugLog("Auto-detected port: %d", port)
	}

	// Create listener
	addr := fmt.Sprintf("%s:%d", s.config.Host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener

	// Create HTTP handler based on mode
	var handler http.Handler
	switch s.config.Mode {
	case ModeStatic:
		handler = newStaticHandler(s.config.Directory, s.debugLog)
	case ModeProxy:
		handler, err = newProxyHandler(s.config.ProxyURL, s.debugLog)
		if err != nil {
			listener.Close()
			return fmt.Errorf("failed to create proxy handler: %w", err)
		}
	}

	// Create HTTP server
	s.httpSrv = &http.Server{
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start file watcher if watch paths are configured
	if len(s.config.WatchPaths) > 0 {
		watcher, err := NewWatcher(WatcherConfig{
			Paths:   s.config.WatchPaths,
			Ignore:  s.config.IgnorePaths,
			OnEvent: s.handleFileChange,
			Debug:   s.config.Debug,
		})
		if err != nil {
			listener.Close()
			return fmt.Errorf("failed to create file watcher: %w", err)
		}
		s.watcher = watcher

		if err := s.watcher.Start(); err != nil {
			listener.Close()
			return fmt.Errorf("failed to start file watcher: %w", err)
		}
		s.debugLog("File watcher started for paths: %v", s.config.WatchPaths)
	}

	// Start HTTP server in background
	go func() {
		s.debugLog("HTTP server started on http://%s", addr)
		if err := s.httpSrv.Serve(listener); err != nil && err != http.ErrServerClosed {
			s.debugLog("HTTP server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the server and file watcher.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	s.mu.Unlock()

	var errs []error

	// Stop file watcher
	if s.watcher != nil {
		if err := s.watcher.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop watcher: %w", err))
		}
	}

	// Stop HTTP server
	if s.httpSrv != nil {
		if err := s.httpSrv.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown HTTP server: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("stop errors: %v", errs)
	}

	s.debugLog("Server stopped")
	return nil
}

// Addr returns the server's listening address.
func (s *Server) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Port returns the server's listening port.
func (s *Server) Port() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.listener == nil {
		return 0
	}
	addr := s.listener.Addr().(*net.TCPAddr)
	return addr.Port
}

// URL returns the server's full URL.
func (s *Server) URL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.listener == nil {
		return ""
	}

	addr := s.listener.Addr().(*net.TCPAddr)
	host := addr.IP.String()
	if host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return fmt.Sprintf("http://%s:%d", host, addr.Port)
}

// IsRunning returns true if the server is running.
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// handleFileChange is called when a file change is detected.
// It debounces events and triggers the reload callback.
func (s *Server) handleFileChange(event FileEvent) {
	s.debugLog("File changed: %s (%s)", event.Path, event.Op)

	if s.config.OnReload != nil {
		s.config.OnReload()
	}
}

// findAvailablePort finds an available port on the given host.
func findAvailablePort(host string) (int, error) {
	// Try common dev ports first
	commonPorts := []int{3000, 8080, 8000, 5000, 4000}

	for _, port := range commonPorts {
		if isPortAvailable(host, port) {
			return port, nil
		}
	}

	// Fall back to OS-assigned port
	addr := fmt.Sprintf("%s:0", host)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	return port, nil
}

// isPortAvailable checks if a port is available for binding.
func isPortAvailable(host string, port int) bool {
	addr := fmt.Sprintf("%s:%d", host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}
