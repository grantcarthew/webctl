package cdp

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// mockConn implements the Conn interface for testing.
type mockConn struct {
	mu           sync.Mutex
	readCh       chan []byte // Channel-based message delivery
	written      [][]byte
	readErr      error
	writeErr     error
	closed       bool
	closeCh      chan struct{}
}

func newMockConn(messages ...[]byte) *mockConn {
	m := &mockConn{
		readCh:  make(chan []byte, len(messages)+10),
		closeCh: make(chan struct{}),
	}
	for _, msg := range messages {
		m.readCh <- msg
	}
	return m
}

func (m *mockConn) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	m.mu.Lock()
	readErr := m.readErr
	m.mu.Unlock()

	if readErr != nil {
		return 0, nil, readErr
	}

	select {
	case msg, ok := <-m.readCh:
		if !ok {
			return 0, nil, errors.New("connection closed")
		}
		return websocket.MessageText, msg, nil
	case <-m.closeCh:
		return 0, nil, errors.New("connection closed")
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	}
}

func (m *mockConn) Write(ctx context.Context, typ websocket.MessageType, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.writeErr != nil {
		return m.writeErr
	}
	m.written = append(m.written, data)
	return nil
}

func (m *mockConn) Close(code websocket.StatusCode, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		close(m.closeCh)
	}
	return nil
}

func (m *mockConn) getWritten() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([][]byte, len(m.written))
	copy(result, m.written)
	return result
}

func (m *mockConn) queueResponse(data []byte) {
	m.readCh <- data
}

func TestClient_Send_CorrelatesResponseByID(t *testing.T) {
	t.Parallel()

	// Use echo mock that responds after each write to avoid race condition
	// where the read loop consumes the response before Send registers its channel
	conn := newEchoMockConnWithResult(`{"frameId":"ABC123"}`)

	client := NewClient(conn)
	defer client.Close()

	result, err := client.Send("Page.navigate", map[string]string{"url": "https://example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `{"frameId":"ABC123"}`
	if string(result) != expected {
		t.Errorf("expected result %s, got %s", expected, string(result))
	}

	// Verify the request was sent correctly
	written := conn.getWritten()
	if len(written) != 1 {
		t.Fatalf("expected 1 written message, got %d", len(written))
	}

	var req Request
	if err := json.Unmarshal(written[0], &req); err != nil {
		t.Fatalf("failed to unmarshal request: %v", err)
	}

	if req.ID != 1 {
		t.Errorf("expected request ID 1, got %d", req.ID)
	}
	if req.Method != "Page.navigate" {
		t.Errorf("expected method Page.navigate, got %s", req.Method)
	}
}

func TestClient_Send_ReturnsErrorOnCDPError(t *testing.T) {
	t.Parallel()

	// Use echo mock that returns error response after each write
	conn := newEchoMockConnWithError(-32000, "Target closed")

	client := NewClient(conn)
	defer client.Close()

	_, err := client.Send("Page.navigate", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var cdpErr *Error
	if !errors.As(err, &cdpErr) {
		t.Fatalf("expected CDP error, got %T: %v", err, err)
	}

	if cdpErr.Code != -32000 {
		t.Errorf("expected error code -32000, got %d", cdpErr.Code)
	}
	if cdpErr.Message != "Target closed" {
		t.Errorf("expected message 'Target closed', got %s", cdpErr.Message)
	}
}

func TestClient_SendContext_TimeoutWaitingForResponse(t *testing.T) {
	t.Parallel()

	// Connection that blocks forever on read (no messages queued)
	conn := newMockConn()

	client := NewClient(conn)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.SendContext(ctx, "Page.navigate", nil)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestClient_Subscribe_DispatchesToHandler(t *testing.T) {
	t.Parallel()

	// Prepare an event
	evt := struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}{
		Method: "Page.loadEventFired",
		Params: json.RawMessage(`{"timestamp":123.456}`),
	}
	evtData, _ := json.Marshal(evt)

	conn := newMockConn(evtData)

	client := NewClient(conn)
	defer client.Close()

	received := make(chan Event, 1)
	client.Subscribe("Page.loadEventFired", func(e Event) {
		received <- e
	})

	// Wait for event
	select {
	case e := <-received:
		if e.Method != "Page.loadEventFired" {
			t.Errorf("expected method Page.loadEventFired, got %s", e.Method)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestClient_Subscribe_MultipleHandlers(t *testing.T) {
	t.Parallel()

	evt := struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}{
		Method: "Network.requestWillBeSent",
		Params: json.RawMessage(`{}`),
	}
	evtData, _ := json.Marshal(evt)

	conn := newMockConn(evtData)

	client := NewClient(conn)
	defer client.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	count := 0
	var mu sync.Mutex

	handler := func(e Event) {
		mu.Lock()
		count++
		mu.Unlock()
		wg.Done()
	}

	client.Subscribe("Network.requestWillBeSent", handler)
	client.Subscribe("Network.requestWillBeSent", handler)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		mu.Lock()
		if count != 2 {
			t.Errorf("expected 2 handler calls, got %d", count)
		}
		mu.Unlock()
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for handlers")
	}
}

func TestClient_Close_CleansUpResources(t *testing.T) {
	t.Parallel()

	conn := newMockConn()

	client := NewClient(conn)

	err := client.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify connection was closed
	conn.mu.Lock()
	closed := conn.closed
	conn.mu.Unlock()

	if !closed {
		t.Error("expected connection to be closed")
	}

	// Verify double-close is safe
	err = client.Close()
	if err != nil {
		t.Errorf("double close returned error: %v", err)
	}
}

func TestClient_ConcurrentSends(t *testing.T) {
	t.Parallel()

	const numRequests = 10

	// Use an echo mock that responds to each write with a matching response
	conn := newEchoMockConn()

	client := NewClient(conn)
	defer client.Close()

	var wg sync.WaitGroup
	errCh := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.Send("Test.method", nil)
			if err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent send error: %v", err)
	}
}

// echoMockConn echoes back a response for each written request.
type echoMockConn struct {
	mu        sync.Mutex
	responses chan []byte
	written   [][]byte
	closed    bool
	closeCh   chan struct{}
	result    json.RawMessage // Custom result to return
}

func newEchoMockConn() *echoMockConn {
	return &echoMockConn{
		responses: make(chan []byte, 100),
		closeCh:   make(chan struct{}),
		result:    json.RawMessage(`{"ok":true}`),
	}
}

func newEchoMockConnWithResult(result string) *echoMockConn {
	return &echoMockConn{
		responses: make(chan []byte, 100),
		closeCh:   make(chan struct{}),
		result:    json.RawMessage(result),
	}
}

// echoMockConnWithError returns an error response for each request.
type echoMockConnWithError struct {
	mu        sync.Mutex
	responses chan []byte
	closed    bool
	closeCh   chan struct{}
	cdpError  *Error
}

func newEchoMockConnWithError(code int, message string) *echoMockConnWithError {
	return &echoMockConnWithError{
		responses: make(chan []byte, 100),
		closeCh:   make(chan struct{}),
		cdpError:  &Error{Code: code, Message: message},
	}
}

func (m *echoMockConnWithError) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	select {
	case resp := <-m.responses:
		return websocket.MessageText, resp, nil
	case <-m.closeCh:
		return 0, nil, errors.New("connection closed")
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	}
}

func (m *echoMockConnWithError) Write(ctx context.Context, typ websocket.MessageType, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return errors.New("connection closed")
	}

	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}

	resp := Response{
		ID:    req.ID,
		Error: m.cdpError,
	}
	respData, _ := json.Marshal(resp)
	m.responses <- respData

	return nil
}

func (m *echoMockConnWithError) Close(code websocket.StatusCode, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		close(m.closeCh)
	}
	return nil
}

func (m *echoMockConn) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	select {
	case resp := <-m.responses:
		return websocket.MessageText, resp, nil
	case <-m.closeCh:
		return 0, nil, errors.New("connection closed")
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	}
}

func (m *echoMockConn) Write(ctx context.Context, typ websocket.MessageType, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return errors.New("connection closed")
	}

	// Track written data
	m.written = append(m.written, data)

	// Parse the request to get the ID
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}

	// Generate a matching response with the configured result
	resp := Response{
		ID:     req.ID,
		Result: m.result,
	}
	respData, _ := json.Marshal(resp)
	m.responses <- respData

	return nil
}

func (m *echoMockConn) getWritten() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([][]byte, len(m.written))
	copy(result, m.written)
	return result
}

func (m *echoMockConn) Close(code websocket.StatusCode, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		close(m.closeCh)
	}
	return nil
}

func TestClient_Send_ConnectionClosedMidRequest(t *testing.T) {
	t.Parallel()

	conn := newMockConn()

	client := NewClient(conn)

	// Close the connection after a short delay
	go func() {
		time.Sleep(20 * time.Millisecond)
		client.Close()
	}()

	_, err := client.Send("Page.navigate", nil)
	if err == nil {
		t.Fatal("expected error when connection closes, got nil")
	}
}

func TestClient_ReadLoop_HandlesUnknownMessageID(t *testing.T) {
	t.Parallel()

	// Use a custom mock that sends an unknown ID first, then the correct response
	conn := newUnknownIDMockConn()

	client := NewClient(conn)
	defer client.Close()

	// Should still work despite unknown message being sent first
	result, err := client.Send("Test.method", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != `{"success":true}` {
		t.Errorf("expected success result, got %s", string(result))
	}
}

// unknownIDMockConn sends an unknown ID response first, then the correct response.
type unknownIDMockConn struct {
	mu        sync.Mutex
	responses chan []byte
	closed    bool
	closeCh   chan struct{}
}

func newUnknownIDMockConn() *unknownIDMockConn {
	return &unknownIDMockConn{
		responses: make(chan []byte, 100),
		closeCh:   make(chan struct{}),
	}
}

func (m *unknownIDMockConn) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	select {
	case resp := <-m.responses:
		return websocket.MessageText, resp, nil
	case <-m.closeCh:
		return 0, nil, errors.New("connection closed")
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	}
}

func (m *unknownIDMockConn) Write(ctx context.Context, typ websocket.MessageType, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return errors.New("connection closed")
	}

	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}

	// First send an unknown ID response
	unknownResp := Response{
		ID:     9999,
		Result: json.RawMessage(`{}`),
	}
	unknownData, _ := json.Marshal(unknownResp)
	m.responses <- unknownData

	// Then send the correct response
	validResp := Response{
		ID:     req.ID,
		Result: json.RawMessage(`{"success":true}`),
	}
	validData, _ := json.Marshal(validResp)
	m.responses <- validData

	return nil
}

func (m *unknownIDMockConn) Close(code websocket.StatusCode, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		close(m.closeCh)
	}
	return nil
}
