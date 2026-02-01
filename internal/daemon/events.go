package daemon

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/grantcarthew/webctl/internal/cdp"
	"github.com/grantcarthew/webctl/internal/ipc"
)

// subscribeEvents subscribes to CDP events and buffers them.
func (d *Daemon) subscribeEvents() {
	cdpClient := d.getCDP()
	if cdpClient == nil {
		d.debugf(false, "CDP client not available for event subscriptions")
		return
	}

	// Target events (browser-level, no sessionId)
	cdpClient.Subscribe("Target.targetCreated", func(evt cdp.Event) {
		d.handleTargetCreated(evt)
	})

	cdpClient.Subscribe("Target.attachedToTarget", func(evt cdp.Event) {
		d.handleTargetAttached(evt)
	})

	cdpClient.Subscribe("Target.detachedFromTarget", func(evt cdp.Event) {
		d.handleTargetDetached(evt)
	})

	cdpClient.Subscribe("Target.targetInfoChanged", func(evt cdp.Event) {
		d.handleTargetInfoChanged(evt)
	})

	// Console events (include sessionId)
	cdpClient.Subscribe("Runtime.consoleAPICalled", func(evt cdp.Event) {
		if entry, ok := d.parseConsoleEvent(evt); ok {
			entry.SessionID = evt.SessionID
			d.consoleBuf.Push(entry)
		}
	})

	cdpClient.Subscribe("Runtime.exceptionThrown", func(evt cdp.Event) {
		if entry, ok := d.parseExceptionEvent(evt); ok {
			entry.SessionID = evt.SessionID
			d.consoleBuf.Push(entry)
		}
	})

	// Network events (include sessionId)
	cdpClient.Subscribe("Network.requestWillBeSent", func(evt cdp.Event) {
		if entry, ok := d.parseRequestEvent(evt); ok {
			entry.SessionID = evt.SessionID
			d.networkBuf.Push(entry)
			d.debugf(false, "Network.requestWillBeSent: requestId=%s, url=%s, type=%s", entry.RequestID, entry.URL, entry.Type)
		}
	})

	cdpClient.Subscribe("Network.responseReceived", func(evt cdp.Event) {
		d.updateResponseEvent(evt)
		var params struct {
			RequestID string `json:"requestId"`
			Type      string `json:"type"`
		}
		if err := json.Unmarshal(evt.Params, &params); err == nil {
			d.debugf(false, "Network.responseReceived: requestId=%s, type=%s", params.RequestID, params.Type)
		}
	})

	cdpClient.Subscribe("Network.loadingFinished", func(evt cdp.Event) {
		d.handleLoadingFinished(evt)
		var params struct {
			RequestID string `json:"requestId"`
		}
		if err := json.Unmarshal(evt.Params, &params); err == nil {
			d.debugf(false, "Network.loadingFinished: requestId=%s", params.RequestID)
		}
	})

	cdpClient.Subscribe("Network.loadingFailed", func(evt cdp.Event) {
		d.handleLoadingFailed(evt)
		var params struct {
			RequestID string `json:"requestId"`
		}
		if err := json.Unmarshal(evt.Params, &params); err == nil {
			d.debugf(false, "Network.loadingFailed: requestId=%s", params.RequestID)
		}
	})

	// Page navigation events for navigation commands
	cdpClient.Subscribe("Page.frameNavigated", func(evt cdp.Event) {
		d.handleFrameNavigated(evt)
	})

	cdpClient.Subscribe("Page.loadEventFired", func(evt cdp.Event) {
		d.handleLoadEventFired(evt)
	})

	cdpClient.Subscribe("Page.domContentEventFired", func(evt cdp.Event) {
		d.handleDOMContentEventFired(evt)
	})

	// Debug: Additional Page events
	cdpClient.Subscribe("Page.frameStartedLoading", func(evt cdp.Event) {
		d.debugf(false, "Page.frameStartedLoading: sessionID=%s", evt.SessionID)
	})

	cdpClient.Subscribe("Page.frameStoppedLoading", func(evt cdp.Event) {
		d.debugf(false, "Page.frameStoppedLoading: sessionID=%s", evt.SessionID)
	})

	cdpClient.Subscribe("Page.lifecycleEvent", func(evt cdp.Event) {
		var params struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(evt.Params, &params); err == nil {
			d.debugf(false, "Page.lifecycleEvent: name=%s, sessionID=%s", params.Name, evt.SessionID)
		}
	})

	// Debug: Runtime execution context events
	cdpClient.Subscribe("Runtime.executionContextCreated", func(evt cdp.Event) {
		var params struct {
			Context struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"context"`
		}
		if err := json.Unmarshal(evt.Params, &params); err == nil {
			d.debugf(false, "Runtime.executionContextCreated: contextId=%d, name=%s", params.Context.ID, params.Context.Name)
		}
	})

	cdpClient.Subscribe("Runtime.executionContextDestroyed", func(evt cdp.Event) {
		var params struct {
			ExecutionContextID int `json:"executionContextId"`
		}
		if err := json.Unmarshal(evt.Params, &params); err == nil {
			d.debugf(false, "Runtime.executionContextDestroyed: contextId=%d", params.ExecutionContextID)
		}
	})

	cdpClient.Subscribe("Runtime.executionContextsCleared", func(evt cdp.Event) {
		d.debugf(false, "Runtime.executionContextsCleared")
	})

	// Debug: DOM events
	cdpClient.Subscribe("DOM.documentUpdated", func(evt cdp.Event) {
		d.debugf(false, "DOM.documentUpdated: sessionID=%s", evt.SessionID)
	})
}

// parseConsoleEvent parses a Runtime.consoleAPICalled event.
// Returns the entry and true on success, or zero value and false on parse error.
func (d *Daemon) parseConsoleEvent(evt cdp.Event) (ipc.ConsoleEntry, bool) {
	var params struct {
		Type      string  `json:"type"`
		Timestamp float64 `json:"timestamp"`
		Args      []struct {
			Type  string `json:"type"`
			Value any    `json:"value"`
		} `json:"args"`
		StackTrace *struct {
			CallFrames []struct {
				URL          string `json:"url"`
				LineNumber   int    `json:"lineNumber"`
				ColumnNumber int    `json:"columnNumber"`
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
		Timestamp        float64 `json:"timestamp"`
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
		RequestID         string `json:"requestId"`
		EncodedDataLength int64  `json:"encodedDataLength"`
	}
	if err := json.Unmarshal(evt.Params, &params); err != nil {
		return
	}

	// Find the entry to get MIME type (quick, non-blocking)
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

	// Fetch the response body asynchronously to avoid blocking the read loop.
	// CRITICAL: CDP calls block waiting for a response that comes through
	// the same read loop. Synchronous CDP calls in event handlers cause deadlock.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cdpClient := d.getCDP()
		if cdpClient == nil {
			return
		}

		result, err := cdpClient.SendToSession(ctx, evt.SessionID, "Network.getResponseBody", map[string]any{
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
	}()
}

// handleLoadingFailed handles the Network.loadingFailed event.
// Marks the request as failed with error details.
func (d *Daemon) handleLoadingFailed(evt cdp.Event) {
	var params struct {
		RequestID string `json:"requestId"`
		ErrorText string `json:"errorText"`
		Canceled  bool   `json:"canceled"`
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

// handleTargetCreated handles Target.targetCreated event.
// Manually attaches to page targets using Target.attachToTarget with flatten:true.
func (d *Daemon) handleTargetCreated(evt cdp.Event) {
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

	// Only attach to page targets
	if params.TargetInfo.Type != "page" {
		return
	}

	d.debugf(false, "Target.targetCreated: targetID=%q, type=%q, url=%q",
		params.TargetInfo.TargetID, params.TargetInfo.Type, params.TargetInfo.URL)

	// Skip if reconnection is in progress (maps are being cleared)
	if d.isReconnecting() {
		d.debugf(false, "Target.targetCreated: reconnection in progress, skipping targetID=%q", params.TargetInfo.TargetID)
		return
	}

	// Check if we've already attached to this target (prevent double-attach)
	if _, alreadyAttached := d.attachedTargets.LoadOrStore(params.TargetInfo.TargetID, true); alreadyAttached {
		d.debugf(false, "Target.targetCreated: already attached to targetID=%q, skipping", params.TargetInfo.TargetID)
		return
	}

	// Attach asynchronously to avoid blocking the event loop
	// (Critical: targetCreated events can fire while waiting for setDiscoverTargets response)
	go func() {
		cdpClient := d.getCDP()
		if cdpClient == nil {
			d.debugf(false, "CDP client not available for target attachment")
			d.attachedTargets.Delete(params.TargetInfo.TargetID)
			return
		}

		// Manually attach to the target with flatten:true.
		// This is critical - without flatten:true, CDP responses may be queued until networkIdle.
		result, err := cdpClient.Send("Target.attachToTarget", map[string]any{
			"targetId": params.TargetInfo.TargetID,
			"flatten":  true,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to attach to target %q: %v\n", params.TargetInfo.TargetID, err)
			// Remove from attachedTargets on failure so we can retry
			d.attachedTargets.Delete(params.TargetInfo.TargetID)
			return
		}

		// The result contains the sessionId, but we'll receive Target.attachedToTarget event anyway
		// which will handle session setup via handleTargetAttached
		d.debugf(false, "Target.attachToTarget result for targetID=%q: %s", params.TargetInfo.TargetID, string(result))
	}()
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

	d.debugf(false, "Target.attachedToTarget: sessionID=%q, targetID=%q, url=%q",
		params.SessionID, params.TargetInfo.TargetID, params.TargetInfo.URL)

	// Add to session manager
	d.sessions.Add(
		params.SessionID,
		params.TargetInfo.TargetID,
		params.TargetInfo.URL,
		params.TargetInfo.Title,
	)

	// Refresh REPL prompt to show new session
	if d.repl != nil {
		d.repl.refreshPrompt()
	}

	// Enable domains for this session (async to not block event loop)
	go func() {
		startEnable := time.Now()
		if err := d.enableDomainsForSession(params.SessionID); err != nil {
			// Log error but don't fail - session is still tracked
			fmt.Fprintf(os.Stderr, "warning: failed to enable domains for session: %v\n", err)
		}
		d.debugf(false, "enableDomainsForSession completed in %v for session %q", time.Since(startEnable), params.SessionID)
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

	d.debugf(false, "Target.detachedFromTarget: sessionID=%q", params.SessionID)

	// Remove from session manager
	newActive, changed := d.sessions.Remove(params.SessionID)
	d.debugf(false, "Session removed: newActiveID=%q, activeChanged=%v", newActive, changed)

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

	d.debugf(false, "Target.targetInfoChanged: targetID=%q, url=%q",
		params.TargetInfo.TargetID, params.TargetInfo.URL)

	// Update session by target ID
	d.sessions.UpdateByTargetID(
		params.TargetInfo.TargetID,
		params.TargetInfo.URL,
		params.TargetInfo.Title,
	)

	// Refresh REPL prompt to show updated URL
	if d.repl != nil {
		d.repl.refreshPrompt()
	}
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

// handleFrameNavigated processes Page.frameNavigated events.
// Signals any waiting navigation operations.
//
// This event is critical for history navigation (back/forward) because Chrome's
// BFCache (Back/Forward Cache) optimization prevents Page.loadEventFired from
// firing when navigating to cached pages. Page.frameNavigated DOES fire for all
// navigation types, making it the reliable choice for history navigation waiting.
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
// Signals any waiting ready operations and marks navigation as complete.
func (d *Daemon) handleLoadEventFired(evt cdp.Event) {
	d.debugf(false, "Page.loadEventFired: sessionID=%s", evt.SessionID)

	// Signal ready waiters
	if ch, ok := d.loadWaiters.LoadAndDelete(evt.SessionID); ok {
		d.debugf(false, "Page.loadEventFired: signaling ready waiter for session %s", evt.SessionID)
		waiter := ch.(chan struct{})
		select {
		case waiter <- struct{}{}:
		default:
		}
	}

	// Mark navigation as complete by closing the navigating channel
	if ch, ok := d.navigating.LoadAndDelete(evt.SessionID); ok {
		d.debugf(false, "Page.loadEventFired: closing navigating channel for session %s", evt.SessionID)
		close(ch.(chan struct{}))
	}
}

// handleDOMContentEventFired processes Page.domContentEventFired events.
// Marks navigation as complete for DOM operations - fires earlier than loadEventFired.
// This allows html/eval commands to proceed once DOM is ready, without waiting
// for all resources (images, scripts, ads) to finish loading.
func (d *Daemon) handleDOMContentEventFired(evt cdp.Event) {
	d.debugf(false, "Page.domContentEventFired: sessionID=%s", evt.SessionID)

	// Mark navigation as complete by closing the navigating channel
	// This fires before loadEventFired, allowing DOM operations to proceed sooner
	if ch, ok := d.navigating.LoadAndDelete(evt.SessionID); ok {
		d.debugf(false, "Page.domContentEventFired: closing navigating channel for session %s", evt.SessionID)
		close(ch.(chan struct{}))
	}
}
