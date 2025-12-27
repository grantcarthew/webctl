package daemon

import (
	"encoding/json"
	"fmt"

	"github.com/grantcarthew/webctl/internal/ipc"
)

// handleTarget lists sessions or switches to a specific session.
func (d *Daemon) handleTarget(query string) ipc.Response {
	// Check if browser is connected (fail-fast if not)
	if ok, resp := d.requireBrowser(); !ok {
		return resp
	}

	// If no query, list all sessions
	if query == "" {
		return ipc.SuccessResponse(ipc.TargetData{
			ActiveSession: d.sessions.ActiveID(),
			Sessions:      d.sessions.All(),
		})
	}

	// Try to find matching session
	matches := d.sessions.FindByQuery(query)

	if len(matches) == 0 {
		return ipc.ErrorResponse(fmt.Sprintf("no session matches query: %s", query))
	}

	if len(matches) > 1 {
		// Ambiguous match
		data := struct {
			Error   string            `json:"error"`
			Matches []ipc.PageSession `json:"matches"`
		}{
			Error:   fmt.Sprintf("ambiguous query '%s', matches multiple sessions", query),
			Matches: matches,
		}
		raw, _ := json.Marshal(data)
		return ipc.Response{OK: false, Error: data.Error, Data: raw}
	}

	// Single match - switch to it
	if !d.sessions.SetActive(matches[0].ID) {
		return ipc.ErrorResponse("failed to set active session")
	}

	return ipc.SuccessResponse(ipc.TargetData{
		ActiveSession: matches[0].ID,
		Sessions:      d.sessions.All(),
	})
}

// handleClear clears the specified buffer.
func (d *Daemon) handleClear(target string) ipc.Response {
	switch target {
	case "console":
		d.consoleBuf.Clear()
	case "network":
		d.networkBuf.Clear()
		clearBodiesDir()
	case "", "all":
		d.consoleBuf.Clear()
		d.networkBuf.Clear()
		clearBodiesDir()
	default:
		return ipc.ErrorResponse(fmt.Sprintf("unknown clear target: %s", target))
	}
	return ipc.SuccessResponse(nil)
}

// noActiveSessionError returns an error response with available sessions.
func (d *Daemon) noActiveSessionError() ipc.Response {
	sessions := d.sessions.All()
	if len(sessions) == 0 {
		return ipc.ErrorResponse("no active session - no pages available")
	}

	// Return error with session list so user can select
	data := struct {
		Error    string            `json:"error"`
		Sessions []ipc.PageSession `json:"sessions"`
	}{
		Error:    "no active session - use 'webctl target <id>' to select",
		Sessions: sessions,
	}

	raw, _ := json.Marshal(data)
	return ipc.Response{OK: false, Error: data.Error, Data: raw}
}
