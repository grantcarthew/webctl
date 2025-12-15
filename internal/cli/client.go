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
	return executor.NewIPCExecutor()
}

func (f defaultFactory) IsDaemonRunning() bool {
	return ipc.IsDaemonRunning()
}

// execFactory is the package-level factory, replaceable for testing.
var execFactory ExecutorFactory = defaultFactory{}

// SetExecutorFactory sets the executor factory (for testing).
func SetExecutorFactory(f ExecutorFactory) {
	execFactory = f
}

// ResetExecutorFactory resets to the default factory.
func ResetExecutorFactory() {
	execFactory = defaultFactory{}
}
