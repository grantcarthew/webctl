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
	// Target events (browser-level, no sessionId)
	d.cdp.Subscribe("Target.targetCreated", func(evt cdp.Event) {
		d.handleTargetCreated(evt)
	})

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
			awaiting := entry.AwaitingRequestBody()
			d.networkBuf.Push(entry)
			d.debugf(false, "Network.requestWillBeSent: requestId=%s, url=%s, type=%s", entry.RequestID, entry.URL, entry.Type)
			// Body advertised but omitted from the event (exceeds maxPostDataSize):
			// fetch it off the read loop, like the response body in handleLoadingFinished.
			if awaiting {
				d.fetchRequestPostData(evt.SessionID, entry.RequestID)
			}
		}
	})

	d.cdp.Subscribe("Network.responseReceived", func(evt cdp.Event) {
		d.updateResponseEvent(evt)
		var params struct {
			RequestID string `json:"requestId"`
			Type      string `json:"type"`
		}
		if err := json.Unmarshal(evt.Params, &params); err == nil {
			d.debugf(false, "Network.responseReceived: requestId=%s, type=%s", params.RequestID, params.Type)
		}
	})

	d.cdp.Subscribe("Network.loadingFinished", func(evt cdp.Event) {
		d.handleLoadingFinished(evt)
		var params struct {
			RequestID string `json:"requestId"`
		}
		if err := json.Unmarshal(evt.Params, &params); err == nil {
			d.debugf(false, "Network.loadingFinished: requestId=%s", params.RequestID)
		}
	})

	d.cdp.Subscribe("Network.loadingFailed", func(evt cdp.Event) {
		d.handleLoadingFailed(evt)
		var params struct {
			RequestID string `json:"requestId"`
		}
		if err := json.Unmarshal(evt.Params, &params); err == nil {
			d.debugf(false, "Network.loadingFailed: requestId=%s", params.RequestID)
		}
	})

	// Page navigation events for navigation commands
	d.cdp.Subscribe("Page.frameNavigated", func(evt cdp.Event) {
		d.handleFrameNavigated(evt)
	})

	d.cdp.Subscribe("Page.loadEventFired", func(evt cdp.Event) {
		d.handleLoadEventFired(evt)
	})

	d.cdp.Subscribe("Page.domContentEventFired", func(evt cdp.Event) {
		d.handleDOMContentEventFired(evt)
	})

	// Debug: Additional Page events
	d.cdp.Subscribe("Page.frameStartedLoading", func(evt cdp.Event) {
		d.debugf(false, "Page.frameStartedLoading: sessionID=%s", evt.SessionID)
	})

	d.cdp.Subscribe("Page.frameStoppedLoading", func(evt cdp.Event) {
		d.debugf(false, "Page.frameStoppedLoading: sessionID=%s", evt.SessionID)
	})

	d.cdp.Subscribe("Page.lifecycleEvent", func(evt cdp.Event) {
		var params struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(evt.Params, &params); err == nil {
			d.debugf(false, "Page.lifecycleEvent: name=%s, sessionID=%s", params.Name, evt.SessionID)
		}
	})

	// Debug: Runtime execution context events
	d.cdp.Subscribe("Runtime.executionContextCreated", func(evt cdp.Event) {
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

	d.cdp.Subscribe("Runtime.executionContextDestroyed", func(evt cdp.Event) {
		var params struct {
			ExecutionContextID int `json:"executionContextId"`
		}
		if err := json.Unmarshal(evt.Params, &params); err == nil {
			d.debugf(false, "Runtime.executionContextDestroyed: contextId=%d", params.ExecutionContextID)
		}
	})

	d.cdp.Subscribe("Runtime.executionContextsCleared", func(evt cdp.Event) {
		d.debugf(false, "Runtime.executionContextsCleared")
	})

	// Debug: DOM events
	d.cdp.Subscribe("DOM.documentUpdated", func(evt cdp.Event) {
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

// networkMaxPostDataSize caps the request-body bytes Chrome includes inline in
// Network.requestWillBeSent. It is sourced from ipc.DefaultMaxBodySize so it
// matches the CLI default --max-body-size: a body that survives truncation
// arrives inline without an extra round trip, and larger bodies fall back to
// fetchRequestPostData. Without this cap set on Network.enable, Chrome omits
// postData entirely and only sets hasPostData.
const networkMaxPostDataSize = ipc.DefaultMaxBodySize

// networkEnableParams builds the Network.enable parameters shared by every
// enable site, so the inline post-data cap cannot drift between them.
func networkEnableParams() map[string]any {
	return map[string]any{"maxPostDataSize": networkMaxPostDataSize}
}

// parseRequestEvent parses a Network.requestWillBeSent event.
// Returns the entry and true on success, or zero value and false on parse error.
func (d *Daemon) parseRequestEvent(evt cdp.Event) (ipc.NetworkEntry, bool) {
	var params struct {
		RequestID string  `json:"requestId"`
		WallTime  float64 `json:"wallTime"` // Unix epoch in seconds
		Request   struct {
			URL         string            `json:"url"`
			Method      string            `json:"method"`
			Headers     map[string]string `json:"headers"`
			PostData    string            `json:"postData"`
			HasPostData bool              `json:"hasPostData"`
		} `json:"request"`
		Type      string `json:"type"`
		Initiator struct {
			Type       string `json:"type"`
			URL        string `json:"url"`
			LineNumber int    `json:"lineNumber"`
			Stack      *struct {
				CallFrames []struct {
					URL        string `json:"url"`
					LineNumber int    `json:"lineNumber"`
				} `json:"callFrames"`
			} `json:"stack"`
		} `json:"initiator"`
	}
	if err := json.Unmarshal(evt.Params, &params); err != nil {
		return ipc.NetworkEntry{}, false
	}

	entry := ipc.NetworkEntry{
		RequestID:      params.RequestID,
		URL:            params.Request.URL,
		Method:         params.Request.Method,
		Type:           params.Type,
		RequestTime:    int64(params.WallTime * 1000), // Convert seconds to milliseconds
		RequestHeaders: params.Request.Headers,
		RequestBody:    params.Request.PostData,
	}

	// Capture the initiator type plus a single source location. CDP carries the
	// location on the Initiator object itself for parser-initiated requests (the
	// common <img>/<script>/<link> case) and only in the stack for script
	// initiators, so read the Initiator's own url/lineNumber first and fall back
	// to the top stack frame. The nested StackTrace parent chain is dropped.
	if params.Initiator.Type != "" {
		init := &ipc.NetworkInitiator{Type: params.Initiator.Type}
		if params.Initiator.URL != "" {
			init.URL = params.Initiator.URL
			init.Line = params.Initiator.LineNumber
		} else if params.Initiator.Stack != nil && len(params.Initiator.Stack.CallFrames) > 0 {
			init.URL = params.Initiator.Stack.CallFrames[0].URL
			init.Line = params.Initiator.Stack.CallFrames[0].LineNumber
		}
		entry.Initiator = init
	}

	// hasPostData with no inline postData means the body exceeded maxPostDataSize
	// and must be fetched separately. Mark the entry so the fetch lands on it
	// rather than on a later redirect hop that reuses this requestId.
	if params.Request.HasPostData && params.Request.PostData == "" {
		entry.AwaitRequestBody()
	}

	return entry, true
}

// fetchRequestPostData retrieves a request body that was advertised but omitted
// from Network.requestWillBeSent and stores it on the awaiting entry.
//
// Like handleLoadingFinished, the CDP call runs on its own goroutine: a
// synchronous call inside an event handler would deadlock, because its response
// travels back through the read loop that is currently blocked in the handler.
//
// The body lands on the newest entry that still carries the awaiting marker for
// this requestId, not merely the newest entry sharing the id. A non-body redirect
// hop (for example a POST that 303-redirects to a GET) shares the requestId, is
// newer, and has an equally empty RequestBody, so matching on emptiness would
// misroute the body onto it. Matching on the marker prevents that theft.
func (d *Daemon) fetchRequestPostData(sessionID, requestID string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		clearMarker := func() {
			d.networkBuf.Update(func(entry *ipc.NetworkEntry) bool {
				if entry.RequestID == requestID && entry.AwaitingRequestBody() {
					entry.ClearAwaitingRequestBody()
					return true
				}
				return false
			})
		}

		result, err := d.cdp.SendToSession(ctx, sessionID, "Network.getRequestPostData", map[string]any{
			"requestId": requestID,
		})
		if err != nil {
			// The expected benign case is "No data found for resource with given
			// identifier" (nothing was sent). Other failures (timeout, closed
			// session, transport error) also land here; in every case we degrade
			// gracefully by clearing the marker, but log so the off-read-loop
			// fetch is diagnosable under --debug.
			d.debugf(false, "Network.getRequestPostData failed: requestId=%s, err=%v", requestID, err)
			clearMarker()
			return
		}

		var bodyResp struct {
			PostData string `json:"postData"`
		}
		if err := json.Unmarshal(result, &bodyResp); err != nil {
			d.debugf(false, "Network.getRequestPostData: failed to parse response: requestId=%s, err=%v", requestID, err)
			clearMarker()
			return
		}

		d.networkBuf.Update(func(entry *ipc.NetworkEntry) bool {
			if entry.RequestID == requestID && entry.AwaitingRequestBody() {
				entry.SetRequestBody(bodyResp.PostData)
				return true
			}
			return false
		})
	}()
}

// updateResponseEvent updates an existing network entry with response data.
func (d *Daemon) updateResponseEvent(evt cdp.Event) {
	var params struct {
		RequestID string `json:"requestId"`
		Response  struct {
			Status            int                `json:"status"`
			StatusText        string             `json:"statusText"`
			MimeType          string             `json:"mimeType"`
			Headers           map[string]string  `json:"headers"`
			RemoteIPAddress   string             `json:"remoteIPAddress"`
			RemotePort        int                `json:"remotePort"`
			Protocol          string             `json:"protocol"`
			FromDiskCache     bool               `json:"fromDiskCache"`
			FromServiceWorker bool               `json:"fromServiceWorker"`
			FromPrefetchCache bool               `json:"fromPrefetchCache"`
			ConnectionID      float64            `json:"connectionId"`
			SecurityState     string             `json:"securityState"`
			Timing            *cdpResourceTiming `json:"timing"`
		} `json:"response"`
	}
	if err := json.Unmarshal(evt.Params, &params); err != nil {
		return
	}

	timing := deriveNetworkTiming(params.Response.Timing)

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
			entry.RemoteIPAddress = params.Response.RemoteIPAddress
			entry.RemotePort = params.Response.RemotePort
			entry.Protocol = params.Response.Protocol
			entry.FromDiskCache = params.Response.FromDiskCache
			entry.FromServiceWorker = params.Response.FromServiceWorker
			entry.FromPrefetchCache = params.Response.FromPrefetchCache
			entry.ConnectionID = params.Response.ConnectionID
			entry.SecurityState = params.Response.SecurityState
			entry.Timing = timing
			entry.ResponseTime = responseTime
			if entry.RequestTime > 0 {
				entry.Duration = float64(entry.ResponseTime-entry.RequestTime) / 1000.0
			}
			return true // stop iteration
		}
		return false
	})
}

// cdpResourceTiming mirrors the subset of CDP's Network.ResourceTiming the
// daemon consumes. Offsets are milliseconds relative to a requestTime baseline;
// a negative value marks a phase boundary that did not occur.
type cdpResourceTiming struct {
	DNSStart          float64 `json:"dnsStart"`
	DNSEnd            float64 `json:"dnsEnd"`
	ConnectStart      float64 `json:"connectStart"`
	ConnectEnd        float64 `json:"connectEnd"`
	SSLStart          float64 `json:"sslStart"`
	SSLEnd            float64 `json:"sslEnd"`
	SendStart         float64 `json:"sendStart"`
	SendEnd           float64 `json:"sendEnd"`
	ReceiveHeadersEnd float64 `json:"receiveHeadersEnd"`
}

// deriveNetworkTiming converts the CDP ResourceTiming offsets into per-phase
// durations. A phase is reported only when both its start and end are present
// and ordered. Returns nil when no phase has a duration, so the entry omits an
// empty timing object.
func deriveNetworkTiming(t *cdpResourceTiming) *ipc.NetworkTiming {
	if t == nil {
		return nil
	}
	// The TLS handshake falls within the connect window, so the raw connect span
	// (connectStart..connectEnd) double-counts the TLS time. When a handshake
	// occurred, narrow connect to its TCP portion (connectStart..sslStart) and
	// report TLS separately, so the phases are disjoint and partition the time.
	connectMs := phaseDuration(t.ConnectStart, t.ConnectEnd)
	if phaseDuration(t.SSLStart, t.SSLEnd) > 0 {
		connectMs = phaseDuration(t.ConnectStart, t.SSLStart)
	}
	timing := ipc.NetworkTiming{
		DNSMs:     phaseDuration(t.DNSStart, t.DNSEnd),
		ConnectMs: connectMs,
		TLSMs:     phaseDuration(t.SSLStart, t.SSLEnd),
		SendMs:    phaseDuration(t.SendStart, t.SendEnd),
		WaitMs:    phaseDuration(t.SendEnd, t.ReceiveHeadersEnd),
	}
	if timing == (ipc.NetworkTiming{}) {
		return nil
	}
	return &timing
}

// phaseDuration returns end-start when both offsets are present (non-negative)
// and ordered, or 0 when the phase did not occur. CDP marks an absent phase
// boundary with a negative offset.
func phaseDuration(start, end float64) float64 {
	if start < 0 || end < 0 || end < start {
		return 0
	}
	return end - start
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
						entry.ResponseBodyPath = bodyPath
					}
				} else {
					// Store text body directly
					if bodyResp.Base64Encoded {
						// Decode base64 for text content
						decoded, err := base64.StdEncoding.DecodeString(bodyResp.Body)
						if err == nil {
							entry.ResponseBody = string(decoded)
						}
					} else {
						entry.ResponseBody = bodyResp.Body
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

	// Check if we've already attached to this target (prevent double-attach)
	if !d.attaches.mark(params.TargetInfo.TargetID) {
		d.debugf(false, "Target.targetCreated: already attached to targetID=%q, skipping", params.TargetInfo.TargetID)
		return
	}

	// Attach asynchronously to avoid blocking the event loop
	// (Critical: targetCreated events can fire while waiting for setDiscoverTargets response)
	go func() {
		// Manually attach to the target with flatten:true.
		// This is critical - without flatten:true, CDP responses may be queued until networkIdle.
		result, err := d.cdp.Send("Target.attachToTarget", map[string]any{
			"targetId": params.TargetInfo.TargetID,
			"flatten":  true,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nwarning: failed to attach to target %q: %v\n", params.TargetInfo.TargetID, err)
			// Clear the mark on failure so we can retry
			d.attaches.clear(params.TargetInfo.TargetID)
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

	// Add to session manager. Add signals any registered tab-new waiter for this
	// targetID under its lock, closing the attach rendezvous.
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
			fmt.Fprintf(os.Stderr, "\nwarning: failed to enable domains for session: %v\n", err)
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

	// Cancel any in-flight navigation with the detach reason so a blocked ready or
	// --wait consumer wakes with the session-closed outcome instead of timing out.
	// This is distinct from the tab-close waiter signalled by Remove below: the two
	// wake different consumers, so their relative order does not matter.
	d.navTracker.clear(params.SessionID)

	// Drop the attach-dedup mark for this target. Resolve the targetID before Remove
	// deletes the session; targetIDs are never reused, so clearing here cannot cause a
	// later double-attach and it keeps the attach set from growing for the daemon's life.
	if targetID := d.sessions.TargetID(params.SessionID); targetID != "" {
		d.attaches.clear(targetID)
	}

	// Remove from session manager. Remove signals any registered tab-close waiter
	// for this sessionID under its lock, closing the detach rendezvous.
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
// Closes the current navigation's FrameNavigated milestone.
//
// This event is critical for history navigation (back/forward) because Chrome's
// BFCache (Back/Forward Cache) optimization prevents Page.loadEventFired from
// firing when navigating to cached pages. Page.frameNavigated DOES fire for all
// navigation types, making it the reliable choice for history navigation waiting.
//
// The page title is intentionally not resolved here: a synchronous CDP call on
// the read-loop goroutine would stall event processing. The consumer resolves the
// title on its own goroutine after waking on the milestone.
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

	if nav := d.navTracker.current(evt.SessionID); nav != nil {
		nav.markFrameNavigated()
	}
}

// handleLoadEventFired processes Page.loadEventFired events, marking the current
// navigation Loaded (which also closes its DOM-ready milestone).
func (d *Daemon) handleLoadEventFired(evt cdp.Event) {
	d.debugf(false, "Page.loadEventFired: sessionID=%s", evt.SessionID)

	if nav := d.navTracker.current(evt.SessionID); nav != nil {
		nav.markLoaded()
	}
}

// handleDOMContentEventFired processes Page.domContentEventFired events, marking
// the current navigation DOM-ready. This fires before loadEventFired, letting
// ready default mode and DOM operations proceed once the DOM is ready without
// waiting for all resources (images, scripts, ads) to finish loading.
func (d *Daemon) handleDOMContentEventFired(evt cdp.Event) {
	d.debugf(false, "Page.domContentEventFired: sessionID=%s", evt.SessionID)

	if nav := d.navTracker.current(evt.SessionID); nav != nil {
		nav.markDOMReady()
	}
}
