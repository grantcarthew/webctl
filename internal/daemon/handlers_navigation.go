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

	// Return immediately - don't wait for frameNavigated.
	// Chrome's Page.navigate response includes the URL we navigated to.
	d.debugf("navigate: returning immediately, frameId=%s", navResp.FrameID)
	return ipc.SuccessResponse(ipc.NavigateData{
		URL:   params.URL,
		Title: "", // Title not available until page loads
	})
}

// handleReload reloads the current page.
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
		close(oldCh.(chan struct{}))
	}
	navDoneCh := make(chan struct{})
	d.navigating.Store(activeID, navDoneCh)

	// Set up waiter before sending reload command
	ch := make(chan *frameNavigatedInfo, 1)
	d.navWaiters.Store(activeID, ch)
	defer d.navWaiters.Delete(activeID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := d.cdp.SendToSession(ctx, activeID, "Page.reload", map[string]any{
		"ignoreCache": params.IgnoreCache,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("reload failed: %v", err))
	}

	// Wait for frameNavigated event
	select {
	case info := <-ch:
		return ipc.SuccessResponse(ipc.NavigateData{
			URL:   info.URL,
			Title: info.Title,
		})
	case <-time.After(cdp.DefaultTimeout):
		return ipc.ErrorResponse("timeout waiting for reload")
	}
}

// handleBack navigates to the previous history entry.
func (d *Daemon) handleBack() ipc.Response {
	return d.navigateHistory(-1)
}

// handleForward navigates to the next history entry.
func (d *Daemon) handleForward() ipc.Response {
	return d.navigateHistory(1)
}

// navigateHistory navigates forward or backward in history.
func (d *Daemon) navigateHistory(delta int) ipc.Response {
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
		close(oldCh.(chan struct{}))
	}
	navDoneCh := make(chan struct{})
	d.navigating.Store(activeID, navDoneCh)

	// Set up waiter before navigating
	ch := make(chan *frameNavigatedInfo, 1)
	d.navWaiters.Store(activeID, ch)
	defer d.navWaiters.Delete(activeID)

	// Navigate to history entry
	_, err = d.cdp.SendToSession(ctx, activeID, "Page.navigateToHistoryEntry", map[string]any{
		"entryId": history.Entries[targetIndex].ID,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to navigate history: %v", err))
	}

	// Wait for frameNavigated event
	select {
	case info := <-ch:
		return ipc.SuccessResponse(ipc.NavigateData{
			URL:   info.URL,
			Title: info.Title,
		})
	case <-time.After(cdp.DefaultTimeout):
		return ipc.ErrorResponse("timeout waiting for history navigation")
	}
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
