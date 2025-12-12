package browser

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Browser represents a running Chrome instance with CDP enabled.
type Browser struct {
	cmd      *exec.Cmd
	port     int
	dataDir  string
	ownsData bool // true if we created the temp data dir
}

// ErrBrowserClosed is returned when operating on a closed browser.
var ErrBrowserClosed = errors.New("browser is closed")

// ErrNoPageTarget is returned when no page target is available.
var ErrNoPageTarget = errors.New("no page target found")

// ErrStartTimeout is returned when the browser fails to start in time.
var ErrStartTimeout = errors.New("browser start timeout")

// Start launches a new Chrome browser with CDP enabled.
// It waits for the CDP endpoint to become available before returning.
func Start(opts LaunchOptions) (*Browser, error) {
	binPath, err := FindChrome()
	if err != nil {
		return nil, err
	}

	return StartWithBinary(binPath, opts)
}

// StartWithBinary launches Chrome using the specified binary path.
func StartWithBinary(binPath string, opts LaunchOptions) (*Browser, error) {
	port := opts.Port
	if port == 0 {
		port = DefaultPort
	}

	cmd, dataDir, err := spawnProcess(binPath, opts)
	if err != nil {
		return nil, err
	}

	b := &Browser{
		cmd:      cmd,
		port:     port,
		dataDir:  dataDir,
		ownsData: opts.UserDataDir == "", // we created temp dir if UserDataDir was empty
	}

	// Wait for CDP endpoint to become available
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := b.waitForCDP(ctx); err != nil {
		b.Close()
		return nil, err
	}

	return b, nil
}

// waitForCDP polls the CDP endpoint until it responds or context is cancelled.
func (b *Browser) waitForCDP(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ErrStartTimeout
		case <-ticker.C:
			_, err := FetchVersion(ctx, "127.0.0.1", b.port)
			if err == nil {
				return nil
			}
		}
	}
}

// Port returns the CDP debugging port.
func (b *Browser) Port() int {
	return b.port
}

// PID returns the browser process ID.
func (b *Browser) PID() int {
	if b.cmd == nil || b.cmd.Process == nil {
		return 0
	}
	return b.cmd.Process.Pid
}

// Targets fetches the list of available CDP targets.
func (b *Browser) Targets(ctx context.Context) ([]Target, error) {
	return FetchTargets(ctx, "127.0.0.1", b.port)
}

// PageTarget returns the first page-type target.
func (b *Browser) PageTarget(ctx context.Context) (*Target, error) {
	targets, err := b.Targets(ctx)
	if err != nil {
		return nil, err
	}

	target := FindPageTarget(targets)
	if target == nil {
		return nil, ErrNoPageTarget
	}

	return target, nil
}

// Version fetches the browser version information.
func (b *Browser) Version(ctx context.Context) (*VersionInfo, error) {
	return FetchVersion(ctx, "127.0.0.1", b.port)
}

// Close terminates the browser process and cleans up resources.
func (b *Browser) Close() error {
	if b.cmd == nil || b.cmd.Process == nil {
		return nil
	}

	// Send SIGTERM for graceful shutdown
	if err := b.cmd.Process.Signal(os.Interrupt); err != nil {
		// Process may have already exited
		if !errors.Is(err, os.ErrProcessDone) {
			// Force kill
			_ = b.cmd.Process.Kill()
		}
	}

	// Wait for process to exit
	_ = b.cmd.Wait()

	// Clean up temp data directory
	if b.ownsData && b.dataDir != "" {
		os.RemoveAll(b.dataDir)
	}

	b.cmd = nil
	return nil
}

// WebSocketURL returns the WebSocket URL for connecting to the first page target.
func (b *Browser) WebSocketURL(ctx context.Context) (string, error) {
	target, err := b.PageTarget(ctx)
	if err != nil {
		return "", err
	}

	if target.WebSocketURL == "" {
		return "", fmt.Errorf("target has no WebSocket URL")
	}

	return target.WebSocketURL, nil
}
