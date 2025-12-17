package ipc

import (
	"encoding/json"
	"log"
)

// CommandExecutor executes CLI commands with arguments.
// Returns true if the command was recognized, false otherwise.
// Used by the REPL to execute commands via Cobra.
type CommandExecutor func(args []string) (recognized bool, err error)

// Request represents a command sent from the CLI to the daemon.
type Request struct {
	Cmd    string          `json:"cmd"`
	Target string          `json:"target,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Response represents a response sent from the daemon to the CLI.
type Response struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// StatusData is the response data for the "status" command.
type StatusData struct {
	Running       bool         `json:"running"`
	PID           int          `json:"pid,omitempty"`
	ActiveSession *PageSession `json:"activeSession,omitempty"`
	Sessions      []PageSession `json:"sessions,omitempty"`
	// Deprecated: use ActiveSession.URL instead
	URL string `json:"url,omitempty"`
	// Deprecated: use ActiveSession.Title instead
	Title string `json:"title,omitempty"`
}

// ConsoleEntry represents a console log entry.
type ConsoleEntry struct {
	SessionID string   `json:"sessionId,omitempty"`
	Type      string   `json:"type"`
	Text      string   `json:"text"`
	Args      []string `json:"args,omitempty"`
	Timestamp int64    `json:"timestamp"`
	URL       string   `json:"url,omitempty"`
	Line      int      `json:"line,omitempty"`
	Column    int      `json:"column,omitempty"`
}

// NetworkEntry represents a network request/response entry.
type NetworkEntry struct {
	SessionID       string            `json:"sessionId,omitempty"`
	RequestID       string            `json:"requestId"`
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	Type            string            `json:"type,omitempty"`
	Status          int               `json:"status,omitempty"`
	StatusText      string            `json:"statusText,omitempty"`
	MimeType        string            `json:"mimeType,omitempty"`
	RequestTime     int64             `json:"requestTime"`
	ResponseTime    int64             `json:"responseTime,omitempty"`
	Duration        float64           `json:"duration,omitempty"`
	Size            int64             `json:"size,omitempty"`
	RequestHeaders  map[string]string `json:"requestHeaders,omitempty"`
	ResponseHeaders map[string]string `json:"responseHeaders,omitempty"`
	Body            string            `json:"body,omitempty"`
	BodyTruncated   bool              `json:"bodyTruncated,omitempty"`
	BodyPath        string            `json:"bodyPath,omitempty"`
	Failed          bool              `json:"failed"`
	Error           string            `json:"error,omitempty"`
}

// ConsoleData is the response data for the "console" command.
type ConsoleData struct {
	Entries []ConsoleEntry `json:"entries"`
	Count   int            `json:"count"`
}

// NetworkData is the response data for the "network" command.
type NetworkData struct {
	Entries []NetworkEntry `json:"entries"`
	Count   int            `json:"count"`
}

// PageSession represents an active CDP page session.
type PageSession struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Active bool   `json:"active,omitempty"`
}

// TargetData is the response data for the "target" command.
type TargetData struct {
	ActiveSession string        `json:"activeSession,omitempty"`
	Sessions      []PageSession `json:"sessions"`
}

// ScreenshotParams represents parameters for the "screenshot" command.
type ScreenshotParams struct {
	FullPage bool `json:"fullPage"`
}

// ScreenshotData is the response data for the "screenshot" command.
type ScreenshotData struct {
	Data []byte `json:"data"`
}

// HTMLParams represents parameters for the "html" command.
type HTMLParams struct {
	Selector string `json:"selector,omitempty"`
}

// HTMLData is the response data for the "html" command.
type HTMLData struct {
	HTML string `json:"html"`
}

// SuccessResponse creates a successful response with the given data.
func SuccessResponse(data any) Response {
	var raw json.RawMessage
	if data != nil {
		var err error
		raw, err = json.Marshal(data)
		if err != nil {
			log.Printf("ipc: failed to marshal response data: %v", err)
			return ErrorResponse("internal error: failed to marshal response")
		}
	}
	return Response{OK: true, Data: raw}
}

// ErrorResponse creates an error response with the given message.
func ErrorResponse(msg string) Response {
	return Response{OK: false, Error: msg}
}
