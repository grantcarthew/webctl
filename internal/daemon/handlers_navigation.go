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
	d.debugf("handleNavigate called")
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
		d.debugf("navigate: closing old navigating channel for session %s", activeID)
		close(oldCh.(chan struct{}))
	}
	navDoneCh := make(chan struct{})
	d.navigating.Store(activeID, navDoneCh)
	d.debugf("navigate: created navigating channel for session %s", activeID)

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
		FrameID   string `json:"frameId"`
	}
	if err := json.Unmarshal(result, &navResp); err == nil && navResp.ErrorText != "" {
		return ipc.ErrorResponse(navResp.ErrorText)
	}

	// If wait requested, wait for page load
	if params.Wait {
		timeout := cdp.DefaultTimeout
		if params.Timeout > 0 {
			timeout = time.Duration(params.Timeout) * time.Millisecond
		}
		d.debugf("navigate: waiting for page load (timeout=%v)", timeout)
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
	d.debugf("navigate: returning immediately, frameId=%s", navResp.FrameID)
	return ipc.SuccessResponse(ipc.NavigateData{
		URL:   params.URL,
		Title: "", // Title not available until page loads
	})
}

// handleReload reloads the current page.
// Returns immediately after sending Page.reload command.
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

	// Mark session as navigating BEFORE sending command.
	if oldCh, loaded := d.navigating.LoadAndDelete(activeID); loaded {
		d.debugf("reload: closing old navigating channel for session %s", activeID)
		close(oldCh.(chan struct{}))
	}
	navDoneCh := make(chan struct{})
	d.navigating.Store(activeID, navDoneCh)
	d.debugf("reload: created navigating channel for session %s", activeID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := d.cdp.SendToSession(ctx, activeID, "Page.reload", map[string]any{
		"ignoreCache": params.IgnoreCache,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("reload failed: %v", err))
	}

	// If wait requested, wait for page load
	if params.Wait {
		timeout := cdp.DefaultTimeout
		if params.Timeout > 0 {
			timeout = time.Duration(params.Timeout) * time.Millisecond
		}
		d.debugf("reload: waiting for page load (timeout=%v)", timeout)
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
	d.debugf("reload: returning immediately")
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
	return d.navigateHistory(-1, params)
}

// handleForward navigates to the next history entry.
func (d *Daemon) handleForward(req ipc.Request) ipc.Response {
	var params ipc.HistoryParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("invalid forward parameters: %v", err))
		}
	}
	return d.navigateHistory(1, params)
}

// navigateHistory navigates forward or backward in history.
// Returns immediately after sending navigation command unless wait=true.
func (d *Daemon) navigateHistory(delta int, params ipc.HistoryParams) ipc.Response {
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

	// Mark session as navigating BEFORE sending command.
	if oldCh, loaded := d.navigating.LoadAndDelete(activeID); loaded {
		d.debugf("navigateHistory: closing old navigating channel for session %s", activeID)
		close(oldCh.(chan struct{}))
	}
	navDoneCh := make(chan struct{})
	d.navigating.Store(activeID, navDoneCh)
	d.debugf("navigateHistory: created navigating channel for session %s", activeID)

	// Navigate to history entry
	_, err = d.cdp.SendToSession(ctx, activeID, "Page.navigateToHistoryEntry", map[string]any{
		"entryId": history.Entries[targetIndex].ID,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to navigate history: %v", err))
	}

	// If wait requested, wait for page load
	if params.Wait {
		timeout := cdp.DefaultTimeout
		if params.Timeout > 0 {
			timeout = time.Duration(params.Timeout) * time.Millisecond
		}
		d.debugf("navigateHistory: waiting for page load (timeout=%v)", timeout)
		if err := d.waitForLoadEvent(activeID, timeout); err != nil {
			return ipc.ErrorResponse(err.Error())
		}
		// Get title after page load
		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()
		title := d.getPageTitle(ctx2, activeID)
		targetURL := history.Entries[targetIndex].URL
		return ipc.SuccessResponse(ipc.NavigateData{
			URL:   targetURL,
			Title: title,
		})
	}

	// Return immediately - don't wait for frameNavigated
	targetURL := history.Entries[targetIndex].URL

	// Update session URL immediately so REPL prompt reflects the change
	d.sessions.Update(activeID, targetURL, "")

	d.debugf("navigateHistory: returning immediately, target URL=%s", targetURL)
	return ipc.SuccessResponse(ipc.NavigateData{
		URL:   targetURL, // We know the target URL from history
		Title: "",        // Title not available until frameNavigated
	})
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

	timeout := cdp.DefaultTimeout
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
