package server

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// newProxyHandler creates an HTTP handler for proxying requests to a backend server.
func newProxyHandler(targetURL string, debugLog func(format string, args ...any)) (http.Handler, error) {
	// Auto-add http:// if no scheme is present
	// This allows "localhost:3000" to work as well as "http://localhost:3000"
	if len(targetURL) > 0 && targetURL[0] != 'h' {
		// Check if it doesn't start with http:// or https://
		if len(targetURL) < 7 || (targetURL[:7] != "http://" && (len(targetURL) < 8 || targetURL[:8] != "https://")) {
			targetURL = "http://" + targetURL
		}
	}

	// Parse target URL
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}

	// Validate target URL
	if target.Scheme != "http" && target.Scheme != "https" {
		return nil, fmt.Errorf("invalid proxy URL scheme: %s (must be http or https)", target.Scheme)
	}
	if target.Host == "" {
		return nil, fmt.Errorf("invalid proxy URL: missing host")
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Customize director to preserve host header option
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Preserve original host in X-Forwarded-Host
		req.Header.Set("X-Forwarded-Host", req.Host)
		// Set target host
		req.Host = target.Host
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
	}

	// Add error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		debugLog("Proxy error for %s: %v", r.URL.Path, err)
		http.Error(w, fmt.Sprintf("Proxy Error: %v", err), http.StatusBadGateway)
	}

	handler := &proxyHandler{
		proxy:    proxy,
		target:   target,
		debugLog: debugLog,
	}

	return handler, nil
}

// proxyHandler wraps the reverse proxy with logging.
type proxyHandler struct {
	proxy    *httputil.ReverseProxy
	target   *url.URL
	debugLog func(format string, args ...any)
}

// ServeHTTP implements http.Handler for reverse proxy.
func (h *proxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Log request
	h.debugLog("%s %s -> %s%s", r.Method, r.URL.Path, h.target.Host, r.URL.Path)

	// Disable caching for development
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Proxy the request
	h.proxy.ServeHTTP(w, r)

	// Log response
	duration := time.Since(start)
	h.debugLog("Proxied: %s (%v)", r.URL.Path, duration)
}
