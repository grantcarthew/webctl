package daemon

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/grantcarthew/webctl/internal/browser"
	"github.com/grantcarthew/webctl/internal/cdp"
	"github.com/grantcarthew/webctl/internal/ipc"
)

// DefaultBufferSize is the default capacity for event buffers.
const DefaultBufferSize = 10000

// Config holds daemon configuration.
type Config struct {
	Headless   bool
	Port       int
	SocketPath string
	PIDPath    string
	BufferSize int
	Debug      bool
	// CommandExecutor is called by REPL for CLI command execution with flags.
	// If nil, REPL falls back to basic IPC-only execution.
	CommandExecutor ipc.CommandExecutor
}

// DefaultConfig returns the default daemon configuration.
func DefaultConfig() Config {
	return Config{
		Headless:   false,
		Port:       9222,
		SocketPath: ipc.DefaultSocketPath(),
		PIDPath:    ipc.DefaultPIDPath(),
		BufferSize: DefaultBufferSize,
	}
}

// Daemon is the persistent webctl daemon process.
type Daemon struct {
	config       Config
	browser      *browser.Browser
	cdp          *cdp.Client
	sessions     *SessionManager
	consoleBuf   *RingBuffer[ipc.ConsoleEntry]
	networkBuf   *RingBuffer[ipc.NetworkEntry]
	server       *ipc.Server
	shutdown     chan struct{}
	shutdownOnce sync.Once
	debug        bool

	// Navigation event waiting
	navWaiters sync.Map // map[string]chan *frameNavigatedInfo for sessionID -> waiter
	loadWaiters sync.Map // map[string]chan struct{} for sessionID -> waiter (loadEventFired)
}

// frameNavigatedInfo contains information from a Page.frameNavigated event.
type frameNavigatedInfo struct {
	URL   string
	Title string
}

// debugf logs a debug message if debug mode is enabled.
func (d *Daemon) debugf(format string, args ...any) {
	if d.debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// New creates a new daemon with the given configuration.
func New(cfg Config) *Daemon {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = DefaultBufferSize
	}

	return &Daemon{
		config:     cfg,
		sessions:   NewSessionManager(),
		consoleBuf: NewRingBuffer[ipc.ConsoleEntry](cfg.BufferSize),
		networkBuf: NewRingBuffer[ipc.NetworkEntry](cfg.BufferSize),
		shutdown:   make(chan struct{}),
		debug:      cfg.Debug,
	}
}

// Handler returns the IPC request handler function.
// Used by the CLI to create a direct executor for REPL command execution.
func (d *Daemon) Handler() ipc.Handler {
	return d.handleRequest
}

// Run starts the daemon and blocks until shutdown.
func (d *Daemon) Run(ctx context.Context) error {
	// Write PID file
	if err := d.writePIDFile(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}
	defer d.removePIDFile()

	// Start browser
	b, err := browser.Start(browser.LaunchOptions{
		Port:     d.config.Port,
		Headless: d.config.Headless,
	})
	if err != nil {
		return fmt.Errorf("failed to start browser: %w", err)
	}
	d.browser = b
	defer d.browser.Close()

	// Connect to browser-level CDP WebSocket (not page target)
	// This allows us to use Target.setAutoAttach for session management
	version, err := d.browser.Version(ctx)
	if err != nil {
		return fmt.Errorf("failed to get browser version: %w", err)
	}

	cdpClient, err := cdp.Dial(ctx, version.WebSocketURL)
	if err != nil {
		return fmt.Errorf("failed to connect to CDP: %w", err)
	}
	d.cdp = cdpClient
	defer d.cdp.Close()

	// Subscribe to events before enabling domains
	d.subscribeEvents()

	// Enable auto-attach for session tracking
	if err := d.enableAutoAttach(); err != nil {
		return fmt.Errorf("failed to enable auto-attach: %w", err)
	}

	// Start IPC server
	server, err := ipc.NewServer(d.config.SocketPath, d.handleRequest)
	if err != nil {
		return fmt.Errorf("failed to start IPC server: %w", err)
	}
	d.server = server
	defer d.server.Close()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Run server in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- d.server.Serve(ctx)
	}()

	// Start REPL if stdin is a TTY.
	// replDone is only closed when REPL exits; if stdin is not a TTY,
	// it stays open so the select below doesn't trigger early exit.
	replDone := make(chan struct{})
	if IsStdinTTY() {
		repl := NewREPL(d.handleRequest, d.config.CommandExecutor, func() { close(d.shutdown) })
		repl.SetSessionProvider(func() (*ipc.PageSession, int) {
			return d.sessions.Active(), d.sessions.Count()
		})
		go func() {
			defer close(replDone)
			repl.Run()
		}()
	}
	// When stdin is not a TTY, replDone remains open - daemon waits for
	// context cancellation, signal, shutdown command, or server error.

	// Wait for shutdown
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-sigCh:
		return nil
	case <-d.shutdown:
		return nil
	case err := <-errCh:
		return err
	case <-replDone:
		// REPL exited (EOF or error)
		return nil
	}
}

// enableAutoAttach enables Target.setAutoAttach for session tracking.
// This is called on the browser-level connection and automatically attaches
// to all page targets, enabling domains for each.
func (d *Daemon) enableAutoAttach() error {
	// Enable target discovery to receive targetInfoChanged events
	_, err := d.cdp.Send("Target.setDiscoverTargets", map[string]any{
		"discover": true,
	})
	if err != nil {
		return fmt.Errorf("failed to set discover targets: %w", err)
	}

	_, err = d.cdp.Send("Target.setAutoAttach", map[string]any{
		"autoAttach":             true,
		"flatten":                true,
		"waitForDebuggerOnStart": true,
	})
	if err != nil {
		return fmt.Errorf("failed to set auto-attach: %w", err)
	}
	return nil
}

// enableDomainsForSession enables CDP domains for a specific session.
func (d *Daemon) enableDomainsForSession(sessionID string) error {
	domains := []string{"Runtime.enable", "Network.enable", "Page.enable", "DOM.enable"}
	for _, method := range domains {
		if _, err := d.cdp.SendToSession(context.Background(), sessionID, method, nil); err != nil {
			return fmt.Errorf("failed to enable %s: %w", method, err)
		}
	}

	// Enable lifecycle events (required to receive Page.lifecycleEvent)
	if _, err := d.cdp.SendToSession(context.Background(), sessionID, "Page.setLifecycleEventsEnabled", map[string]any{"enabled": true}); err != nil {
		return fmt.Errorf("failed to enable lifecycle events: %w", err)
	}

	// Resume the target (it's paused due to waitForDebuggerOnStart)
	if _, err := d.cdp.SendToSession(context.Background(), sessionID, "Runtime.runIfWaitingForDebugger", nil); err != nil {
		return fmt.Errorf("failed to resume debugger: %w", err)
	}

	return nil
}

// subscribeEvents subscribes to CDP events and buffers them.
func (d *Daemon) subscribeEvents() {
	// Target events (browser-level, no sessionId)
	d.cdp.Subscribe("Target.attachedToTarget", func(evt cdp.Event) {
		d.handleTargetAttached(evt)
	})

	d.cdp.Subscribe("Target.detachedFromTarget", func(evt cdp.Event) {
		d.handleTargetDetached(evt)
	})

	d.cdp.Subscribe("Target.targetInfoChanged", func(evt cdp.Event) {
		d.handleTargetInfoChanged(evt)
	})

	// Console events (include sessionId)
	d.cdp.Subscribe("Runtime.consoleAPICalled", func(evt cdp.Event) {
		if entry, ok := d.parseConsoleEvent(evt); ok {
			entry.SessionID = evt.SessionID
			d.consoleBuf.Push(entry)
		}
	})

	d.cdp.Subscribe("Runtime.exceptionThrown", func(evt cdp.Event) {
		if entry, ok := d.parseExceptionEvent(evt); ok {
			entry.SessionID = evt.SessionID
			d.consoleBuf.Push(entry)
		}
	})

	// Network events (include sessionId)
	d.cdp.Subscribe("Network.requestWillBeSent", func(evt cdp.Event) {
		if entry, ok := d.parseRequestEvent(evt); ok {
			entry.SessionID = evt.SessionID
			d.networkBuf.Push(entry)
		}
	})

	d.cdp.Subscribe("Network.responseReceived", func(evt cdp.Event) {
		d.updateResponseEvent(evt)
	})

	d.cdp.Subscribe("Network.loadingFinished", func(evt cdp.Event) {
		d.handleLoadingFinished(evt)
	})

	d.cdp.Subscribe("Network.loadingFailed", func(evt cdp.Event) {
		d.handleLoadingFailed(evt)
	})

	// Page navigation events for navigation commands
	d.cdp.Subscribe("Page.frameNavigated", func(evt cdp.Event) {
		d.handleFrameNavigated(evt)
	})

	d.cdp.Subscribe("Page.loadEventFired", func(evt cdp.Event) {
		d.handleLoadEventFired(evt)
	})
}

// parseConsoleEvent parses a Runtime.consoleAPICalled event.
// Returns the entry and true on success, or zero value and false on parse error.
func (d *Daemon) parseConsoleEvent(evt cdp.Event) (ipc.ConsoleEntry, bool) {
	var params struct {
		Type      string `json:"type"`
		Timestamp float64 `json:"timestamp"`
		Args      []struct {
			Type  string `json:"type"`
			Value any    `json:"value"`
		} `json:"args"`
		StackTrace *struct {
			CallFrames []struct {
				URL        string `json:"url"`
				LineNumber int    `json:"lineNumber"`
				ColumnNumber int  `json:"columnNumber"`
			} `json:"callFrames"`
		} `json:"stackTrace"`
	}
	if err := json.Unmarshal(evt.Params, &params); err != nil {
		return ipc.ConsoleEntry{}, false
	}

	entry := ipc.ConsoleEntry{
		Type:      params.Type,
		Timestamp: int64(params.Timestamp),
	}

	// Extract text from args
	var args []string
	for _, arg := range params.Args {
		if s, ok := arg.Value.(string); ok {
			args = append(args, s)
		} else {
			data, _ := json.Marshal(arg.Value)
			args = append(args, string(data))
		}
	}
	if len(args) > 0 {
		entry.Text = args[0]
		entry.Args = args
	}

	// Extract stack trace info
	if params.StackTrace != nil && len(params.StackTrace.CallFrames) > 0 {
		frame := params.StackTrace.CallFrames[0]
		entry.URL = frame.URL
		entry.Line = frame.LineNumber
		entry.Column = frame.ColumnNumber
	}

	return entry, true
}

// parseExceptionEvent parses a Runtime.exceptionThrown event.
// Returns the entry and true on success, or zero value and false on parse error.
func (d *Daemon) parseExceptionEvent(evt cdp.Event) (ipc.ConsoleEntry, bool) {
	var params struct {
		Timestamp float64 `json:"timestamp"`
		ExceptionDetails struct {
			Text      string `json:"text"`
			URL       string `json:"url"`
			Line      int    `json:"lineNumber"`
			Column    int    `json:"columnNumber"`
			Exception *struct {
				Description string `json:"description"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(evt.Params, &params); err != nil {
		return ipc.ConsoleEntry{}, false
	}

	text := params.ExceptionDetails.Text
	if params.ExceptionDetails.Exception != nil && params.ExceptionDetails.Exception.Description != "" {
		text = params.ExceptionDetails.Exception.Description
	}

	return ipc.ConsoleEntry{
		Type:      "error",
		Text:      text,
		Timestamp: int64(params.Timestamp),
		URL:       params.ExceptionDetails.URL,
		Line:      params.ExceptionDetails.Line,
		Column:    params.ExceptionDetails.Column,
	}, true
}

// parseRequestEvent parses a Network.requestWillBeSent event.
// Returns the entry and true on success, or zero value and false on parse error.
func (d *Daemon) parseRequestEvent(evt cdp.Event) (ipc.NetworkEntry, bool) {
	var params struct {
		RequestID string  `json:"requestId"`
		WallTime  float64 `json:"wallTime"` // Unix epoch in seconds
		Request   struct {
			URL     string            `json:"url"`
			Method  string            `json:"method"`
			Headers map[string]string `json:"headers"`
		} `json:"request"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(evt.Params, &params); err != nil {
		return ipc.NetworkEntry{}, false
	}

	return ipc.NetworkEntry{
		RequestID:      params.RequestID,
		URL:            params.Request.URL,
		Method:         params.Request.Method,
		Type:           params.Type,
		RequestTime:    int64(params.WallTime * 1000), // Convert seconds to milliseconds
		RequestHeaders: params.Request.Headers,
	}, true
}

// updateResponseEvent updates an existing network entry with response data.
func (d *Daemon) updateResponseEvent(evt cdp.Event) {
	var params struct {
		RequestID string `json:"requestId"`
		Response  struct {
			Status     int               `json:"status"`
			StatusText string            `json:"statusText"`
			MimeType   string            `json:"mimeType"`
			Headers    map[string]string `json:"headers"`
		} `json:"response"`
	}
	if err := json.Unmarshal(evt.Params, &params); err != nil {
		return
	}

	// Use current wall time for response timestamp since CDP's Network.responseReceived
	// only provides monotonic timestamp, not wallTime. This is accurate because events
	// are processed in real-time.
	responseTime := time.Now().UnixMilli()

	// Find and update the matching entry in-place.
	// Iterates newest-to-oldest; responses typically arrive shortly after requests,
	// so the match is usually found within the first few items despite O(n) worst case.
	d.networkBuf.Update(func(entry *ipc.NetworkEntry) bool {
		if entry.RequestID == params.RequestID {
			entry.Status = params.Response.Status
			entry.StatusText = params.Response.StatusText
			entry.MimeType = params.Response.MimeType
			entry.ResponseHeaders = params.Response.Headers
			entry.ResponseTime = responseTime
			if entry.RequestTime > 0 {
				entry.Duration = float64(entry.ResponseTime-entry.RequestTime) / 1000.0
			}
			return true // stop iteration
		}
		return false
	})
}

// handleLoadingFinished handles the Network.loadingFinished event.
// Fetches response body and stores it (as text or file for binary).
func (d *Daemon) handleLoadingFinished(evt cdp.Event) {
	var params struct {
		RequestID         string  `json:"requestId"`
		EncodedDataLength int64   `json:"encodedDataLength"`
	}
	if err := json.Unmarshal(evt.Params, &params); err != nil {
		return
	}

	// Find the entry to get MIME type
	var mimeType string
	var entryURL string
	d.networkBuf.Update(func(entry *ipc.NetworkEntry) bool {
		if entry.RequestID == params.RequestID {
			mimeType = entry.MimeType
			entryURL = entry.URL
			entry.Size = params.EncodedDataLength
			return true
		}
		return false
	})

	// Fetch the response body using the session ID from the event
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := d.cdp.SendToSession(ctx, evt.SessionID, "Network.getResponseBody", map[string]any{
		"requestId": params.RequestID,
	})
	if err != nil {
		// Body may not be available (e.g., redirects, cached responses)
		return
	}

	var bodyResp struct {
		Body          string `json:"body"`
		Base64Encoded bool   `json:"base64Encoded"`
	}
	if err := json.Unmarshal(result, &bodyResp); err != nil {
		return
	}

	// Update the entry with body data
	d.networkBuf.Update(func(entry *ipc.NetworkEntry) bool {
		if entry.RequestID == params.RequestID {
			if isBinaryMimeType(mimeType) {
				// Save binary to file
				bodyPath, err := saveBinaryBody(params.RequestID, entryURL, mimeType, bodyResp.Body, bodyResp.Base64Encoded)
				if err == nil {
					entry.BodyPath = bodyPath
				}
			} else {
				// Store text body directly
				if bodyResp.Base64Encoded {
					// Decode base64 for text content
					decoded, err := base64.StdEncoding.DecodeString(bodyResp.Body)
					if err == nil {
						entry.Body = string(decoded)
					}
				} else {
					entry.Body = bodyResp.Body
				}
			}
			return true
		}
		return false
	})
}

// handleLoadingFailed handles the Network.loadingFailed event.
// Marks the request as failed with error details.
func (d *Daemon) handleLoadingFailed(evt cdp.Event) {
	var params struct {
		RequestID    string  `json:"requestId"`
		ErrorText    string  `json:"errorText"`
		Canceled     bool    `json:"canceled"`
	}
	if err := json.Unmarshal(evt.Params, &params); err != nil {
		return
	}

	failTime := time.Now().UnixMilli()

	d.networkBuf.Update(func(entry *ipc.NetworkEntry) bool {
		if entry.RequestID == params.RequestID {
			entry.Failed = true
			if params.Canceled {
				entry.Error = "canceled"
			} else {
				entry.Error = params.ErrorText
			}
			entry.ResponseTime = failTime
			if entry.RequestTime > 0 {
				entry.Duration = float64(entry.ResponseTime-entry.RequestTime) / 1000.0
			}
			return true
		}
		return false
	})
}

// handleTargetAttached handles Target.attachedToTarget event.
// Adds the new session to tracking and enables CDP domains.
func (d *Daemon) handleTargetAttached(evt cdp.Event) {
	var params struct {
		SessionID  string `json:"sessionId"`
		TargetInfo struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
			Title    string `json:"title"`
			URL      string `json:"url"`
		} `json:"targetInfo"`
	}
	if err := json.Unmarshal(evt.Params, &params); err != nil {
		return
	}

	// Only track page targets
	if params.TargetInfo.Type != "page" {
		return
	}

	d.debugf("Target.attachedToTarget: sessionID=%q, targetID=%q, url=%q",
		params.SessionID, params.TargetInfo.TargetID, params.TargetInfo.URL)

	// Add to session manager
	d.sessions.Add(
		params.SessionID,
		params.TargetInfo.TargetID,
		params.TargetInfo.URL,
		params.TargetInfo.Title,
	)

	// Enable domains for this session (async to not block event loop)
	go func() {
		startEnable := time.Now()
		if err := d.enableDomainsForSession(params.SessionID); err != nil {
			// Log error but don't fail - session is still tracked
			fmt.Fprintf(os.Stderr, "warning: failed to enable domains for session: %v\n", err)
		}
		d.debugf("enableDomainsForSession completed in %v for session %q", time.Since(startEnable), params.SessionID)
	}()
}

// handleTargetDetached handles Target.detachedFromTarget event.
// Removes the session and purges its buffer entries.
func (d *Daemon) handleTargetDetached(evt cdp.Event) {
	var params struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(evt.Params, &params); err != nil {
		return
	}

	d.debugf("Target.detachedFromTarget: sessionID=%q", params.SessionID)

	// Remove from session manager
	newActive, changed := d.sessions.Remove(params.SessionID)
	d.debugf("Session removed: newActiveID=%q, activeChanged=%v", newActive, changed)

	// Purge entries for this session
	d.purgeSessionEntries(params.SessionID)
}

// handleTargetInfoChanged handles Target.targetInfoChanged event.
// Updates session URL and title when page navigates.
func (d *Daemon) handleTargetInfoChanged(evt cdp.Event) {
	var params struct {
		TargetInfo struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
			Title    string `json:"title"`
			URL      string `json:"url"`
		} `json:"targetInfo"`
	}
	if err := json.Unmarshal(evt.Params, &params); err != nil {
		return
	}

	// Only track page targets
	if params.TargetInfo.Type != "page" {
		return
	}

	d.debugf("Target.targetInfoChanged: targetID=%q, url=%q",
		params.TargetInfo.TargetID, params.TargetInfo.URL)

	// Update session by target ID
	d.sessions.UpdateByTargetID(
		params.TargetInfo.TargetID,
		params.TargetInfo.URL,
		params.TargetInfo.Title,
	)
}

// purgeSessionEntries removes all buffer entries for a session.
func (d *Daemon) purgeSessionEntries(sessionID string) {
	d.consoleBuf.RemoveIf(func(entry *ipc.ConsoleEntry) bool {
		return entry.SessionID == sessionID
	})
	d.networkBuf.RemoveIf(func(entry *ipc.NetworkEntry) bool {
		return entry.SessionID == sessionID
	})
}

// isBinaryMimeType returns true if the MIME type represents binary content.
func isBinaryMimeType(mimeType string) bool {
	mimeType = strings.ToLower(mimeType)

	// Check for binary prefixes
	binaryPrefixes := []string{
		"image/",
		"audio/",
		"video/",
		"font/",
	}
	for _, prefix := range binaryPrefixes {
		if strings.HasPrefix(mimeType, prefix) {
			return true
		}
	}

	// Check for specific binary types
	binaryTypes := []string{
		"application/octet-stream",
		"application/pdf",
		"application/zip",
		"application/gzip",
		"application/x-tar",
		"application/x-rar-compressed",
		"application/x-7z-compressed",
		"application/wasm",
	}
	for _, t := range binaryTypes {
		if mimeType == t {
			return true
		}
	}

	return false
}

// getBodiesDir returns the path to the bodies storage directory.
func getBodiesDir() string {
	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			// Fallback to temp directory if home cannot be determined
			return filepath.Join(os.TempDir(), "webctl-bodies")
		}
		stateHome = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(stateHome, "webctl", "bodies")
}

// saveBinaryBody saves binary body content to a file and returns the path.
func saveBinaryBody(requestID, url, mimeType, body string, isBase64 bool) (string, error) {
	// Create bodies directory
	bodiesDir := getBodiesDir()
	if err := os.MkdirAll(bodiesDir, 0700); err != nil {
		return "", err
	}

	// Generate filename
	ts := time.Now().Format("2006-01-02-150405")

	// Extract basename from URL
	basename := filepath.Base(url)
	if idx := strings.Index(basename, "?"); idx != -1 {
		basename = basename[:idx]
	}
	if basename == "" || basename == "/" {
		basename = "body"
	}

	// Ensure filename has extension based on MIME type
	ext := extensionFromMimeType(mimeType)
	if ext != "" && !strings.HasSuffix(basename, ext) {
		basename = basename + ext
	}

	// Sanitize request ID for filename (replace dots with dashes)
	safeRequestID := strings.ReplaceAll(requestID, ".", "-")

	filename := fmt.Sprintf("%s-%s-%s", ts, safeRequestID, basename)
	filePath := filepath.Join(bodiesDir, filename)

	// Decode body if base64
	var data []byte
	if isBase64 {
		var err error
		data, err = base64.StdEncoding.DecodeString(body)
		if err != nil {
			return "", err
		}
	} else {
		data = []byte(body)
	}

	// Write file
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return "", err
	}

	return filePath, nil
}

// extensionFromMimeType returns a file extension for the given MIME type.
func extensionFromMimeType(mimeType string) string {
	mimeType = strings.ToLower(mimeType)

	// Remove parameters (e.g., "text/html; charset=utf-8" -> "text/html")
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}

	extensions := map[string]string{
		"image/png":       ".png",
		"image/jpeg":      ".jpg",
		"image/gif":       ".gif",
		"image/webp":      ".webp",
		"image/svg+xml":   ".svg",
		"image/x-icon":    ".ico",
		"font/woff":       ".woff",
		"font/woff2":      ".woff2",
		"font/ttf":        ".ttf",
		"font/otf":        ".otf",
		"audio/mpeg":      ".mp3",
		"audio/ogg":       ".ogg",
		"audio/wav":       ".wav",
		"video/mp4":       ".mp4",
		"video/webm":      ".webm",
		"application/pdf": ".pdf",
		"application/zip": ".zip",
	}

	if ext, ok := extensions[mimeType]; ok {
		return ext
	}
	return ""
}

// clearBodiesDir removes all files in the bodies directory.
func clearBodiesDir() error {
	bodiesDir := getBodiesDir()
	entries, err := os.ReadDir(bodiesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			os.Remove(filepath.Join(bodiesDir, entry.Name()))
		}
	}
	return nil
}

// handleRequest processes an IPC request and returns a response.
func (d *Daemon) handleRequest(req ipc.Request) ipc.Response {
	switch req.Cmd {
	case "status":
		return d.handleStatus()
	case "console":
		return d.handleConsole()
	case "network":
		return d.handleNetwork()
	case "screenshot":
		return d.handleScreenshot(req)
	case "html":
		return d.handleHTML(req)
	case "target":
		return d.handleTarget(req.Target)
	case "clear":
		return d.handleClear(req.Target)
	case "cdp":
		return d.handleCDP(req)
	case "navigate":
		return d.handleNavigate(req)
	case "reload":
		return d.handleReload(req)
	case "back":
		return d.handleBack()
	case "forward":
		return d.handleForward()
	case "ready":
		return d.handleReady(req)
	case "click":
		return d.handleClick(req)
	case "focus":
		return d.handleFocus(req)
	case "type":
		return d.handleType(req)
	case "key":
		return d.handleKey(req)
	case "select":
		return d.handleSelect(req)
	case "scroll":
		return d.handleScroll(req)
	case "shutdown":
		return d.handleShutdown()
	default:
		return ipc.ErrorResponse(fmt.Sprintf("unknown command: %s", req.Cmd))
	}
}

// handleShutdown signals the daemon to shut down.
func (d *Daemon) handleShutdown() ipc.Response {
	// Signal shutdown in a goroutine so we can return the response first.
	// Use sync.Once to prevent panic from closing an already-closed channel.
	go func() {
		d.shutdownOnce.Do(func() {
			close(d.shutdown)
		})
	}()
	return ipc.SuccessResponse(map[string]string{
		"message": "shutting down",
	})
}

// handleStatus returns the daemon status.
func (d *Daemon) handleStatus() ipc.Response {
	status := ipc.StatusData{
		Running:  true,
		PID:      os.Getpid(),
		Sessions: d.sessions.All(),
	}

	// Get active session info
	active := d.sessions.Active()
	if active != nil {
		status.ActiveSession = active
		// Populate deprecated fields for backwards compatibility
		status.URL = active.URL
		status.Title = active.Title
	}

	return ipc.SuccessResponse(status)
}

// handleConsole returns buffered console entries filtered to active session.
func (d *Daemon) handleConsole() ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	allEntries := d.consoleBuf.All()
	var filtered []ipc.ConsoleEntry
	for _, e := range allEntries {
		if e.SessionID == activeID {
			filtered = append(filtered, e)
		}
	}

	return ipc.SuccessResponse(ipc.ConsoleData{
		Entries: filtered,
		Count:   len(filtered),
	})
}

// handleNetwork returns buffered network entries filtered to active session.
func (d *Daemon) handleNetwork() ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	allEntries := d.networkBuf.All()
	var filtered []ipc.NetworkEntry
	for _, e := range allEntries {
		if e.SessionID == activeID {
			filtered = append(filtered, e)
		}
	}

	return ipc.SuccessResponse(ipc.NetworkData{
		Entries: filtered,
		Count:   len(filtered),
	})
}

// handleScreenshot captures a screenshot of the active session.
func (d *Daemon) handleScreenshot(req ipc.Request) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	// Parse screenshot parameters
	var params ipc.ScreenshotParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("invalid screenshot parameters: %v", err))
		}
	}

	// Build CDP request parameters
	cdpParams := map[string]any{
		"format": "png",
	}

	// Add captureBeyondViewport for full-page screenshots
	if params.FullPage {
		cdpParams["captureBeyondViewport"] = true
	}

	// Call Page.captureScreenshot
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := d.cdp.SendToSession(ctx, activeID, "Page.captureScreenshot", cdpParams)
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to capture screenshot: %v", err))
	}

	// Parse CDP response
	var cdpResp struct {
		Data string `json:"data"` // base64-encoded PNG
	}
	if err := json.Unmarshal(result, &cdpResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse screenshot response: %v", err))
	}

	// Decode base64 data
	pngData, err := base64.StdEncoding.DecodeString(cdpResp.Data)
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to decode screenshot data: %v", err))
	}

	return ipc.SuccessResponse(ipc.ScreenshotData{
		Data: pngData,
	})
}

// handleHTML extracts HTML from the current page or specified selector.
// Uses JavaScript-based extraction (like Rod) to avoid DOM.getDocument blocking during navigation.
func (d *Daemon) handleHTML(req ipc.Request) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	// Parse HTML parameters
	var params ipc.HTMLParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("invalid html parameters: %v", err))
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// If no selector, get full page HTML using JavaScript
	if params.Selector == "" {
		js := `document.documentElement.outerHTML`

		result, err := d.cdp.SendToSession(ctx, activeID, "Runtime.evaluate", map[string]any{
			"expression":    js,
			"returnByValue": true,
		})
		if err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("failed to get HTML: %v", err))
		}

		var evalResp struct {
			Result struct {
				Value string `json:"value"`
			} `json:"result"`
			ExceptionDetails *struct {
				Text string `json:"text"`
			} `json:"exceptionDetails"`
		}
		if err := json.Unmarshal(result, &evalResp); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("failed to parse HTML response: %v", err))
		}
		if evalResp.ExceptionDetails != nil {
			return ipc.ErrorResponse(fmt.Sprintf("JavaScript error: %s", evalResp.ExceptionDetails.Text))
		}

		return ipc.SuccessResponse(ipc.HTMLData{
			HTML: evalResp.Result.Value,
		})
	}

	// For selector queries, use JavaScript querySelectorAll
	js := fmt.Sprintf(`(() => {
		const elements = document.querySelectorAll(%q);
		if (elements.length === 0) {
			return null;
		}
		const results = [];
		elements.forEach((el, i) => {
			if (elements.length > 1) {
				results.push('<!-- Element ' + (i+1) + ' of ' + elements.length + ': %s -->');
			}
			results.push(el.outerHTML);
		});
		return results.join('\n\n');
	})()`, params.Selector, params.Selector)

	result, err := d.cdp.SendToSession(ctx, activeID, "Runtime.evaluate", map[string]any{
		"expression":    js,
		"returnByValue": true,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to query selector: %v", err))
	}

	// Parse result - null means no matches, string means success
	var evalResp struct {
		Result struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse query response: %v", err))
	}
	if evalResp.ExceptionDetails != nil {
		return ipc.ErrorResponse(fmt.Sprintf("JavaScript error: %s", evalResp.ExceptionDetails.Text))
	}
	// null result means no elements matched
	if evalResp.Result.Type == "object" && evalResp.Result.Value == "" {
		return ipc.ErrorResponse(fmt.Sprintf("selector '%s' matched no elements", params.Selector))
	}

	return ipc.SuccessResponse(ipc.HTMLData{
		HTML: evalResp.Result.Value,
	})
}

// noActiveSessionError returns an error response with available sessions.
func (d *Daemon) noActiveSessionError() ipc.Response {
	sessions := d.sessions.All()
	if len(sessions) == 0 {
		return ipc.ErrorResponse("no active session - no pages available")
	}

	// Return error with session list so user can select
	data := struct {
		Error    string            `json:"error"`
		Sessions []ipc.PageSession `json:"sessions"`
	}{
		Error:    "no active session - use 'webctl target <id>' to select",
		Sessions: sessions,
	}

	raw, _ := json.Marshal(data)
	return ipc.Response{OK: false, Error: data.Error, Data: raw}
}

// handleTarget lists sessions or switches to a specific session.
func (d *Daemon) handleTarget(query string) ipc.Response {
	// If no query, list all sessions
	if query == "" {
		return ipc.SuccessResponse(ipc.TargetData{
			ActiveSession: d.sessions.ActiveID(),
			Sessions:      d.sessions.All(),
		})
	}

	// Try to find matching session
	matches := d.sessions.FindByQuery(query)

	if len(matches) == 0 {
		return ipc.ErrorResponse(fmt.Sprintf("no session matches query: %s", query))
	}

	if len(matches) > 1 {
		// Ambiguous match
		data := struct {
			Error   string            `json:"error"`
			Matches []ipc.PageSession `json:"matches"`
		}{
			Error:   fmt.Sprintf("ambiguous query '%s', matches multiple sessions", query),
			Matches: matches,
		}
		raw, _ := json.Marshal(data)
		return ipc.Response{OK: false, Error: data.Error, Data: raw}
	}

	// Single match - switch to it
	if !d.sessions.SetActive(matches[0].ID) {
		return ipc.ErrorResponse("failed to set active session")
	}

	return ipc.SuccessResponse(ipc.TargetData{
		ActiveSession: matches[0].ID,
		Sessions:      d.sessions.All(),
	})
}

// handleClear clears the specified buffer.
func (d *Daemon) handleClear(target string) ipc.Response {
	switch target {
	case "console":
		d.consoleBuf.Clear()
	case "network":
		d.networkBuf.Clear()
		clearBodiesDir()
	case "", "all":
		d.consoleBuf.Clear()
		d.networkBuf.Clear()
		clearBodiesDir()
	default:
		return ipc.ErrorResponse(fmt.Sprintf("unknown clear target: %s", target))
	}
	return ipc.SuccessResponse(nil)
}

// handleCDP forwards a raw CDP command to the browser.
// Request format: {"cmd": "cdp", "target": "Method.name", "params": {...}}
// Commands are sent to the active session. Use Target.* methods for browser-level commands.
func (d *Daemon) handleCDP(req ipc.Request) ipc.Response {
	if req.Target == "" {
		return ipc.ErrorResponse("cdp command requires target (CDP method name)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var params any
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("invalid params: %v", err))
		}
	}

	// Target.* methods are browser-level, send without session ID
	// All other methods go to the active session
	var result json.RawMessage
	var err error

	if strings.HasPrefix(req.Target, "Target.") {
		result, err = d.cdp.SendContext(ctx, req.Target, params)
	} else {
		activeID := d.sessions.ActiveID()
		if activeID == "" {
			return d.noActiveSessionError()
		}
		result, err = d.cdp.SendToSession(ctx, activeID, req.Target, params)
	}

	if err != nil {
		return ipc.ErrorResponse(err.Error())
	}

	return ipc.Response{OK: true, Data: result}
}

// writePIDFile writes the daemon PID to a file.
func (d *Daemon) writePIDFile() error {
	dir := filepath.Dir(d.config.PIDPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	pid := strconv.Itoa(os.Getpid())
	return os.WriteFile(d.config.PIDPath, []byte(pid), 0600)
}

// removePIDFile removes the PID file.
func (d *Daemon) removePIDFile() {
	os.Remove(d.config.PIDPath)
}

// handleFrameNavigated processes Page.frameNavigated events.
// Signals any waiting navigation operations.
func (d *Daemon) handleFrameNavigated(evt cdp.Event) {
	var params struct {
		Frame struct {
			ID       string `json:"id"`
			ParentID string `json:"parentId"`
			URL      string `json:"url"`
			Name     string `json:"name"`
		} `json:"frame"`
	}
	if err := json.Unmarshal(evt.Params, &params); err != nil {
		return
	}

	// Only care about main frame navigations (no parent)
	if params.Frame.ParentID != "" {
		return
	}

	// Check if anyone is waiting for this session's navigation
	if ch, ok := d.navWaiters.LoadAndDelete(evt.SessionID); ok {
		waiter := ch.(chan *frameNavigatedInfo)
		// Get title via JavaScript since frameNavigated doesn't include it
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		title := d.getPageTitle(ctx, evt.SessionID)
		select {
		case waiter <- &frameNavigatedInfo{URL: params.Frame.URL, Title: title}:
		default:
		}
	}
}

// handleLoadEventFired processes Page.loadEventFired events.
// Signals any waiting ready operations.
func (d *Daemon) handleLoadEventFired(evt cdp.Event) {
	if ch, ok := d.loadWaiters.LoadAndDelete(evt.SessionID); ok {
		waiter := ch.(chan struct{})
		select {
		case waiter <- struct{}{}:
		default:
		}
	}
}

// getPageTitle retrieves the current page title via JavaScript.
func (d *Daemon) getPageTitle(ctx context.Context, sessionID string) string {
	result, err := d.cdp.SendToSession(ctx, sessionID, "Runtime.evaluate", map[string]any{
		"expression":    "document.title",
		"returnByValue": true,
	})
	if err != nil {
		return ""
	}
	var resp struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return ""
	}
	return resp.Result.Value
}

// waitForFrameNavigated waits for a Page.frameNavigated event for the given session.
func (d *Daemon) waitForFrameNavigated(sessionID string, timeout time.Duration) (*frameNavigatedInfo, error) {
	ch := make(chan *frameNavigatedInfo, 1)
	d.navWaiters.Store(sessionID, ch)
	defer d.navWaiters.Delete(sessionID)

	select {
	case info := <-ch:
		return info, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for navigation")
	}
}

// waitForLoadEvent waits for a Page.loadEventFired event for the given session.
func (d *Daemon) waitForLoadEvent(sessionID string, timeout time.Duration) error {
	ch := make(chan struct{}, 1)
	d.loadWaiters.Store(sessionID, ch)
	defer d.loadWaiters.Delete(sessionID)

	select {
	case <-ch:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for page load")
	}
}

// handleNavigate navigates to a URL and waits for navigation to commit.
func (d *Daemon) handleNavigate(req ipc.Request) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	var params ipc.NavigateParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("invalid navigate parameters: %v", err))
	}

	if params.URL == "" {
		return ipc.ErrorResponse("url is required")
	}

	// Set up waiter before sending navigate command
	ch := make(chan *frameNavigatedInfo, 1)
	d.navWaiters.Store(activeID, ch)
	defer d.navWaiters.Delete(activeID)

	// Send navigate command
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := d.cdp.SendToSession(ctx, activeID, "Page.navigate", map[string]any{
		"url": params.URL,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("navigation failed: %v", err))
	}

	// Check for navigation errors in response
	var navResp struct {
		ErrorText string `json:"errorText"`
	}
	if err := json.Unmarshal(result, &navResp); err == nil && navResp.ErrorText != "" {
		return ipc.ErrorResponse(navResp.ErrorText)
	}

	// Wait for frameNavigated event
	select {
	case info := <-ch:
		return ipc.SuccessResponse(ipc.NavigateData{
			URL:   info.URL,
			Title: info.Title,
		})
	case <-time.After(30 * time.Second):
		return ipc.ErrorResponse("timeout waiting for navigation")
	case <-ctx.Done():
		return ipc.ErrorResponse("navigation cancelled")
	}
}

// handleReload reloads the current page.
func (d *Daemon) handleReload(req ipc.Request) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	var params ipc.ReloadParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("invalid reload parameters: %v", err))
		}
	}

	// Set up waiter before sending reload command
	ch := make(chan *frameNavigatedInfo, 1)
	d.navWaiters.Store(activeID, ch)
	defer d.navWaiters.Delete(activeID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := d.cdp.SendToSession(ctx, activeID, "Page.reload", map[string]any{
		"ignoreCache": params.IgnoreCache,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("reload failed: %v", err))
	}

	// Wait for frameNavigated event
	select {
	case info := <-ch:
		return ipc.SuccessResponse(ipc.NavigateData{
			URL:   info.URL,
			Title: info.Title,
		})
	case <-time.After(30 * time.Second):
		return ipc.ErrorResponse("timeout waiting for reload")
	}
}

// handleBack navigates to the previous history entry.
func (d *Daemon) handleBack() ipc.Response {
	return d.navigateHistory(-1)
}

// handleForward navigates to the next history entry.
func (d *Daemon) handleForward() ipc.Response {
	return d.navigateHistory(1)
}

// navigateHistory navigates forward or backward in history.
func (d *Daemon) navigateHistory(delta int) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get navigation history
	result, err := d.cdp.SendToSession(ctx, activeID, "Page.getNavigationHistory", nil)
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to get history: %v", err))
	}

	var history struct {
		CurrentIndex int `json:"currentIndex"`
		Entries      []struct {
			ID  int    `json:"id"`
			URL string `json:"url"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(result, &history); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse history: %v", err))
	}

	targetIndex := history.CurrentIndex + delta
	if targetIndex < 0 {
		return ipc.ErrorResponse("no previous page in history")
	}
	if targetIndex >= len(history.Entries) {
		return ipc.ErrorResponse("no next page in history")
	}

	// Set up waiter before navigating
	ch := make(chan *frameNavigatedInfo, 1)
	d.navWaiters.Store(activeID, ch)
	defer d.navWaiters.Delete(activeID)

	// Navigate to history entry
	_, err = d.cdp.SendToSession(ctx, activeID, "Page.navigateToHistoryEntry", map[string]any{
		"entryId": history.Entries[targetIndex].ID,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to navigate history: %v", err))
	}

	// Wait for frameNavigated event
	select {
	case info := <-ch:
		return ipc.SuccessResponse(ipc.NavigateData{
			URL:   info.URL,
			Title: info.Title,
		})
	case <-time.After(30 * time.Second):
		return ipc.ErrorResponse("timeout waiting for history navigation")
	}
}

// handleReady waits for the page to finish loading.
func (d *Daemon) handleReady(req ipc.Request) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	var params ipc.ReadyParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("invalid ready parameters: %v", err))
		}
	}

	timeout := 30 * time.Second
	if params.Timeout > 0 {
		timeout = time.Duration(params.Timeout) * time.Millisecond
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// First check if page is already loaded via document.readyState
	result, err := d.cdp.SendToSession(ctx, activeID, "Runtime.evaluate", map[string]any{
		"expression":    "document.readyState",
		"returnByValue": true,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to check page state: %v", err))
	}

	var evalResp struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse page state: %v", err))
	}

	// If already complete, return immediately
	if evalResp.Result.Value == "complete" {
		return ipc.SuccessResponse(nil)
	}

	// Page not yet loaded, wait for loadEventFired
	if err := d.waitForLoadEvent(activeID, timeout); err != nil {
		return ipc.ErrorResponse(err.Error())
	}

	return ipc.SuccessResponse(nil)
}

// handleClick clicks an element by selector.
func (d *Daemon) handleClick(req ipc.Request) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	var params ipc.ClickParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("invalid click parameters: %v", err))
	}

	if params.Selector == "" {
		return ipc.ErrorResponse("selector is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get element coordinates using JavaScript
	js := fmt.Sprintf(`(() => {
		const el = document.querySelector(%q);
		if (!el) return null;
		const rect = el.getBoundingClientRect();
		return {
			x: rect.left + rect.width / 2,
			y: rect.top + rect.height / 2
		};
	})()`, params.Selector)

	result, err := d.cdp.SendToSession(ctx, activeID, "Runtime.evaluate", map[string]any{
		"expression":    js,
		"returnByValue": true,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to find element: %v", err))
	}

	var evalResp struct {
		Result struct {
			Type  string `json:"type"`
			Value struct {
				X float64 `json:"x"`
				Y float64 `json:"y"`
			} `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse element position: %v", err))
	}
	if evalResp.Result.Type == "undefined" || (evalResp.Result.Value.X == 0 && evalResp.Result.Value.Y == 0) {
		return ipc.ErrorResponse(fmt.Sprintf("element not found: %s", params.Selector))
	}

	x := evalResp.Result.Value.X
	y := evalResp.Result.Value.Y

	// Send mouse events
	// mousePressed
	_, err = d.cdp.SendToSession(ctx, activeID, "Input.dispatchMouseEvent", map[string]any{
		"type":       "mousePressed",
		"x":          x,
		"y":          y,
		"button":     "left",
		"clickCount": 1,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to click: %v", err))
	}

	// mouseReleased
	_, err = d.cdp.SendToSession(ctx, activeID, "Input.dispatchMouseEvent", map[string]any{
		"type":       "mouseReleased",
		"x":          x,
		"y":          y,
		"button":     "left",
		"clickCount": 1,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to click: %v", err))
	}

	return ipc.SuccessResponse(nil)
}

// handleFocus focuses an element by selector.
func (d *Daemon) handleFocus(req ipc.Request) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	var params ipc.FocusParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("invalid focus parameters: %v", err))
	}

	if params.Selector == "" {
		return ipc.ErrorResponse("selector is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Focus using JavaScript
	js := fmt.Sprintf(`(() => {
		const el = document.querySelector(%q);
		if (!el) return false;
		el.focus();
		return true;
	})()`, params.Selector)

	result, err := d.cdp.SendToSession(ctx, activeID, "Runtime.evaluate", map[string]any{
		"expression":    js,
		"returnByValue": true,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to focus element: %v", err))
	}

	var evalResp struct {
		Result struct {
			Value bool `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse focus result: %v", err))
	}
	if !evalResp.Result.Value {
		return ipc.ErrorResponse(fmt.Sprintf("element not found: %s", params.Selector))
	}

	return ipc.SuccessResponse(nil)
}

// handleType types text into an element.
func (d *Daemon) handleType(req ipc.Request) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	var params ipc.TypeParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("invalid type parameters: %v", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// If selector provided, focus the element first
	if params.Selector != "" {
		focusResp := d.handleFocus(ipc.Request{
			Params: func() json.RawMessage {
				b, _ := json.Marshal(ipc.FocusParams{Selector: params.Selector})
				return b
			}(),
		})
		if !focusResp.OK {
			return focusResp
		}
	}

	// If clear flag, send Ctrl+A then Backspace
	if params.Clear {
		// Select all
		keyResp := d.handleKey(ipc.Request{
			Params: func() json.RawMessage {
				b, _ := json.Marshal(ipc.KeyParams{Key: "a", Ctrl: true})
				return b
			}(),
		})
		if !keyResp.OK {
			return keyResp
		}
		// Delete
		keyResp = d.handleKey(ipc.Request{
			Params: func() json.RawMessage {
				b, _ := json.Marshal(ipc.KeyParams{Key: "Backspace"})
				return b
			}(),
		})
		if !keyResp.OK {
			return keyResp
		}
	}

	// Insert text
	if params.Text != "" {
		_, err := d.cdp.SendToSession(ctx, activeID, "Input.insertText", map[string]any{
			"text": params.Text,
		})
		if err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("failed to type text: %v", err))
		}
	}

	// If key specified, send it
	if params.Key != "" {
		keyResp := d.handleKey(ipc.Request{
			Params: func() json.RawMessage {
				b, _ := json.Marshal(ipc.KeyParams{Key: params.Key})
				return b
			}(),
		})
		if !keyResp.OK {
			return keyResp
		}
	}

	return ipc.SuccessResponse(nil)
}

// handleKey sends a keyboard key event.
func (d *Daemon) handleKey(req ipc.Request) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	var params ipc.KeyParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("invalid key parameters: %v", err))
	}

	if params.Key == "" {
		return ipc.ErrorResponse("key is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Calculate modifiers bitmap: Alt=1, Ctrl=2, Meta=4, Shift=8
	modifiers := 0
	if params.Alt {
		modifiers |= 1
	}
	if params.Ctrl {
		modifiers |= 2
	}
	if params.Meta {
		modifiers |= 4
	}
	if params.Shift {
		modifiers |= 8
	}

	// Map key names to CDP key info
	keyInfo := getKeyInfo(params.Key)

	// keyDown
	_, err := d.cdp.SendToSession(ctx, activeID, "Input.dispatchKeyEvent", map[string]any{
		"type":                  "keyDown",
		"key":                   keyInfo.key,
		"code":                  keyInfo.code,
		"windowsVirtualKeyCode": keyInfo.keyCode,
		"modifiers":             modifiers,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to send key: %v", err))
	}

	// keyUp
	_, err = d.cdp.SendToSession(ctx, activeID, "Input.dispatchKeyEvent", map[string]any{
		"type":                  "keyUp",
		"key":                   keyInfo.key,
		"code":                  keyInfo.code,
		"windowsVirtualKeyCode": keyInfo.keyCode,
		"modifiers":             modifiers,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to send key: %v", err))
	}

	return ipc.SuccessResponse(nil)
}

// keyInfo holds CDP key event parameters.
type keyInfo struct {
	key     string
	code    string
	keyCode int
}

// getKeyInfo returns CDP key parameters for a key name.
func getKeyInfo(key string) keyInfo {
	// Common key mappings
	switch key {
	case "Enter":
		return keyInfo{key: "Enter", code: "Enter", keyCode: 13}
	case "Tab":
		return keyInfo{key: "Tab", code: "Tab", keyCode: 9}
	case "Escape":
		return keyInfo{key: "Escape", code: "Escape", keyCode: 27}
	case "Backspace":
		return keyInfo{key: "Backspace", code: "Backspace", keyCode: 8}
	case "Delete":
		return keyInfo{key: "Delete", code: "Delete", keyCode: 46}
	case "ArrowUp":
		return keyInfo{key: "ArrowUp", code: "ArrowUp", keyCode: 38}
	case "ArrowDown":
		return keyInfo{key: "ArrowDown", code: "ArrowDown", keyCode: 40}
	case "ArrowLeft":
		return keyInfo{key: "ArrowLeft", code: "ArrowLeft", keyCode: 37}
	case "ArrowRight":
		return keyInfo{key: "ArrowRight", code: "ArrowRight", keyCode: 39}
	case "Home":
		return keyInfo{key: "Home", code: "Home", keyCode: 36}
	case "End":
		return keyInfo{key: "End", code: "End", keyCode: 35}
	case "PageUp":
		return keyInfo{key: "PageUp", code: "PageUp", keyCode: 33}
	case "PageDown":
		return keyInfo{key: "PageDown", code: "PageDown", keyCode: 34}
	case "Space":
		return keyInfo{key: " ", code: "Space", keyCode: 32}
	default:
		// Single character keys
		if len(key) == 1 {
			keyCode := int(key[0])
			if key[0] >= 'a' && key[0] <= 'z' {
				keyCode = int(key[0]) - 32 // Convert to uppercase keyCode
			}
			return keyInfo{key: key, code: "Key" + strings.ToUpper(key), keyCode: keyCode}
		}
		// Unknown key, return as-is
		return keyInfo{key: key, code: key, keyCode: 0}
	}
}

// handleSelect selects an option in a dropdown.
func (d *Daemon) handleSelect(req ipc.Request) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	var params ipc.SelectParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("invalid select parameters: %v", err))
	}

	if params.Selector == "" {
		return ipc.ErrorResponse("selector is required")
	}
	if params.Value == "" {
		return ipc.ErrorResponse("value is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Select using JavaScript
	js := fmt.Sprintf(`(() => {
		const el = document.querySelector(%q);
		if (!el) return 'not_found';
		if (el.tagName !== 'SELECT') return 'not_select';
		el.value = %q;
		el.dispatchEvent(new Event('change', {bubbles: true}));
		return 'ok';
	})()`, params.Selector, params.Value)

	result, err := d.cdp.SendToSession(ctx, activeID, "Runtime.evaluate", map[string]any{
		"expression":    js,
		"returnByValue": true,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to select option: %v", err))
	}

	var evalResp struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse select result: %v", err))
	}

	switch evalResp.Result.Value {
	case "not_found":
		return ipc.ErrorResponse(fmt.Sprintf("element not found: %s", params.Selector))
	case "not_select":
		return ipc.ErrorResponse(fmt.Sprintf("element is not a select: %s", params.Selector))
	case "ok":
		return ipc.SuccessResponse(nil)
	default:
		return ipc.ErrorResponse("unexpected select result")
	}
}

// handleScroll scrolls to an element or position.
func (d *Daemon) handleScroll(req ipc.Request) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	var params ipc.ScrollParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("invalid scroll parameters: %v", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var js string
	switch params.Mode {
	case "element":
		if params.Selector == "" {
			return ipc.ErrorResponse("selector is required for element scroll")
		}
		js = fmt.Sprintf(`(() => {
			const el = document.querySelector(%q);
			if (!el) return false;
			el.scrollIntoView({block: 'center', behavior: 'instant'});
			return true;
		})()`, params.Selector)
	case "to":
		js = fmt.Sprintf(`(() => {
			window.scrollTo({left: %d, top: %d, behavior: 'instant'});
			return true;
		})()`, params.ToX, params.ToY)
	case "by":
		js = fmt.Sprintf(`(() => {
			window.scrollBy({left: %d, top: %d, behavior: 'instant'});
			return true;
		})()`, params.ByX, params.ByY)
	default:
		return ipc.ErrorResponse("invalid scroll mode: must be 'element', 'to', or 'by'")
	}

	result, err := d.cdp.SendToSession(ctx, activeID, "Runtime.evaluate", map[string]any{
		"expression":    js,
		"returnByValue": true,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to scroll: %v", err))
	}

	var evalResp struct {
		Result struct {
			Value bool `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse scroll result: %v", err))
	}
	if !evalResp.Result.Value {
		return ipc.ErrorResponse(fmt.Sprintf("element not found: %s", params.Selector))
	}

	return ipc.SuccessResponse(nil)
}
