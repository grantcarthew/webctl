package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grantcarthew/webctl/internal/cdp"
	"github.com/grantcarthew/webctl/internal/ipc"
)

// handleNavigate navigates to a URL.
// Returns immediately after sending Page.navigate without waiting for frameNavigated.
// This avoids Chrome's internal blocking that occurs when waiting for navigation events.
func (d *Daemon) handleNavigate(req ipc.Request) ipc.Response {
	d.debugf(false, "handleNavigate called")

	// Check if browser is connected (fail-fast if not)
	if ok, resp := d.requireBrowser(); !ok {
		return resp
	}

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

	// Begin a navigation unconditionally, independent of --wait, so a later ready
	// default-mode call can detect this navigation as in-flight. begin atomically
	// cancels and replaces any prior navigation for the session.
	nav := d.navTracker.begin(activeID)
	d.debugf(false, "navigate: began navigation for session %s", activeID)

	// Send navigate command
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := d.sendToSession(ctx, activeID, "Page.navigate", map[string]any{
		"url": params.URL,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("navigation failed: %v", err))
	}

	// Check for navigation errors in response
	var navResp struct {
		ErrorText string `json:"errorText"`
		FrameID   string `json:"frameId"`
	}
	if err := json.Unmarshal(result, &navResp); err == nil && navResp.ErrorText != "" {
		return ipc.ErrorResponse(navResp.ErrorText)
	}

	// If wait requested, wait for full page load (Loaded milestone).
	if params.Wait {
		timeout := cdp.DefaultTimeout
		if params.Timeout > 0 {
			timeout = time.Duration(params.Timeout) * time.Second
		}
		d.debugf(false, "navigate: waiting for page load (timeout=%v)", timeout)

		switch awaitMilestone(nav.Loaded(), nav.Cancelled(), timeout) {
		case navCancelled:
			return cancelledNavResponse(nav, activeID)
		case navTimedOut:
			return ipc.ErrorResponse("timeout waiting for page load")
		}

		// Get title after page load
		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()
		title := d.getPageTitle(ctx2, activeID)
		return ipc.SuccessResponse(ipc.NavigateData{
			URL:   params.URL,
			Title: title,
		})
	}

	// Update session URL immediately so REPL prompt reflects the change
	d.sessions.Update(activeID, params.URL, "")

	// Return immediately - don't wait for frameNavigated.
	// Chrome's Page.navigate response includes the URL we navigated to.
	d.debugf(false, "navigate: returning immediately, frameId=%s", navResp.FrameID)
	return ipc.SuccessResponse(ipc.NavigateData{
		URL:   params.URL,
		Title: "", // Title not available until page loads
	})
}

// handleReload reloads the current page.
// Returns immediately after sending Page.reload command.
func (d *Daemon) handleReload(req ipc.Request) ipc.Response {
	// Check if browser is connected (fail-fast if not)
	if ok, resp := d.requireBrowser(); !ok {
		return resp
	}

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

	// Begin a navigation unconditionally so a later ready can detect the reload as
	// in-flight, independent of --wait.
	nav := d.navTracker.begin(activeID)
	d.debugf(false, "reload: began navigation for session %s", activeID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := d.sendToSession(ctx, activeID, "Page.reload", map[string]any{
		"ignoreCache": params.IgnoreCache,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("reload failed: %v", err))
	}

	// If wait requested, wait for full page load (Loaded milestone).
	if params.Wait {
		timeout := cdp.DefaultTimeout
		if params.Timeout > 0 {
			timeout = time.Duration(params.Timeout) * time.Second
		}
		d.debugf(false, "reload: waiting for page load (timeout=%v)", timeout)

		switch awaitMilestone(nav.Loaded(), nav.Cancelled(), timeout) {
		case navCancelled:
			return cancelledNavResponse(nav, activeID)
		case navTimedOut:
			return ipc.ErrorResponse("timeout waiting for page load")
		}

		// Get URL and title after page load
		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()
		title := d.getPageTitle(ctx2, activeID)
		// Get current URL from session
		session := d.sessions.Get(activeID)
		url := ""
		if session != nil {
			url = session.URL
		}
		return ipc.SuccessResponse(ipc.NavigateData{
			URL:   url,
			Title: title,
		})
	}

	// Get current URL from session for response
	session := d.sessions.Get(activeID)
	currentURL := ""
	if session != nil {
		currentURL = session.URL
	}

	// Return immediately - don't wait for frameNavigated
	// Session URL stays the same for reload, so no need to update
	d.debugf(false, "reload: returning immediately")
	return ipc.SuccessResponse(ipc.NavigateData{
		URL:   currentURL,
		Title: "", // Title not available until frameNavigated
	})
}

// handleBack navigates to the previous history entry.
func (d *Daemon) handleBack(req ipc.Request) ipc.Response {
	var params ipc.HistoryParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("invalid back parameters: %v", err))
		}
	}
	return d.navigateHistory(-1, params, req.Debug)
}

// handleForward navigates to the next history entry.
func (d *Daemon) handleForward(req ipc.Request) ipc.Response {
	var params ipc.HistoryParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("invalid forward parameters: %v", err))
		}
	}
	return d.navigateHistory(1, params, req.Debug)
}

// navigateHistory navigates forward or backward in history.
// Returns immediately after sending navigation command unless wait=true.
func (d *Daemon) navigateHistory(delta int, params ipc.HistoryParams, debug bool) ipc.Response {
	// Check if browser is connected (fail-fast if not)
	if ok, resp := d.requireBrowser(); !ok {
		return resp
	}

	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get navigation history
	result, err := d.sendToSession(ctx, activeID, "Page.getNavigationHistory", nil)
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

	// Begin a navigation unconditionally so a later ready can detect the history
	// navigation as in-flight, independent of --wait.
	nav := d.navTracker.begin(activeID)
	d.debugf(debug, "navigateHistory: began navigation for session %s", activeID)

	// Navigate to history entry
	_, err = d.sendToSession(ctx, activeID, "Page.navigateToHistoryEntry", map[string]any{
		"entryId": history.Entries[targetIndex].ID,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to navigate history: %v", err))
	}

	// If wait requested, wait for frame navigation (not loadEventFired, which
	// doesn't fire for BFCache), then resolve the title off the read loop.
	if params.Wait {
		timeout := cdp.DefaultTimeout
		if params.Timeout > 0 {
			timeout = time.Duration(params.Timeout) * time.Second
		}
		d.debugf(debug, "navigateHistory: waiting for frame navigation (timeout=%v)", timeout)

		targetURL := history.Entries[targetIndex].URL
		switch awaitMilestone(nav.FrameNavigated(), nav.Cancelled(), timeout) {
		case navCancelled:
			return cancelledNavResponse(nav, activeID)
		case navTimedOut:
			return ipc.ErrorResponse(fmt.Sprintf("timeout waiting for navigation to %s", targetURL))
		}

		// FrameNavigated has closed; report the requested history-entry URL to stay
		// consistent with navigate --wait and the non-wait history path, which both
		// return the requested URL rather than the resolved one. Resolve the title
		// off the read loop.
		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()
		title := d.getPageTitle(ctx2, activeID)
		return ipc.SuccessResponse(ipc.NavigateData{
			URL:   targetURL,
			Title: title,
		})
	}

	// Return immediately - don't wait for frameNavigated
	targetURL := history.Entries[targetIndex].URL

	// Update session URL immediately so REPL prompt reflects the change
	d.sessions.Update(activeID, targetURL, "")

	d.debugf(debug, "navigateHistory: returning immediately, target URL=%s", targetURL)
	return ipc.SuccessResponse(ipc.NavigateData{
		URL:   targetURL, // We know the target URL from history
		Title: "",        // Title not available until frameNavigated
	})
}

// handleReady waits for the page or application to be ready.
// Supports multiple modes: page load, selector, network idle, and eval.
func (d *Daemon) handleReady(req ipc.Request) ipc.Response {
	// Check if browser is connected (fail-fast if not)
	if ok, resp := d.requireBrowser(); !ok {
		return resp
	}

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

	timeout := cdp.DefaultTimeout
	if params.Timeout > 0 {
		timeout = time.Duration(params.Timeout) * time.Second
	}

	// Mode detection (order matters)
	if params.NetworkIdle {
		return d.handleReadyNetworkIdle(activeID, timeout)
	} else if params.Eval != "" {
		return d.handleReadyEval(activeID, params.Eval, timeout)
	} else if params.Selector != "" {
		return d.handleReadySelector(activeID, params.Selector, timeout)
	} else {
		// Default: page load mode
		return d.handleReadyPageLoad(activeID, timeout)
	}
}

// handleReadyPageLoad implements ready default mode: it returns immediately when
// document.readyState is already "complete", otherwise it waits for the current
// navigation (if any) to reach DOM-ready.
func (d *Daemon) handleReadyPageLoad(sessionID string, timeout time.Duration) ipc.Response {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// First check if page is already loaded via document.readyState
	result, err := d.sendToSession(ctx, sessionID, "Runtime.evaluate", map[string]any{
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

	// Page not yet loaded, wait for the navigation to reach DOM-ready
	if err := d.waitForDOMReady(sessionID, timeout); err != nil {
		return ipc.ErrorResponse(err.Error())
	}

	return ipc.SuccessResponse(nil)
}

// handleReadySelector waits for an element matching the CSS selector to appear.
func (d *Daemon) handleReadySelector(sessionID, selector string, timeout time.Duration) ipc.Response {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ipc.ErrorResponse(fmt.Sprintf("timeout waiting for: %s", selector))
		case <-ticker.C:
			// Try to find the element
			found, err := d.querySelector(ctx, sessionID, selector)
			if err != nil {
				// Continue polling on error (element might not exist yet)
				continue
			}
			if found {
				return ipc.SuccessResponse(nil)
			}
		}
	}
}

// handleReadyNetworkIdle waits for all pending network requests to complete.
func (d *Daemon) handleReadyNetworkIdle(sessionID string, timeout time.Duration) ipc.Response {
	// Ensure Network domain is enabled (needed for tracking requests)
	if err := d.ensureNetworkEnabled(sessionID); err != nil {
		return ipc.ErrorResponse(err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	idleThreshold := 500 * time.Millisecond
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	var idleStart time.Time

	for {
		select {
		case <-ctx.Done():
			return ipc.ErrorResponse("timeout waiting for network idle")
		case <-ticker.C:
			pending := d.getPendingRequestCount(sessionID)
			if pending == 0 {
				if idleStart.IsZero() {
					idleStart = time.Now()
				} else if time.Since(idleStart) >= idleThreshold {
					return ipc.SuccessResponse(nil)
				}
			} else {
				idleStart = time.Time{} // Reset
			}
		}
	}
}

// handleReadyEval waits for a JavaScript expression to evaluate to a truthy value.
func (d *Daemon) handleReadyEval(sessionID, expression string, timeout time.Duration) ipc.Response {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ipc.ErrorResponse(fmt.Sprintf("timeout waiting for: %s", expression))
		case <-ticker.C:
			result, err := d.sendToSession(ctx, sessionID, "Runtime.evaluate", map[string]any{
				"expression":    expression,
				"returnByValue": true,
			})
			if err != nil {
				// Continue polling on error (expression might fail initially)
				continue
			}

			var resp struct {
				Result struct {
					Value any `json:"value"`
				} `json:"result"`
			}
			if err := json.Unmarshal(result, &resp); err != nil {
				continue
			}

			// Check if truthy
			if isTruthy(resp.Result.Value) {
				return ipc.SuccessResponse(nil)
			}
		}
	}
}

// querySelector checks if an element matching the selector exists.
// Returns true if found, false if not found.
func (d *Daemon) querySelector(ctx context.Context, sessionID, selector string) (bool, error) {
	// Get document root
	docResult, err := d.sendToSession(ctx, sessionID, "DOM.getDocument", nil)
	if err != nil {
		return false, err
	}

	var docResp struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(docResult, &docResp); err != nil {
		return false, err
	}

	// Query for selector
	queryResult, err := d.sendToSession(ctx, sessionID, "DOM.querySelector", map[string]any{
		"nodeId":   docResp.Root.NodeID,
		"selector": selector,
	})
	if err != nil {
		return false, err
	}

	var queryResp struct {
		NodeID int `json:"nodeId"`
	}
	if err := json.Unmarshal(queryResult, &queryResp); err != nil {
		return false, err
	}

	// NodeID of 0 means element not found
	return queryResp.NodeID != 0, nil
}

// ensureNetworkEnabled ensures the Network domain is enabled for the session,
// at most once. It claims the enable, sends Network.enable outside the lock, and
// clears the claim on failure so a later caller can retry.
func (d *Daemon) ensureNetworkEnabled(sessionID string) error {
	if !d.sessions.ClaimNetworkEnable(sessionID) {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := d.sendToSession(ctx, sessionID, "Network.enable", nil); err != nil {
		d.sessions.ClearNetworkEnabled(sessionID)
		return fmt.Errorf("failed to enable Network domain: %v", err)
	}

	return nil
}

// getPendingRequestCount returns the number of pending network requests for the session.
// This is a simplified implementation that counts requests in the buffer.
// TODO: Implement proper request tracking with requestID mapping.
func (d *Daemon) getPendingRequestCount(sessionID string) int {
	// For now, check if there are any recent requests without responses
	// This is a simplified heuristic - proper implementation would track request/response pairs
	recentRequests := make(map[string]bool)

	// Get all network entries
	entries := d.networkBuf.All()

	// Scan the network buffer for recent activity
	for _, entry := range entries {
		// Only count entries for this session
		if entry.SessionID != sessionID {
			continue
		}

		// Check if this is a recent request (within last 5 seconds)
		age := time.Now().Unix() - entry.RequestTime/1000
		if age > 5 {
			continue // Skip old entries
		}

		// Track request by ID
		if entry.Type == "" { // This is a request entry
			recentRequests[entry.RequestID] = true
		}

		// If we've seen a response for this request, it's complete
		if entry.ResponseTime > 0 || entry.Failed {
			delete(recentRequests, entry.RequestID)
		}
	}

	return len(recentRequests)
}

// isTruthy checks if a value is truthy in JavaScript terms.
func isTruthy(value any) bool {
	if value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != ""
	case float64:
		return v != 0
	default:
		return true
	}
}

// getPageTitle retrieves the current page title via JavaScript.
func (d *Daemon) getPageTitle(ctx context.Context, sessionID string) string {
	result, err := d.sendToSession(ctx, sessionID, "Runtime.evaluate", map[string]any{
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

// waitForDOMReady blocks until the session's current navigation reaches DOM-ready
// (ready default mode), returning immediately when no navigation is in flight.
//
// Awaiting an already-reached DOM-ready milestone returns at once, so the legacy
// register-then-double-check dance is unnecessary. A superseding navigation
// re-binds the wait to the newer navigation rather than erroring, because ready's
// contract is to block until the page is ready and the page is now loading the
// newer URL. A detach returns an error naming the closed session. The overall
// timeout bounds the whole wait, including any re-binds.
func (d *Daemon) waitForDOMReady(sessionID string, timeout time.Duration) error {
	nav := d.navTracker.current(sessionID)
	if nav == nil {
		// No navigation in flight; ready has nothing to wait for.
		d.debugf(false, "waitForDOMReady: no navigation in flight for session %s", sessionID)
		return nil
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	deadline := timer.C
	for {
		// DOM-ready takes priority: if the page is ready, report success even when
		// the navigation was also superseded or its session is detaching.
		select {
		case <-nav.DOMReady():
			return nil
		default:
		}
		select {
		case <-nav.DOMReady():
			return nil
		case <-nav.Cancelled():
			if nav.CancelReason() == cancelDetached {
				return fmt.Errorf("session %s closed while waiting for page load", sessionID)
			}
			// Superseded: re-bind to the newer navigation and keep waiting. A nil
			// replacement means the navigation was cleared, which only happens on
			// detach, so report the session as closed rather than a false success.
			d.debugf(false, "waitForDOMReady: navigation superseded, re-binding for session %s", sessionID)
			nav = d.navTracker.current(sessionID)
			if nav == nil {
				return fmt.Errorf("session %s closed while waiting for page load", sessionID)
			}
		case <-deadline:
			return fmt.Errorf("timeout waiting for page load")
		}
	}
}

// navOutcome reports how awaitMilestone returned.
type navOutcome int

const (
	navReached   navOutcome = iota // the awaited milestone closed
	navCancelled                   // the navigation was cancelled (superseded or detached)
	navTimedOut                    // the timeout elapsed first
)

// errNavigationSuperseded is the uniform error the --wait navigation commands
// return when a newer navigation supersedes theirs.
const errNavigationSuperseded = "navigation superseded by a newer navigation"

// awaitMilestone blocks until the milestone closes, the navigation is cancelled,
// or the timeout elapses, reporting which happened. It is pure rendezvous logic so
// the consumers stay testable without a browser.
//
// The milestone takes priority: a navigation that reached its milestone before a
// superseding navigation cancelled it has succeeded, so report success rather than
// letting a plain select pick at random when both channels are closed.
func awaitMilestone(milestone, cancelled <-chan struct{}, timeout time.Duration) navOutcome {
	select {
	case <-milestone:
		return navReached
	default:
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-milestone:
		return navReached
	case <-cancelled:
		return navCancelled
	case <-timer.C:
		return navTimedOut
	}
}

// cancelledNavResponse maps a closed Cancelled milestone to the error a --wait
// command returns: the supersession message, or a message naming the closed
// session when the real cause was the session detaching.
func cancelledNavResponse(nav *Navigation, sessionID string) ipc.Response {
	if nav.CancelReason() == cancelDetached {
		return ipc.ErrorResponse(fmt.Sprintf("session %s closed during navigation", sessionID))
	}
	return ipc.ErrorResponse(errNavigationSuperseded)
}
