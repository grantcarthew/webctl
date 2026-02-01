package daemon

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// ConnectionState represents the current state of the CDP connection.
type ConnectionState int

const (
	// StateConnected indicates an active, healthy CDP connection.
	StateConnected ConnectionState = iota
	// StateReconnecting indicates the daemon is attempting to reconnect.
	StateReconnecting
	// StateDisconnected indicates the connection is lost and not recovering.
	StateDisconnected
)

// String returns a human-readable name for the connection state.
func (s ConnectionState) String() string {
	switch s {
	case StateConnected:
		return "connected"
	case StateReconnecting:
		return "reconnecting"
	case StateDisconnected:
		return "disconnected"
	default:
		return "unknown"
	}
}

// DisconnectReason describes why a disconnect occurred.
type DisconnectReason int

const (
	// ReasonUnknown is the default when reason cannot be determined.
	ReasonUnknown DisconnectReason = iota
	// ReasonGraceful indicates user-initiated close (codes 1000, 1001).
	ReasonGraceful
	// ReasonAbnormal indicates unexpected disconnect (code 1006, timeout).
	ReasonAbnormal
)

// String returns a human-readable name for the disconnect reason.
func (r DisconnectReason) String() string {
	switch r {
	case ReasonGraceful:
		return "graceful"
	case ReasonAbnormal:
		return "abnormal"
	default:
		return "unknown"
	}
}

// ConnectionInfo holds connection health information for status reporting.
type ConnectionInfo struct {
	State          ConnectionState `json:"state"`
	StateString    string          `json:"stateString"`
	LastHeartbeat  time.Time       `json:"lastHeartbeat,omitempty"`
	ReconnectCount int             `json:"reconnectCount,omitempty"`
	LastError      string          `json:"lastError,omitempty"`
}

// connectionManager manages CDP connection state and reconnection logic.
type connectionManager struct {
	mu sync.RWMutex

	state          ConnectionState
	lastHeartbeat  time.Time
	reconnectCount int
	lastError      error

	// Reconnection configuration
	maxAttempts    int
	initialDelay   time.Duration
	maxDelay       time.Duration
	backoffFactor  float64
	jitterPercent  float64
}

// newConnectionManager creates a new connection manager with default settings.
func newConnectionManager() *connectionManager {
	return &connectionManager{
		state:         StateConnected,
		lastHeartbeat: time.Now(),
		maxAttempts:   5,
		initialDelay:  1 * time.Second,
		maxDelay:      30 * time.Second,
		backoffFactor: 2.0,
		jitterPercent: 0.1,
	}
}

// State returns the current connection state.
func (m *connectionManager) State() ConnectionState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// Info returns connection health information for status reporting.
func (m *connectionManager) Info() ConnectionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info := ConnectionInfo{
		State:          m.state,
		StateString:    m.state.String(),
		LastHeartbeat:  m.lastHeartbeat,
		ReconnectCount: m.reconnectCount,
	}
	if m.lastError != nil {
		info.LastError = m.lastError.Error()
	}
	return info
}

// SetConnected transitions to connected state and resets counters.
func (m *connectionManager) SetConnected() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != StateConnected {
		m.logTransition(StateConnected, "")
	}
	m.state = StateConnected
	m.lastHeartbeat = time.Now()
	m.reconnectCount = 0
	m.lastError = nil
}

// SetReconnecting transitions to reconnecting state.
func (m *connectionManager) SetReconnecting(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.reconnectCount++
	m.lastError = err

	if m.state != StateReconnecting {
		m.logTransition(StateReconnecting, fmt.Sprintf("attempt %d/%d", m.reconnectCount, m.maxAttempts))
	} else {
		fmt.Fprintf(os.Stderr, "Reconnecting (attempt %d/%d)...\n", m.reconnectCount, m.maxAttempts)
	}
	m.state = StateReconnecting
}

// SetDisconnected transitions to disconnected state.
func (m *connectionManager) SetDisconnected(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastError = err
	if m.state != StateDisconnected {
		m.logTransition(StateDisconnected, "")
	}
	m.state = StateDisconnected
}

// RecordHeartbeat records a successful heartbeat.
func (m *connectionManager) RecordHeartbeat() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastHeartbeat = time.Now()
}

// ReconnectCount returns the current reconnect attempt count.
func (m *connectionManager) ReconnectCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.reconnectCount
}

// MaxAttemptsReached returns true if max reconnect attempts have been exceeded.
func (m *connectionManager) MaxAttemptsReached() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.maxAttempts > 0 && m.reconnectCount >= m.maxAttempts
}

// NextDelay calculates the delay before the next reconnect attempt.
func (m *connectionManager) NextDelay() time.Duration {
	m.mu.RLock()
	count := m.reconnectCount
	m.mu.RUnlock()

	if count <= 0 {
		count = 1
	}

	// Exponential backoff: initialDelay * (factor ^ (count-1))
	delay := float64(m.initialDelay)
	for i := 1; i < count; i++ {
		delay *= m.backoffFactor
	}

	if delay > float64(m.maxDelay) {
		delay = float64(m.maxDelay)
	}

	// Add jitter: delay * [0, jitterPercent]
	// Using math/rand for proper distribution.
	// REQUIRES Go 1.20+: Global rand is thread-safe and auto-seeded in Go 1.20+.
	// For Go < 1.20, use rand.New(rand.NewSource()) with mutex protection.
	jitter := delay * m.jitterPercent * rand.Float64()
	delay += jitter

	return time.Duration(delay)
}

// logTransition logs a state transition to stderr.
func (m *connectionManager) logTransition(newState ConnectionState, extra string) {
	var msg string
	switch newState {
	case StateConnected:
		msg = "Reconnected successfully"
	case StateReconnecting:
		msg = fmt.Sprintf("Reconnecting (%s)...", extra)
	case StateDisconnected:
		if m.lastError != nil {
			msg = fmt.Sprintf("Connection lost (%v)", m.lastError)
		} else {
			msg = "Connection lost"
		}
	}
	fmt.Fprintln(os.Stderr, msg)
}

// ClassifyCloseCode determines whether a disconnect is recoverable based on WebSocket close code.
// Returns the disconnect reason and whether automatic reconnection should be attempted.
func ClassifyCloseCode(err error) (reason DisconnectReason, shouldReconnect bool) {
	if err == nil {
		return ReasonUnknown, false
	}

	code := websocket.CloseStatus(err)
	switch code {
	case websocket.StatusNormalClosure, websocket.StatusGoingAway:
		// User-initiated close (browser closed normally)
		return ReasonGraceful, false
	case websocket.StatusAbnormalClosure:
		// No close frame received (crash, network issue)
		return ReasonAbnormal, true
	case -1:
		// Not a WebSocket close error (timeout, network error, etc.)
		return ReasonAbnormal, true
	default:
		// Other close codes - treat as abnormal
		return ReasonAbnormal, true
	}
}
