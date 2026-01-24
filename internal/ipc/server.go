package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
)

// Handler processes IPC requests and returns responses.
type Handler func(req Request) Response

// Server is a Unix socket IPC server.
type Server struct {
	socketPath string
	listener   net.Listener
	handler    Handler
	wg         sync.WaitGroup
	closed     chan struct{}
	closeOnce  sync.Once
}

// NewServer creates a new Unix socket server.
// The socket file is created at the specified path.
func NewServer(socketPath string, handler Handler) (*Server, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(socketPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove existing socket file if present
	if err := os.Remove(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to remove existing socket: %w", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create unix socket: %w", err)
	}

	// Set socket permissions to owner-only
	if err := os.Chmod(socketPath, 0600); err != nil {
		_ = listener.Close()
		return nil, fmt.Errorf("failed to set socket permissions: %w", err)
	}

	return &Server{
		socketPath: socketPath,
		listener:   listener,
		handler:    handler,
		closed:     make(chan struct{}),
	}, nil
}

// Serve starts accepting connections. Blocks until Close is called.
func (s *Server) Serve(ctx context.Context) error {
	go func() {
		select {
		case <-ctx.Done():
			_ = s.Close()
		case <-s.closed:
		}
	}()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.closed:
				return nil
			default:
				return fmt.Errorf("accept error: %w", err)
			}
		}

		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

// handleConn processes a single client connection.
func (s *Server) handleConn(conn net.Conn) {
	defer s.wg.Done()
	defer func() { _ = conn.Close() }()

	reader := bufio.NewReader(conn)

	for {
		// Read newline-delimited JSON
		line, err := reader.ReadBytes('\n')
		if err != nil {
			// EOF means client closed connection normally.
			// net.ErrClosed occurs during server shutdown.
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				log.Printf("ipc: unexpected read error: %v", err)
			}
			return
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			resp := ErrorResponse("invalid request format")
			if err := s.writeResponse(conn, resp); err != nil {
				return
			}
			continue
		}

		resp := s.handler(req)
		if err := s.writeResponse(conn, resp); err != nil {
			return
		}
	}
}

// writeResponse sends a JSON response to the client.
func (s *Server) writeResponse(conn net.Conn, resp Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = conn.Write(data)
	return err
}

// SocketPath returns the path to the Unix socket.
func (s *Server) SocketPath() string {
	return s.socketPath
}

// Close stops the server and cleans up resources.
// Safe to call multiple times concurrently.
func (s *Server) Close() error {
	var err error
	s.closeOnce.Do(func() {
		close(s.closed)
		err = s.listener.Close()
		s.wg.Wait()
		// Clean up socket file
		_ = os.Remove(s.socketPath)
	})
	return err
}

// DefaultSocketPath returns the XDG-compliant socket path.
func DefaultSocketPath() string {
	// Try XDG_RUNTIME_DIR first
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		return filepath.Join(runtimeDir, "webctl", "webctl.sock")
	}

	// Fallback to /tmp/webctl-<uid>/
	return filepath.Join(fmt.Sprintf("/tmp/webctl-%d", os.Getuid()), "webctl.sock")
}

// DefaultPIDPath returns the XDG-compliant PID file path.
func DefaultPIDPath() string {
	// Try XDG_RUNTIME_DIR first
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		return filepath.Join(runtimeDir, "webctl", "webctl.pid")
	}

	// Fallback to /tmp/webctl-<uid>/
	return filepath.Join(fmt.Sprintf("/tmp/webctl-%d", os.Getuid()), "webctl.pid")
}
