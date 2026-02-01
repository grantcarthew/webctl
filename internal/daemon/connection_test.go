package daemon

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestConnectionState_String(t *testing.T) {
	tests := []struct {
		state ConnectionState
		want  string
	}{
		{StateConnected, "connected"},
		{StateReconnecting, "reconnecting"},
		{StateDisconnected, "disconnected"},
		{ConnectionState(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.state.String()
		if got != tt.want {
			t.Errorf("ConnectionState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestDisconnectReason_String(t *testing.T) {
	tests := []struct {
		reason DisconnectReason
		want   string
	}{
		{ReasonGraceful, "graceful"},
		{ReasonAbnormal, "abnormal"},
		{ReasonUnknown, "unknown"},
		{DisconnectReason(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.reason.String()
		if got != tt.want {
			t.Errorf("DisconnectReason(%d).String() = %q, want %q", tt.reason, got, tt.want)
		}
	}
}

func TestConnectionManager_StateTransitions(t *testing.T) {
	m := newConnectionManager()

	// Initial state
	if m.State() != StateConnected {
		t.Errorf("initial state = %v, want Connected", m.State())
	}

	// Transition to reconnecting
	m.SetReconnecting(errors.New("test error"))
	if m.State() != StateReconnecting {
		t.Errorf("state after SetReconnecting = %v, want Reconnecting", m.State())
	}
	if m.ReconnectCount() != 1 {
		t.Errorf("reconnect count = %d, want 1", m.ReconnectCount())
	}

	// Another reconnect attempt
	m.SetReconnecting(errors.New("another error"))
	if m.ReconnectCount() != 2 {
		t.Errorf("reconnect count = %d, want 2", m.ReconnectCount())
	}

	// Transition back to connected
	m.SetConnected()
	if m.State() != StateConnected {
		t.Errorf("state after SetConnected = %v, want Connected", m.State())
	}
	if m.ReconnectCount() != 0 {
		t.Errorf("reconnect count after SetConnected = %d, want 0", m.ReconnectCount())
	}

	// Transition to disconnected
	m.SetDisconnected(errors.New("final error"))
	if m.State() != StateDisconnected {
		t.Errorf("state after SetDisconnected = %v, want Disconnected", m.State())
	}
}

func TestConnectionManager_Info(t *testing.T) {
	m := newConnectionManager()

	// Record a heartbeat
	m.RecordHeartbeat()

	info := m.Info()
	if info.State != StateConnected {
		t.Errorf("info.State = %v, want Connected", info.State)
	}
	if info.StateString != "connected" {
		t.Errorf("info.StateString = %q, want connected", info.StateString)
	}
	if info.LastHeartbeat.IsZero() {
		t.Error("info.LastHeartbeat should not be zero")
	}
	if info.LastError != "" {
		t.Errorf("info.LastError = %q, want empty", info.LastError)
	}

	// Set an error
	m.SetReconnecting(errors.New("test error"))
	info = m.Info()
	if info.LastError != "test error" {
		t.Errorf("info.LastError = %q, want 'test error'", info.LastError)
	}
}

func TestConnectionManager_MaxAttemptsReached(t *testing.T) {
	m := newConnectionManager()
	m.maxAttempts = 3

	if m.MaxAttemptsReached() {
		t.Error("MaxAttemptsReached should be false initially")
	}

	for i := 0; i < 3; i++ {
		m.SetReconnecting(errors.New("error"))
	}

	if !m.MaxAttemptsReached() {
		t.Error("MaxAttemptsReached should be true after 3 attempts")
	}
}

func TestConnectionManager_NextDelay(t *testing.T) {
	m := newConnectionManager()
	m.initialDelay = 1 * time.Second
	m.maxDelay = 30 * time.Second
	m.backoffFactor = 2.0
	m.jitterPercent = 0 // Disable jitter for predictable tests

	// Before any attempts (count=0, treated as 1): 1s
	delay := m.NextDelay()
	if delay < 1*time.Second || delay > 1100*time.Millisecond {
		t.Errorf("delay before attempts = %v, want ~1s", delay)
	}

	// After first attempt (count=1): 1s
	m.SetReconnecting(errors.New("error"))
	delay = m.NextDelay()
	if delay < 1*time.Second || delay > 1100*time.Millisecond {
		t.Errorf("delay after 1 attempt = %v, want ~1s", delay)
	}

	// After second attempt (count=2): 2s
	m.SetReconnecting(errors.New("error"))
	delay = m.NextDelay()
	if delay < 2*time.Second || delay > 2100*time.Millisecond {
		t.Errorf("delay after 2 attempts = %v, want ~2s", delay)
	}

	// After third attempt (count=3): 4s
	m.SetReconnecting(errors.New("error"))
	delay = m.NextDelay()
	if delay < 4*time.Second || delay > 4100*time.Millisecond {
		t.Errorf("delay after 3 attempts = %v, want ~4s", delay)
	}
}

func TestClassifyCloseCode(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		wantReason      DisconnectReason
		wantShouldRetry bool
	}{
		{
			name:            "nil error",
			err:             nil,
			wantReason:      ReasonUnknown,
			wantShouldRetry: false,
		},
		{
			name:            "normal closure",
			err:             websocket.CloseError{Code: websocket.StatusNormalClosure, Reason: "normal"},
			wantReason:      ReasonGraceful,
			wantShouldRetry: false,
		},
		{
			name:            "going away",
			err:             websocket.CloseError{Code: websocket.StatusGoingAway, Reason: "going away"},
			wantReason:      ReasonGraceful,
			wantShouldRetry: false,
		},
		{
			name:            "abnormal closure",
			err:             websocket.CloseError{Code: websocket.StatusAbnormalClosure, Reason: "crashed"},
			wantReason:      ReasonAbnormal,
			wantShouldRetry: true,
		},
		{
			name:            "non-websocket error",
			err:             errors.New("network timeout"),
			wantReason:      ReasonAbnormal,
			wantShouldRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, shouldRetry := ClassifyCloseCode(tt.err)
			if reason != tt.wantReason {
				t.Errorf("reason = %v, want %v", reason, tt.wantReason)
			}
			if shouldRetry != tt.wantShouldRetry {
				t.Errorf("shouldRetry = %v, want %v", shouldRetry, tt.wantShouldRetry)
			}
		})
	}
}

func TestConnectionManager_NextDelay_MaxCap(t *testing.T) {
	m := newConnectionManager()
	m.initialDelay = 1 * time.Second
	m.maxDelay = 5 * time.Second // Low max for testing
	m.backoffFactor = 2.0
	m.jitterPercent = 0

	// Simulate many reconnect attempts to exceed max delay
	for i := 0; i < 10; i++ {
		m.SetReconnecting(errors.New("error"))
	}

	delay := m.NextDelay()
	// With 10 attempts, backoff would be 512s without cap
	// Should be capped at maxDelay (5s)
	if delay > m.maxDelay+100*time.Millisecond {
		t.Errorf("delay = %v, should be capped at %v", delay, m.maxDelay)
	}
}

func TestConnectionManager_NextDelay_WithJitter(t *testing.T) {
	m := newConnectionManager()
	m.initialDelay = 1 * time.Second
	m.maxDelay = 30 * time.Second
	m.backoffFactor = 1.0 // No backoff to isolate jitter
	m.jitterPercent = 0.1 // 10% jitter

	// Run multiple times to verify jitter varies
	delays := make(map[time.Duration]bool)
	for i := 0; i < 10; i++ {
		delay := m.NextDelay()
		delays[delay] = true
		// Delay should be between 1s and 1.1s (1s + 10% jitter)
		if delay < 1*time.Second || delay > 1100*time.Millisecond {
			t.Errorf("delay = %v, should be between 1s and 1.1s", delay)
		}
	}
	// Jitter should produce some variation (at least 2 unique values in 10 runs)
	// Note: This is probabilistic but highly likely
	if len(delays) < 2 {
		t.Logf("Warning: jitter produced only %d unique values in 10 runs", len(delays))
	}
}

func TestConnectionManager_ConcurrentAccess(t *testing.T) {
	m := newConnectionManager()
	done := make(chan bool)

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			_ = m.State()
			_ = m.Info()
			_ = m.ReconnectCount()
			_ = m.MaxAttemptsReached()
		}
		done <- true
	}()

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			m.SetReconnecting(errors.New("test"))
			m.RecordHeartbeat()
			m.SetConnected()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done
}

func TestConnectionManager_SetConnected_ResetsState(t *testing.T) {
	m := newConnectionManager()
	m.maxAttempts = 5

	// Simulate reconnection attempts with an error
	for i := 0; i < 3; i++ {
		m.SetReconnecting(errors.New("test error"))
	}

	// Verify state before SetConnected
	if m.ReconnectCount() != 3 {
		t.Errorf("reconnect count = %d, want 3", m.ReconnectCount())
	}
	info := m.Info()
	if info.LastError != "test error" {
		t.Errorf("last error = %q, want 'test error'", info.LastError)
	}

	// SetConnected should reset everything
	m.SetConnected()

	if m.State() != StateConnected {
		t.Errorf("state = %v, want Connected", m.State())
	}
	if m.ReconnectCount() != 0 {
		t.Errorf("reconnect count after SetConnected = %d, want 0", m.ReconnectCount())
	}
	info = m.Info()
	if info.LastError != "" {
		t.Errorf("last error after SetConnected = %q, want empty", info.LastError)
	}
}

func TestConnectionManager_RecordHeartbeat(t *testing.T) {
	m := newConnectionManager()

	before := time.Now()
	time.Sleep(10 * time.Millisecond)
	m.RecordHeartbeat()
	after := time.Now()

	info := m.Info()
	if info.LastHeartbeat.Before(before) || info.LastHeartbeat.After(after) {
		t.Errorf("LastHeartbeat = %v, should be between %v and %v",
			info.LastHeartbeat, before, after)
	}
}

func TestConnectionManager_InfiniteAttempts(t *testing.T) {
	m := newConnectionManager()
	m.maxAttempts = 0 // 0 = infinite attempts

	// Even with many attempts, MaxAttemptsReached should return false
	for i := 0; i < 100; i++ {
		m.SetReconnecting(errors.New("error"))
	}

	if m.MaxAttemptsReached() {
		t.Error("MaxAttemptsReached should be false when maxAttempts=0 (infinite)")
	}
}

func TestConnectionManager_StateTransitions_AllPaths(t *testing.T) {
	tests := []struct {
		name      string
		from      ConnectionState
		action    string
		wantState ConnectionState
	}{
		{"Connected->Reconnecting", StateConnected, "reconnect", StateReconnecting},
		{"Connected->Disconnected", StateConnected, "disconnect", StateDisconnected},
		{"Reconnecting->Connected", StateReconnecting, "connect", StateConnected},
		{"Reconnecting->Disconnected", StateReconnecting, "disconnect", StateDisconnected},
		{"Disconnected->Connected", StateDisconnected, "connect", StateConnected},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newConnectionManager()

			// Set initial state
			switch tt.from {
			case StateReconnecting:
				m.SetReconnecting(errors.New("test"))
			case StateDisconnected:
				m.SetDisconnected(errors.New("test"))
			}

			// Perform action
			switch tt.action {
			case "connect":
				m.SetConnected()
			case "reconnect":
				m.SetReconnecting(errors.New("test"))
			case "disconnect":
				m.SetDisconnected(errors.New("test"))
			}

			if m.State() != tt.wantState {
				t.Errorf("state = %v, want %v", m.State(), tt.wantState)
			}
		})
	}
}

// TestConnectionManager_ReconnectionLoopPattern verifies that the correct
// "check-before-increment" pattern yields exactly maxAttempts attempts.
// This test simulates the loop structure used in attemptAutoReconnect.
func TestConnectionManager_ReconnectionLoopPattern(t *testing.T) {
	tests := []struct {
		maxAttempts  int
		wantAttempts int
	}{
		{1, 1},
		{3, 3},
		{5, 5},
		{10, 10},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("max=%d", tt.maxAttempts), func(t *testing.T) {
			m := newConnectionManager()
			m.maxAttempts = tt.maxAttempts

			attempts := 0

			// Simulate the reconnection loop pattern from attemptAutoReconnect:
			// Check BEFORE incrementing to ensure exactly maxAttempts attempts.
			for {
				if m.MaxAttemptsReached() {
					break
				}
				m.SetReconnecting(errors.New("test"))
				attempts++ // This represents an actual reconnection attempt
			}

			if attempts != tt.wantAttempts {
				t.Errorf("made %d attempts, want %d", attempts, tt.wantAttempts)
			}
			if m.ReconnectCount() != tt.wantAttempts {
				t.Errorf("reconnectCount = %d, want %d", m.ReconnectCount(), tt.wantAttempts)
			}
		})
	}
}
