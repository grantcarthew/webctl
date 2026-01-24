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
	sessions := d.sessions.All()

	// Look up HTTP status for each session from network buffer
	d.enrichSessionsWithHTTPStatus(sessions)

	status := ipc.StatusData{
		Running:  true,
		PID:      os.Getpid(),
		Sessions: sessions,
	}

	// Get active session info (find it in the already-enriched sessions list)
	for i := range sessions {
		if sessions[i].Active {
			status.ActiveSession = &sessions[i]
			break
		}
	}

	return ipc.SuccessResponse(status)
}

// enrichSessionsWithHTTPStatus looks up the HTTP status code for each session
// from the network buffer. Finds the most recent Document-type request matching
// each session's URL.
func (d *Daemon) enrichSessionsWithHTTPStatus(sessions []ipc.PageSession) {
	if len(sessions) == 0 {
		return
	}

	// Build a map of URL -> most recent Document status
	// Network entries are ordered oldest-to-newest, so later entries overwrite
	urlStatus := make(map[string]int)
	for _, entry := range d.networkBuf.All() {
		if entry.Type == "Document" && entry.Status > 0 {
			urlStatus[entry.URL] = entry.Status
		}
	}

	// Apply status to sessions
	for i := range sessions {
		if status, ok := urlStatus[sessions[i].URL]; ok {
			sessions[i].Status = status
		}
	}
}

// handleConsole returns buffered console entries filtered to active session.
func (d *Daemon) handleConsole() ipc.Response {
	// Check if browser is connected (fail-fast if not)
	if ok, resp := d.requireBrowser(); !ok {
		return resp
	}

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
	// Check if browser is connected (fail-fast if not)
	if ok, resp := d.requireBrowser(); !ok {
		return resp
	}

	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	// Enable Network domain lazily for this session
	if _, loaded := d.networkEnabled.LoadOrStore(activeID, true); !loaded {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := d.sendToSession(ctx, activeID, "Network.enable", nil); err != nil {
			d.debugf(false, "warning: failed to enable Network domain: %v", err)
		} else {
			d.debugf(false, "Network domain enabled lazily for session %s", activeID)
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
	// Check if browser is connected (fail-fast if not)
	if ok, resp := d.requireBrowser(); !ok {
		return resp
	}

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

	result, err := d.sendToSession(ctx, activeID, "Page.captureScreenshot", cdpParams)
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
	d.debugf(false, "handleHTML called")

	// Check if browser is connected (fail-fast if not)
	if ok, resp := d.requireBrowser(); !ok {
		return resp
	}

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
		d.debugf(false, "html: calling Runtime.evaluate for window")
		windowResult, err := d.sendToSession(ctx, activeID, "Runtime.evaluate", map[string]any{
			"expression": "window",
		})
		d.debugf(false, "html: Runtime.evaluate(window) completed in %v", time.Since(start))
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
		d.debugf(false, "html: calling Runtime.callFunctionOn for document.documentElement")
		callStart := time.Now()
		callResult, err := d.sendToSession(ctx, activeID, "Runtime.callFunctionOn", map[string]any{
			"objectId":            windowResp.Result.ObjectID,
			"functionDeclaration": "function() { return document.documentElement; }",
			"returnByValue":       false,
		})
		d.debugf(false, "html: Runtime.callFunctionOn completed in %v", time.Since(callStart))
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
		d.debugf(false, "html: calling DOM.getOuterHTML with objectId=%s", callResp.Result.ObjectID)
		htmlStart := time.Now()
		htmlResult, err := d.sendToSession(ctx, activeID, "DOM.getOuterHTML", map[string]any{
			"objectId": callResp.Result.ObjectID,
		})
		d.debugf(false, "html: DOM.getOuterHTML completed in %v", time.Since(htmlStart))
		if err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("failed to get outer HTML: %v", err))
		}

		var htmlResp struct {
			OuterHTML string `json:"outerHTML"`
		}
		if err := json.Unmarshal(htmlResult, &htmlResp); err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("failed to parse HTML response: %v", err))
		}

		d.debugf(false, "html: total time: %v", time.Since(start))

		return ipc.SuccessResponse(ipc.HTMLData{
			HTML: htmlResp.OuterHTML,
		})
	}

	// For selector queries, use JavaScript querySelectorAll with Promise-based wait
	js := fmt.Sprintf(`(function() {
		// Extract element metadata (tag, id, first class)
		function getElementMeta(el) {
			const id = (el.id || '').trim();
			const classAttr = el.getAttribute('class');
			const classes = (classAttr || '')
				.split(/\s+/)
				.map(c => c.trim())
				.filter(c => c.length > 0);
			const firstClass = classes.length > 0 ? classes[0] : null;

			return {
				tag: el.tagName.toLowerCase(),
				id: id || null,
				class: firstClass
			};
		}

		return new Promise((resolve, reject) => {
			const queryElements = () => {
				const elements = document.querySelectorAll(%q);
				if (elements.length === 0) {
					resolve(null);
					return;
				}
				resolve(Array.from(elements).map((el) => ({
					...getElementMeta(el),
					html: el.outerHTML
				})));
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
	})()`, params.Selector)

	result, err := d.sendToSession(ctx, activeID, "Runtime.evaluate", map[string]any{
		"expression":    js,
		"returnByValue": true,
		"awaitPromise":  true,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to query selector: %v", err))
	}

	// Parse result - null means no matches, array means success
	var evalResp struct {
		Result struct {
			Type  string                `json:"type"`
			Value []ipc.ElementWithHTML `json:"value"`
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
	if evalResp.Result.Type == "object" && evalResp.Result.Value == nil {
		return ipc.ErrorResponse(fmt.Sprintf("selector '%s' matched no elements", params.Selector))
	}

	// Build legacy HTML field with -- separators for backward compatibility
	// For N elements: N HTML strings + (N-1) separators = 2N-1 elements
	htmlParts := make([]string, 0, len(evalResp.Result.Value)*2-1)
	for i, elem := range evalResp.Result.Value {
		if i > 0 {
			htmlParts = append(htmlParts, ipc.MultiElementSeparator)
		}
		htmlParts = append(htmlParts, elem.HTML)
	}

	return ipc.SuccessResponse(ipc.HTMLData{
		HTML:      strings.Join(htmlParts, "\n"),
		HTMLMulti: evalResp.Result.Value,
	})
}

// handleEval evaluates JavaScript in the browser context.
func (d *Daemon) handleEval(req ipc.Request) ipc.Response {
	// Check if browser is connected (fail-fast if not)
	if ok, resp := d.requireBrowser(); !ok {
		return resp
	}

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
		timeout = time.Duration(params.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := d.sendToSession(ctx, activeID, "Runtime.evaluate", map[string]any{
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

// handleCookies manages browser cookies (list, set, delete).
func (d *Daemon) handleCookies(req ipc.Request) ipc.Response {
	// Check if browser is connected (fail-fast if not)
	if ok, resp := d.requireBrowser(); !ok {
		return resp
	}

	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	var params ipc.CookiesParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("invalid cookies parameters: %v", err))
	}

	switch params.Action {
	case "list":
		return d.handleCookiesList(activeID)
	case "set":
		return d.handleCookiesSet(activeID, params)
	case "delete":
		return d.handleCookiesDelete(activeID, params)
	default:
		return ipc.ErrorResponse(fmt.Sprintf("unknown cookies action: %s", params.Action))
	}
}

// handleCookiesList retrieves all cookies for the active session.
func (d *Daemon) handleCookiesList(sessionID string) ipc.Response {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := d.sendToSession(ctx, sessionID, "Network.getCookies", map[string]any{})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to get cookies: %v", err))
	}

	var cdpResp struct {
		Cookies []ipc.Cookie `json:"cookies"`
	}
	if err := json.Unmarshal(result, &cdpResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse cookies response: %v", err))
	}

	return ipc.SuccessResponse(ipc.CookiesData{
		Cookies: cdpResp.Cookies,
		Count:   len(cdpResp.Cookies),
	})
}

// handleCookiesSet sets a cookie in the active session.
func (d *Daemon) handleCookiesSet(sessionID string, params ipc.CookiesParams) ipc.Response {
	if params.Name == "" {
		return ipc.ErrorResponse("cookie name is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get current URL from session - CDP requires either url or domain
	session := d.sessions.Get(sessionID)
	if session == nil || session.URL == "" {
		return ipc.ErrorResponse("no active page URL")
	}

	// Build CDP params
	cdpParams := map[string]any{
		"name":  params.Name,
		"value": params.Value,
		"url":   session.URL, // CDP uses URL to determine the domain
	}

	// Override domain if explicitly provided
	if params.Domain != "" {
		cdpParams["domain"] = params.Domain
	}

	if params.Path != "" {
		cdpParams["path"] = params.Path
	}

	if params.Secure {
		cdpParams["secure"] = true
	}

	if params.HTTPOnly {
		cdpParams["httpOnly"] = true
	}

	if params.SameSite != "" {
		cdpParams["sameSite"] = params.SameSite
	}

	// Convert max-age to expires timestamp
	if params.MaxAge > 0 {
		cdpParams["expires"] = float64(time.Now().Unix() + int64(params.MaxAge))
	}

	result, err := d.sendToSession(ctx, sessionID, "Network.setCookie", cdpParams)
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to set cookie: %v", err))
	}

	var cdpResp struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(result, &cdpResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse set cookie response: %v", err))
	}

	if !cdpResp.Success {
		return ipc.ErrorResponse("failed to set cookie (CDP reported failure)")
	}

	return ipc.SuccessResponse(nil)
}

// handleCookiesDelete deletes a cookie from the active session.
func (d *Daemon) handleCookiesDelete(sessionID string, params ipc.CookiesParams) ipc.Response {
	if params.Name == "" {
		return ipc.ErrorResponse("cookie name is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// First, get all cookies to find matches
	result, err := d.sendToSession(ctx, sessionID, "Network.getCookies", map[string]any{})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to get cookies: %v", err))
	}

	var cdpResp struct {
		Cookies []ipc.Cookie `json:"cookies"`
	}
	if err := json.Unmarshal(result, &cdpResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse cookies response: %v", err))
	}

	// Find matches by name
	var matches []ipc.Cookie
	for _, cookie := range cdpResp.Cookies {
		if cookie.Name == params.Name {
			matches = append(matches, cookie)
		}
	}

	// No matches - idempotent success
	if len(matches) == 0 {
		return ipc.SuccessResponse(nil)
	}

	// Multiple matches without domain specified - error
	if len(matches) > 1 && params.Domain == "" {
		resp := ipc.ErrorResponse(fmt.Sprintf("multiple cookies named '%s' found", params.Name))
		resp.Data, _ = json.Marshal(ipc.CookiesData{Matches: matches})
		return resp
	}

	// Find the cookie to delete
	var targetCookie *ipc.Cookie
	if len(matches) == 1 {
		targetCookie = &matches[0]
	} else {
		// Multiple matches with domain specified
		for i := range matches {
			if matches[i].Domain == params.Domain {
				targetCookie = &matches[i]
				break
			}
		}
		if targetCookie == nil {
			return ipc.ErrorResponse(fmt.Sprintf("no cookie named '%s' found with domain '%s'", params.Name, params.Domain))
		}
	}

	// Delete the cookie
	deleteParams := map[string]any{
		"name":   targetCookie.Name,
		"domain": targetCookie.Domain,
	}

	_, err = d.sendToSession(ctx, sessionID, "Network.deleteCookies", deleteParams)
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to delete cookie: %v", err))
	}

	return ipc.SuccessResponse(nil)
}

// handleFind searches HTML content for text patterns.
// handleFind has been removed per DR-030 (Find Command Removal).
// The standalone 'find' command has been replaced with universal --find flags
// on all observation commands (html, css, console, network, cookies).
//
// If this handler is ever called, it returns an error directing users to use
// the html --find flag instead.
func (d *Daemon) handleFind(req ipc.Request) ipc.Response {
	return ipc.ErrorResponse("find command has been removed - use 'html --find <text>' instead (see DR-030)")
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
		result, err = d.sendToSession(ctx, activeID, req.Target, params)
	}

	if err != nil {
		return ipc.ErrorResponse(err.Error())
	}

	return ipc.Response{OK: true, Data: result}
}
