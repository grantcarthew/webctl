package ipc

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

// ErrDaemonNotRunning is returned when the daemon is not running.
var ErrDaemonNotRunning = errors.New("daemon is not running")

// Client is a Unix socket IPC client.
type Client struct {
	conn   net.Conn
	reader *bufio.Reader
}

// Dial connects to the daemon at the default socket path.
func Dial() (*Client, error) {
	return DialPath(DefaultSocketPath())
}

// DialPath connects to the daemon at the specified socket path.
func DialPath(socketPath string) (*Client, error) {
	// Check if socket exists
	if _, err := os.Stat(socketPath); errors.Is(err, os.ErrNotExist) {
		return nil, ErrDaemonNotRunning
	}

	conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}

	return &Client{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}, nil
}

// Send sends a request to the daemon and returns the response.
func (c *Client) Send(req Request) (Response, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return Response{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	data = append(data, '\n')
	if _, err := c.conn.Write(data); err != nil {
		return Response{}, fmt.Errorf("failed to send request: %w", err)
	}

	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		return Response{}, fmt.Errorf("failed to read response: %w", err)
	}

	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return Response{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return resp, nil
}

// SendCmd is a convenience method for sending a simple command.
func (c *Client) SendCmd(cmd string) (Response, error) {
	return c.Send(Request{Cmd: cmd})
}

// Close closes the connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// IsDaemonRunning checks if the daemon is running by checking for the socket.
func IsDaemonRunning() bool {
	return IsDaemonRunningAt(DefaultSocketPath())
}

// IsDaemonRunningAt checks if the daemon is running at the specified socket path.
func IsDaemonRunningAt(socketPath string) bool {
	if _, err := os.Stat(socketPath); errors.Is(err, os.ErrNotExist) {
		return false
	}

	// Try to connect to verify it's actually running
	conn, err := net.DialTimeout("unix", socketPath, time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
