package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/grantcarthew/webctl/internal/cdp"
	"github.com/grantcarthew/webctl/internal/ipc"
)

const (
	// HeartbeatInterval is the time between heartbeat checks.
	HeartbeatInterval = 5 * time.Second
	// HeartbeatTimeout is the maximum time to wait for a heartbeat response.
	HeartbeatTimeout = 5 * time.Second
	// TargetAttachmentWait is the time to wait for CDP target attachment events
	// after reconnection. This allows the browser to send targetCreated events
	// for existing tabs before we attempt navigation. 500ms is sufficient for
	// local connections; increase if targeting remote browsers with latency.
	TargetAttachmentWait = 500 * time.Millisecond
)

// setReconnecting atomically sets the reconnecting flag.
func (d *Daemon) setReconnecting(reconnecting bool) {
	if reconnecting {
		atomic.StoreInt32(&d.reconnectingFlag, 1)
	} else {
		atomic.StoreInt32(&d.reconnectingFlag, 0)
	}
}

// isReconnecting atomically checks if reconnection is in progress.
func (d *Daemon) isReconnecting() bool {
	return atomic.LoadInt32(&d.reconnectingFlag) == 1
}

// startHeartbeat starts the heartbeat goroutine that monitors connection health.
// It runs until the context is cancelled or a disconnect is detected.
// Returns a channel that signals when a disconnect is detected.
//
// Note: The disconnect channel is buffered (size 1) and only the first error
// is sent before the goroutine exits. This is intentional - the first disconnect
// triggers recovery, and subsequent errors are ignored until recovery completes.
func (d *Daemon) startHeartbeat(ctx context.Context) <-chan error {
	disconnectCh := make(chan error, 1)

	// Create a cancellable context for this specific heartbeat instance
	heartbeatCtx, cancel := context.WithCancel(ctx)

	// Store the cancel function so we can stop the old heartbeat
	d.heartbeatCancelMu.Lock()
	// Cancel any previous heartbeat before starting new one
	if d.heartbeatCancel != nil {
		d.heartbeatCancel()
	}
	d.heartbeatCancel = cancel
	d.heartbeatCancelMu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("heartbeat panic: %v", r)
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				d.connMgr.SetDisconnected(err)
				select {
				case disconnectCh <- err:
				default:
				}
			}
		}()

		ticker := time.NewTicker(HeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-heartbeatCtx.Done():
				d.debugf(false, "Heartbeat cancelled")
				return
			case <-d.shutdown:
				return
			case <-ticker.C:
				if err := d.performHeartbeat(); err != nil {
					d.debugf(false, "Heartbeat failed: %v", err)
					select {
					case disconnectCh <- err:
					default:
					}
					return
				}
			}
		}
	}()

	return disconnectCh
}

// performHeartbeat sends a Browser.getVersion command to verify connection health.
func (d *Daemon) performHeartbeat() error {
	cdpClient := d.getCDP()
	if cdpClient == nil {
		return errors.New("CDP client not initialized")
	}

	// Check if already disconnected
	if d.connMgr.State() == StateDisconnected {
		return errors.New("already disconnected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), HeartbeatTimeout)
	defer cancel()

	// Browser.getVersion is a lightweight command that validates the full CDP stack
	_, err := cdpClient.SendContext(ctx, "Browser.getVersion", nil)
	if err != nil {
		// Classify the error to determine if we should reconnect
		reason, shouldReconnect := ClassifyCloseCode(err)
		d.debugf(false, "Heartbeat error: %v (reason=%s, shouldReconnect=%t)", err, reason, shouldReconnect)

		if shouldReconnect {
			// Record the error - reconnection logic will handle state transition
			return err
		}

		// Graceful close - no reconnection
		d.connMgr.SetDisconnected(err)
		return err
	}

	// Success - record heartbeat
	d.connMgr.RecordHeartbeat()
	return nil
}

// handleDisconnectAndRecover processes a detected disconnect and attempts recovery.
// Returns true if recovery succeeded, false if we should shut down.
func (d *Daemon) handleDisconnectAndRecover(ctx context.Context, err error) bool {
	d.debugf(false, "handleDisconnectAndRecover called with error: %v", err)

	// Set reconnecting flag to prevent event handlers from modifying sync.Maps.
	// Note: Small TOCTOU race window exists between event handlers checking the
	// flag and this function setting it or clearing maps. Impact is minimal:
	//   - During disconnection, the WebSocket is closing so new events are unlikely
	//   - Worst case: stale map entry that gets cleaned up on next reconnection
	//   - Full atomic protection would require complex locking with deadlock risks
	d.setReconnecting(true)
	defer d.setReconnecting(false)

	// Preserve last URL before clearing sessions
	if active := d.sessions.Active(); active != nil {
		d.lastURL = active.URL
		d.debugf(false, "Preserved lastURL for recovery: %s", d.lastURL)
	}

	// Classify the disconnect
	reason, shouldReconnect := ClassifyCloseCode(err)
	d.debugf(false, "Disconnect classified: reason=%s, shouldReconnect=%t", reason, shouldReconnect)

	// Clear sessions but keep buffers
	d.sessions.Clear()

	// Clear all session-related sync.Maps to prevent stale entries
	// The reconnecting flag prevents event handlers from adding new entries
	clearSyncMap := func(m *sync.Map) {
		m.Range(func(key, _ any) bool {
			m.Delete(key)
			return true
		})
	}
	clearSyncMap(&d.attachedTargets)
	clearSyncMap(&d.networkEnabled)
	clearSyncMap(&d.navigating)
	clearSyncMap(&d.loadWaiters)
	clearSyncMap(&d.navWaiters)

	if !shouldReconnect {
		// Graceful close - user closed browser, trigger shutdown
		d.connMgr.SetDisconnected(err)
		d.browserLostMu.Lock()
		d.browserLost = true
		d.browserLostMu.Unlock()
		go d.shutdownOnce.Do(func() {
			close(d.shutdown)
		})
		return false
	}

	// Abnormal disconnect - attempt synchronous reconnection
	return d.attemptAutoReconnect(ctx, err)
}

// attemptAutoReconnect tries to reconnect with exponential backoff.
// Returns true if reconnection succeeded.
func (d *Daemon) attemptAutoReconnect(ctx context.Context, initialErr error) bool {
	// Track the last error for logging; start with the initial disconnect error
	lastErr := initialErr

	for {
		// Check BEFORE incrementing to ensure exactly maxAttempts attempts are made.
		// Without this ordering, the initial SetReconnecting would consume one count
		// without making an attempt, resulting in maxAttempts-1 actual attempts.
		if d.connMgr.MaxAttemptsReached() {
			break
		}

		// Transition to reconnecting state and increment attempt counter
		d.connMgr.SetReconnecting(lastErr)

		// Wait before attempting reconnection
		delay := d.connMgr.NextDelay()
		d.debugf(false, "Waiting %v before reconnection attempt", delay)

		select {
		case <-d.shutdown:
			return false
		case <-ctx.Done():
			return false
		case <-time.After(delay):
		}

		// Attempt reconnection
		err := d.attemptReconnect()
		if err == nil {
			// Reconnection successful
			d.connMgr.SetConnected()
			return true
		}

		d.debugf(false, "Reconnection attempt failed: %v", err)
		lastErr = err
	}

	// Max attempts reached
	fmt.Fprintln(os.Stderr, "Error: max reconnection attempts reached - giving up")
	d.connMgr.SetDisconnected(errors.New("max reconnection attempts exceeded"))
	d.triggerShutdown()
	return false
}

// attemptReconnect attempts to reconnect to the browser and restore state.
func (d *Daemon) attemptReconnect() error {
	d.debugf(false, "Attempting reconnection...")

	// Check if browser is still alive via HTTP
	ctx, cancel := context.WithTimeout(context.Background(), HeartbeatTimeout)
	defer cancel()

	version, err := d.browser.Version(ctx)
	if err != nil {
		return fmt.Errorf("browser not responding: %w", err)
	}
	d.debugf(false, "Browser still alive at %s", version.WebSocketURL)

	// Dial new CDP WebSocket connection
	newClient, err := d.dialCDP(ctx, version.WebSocketURL)
	if err != nil {
		return fmt.Errorf("failed to connect to CDP: %w", err)
	}

	// Atomically replace CDP client (avoid race between close and swap)
	d.cdpMu.Lock()
	oldClient := d.cdp
	d.cdp = newClient
	d.cdpMu.Unlock()

	// Close old client after swap (safe to do outside lock)
	if oldClient != nil {
		_ = oldClient.Close()
	}
	d.debugf(false, "CDP client reconnected")

	// Re-subscribe to events
	d.subscribeEvents()
	d.debugf(false, "Event subscriptions restored")

	// Re-enable auto-attach to discover and attach to existing targets
	if err := d.enableAutoAttach(); err != nil {
		return fmt.Errorf("failed to enable auto-attach: %w", err)
	}
	d.debugf(false, "Auto-attach re-enabled")

	// Wait for target attachment events before attempting navigation.
	// Note: This is a fixed heuristic delay. Future enhancement could replace
	// this with polling for active sessions or waiting for specific CDP events.
	// Current approach is acceptable for local browsers but may be insufficient
	// for remote browsers with high latency.
	time.Sleep(TargetAttachmentWait)

	// Re-navigate to last URL if we have one and there's an active session
	if d.lastURL != "" {
		d.debugf(false, "Attempting to restore last URL: %s", d.lastURL)
		if active := d.sessions.Active(); active != nil {
			navCtx, navCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer navCancel()

			cdpClient := d.getCDP()
			if cdpClient != nil {
				_, err := cdpClient.SendToSession(navCtx, active.ID, "Page.navigate", map[string]any{
					"url": d.lastURL,
				})
				if err != nil {
					d.debugf(false, "Warning: failed to restore last URL: %v", err)
					// Don't fail reconnection for navigation errors - the CDP connection
					// is restored and functional, session recovery is a best-effort feature
				} else {
					d.debugf(false, "Restored last URL successfully")
				}
			}
		}
	}

	return nil
}

// dialCDP creates a new CDP client connection.
func (d *Daemon) dialCDP(ctx context.Context, wsURL string) (*cdp.Client, error) {
	return cdp.Dial(ctx, wsURL)
}

// triggerShutdown initiates daemon shutdown.
func (d *Daemon) triggerShutdown() {
	d.browserLostMu.Lock()
	d.browserLost = true
	d.browserLostMu.Unlock()
	go d.shutdownOnce.Do(func() {
		close(d.shutdown)
	})
}

// handleReconnect handles the manual reconnect IPC command.
// Note: Manual reconnect is primarily useful when already connected but wanting
// to force a fresh connection. If automatic reconnection has failed and the daemon
// is in StateDisconnected, the daemon will be shutting down and manual reconnect
// may not be processed in time. In that case, restart the daemon with `webctl start`.
func (d *Daemon) handleReconnect() ipc.Response {
	state := d.connMgr.State()

	// If already connected, return success
	if state == StateConnected {
		return ipc.SuccessResponse(map[string]any{
			"message": "already connected",
			"state":   state.String(),
		})
	}

	// If already reconnecting, return status
	if state == StateReconnecting {
		return ipc.SuccessResponse(map[string]any{
			"message":        "reconnection in progress",
			"state":          state.String(),
			"reconnectCount": d.connMgr.ReconnectCount(),
		})
	}

	// Attempt manual reconnection
	err := d.attemptReconnect()
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("reconnection failed: %v", err))
	}

	// Success
	d.connMgr.SetConnected()
	return ipc.SuccessResponse(map[string]any{
		"message": "reconnected successfully",
		"state":   StateConnected.String(),
		"url":     d.lastURL,
	})
}
