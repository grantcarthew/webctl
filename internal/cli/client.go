package cli

import (
	"github.com/grantcarthew/webctl/internal/executor"
	"github.com/grantcarthew/webctl/internal/ipc"
)

// ExecutorFactory creates executors and checks daemon status.
type ExecutorFactory interface {
	NewExecutor() (executor.Executor, error)
	IsDaemonRunning() bool
}

// defaultFactory uses IPC executor.
type defaultFactory struct{}

func (f defaultFactory) NewExecutor() (executor.Executor, error) {
	return executor.NewIPCExecutorWithDebug(Debug)
}

func (f defaultFactory) IsDaemonRunning() bool {
	return ipc.IsDaemonRunning()
}

// DirectExecutorFactory creates direct executors for REPL use.
// It calls the daemon handler directly without IPC.
type DirectExecutorFactory struct {
	handler ipc.Handler
}

// NewDirectExecutorFactory creates a factory that uses direct execution.
func NewDirectExecutorFactory(handler ipc.Handler) *DirectExecutorFactory {
	return &DirectExecutorFactory{handler: handler}
}

func (f *DirectExecutorFactory) NewExecutor() (executor.Executor, error) {
	return executor.NewDirectExecutor(f.handler), nil
}

func (f *DirectExecutorFactory) IsDaemonRunning() bool {
	return true // Always true for direct execution (REPL runs inside daemon)
}

// execFactory is the package-level factory, replaceable for testing.
var execFactory ExecutorFactory = defaultFactory{}

// SetExecutorFactory sets the executor factory (for testing or REPL).
func SetExecutorFactory(f ExecutorFactory) {
	execFactory = f
}

// ResetExecutorFactory resets to the default factory.
func ResetExecutorFactory() {
	execFactory = defaultFactory{}
}
