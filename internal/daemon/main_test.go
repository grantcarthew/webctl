package daemon

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		// Ignore net/http persistent connection goroutines from integration tests.
		// These are expected when navigating to external URLs like example.com.
		// Use IgnoreAnyFunction to match goroutines with these functions anywhere in the stack,
		// not just at the top (since they're often blocked on I/O).
		goleak.IgnoreAnyFunction("net/http.(*persistConn).readLoop"),
		goleak.IgnoreAnyFunction("net/http.(*persistConn).writeLoop"),
	)
}
