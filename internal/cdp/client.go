package cdp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

// DefaultTimeout is the default timeout for CDP commands.
const DefaultTimeout = 30 * time.Second

// Client is a CDP protocol client.
type Client struct {
	conn    Conn
	writeMu sync.Mutex
	msgID   atomic.Int64

	// pending maps command IDs to response channels
	pending   sync.Map // map[int64]chan *Response
	listeners sync.Map // map[string][]func(Event)

	// closed signals that the client is shutting down
	closed   atomic.Bool
	closedCh chan struct{}
	closeErr error
	closeMu  sync.Mutex

	// done signals that the read loop has exited
	done chan struct{}
}

// NewClient creates a new CDP client with the given connection.
func NewClient(conn Conn) *Client {
	c := &Client{
		conn:     conn,
		closedCh: make(chan struct{}),
		done:     make(chan struct{}),
	}
	go c.readLoop()
	return c
}

// Dial connects to a CDP endpoint and returns a new client.
func Dial(ctx context.Context, wsURL string) (*Client, error) {
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to CDP endpoint: %w", err)
	}
	return NewClient(conn), nil
}

// Send sends a CDP command and waits for the response.
// Uses the default timeout.
func (c *Client) Send(method string, params interface{}) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	return c.SendContext(ctx, method, params)
}

// SendContext sends a CDP command with a context for cancellation.
func (c *Client) SendContext(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	if c.closed.Load() {
		return nil, errors.New("client is closed")
	}

	id := c.msgID.Add(1)
	req := Request{
		ID:     id,
		Method: method,
		Params: params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create response channel before sending
	respCh := make(chan *Response, 1)
	c.pending.Store(id, respCh)
	defer c.pending.Delete(id)

	// Send the request
	c.writeMu.Lock()
	err = c.conn.Write(ctx, websocket.MessageText, data)
	c.writeMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Wait for response
	select {
	case resp := <-respCh:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("request timed out: %w", ctx.Err())
	case <-c.closedCh:
		return nil, errors.New("client closed while waiting for response")
	}
}

// Subscribe registers a handler for CDP events matching the given method.
// Multiple handlers can be registered for the same method.
func (c *Client) Subscribe(method string, handler func(Event)) {
	actual, _ := c.listeners.LoadOrStore(method, &eventHandlers{})
	handlers := actual.(*eventHandlers)
	handlers.add(handler)
}

// Close closes the client connection and stops the read loop.
func (c *Client) Close() error {
	if c.closed.Swap(true) {
		return nil // Already closed
	}

	close(c.closedCh)

	c.closeMu.Lock()
	err := c.conn.Close(websocket.StatusNormalClosure, "client closing")
	c.closeMu.Unlock()

	// Wait for read loop to exit
	<-c.done

	return err
}

// Err returns any error that caused the client to close.
func (c *Client) Err() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	return c.closeErr
}

// readLoop reads messages from the connection and dispatches them.
func (c *Client) readLoop() {
	defer close(c.done)

	ctx := context.Background()
	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			if !c.closed.Load() {
				c.closeMu.Lock()
				c.closeErr = err
				c.closeMu.Unlock()
				c.closed.Store(true)
				close(c.closedCh)
			}
			return
		}

		resp, evt, err := parseMessage(data)
		if err != nil {
			continue // Skip malformed messages
		}

		if resp != nil {
			c.dispatchResponse(resp)
		} else if evt != nil {
			c.dispatchEvent(evt)
		}
	}
}

// dispatchResponse sends a response to the waiting caller.
func (c *Client) dispatchResponse(resp *Response) {
	if ch, ok := c.pending.Load(resp.ID); ok {
		respCh := ch.(chan *Response)
		select {
		case respCh <- resp:
		default:
			// Channel full or closed, response dropped
		}
	}
}

// dispatchEvent calls all registered handlers for an event.
func (c *Client) dispatchEvent(evt *Event) {
	if actual, ok := c.listeners.Load(evt.Method); ok {
		handlers := actual.(*eventHandlers)
		handlers.call(*evt)
	}
}

// eventHandlers manages a thread-safe list of event handlers.
type eventHandlers struct {
	mu       sync.RWMutex
	handlers []func(Event)
}

func (h *eventHandlers) add(handler func(Event)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handlers = append(h.handlers, handler)
}

func (h *eventHandlers) call(evt Event) {
	h.mu.RLock()
	handlers := h.handlers
	h.mu.RUnlock()

	for _, handler := range handlers {
		handler(evt)
	}
}
