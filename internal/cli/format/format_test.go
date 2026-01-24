package format

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/grantcarthew/webctl/internal/ipc"
)

func init() {
	// Disable colors in tests for consistent output
	color.NoColor = true
}

func TestNewOutputOptions(t *testing.T) {
	tests := []struct {
		name             string
		jsonOutput       bool
		noColorFlag      bool
		noColorEnv       string
		expectedUseColor bool
	}{
		{
			name:             "JSON output disables color",
			jsonOutput:       true,
			noColorFlag:      false,
			expectedUseColor: false,
		},
		{
			name:             "no-color flag disables color",
			jsonOutput:       false,
			noColorFlag:      true,
			expectedUseColor: false,
		},
		{
			name:             "NO_COLOR env disables color",
			jsonOutput:       false,
			noColorFlag:      false,
			noColorEnv:       "1",
			expectedUseColor: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set NO_COLOR environment variable
			if tt.noColorEnv != "" {
				old := os.Getenv("NO_COLOR")
				os.Setenv("NO_COLOR", tt.noColorEnv)
				defer os.Setenv("NO_COLOR", old)
			}

			opts := NewOutputOptions(tt.jsonOutput, tt.noColorFlag)
			if opts.UseColor != tt.expectedUseColor {
				t.Errorf("UseColor = %v, want %v", opts.UseColor, tt.expectedUseColor)
			}
		})
	}
}

func TestActionSuccess(t *testing.T) {
	var buf bytes.Buffer
	err := ActionSuccess(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	expected := "OK\n"
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestActionError(t *testing.T) {
	var buf bytes.Buffer
	opts := OutputOptions{UseColor: false}
	err := ActionError(&buf, "test error", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	expected := "Error: test error\n"
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestStatus(t *testing.T) {
	tests := []struct {
		name     string
		data     ipc.StatusData
		expected string
	}{
		{
			name:     "not running",
			data:     ipc.StatusData{Running: false},
			expected: "Not running (start with: webctl start)\n",
		},
		{
			name:     "running with PID but no browser",
			data:     ipc.StatusData{Running: true, PID: 1234, Sessions: []ipc.PageSession{}},
			expected: "No browser\npid: 1234\n",
		},
		{
			name: "running with active session",
			data: ipc.StatusData{
				Running: true,
				PID:     1234,
				ActiveSession: &ipc.PageSession{
					ID:  "session1",
					URL: "https://example.com",
				},
				Sessions: []ipc.PageSession{
					{ID: "session1", URL: "https://example.com", Active: true},
				},
			},
			expected: "OK\npid: 1234\nsessions:\n  * https://example.com\n",
		},
	}

	opts := OutputOptions{UseColor: false}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := Status(&buf, tt.data, opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := buf.String()
			if got != tt.expected {
				t.Errorf("got:\n%q\nwant:\n%q", got, tt.expected)
			}
		})
	}
}

func TestConsole(t *testing.T) {
	entries := []ipc.ConsoleEntry{
		{Type: "log", Text: "test message", Timestamp: 1609459200000, URL: "http://example.com", Line: 42},
		{Type: "error", Text: "error message", Timestamp: 1609459200000},
	}

	var buf bytes.Buffer
	opts := OutputOptions{UseColor: false}
	err := Console(&buf, entries, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Check that output contains expected elements
	if !strings.Contains(output, "LOG test message") {
		t.Error("output should contain log message")
	}
	if !strings.Contains(output, "ERROR error message") {
		t.Error("output should contain error message")
	}
	if !strings.Contains(output, "http://example.com:42") {
		t.Error("output should contain URL and line number")
	}
}

func TestNetwork(t *testing.T) {
	entries := []ipc.NetworkEntry{
		{Method: "GET", URL: "https://api.example.com", Status: 200, Duration: 0.123},
		{Method: "POST", URL: "https://api.example.com", Status: 404, Duration: 0.456, Body: `{"key":"value"}`},
	}

	var buf bytes.Buffer
	opts := OutputOptions{UseColor: false}
	err := Network(&buf, entries, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "GET https://api.example.com 200 123ms") {
		t.Error("output should contain GET request")
	}
	if !strings.Contains(output, "POST https://api.example.com 404 456ms") {
		t.Error("output should contain POST request")
	}
	if !strings.Contains(output, `{"key":"value"}`) {
		t.Error("output should contain request body")
	}
}

func TestCookies(t *testing.T) {
	cookies := []ipc.Cookie{
		{Name: "session", Value: "abc123", Domain: ".example.com", Path: "/", Secure: true, HTTPOnly: true},
		{Name: "simple", Value: "value"},
	}

	var buf bytes.Buffer
	opts := OutputOptions{UseColor: false}
	err := Cookies(&buf, cookies, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "session=abc123") {
		t.Error("output should contain session cookie")
	}
	if !strings.Contains(output, "domain=.example.com") {
		t.Error("output should contain domain")
	}
	if !strings.Contains(output, "secure") {
		t.Error("output should contain secure flag")
	}
	if !strings.Contains(output, "httponly") {
		t.Error("output should contain httponly flag")
	}
}

func TestFilePath(t *testing.T) {
	var buf bytes.Buffer
	err := FilePath(&buf, "/tmp/test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	expected := "/tmp/test.txt\n"
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestEvalResult(t *testing.T) {
	tests := []struct {
		name     string
		data     ipc.EvalData
		expected string
	}{
		{
			name:     "undefined",
			data:     ipc.EvalData{HasValue: false},
			expected: "undefined\n",
		},
		{
			name:     "null",
			data:     ipc.EvalData{HasValue: true, Value: nil},
			expected: "null\n",
		},
		{
			name:     "string",
			data:     ipc.EvalData{HasValue: true, Value: "hello"},
			expected: "hello\n",
		},
		{
			name:     "number",
			data:     ipc.EvalData{HasValue: true, Value: float64(42)},
			expected: "42\n",
		},
		{
			name:     "boolean",
			data:     ipc.EvalData{HasValue: true, Value: true},
			expected: "true\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := EvalResult(&buf, tt.data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := buf.String()
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestTarget(t *testing.T) {
	data := ipc.TargetData{
		ActiveSession: "session1",
		Sessions: []ipc.PageSession{
			{ID: "session1", URL: "https://example.com", Title: "Example"},
			{ID: "session2", URL: "https://other.com", Title: "Other"},
		},
	}

	var buf bytes.Buffer
	opts := OutputOptions{UseColor: false}
	err := Target(&buf, data, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "* https://example.com") {
		t.Error("output should mark active session with *")
	}
	if !strings.Contains(output, "  https://other.com") {
		t.Error("output should show inactive session with spaces")
	}
}


func TestComputedStyles(t *testing.T) {
	styles := map[string]string{
		"color":      "red",
		"background": "blue",
	}

	var buf bytes.Buffer
	err := ComputedStyles(&buf, styles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "color: red") {
		t.Error("output should contain color property")
	}
	if !strings.Contains(output, "background: blue") {
		t.Error("output should contain background property")
	}
}

func TestPropertyValue(t *testing.T) {
	var buf bytes.Buffer
	err := PropertyValue(&buf, "red")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	expected := "red\n"
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestInlineStyles(t *testing.T) {
	tests := []struct {
		name     string
		styles   []string
		expected string
	}{
		{
			name:     "single style",
			styles:   []string{"color: red; font-size: 16px;"},
			expected: "color: red; font-size: 16px;\n",
		},
		{
			name:     "multiple styles",
			styles:   []string{"color: red;", "background: blue;", "margin: 10px;"},
			expected: "color: red;\n--\nbackground: blue;\n--\nmargin: 10px;\n",
		},
		{
			name:     "empty style",
			styles:   []string{""},
			expected: "(empty)\n",
		},
		{
			name:     "mixed empty and non-empty",
			styles:   []string{"color: red;", "", "margin: 10px;"},
			expected: "color: red;\n--\n(empty)\n--\nmargin: 10px;\n",
		},
		{
			name:     "no styles",
			styles:   []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := InlineStyles(&buf, tt.styles)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := buf.String()
			if got != tt.expected {
				t.Errorf("got:\n%q\nwant:\n%q", got, tt.expected)
			}
		})
	}
}

func TestComputedStylesMulti(t *testing.T) {
	tests := []struct {
		name       string
		stylesList []map[string]string
		wantSep    bool
	}{
		{
			name:       "empty list",
			stylesList: []map[string]string{},
			wantSep:    false,
		},
		{
			name: "single element",
			stylesList: []map[string]string{
				{"color": "red"},
			},
			wantSep: false,
		},
		{
			name: "multiple elements",
			stylesList: []map[string]string{
				{"color": "red"},
				{"color": "blue"},
			},
			wantSep: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := ComputedStylesMulti(&buf, tt.stylesList)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := buf.String()
			hasSep := strings.Contains(got, "--")
			if hasSep != tt.wantSep {
				t.Errorf("separator present = %v, want %v, output: %q", hasSep, tt.wantSep, got)
			}
		})
	}
}

func TestMatchedRules(t *testing.T) {
	tests := []struct {
		name     string
		rules    []ipc.CSSMatchedRule
		expected string
	}{
		{
			name:     "empty rules",
			rules:    []ipc.CSSMatchedRule{},
			expected: "",
		},
		{
			name: "single rule",
			rules: []ipc.CSSMatchedRule{
				{
					Selector:   ".header",
					Properties: map[string]string{"color": "red"},
				},
			},
			expected: "", // Check contains instead
		},
		{
			name: "multiple rules",
			rules: []ipc.CSSMatchedRule{
				{
					Selector:   "(inline)",
					Properties: map[string]string{"color": "red"},
					Source:     "inline",
				},
				{
					Selector:   ".header",
					Properties: map[string]string{"background": "blue"},
				},
			},
			expected: "", // Check contains instead
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := MatchedRules(&buf, tt.rules)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := buf.String()
			if tt.expected != "" && got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}

	// Test specific behavior
	t.Run("contains selector comment", func(t *testing.T) {
		rules := []ipc.CSSMatchedRule{
			{Selector: ".header", Properties: map[string]string{"color": "red"}},
		}
		var buf bytes.Buffer
		_ = MatchedRules(&buf, rules)
		output := buf.String()
		if !strings.Contains(output, "/* .header */") {
			t.Errorf("output should contain selector as comment, got: %s", output)
		}
	})

	t.Run("contains separator between rules", func(t *testing.T) {
		rules := []ipc.CSSMatchedRule{
			{Selector: ".a", Properties: map[string]string{"color": "red"}},
			{Selector: ".b", Properties: map[string]string{"color": "blue"}},
		}
		var buf bytes.Buffer
		_ = MatchedRules(&buf, rules)
		output := buf.String()
		if !strings.Contains(output, "--") {
			t.Errorf("output should contain separator, got: %s", output)
		}
	})
}
