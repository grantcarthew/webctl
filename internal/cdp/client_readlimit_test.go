package cdp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// TestDial_LargeMessageDelivered verifies that a CDP message larger than the
// former 16MB read limit is delivered to subscribers rather than tearing down
// the connection. Regression for the crash where logged-in pages (e.g. a large
// Network.getResponseBody body) produced an oversized message, which coder/websocket
// treats as fatal, causing the daemon to shut down and kill a healthy browser.
func TestDial_LargeMessageDelivered(t *testing.T) {
	const bodySize = 17 * 1024 * 1024 // above the old 16MB cap

	event, err := json.Marshal(map[string]any{
		"method": "Test.large",
		"params": map[string]string{"body": strings.Repeat("a", bodySize)},
	})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	// subscribed gates the server write until the test has registered its
	// listener, so the read loop cannot dispatch (and drop) the event before
	// Subscribe runs. Without this the test relies on the 17MB read out-lasting
	// the subscribe call, which a scheduler hiccup could flip into a 10s hang.
	subscribed := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		ctx := r.Context()
		<-subscribed
		if err := c.Write(ctx, websocket.MessageText, event); err != nil {
			c.Close(websocket.StatusInternalError, "write failed")
			return
		}
		// Reading drives the close handshake; returns once the client closes.
		_, _, _ = c.Read(ctx)
		c.Close(websocket.StatusNormalClosure, "")
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	client, err := Dial(context.Background(), wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	got := make(chan int, 1)
	client.Subscribe("Test.large", func(evt Event) {
		var p struct {
			Body string `json:"body"`
		}
		if err := json.Unmarshal(evt.Params, &p); err != nil {
			t.Errorf("unmarshal params: %v", err)
			got <- -1
			return
		}
		got <- len(p.Body)
	})
	close(subscribed) // listener registered; let the server send

	select {
	case n := <-got:
		if n != bodySize {
			t.Fatalf("delivered body size = %d, want %d", n, bodySize)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("large message was not delivered (read limit treated it as fatal?)")
	}
}
