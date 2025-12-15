package executor

import "github.com/grantcarthew/webctl/internal/ipc"

// Executor executes commands and returns responses.
// Implementations handle the transport mechanism (IPC, TCP, direct call).
type Executor interface {
	Execute(req ipc.Request) (ipc.Response, error)
	Close() error
}
