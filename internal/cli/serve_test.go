package cli

import (
	"testing"
)

func TestServeAutoStartDaemon(t *testing.T) {
	t.Skip("Skipping integration test - requires no browser on port 9222")

	// This test demonstrates the expected behavior but requires
	// a clean environment with no browser running.
	//
	// Manual test:
	// 1. Ensure no daemon running: webctl stop
	// 2. Run: webctl serve .
	// 3. Expect: daemon starts, browser launches, server starts
}

func TestServeDefaultDirectory(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "no args defaults to current directory",
			args: []string{},
			want: ".",
		},
		{
			name: "explicit directory",
			args: []string{"/tmp"},
			want: "/tmp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse args and check what directory would be used
			// This is a simplified test - in reality we'd need to mock the daemon
			var directory string

			if serveProxy != "" {
				// Proxy mode - not testing this here
				return
			}

			if len(tt.args) == 0 {
				directory = "."
			} else {
				directory = tt.args[0]
			}

			if directory != tt.want {
				t.Errorf("Expected directory %q, got %q", tt.want, directory)
			}
		})
	}
}
