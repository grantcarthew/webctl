package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
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
	navWaiters  sync.Map // map[string]chan *frameNavigatedInfo for sessionID -> waiter
	loadWaiters sync.Map // map[string]chan struct{} for sessionID -> waiter (loadEventFired)

	// Navigation state tracking
	// navigating tracks sessions currently in navigation (before loadEventFired)
	// Value is a chan struct{} that will be closed when load completes
	navigating sync.Map // map[string]chan struct{} for sessionID -> done channel

	// Target attachment tracking
	// attachedTargets tracks which targetIDs we've already attached to (prevents double-attach)
	attachedTargets sync.Map // map[string]bool for targetID -> attached

	// Network domain lazy enablement
	// networkEnabled tracks which sessions have Network.enable called
	// We enable Network lazily because it causes Runtime.evaluate to block until networkIdle
	networkEnabled sync.Map // map[string]bool for sessionID -> enabled
}

// frameNavigatedInfo contains information from a Page.frameNavigated event.
type frameNavigatedInfo struct {
	URL   string
	Title string
}

// debugf logs a debug message if debug mode is enabled.
func (d *Daemon) debugf(format string, args ...any) {
	if d.debug {
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Fprintf(os.Stderr, "[DEBUG] [%s] "+format+"\n", append([]any{timestamp}, args...)...)
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

	// Update config with actual port used (may differ from requested if auto-selected)
	d.config.Port = b.Port()

	// Connect to browser-level CDP WebSocket (not page target)
	// This allows us to use Target.setAutoAttach for session management
	version, err := d.browser.Version(ctx)
	if err != nil {
		return fmt.Errorf("failed to get browser version: %w", err)
	}
	d.debugf("Browser version info: %+v", version)
	d.debugf("Connecting to CDP WebSocket: %s", version.WebSocketURL)

	cdpClient, err := cdp.Dial(ctx, version.WebSocketURL)
	if err != nil {
		return fmt.Errorf("failed to connect to CDP: %w", err)
	}
	d.cdp = cdpClient
	defer d.cdp.Close()
	d.debugf("CDP client connected successfully")

	// Subscribe to events before enabling domains
	d.debugf("Subscribing to CDP events")
	d.subscribeEvents()
	d.debugf("Event subscriptions complete")

	// Enable auto-attach for session tracking
	d.debugf("Enabling target discovery and attachment")
	if err := d.enableAutoAttach(); err != nil {
		return fmt.Errorf("failed to enable auto-attach: %w", err)
	}
	d.debugf("Target discovery and attachment enabled")

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

// enableAutoAttach enables Target.setDiscoverTargets for target discovery.
// We use manual Target.attachToTarget with flatten:true for each discovered target.
func (d *Daemon) enableAutoAttach() error {
	d.debugf("Calling Target.setDiscoverTargets...")
	// Enable target discovery to receive targetCreated/targetInfoChanged/targetDestroyed events
	_, err := d.cdp.Send("Target.setDiscoverTargets", map[string]any{
		"discover": true,
	})
	if err != nil {
		return fmt.Errorf("failed to set discover targets: %w", err)
	}
	d.debugf("Target.setDiscoverTargets succeeded")

	// NOTE: We do NOT use Target.setAutoAttach here.
	// Instead, we manually call Target.attachToTarget for each target in handleTargetCreated.
	// Using flatten:true in attachToTarget (not setAutoAttach) avoids networkIdle blocking.

	// Attach to any existing targets that were created before we enabled discovery
	d.debugf("Calling Target.getTargets to find existing targets...")
	result, err := d.cdp.Send("Target.getTargets", nil)
	if err != nil {
		return fmt.Errorf("failed to get existing targets: %w", err)
	}
	d.debugf("Target.getTargets succeeded, parsing results...")

	var targetsResult struct {
		TargetInfos []struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
			Title    string `json:"title"`
			URL      string `json:"url"`
		} `json:"targetInfos"`
	}
	if err := json.Unmarshal(result, &targetsResult); err != nil {
		return fmt.Errorf("failed to parse targets: %w", err)
	}
	d.debugf("Found %d total targets", len(targetsResult.TargetInfos))

	// Attach to existing page targets asynchronously
	for _, targetInfo := range targetsResult.TargetInfos {
		d.debugf("  Target: type=%q, targetID=%q, url=%q", targetInfo.Type, targetInfo.TargetID, targetInfo.URL)
		if targetInfo.Type == "page" {
			// Check if we've already attached (targetCreated might have fired before getTargets returned)
			if _, alreadyAttached := d.attachedTargets.LoadOrStore(targetInfo.TargetID, true); alreadyAttached {
				d.debugf("  Already attached to targetID=%q, skipping", targetInfo.TargetID)
				continue
			}

			targetID := targetInfo.TargetID // capture for goroutine
			go func() {
				d.debugf("  Attaching to existing page target: targetID=%q", targetID)
				_, err := d.cdp.Send("Target.attachToTarget", map[string]any{
					"targetId": targetID,
					"flatten":  true,
				})
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: failed to attach to existing target %q: %v\n", targetID, err)
					// Remove from attachedTargets on failure so we can retry
					d.attachedTargets.Delete(targetID)
				} else {
					d.debugf("  Successfully attached to target %q", targetID)
				}
			}()
		}
	}

	return nil
}

// enableDomainsForSession enables CDP domains for a specific session.
func (d *Daemon) enableDomainsForSession(sessionID string) error {
	// NOTE: Enabling Network domain causes Chrome to track network activity
	// and block Runtime.evaluate until networkIdle.
	// We enable minimal domains and add Network only when needed.
	domains := []string{"Runtime.enable", "Page.enable", "DOM.enable"}
	for _, method := range domains {
		if _, err := d.cdp.SendToSession(context.Background(), sessionID, method, nil); err != nil {
			return fmt.Errorf("failed to enable %s: %w", method, err)
		}
	}

	// Enable lifecycle events (required to receive Page.lifecycleEvent)
	if _, err := d.cdp.SendToSession(context.Background(), sessionID, "Page.setLifecycleEventsEnabled", map[string]any{"enabled": true}); err != nil {
		return fmt.Errorf("failed to enable lifecycle events: %w", err)
	}

	// NOTE: We don't use waitForDebuggerOnStart with manual Target.attachToTarget,
	// so no need to call Runtime.runIfWaitingForDebugger

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
		return d.handleBack(req)
	case "forward":
		return d.handleForward(req)
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
	case "eval":
		return d.handleEval(req)
	case "cookies":
		return d.handleCookies(req)
	case "find":
		return d.handleFind(req)
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
