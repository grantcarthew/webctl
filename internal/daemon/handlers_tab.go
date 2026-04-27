package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
)

// tabWaiterTimeout bounds how long handleTab waits for an attach/detach event
// to be observed by SessionManager after sending a CDP request.
const tabWaiterTimeout = 10 * time.Second

// handleTab dispatches "tab" sub-actions: list, switch, new, close.
func (d *Daemon) handleTab(req ipc.Request) ipc.Response {
	if ok, resp := d.requireBrowser(); !ok {
		return resp
	}

	var params ipc.TabParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("invalid tab parameters: %v", err))
		}
	}

	switch params.Action {
	case "", "list":
		return d.handleTabList()
	case "switch":
		return d.handleTabSwitch(params.Query)
	case "new":
		return d.handleTabNew(params.URL)
	case "close":
		return d.handleTabClose(params.Query)
	default:
		return ipc.ErrorResponse(fmt.Sprintf("unknown tab action: %s", params.Action))
	}
}

// handleTabList returns all tabs.
func (d *Daemon) handleTabList() ipc.Response {
	return ipc.SuccessResponse(ipc.TabData{
		ActiveSession: d.sessions.ActiveID(),
		Sessions:      d.sessions.All(),
	})
}

// handleTabSwitch sets the active session and foregrounds the tab.
func (d *Daemon) handleTabSwitch(query string) ipc.Response {
	if query == "" {
		return ipc.ErrorResponse("query is required for tab switch")
	}

	matches := d.sessions.FindByQuery(query)
	if len(matches) == 0 {
		return ipc.ErrorResponse(fmt.Sprintf("no tab matches query: %s", query))
	}
	if len(matches) > 1 {
		return ambiguousTabError(query, matches)
	}

	sessionID := matches[0].ID
	if !d.sessions.SetActive(sessionID) {
		return ipc.ErrorResponse("failed to set active tab")
	}

	// Foreground the tab in the browser via Target.activateTarget.
	targetID := d.sessions.TargetID(sessionID)
	if targetID != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := d.cdp.SendContext(ctx, "Target.activateTarget", map[string]any{
			"targetId": targetID,
		}); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("failed to activate tab: %v", err))
		}
	}

	// Refresh REPL prompt so it reflects the new active session immediately.
	if d.repl != nil {
		d.repl.refreshPrompt()
	}

	return ipc.SuccessResponse(ipc.TabData{
		ActiveSession: d.sessions.ActiveID(),
		Sessions:      d.sessions.All(),
	})
}

// handleTabNew creates a new tab and waits for it to be registered.
func (d *Daemon) handleTabNew(url string) ipc.Response {
	if url == "" {
		url = "about:blank"
	}

	// Send Target.createTarget. newWindow:false ensures it opens in the existing window.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := d.cdp.SendContext(ctx, "Target.createTarget", map[string]any{
		"url":       url,
		"newWindow": false,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to create tab: %v", err))
	}

	var createResp struct {
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(result, &createResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse createTarget response: %v", err))
	}
	if createResp.TargetID == "" {
		return ipc.ErrorResponse("createTarget returned empty targetId")
	}

	// Register a one-shot buffered waiter keyed by targetID BEFORE checking
	// SessionManager so we don't miss the attach event if it has not yet fired.
	waiter := make(chan struct{}, 1)
	d.tabAttachWaiters.Store(createResp.TargetID, waiter)
	defer d.tabAttachWaiters.Delete(createResp.TargetID)

	// Check whether the attach event already landed before we registered.
	session := d.sessions.GetByTargetID(createResp.TargetID)
	if session == nil {
		select {
		case <-waiter:
			session = d.sessions.GetByTargetID(createResp.TargetID)
		case <-time.After(tabWaiterTimeout):
			return ipc.ErrorResponse("timeout waiting for new tab to attach")
		}
	}

	if session == nil {
		return ipc.ErrorResponse("new tab attach event observed but session not found")
	}

	// Make the new tab the active session. CDP foregrounds the new tab by default,
	// so no explicit Target.activateTarget is required.
	d.sessions.SetActive(session.ID)

	if d.repl != nil {
		d.repl.refreshPrompt()
	}

	return ipc.SuccessResponse(ipc.NewTabData{
		ID:    session.ID,
		URL:   session.URL,
		Title: session.Title,
	})
}

// handleTabClose closes the tab matching query, or the active tab if query is empty.
func (d *Daemon) handleTabClose(query string) ipc.Response {
	var sessionID string

	if query == "" {
		sessionID = d.sessions.ActiveID()
		if sessionID == "" {
			return ipc.ErrorResponse("no active tab")
		}
	} else {
		matches := d.sessions.FindByQuery(query)
		if len(matches) == 0 {
			return ipc.ErrorResponse(fmt.Sprintf("no tab matches query: %s", query))
		}
		if len(matches) > 1 {
			return ambiguousTabError(query, matches)
		}
		sessionID = matches[0].ID
	}

	// Last-tab guard runs before the CDP call.
	if d.sessions.Count() <= 1 {
		return ipc.ErrorResponse("cannot close the last tab; use 'webctl stop' to shut down the browser")
	}

	targetID := d.sessions.TargetID(sessionID)
	if targetID == "" {
		return ipc.ErrorResponse("internal error: targetID not found for session")
	}

	wasActive := d.sessions.ActiveID() == sessionID

	// Register detach waiter BEFORE sending the close, keyed by sessionID.
	waiter := make(chan struct{}, 1)
	d.tabDetachWaiters.Store(sessionID, waiter)
	defer d.tabDetachWaiters.Delete(sessionID)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := d.cdp.SendContext(ctx, "Target.closeTarget", map[string]any{
		"targetId": targetID,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to close tab: %v", err))
	}

	// CDP returns {success: bool}. Treat false or a malformed payload as an error.
	var closeResp struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(result, &closeResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("invalid closeTarget response: %v", err))
	}
	if !closeResp.Success {
		return ipc.ErrorResponse("browser refused to close tab")
	}

	// If the session is already gone (race), skip the wait.
	if d.sessions.Get(sessionID) != nil {
		select {
		case <-waiter:
		case <-time.After(tabWaiterTimeout):
			return ipc.ErrorResponse("timeout waiting for tab to close")
		}
	}

	// If the closed tab was active, foreground the new active.
	newActiveID := d.sessions.ActiveID()
	if wasActive && newActiveID != "" {
		newTargetID := d.sessions.TargetID(newActiveID)
		if newTargetID != "" {
			ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel2()
			_, _ = d.cdp.SendContext(ctx2, "Target.activateTarget", map[string]any{
				"targetId": newTargetID,
			})
		}
	}

	if d.repl != nil {
		d.repl.refreshPrompt()
	}

	return ipc.SuccessResponse(ipc.TabData{
		ActiveSession: newActiveID,
		Sessions:      d.sessions.All(),
	})
}

// ambiguousTabError builds an ambiguous-query error response with the candidate matches.
func ambiguousTabError(query string, matches []ipc.PageSession) ipc.Response {
	msg := fmt.Sprintf("ambiguous query '%s', matches multiple tabs", query)
	raw, _ := json.Marshal(struct {
		Error   string            `json:"error"`
		Matches []ipc.PageSession `json:"matches"`
	}{
		Error:   msg,
		Matches: matches,
	})
	return ipc.Response{OK: false, Error: msg, Data: raw}
}
