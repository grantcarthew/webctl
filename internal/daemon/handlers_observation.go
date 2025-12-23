package daemon

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/grantcarthew/webctl/internal/cdp"
	"github.com/grantcarthew/webctl/internal/ipc"
)

// handleStatus returns the daemon status.
func (d *Daemon) handleStatus() ipc.Response {
	status := ipc.StatusData{
		Running:  true,
		PID:      os.Getpid(),
		Sessions: d.sessions.All(),
	}

	// Get active session info
	active := d.sessions.Active()
	if active != nil {
		status.ActiveSession = active
		// Populate deprecated fields for backwards compatibility
		status.URL = active.URL
		status.Title = active.Title
	}

	return ipc.SuccessResponse(status)
}

// handleConsole returns buffered console entries filtered to active session.
func (d *Daemon) handleConsole() ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	allEntries := d.consoleBuf.All()
	var filtered []ipc.ConsoleEntry
	for _, e := range allEntries {
		if e.SessionID == activeID {
			filtered = append(filtered, e)
		}
	}

	return ipc.SuccessResponse(ipc.ConsoleData{
		Entries: filtered,
		Count:   len(filtered),
	})
}

// handleNetwork returns buffered network entries filtered to active session.
// Enables Network domain lazily on first call to avoid blocking Runtime.evaluate.
func (d *Daemon) handleNetwork() ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	// Enable Network domain lazily for this session
	if _, loaded := d.networkEnabled.LoadOrStore(activeID, true); !loaded {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := d.cdp.SendToSession(ctx, activeID, "Network.enable", nil); err != nil {
			d.debugf("warning: failed to enable Network domain: %v", err)
		} else {
			d.debugf("Network domain enabled lazily for session %s", activeID)
		}
	}

	allEntries := d.networkBuf.All()
	var filtered []ipc.NetworkEntry
	for _, e := range allEntries {
		if e.SessionID == activeID {
			filtered = append(filtered, e)
		}
	}

	return ipc.SuccessResponse(ipc.NetworkData{
		Entries: filtered,
		Count:   len(filtered),
	})
}

// handleScreenshot captures a screenshot of the active session.
func (d *Daemon) handleScreenshot(req ipc.Request) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	// Parse screenshot parameters
	var params ipc.ScreenshotParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("invalid screenshot parameters: %v", err))
		}
	}

	// Build CDP request parameters
	cdpParams := map[string]any{
		"format": "png",
	}

	// Add captureBeyondViewport for full-page screenshots
	if params.FullPage {
		cdpParams["captureBeyondViewport"] = true
	}

	// Call Page.captureScreenshot
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := d.cdp.SendToSession(ctx, activeID, "Page.captureScreenshot", cdpParams)
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to capture screenshot: %v", err))
	}

	// Parse CDP response
	var cdpResp struct {
		Data string `json:"data"` // base64-encoded PNG
	}
	if err := json.Unmarshal(result, &cdpResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse screenshot response: %v", err))
	}

	// Decode base64 data
	pngData, err := base64.StdEncoding.DecodeString(cdpResp.Data)
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to decode screenshot data: %v", err))
	}

	return ipc.SuccessResponse(ipc.ScreenshotData{
		Data: pngData,
	})
}

// handleHTML extracts HTML from the current page or specified selector.
// Gets window ObjectID first, then uses Runtime.callFunctionOn.
// This avoids the networkIdle blocking that occurs with direct Runtime.evaluate.
func (d *Daemon) handleHTML(req ipc.Request) ipc.Response {
	d.debugf("handleHTML called")
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	// Parse HTML parameters
	var params ipc.HTMLParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("invalid html parameters: %v", err))
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get full page HTML or query selector
	if params.Selector == "" {
		start := time.Now()

		// NOTE: We do NOT call Page.stopLoading here. Testing showed it blocks for 10 seconds.
		// The issue is that Chrome blocks CDP method calls during page load.

		// Step 1: Get window ObjectID using Runtime.evaluate.
		// Chrome handles "window" specially - it's always available.
		d.debugf("html: calling Runtime.evaluate for window")
		windowResult, err := d.cdp.SendToSession(ctx, activeID, "Runtime.evaluate", map[string]any{
			"expression": "window",
		})
		d.debugf("html: Runtime.evaluate(window) completed in %v", time.Since(start))
		if err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("failed to get window: %v", err))
		}

		var windowResp struct {
			Result struct {
				ObjectID string `json:"objectId"`
			} `json:"result"`
			ExceptionDetails *struct {
				Text string `json:"text"`
			} `json:"exceptionDetails"`
		}
		if err := json.Unmarshal(windowResult, &windowResp); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("failed to parse window response: %v", err))
		}
		if windowResp.ExceptionDetails != nil {
			return ipc.ErrorResponse(fmt.Sprintf("JavaScript error getting window: %s", windowResp.ExceptionDetails.Text))
		}
		if windowResp.Result.ObjectID == "" {
			return ipc.ErrorResponse("window objectId is empty")
		}

		// Step 2: Use Runtime.callFunctionOn to get document.documentElement.
		// By targeting the window object directly, we avoid context creation delays.
		d.debugf("html: calling Runtime.callFunctionOn for document.documentElement")
		callStart := time.Now()
		callResult, err := d.cdp.SendToSession(ctx, activeID, "Runtime.callFunctionOn", map[string]any{
			"objectId":            windowResp.Result.ObjectID,
			"functionDeclaration": "function() { return document.documentElement; }",
			"returnByValue":       false,
		})
		d.debugf("html: Runtime.callFunctionOn completed in %v", time.Since(callStart))
		if err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("failed to get documentElement: %v", err))
		}

		var callResp struct {
			Result struct {
				ObjectID string `json:"objectId"`
			} `json:"result"`
			ExceptionDetails *struct {
				Text string `json:"text"`
			} `json:"exceptionDetails"`
		}
		if err := json.Unmarshal(callResult, &callResp); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("failed to parse callFunctionOn response: %v", err))
		}
		if callResp.ExceptionDetails != nil {
			return ipc.ErrorResponse(fmt.Sprintf("JavaScript error: %s", callResp.ExceptionDetails.Text))
		}
		if callResp.Result.ObjectID == "" {
			return ipc.ErrorResponse("documentElement objectId is empty")
		}

		// Step 3: Get outer HTML using DOM.getOuterHTML with the ObjectID.
		d.debugf("html: calling DOM.getOuterHTML with objectId=%s", callResp.Result.ObjectID)
		htmlStart := time.Now()
		htmlResult, err := d.cdp.SendToSession(ctx, activeID, "DOM.getOuterHTML", map[string]any{
			"objectId": callResp.Result.ObjectID,
		})
		d.debugf("html: DOM.getOuterHTML completed in %v", time.Since(htmlStart))
		if err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("failed to get outer HTML: %v", err))
		}

		var htmlResp struct {
			OuterHTML string `json:"outerHTML"`
		}
		if err := json.Unmarshal(htmlResult, &htmlResp); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("failed to parse HTML response: %v", err))
		}

		d.debugf("html: total time: %v", time.Since(start))

		return ipc.SuccessResponse(ipc.HTMLData{
			HTML: htmlResp.OuterHTML,
		})
	}

	// For selector queries, use JavaScript querySelectorAll with Promise-based wait
	js := fmt.Sprintf(`(function() {
		return new Promise((resolve, reject) => {
			const queryElements = () => {
				const elements = document.querySelectorAll(%q);
				if (elements.length === 0) {
					resolve(null);
					return;
				}
				const results = [];
				elements.forEach((el, i) => {
					if (elements.length > 1) {
						results.push('<!-- Element ' + (i+1) + ' of ' + elements.length + ': %s -->');
					}
					results.push(el.outerHTML);
				});
				resolve(results.join('\n\n'));
			};

			if (document.readyState === 'complete') {
				queryElements();
			} else {
				let resolved = false;
				const onLoad = () => {
					if (!resolved) {
						resolved = true;
						queryElements();
					}
				};
				window.addEventListener('load', onLoad);
				if (document.readyState === 'interactive') {
					setTimeout(() => {
						if (!resolved) {
							resolved = true;
							window.removeEventListener('load', onLoad);
							queryElements();
						}
					}, 100);
				}
			}
		});
	})()`, params.Selector, params.Selector)

	result, err := d.cdp.SendToSession(ctx, activeID, "Runtime.evaluate", map[string]any{
		"expression":    js,
		"returnByValue": true,
		"awaitPromise":  true,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to query selector: %v", err))
	}

	// Parse result - null means no matches, string means success
	var evalResp struct {
		Result struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse query response: %v", err))
	}
	if evalResp.ExceptionDetails != nil {
		return ipc.ErrorResponse(fmt.Sprintf("JavaScript error: %s", evalResp.ExceptionDetails.Text))
	}
	// null result means no elements matched
	if evalResp.Result.Type == "object" && evalResp.Result.Value == "" {
		return ipc.ErrorResponse(fmt.Sprintf("selector '%s' matched no elements", params.Selector))
	}

	return ipc.SuccessResponse(ipc.HTMLData{
		HTML: evalResp.Result.Value,
	})
}

// handleEval evaluates JavaScript in the browser context.
func (d *Daemon) handleEval(req ipc.Request) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	var params ipc.EvalParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("invalid eval parameters: %v", err))
	}

	if params.Expression == "" {
		return ipc.ErrorResponse("expression is required")
	}

	timeout := cdp.DefaultTimeout
	if params.Timeout > 0 {
		timeout = time.Duration(params.Timeout) * time.Millisecond
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := d.cdp.SendToSession(ctx, activeID, "Runtime.evaluate", map[string]any{
		"expression":    params.Expression,
		"awaitPromise":  true,
		"returnByValue": true,
	})
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return ipc.ErrorResponse(fmt.Sprintf("evaluation timed out after %s", timeout))
		}
		return ipc.ErrorResponse(fmt.Sprintf("failed to evaluate expression: %v", err))
	}

	// Parse the CDP response
	var cdpResp struct {
		Result struct {
			Type  string `json:"type"`
			Value any    `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text      string `json:"text"`
			Exception struct {
				Description string `json:"description"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(result, &cdpResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse evaluation result: %v", err))
	}

	// Check for JavaScript errors
	if cdpResp.ExceptionDetails != nil {
		errMsg := cdpResp.ExceptionDetails.Exception.Description
		if errMsg == "" {
			errMsg = cdpResp.ExceptionDetails.Text
		}
		return ipc.ErrorResponse(errMsg)
	}

	// Return the result - omit value field if undefined
	if cdpResp.Result.Type == "undefined" {
		return ipc.SuccessResponse(ipc.EvalData{HasValue: false})
	}

	return ipc.SuccessResponse(ipc.EvalData{Value: cdpResp.Result.Value, HasValue: true})
}

// handleCDP forwards a raw CDP command to the browser.
// Request format: {"cmd": "cdp", "target": "Method.name", "params": {...}}
// Commands are sent to the active session. Use Target.* methods for browser-level commands.
func (d *Daemon) handleCDP(req ipc.Request) ipc.Response {
	if req.Target == "" {
		return ipc.ErrorResponse("cdp command requires target (CDP method name)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var params any
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("invalid params: %v", err))
		}
	}

	// Target.* methods are browser-level, send without session ID
	// All other methods go to the active session
	var result json.RawMessage
	var err error

	if strings.HasPrefix(req.Target, "Target.") {
		result, err = d.cdp.SendContext(ctx, req.Target, params)
	} else {
		activeID := d.sessions.ActiveID()
		if activeID == "" {
			return d.noActiveSessionError()
		}
		result, err = d.cdp.SendToSession(ctx, activeID, req.Target, params)
	}

	if err != nil {
		return ipc.ErrorResponse(err.Error())
	}

	return ipc.Response{OK: true, Data: result}
}
