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

	// Add to session manager
	d.sessions.Add(
		params.SessionID,
		params.TargetInfo.TargetID,
		params.TargetInfo.URL,
		params.TargetInfo.Title,
	)

	// Enable domains for this session (async to not block event loop)
	go func() {
		if err := d.enableDomainsForSession(params.SessionID); err != nil {
			// Log error but don't fail - session is still tracked
			fmt.Fprintf(os.Stderr, "warning: failed to enable domains for session: %v\n", err)
		}
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

	// Remove from session manager
	_, _ = d.sessions.Remove(params.SessionID)

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

	// Get document root node
	docResult, err := d.cdp.SendToSession(ctx, activeID, "DOM.getDocument", map[string]any{
		"depth":  -1,
		"pierce": false,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to get document: %v", err))
	}

	var docResp struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(docResult, &docResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse document response: %v", err))
	}

	// If no selector, get full page HTML
	if params.Selector == "" {
		htmlResult, err := d.cdp.SendToSession(ctx, activeID, "DOM.getOuterHTML", map[string]any{
			"nodeId": docResp.Root.NodeID,
		})
		if err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("failed to get HTML: %v", err))
		}

		var htmlResp struct {
			OuterHTML string `json:"outerHTML"`
		}
		if err := json.Unmarshal(htmlResult, &htmlResp); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("failed to parse HTML response: %v", err))
		}

		return ipc.SuccessResponse(ipc.HTMLData{
			HTML: htmlResp.OuterHTML,
		})
	}

	// Query selector for matching elements
	queryResult, err := d.cdp.SendToSession(ctx, activeID, "DOM.querySelectorAll", map[string]any{
		"nodeId":   docResp.Root.NodeID,
		"selector": params.Selector,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to query selector: %v", err))
	}

	var queryResp struct {
		NodeIDs []int `json:"nodeIds"`
	}
	if err := json.Unmarshal(queryResult, &queryResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse query response: %v", err))
	}

	// Check if selector matched any elements
	if len(queryResp.NodeIDs) == 0 {
		return ipc.ErrorResponse(fmt.Sprintf("selector '%s' matched no elements", params.Selector))
	}

	// Get HTML for each matching element
	var htmlParts []string
	for i, nodeID := range queryResp.NodeIDs {
		htmlResult, err := d.cdp.SendToSession(ctx, activeID, "DOM.getOuterHTML", map[string]any{
			"nodeId": nodeID,
		})
		if err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("failed to get HTML for element %d: %v", i+1, err))
		}

		var htmlResp struct {
			OuterHTML string `json:"outerHTML"`
		}
		if err := json.Unmarshal(htmlResult, &htmlResp); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("failed to parse HTML response for element %d: %v", i+1, err))
		}

		// Add comment separator for multiple matches
		if len(queryResp.NodeIDs) > 1 {
			htmlParts = append(htmlParts, fmt.Sprintf("<!-- Element %d of %d: %s -->", i+1, len(queryResp.NodeIDs), params.Selector))
		}
		htmlParts = append(htmlParts, htmlResp.OuterHTML)
	}

	// Join with newlines
	html := ""
	for i, part := range htmlParts {
		html += part
		if i < len(htmlParts)-1 {
			html += "\n\n"
		}
	}

	return ipc.SuccessResponse(ipc.HTMLData{
		HTML: html,
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
