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

	// Mark session as navigating BEFORE sending command.
	// Close any existing navigation channel first (handles rapid navigation).
	if oldCh, loaded := d.navigating.LoadAndDelete(activeID); loaded {
		d.debugf(false, "navigate: closing old navigating channel for session %s", activeID)
		close(oldCh.(chan struct{}))
	}
	navDoneCh := make(chan struct{})
	d.navigating.Store(activeID, navDoneCh)
	d.debugf(false, "navigate: created navigating channel for session %s", activeID)

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

	// If wait requested, wait for page load
	if params.Wait {
		timeout := cdp.DefaultTimeout
		if params.Timeout > 0 {
			timeout = time.Duration(params.Timeout) * time.Second
		}
		d.debugf(false, "navigate: waiting for page load (timeout=%v)", timeout)
		if err := d.waitForLoadEvent(activeID, timeout); err != nil {
			return ipc.ErrorResponse(err.Error())
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

	// Mark session as navigating BEFORE sending command.
	if oldCh, loaded := d.navigating.LoadAndDelete(activeID); loaded {
		d.debugf(false, "reload: closing old navigating channel for session %s", activeID)
		close(oldCh.(chan struct{}))
	}
	navDoneCh := make(chan struct{})
	d.navigating.Store(activeID, navDoneCh)
	d.debugf(false, "reload: created navigating channel for session %s", activeID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := d.sendToSession(ctx, activeID, "Page.reload", map[string]any{
		"ignoreCache": params.IgnoreCache,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("reload failed: %v", err))
	}

	// If wait requested, wait for page load
	if params.Wait {
		timeout := cdp.DefaultTimeout
		if params.Timeout > 0 {
			timeout = time.Duration(params.Timeout) * time.Second
		}
		d.debugf(false, "reload: waiting for page load (timeout=%v)", timeout)
		if err := d.waitForLoadEvent(activeID, timeout); err != nil {
			return ipc.ErrorResponse(err.Error())
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

	// Mark session as navigating BEFORE sending command.
	if oldCh, loaded := d.navigating.LoadAndDelete(activeID); loaded {
		d.debugf(debug, "navigateHistory: closing old navigating channel for session %s", activeID)
		close(oldCh.(chan struct{}))
	}
	navDoneCh := make(chan struct{})
	d.navigating.Store(activeID, navDoneCh)
	d.debugf(debug, "navigateHistory: created navigating channel for session %s", activeID)

	// If wait requested, register frame navigation waiter BEFORE sending command to avoid race
	var frameNavCh chan *frameNavigatedInfo
	if params.Wait {
		frameNavCh = make(chan *frameNavigatedInfo, 1)
		d.navWaiters.Store(activeID, frameNavCh)
		d.debugf(debug, "navigateHistory: registered frame navigation waiter before sending command")
	}

	// Navigate to history entry
	_, err = d.sendToSession(ctx, activeID, "Page.navigateToHistoryEntry", map[string]any{
		"entryId": history.Entries[targetIndex].ID,
	})
	if err != nil {
		// Clean up waiter on error
		if params.Wait {
			d.navWaiters.Delete(activeID)
		}
		return ipc.ErrorResponse(fmt.Sprintf("failed to navigate history: %v", err))
	}

	// If wait requested, wait for frame navigation (not loadEventFired, which doesn't fire for BFCache)
	if params.Wait {
		timeout := cdp.DefaultTimeout
		if params.Timeout > 0 {
			timeout = time.Duration(params.Timeout) * time.Second
		}
		d.debugf(debug, "navigateHistory: waiting for frame navigation (timeout=%v)", timeout)

		defer d.navWaiters.Delete(activeID)

		targetURL := history.Entries[targetIndex].URL
		select {
		case info := <-frameNavCh:
			return ipc.SuccessResponse(ipc.NavigateData{
				URL:   targetURL,
				Title: info.Title,
			})
		case <-time.After(timeout):
			return ipc.ErrorResponse(fmt.Sprintf("timeout waiting for navigation to %s", targetURL))
		}
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

// handleReadyPageLoad waits for the page to finish loading (existing behavior).
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

	// Page not yet loaded, wait for loadEventFired
	if err := d.waitForLoadEvent(sessionID, timeout); err != nil {
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

// ensureNetworkEnabled ensures the Network domain is enabled for the session.
func (d *Daemon) ensureNetworkEnabled(sessionID string) error {
	// Check if already enabled
	if _, loaded := d.networkEnabled.Load(sessionID); loaded {
		return nil
	}

	// Enable Network domain
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := d.sendToSession(ctx, sessionID, "Network.enable", nil)
	if err != nil {
		return fmt.Errorf("failed to enable Network domain: %v", err)
	}

	d.networkEnabled.Store(sessionID, true)
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
// Creates and registers a waiter channel internally.
func (d *Daemon) waitForLoadEvent(sessionID string, timeout time.Duration) error {
	// Check if navigation already completed (navigating channel was closed/deleted)
	// This handles race condition where fast navigations (e.g., from cache) complete
	// before we register the waiter.
	if _, navigating := d.navigating.Load(sessionID); !navigating {
		d.debugf(false, "waitForLoadEvent: navigation already complete for session %s", sessionID)
		return nil
	}

	ch := make(chan struct{}, 1)
	d.loadWaiters.Store(sessionID, ch)
	defer d.loadWaiters.Delete(sessionID)

	// Check again after registering waiter (double-check pattern)
	if _, navigating := d.navigating.Load(sessionID); !navigating {
		d.debugf(false, "waitForLoadEvent: navigation completed while registering waiter for session %s", sessionID)
		return nil
	}

	select {
	case <-ch:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for page load")
	}
}

