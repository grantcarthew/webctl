package cli

import "github.com/grantcarthew/webctl/internal/ipc"

// IPCClient is the interface for IPC operations.
type IPCClient interface {
	Send(req ipc.Request) (ipc.Response, error)
	SendCmd(cmd string) (ipc.Response, error)
	Close() error
}

// Dialer creates IPC clients and checks daemon status.
type Dialer interface {
	Dial() (IPCClient, error)
	IsDaemonRunning() bool
}

// defaultDialer uses the real ipc package.
type defaultDialer struct{}

func (d defaultDialer) Dial() (IPCClient, error) {
	return ipc.Dial()
}

func (d defaultDialer) IsDaemonRunning() bool {
	return ipc.IsDaemonRunning()
}

// dialer is the package-level dialer, replaceable for testing.
var dialer Dialer = defaultDialer{}
