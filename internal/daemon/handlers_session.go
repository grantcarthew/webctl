package daemon

import (
	"encoding/json"
	"fmt"

	"github.com/grantcarthew/webctl/internal/ipc"
)

// handleClear clears the specified buffer.
func (d *Daemon) handleClear(target string) ipc.Response {
	switch target {
	case "console":
		d.consoleBuf.Clear()
	case "network":
		d.networkBuf.Clear()
		_ = clearBodiesDir()
	case "", "all":
		d.consoleBuf.Clear()
		d.networkBuf.Clear()
		_ = clearBodiesDir()
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
		Error:    "no active tab - use 'webctl tab switch <query>' to select",
		Sessions: sessions,
	}

	raw, _ := json.Marshal(data)
	return ipc.Response{OK: false, Error: data.Error, Data: raw}
}
