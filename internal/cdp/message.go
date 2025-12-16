package cdp

import (
	"encoding/json"
	"fmt"
)

// Request represents a CDP command request.
type Request struct {
	ID        int64       `json:"id"`
	Method    string      `json:"method"`
	Params    interface{} `json:"params,omitempty"`
	SessionID string      `json:"sessionId,omitempty"`
}

// Response represents a CDP command response.
type Response struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

// Event represents a CDP event notification.
type Event struct {
	Method    string          `json:"method"`
	Params    json.RawMessage `json:"params"`
	SessionID string          `json:"sessionId,omitempty"`
}

// Error represents a CDP protocol error.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Data != "" {
		return fmt.Sprintf("cdp error %d: %s (%s)", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("cdp error %d: %s", e.Code, e.Message)
}

// message is used internally to determine message type during parsing.
type message struct {
	ID        int64           `json:"id,omitempty"`
	Method    string          `json:"method,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *Error          `json:"error,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
}

// parseMessage parses a raw CDP message and returns either a Response or Event.
// Returns (response, nil, nil) for command responses.
// Returns (nil, event, nil) for events.
// Returns (nil, nil, error) for parse errors.
func parseMessage(data []byte) (*Response, *Event, error) {
	var msg message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, nil, fmt.Errorf("failed to parse CDP message: %w", err)
	}

	// Messages with an ID are responses to commands
	if msg.ID != 0 {
		return &Response{
			ID:     msg.ID,
			Result: msg.Result,
			Error:  msg.Error,
		}, nil, nil
	}

	// Messages with a method but no ID are events
	if msg.Method != "" {
		return nil, &Event{
			Method:    msg.Method,
			Params:    msg.Params,
			SessionID: msg.SessionID,
		}, nil
	}

	return nil, nil, fmt.Errorf("unknown CDP message format: %s", string(data))
}
