package ipc

import "encoding/json"

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
	Running bool   `json:"running"`
	URL     string `json:"url,omitempty"`
	Title   string `json:"title,omitempty"`
	PID     int    `json:"pid,omitempty"`
}

// ConsoleEntry represents a console log entry.
type ConsoleEntry struct {
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
	RequestID    string            `json:"requestId"`
	URL          string            `json:"url"`
	Method       string            `json:"method"`
	Status       int               `json:"status,omitempty"`
	StatusText   string            `json:"statusText,omitempty"`
	Type         string            `json:"type,omitempty"`
	MimeType     string            `json:"mimeType,omitempty"`
	RequestTime  int64             `json:"requestTime"`
	ResponseTime int64             `json:"responseTime,omitempty"`
	Duration     float64           `json:"duration,omitempty"`
	Size         int64             `json:"size,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	Body         string            `json:"body,omitempty"`
	Error        string            `json:"error,omitempty"`
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

// SuccessResponse creates a successful response with the given data.
func SuccessResponse(data any) Response {
	var raw json.RawMessage
	if data != nil {
		raw, _ = json.Marshal(data)
	}
	return Response{OK: true, Data: raw}
}

// ErrorResponse creates an error response with the given message.
func ErrorResponse(msg string) Response {
	return Response{OK: false, Error: msg}
}
