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

// NavigateParams represents parameters for the "navigate" command.
type NavigateParams struct {
	URL     string `json:"url"`
	Wait    bool   `json:"wait"`    // wait for page load completion
	Timeout int    `json:"timeout"` // timeout in milliseconds (when wait=true)
}

// NavigateData is the response data for the "navigate" command.
type NavigateData struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// ReloadParams represents parameters for the "reload" command.
type ReloadParams struct {
	IgnoreCache bool `json:"ignoreCache"`
	Wait        bool `json:"wait"`    // wait for page load completion
	Timeout     int  `json:"timeout"` // timeout in milliseconds (when wait=true)
}

// HistoryParams represents parameters for the "back" and "forward" commands.
type HistoryParams struct {
	Wait    bool `json:"wait"`    // wait for page load completion
	Timeout int  `json:"timeout"` // timeout in milliseconds (when wait=true)
}

// ReadyParams represents parameters for the "ready" command.
type ReadyParams struct {
	Timeout int `json:"timeout"` // timeout in milliseconds
}

// ClickParams represents parameters for the "click" command.
type ClickParams struct {
	Selector string `json:"selector"`
}

// FocusParams represents parameters for the "focus" command.
type FocusParams struct {
	Selector string `json:"selector"`
}

// TypeParams represents parameters for the "type" command.
type TypeParams struct {
	Selector string `json:"selector,omitempty"`
	Text     string `json:"text"`
	Key      string `json:"key,omitempty"`
	Clear    bool   `json:"clear,omitempty"`
}

// KeyParams represents parameters for the "key" command.
type KeyParams struct {
	Key   string `json:"key"`
	Ctrl  bool   `json:"ctrl,omitempty"`
	Alt   bool   `json:"alt,omitempty"`
	Shift bool   `json:"shift,omitempty"`
	Meta  bool   `json:"meta,omitempty"`
}

// SelectParams represents parameters for the "select" command.
type SelectParams struct {
	Selector string `json:"selector"`
	Value    string `json:"value"`
}

// ScrollParams represents parameters for the "scroll" command.
type ScrollParams struct {
	Selector string `json:"selector,omitempty"`
	ToX      int    `json:"toX,omitempty"`
	ToY      int    `json:"toY,omitempty"`
	ByX      int    `json:"byX,omitempty"`
	ByY      int    `json:"byY,omitempty"`
	Mode     string `json:"mode"` // "element", "to", or "by"
}

// EvalParams represents parameters for the "eval" command.
type EvalParams struct {
	Expression string `json:"expression"`
	Timeout    int    `json:"timeout,omitempty"` // timeout in milliseconds
}

// EvalData is the response data for the "eval" command.
type EvalData struct {
	Value    any  `json:"value,omitempty"`
	HasValue bool `json:"hasValue,omitempty"`
}

// CookiesParams represents parameters for the "cookies" command.
type CookiesParams struct {
	Action   string `json:"action"` // "list", "set", or "delete"
	Name     string `json:"name,omitempty"`
	Value    string `json:"value,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Path     string `json:"path,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	HTTPOnly bool   `json:"httpOnly,omitempty"`
	MaxAge   int    `json:"maxAge,omitempty"`   // seconds
	SameSite string `json:"sameSite,omitempty"` // "Strict", "Lax", or "None"
}

// Cookie represents a browser cookie with all CDP attributes.
type Cookie struct {
	Name         string  `json:"name"`
	Value        string  `json:"value"`
	Domain       string  `json:"domain"`
	Path         string  `json:"path"`
	Expires      float64 `json:"expires"`
	Size         int     `json:"size"`
	HTTPOnly     bool    `json:"httpOnly"`
	Secure       bool    `json:"secure"`
	Session      bool    `json:"session"`
	SameSite     string  `json:"sameSite,omitempty"`
	Priority     string  `json:"priority,omitempty"`
	SameParty    bool    `json:"sameParty,omitempty"`
	SourceScheme string  `json:"sourceScheme,omitempty"`
	SourcePort   int     `json:"sourcePort,omitempty"`
}

// CookiesData is the response data for the "cookies" command.
type CookiesData struct {
	Cookies []Cookie `json:"cookies,omitempty"`
	Count   int      `json:"count,omitempty"`
	// For ambiguous delete errors
	Matches []Cookie `json:"matches,omitempty"`
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
