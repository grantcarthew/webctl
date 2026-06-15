package cdp

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/coder/websocket"
)

// raceConn lets a test make readLoop's Read return an error on demand while
// Close races against it. trigger() is idempotent so the test and Close can
// both fire it without a double-close panic in the mock itself.
type raceConn struct {
	releaseOnce sync.Once
	release     chan struct{}
}

func newRaceConn() *raceConn {
	return &raceConn{release: make(chan struct{})}
}

func (m *raceConn) trigger() {
	m.releaseOnce.Do(func() { close(m.release) })
}

func (m *raceConn) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	select {
	case <-m.release:
		return 0, nil, errors.New("simulated disconnect")
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	}
}

func (m *raceConn) Write(ctx context.Context, typ websocket.MessageType, p []byte) error {
	return nil
}

func (m *raceConn) Close(code websocket.StatusCode, reason string) error {
	m.trigger() // unblock readLoop so it can exit
	return nil
}

// TestClient_Close_RacesReadLoopDisconnect verifies that a deliberate Close and
// a read-loop disconnect happening at the same instant do not both close
// closedCh. Before markClosed consolidated the transition, the two paths closed
// the channel under non-atomic guards and panicked with "close of closed
// channel". Run under -race.
func TestClient_Close_RacesReadLoopDisconnect(t *testing.T) {
	t.Parallel()

	for i := 0; i < 500; i++ {
		conn := newRaceConn()
		client := NewClient(conn)

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			conn.trigger() // disconnect: readLoop's Read errors -> markClosed
		}()
		go func() {
			defer wg.Done()
			_ = client.Close() // deliberate close -> markClosed
		}()
		wg.Wait()

		// Close waited on done, so the read loop has exited; a second Close
		// must be a no-op returning nil.
		if err := client.Close(); err != nil {
			t.Fatalf("iteration %d: second close returned %v", i, err)
		}
	}
}
