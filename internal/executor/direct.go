package executor

import "github.com/grantcarthew/webctl/internal/ipc"

// DirectExecutor executes commands by calling the handler directly.
// Used by the REPL to avoid IPC round-trip.
type DirectExecutor struct {
	handler ipc.Handler
}

// NewDirectExecutor creates a new direct executor with the given handler.
func NewDirectExecutor(handler ipc.Handler) *DirectExecutor {
	return &DirectExecutor{handler: handler}
}

// Execute calls the handler directly and returns the response.
func (e *DirectExecutor) Execute(req ipc.Request) (ipc.Response, error) {
	return e.handler(req), nil
}

// Close is a no-op for direct executor.
func (e *DirectExecutor) Close() error {
	return nil
}
