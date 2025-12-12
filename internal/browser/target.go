package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Target represents a CDP target (page, worker, etc).
type Target struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	WebSocketURL string `json:"webSocketDebuggerUrl"`
}

// VersionInfo contains browser version information from /json/version.
type VersionInfo struct {
	Browser      string `json:"Browser"`
	ProtocolVer  string `json:"Protocol-Version"`
	UserAgent    string `json:"User-Agent"`
	V8Version    string `json:"V8-Version"`
	WebKitVersion string `json:"WebKit-Version"`
	WebSocketURL string `json:"webSocketDebuggerUrl"`
}

// FetchTargets retrieves the list of available targets from the CDP endpoint.
// Uses http.DefaultClient which has no timeout; callers must provide a context
// with timeout. This is acceptable for local CDP calls where network issues are rare.
func FetchTargets(ctx context.Context, host string, port int) ([]Target, error) {
	url := fmt.Sprintf("http://%s:%d/json", host, port)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch targets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var targets []Target
	if err := json.Unmarshal(body, &targets); err != nil {
		return nil, fmt.Errorf("parse targets: %w", err)
	}

	return targets, nil
}

// FetchVersion retrieves browser version info from the CDP endpoint.
// Uses http.DefaultClient which has no timeout; callers must provide a context
// with timeout. This is acceptable for local CDP calls where network issues are rare.
func FetchVersion(ctx context.Context, host string, port int) (*VersionInfo, error) {
	url := fmt.Sprintf("http://%s:%d/json/version", host, port)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var info VersionInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("parse version: %w", err)
	}

	return &info, nil
}

// FindPageTarget returns the first page-type target from the list.
func FindPageTarget(targets []Target) *Target {
	for i := range targets {
		if targets[i].Type == "page" {
			return &targets[i]
		}
	}
	return nil
}
