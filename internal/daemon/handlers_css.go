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
	case "inline":
		return d.handleCSSInline(activeID, params)
	case "matched":
		return d.handleCSSMatched(activeID, params)
	default:
		return ipc.ErrorResponse(fmt.Sprintf("unknown css action: %s", params.Action))
	}
}

// handleCSSSave extracts all stylesheets from the page.
// The selector filtering is now done client-side using FilterRulesBySelector.
func (d *Daemon) handleCSSSave(sessionID string, _ ipc.CSSParams) ipc.Response {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Always extract all stylesheets - selector filtering is done in CLI
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

// handleCSSComputed gets computed styles for all matching elements.
func (d *Daemon) handleCSSComputed(sessionID string, params ipc.CSSParams) ipc.Response {
	if params.Selector == "" {
		return ipc.ErrorResponse("selector is required for computed styles")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get computed styles for all matching elements
	js := fmt.Sprintf(`(function() {
		const elements = document.querySelectorAll(%q);
		if (elements.length === 0) {
			return null;
		}
		const results = [];
		for (const element of elements) {
			const styles = window.getComputedStyle(element);
			const result = {};
			for (let i = 0; i < styles.length; i++) {
				const prop = styles[i];
				result[prop] = styles.getPropertyValue(prop);
			}
			results.push(result);
		}
		return results;
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
			Type  string              `json:"type"`
			Value []map[string]string `json:"value"`
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

	// For backward compatibility, also set Styles if there's only one element
	var styles map[string]string
	if len(evalResp.Result.Value) == 1 {
		styles = evalResp.Result.Value[0]
	}

	return ipc.SuccessResponse(ipc.CSSData{
		Styles:        styles,
		ComputedMulti: evalResp.Result.Value,
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

// handleCSSInline gets inline style attributes for matching elements.
func (d *Daemon) handleCSSInline(sessionID string, params ipc.CSSParams) ipc.Response {
	if params.Selector == "" {
		return ipc.ErrorResponse("selector is required for inline styles")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get inline style attributes for all matching elements
	js := fmt.Sprintf(`(function() {
		const elements = document.querySelectorAll(%q);
		if (elements.length === 0) {
			return null;
		}
		const styles = [];
		for (const el of elements) {
			const style = el.getAttribute('style') || '';
			styles.push(style);
		}
		return styles;
	})()`, params.Selector)

	result, err := d.sendToSession(ctx, sessionID, "Runtime.evaluate", map[string]any{
		"expression":    js,
		"returnByValue": true,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to get inline styles: %v", err))
	}

	var evalResp struct {
		Result struct {
			Type  string   `json:"type"`
			Value []string `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(result, &evalResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse inline styles response: %v", err))
	}
	if evalResp.ExceptionDetails != nil {
		return ipc.ErrorResponse(fmt.Sprintf("JavaScript error: %s", evalResp.ExceptionDetails.Text))
	}

	// null result means no element matched
	if evalResp.Result.Type == "object" && evalResp.Result.Value == nil {
		return ipc.ErrorResponse(fmt.Sprintf("selector '%s' matched no elements", params.Selector))
	}

	return ipc.SuccessResponse(ipc.CSSData{
		Inline: evalResp.Result.Value,
	})
}

// handleCSSMatched gets matched CSS rules for an element using CDP CSS.getMatchedStylesForNode.
func (d *Daemon) handleCSSMatched(sessionID string, params ipc.CSSParams) ipc.Response {
	if params.Selector == "" {
		return ipc.ErrorResponse("selector is required for matched styles")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First, we need to get the node ID for the element
	// Enable CSS domain first
	_, err := d.sendToSession(ctx, sessionID, "CSS.enable", nil)
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to enable CSS domain: %v", err))
	}

	// Get the document root
	docResult, err := d.sendToSession(ctx, sessionID, "DOM.getDocument", map[string]any{
		"depth": 0,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to get document: %v", err))
	}

	var docResp struct {
		Root struct {
			NodeID int `json:"nodeId"`
		} `json:"root"`
	}
	if err := json.Unmarshal(docResult, &docResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse document response: %v", err))
	}

	// Query for the element
	queryResult, err := d.sendToSession(ctx, sessionID, "DOM.querySelector", map[string]any{
		"nodeId":   docResp.Root.NodeID,
		"selector": params.Selector,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to query selector: %v", err))
	}

	var queryResp struct {
		NodeID int `json:"nodeId"`
	}
	if err := json.Unmarshal(queryResult, &queryResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse query response: %v", err))
	}

	if queryResp.NodeID == 0 {
		return ipc.ErrorResponse(fmt.Sprintf("selector '%s' matched no elements", params.Selector))
	}

	// Get matched styles for the node
	matchedResult, err := d.sendToSession(ctx, sessionID, "CSS.getMatchedStylesForNode", map[string]any{
		"nodeId": queryResp.NodeID,
	})
	if err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to get matched styles: %v", err))
	}

	// Parse the matched styles response
	var matchedResp struct {
		MatchedCSSRules []struct {
			Rule struct {
				SelectorList struct {
					Text string `json:"text"`
				} `json:"selectorList"`
				Style struct {
					CSSProperties []struct {
						Name  string `json:"name"`
						Value string `json:"value"`
					} `json:"cssProperties"`
				} `json:"style"`
				Origin     string `json:"origin"`
				StyleSheetId string `json:"styleSheetId"`
			} `json:"rule"`
		} `json:"matchedCSSRules"`
		InlineStyle *struct {
			CSSProperties []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"cssProperties"`
		} `json:"inlineStyle"`
	}
	if err := json.Unmarshal(matchedResult, &matchedResp); err != nil {
		return ipc.ErrorResponse(fmt.Sprintf("failed to parse matched styles response: %v", err))
	}

	// Build the response
	var rules []ipc.CSSMatchedRule

	// Add inline styles first if present
	if matchedResp.InlineStyle != nil && len(matchedResp.InlineStyle.CSSProperties) > 0 {
		props := make(map[string]string)
		for _, p := range matchedResp.InlineStyle.CSSProperties {
			if p.Name != "" && p.Value != "" {
				props[p.Name] = p.Value
			}
		}
		if len(props) > 0 {
			rules = append(rules, ipc.CSSMatchedRule{
				Selector:   "(inline)",
				Properties: props,
				Source:     "inline",
			})
		}
	}

	// Add matched CSS rules
	for _, match := range matchedResp.MatchedCSSRules {
		// Skip user-agent stylesheets
		if match.Rule.Origin == "user-agent" {
			continue
		}
		props := make(map[string]string)
		for _, p := range match.Rule.Style.CSSProperties {
			if p.Name != "" && p.Value != "" {
				props[p.Name] = p.Value
			}
		}
		if len(props) > 0 {
			rules = append(rules, ipc.CSSMatchedRule{
				Selector:   match.Rule.SelectorList.Text,
				Properties: props,
			})
		}
	}

	return ipc.SuccessResponse(ipc.CSSData{
		Matched: rules,
	})
}

