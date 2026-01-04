package executor

import "github.com/grantcarthew/webctl/internal/ipc"

// IPCExecutor executes commands via Unix socket IPC.
type IPCExecutor struct {
	client *ipc.Client
	debug  bool
}

// NewIPCExecutor creates a new IPC executor connected to the daemon.
func NewIPCExecutor() (*IPCExecutor, error) {
	client, err := ipc.Dial()
	if err != nil {
		return nil, err
	}
	return &IPCExecutor{client: client, debug: false}, nil
}

// NewIPCExecutorPath creates a new IPC executor connected to a specific socket path.
func NewIPCExecutorPath(socketPath string) (*IPCExecutor, error) {
	client, err := ipc.DialPath(socketPath)
	if err != nil {
		return nil, err
	}
	return &IPCExecutor{client: client, debug: false}, nil
}

// NewIPCExecutorWithDebug creates a new IPC executor with debug flag.
func NewIPCExecutorWithDebug(debug bool) (*IPCExecutor, error) {
	client, err := ipc.Dial()
	if err != nil {
		return nil, err
	}
	return &IPCExecutor{client: client, debug: debug}, nil
}

// Execute sends a request via IPC and returns the response.
// Automatically sets the Debug flag on the request based on executor config.
func (e *IPCExecutor) Execute(req ipc.Request) (ipc.Response, error) {
	req.Debug = e.debug
	return e.client.Send(req)
}

// Close closes the IPC connection.
func (e *IPCExecutor) Close() error {
	return e.client.Close()
}
