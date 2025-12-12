// Package cdp provides a minimal Chrome DevTools Protocol client.
package cdp

import (
	"context"

	"github.com/coder/websocket"
)

// Conn defines the interface for a WebSocket connection.
// This abstraction enables testing with mock connections.
type Conn interface {
	// Read reads a message from the connection.
	// Returns message type, payload, and any error.
	Read(ctx context.Context) (websocket.MessageType, []byte, error)

	// Write writes a message to the connection.
	Write(ctx context.Context, typ websocket.MessageType, p []byte) error

	// Close closes the connection with a status code and reason.
	Close(code websocket.StatusCode, reason string) error
}
