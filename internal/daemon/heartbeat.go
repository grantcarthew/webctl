package daemon

import (
	"context"
	"errors"
	"time"

	"github.com/coder/websocket"
)

const (
	heartbeatInterval = 5 * time.Second
	heartbeatTimeout  = 5 * time.Second
)

// classifyDisconnect returns a human-readable disconnect reason from a CDP/websocket error.
func classifyDisconnect(err error) string {
	if err == nil {
		return "browser disconnected"
	}

	code := websocket.CloseStatus(err)
	switch code {
	case websocket.StatusNormalClosure, websocket.StatusGoingAway:
		return "browser closed normally"
	default:
		if errors.Is(err, context.DeadlineExceeded) {
			return "browser unresponsive (heartbeat timeout)"
		}
		return "browser connection lost"
	}
}

// startHeartbeat launches a goroutine that periodically sends Browser.getVersion
// to detect silent browser disconnections. On failure, it sends the underlying
// error to disconnectCh for classification by Run().
func (d *Daemon) startHeartbeat(ctx context.Context, disconnectCh chan<- error) {
	go func() {
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				hbCtx, cancel := context.WithTimeout(ctx, heartbeatTimeout)
				_, err := d.cdp.SendContext(hbCtx, "Browser.getVersion", nil)
				timedOut := hbCtx.Err() == context.DeadlineExceeded
				cancel()

				if err == nil {
					continue
				}

				// Normal shutdown in progress — exit silently.
				if ctx.Err() != nil {
					return
				}

				// Heartbeat's own deadline expired — browser unresponsive.
				if timedOut {
					select {
					case disconnectCh <- context.DeadlineExceeded:
					default:
					}
					return
				}

				// Underlying websocket error from the CDP client.
				cdpErr := d.cdp.Err()
				if cdpErr == nil {
					cdpErr = err
				}
				select {
				case disconnectCh <- cdpErr:
				default:
				}
				return
			}
		}
	}()

	d.debugf(false, "Heartbeat started (interval=%s, timeout=%s)", heartbeatInterval, heartbeatTimeout)
}
