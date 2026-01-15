package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

func TestRunCookiesDefault_DaemonNotRunning(t *testing.T) {
	enableJSONOutput(t)

	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runCookiesDefault(cookiesCmd, nil)

	w.Close()
	os.Stderr = old

	if err == nil {
		t.Fatal("expected error when daemon not running")
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if !strings.Contains(err.Error(), "daemon not running") {
		t.Errorf("expected 'daemon not running' error, got: %v", err)
	}
}

func TestRunCookiesDefault_Success(t *testing.T) {
	enableJSONOutput(t)

	cookies := []ipc.Cookie{
		{
			Name:     "session_id",
			Value:    "abc123",
			Domain:   ".example.com",
			Path:     "/",
			Expires:  1735084800,
			HTTPOnly: true,
			Secure:   true,
			Session:  false,
			SameSite: "Lax",
		},
		{
			Name:     "remember_me",
			Value:    "yes",
			Domain:   "example.com",
			Path:     "/",
			HTTPOnly: false,
			Secure:   true,
			Session:  true,
		},
	}

	cookiesData := ipc.CookiesData{
		Cookies: cookies,
		Count:   len(cookies),
	}
	cookiesJSON, _ := json.Marshal(cookiesData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "cookies" {
				t.Errorf("expected cmd=cookies, got %s", req.Cmd)
			}
			return ipc.Response{OK: true, Data: cookiesJSON}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runCookiesDefault(cookiesCmd, nil)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if result["ok"] != true {
		t.Errorf("expected ok=true, got %v", result["ok"])
	}

	// Default mode now outputs to stdout, so we should have cookies and count (not path)
	if result["count"] != float64(2) {
		t.Errorf("expected count=2, got %v", result["count"])
	}

	resultCookies, ok := result["cookies"].([]any)
	if !ok {
		t.Fatalf("expected cookies to be array, got %T", result["cookies"])
	}

	if len(resultCookies) != 2 {
		t.Errorf("expected 2 cookies, got %d", len(resultCookies))
	}
}

func TestRunCookiesDefault_UnknownSubcommand(t *testing.T) {
	enableJSONOutput(t)

	restore := setMockFactory(&mockFactory{daemonRunning: true})
	defer restore()

	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runCookiesDefault(cookiesCmd, []string{"invalid"})

	w.Close()
	os.Stderr = old

	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}

	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected 'unknown command' error, got: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
}

func TestRunCookiesShow_Success(t *testing.T) {
	enableJSONOutput(t)

	cookies := []ipc.Cookie{
		{
			Name:     "session_id",
			Value:    "abc123",
			Domain:   ".example.com",
			Path:     "/",
			HTTPOnly: true,
			Secure:   true,
			Session:  true,
			SameSite: "Lax",
		},
	}

	cookiesData := ipc.CookiesData{
		Cookies: cookies,
		Count:   len(cookies),
	}
	cookiesJSON, _ := json.Marshal(cookiesData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: true, Data: cookiesJSON}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runCookiesDefault(cookiesCmd, nil)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if result["ok"] != true {
		t.Errorf("expected ok=true, got %v", result["ok"])
	}

	cookiesResult, ok := result["cookies"].([]any)
	if !ok {
		t.Fatalf("expected cookies to be array, got %T", result["cookies"])
	}

	if len(cookiesResult) != 1 {
		t.Errorf("expected 1 cookie, got %d", len(cookiesResult))
	}

	count, ok := result["count"].(float64)
	if !ok {
		t.Fatalf("expected count to be number, got %T", result["count"])
	}

	if int(count) != 1 {
		t.Errorf("expected count=1, got %v", count)
	}
}

func TestRunCookiesSave_ToFile(t *testing.T) {
	enableJSONOutput(t)

	cookies := []ipc.Cookie{
		{
			Name:    "test",
			Value:   "value",
			Domain:  "example.com",
			Path:    "/",
			Session: true,
		},
	}

	cookiesData := ipc.CookiesData{
		Cookies: cookies,
		Count:   len(cookies),
	}
	cookiesJSON, _ := json.Marshal(cookiesData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: true, Data: cookiesJSON}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	// Create temp file path
	tmpFile := filepath.Join(t.TempDir(), "cookies.json")

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runCookiesSave(cookiesSaveCmd, []string{tmpFile})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if result["ok"] != true {
		t.Errorf("expected ok=true, got %v", result["ok"])
	}

	path, ok := result["path"].(string)
	if !ok {
		t.Fatalf("expected path to be string, got %T", result["path"])
	}

	if path != tmpFile {
		t.Errorf("expected path=%s, got %s", tmpFile, path)
	}

	// Verify file was created
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Errorf("expected file to be created at %s", tmpFile)
	}

	// Verify file contents
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var fileData map[string]any
	if err := json.Unmarshal(data, &fileData); err != nil {
		t.Fatalf("failed to parse file contents: %v", err)
	}

	if fileData["ok"] != true {
		t.Errorf("expected file ok=true, got %v", fileData["ok"])
	}
}

func TestRunCookiesSave_ToDirectory(t *testing.T) {
	enableJSONOutput(t)

	cookies := []ipc.Cookie{
		{
			Name:    "test",
			Value:   "value",
			Domain:  "example.com",
			Path:    "/",
			Session: true,
		},
	}

	cookiesData := ipc.CookiesData{
		Cookies: cookies,
		Count:   len(cookies),
	}
	cookiesJSON, _ := json.Marshal(cookiesData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: true, Data: cookiesJSON}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	// Use temp directory
	tmpDir := t.TempDir()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Add trailing slash to indicate directory (new trailing slash convention)
	err := runCookiesSave(cookiesSaveCmd, []string{tmpDir + "/"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	path, ok := result["path"].(string)
	if !ok {
		t.Fatalf("expected path to be string, got %T", result["path"])
	}

	// Verify path starts with tmpDir
	if !strings.HasPrefix(path, tmpDir) {
		t.Errorf("expected path to start with %s, got %s", tmpDir, path)
	}

	// Verify filename pattern
	filename := filepath.Base(path)
	if !strings.HasSuffix(filename, "-cookies.json") {
		t.Errorf("expected filename to end with -cookies.json, got %s", filename)
	}

	// Verify file was created
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file to be created at %s", path)
	}
}

func TestFilterCookiesByDomain(t *testing.T) {
	cookies := []ipc.Cookie{
		{Name: "cookie1", Domain: ".example.com"},
		{Name: "cookie2", Domain: "www.example.com"},
		{Name: "cookie3", Domain: ".github.com"},
		{Name: "cookie4", Domain: "api.example.com"},
	}

	tests := []struct {
		name     string
		domain   string
		expected int
	}{
		{"exact match", ".example.com", 1},
		{"suffix match", "example.com", 3},
		{"no match", ".other.com", 0},
		{"github", ".github.com", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterCookiesByDomain(cookies, tt.domain)
			if len(filtered) != tt.expected {
				t.Errorf("expected %d cookies, got %d", tt.expected, len(filtered))
			}
		})
	}
}

func TestFilterCookiesByName(t *testing.T) {
	cookies := []ipc.Cookie{
		{Name: "session_id", Domain: ".example.com"},
		{Name: "session_id", Domain: "api.example.com"},
		{Name: "remember_me", Domain: ".example.com"},
		{Name: "tracking", Domain: ".example.com"},
	}

	tests := []struct {
		name     string
		filter   string
		expected int
	}{
		{"exact match single", "remember_me", 1},
		{"exact match multiple", "session_id", 2},
		{"no match", "nonexistent", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterCookiesByName(cookies, tt.filter)
			if len(filtered) != tt.expected {
				t.Errorf("expected %d cookies, got %d", tt.expected, len(filtered))
			}
		})
	}
}

func TestFilterCookiesByText(t *testing.T) {
	cookies := []ipc.Cookie{
		{Name: "session_id", Value: "abc123"},
		{Name: "remember_me", Value: "yes"},
		{Name: "tracking", Value: "SESSION_xyz"},
		{Name: "auth_token", Value: "bearer_token"},
	}

	tests := []struct {
		name     string
		search   string
		expected int
	}{
		{"search in name", "session", 2},   // matches session_id and tracking (SESSION_xyz)
		{"search in value", "token", 1},    // matches auth_token
		{"search both", "yes", 1},          // matches remember_me
		{"case insensitive", "SESSION", 2}, // matches session_id and tracking
		{"no match", "nonexistent", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterCookiesByText(cookies, tt.search)
			if len(filtered) != tt.expected {
				t.Errorf("expected %d cookies, got %d", tt.expected, len(filtered))
			}
		})
	}
}

func TestGetCookiesFromDaemon_WithFilters(t *testing.T) {
	cookies := []ipc.Cookie{
		{Name: "session_id", Value: "abc123", Domain: ".example.com"},
		{Name: "session_id", Value: "xyz789", Domain: ".github.com"},
		{Name: "remember_me", Value: "yes", Domain: ".example.com"},
		{Name: "tracking", Value: "id123", Domain: ".example.com"},
	}

	cookiesData := ipc.CookiesData{
		Cookies: cookies,
		Count:   len(cookies),
	}
	cookiesJSON, _ := json.Marshal(cookiesData)

	tests := []struct {
		name     string
		flags    map[string]string
		expected int
	}{
		{"no filters", map[string]string{}, 4},
		{"domain filter", map[string]string{"domain": ".example.com"}, 3},
		{"name filter", map[string]string{"name": "session_id"}, 2},
		{"find filter", map[string]string{"find": "session"}, 2},
		{"combined filters", map[string]string{"domain": ".example.com", "name": "session_id"}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &mockExecutor{
				executeFunc: func(req ipc.Request) (ipc.Response, error) {
					return ipc.Response{OK: true, Data: cookiesJSON}, nil
				},
			}

			restore := setMockFactory(&mockFactory{
				daemonRunning: true,
				executor:      exec,
			})
			defer restore()

			// Set flags on command
			cmd := &cobra.Command{}
			cmd.Flags().String("domain", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().String("find", "", "")

			for key, value := range tt.flags {
				_ = cmd.Flags().Set(key, value)
			}

			result, err := getCookiesFromDaemon(cmd)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != tt.expected {
				t.Errorf("expected %d cookies, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestGetCookiesFromDaemon_NoMatches(t *testing.T) {
	cookies := []ipc.Cookie{
		{Name: "session_id", Value: "abc123", Domain: ".example.com"},
	}

	cookiesData := ipc.CookiesData{
		Cookies: cookies,
		Count:   len(cookies),
	}
	cookiesJSON, _ := json.Marshal(cookiesData)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: true, Data: cookiesJSON}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	cmd := &cobra.Command{}
	cmd.Flags().String("find", "", "")
	_ = cmd.Flags().Set("find", "nonexistent")

	_, err := getCookiesFromDaemon(cmd)
	if err == nil {
		t.Fatal("expected error when no matches found")
	}

	if !strings.Contains(err.Error(), "no matches found") {
		t.Errorf("expected 'no matches found' error, got: %v", err)
	}
}

func TestRunCookiesSet_Success(t *testing.T) {
	enableJSONOutput(t)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "cookies" {
				t.Errorf("expected cmd=cookies, got %s", req.Cmd)
			}

			var params ipc.CookiesParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				t.Fatalf("failed to unmarshal params: %v", err)
			}

			if params.Action != "set" {
				t.Errorf("expected action=set, got %s", params.Action)
			}

			if params.Name != "session" {
				t.Errorf("expected name=session, got %s", params.Name)
			}

			if params.Value != "abc123" {
				t.Errorf("expected value=abc123, got %s", params.Value)
			}

			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runCookiesSet(cookiesSetCmd, []string{"session", "abc123"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if result["ok"] != true {
		t.Errorf("expected ok=true, got %v", result["ok"])
	}
}

func TestRunCookiesDelete_BasicSuccess(t *testing.T) {
	enableJSONOutput(t)

	exec := &mockExecutor{
		executeFunc: func(req ipc.Request) (ipc.Response, error) {
			if req.Cmd != "cookies" {
				t.Errorf("expected cmd=cookies, got %s", req.Cmd)
			}

			var params ipc.CookiesParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				t.Fatalf("failed to unmarshal params: %v", err)
			}

			if params.Action != "delete" {
				t.Errorf("expected action=delete, got %s", params.Action)
			}

			if params.Name != "session" {
				t.Errorf("expected name=session, got %s", params.Name)
			}

			return ipc.Response{OK: true}, nil
		},
	}

	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor:      exec,
	})
	defer restore()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runCookiesDelete(cookiesDeleteCmd, []string{"session"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if result["ok"] != true {
		t.Errorf("expected ok=true, got %v", result["ok"])
	}
}

func TestWriteCookiesToFile(t *testing.T) {
	cookies := []ipc.Cookie{
		{
			Name:    "test",
			Value:   "value",
			Domain:  "example.com",
			Path:    "/",
			Session: true,
		},
	}

	tmpFile := filepath.Join(t.TempDir(), "test-cookies.json")

	err := writeCookiesToFile(tmpFile, cookies)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Errorf("expected file to be created at %s", tmpFile)
	}

	// Verify file contents
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var fileData map[string]any
	if err := json.Unmarshal(data, &fileData); err != nil {
		t.Fatalf("failed to parse file contents: %v", err)
	}

	if fileData["ok"] != true {
		t.Errorf("expected ok=true, got %v", fileData["ok"])
	}

	cookiesResult, ok := fileData["cookies"].([]any)
	if !ok {
		t.Fatalf("expected cookies to be array, got %T", fileData["cookies"])
	}

	if len(cookiesResult) != 1 {
		t.Errorf("expected 1 cookie, got %d", len(cookiesResult))
	}

	count, ok := fileData["count"].(float64)
	if !ok {
		t.Fatalf("expected count to be number, got %T", fileData["count"])
	}

	if int(count) != 1 {
		t.Errorf("expected count=1, got %v", count)
	}
}

func TestGenerateCookiesFilename(t *testing.T) {
	filename := generateCookiesFilename()

	if !strings.HasSuffix(filename, "-cookies.json") {
		t.Errorf("expected filename to end with -cookies.json, got %s", filename)
	}

	// Verify timestamp format (YY-MM-DD-HHMMSS)
	parts := strings.Split(filename, "-")
	if len(parts) != 5 {
		t.Errorf("expected 5 parts in filename, got %d: %s", len(parts), filename)
	}
}

func TestGenerateCookiesPath(t *testing.T) {
	path, err := generateCookiesPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(path, "/tmp/webctl-cookies/") {
		t.Errorf("expected path to start with /tmp/webctl-cookies/, got %s", path)
	}

	if !strings.HasSuffix(path, "-cookies.json") {
		t.Errorf("expected path to end with -cookies.json, got %s", path)
	}
}
