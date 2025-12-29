package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
)

// handleCSS manages CSS operations (save, computed, get).
func (d *Daemon) handleCSS(req ipc.Request) ipc.Response {
	// Check if browser is connected (fail-fast if not)
	if ok, resp := d.requireBrowser(); !ok {
		return resp
	}

	activeID := d.sessions.ActiveID()
	if activeID == "" {
		return d.noActiveSessionError()
	}

	var params ipc.CSSParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("invalid css parameters: %v", err))
	}

	switch params.Action {
	case "save":
		return d.handleCSSSave(activeID, params)
	case "computed":
		return d.handleCSSComputed(activeID, params)
	case "get":
		return d.handleCSSGet(activeID, params)
	default:
		return ipc.ErrorResponse(fmt.Sprintf("unknown css action: %s", params.Action))
	}
}

// handleCSSSave extracts CSS (all stylesheets or computed styles for selector).
func (d *Daemon) handleCSSSave(sessionID string, params ipc.CSSParams) ipc.Response {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if params.Selector == "" {
		// Extract all stylesheets
		js := `(function() {
			const results = [];
			try {
				for (let i = 0; i < document.styleSheets.length; i++) {
					const sheet = document.styleSheets[i];
					try {
						const rules = Array.from(sheet.cssRules || sheet.rules || []);
						const css = rules.map(rule => rule.cssText).join('\n');
						if (css) {
							results.push(css);
						}
					} catch (e) {
						// Cross-origin stylesheet - cannot access
						results.push('/* Stylesheet from ' + (sheet.href || 'inline') + ' - blocked by CORS */');
					}
				}
			} catch (e) {
				return 'Error: ' + e.message;
			}
			return results.join('\n\n');
		})()`

		result, err := d.sendToSession(ctx, sessionID, "Runtime.evaluate", map[string]any{
			"expression":    js,
			"returnByValue": true,
		})
		if err != nil {
			return ipc.ErrorResponse(fmt.Sprintf("failed to extract CSS: %v", err))
		}

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
			return ipc.ErrorResponse(fmt.Sprintf("failed to parse CSS response: %v", err))
		}
		if evalResp.ExceptionDetails != nil {
			return ipc.ErrorResponse(fmt.Sprintf("JavaScript error: %s", evalResp.ExceptionDetails.Text))
		}

		return ipc.SuccessResponse(ipc.CSSData{
			CSS: evalResp.Result.Value,
		})
	}

	// Extract computed styles for selector
	return d.handleCSSComputed(sessionID, params)
}

// handleCSSComputed gets computed styles for a selector.
func (d *Daemon) handleCSSComputed(sessionID string, params ipc.CSSParams) ipc.Response {
	if params.Selector == "" {
		return ipc.ErrorResponse("selector is required for computed styles")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get computed styles for element
	js := fmt.Sprintf(`(function() {
		const element = document.querySelector(%q);
		if (!element) {
			return null;
		}
		const styles = window.getComputedStyle(element);
		const result = {};
		for (let i = 0; i < styles.length; i++) {
			const prop = styles[i];
			result[prop] = styles.getPropertyValue(prop);
		}
		return result;
	})()`, params.Selector)

	result, err := d.sendToSession(ctx, sessionID, "Runtime.evaluate", map[string]any{
		"expression":    js,
		"returnByValue": true,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to get computed styles: %v", err))
	}

	var evalResp struct {
		Result struct {
			Type  string            `json:"type"`
			Value map[string]string `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse computed styles response: %v", err))
	}
	if evalResp.ExceptionDetails != nil {
		return ipc.ErrorResponse(fmt.Sprintf("JavaScript error: %s", evalResp.ExceptionDetails.Text))
	}

	// null result means no element matched
	if evalResp.Result.Type == "object" && evalResp.Result.Value == nil {
		return ipc.ErrorResponse(fmt.Sprintf("selector '%s' matched no elements", params.Selector))
	}

	return ipc.SuccessResponse(ipc.CSSData{
		Styles: evalResp.Result.Value,
	})
}

// handleCSSGet gets a single CSS property value for a selector.
func (d *Daemon) handleCSSGet(sessionID string, params ipc.CSSParams) ipc.Response {
	if params.Selector == "" {
		return ipc.ErrorResponse("selector is required")
	}
	if params.Property == "" {
		return ipc.ErrorResponse("property is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get single property value
	js := fmt.Sprintf(`(function() {
		const element = document.querySelector(%q);
		if (!element) {
			return null;
		}
		return window.getComputedStyle(element).getPropertyValue(%q);
	})()`, params.Selector, params.Property)

	result, err := d.sendToSession(ctx, sessionID, "Runtime.evaluate", map[string]any{
		"expression":    js,
		"returnByValue": true,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to get property: %v", err))
	}

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
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse property response: %v", err))
	}
	if evalResp.ExceptionDetails != nil {
		return ipc.ErrorResponse(fmt.Sprintf("JavaScript error: %s", evalResp.ExceptionDetails.Text))
	}

	// null result means no element matched
	if evalResp.Result.Type == "object" {
		return ipc.ErrorResponse(fmt.Sprintf("selector '%s' matched no elements", params.Selector))
	}

	return ipc.SuccessResponse(ipc.CSSData{
		Value: evalResp.Result.Value,
	})
}

