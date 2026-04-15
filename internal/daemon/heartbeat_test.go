package daemon

import (
	"context"
	"fmt"
	"testing"

	"github.com/coder/websocket"
)

func TestClassifyDisconnect(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "nil error",
			err:  nil,
			want: "browser disconnected",
		},
		{
			name: "normal closure 1000",
			err:  websocket.CloseError{Code: websocket.StatusNormalClosure, Reason: ""},
			want: "browser closed normally",
		},
		{
			name: "going away 1001",
			err:  websocket.CloseError{Code: websocket.StatusGoingAway, Reason: ""},
			want: "browser closed normally",
		},
		{
			name: "context deadline exceeded",
			err:  context.DeadlineExceeded,
			want: "browser unresponsive (heartbeat timeout)",
		},
		{
			name: "wrapped deadline exceeded",
			err:  fmt.Errorf("heartbeat failed: %w", context.DeadlineExceeded),
			want: "browser unresponsive (heartbeat timeout)",
		},
		{
			name: "EOF",
			err:  fmt.Errorf("unexpected EOF"),
			want: "browser connection lost",
		},
		{
			name: "connection reset",
			err:  fmt.Errorf("read tcp: connection reset by peer"),
			want: "browser connection lost",
		},
		{
			name: "unknown error",
			err:  fmt.Errorf("something went wrong"),
			want: "browser connection lost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyDisconnect(tt.err)
			if got != tt.want {
				t.Errorf("classifyDisconnect(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}
