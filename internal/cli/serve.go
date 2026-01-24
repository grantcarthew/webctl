package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grantcarthew/webctl/internal/daemon"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve [directory] [--proxy url]",
	Short: "Start development server with hot reload",
	Long: `Start a development web server with automatic hot reload capabilities.

Auto-Start Behavior:
  Automatically starts the daemon and browser if not already running.
  One command to start everything: webctl serve

Two modes available:

Static Mode (serve directory):
  webctl serve                     # Serve current directory (default)
  webctl serve <directory>         # Serve static files from directory
  webctl serve ./dist              # Serve ./dist directory
  webctl serve .                   # Serve current directory (explicit)

Proxy Mode (proxy to backend):
  webctl serve --proxy <url>       # Proxy requests to backend server
  webctl serve --proxy localhost:3000
  webctl serve --proxy http://api.example.com:8080

Features:
- Auto-starts daemon and browser if needed
- Automatic file watching and hot reload (static mode)
- Auto-detect available port or use --port flag
- Network binding options (localhost vs 0.0.0.0)
- Automatic browser navigation to served URL
- Full access to webctl debugging commands (console, network, etc.)

Examples:

Static mode (default):
  serve ./public                   # Serve ./public directory
  serve . --port 3000              # Serve current dir on port 3000
  serve ./dist --host 0.0.0.0      # Accessible from network

Proxy mode:
  serve --proxy localhost:8080     # Proxy to localhost:8080
  serve --proxy http://api.local:3000 --port 3001

Custom watch paths:
  serve ./public --watch src/,assets/
  serve ./dist --ignore "*.tmp,*.log"

Server lifecycle:
  serve <dir>                      # Start server
  <Ctrl+C> or webctl stop          # Stop server and daemon

Integration with webctl commands:
  serve ./public                   # Start server
  console                          # Monitor console logs
  network --status 4xx             # Monitor network errors
  html --select "#app"             # Inspect rendered HTML`,
	Args: cobra.MaximumNArgs(1),
	RunE: runServe,
}

var (
	serveProxy  string
	servePort   int
	serveHost   string
	serveWatch  []string
	serveIgnore []string
)

func init() {
	serveCmd.Flags().StringVar(&serveProxy, "proxy", "", "Backend URL to proxy (enables proxy mode)")
	serveCmd.Flags().IntVar(&servePort, "port", 0, "Server port (0 = auto-detect)")
	serveCmd.Flags().StringVar(&serveHost, "host", "localhost", "Bind host (localhost or 0.0.0.0)")
	serveCmd.Flags().StringSliceVar(&serveWatch, "watch", nil, "Additional paths to watch (comma-separated)")
	serveCmd.Flags().StringSliceVar(&serveIgnore, "ignore", nil, "Glob patterns to ignore (comma-separated)")

	rootCmd.AddCommand(serveCmd)
}

// runServeWithDaemon starts the daemon and server together when daemon is not running
func runServeWithDaemon(mode, directory, proxyURL string) error {
	// Create daemon config
	cfg := daemon.DefaultConfig()
	cfg.Headless = false // Default to headed mode for serve
	cfg.Port = 0         // Auto-detect available CDP port
	cfg.Debug = Debug

	// Declare d first so the closure can capture it
	var d *daemon.Daemon

	// Create command executor for REPL
	cfg.CommandExecutor = func(args []string) (bool, error) {
		factory := NewDirectExecutorFactory(d.Handler())
		SetExecutorFactory(factory)
		defer ResetExecutorFactory()
		return ExecuteArgs(args)
	}

	d = daemon.New(cfg)

	// Output startup message
	fmt.Println("Starting daemon and server...")

	// Start daemon in background goroutine
	daemonErr := make(chan error, 1)
	go func() {
		daemonErr <- d.Run(context.Background())
	}()

	// Wait for daemon to start (give it a moment to initialize)
	time.Sleep(500 * time.Millisecond)

	// Check if daemon failed to start
	select {
	case err := <-daemonErr:
		outErr := outputError(fmt.Sprintf("failed to start daemon: %v", err))
		if strings.Contains(err.Error(), "port") || strings.Contains(err.Error(), "in use") {
			outputHint("use 'webctl stop --force' to kill orphaned processes")
		}
		return outErr
	default:
		// Daemon started successfully
	}

	// Use direct executor since daemon is in-process
	factory := NewDirectExecutorFactory(d.Handler())
	exec, err := factory.NewExecutor()
	if err != nil {
		return outputError(fmt.Sprintf("failed to create executor: %v", err))
	}
	defer func() { _ = exec.Close() }()

	// Build serve parameters
	params, err := json.Marshal(ipc.ServeParams{
		Action:      "start",
		Mode:        mode,
		Directory:   directory,
		ProxyURL:    proxyURL,
		Port:        servePort,
		Host:        serveHost,
		WatchPaths:  serveWatch,
		IgnorePaths: serveIgnore,
	})
	if err != nil {
		return outputError(err.Error())
	}

	// Execute serve command
	resp, err := exec.Execute(ipc.Request{
		Cmd:    "serve",
		Params: params,
	})
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		outErr := outputError(resp.Error)
		if strings.Contains(resp.Error, "already running") {
			outputHint("use 'webctl stop' to stop the server, or 'webctl stop --force' to force cleanup")
		}
		return outErr
	}

	// Parse response data
	var data ipc.ServeData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return outputError(err.Error())
	}

	// Output result
	if !JSONOutput {
		fmt.Printf("Server started: %s\n", data.URL)
		fmt.Printf("Mode: %s\n", mode)
		if mode == "static" {
			fmt.Printf("Directory: %s\n", directory)
		} else {
			fmt.Printf("Proxying to: %s\n", proxyURL)
		}
		fmt.Printf("Port: %d\n", data.Port)

		if len(serveWatch) > 0 || mode == "static" {
			fmt.Println("\nWatching for file changes (hot reload enabled)")
		}

		fmt.Println("\nPress Ctrl+C to stop the server and daemon")
	}

	// Wait for daemon to exit (blocks here)
	return <-daemonErr
}

func runServe(cmd *cobra.Command, args []string) error {
	t := startTimer("serve")
	defer t.log()

	// Determine mode and validate arguments
	var mode string
	var directory string
	var proxyURL string

	if serveProxy != "" {
		// Proxy mode
		mode = "proxy"
		proxyURL = serveProxy

		if len(args) > 0 {
			return outputError("cannot specify both directory and --proxy flag")
		}
	} else {
		// Static mode - defaults to current directory
		mode = "static"

		if len(args) == 0 {
			directory = "."
		} else {
			directory = args[0]
		}

		// Validate directory exists
		if _, err := os.Stat(directory); os.IsNotExist(err) {
			return outputError(fmt.Sprintf("directory does not exist: %s", directory))
		}

		// Resolve to absolute path for display
		absDir, err := filepath.Abs(directory)
		if err == nil {
			directory = absDir
		}
	}

	debugParam("mode=%s directory=%q proxy=%q port=%d host=%q", mode, directory, proxyURL, servePort, serveHost)

	// If daemon is not running, start it with the server
	if !execFactory.IsDaemonRunning() {
		return runServeWithDaemon(mode, directory, proxyURL)
	}

	// Create executor
	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer func() { _ = exec.Close() }()

	// Build serve parameters
	params, err := json.Marshal(ipc.ServeParams{
		Action:      "start",
		Mode:        mode,
		Directory:   directory,
		ProxyURL:    proxyURL,
		Port:        servePort,
		Host:        serveHost,
		WatchPaths:  serveWatch,
		IgnorePaths: serveIgnore,
	})
	if err != nil {
		return outputError(err.Error())
	}

	debugRequest("serve", fmt.Sprintf("mode=%s port=%d", mode, servePort))
	ipcStart := time.Now()

	// Execute serve command
	resp, err := exec.Execute(ipc.Request{
		Cmd:    "serve",
		Params: params,
	})

	debugResponse(err == nil && resp.OK, len(resp.Data), time.Since(ipcStart))

	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		outErr := outputError(resp.Error)
		if strings.Contains(resp.Error, "already running") {
			outputHint("use 'webctl stop' to stop the server, or 'webctl stop --force' to force cleanup")
		}
		return outErr
	}

	// Parse response data
	var data ipc.ServeData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return outputError(err.Error())
	}

	// Output result
	if JSONOutput {
		return outputJSON(os.Stdout, map[string]any{
			"ok":   true,
			"mode": mode,
			"url":  data.URL,
			"port": data.Port,
		})
	}

	// Text mode output
	fmt.Printf("Server started: %s\n", data.URL)
	fmt.Printf("Mode: %s\n", mode)
	if mode == "static" {
		fmt.Printf("Directory: %s\n", directory)
	} else {
		fmt.Printf("Proxying to: %s\n", proxyURL)
	}
	fmt.Printf("Port: %d\n", data.Port)

	if len(serveWatch) > 0 || mode == "static" {
		fmt.Println("\nWatching for file changes (hot reload enabled)")
	}

	fmt.Println("\nPress Ctrl+C or run 'webctl stop' to stop the server")

	return nil
}
