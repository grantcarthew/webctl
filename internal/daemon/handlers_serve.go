package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/grantcarthew/webctl/internal/server"
)

// handleServe handles the "serve" command.
func (d *Daemon) handleServe(req ipc.Request) ipc.Response {
	var params ipc.ServeParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("invalid params: %v", err))
		}
	}

	switch params.Action {
	case "start":
		return d.handleServeStart(params)
	case "stop":
		return d.handleServeStop()
	case "status":
		return d.handleServeStatus()
	default:
		return ipc.ErrorResponse(fmt.Sprintf("unknown action: %s", params.Action))
	}
}

// handleServeStart starts the development server.
func (d *Daemon) handleServeStart(params ipc.ServeParams) ipc.Response {
	d.devServerMu.Lock()
	defer d.devServerMu.Unlock()

	// Check if server is already running
	if d.devServer != nil && d.devServer.IsRunning() {
		return ipc.ErrorResponse("server already running")
	}

	// Validate mode
	var mode server.Mode
	switch params.Mode {
	case "static":
		mode = server.ModeStatic
	case "proxy":
		mode = server.ModeProxy
	default:
		return ipc.ErrorResponse("mode must be 'static' or 'proxy'")
	}

	// Create server config
	cfg := server.Config{
		Mode:        mode,
		Directory:   params.Directory,
		ProxyURL:    params.ProxyURL,
		Port:        params.Port,
		Host:        params.Host,
		WatchPaths:  params.WatchPaths,
		IgnorePaths: params.IgnorePaths,
		OnReload:    d.handleServerReload,
		Debug:       d.debug,
	}

	// Create server
	srv, err := server.New(cfg)
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to create server: %v", err))
	}

	// Start server
	ctx := context.Background()
	if err := srv.Start(ctx); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to start server: %v", err))
	}

	d.devServer = srv
	d.debugf(false, "Development server started: %s", srv.URL())

	// Navigate browser to server URL if browser is running
	if d.browserConnected() {
		// Get active session
		session := d.sessions.Active()
		if session != nil {
			// Navigate to server URL
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				_, err := d.sendToSession(ctx, session.ID, "Page.navigate", map[string]any{
					"url": srv.URL(),
				})
				if err != nil {
					d.debugf(false, "Failed to navigate to server URL: %v", err)
				} else {
					d.debugf(false, "Navigated to server URL: %s", srv.URL())
				}
			}()
		}
	}

	return ipc.SuccessResponse(ipc.ServeData{
		Running: true,
		Mode:    params.Mode,
		URL:     srv.URL(),
		Port:    srv.Port(),
	})
}

// handleServeStop stops the development server.
func (d *Daemon) handleServeStop() ipc.Response {
	d.devServerMu.Lock()
	defer d.devServerMu.Unlock()

	if d.devServer == nil || !d.devServer.IsRunning() {
		return ipc.ErrorResponse("server not running")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := d.devServer.Stop(ctx); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to stop server: %v", err))
	}

	d.devServer = nil
	d.debugf(false, "Development server stopped")

	return ipc.SuccessResponse(ipc.ServeData{
		Running: false,
	})
}

// handleServeStatus returns the status of the development server.
func (d *Daemon) handleServeStatus() ipc.Response {
	d.devServerMu.Lock()
	defer d.devServerMu.Unlock()

	if d.devServer == nil || !d.devServer.IsRunning() {
		return ipc.SuccessResponse(ipc.ServeData{
			Running: false,
		})
	}

	return ipc.SuccessResponse(ipc.ServeData{
		Running: true,
		URL:     d.devServer.URL(),
		Port:    d.devServer.Port(),
	})
}

// handleServerReload is called when files change - triggers page reload via CDP.
func (d *Daemon) handleServerReload() {
	d.debugf(false, "File change detected - reloading page")

	// Check if browser is connected
	if !d.browserConnected() {
		d.debugf(false, "Browser not connected - skipping reload")
		return
	}

	// Get active session
	session := d.sessions.Active()
	if session == nil {
		d.debugf(false, "No active session - skipping reload")
		return
	}

	// Reload page via CDP
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := d.sendToSession(ctx, session.ID, "Page.reload", map[string]any{
			"ignoreCache": false,
		})
		if err != nil {
			d.debugf(false, "Failed to reload page: %v", err)
		} else {
			d.debugf(false, "Page reloaded successfully")
		}
	}()
}
