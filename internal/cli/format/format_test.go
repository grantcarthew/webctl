package format

import (
	"bytes"
	"encoding/json"
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
				_ = os.Setenv("NO_COLOR", tt.noColorEnv)
				defer func() { _ = os.Setenv("NO_COLOR", old) }()
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
		{Seq: 1, Type: "log", Text: "test message", Timestamp: 1609459200000, URL: "http://example.com", Line: 42, Column: 7},
		{Seq: 2, Type: "error", Text: "error message", Timestamp: 1609459200000},
	}

	var buf bytes.Buffer
	opts := OutputOptions{UseColor: false}
	err := Console(&buf, entries, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Each entry is one seq-prefixed summary line: SEQ [HH:MM:SS] LEVEL frame msg.
	if !strings.Contains(output, "01 [") || !strings.Contains(output, "LOG http://example.com:42:7 test message") {
		t.Errorf("output should contain the indexed log summary line:\n%s", output)
	}
	if !strings.Contains(output, "02 [") || !strings.Contains(output, "ERROR error message") {
		t.Errorf("output should contain the indexed error summary line:\n%s", output)
	}
}

func TestConsole_ColumnZeroRenders(t *testing.T) {
	// CDP columns are 0-based; column 0 is the first column and must appear on
	// the locator rather than being treated as "absent".
	entries := []ipc.ConsoleEntry{
		{
			Seq: 1, Type: "error", Text: "boom", Timestamp: 1609459200000,
			Stack: []ipc.ConsoleFrame{
				{Function: "onClick", URL: "app.js", Line: 30, Column: 0},
			},
		},
	}

	var buf bytes.Buffer
	if err := Console(&buf, entries, OutputOptions{UseColor: false}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "onClick app.js:30:0") {
		t.Errorf("column 0 must render on the locator:\n%s", output)
	}
}

func TestConsole_LineZeroRenders(t *testing.T) {
	// CDP lines are 0-based; line 0 is the first line (and the only line on a
	// minified one-liner). Dropping it would hide the column that locates the
	// call site on bundled scripts.
	entries := []ipc.ConsoleEntry{
		{
			Seq: 1, Type: "error", Text: "boom", Timestamp: 1609459200000,
			Stack: []ipc.ConsoleFrame{
				{Function: "bundle", URL: "app.min.js", Line: 0, Column: 1234},
			},
		},
	}

	var buf bytes.Buffer
	if err := Console(&buf, entries, OutputOptions{UseColor: false}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "bundle app.min.js:0:1234") {
		t.Errorf("line 0 must render on the locator with its column:\n%s", output)
	}
}

func TestConsole_SeqPrefixZeroPadded(t *testing.T) {
	// The seq prefix is zero-padded to a minimum of two digits and grows naturally
	// beyond, with no surrounding brackets, so input and output match.
	entries := []ipc.ConsoleEntry{
		{Seq: 9, Type: "log", Text: "a", Timestamp: 1609459200000},
		{Seq: 100, Type: "log", Text: "b", Timestamp: 1609459200000},
	}

	var buf bytes.Buffer
	if err := Console(&buf, entries, OutputOptions{UseColor: false}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "09 [") {
		t.Errorf("single-digit seq should zero-pad to two digits:\n%s", output)
	}
	if !strings.Contains(output, "100 [") {
		t.Errorf("a three-digit seq should render its full width:\n%s", output)
	}
}

func TestConsole_MultiLineTextCondensedToFirstLine(t *testing.T) {
	// A multi-line message contributes only its first line to the index, so each
	// entry stays exactly one physical line.
	entries := []ipc.ConsoleEntry{
		{Seq: 1, Type: "error", Text: "TypeError: boom\n    at foo (app.js:42:10)\n    at bar (app.js:9:3)", Timestamp: 1609459200000},
	}

	var buf bytes.Buffer
	if err := Console(&buf, entries, OutputOptions{UseColor: false}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if strings.Count(output, "\n") != 1 {
		t.Errorf("a multi-line message must render as one physical line:\n%s", output)
	}
	if !strings.Contains(output, "TypeError: boom") || strings.Contains(output, "at foo") {
		t.Errorf("only the first line of Text belongs on the summary line:\n%s", output)
	}
}

func TestConsoleDetail_RendersFullEntry(t *testing.T) {
	// Drill-down renders the summary line plus the full stack, arguments,
	// exception, and Log-domain correlation on seven-space subordinate lines.
	entry := ipc.ConsoleEntry{
		Seq: 7, Type: "error", Text: "TypeError: boom\n    at foo",
		Timestamp: 1609459200000,
		Stack: []ipc.ConsoleFrame{
			{Function: "foo", URL: "app.js", Line: 42, Column: 10},
			{Function: "bar", URL: "app.js", Line: 9, Column: 3},
		},
		Args: []ipc.ConsoleArg{
			{Type: "string", Value: json.RawMessage(`"hello"`)},
			{Type: "object", Subtype: "array", Description: "Array(2)", Preview: []ipc.ConsolePreviewProp{{Name: "0", Value: "1"}, {Name: "1", Value: "2"}}},
		},
		ExceptionClass: "TypeError", ExceptionSubtype: "error",
	}

	var buf bytes.Buffer
	if err := ConsoleDetail(&buf, entry, OutputOptions{UseColor: false}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	checks := []string{
		"07 [",
		"       message:\n         TypeError: boom\n",
		"       stack:\n         foo app.js:42:10\n         bar app.js:9:3\n",
		"       args:\n         [0] string \"hello\"\n         [1] object/array Array(2) { 0: 1, 1: 2 }\n",
		"       exception: TypeError (error)\n",
	}
	for _, want := range checks {
		if !strings.Contains(output, want) {
			t.Errorf("drill-down output missing %q:\n%s", want, output)
		}
	}
}

func TestConsoleDetail_TrailingNewlineOnlySkipsMessageBlock(t *testing.T) {
	// A sole trailing newline is not multi-line content; the message block would
	// only restate the summary line and print an empty indented line.
	entry := ipc.ConsoleEntry{
		Seq: 1, Type: "error", Text: "TypeError: boom\n", Timestamp: 1609459200000,
	}

	var buf bytes.Buffer
	if err := ConsoleDetail(&buf, entry, OutputOptions{UseColor: false}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "message:") {
		t.Errorf("trailing-only newline must not open a message block:\n%s", output)
	}
	if !strings.Contains(output, "TypeError: boom") {
		t.Errorf("summary line should still carry the first line of Text:\n%s", output)
	}
}

func TestConsoleDetail_MultiLineTrailingNewlineNoBlankRow(t *testing.T) {
	// Exception descriptions often end with a trailing newline. After content
	// lines that is not multi-line substance — strip it so the message block
	// does not close with an empty nine-space row.
	entry := ipc.ConsoleEntry{
		Seq: 1, Type: "error", Text: "TypeError: boom\n    at foo\n", Timestamp: 1609459200000,
	}

	var buf bytes.Buffer
	if err := ConsoleDetail(&buf, entry, OutputOptions{UseColor: false}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	want := "       message:\n         TypeError: boom\n             at foo\n"
	if !strings.Contains(output, want) {
		t.Errorf("message block missing expected lines:\n%s", output)
	}
	// No blank subordinate line between the last message line and whatever
	// follows (or end of output). The message block must end at "at foo".
	if strings.Contains(output, "at foo\n         \n") || strings.Contains(output, "at foo\n"+netIndent2+"\n") {
		t.Errorf("trailing newline must not invent an empty message row:\n%s", output)
	}
}

func TestConsoleDetail_LogDomainCorrelation(t *testing.T) {
	// A Log-domain entry surfaces its source, network request id, and worker id
	// on drill-down so agents can correlate across buffers and workers.
	entry := ipc.ConsoleEntry{
		Seq: 3, Type: "error", Text: "Failed to load resource", Timestamp: 1609459200000,
		Source: "network", NetworkRequestID: "1234.5", WorkerID: "worker.1",
	}

	var buf bytes.Buffer
	if err := ConsoleDetail(&buf, entry, OutputOptions{UseColor: false}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "       source: network\n") {
		t.Errorf("Log-domain source should render:\n%s", output)
	}
	if !strings.Contains(output, "       network-request: 1234.5\n") {
		t.Errorf("network request id should render as the correlating identity:\n%s", output)
	}
	if !strings.Contains(output, "       worker: worker.1\n") {
		t.Errorf("worker id should render when present:\n%s", output)
	}
}

func TestConsoleDetail_AsyncStackBoundary(t *testing.T) {
	// An asynchronous continuation boundary prints its parent group description
	// on its own line so the sync/async split stays legible in the stack block.
	entry := ipc.ConsoleEntry{
		Seq: 4, Type: "error", Text: "boom", Timestamp: 1609459200000,
		Stack: []ipc.ConsoleFrame{
			{Function: "handler", URL: "app.js", Line: 10, Column: 2},
			{Function: "then", URL: "app.js", Line: 20, Column: 4, Async: "Promise.then"},
		},
	}

	var buf bytes.Buffer
	if err := ConsoleDetail(&buf, entry, OutputOptions{UseColor: false}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	want := "       stack:\n         handler app.js:10:2\n         async Promise.then\n         then app.js:20:4\n"
	if !strings.Contains(output, want) {
		t.Errorf("async boundary missing or mis-ordered in stack block:\n%s", output)
	}
}

// netStd and netFull build the standard and full detail options the network
// text tests exercise, keeping color off for stable string assertions.
func netStd() OutputOptions  { return OutputOptions{UseColor: false, Detail: DetailStandard} }
func netFull() OutputOptions { return OutputOptions{UseColor: false, Detail: DetailFull} }

func TestNetwork(t *testing.T) {
	entries := []ipc.NetworkEntry{
		{Seq: 1, Method: "GET", URL: "https://api.example.com", Status: 200, Duration: 0.123},
		{
			Seq:          2,
			Method:       "POST",
			URL:          "https://api.example.com",
			Status:       404,
			Duration:     0.456,
			RequestBody:  `{"user":"grant"}`,
			ResponseBody: `{"key":"value"}`,
		},
	}

	var buf bytes.Buffer
	err := Network(&buf, entries, netFull())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "01 GET https://api.example.com 200 123ms") {
		t.Errorf("output should contain the seq-prefixed GET request:\n%s", output)
	}
	if !strings.Contains(output, "02 POST https://api.example.com 404 456ms") {
		t.Errorf("output should contain the seq-prefixed POST request:\n%s", output)
	}
	if !strings.Contains(output, `       request: {"user":"grant"}`) {
		t.Errorf("output should label the request body at seven-space indent:\n%s", output)
	}
	if !strings.Contains(output, `       response: {"key":"value"}`) {
		t.Errorf("output should label the response body at seven-space indent:\n%s", output)
	}
}

func TestNetwork_SeqPrefixZeroPadded(t *testing.T) {
	// The seq prefix is zero-padded to a minimum of two digits and grows naturally
	// beyond, with no surrounding brackets, so input and output match.
	entries := []ipc.NetworkEntry{
		{Seq: 9, Method: "GET", URL: "https://example.com/a", Status: 200},
		{Seq: 100, Method: "GET", URL: "https://example.com/b", Status: 200},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, OutputOptions{UseColor: false, Detail: DetailSummary}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "09 GET https://example.com/a") {
		t.Errorf("single-digit seq should zero-pad to two digits:\n%s", output)
	}
	if !strings.Contains(output, "100 GET https://example.com/b") {
		t.Errorf("a three-digit seq should render its full width:\n%s", output)
	}
	if strings.Contains(output, "[") || strings.Contains(output, "]") {
		t.Errorf("the seq prefix must carry no brackets:\n%s", output)
	}
}

func TestNetwork_DetailLevels(t *testing.T) {
	entry := ipc.NetworkEntry{
		Seq: 5, Method: "GET", URL: "https://example.com/", Status: 200, Duration: 0.045,
		RemoteIPAddress: "93.184.216.34", RemotePort: 443, Protocol: "h2",
		RequestBody: `{"q":"x"}`, ResponseBody: `{"ok":true}`,
	}

	render := func(d DetailLevel) string {
		var buf bytes.Buffer
		if err := Network(&buf, []ipc.NetworkEntry{entry}, OutputOptions{UseColor: false, Detail: d}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return buf.String()
	}

	// Summary: main line only.
	summary := render(DetailSummary)
	if !strings.Contains(summary, "05 GET https://example.com/ 200 45ms") {
		t.Errorf("summary should show the main line:\n%s", summary)
	}
	if strings.Contains(summary, "remote:") || strings.Contains(summary, "response:") {
		t.Errorf("summary must show no transport block and no bodies:\n%s", summary)
	}

	// Standard: main line plus transport block, no bodies.
	standard := render(DetailStandard)
	if !strings.Contains(standard, "       remote: 93.184.216.34:443 h2") {
		t.Errorf("standard should show the transport block:\n%s", standard)
	}
	if strings.Contains(standard, "response:") || strings.Contains(standard, "request:") {
		t.Errorf("standard must not show bodies:\n%s", standard)
	}

	// Full: transport block plus bodies.
	full := render(DetailFull)
	if !strings.Contains(full, "       remote: 93.184.216.34:443 h2") {
		t.Errorf("full should show the transport block:\n%s", full)
	}
	if !strings.Contains(full, `       request: {"q":"x"}`) || !strings.Contains(full, `       response: {"ok":true}`) {
		t.Errorf("full should show request and response bodies:\n%s", full)
	}
}

func TestNetwork_HeadersShownWhenEnabled(t *testing.T) {
	// With ShowHeaders set, request and response headers render as indented
	// subordinate lines with keys sorted for stable output.
	entries := []ipc.NetworkEntry{
		{
			Method: "GET", URL: "https://api.example.com/users", Status: 200, Duration: 0.045,
			RequestHeaders:  map[string]string{"accept": "application/json", "authorization": "Bearer x"},
			ResponseHeaders: map[string]string{"content-type": "application/json", "cache-control": "no-store"},
		},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, OutputOptions{UseColor: false, ShowHeaders: true, Detail: DetailStandard}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "       request-headers:\n         accept: application/json\n         authorization: Bearer x\n") {
		t.Errorf("request headers should render sorted under a label:\n%s", output)
	}
	if !strings.Contains(output, "       response-headers:\n         cache-control: no-store\n         content-type: application/json\n") {
		t.Errorf("response headers should render sorted under a label:\n%s", output)
	}
}

func TestNetwork_HeadersHiddenByDefault(t *testing.T) {
	// Without ShowHeaders, headers stay out of the default text view.
	entries := []ipc.NetworkEntry{
		{
			Method: "GET", URL: "https://api.example.com/users", Status: 200, Duration: 0.045,
			RequestHeaders:  map[string]string{"authorization": "Bearer x"},
			ResponseHeaders: map[string]string{"content-type": "application/json"},
		},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, netStd()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "headers:") || strings.Contains(output, "authorization") {
		t.Errorf("headers must not appear without --headers:\n%s", output)
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0B"},
		{512, "512B"},
		{1024, "1.0KB"},
		{3486, "3.4KB"},
		{1048576, "1.0MB"},
		{1073741824, "1.0GB"},
	}
	for _, c := range cases {
		if got := formatBytes(c.in); got != c.want {
			t.Errorf("formatBytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNetwork_SizeShownHumanReadable(t *testing.T) {
	// Response size appends to the main line in human-readable form; an entry with
	// no captured size (Size == 0) omits the token.
	entries := []ipc.NetworkEntry{
		{Method: "GET", URL: "https://api.example.com/data", Status: 200, Duration: 0.045, Type: "xhr", Size: 3486},
		{Method: "GET", URL: "https://api.example.com/empty", Status: 204, Duration: 0.010},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, OutputOptions{UseColor: false}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "200 45ms xhr 3.4KB") {
		t.Errorf("response size should append to the main line, human-readable:\n%s", output)
	}
	if !strings.Contains(output, "https://api.example.com/empty 204 10ms\n") {
		t.Errorf("an entry with no size should omit the size token:\n%s", output)
	}
}

func TestNetwork_RemoteLineShown(t *testing.T) {
	// The contacted endpoint and negotiated protocol render on a subordinate
	// remote: line when captured; an entry with neither stays quiet.
	entries := []ipc.NetworkEntry{
		{Method: "GET", URL: "https://example.com/", Status: 200, Duration: 0.045,
			Type: "document", RemoteIPAddress: "93.184.216.34", RemotePort: 443, Protocol: "h2", ConnectionID: 1186},
		{Method: "GET", URL: "https://example.com/bare", Status: 200, Duration: 0.010},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, netStd()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "       remote: 93.184.216.34:443 h2 conn:1186\n") {
		t.Errorf("remote line should show endpoint, protocol, and connection id:\n%s", output)
	}
	if strings.Contains(output, "/bare 200 10ms\n       remote:") {
		t.Errorf("an entry without transport data should omit the remote line:\n%s", output)
	}
}

func TestNetwork_SecurityStateShownOnlyWhenNotSecure(t *testing.T) {
	// A non-secure posture surfaces on the remote: line; the common "secure"
	// state stays silent so it does not clutter every HTTPS row.
	entries := []ipc.NetworkEntry{
		{Method: "GET", URL: "http://img.example.com/x.png", Status: 200,
			RemoteIPAddress: "203.0.113.9", RemotePort: 80, Protocol: "http/1.1", SecurityState: "insecure"},
		{Method: "GET", URL: "https://example.com/", Status: 200,
			RemoteIPAddress: "93.184.216.34", RemotePort: 443, Protocol: "h2", SecurityState: "secure"},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, netStd()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "203.0.113.9:80 http/1.1 insecure\n") {
		t.Errorf("non-secure state should show on the remote line:\n%s", output)
	}
	if strings.Contains(output, "secure\n") && strings.Contains(output, "93.184.216.34:443 h2 secure") {
		t.Errorf("the secure state should be omitted from the remote line:\n%s", output)
	}
}

func TestNetwork_RemoteLineOmitsPortWhenAbsent(t *testing.T) {
	// A captured address with no port (cached/local responses) renders the IP
	// alone rather than a dangling ":0".
	entries := []ipc.NetworkEntry{
		{Method: "GET", URL: "https://example.com/", Status: 200, RemoteIPAddress: "93.184.216.34", Protocol: "h2"},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, netStd()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out := buf.String(); !strings.Contains(out, "       remote: 93.184.216.34 h2\n") {
		t.Errorf("remote line should omit a zero port:\n%s", out)
	}
}

func TestNetwork_CacheOriginToken(t *testing.T) {
	// Each cache origin renders as a single self-describing main-line token; an
	// uncached response carries none.
	cases := []struct {
		name  string
		entry ipc.NetworkEntry
		token string
	}{
		{"disk", ipc.NetworkEntry{Method: "GET", URL: "https://example.com/a", Status: 200, FromDiskCache: true}, "(disk)"},
		{"service worker", ipc.NetworkEntry{Method: "GET", URL: "https://example.com/b", Status: 200, FromServiceWorker: true}, "(service-worker)"},
		{"prefetch", ipc.NetworkEntry{Method: "GET", URL: "https://example.com/c", Status: 200, FromPrefetchCache: true}, "(prefetch)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := Network(&buf, []ipc.NetworkEntry{tc.entry}, OutputOptions{UseColor: false}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out := buf.String(); !strings.Contains(out, tc.token) {
				t.Errorf("expected cache token %q in:\n%s", tc.token, out)
			}
		})
	}

	var buf bytes.Buffer
	if err := Network(&buf, []ipc.NetworkEntry{{Method: "GET", URL: "https://example.com/net", Status: 200}}, OutputOptions{UseColor: false}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out := buf.String(); strings.Contains(out, "(disk)") || strings.Contains(out, "(service-worker)") || strings.Contains(out, "(prefetch)") {
		t.Errorf("an uncached response should carry no cache token:\n%s", out)
	}
}

func TestNetwork_TimingLine(t *testing.T) {
	// Present phases render as integer ms in request order; a sub-millisecond
	// phase (send) rounds to zero and is dropped.
	entries := []ipc.NetworkEntry{
		{Method: "GET", URL: "https://example.com/", Status: 200, Duration: 0.962,
			Timing: &ipc.NetworkTiming{DNSMs: 122.188, ConnectMs: 420.256, TLSMs: 213.257, SendMs: 0.296, WaitMs: 206.64}},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, netStd()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "       timing: dns 122ms connect 420ms tls 213ms wait 207ms\n") {
		t.Errorf("timing line should list present phases as integer ms, dropping sub-ms send:\n%s", output)
	}
}

func TestNetwork_InitiatorLine(t *testing.T) {
	// An initiator with a location renders "type url:line"; a bare "other"
	// initiator with no location is omitted.
	entries := []ipc.NetworkEntry{
		{Method: "GET", URL: "https://example.com/app.js", Status: 200,
			Initiator: &ipc.NetworkInitiator{Type: "parser", URL: "https://example.com/", Line: 5}},
		{Method: "GET", URL: "https://example.com/", Status: 200, Type: "Document",
			Initiator: &ipc.NetworkInitiator{Type: "other"}},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, netStd()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "       initiator: parser https://example.com/:5\n") {
		t.Errorf("initiator line should render type and location:\n%s", output)
	}
	if strings.Contains(output, "initiator: other") {
		t.Errorf("a locationless 'other' initiator should be omitted:\n%s", output)
	}
}

func TestNetwork_InitiatorLineOmitsZeroLine(t *testing.T) {
	// CDP line numbers are 0-based, so line 0 is the top of a document. The
	// initiator line drops the ":line" suffix in that case, matching the console
	// formatter, rather than rendering a dangling ":0".
	entries := []ipc.NetworkEntry{
		{Method: "GET", URL: "https://example.com/", Status: 200, Type: "Document",
			Initiator: &ipc.NetworkInitiator{Type: "parser", URL: "https://example.com/", Line: 0}},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, netStd()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "       initiator: parser https://example.com/\n") {
		t.Errorf("line-0 initiator should render the URL without a line suffix:\n%s", output)
	}
	if strings.Contains(output, "https://example.com/:0") {
		t.Errorf("line-0 initiator should not render a dangling ':0':\n%s", output)
	}
}

func TestNetwork_TimingLineOmittedWhenAbsent(t *testing.T) {
	// No timing captured: no timing line.
	entries := []ipc.NetworkEntry{
		{Method: "GET", URL: "https://example.com/", Status: 200, Duration: 0.045},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, netStd()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(buf.String(), "timing:") {
		t.Errorf("an entry with no timing should omit the timing line:\n%s", buf.String())
	}
}

func TestNetwork_TypeShownOnMainLine(t *testing.T) {
	// The CDP resource type appends to the main line so a reader can tell an xhr
	// from a document or image at a glance.
	entries := []ipc.NetworkEntry{
		{Method: "GET", URL: "https://api.example.com/data", Status: 200, Duration: 0.045, Type: "xhr"},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, OutputOptions{UseColor: false}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "GET https://api.example.com/data 200 45ms xhr") {
		t.Errorf("resource type should append to the main line:\n%s", output)
	}
}

func TestNetwork_FailedRequestShowsReason(t *testing.T) {
	// A failed request must surface its reason and a distinct FAILED token rather
	// than a bare status of 0.
	entries := []ipc.NetworkEntry{
		{Method: "GET", URL: "https://example.com/x", Failed: true, Error: "net::ERR_NAME_NOT_RESOLVED", Duration: 0.012, Type: "document",
			Initiator: &ipc.NetworkInitiator{Type: "script", URL: "https://example.com/app.js", Line: 1}},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, netStd()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "GET https://example.com/x FAILED 12ms document") {
		t.Errorf("failed request should show a FAILED token and resource type, not a bare status:\n%s", output)
	}
	if !strings.Contains(output, "       error: net::ERR_NAME_NOT_RESOLVED") {
		t.Errorf("failed request should show its reason on an indented error line:\n%s", output)
	}
	// The error reason leads the detail block, before any initiator metadata.
	if errIdx, initIdx := strings.Index(output, "       error:"), strings.Index(output, "       initiator:"); errIdx == -1 || initIdx == -1 || errIdx > initIdx {
		t.Errorf("error line should precede the initiator line on a failed entry:\n%s", output)
	}
}

func TestNetwork_FailedRequestShowsRequestBody(t *testing.T) {
	// A failed request has no response, but its outgoing body is diagnostic and
	// must still render, as it does on the success path.
	entries := []ipc.NetworkEntry{
		{Method: "POST", URL: "https://example.com/submit", Failed: true, Error: "canceled", RequestBody: `{"user":"grant"}`},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, netFull()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `       request: {"user":"grant"}`) {
		t.Errorf("a failed request should still show its request body at full:\n%s", output)
	}
}

func TestNetwork_FailedRequestShowsRequestHeaders(t *testing.T) {
	// With ShowHeaders set, a failed request renders its request headers, since the
	// request side is captured and diagnostic. It has no response, so response
	// headers must never appear even when ShowHeaders is set.
	entries := []ipc.NetworkEntry{
		{
			Method: "GET", URL: "https://example.com/x", Failed: true, Error: "canceled",
			RequestHeaders:  map[string]string{"accept": "application/json"},
			ResponseHeaders: map[string]string{"content-type": "text/html"},
		},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, OutputOptions{UseColor: false, ShowHeaders: true, Detail: DetailStandard}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "       request-headers:\n         accept: application/json\n") {
		t.Errorf("a failed request should render its request headers under --headers:\n%s", output)
	}
	if strings.Contains(output, "response-headers:") {
		t.Errorf("a failed request has no response, so response headers must not render:\n%s", output)
	}
}

func TestNetwork_FailedWithoutReasonStillDistinct(t *testing.T) {
	// A failed entry with no captured reason must still render FAILED and must not
	// print an empty error line.
	entries := []ipc.NetworkEntry{
		{Method: "POST", URL: "https://example.com/y", Failed: true, Duration: 0.001},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, OutputOptions{UseColor: false}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "POST https://example.com/y FAILED 1ms") {
		t.Errorf("a failed entry with no reason should still show FAILED:\n%s", output)
	}
	if strings.Contains(output, "error:") {
		t.Errorf("no error line should print when Error is empty:\n%s", output)
	}
}

func TestNetwork_ZeroStatusNotFailed(t *testing.T) {
	// The failure branch keys on Failed, not status == 0; a non-failed entry with
	// status 0 must render its bare status, never FAILED.
	entries := []ipc.NetworkEntry{
		{Method: "GET", URL: "https://example.com/pending", Status: 0, Duration: 0.005},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, OutputOptions{UseColor: false}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "FAILED") {
		t.Errorf("a zero-status non-failed entry must not render as FAILED:\n%s", output)
	}
	if !strings.Contains(output, "GET https://example.com/pending 0 5ms") {
		t.Errorf("a zero-status non-failed entry should render its bare status:\n%s", output)
	}
}

func TestNetwork_GETResponseBodyPrints(t *testing.T) {
	// A GET response body must print: dropping the old Method != GET gate means a
	// GET's payload is shown like any other method's.
	entries := []ipc.NetworkEntry{
		{Method: "GET", URL: "https://api.example.com/data", Status: 200, ResponseBody: `{"ok":true}`},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, netFull()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "request:") {
		t.Error("a GET with no request body should not print a request line")
	}
	if !strings.Contains(output, `       response: {"ok":true}`) {
		t.Errorf("GET response body should print:\n%s", output)
	}
}

func TestNetwork_MultiLineBody(t *testing.T) {
	entries := []ipc.NetworkEntry{
		{Method: "POST", URL: "https://api.example.com", Status: 200, RequestBody: "line1\nline2"},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, netFull()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Multi-line body: bare label line at seven spaces, each line nested at nine.
	if !strings.Contains(output, "       request:\n") {
		t.Errorf("multi-line body should print a bare label line:\n%s", output)
	}
	if !strings.Contains(output, "         line1\n") || !strings.Contains(output, "         line2\n") {
		t.Errorf("multi-line body lines should be nested nine spaces:\n%s", output)
	}
}

func TestNetwork_TruncatedBodyMarker(t *testing.T) {
	// A body the CLI bounded to --max-body-size reaches the formatter with its
	// truncation flag set; text output must flag the cut so a human reader is not
	// misled by a payload that silently ends.
	entries := []ipc.NetworkEntry{
		{
			Method:                "POST",
			URL:                   "https://api.example.com",
			Status:                200,
			RequestBody:           `{"chunk":"AAAA`,
			RequestBodyTruncated:  true,
			ResponseBody:          `{"ok":tr`,
			ResponseBodyTruncated: true,
		},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, netFull()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if strings.Count(output, "… [truncated]") != 2 {
		t.Errorf("both truncated bodies should print a marker:\n%s", output)
	}
}

func TestNetwork_NoMarkerWhenNotTruncated(t *testing.T) {
	entries := []ipc.NetworkEntry{
		{Method: "POST", URL: "https://api.example.com", Status: 200, RequestBody: `{"ok":true}`},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, netFull()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(buf.String(), "truncated") {
		t.Errorf("a body within budget should not print a truncation marker:\n%s", buf.String())
	}
}

func TestNetwork_BinaryBodyPathPrints(t *testing.T) {
	// A binary response body is filed rather than stored on ResponseBody; text
	// output must point the reader at the saved file instead of printing nothing.
	entries := []ipc.NetworkEntry{
		{
			Method:           "GET",
			URL:              "https://api.example.com/image.png",
			Status:           200,
			ResponseBodyPath: "/tmp/webctl/image.png",
		},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, netFull()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "       response: [binary saved to /tmp/webctl/image.png]\n") {
		t.Errorf("binary response body should print its saved path:\n%s", output)
	}
	if strings.Count(output, "response:") != 1 {
		t.Errorf("a filed binary body should print exactly one response line:\n%s", output)
	}
}

func TestNetwork_TextBodySuppressesBinaryPath(t *testing.T) {
	// A text body and a filed binary path are mutually exclusive per entry; if
	// both were ever set the formatter must print the payload, not a second
	// response line pointing at a file.
	entries := []ipc.NetworkEntry{
		{
			Method:           "GET",
			URL:              "https://api.example.com/data",
			Status:           200,
			ResponseBody:     `{"ok":true}`,
			ResponseBodyPath: "/tmp/webctl/data.bin",
		},
	}

	var buf bytes.Buffer
	if err := Network(&buf, entries, netFull()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `       response: {"ok":true}`) {
		t.Errorf("a text response body should print its payload:\n%s", output)
	}
	if strings.Contains(output, "binary saved to") {
		t.Errorf("a present text body should suppress the binary path line:\n%s", output)
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

func TestTab(t *testing.T) {
	data := ipc.TabData{
		ActiveSession: "session1",
		Sessions: []ipc.PageSession{
			{ID: "session1", URL: "https://example.com", Title: "Example"},
			{ID: "session2", URL: "https://other.com", Title: "Other"},
		},
	}

	var buf bytes.Buffer
	opts := OutputOptions{UseColor: false}
	err := Tab(&buf, data, opts)
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

func TestTabError_AmbiguousMatches(t *testing.T) {
	matches := []ipc.PageSession{
		{ID: "abc12345", Title: "Test 1"},
		{ID: "def67890", Title: "Test 2"},
	}

	var buf bytes.Buffer
	opts := OutputOptions{UseColor: false}
	err := TabError(&buf, "ambiguous query 'test', matches multiple tabs", nil, matches, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Error:") {
		t.Error("expected Error: prefix")
	}
	if !strings.Contains(output, "Matching tabs:") {
		t.Error("expected Matching tabs: header")
	}
	if !strings.Contains(output, "abc12345") || !strings.Contains(output, "def67890") {
		t.Error("expected match IDs in output")
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
		elements []ipc.ElementWithStyles
		expected string
	}{
		{
			name: "single element with id",
			elements: []ipc.ElementWithStyles{
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", ID: "header"},
					Inline:      "color: red; font-size: 16px;",
				},
			},
			expected: "#header\ncolor: red; font-size: 16px;\n",
		},
		{
			name: "multiple elements with different identifiers",
			elements: []ipc.ElementWithStyles{
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", ID: "header"},
					Inline:      "color: red;",
				},
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", Class: "panel"},
					Inline:      "background: blue;",
				},
				{
					ElementMeta: ipc.ElementMeta{Tag: "span"},
					Inline:      "margin: 10px;",
				},
			},
			expected: "#header\ncolor: red;\n--\n.panel:2\nbackground: blue;\n--\nspan:3\nmargin: 10px;\n",
		},
		{
			name: "empty inline style",
			elements: []ipc.ElementWithStyles{
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", ID: "empty"},
					Inline:      "",
				},
			},
			expected: "#empty\n(empty)\n",
		},
		{
			name: "mixed empty and non-empty",
			elements: []ipc.ElementWithStyles{
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", Class: "one"},
					Inline:      "color: red;",
				},
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", Class: "two"},
					Inline:      "",
				},
				{
					ElementMeta: ipc.ElementMeta{Tag: "div"},
					Inline:      "margin: 10px;",
				},
			},
			expected: ".one:1\ncolor: red;\n--\n.two:2\n(empty)\n--\ndiv:3\nmargin: 10px;\n",
		},
		{
			name:     "no elements",
			elements: []ipc.ElementWithStyles{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := InlineStyles(&buf, tt.elements)
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
		name     string
		elements []ipc.ElementWithStyles
		wantSep  bool
		expected string
	}{
		{
			name:     "empty list",
			elements: []ipc.ElementWithStyles{},
			wantSep:  false,
			expected: "",
		},
		{
			name: "single element with id",
			elements: []ipc.ElementWithStyles{
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", ID: "header"},
					Styles:      map[string]string{"color": "red"},
				},
			},
			wantSep:  false,
			expected: "#header\ncolor: red\n",
		},
		{
			name: "multiple elements with different identifiers",
			elements: []ipc.ElementWithStyles{
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", ID: "header"},
					Styles:      map[string]string{"color": "red"},
				},
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", Class: "panel"},
					Styles:      map[string]string{"color": "blue"},
				},
			},
			wantSep:  true,
			expected: "#header\ncolor: red\n--\n.panel:2\ncolor: blue\n",
		},
		{
			name: "multiple elements same class",
			elements: []ipc.ElementWithStyles{
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", Class: "item"},
					Styles:      map[string]string{"margin": "10px"},
				},
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", Class: "item"},
					Styles:      map[string]string{"margin": "20px"},
				},
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", Class: "item"},
					Styles:      map[string]string{"margin": "30px"},
				},
			},
			wantSep:  true,
			expected: ".item:1\nmargin: 10px\n--\n.item:2\nmargin: 20px\n--\n.item:3\nmargin: 30px\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := ComputedStylesMulti(&buf, tt.elements)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := buf.String()
			hasSep := strings.Contains(got, "--")
			if hasSep != tt.wantSep {
				t.Errorf("separator present = %v, want %v, output: %q", hasSep, tt.wantSep, got)
			}

			if tt.expected != "" && got != tt.expected {
				t.Errorf("got:\n%q\nwant:\n%q", got, tt.expected)
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

// Tests for element identification feature

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "alphanumeric",
			input:    "header123",
			expected: "header123",
		},
		{
			name:     "with hyphens",
			input:    "my-header",
			expected: "my-header",
		},
		{
			name:     "with underscores",
			input:    "my_header",
			expected: "my_header",
		},
		{
			name:     "mixed valid chars",
			input:    "my-header_123",
			expected: "my-header_123",
		},
		{
			name:     "with spaces",
			input:    "my header",
			expected: "myheader",
		},
		{
			name:     "with special chars",
			input:    "header@#$%",
			expected: "header",
		},
		{
			name:     "with dots",
			input:    "my.header.class",
			expected: "myheaderclass",
		},
		{
			name:     "with brackets",
			input:    "header[data]",
			expected: "headerdata",
		},
		{
			name:     "only special chars",
			input:    "@#$%",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "unicode chars",
			input:    "header™©",
			expected: "header",
		},
		{
			name:     "mixed case",
			input:    "MyHeader",
			expected: "MyHeader",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeIdentifier(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeIdentifier(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFormatElementIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		meta     ipc.ElementMeta
		index    int
		expected string
	}{
		{
			name:     "element with id",
			meta:     ipc.ElementMeta{Tag: "div", ID: "header"},
			index:    0,
			expected: "#header",
		},
		{
			name:     "element with id (ignores class)",
			meta:     ipc.ElementMeta{Tag: "div", ID: "header", Class: "panel"},
			index:    0,
			expected: "#header",
		},
		{
			name:     "element with class only",
			meta:     ipc.ElementMeta{Tag: "div", Class: "panel"},
			index:    0,
			expected: ".panel:1",
		},
		{
			name:     "element with class at index 2",
			meta:     ipc.ElementMeta{Tag: "div", Class: "panel"},
			index:    2,
			expected: ".panel:3",
		},
		{
			name:     "element with tag only",
			meta:     ipc.ElementMeta{Tag: "div"},
			index:    0,
			expected: "div:1",
		},
		{
			name:     "element with tag at index 5",
			meta:     ipc.ElementMeta{Tag: "span"},
			index:    5,
			expected: "span:6",
		},
		{
			name:     "id with special chars",
			meta:     ipc.ElementMeta{Tag: "div", ID: "header@#$"},
			index:    0,
			expected: "#header",
		},
		{
			name:     "id with only special chars (falls back to class)",
			meta:     ipc.ElementMeta{Tag: "div", ID: "@#$", Class: "panel"},
			index:    0,
			expected: ".panel:1",
		},
		{
			name:     "id with only special chars (falls back to tag)",
			meta:     ipc.ElementMeta{Tag: "div", ID: "@#$"},
			index:    0,
			expected: "div:1",
		},
		{
			name:     "class with special chars",
			meta:     ipc.ElementMeta{Tag: "div", Class: "panel@#$"},
			index:    0,
			expected: ".panel:1",
		},
		{
			name:     "class with only special chars (falls back to tag)",
			meta:     ipc.ElementMeta{Tag: "div", Class: "@#$"},
			index:    0,
			expected: "div:1",
		},
		{
			name:     "empty id (uses class)",
			meta:     ipc.ElementMeta{Tag: "div", ID: "", Class: "panel"},
			index:    0,
			expected: ".panel:1",
		},
		{
			name:     "empty class (uses tag)",
			meta:     ipc.ElementMeta{Tag: "div", ID: "", Class: ""},
			index:    0,
			expected: "div:1",
		},
		{
			name:     "id with spaces",
			meta:     ipc.ElementMeta{Tag: "div", ID: "my header"},
			index:    0,
			expected: "#myheader",
		},
		{
			name:     "class with hyphens",
			meta:     ipc.ElementMeta{Tag: "div", Class: "my-panel"},
			index:    0,
			expected: ".my-panel:1",
		},
		{
			name:     "class with underscores",
			meta:     ipc.ElementMeta{Tag: "div", Class: "my_panel"},
			index:    0,
			expected: ".my_panel:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatElementIdentifier(tt.meta, tt.index)
			if got != tt.expected {
				t.Errorf("formatElementIdentifier(%+v, %d) = %q, want %q", tt.meta, tt.index, got, tt.expected)
			}
		})
	}
}

func TestInlineStylesWithElementIdentification(t *testing.T) {
	tests := []struct {
		name     string
		elements []ipc.ElementWithStyles
		wantID   string
		wantCSS  string
	}{
		{
			name: "id-based identification",
			elements: []ipc.ElementWithStyles{
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", ID: "header"},
					Inline:      "color: blue;",
				},
			},
			wantID:  "#header",
			wantCSS: "color: blue;",
		},
		{
			name: "class-based identification",
			elements: []ipc.ElementWithStyles{
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", Class: "panel"},
					Inline:      "margin: 10px;",
				},
			},
			wantID:  ".panel:1",
			wantCSS: "margin: 10px;",
		},
		{
			name: "tag-based identification",
			elements: []ipc.ElementWithStyles{
				{
					ElementMeta: ipc.ElementMeta{Tag: "span"},
					Inline:      "font-size: 14px;",
				},
			},
			wantID:  "span:1",
			wantCSS: "font-size: 14px;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := InlineStyles(&buf, tt.elements)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := buf.String()
			if !strings.Contains(output, tt.wantID) {
				t.Errorf("output should contain identifier %q, got: %s", tt.wantID, output)
			}
			if !strings.Contains(output, tt.wantCSS) {
				t.Errorf("output should contain CSS %q, got: %s", tt.wantCSS, output)
			}
		})
	}
}

func TestComputedStylesMultiWithElementIdentification(t *testing.T) {
	tests := []struct {
		name     string
		elements []ipc.ElementWithStyles
		wantIDs  []string
	}{
		{
			name: "multiple elements with unique identifiers",
			elements: []ipc.ElementWithStyles{
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", ID: "header"},
					Styles:      map[string]string{"color": "red"},
				},
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", Class: "panel"},
					Styles:      map[string]string{"background": "blue"},
				},
				{
					ElementMeta: ipc.ElementMeta{Tag: "span"},
					Styles:      map[string]string{"margin": "5px"},
				},
			},
			wantIDs: []string{"#header", ".panel:2", "span:3"},
		},
		{
			name: "multiple elements same class",
			elements: []ipc.ElementWithStyles{
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", Class: "item"},
					Styles:      map[string]string{"padding": "10px"},
				},
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", Class: "item"},
					Styles:      map[string]string{"padding": "20px"},
				},
			},
			wantIDs: []string{".item:1", ".item:2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := ComputedStylesMulti(&buf, tt.elements)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := buf.String()
			for _, wantID := range tt.wantIDs {
				if !strings.Contains(output, wantID) {
					t.Errorf("output should contain identifier %q, got: %s", wantID, output)
				}
			}
		})
	}
}

func TestElementIdentificationEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		elements []ipc.ElementWithStyles
		expected string
	}{
		{
			name: "empty inline style shows (empty)",
			elements: []ipc.ElementWithStyles{
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", ID: "empty"},
					Inline:      "",
				},
			},
			expected: "#empty\n(empty)\n",
		},
		{
			name: "id with whitespace",
			elements: []ipc.ElementWithStyles{
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", ID: "   "},
					Inline:      "color: red;",
				},
			},
			expected: "div:1\ncolor: red;\n",
		},
		{
			name: "class with whitespace",
			elements: []ipc.ElementWithStyles{
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", Class: "   "},
					Inline:      "color: red;",
				},
			},
			expected: "div:1\ncolor: red;\n",
		},
		{
			name: "separator between elements",
			elements: []ipc.ElementWithStyles{
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", ID: "first"},
					Inline:      "color: red;",
				},
				{
					ElementMeta: ipc.ElementMeta{Tag: "div", ID: "second"},
					Inline:      "color: blue;",
				},
			},
			expected: "#first\ncolor: red;\n--\n#second\ncolor: blue;\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := InlineStyles(&buf, tt.elements)
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
