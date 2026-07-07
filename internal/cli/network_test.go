package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/grantcarthew/webctl/internal/cli/format"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

func TestResolveMaxBodySize(t *testing.T) {
	// Mirror the real tree: --max-body-size is a persistent flag on `network`,
	// inherited by the `save` subcommand. The unset default is mode-dependent, so
	// the caller supplies it; each test passes a distinctive sentinel to prove the
	// resolver returns it verbatim when the flag is unset and the explicit value
	// otherwise.
	const unsetDefault = 55555
	newTree := func(got *int) *cobra.Command {
		parent := &cobra.Command{Use: "network", Run: func(c *cobra.Command, _ []string) {
			*got = resolveMaxBodySize(c, unsetDefault)
		}}
		// Registered default 0 matches production, where pflag omits a misleading
		// "(default N)" and resolution keys on Changed rather than this value.
		parent.PersistentFlags().Int("max-body-size", 0, "")
		child := &cobra.Command{Use: "save", Run: func(c *cobra.Command, _ []string) {
			*got = resolveMaxBodySize(c, unsetDefault)
		}}
		parent.AddCommand(child)
		return parent
	}

	tests := []struct {
		name string
		args []string
		want int
	}{
		{"unset uses caller default", []string{}, unsetDefault},
		{"explicit value", []string{"--max-body-size", "2048"}, 2048},
		{"explicit zero is honoured", []string{"--max-body-size", "0"}, 0},
		{"explicit unlimited is honoured", []string{"--max-body-size", "-1"}, ipc.MaxBodySizeUnlimited},
		{"save unset uses caller default", []string{"save"}, unsetDefault},
		{"save inherits explicit value", []string{"save", "--max-body-size", "4096"}, 4096},
		{"save inherits explicit zero", []string{"save", "--max-body-size", "0"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got int
			root := newTree(&got)
			root.SetArgs(tt.args)
			if err := root.Execute(); err != nil {
				t.Fatalf("execute: %v", err)
			}
			if got != tt.want {
				t.Errorf("resolveMaxBodySize = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTruncateBody(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		maxBytes  int
		want      string
		truncated bool
	}{
		{"under budget", "hello", 10, "hello", false},
		{"at budget", "hello", 5, "hello", false},
		{"over budget ascii", "hello world", 5, "hello", true},
		{"zero budget", "hello", 0, "", true},
		{"negative budget clamps to zero", "hello", -1, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, truncated := truncateBody(tt.body, tt.maxBytes)
			if got != tt.want || truncated != tt.truncated {
				t.Errorf("truncateBody(%q, %d) = (%q, %v), want (%q, %v)",
					tt.body, tt.maxBytes, got, truncated, tt.want, tt.truncated)
			}
		})
	}
}

func TestTruncateBody_RuneBoundary(t *testing.T) {
	// "é" is two bytes (0xC3 0xA9). A budget that lands mid-rune must back up to
	// the boundary, never splitting the multibyte rune.
	body := "aéb" // bytes: 'a'(1) 'é'(2) 'b'(1) = 4 bytes
	got, truncated := truncateBody(body, 2)
	if !truncated {
		t.Fatal("expected truncation")
	}
	if got != "a" {
		t.Errorf("truncateBody = %q, want %q (must not split the multibyte rune)", got, "a")
	}
	if !isValidUTF8(got) {
		t.Errorf("result %q is not valid UTF-8", got)
	}
}

func isValidUTF8(s string) bool {
	return strings.ToValidUTF8(s, "�") == s
}

func TestApplyBodyTruncation_RequestAndResponse(t *testing.T) {
	entries := []ipc.NetworkEntry{
		{
			RequestBody:  "0123456789",
			ResponseBody: "abcdefghij",
		},
	}
	applyBodyTruncation(entries, 4)

	if entries[0].RequestBody != "0123" {
		t.Errorf("RequestBody = %q, want %q", entries[0].RequestBody, "0123")
	}
	if !entries[0].RequestBodyTruncated {
		t.Error("RequestBodyTruncated should be true")
	}
	if entries[0].ResponseBody != "abcd" {
		t.Errorf("ResponseBody = %q, want %q", entries[0].ResponseBody, "abcd")
	}
	if !entries[0].ResponseBodyTruncated {
		t.Error("ResponseBodyTruncated should be true")
	}
}

func TestApplyBodyTruncation_NoFlagWhenWithinBudget(t *testing.T) {
	entries := []ipc.NetworkEntry{{RequestBody: "tiny", ResponseBody: "small"}}
	applyBodyTruncation(entries, 1024)

	if entries[0].RequestBodyTruncated || entries[0].ResponseBodyTruncated {
		t.Error("no truncation flag should be set when bodies fit the budget")
	}
}

func TestFilterNetworkByText_MatchesRequestBody(t *testing.T) {
	entries := []ipc.NetworkEntry{
		{URL: "https://api.example.com/login", RequestBody: `{"username":"grant"}`},
		{URL: "https://api.example.com/other", ResponseBody: `{"result":"ok"}`},
	}

	matched := filterNetworkByText(entries, "grant")
	if len(matched) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matched))
	}
	if matched[0].URL != "https://api.example.com/login" {
		t.Errorf("matched wrong entry: %s", matched[0].URL)
	}
}

func TestFilterNetworkByText_StillMatchesResponseBody(t *testing.T) {
	entries := []ipc.NetworkEntry{
		{URL: "https://api.example.com/login", RequestBody: `{"username":"grant"}`},
		{URL: "https://api.example.com/other", ResponseBody: `{"token":"abc123"}`},
	}

	matched := filterNetworkByText(entries, "abc123")
	if len(matched) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matched))
	}
	if matched[0].URL != "https://api.example.com/other" {
		t.Errorf("matched wrong entry: %s", matched[0].URL)
	}
}

func TestApplyBodyTruncation_UnlimitedLeavesBodiesIntact(t *testing.T) {
	entries := []ipc.NetworkEntry{{RequestBody: "0123456789", ResponseBody: "abcdefghij"}}
	applyBodyTruncation(entries, ipc.MaxBodySizeUnlimited)

	if entries[0].RequestBody != "0123456789" || entries[0].ResponseBody != "abcdefghij" {
		t.Error("unlimited (-1) must leave bodies untouched")
	}
	if entries[0].RequestBodyTruncated || entries[0].ResponseBodyTruncated {
		t.Error("unlimited (-1) must not set truncation flags")
	}
}

// sparseEntries builds a set whose held seqs are non-contiguous, mirroring an
// active session interleaved with background-tab traffic: 318, 320, 421, 425.
func sparseEntries() []ipc.NetworkEntry {
	return []ipc.NetworkEntry{
		{Seq: 318, URL: "https://a"},
		{Seq: 320, URL: "https://b"},
		{Seq: 421, URL: "https://c"},
		{Seq: 425, URL: "https://d"},
	}
}

func TestApplyNetworkLimiting_SeqRangeInclusiveMembership(t *testing.T) {
	// Endpoints 319 and 422 are absent, yet interior held seqs match.
	got, err := applyNetworkLimiting(sparseEntries(), 0, 0, "319-422")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var seqs []uint64
	for _, e := range got {
		seqs = append(seqs, e.Seq)
	}
	want := []uint64{320, 421}
	if len(seqs) != len(want) || seqs[0] != want[0] || seqs[1] != want[1] {
		t.Errorf("range 319-422 = %v, want %v", seqs, want)
	}
}

func TestApplyNetworkLimiting_SeqRangeIncludesPresentEndpoints(t *testing.T) {
	got, err := applyNetworkLimiting(sparseEntries(), 0, 0, "318-425")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 4 {
		t.Errorf("range 318-425 should hold all 4 entries, got %d", len(got))
	}
}

func TestApplyNetworkLimiting_SeqRangeEmptyInterval(t *testing.T) {
	got, err := applyNetworkLimiting(sparseEntries(), 0, 0, "321-420")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("range 321-420 holds no seq, want empty, got %d", len(got))
	}
}

func TestApplyNetworkLimiting_RangeInvalidFormat(t *testing.T) {
	if _, err := applyNetworkLimiting(sparseEntries(), 0, 0, "notarange"); err == nil {
		t.Error("expected error for malformed range")
	}
}

func TestApplyNetworkLimiting_HeadTailAreCounts(t *testing.T) {
	head, err := applyNetworkLimiting(sparseEntries(), 2, 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(head) != 2 || head[0].Seq != 318 || head[1].Seq != 320 {
		t.Errorf("head 2 = %v, want first two entries", head)
	}
	tail, err := applyNetworkLimiting(sparseEntries(), 0, 2, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tail) != 2 || tail[0].Seq != 421 || tail[1].Seq != 425 {
		t.Errorf("tail 2 = %v, want last two entries", tail)
	}
}

func TestFindNetworkEntryBySeq(t *testing.T) {
	entries := sparseEntries()

	// A held seq resolves by exact membership.
	if e, ok := findNetworkEntryBySeq(entries, 421); !ok || e.URL != "https://c" {
		t.Errorf("seq 421 should resolve to https://c, got ok=%v entry=%+v", ok, e)
	}

	// A seq between the lowest and highest but not held must miss: membership, not
	// range.
	if _, ok := findNetworkEntryBySeq(entries, 400); ok {
		t.Error("seq 400 is between bounds but not held; must miss")
	}

	// A negative index never matches.
	if _, ok := findNetworkEntryBySeq(entries, -1); ok {
		t.Error("negative index must miss")
	}
}

func TestNetworkSeqBounds(t *testing.T) {
	lo, hi, ok := networkSeqBounds(sparseEntries())
	if !ok || lo != 318 || hi != 425 {
		t.Errorf("bounds = (%d, %d, %v), want (318, 425, true)", lo, hi, ok)
	}

	if _, _, ok := networkSeqBounds(nil); ok {
		t.Error("empty set should report no bounds")
	}
}

func TestNetworkDrilldownMissMessage(t *testing.T) {
	// Sparse held set: a between-bounds miss names the bounds as orientation.
	got := networkDrilldownMissMessage(400, sparseEntries())
	want := "entry 400 not in buffer (holds seq 318-425; run network to list)"
	if got != want {
		t.Errorf("miss message = %q, want %q", got, want)
	}

	// Empty buffer: there is no bound to name.
	got = networkDrilldownMissMessage(42, nil)
	want = "entry 42 not in buffer (buffer empty)"
	if got != want {
		t.Errorf("empty-buffer message = %q, want %q", got, want)
	}
}

func TestBuildSchema_UnionCollapsedArray(t *testing.T) {
	// A heterogeneous object array must union its keys so no field is hidden.
	body := `{"count":3,"vehicles":[{"id":1,"name":"a"},{"id":2,"options":["x"]}]}`
	var parsed any
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	skeleton := buildSchema(parsed)
	out, err := json.Marshal(skeleton)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}

	got := string(out)
	want := `{"count":"number","vehicles":[{"id":"number","name":"string","options":["string"]}]}`
	if got != want {
		t.Errorf("schema = %s, want %s", got, want)
	}
}

func TestBuildSchema_ScalarLeaves(t *testing.T) {
	cases := map[string]string{
		`"hi"`:    `"string"`,
		`5`:       `"number"`,
		`true`:    `"boolean"`,
		`null`:    `"null"`,
		`[]`:      `[]`,
		`{"a":1}`: `{"a":"number"}`,
	}
	for body, want := range cases {
		var parsed any
		if err := json.Unmarshal([]byte(body), &parsed); err != nil {
			t.Fatalf("unmarshal %s: %v", body, err)
		}
		out, err := json.Marshal(buildSchema(parsed))
		if err != nil {
			t.Fatalf("marshal %s: %v", body, err)
		}
		if string(out) != want {
			t.Errorf("buildSchema(%s) = %s, want %s", body, out, want)
		}
	}
}

func TestResolveDetailLevel(t *testing.T) {
	newCmd := func(level string) *cobra.Command {
		c := &cobra.Command{Use: "network"}
		c.Flags().String("detail", "standard", "")
		if level != "" {
			_ = c.Flags().Set("detail", level)
		}
		return c
	}

	cases := []struct {
		level string
		want  format.DetailLevel
		err   bool
	}{
		{"summary", format.DetailSummary, false},
		{"standard", format.DetailStandard, false},
		{"full", format.DetailFull, false},
		{"", format.DetailStandard, false},
		{"verbose", format.DetailStandard, true},
	}
	for _, tc := range cases {
		got, err := resolveDetailLevel(newCmd(tc.level))
		if (err != nil) != tc.err {
			t.Errorf("detail %q: err = %v, want err=%v", tc.level, err, tc.err)
		}
		if got != tc.want {
			t.Errorf("detail %q = %v, want %v", tc.level, got, tc.want)
		}
	}
}

// captureStream redirects the given *os.File (os.Stdout or os.Stderr) for the
// duration of fn and returns what was written. Output here is small, so the pipe
// never fills before it is drained.
func captureStream(t *testing.T, stream **os.File, fn func()) string {
	t.Helper()
	old := *stream
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	*stream = w
	// Restore via defer so a t.Fatalf inside fn (which unwinds via runtime.Goexit)
	// cannot leave the process stream pointed at a closed pipe.
	defer func() { *stream = old }()
	fn()
	_ = w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

// setNetworkFlag sets a flag on the shared networkCmd and restores its prior
// value and Changed state after the test, so command-level tests do not bleed
// flag state into one another.
func setNetworkFlag(t *testing.T, name, value string) {
	t.Helper()
	f := networkCmd.Flags().Lookup(name)
	if f == nil {
		t.Fatalf("flag %q not found on networkCmd", name)
	}
	old, oldChanged := f.Value.String(), f.Changed
	if err := networkCmd.Flags().Set(name, value); err != nil {
		t.Fatalf("set %s=%s: %v", name, value, err)
	}
	t.Cleanup(func() {
		_ = f.Value.Set(old)
		f.Changed = oldChanged
	})
}

func TestOutputNetworkSchema_JSONBody(t *testing.T) {
	entry := ipc.NetworkEntry{
		MimeType:     "application/json",
		ResponseBody: `{"count":2,"items":[{"id":1},{"id":2,"tag":"x"}]}`,
	}

	out := captureStream(t, &os.Stdout, func() {
		if err := outputNetworkSchema(entry); err != nil {
			t.Fatalf("outputNetworkSchema: %v", err)
		}
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("parse schema envelope: %v\n%s", err, out)
	}
	if resp["ok"] != true {
		t.Errorf("expected ok=true, got %v", resp["ok"])
	}
	if _, hasNotice := resp["notice"]; hasNotice {
		t.Errorf("a parsed body must carry no notice:\n%s", out)
	}

	// The array unions keys across its elements (id and tag) with type-name leaves.
	skeleton, err := json.Marshal(resp["schema"])
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	want := `{"count":"number","items":[{"id":"number","tag":"string"}]}`
	if string(skeleton) != want {
		t.Errorf("schema = %s, want %s", skeleton, want)
	}
}

func TestOutputNetworkSchema_NonJSONBody(t *testing.T) {
	cases := []struct {
		name       string
		entry      ipc.NetworkEntry
		wantNotice string
	}{
		{"html", ipc.NetworkEntry{MimeType: "text/html", ResponseBody: "<html></html>"}, "response body is not JSON (text/html)"},
		{"empty", ipc.NetworkEntry{ResponseBody: ""}, "response body is not JSON"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := captureStream(t, &os.Stdout, func() {
				if err := outputNetworkSchema(tc.entry); err != nil {
					t.Fatalf("outputNetworkSchema: %v", err)
				}
			})

			var resp map[string]any
			if err := json.Unmarshal([]byte(out), &resp); err != nil {
				t.Fatalf("parse schema envelope: %v\n%s", err, out)
			}
			if resp["ok"] != true {
				t.Errorf("expected ok=true, got %v", resp["ok"])
			}
			if schema, present := resp["schema"]; !present || schema != nil {
				t.Errorf("non-JSON body must report schema null, got %v", schema)
			}
			if resp["notice"] != tc.wantNotice {
				t.Errorf("notice = %v, want %q", resp["notice"], tc.wantNotice)
			}
		})
	}
}

func TestRunNetwork_SchemaRequiresIndex(t *testing.T) {
	enableJSONOutput(t)
	setNetworkFlag(t, "schema", "true")
	// The index check fires before the daemon check, so no executor is needed.
	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	var err error
	out := captureStream(t, &os.Stderr, func() {
		err = runNetworkDefault(networkCmd, []string{})
	})
	if err == nil {
		t.Fatal("expected an error when --schema is used without an index")
	}
	if !strings.Contains(out, "requires an entry index") {
		t.Errorf("error should direct the user to supply an index:\n%s", out)
	}
}

func TestRunNetwork_NonIntegerArgIsUnknownCommand(t *testing.T) {
	enableJSONOutput(t)
	restore := setMockFactory(&mockFactory{daemonRunning: true})
	defer restore()

	var err error
	out := captureStream(t, &os.Stderr, func() {
		err = runNetworkDefault(networkCmd, []string{"bogus"})
	})
	if err == nil {
		t.Fatal("expected an error for a non-integer positional argument")
	}
	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("parse error envelope: %v\n%s", err, out)
	}
	msg, _ := resp["error"].(string)
	if !strings.Contains(msg, `unknown command "bogus"`) {
		t.Errorf("a non-integer argument should keep the unknown-command error, got %q", msg)
	}
}

func TestRunNetwork_DrilldownHitReturnsSingleEntry(t *testing.T) {
	enableJSONOutput(t)
	data := ipc.NetworkData{
		Entries: []ipc.NetworkEntry{
			{Seq: 5, RequestID: "5", URL: "https://x/a", Method: "GET", Status: 200},
			{Seq: 9, RequestID: "9", URL: "https://x/b", Method: "POST", Status: 201},
		},
		Count: 2,
	}
	raw, _ := json.Marshal(data)
	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor: &mockExecutor{executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: true, Data: raw}, nil
		}},
	})
	defer restore()

	out := captureStream(t, &os.Stdout, func() {
		if err := runNetworkDefault(networkCmd, []string{"9"}); err != nil {
			t.Fatalf("drill-down: %v", err)
		}
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("parse drill-down envelope: %v\n%s", err, out)
	}
	if resp["count"] != float64(1) {
		t.Errorf("drill-down count = %v, want 1", resp["count"])
	}
	entries, ok := resp["entries"].([]any)
	if !ok || len(entries) != 1 {
		t.Fatalf("expected exactly one entry, got %v", resp["entries"])
	}
	entry := entries[0].(map[string]any)
	if entry["seq"] != float64(9) {
		t.Errorf("drill-down returned seq %v, want 9", entry["seq"])
	}
}

func TestRunNetwork_DrilldownMissNamesBounds(t *testing.T) {
	enableJSONOutput(t)
	// Held seqs 5 and 9; 7 falls between them but is not held, so the lookup must
	// miss and the error must name the held bounds.
	data := ipc.NetworkData{
		Entries: []ipc.NetworkEntry{
			{Seq: 5, RequestID: "5", URL: "https://x/a", Method: "GET", Status: 200},
			{Seq: 9, RequestID: "9", URL: "https://x/b", Method: "GET", Status: 200},
		},
		Count: 2,
	}
	raw, _ := json.Marshal(data)
	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor: &mockExecutor{executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: true, Data: raw}, nil
		}},
	})
	defer restore()

	var err error
	out := captureStream(t, &os.Stderr, func() {
		err = runNetworkDefault(networkCmd, []string{"7"})
	})
	if err == nil {
		t.Fatal("expected an error drilling into an unheld seq")
	}
	if !strings.Contains(out, "entry 7 not in buffer") || !strings.Contains(out, "holds seq 5-9") {
		t.Errorf("miss error should name the held bounds:\n%s", out)
	}
}

func TestRunNetwork_InvalidDetailRejectedBeforeDaemon(t *testing.T) {
	enableJSONOutput(t)
	setNetworkFlag(t, "detail", "verbose")
	// --detail is validated up front, before the daemon check and regardless of
	// output mode, so a malformed value is a deterministic usage error. Daemon is
	// down and JSON is on to prove neither shortcut hides the invalid value.
	restore := setMockFactory(&mockFactory{daemonRunning: false})
	defer restore()

	var err error
	out := captureStream(t, &os.Stderr, func() {
		err = runNetworkDefault(networkCmd, []string{})
	})
	if err == nil {
		t.Fatal("expected an error for an invalid --detail value")
	}
	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("parse error envelope: %v\n%s", err, out)
	}
	msg, _ := resp["error"].(string)
	if !strings.Contains(msg, `invalid --detail "verbose"`) {
		t.Errorf("error should reject the malformed --detail value, not report the daemon:\n%s", msg)
	}
}

func TestRunNetwork_SchemaMissNamesBounds(t *testing.T) {
	enableJSONOutput(t)
	setNetworkFlag(t, "schema", "true")
	// --schema resolves its index through the same exact-membership lookup as
	// drill-down, so a miss returns the same eviction-aware error, not an empty
	// schema. Held seqs 5 and 9; 7 falls between them but is not held.
	data := ipc.NetworkData{
		Entries: []ipc.NetworkEntry{
			{Seq: 5, RequestID: "5", URL: "https://x/a", Method: "GET", Status: 200},
			{Seq: 9, RequestID: "9", URL: "https://x/b", Method: "GET", Status: 200},
		},
		Count: 2,
	}
	raw, _ := json.Marshal(data)
	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor: &mockExecutor{executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: true, Data: raw}, nil
		}},
	})
	defer restore()

	var err error
	out := captureStream(t, &os.Stderr, func() {
		err = runNetworkDefault(networkCmd, []string{"7"})
	})
	if err == nil {
		t.Fatal("expected an error for --schema on an unheld seq")
	}
	if !strings.Contains(out, "entry 7 not in buffer") || !strings.Contains(out, "holds seq 5-9") {
		t.Errorf("schema miss should return the drill-down bounds error:\n%s", out)
	}
	if strings.Contains(out, "schema") {
		t.Errorf("a miss must not emit a schema envelope:\n%s", out)
	}
}
