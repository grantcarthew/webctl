package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
)

// handleClick clicks an element by selector.
// Scrolls element into view, checks visibility, then dispatches mouse events.
func (d *Daemon) handleClick(req ipc.Request) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	var params ipc.ClickParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("invalid click parameters: %v", err))
	}

	if params.Selector == "" {
		return ipc.ErrorResponse("selector is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Scroll element into view, get coordinates, and check if covered
	js := fmt.Sprintf(`(() => {
		const el = document.querySelector(%q);
		if (!el) return {error: 'not_found'};

		// Scroll into view
		el.scrollIntoView({block: 'center', behavior: 'instant'});

		// Get center coordinates
		const rect = el.getBoundingClientRect();
		const x = rect.left + rect.width / 2;
		const y = rect.top + rect.height / 2;

		// Check if element is covered by something else
		const topEl = document.elementFromPoint(x, y);
		const isCovered = topEl !== el && !el.contains(topEl);

		return {x, y, covered: isCovered};
	})()`, params.Selector)

	result, err := d.cdp.SendToSession(ctx, activeID, "Runtime.evaluate", map[string]any{
		"expression":    js,
		"returnByValue": true,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to find element: %v", err))
	}

	var evalResp struct {
		Result struct {
			Type  string `json:"type"`
			Value struct {
				Error   string  `json:"error"`
				X       float64 `json:"x"`
				Y       float64 `json:"y"`
				Covered bool    `json:"covered"`
			} `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse element position: %v", err))
	}
	if evalResp.Result.Type == "undefined" || evalResp.Result.Value.Error == "not_found" {
		return ipc.ErrorResponse(fmt.Sprintf("element not found: %s", params.Selector))
	}

	x := evalResp.Result.Value.X
	y := evalResp.Result.Value.Y
	covered := evalResp.Result.Value.Covered

	// Send mouse events
	// mousePressed
	_, err = d.cdp.SendToSession(ctx, activeID, "Input.dispatchMouseEvent", map[string]any{
		"type":       "mousePressed",
		"x":          x,
		"y":          y,
		"button":     "left",
		"clickCount": 1,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to click: %v", err))
	}

	// mouseReleased
	_, err = d.cdp.SendToSession(ctx, activeID, "Input.dispatchMouseEvent", map[string]any{
		"type":       "mouseReleased",
		"x":          x,
		"y":          y,
		"button":     "left",
		"clickCount": 1,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to click: %v", err))
	}

	// Return success with optional warning if element was covered
	if covered {
		return ipc.SuccessResponse(map[string]any{
			"warning": fmt.Sprintf("element may be covered by another element: %s", params.Selector),
		})
	}

	return ipc.SuccessResponse(nil)
}

// handleFocus focuses an element by selector.
func (d *Daemon) handleFocus(req ipc.Request) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	var params ipc.FocusParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("invalid focus parameters: %v", err))
	}

	if params.Selector == "" {
		return ipc.ErrorResponse("selector is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Focus using JavaScript
	js := fmt.Sprintf(`(() => {
		const el = document.querySelector(%q);
		if (!el) return false;
		el.focus();
		return true;
	})()`, params.Selector)

	result, err := d.cdp.SendToSession(ctx, activeID, "Runtime.evaluate", map[string]any{
		"expression":    js,
		"returnByValue": true,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to focus element: %v", err))
	}

	var evalResp struct {
		Result struct {
			Value bool `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse focus result: %v", err))
	}
	if !evalResp.Result.Value {
		return ipc.ErrorResponse(fmt.Sprintf("element not found: %s", params.Selector))
	}

	return ipc.SuccessResponse(nil)
}

// handleType types text into an element.
func (d *Daemon) handleType(req ipc.Request) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	var params ipc.TypeParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("invalid type parameters: %v", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// If selector provided, focus the element first
	if params.Selector != "" {
		focusResp := d.handleFocus(ipc.Request{
			Params: func() json.RawMessage {
				b, _ := json.Marshal(ipc.FocusParams{Selector: params.Selector})
				return b
			}(),
		})
		if !focusResp.OK {
			return focusResp
		}
	}

	// If clear flag, send select-all then Backspace
	// Use Meta+A on macOS, Ctrl+A on Linux
	if params.Clear {
		// Select all (OS-aware)
		selectAllParams := ipc.KeyParams{Key: "a"}
		if runtime.GOOS == "darwin" {
			selectAllParams.Meta = true
		} else {
			selectAllParams.Ctrl = true
		}
		keyResp := d.handleKey(ipc.Request{
			Params: func() json.RawMessage {
				b, _ := json.Marshal(selectAllParams)
				return b
			}(),
		})
		if !keyResp.OK {
			return keyResp
		}
		// Delete
		keyResp = d.handleKey(ipc.Request{
			Params: func() json.RawMessage {
				b, _ := json.Marshal(ipc.KeyParams{Key: "Backspace"})
				return b
			}(),
		})
		if !keyResp.OK {
			return keyResp
		}
	}

	// Insert text
	if params.Text != "" {
		_, err := d.cdp.SendToSession(ctx, activeID, "Input.insertText", map[string]any{
			"text": params.Text,
		})
		if err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("failed to type text: %v", err))
		}
	}

	// If key specified, send it
	if params.Key != "" {
		keyResp := d.handleKey(ipc.Request{
			Params: func() json.RawMessage {
				b, _ := json.Marshal(ipc.KeyParams{Key: params.Key})
				return b
			}(),
		})
		if !keyResp.OK {
			return keyResp
		}
	}

	return ipc.SuccessResponse(nil)
}

// handleKey sends a keyboard key event.
func (d *Daemon) handleKey(req ipc.Request) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	var params ipc.KeyParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("invalid key parameters: %v", err))
	}

	if params.Key == "" {
		return ipc.ErrorResponse("key is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Calculate modifiers bitmap: Alt=1, Ctrl=2, Meta=4, Shift=8
	modifiers := 0
	if params.Alt {
		modifiers |= 1
	}
	if params.Ctrl {
		modifiers |= 2
	}
	if params.Meta {
		modifiers |= 4
	}
	if params.Shift {
		modifiers |= 8
	}

	// Map key names to CDP key info
	keyInfo := getKeyInfo(params.Key)

	// keyDown
	_, err := d.cdp.SendToSession(ctx, activeID, "Input.dispatchKeyEvent", map[string]any{
		"type":                  "keyDown",
		"key":                   keyInfo.key,
		"code":                  keyInfo.code,
		"windowsVirtualKeyCode": keyInfo.keyCode,
		"modifiers":             modifiers,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to send key: %v", err))
	}

	// keyUp
	_, err = d.cdp.SendToSession(ctx, activeID, "Input.dispatchKeyEvent", map[string]any{
		"type":                  "keyUp",
		"key":                   keyInfo.key,
		"code":                  keyInfo.code,
		"windowsVirtualKeyCode": keyInfo.keyCode,
		"modifiers":             modifiers,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to send key: %v", err))
	}

	return ipc.SuccessResponse(nil)
}

// keyInfo holds CDP key event parameters.
type keyInfo struct {
	key     string
	code    string
	keyCode int
}

// getKeyInfo returns CDP key parameters for a key name.
func getKeyInfo(key string) keyInfo {
	// Common key mappings
	switch key {
	case "Enter":
		return keyInfo{key: "Enter", code: "Enter", keyCode: 13}
	case "Tab":
		return keyInfo{key: "Tab", code: "Tab", keyCode: 9}
	case "Escape":
		return keyInfo{key: "Escape", code: "Escape", keyCode: 27}
	case "Backspace":
		return keyInfo{key: "Backspace", code: "Backspace", keyCode: 8}
	case "Delete":
		return keyInfo{key: "Delete", code: "Delete", keyCode: 46}
	case "ArrowUp":
		return keyInfo{key: "ArrowUp", code: "ArrowUp", keyCode: 38}
	case "ArrowDown":
		return keyInfo{key: "ArrowDown", code: "ArrowDown", keyCode: 40}
	case "ArrowLeft":
		return keyInfo{key: "ArrowLeft", code: "ArrowLeft", keyCode: 37}
	case "ArrowRight":
		return keyInfo{key: "ArrowRight", code: "ArrowRight", keyCode: 39}
	case "Home":
		return keyInfo{key: "Home", code: "Home", keyCode: 36}
	case "End":
		return keyInfo{key: "End", code: "End", keyCode: 35}
	case "PageUp":
		return keyInfo{key: "PageUp", code: "PageUp", keyCode: 33}
	case "PageDown":
		return keyInfo{key: "PageDown", code: "PageDown", keyCode: 34}
	case "Space":
		return keyInfo{key: " ", code: "Space", keyCode: 32}
	default:
		// Single character keys
		if len(key) == 1 {
			keyCode := int(key[0])
			if key[0] >= 'a' && key[0] <= 'z' {
				keyCode = int(key[0]) - 32 // Convert to uppercase keyCode
			}
			return keyInfo{key: key, code: "Key" + strings.ToUpper(key), keyCode: keyCode}
		}
		// Unknown key, return as-is
		return keyInfo{key: key, code: key, keyCode: 0}
	}
}

// handleSelect selects an option in a dropdown.
func (d *Daemon) handleSelect(req ipc.Request) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	var params ipc.SelectParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("invalid select parameters: %v", err))
	}

	if params.Selector == "" {
		return ipc.ErrorResponse("selector is required")
	}
	if params.Value == "" {
		return ipc.ErrorResponse("value is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Select using JavaScript
	js := fmt.Sprintf(`(() => {
		const el = document.querySelector(%q);
		if (!el) return 'not_found';
		if (el.tagName !== 'SELECT') return 'not_select';
		el.value = %q;
		el.dispatchEvent(new Event('change', {bubbles: true}));
		return 'ok';
	})()`, params.Selector, params.Value)

	result, err := d.cdp.SendToSession(ctx, activeID, "Runtime.evaluate", map[string]any{
		"expression":    js,
		"returnByValue": true,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to select option: %v", err))
	}

	var evalResp struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse select result: %v", err))
	}

	switch evalResp.Result.Value {
	case "not_found":
		return ipc.ErrorResponse(fmt.Sprintf("element not found: %s", params.Selector))
	case "not_select":
		return ipc.ErrorResponse(fmt.Sprintf("element is not a select: %s", params.Selector))
	case "ok":
		return ipc.SuccessResponse(nil)
	default:
		return ipc.ErrorResponse("unexpected select result")
	}
}

// handleScroll scrolls to an element or position.
func (d *Daemon) handleScroll(req ipc.Request) ipc.Response {
	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	var params ipc.ScrollParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("invalid scroll parameters: %v", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var js string
	switch params.Mode {
	case "element":
		if params.Selector == "" {
			return ipc.ErrorResponse("selector is required for element scroll")
		}
		js = fmt.Sprintf(`(() => {
			const el = document.querySelector(%q);
			if (!el) return false;
			el.scrollIntoView({block: 'center', behavior: 'instant'});
			return true;
		})()`, params.Selector)
	case "to":
		js = fmt.Sprintf(`(() => {
			window.scrollTo({left: %d, top: %d, behavior: 'instant'});
			return true;
		})()`, params.ToX, params.ToY)
	case "by":
		js = fmt.Sprintf(`(() => {
			window.scrollBy({left: %d, top: %d, behavior: 'instant'});
			return true;
		})()`, params.ByX, params.ByY)
	default:
		return ipc.ErrorResponse("invalid scroll mode: must be 'element', 'to', or 'by'")
	}

	result, err := d.cdp.SendToSession(ctx, activeID, "Runtime.evaluate", map[string]any{
		"expression":    js,
		"returnByValue": true,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to scroll: %v", err))
	}

	var evalResp struct {
		Result struct {
			Value bool `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse scroll result: %v", err))
	}
	if !evalResp.Result.Value {
		return ipc.ErrorResponse(fmt.Sprintf("element not found: %s", params.Selector))
	}

	return ipc.SuccessResponse(nil)
}
