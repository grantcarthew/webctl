package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/grantcarthew/webctl/internal/ipc"
)

// sparseConsoleEntries builds a set whose held seqs are non-contiguous, mirroring
// an active session interleaved with background traffic: 318, 320, 421, 425.
func sparseConsoleEntries() []ipc.ConsoleEntry {
	return []ipc.ConsoleEntry{
		{Seq: 318, Type: "log", Text: "a"},
		{Seq: 320, Type: "log", Text: "b"},
		{Seq: 421, Type: "error", Text: "c"},
		{Seq: 425, Type: "log", Text: "d"},
	}
}

func TestFindConsoleEntryBySeq(t *testing.T) {
	entries := sparseConsoleEntries()

	// A held seq resolves by exact membership.
	if e, ok := findConsoleEntryBySeq(entries, 421); !ok || e.Text != "c" {
		t.Errorf("seq 421 should resolve to entry c, got ok=%v entry=%+v", ok, e)
	}

	// A seq between the lowest and highest but not held must miss: membership, not
	// range.
	if _, ok := findConsoleEntryBySeq(entries, 400); ok {
		t.Error("seq 400 is between bounds but not held; must miss")
	}

	// A negative index never matches.
	if _, ok := findConsoleEntryBySeq(entries, -1); ok {
		t.Error("negative index must miss")
	}
}

func TestConsoleSeqBounds(t *testing.T) {
	lo, hi, ok := consoleSeqBounds(sparseConsoleEntries())
	if !ok || lo != 318 || hi != 425 {
		t.Errorf("bounds = (%d, %d, %v), want (318, 425, true)", lo, hi, ok)
	}

	if _, _, ok := consoleSeqBounds(nil); ok {
		t.Error("empty set should report no bounds")
	}
}

func TestConsoleDrilldownMissMessage(t *testing.T) {
	// Sparse held set: a between-bounds miss names the bounds as orientation.
	got := consoleDrilldownMissMessage(400, sparseConsoleEntries())
	want := "entry 400 not in buffer (holds seq 318-425; run console to list)"
	if got != want {
		t.Errorf("miss message = %q, want %q", got, want)
	}

	// Empty buffer: there is no bound to name.
	got = consoleDrilldownMissMessage(42, nil)
	want = "entry 42 not in buffer (buffer empty)"
	if got != want {
		t.Errorf("empty-buffer message = %q, want %q", got, want)
	}
}

// mockConsoleDaemon wires a mock factory returning the given entries for the
// console IPC command.
func mockConsoleDaemon(t *testing.T, entries []ipc.ConsoleEntry) {
	t.Helper()
	raw, _ := json.Marshal(ipc.ConsoleData{Entries: entries, Count: len(entries)})
	restore := setMockFactory(&mockFactory{
		daemonRunning: true,
		executor: &mockExecutor{executeFunc: func(req ipc.Request) (ipc.Response, error) {
			return ipc.Response{OK: true, Data: raw}, nil
		}},
	})
	t.Cleanup(restore)
}

func TestRunConsole_DrilldownJSONSingleEntry(t *testing.T) {
	enableJSONOutput(t)
	mockConsoleDaemon(t, []ipc.ConsoleEntry{
		{Seq: 5, Type: "log", Text: "first"},
		{Seq: 9, Type: "error", Text: "second"},
	})

	out := captureStream(t, &os.Stdout, func() {
		if err := runConsoleDefault(consoleCmd, []string{"9"}); err != nil {
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

func TestRunConsole_DrilldownIgnoresFilters(t *testing.T) {
	enableJSONOutput(t)
	// A --type filter that would exclude the target must not hide it on drill-down:
	// the lookup is an identity resolution over the unfiltered set.
	mockConsoleDaemon(t, []ipc.ConsoleEntry{
		{Seq: 5, Type: "log", Text: "kept"},
		{Seq: 9, Type: "error", Text: "target"},
	})
	// StringSlice flags must be Replace'd (Set appends and cannot clear).
	setConsoleStringSlice("type", []string{"log"})
	t.Cleanup(func() { setConsoleStringSlice("type", nil) })

	out := captureStream(t, &os.Stdout, func() {
		if err := runConsoleDefault(consoleCmd, []string{"9"}); err != nil {
			t.Fatalf("drill-down: %v", err)
		}
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("parse envelope: %v\n%s", err, out)
	}
	entries, ok := resp["entries"].([]any)
	if !ok || len(entries) != 1 {
		t.Fatalf("drill-down should resolve the entry despite --type; got %v", resp["entries"])
	}
	if entries[0].(map[string]any)["seq"] != float64(9) {
		t.Errorf("expected seq 9 despite a narrowing --type filter, got %v", entries[0])
	}
}

func TestRunConsole_DrilldownMissNamesBounds(t *testing.T) {
	enableJSONOutput(t)
	// Held seqs 5 and 9; 7 falls between them but is not held, so the lookup must
	// miss and the error must name the held bounds.
	mockConsoleDaemon(t, []ipc.ConsoleEntry{
		{Seq: 5, Type: "log", Text: "a"},
		{Seq: 9, Type: "log", Text: "b"},
	})

	var err error
	out := captureStream(t, &os.Stderr, func() {
		err = runConsoleDefault(consoleCmd, []string{"7"})
	})
	if err == nil {
		t.Fatal("expected an error drilling into an unheld seq")
	}
	if !strings.Contains(out, "entry 7 not in buffer") || !strings.Contains(out, "holds seq 5-9") {
		t.Errorf("miss error should name the held bounds:\n%s", out)
	}
}

func TestRunConsole_DrilldownTextRendersFull(t *testing.T) {
	oldJSON := JSONOutput
	JSONOutput = false
	t.Cleanup(func() { JSONOutput = oldJSON })

	mockConsoleDaemon(t, []ipc.ConsoleEntry{
		{
			Seq: 9, Type: "error", Text: "TypeError: boom", Timestamp: 1609459200000,
			Stack: []ipc.ConsoleFrame{
				{Function: "foo", URL: "app.js", Line: 42, Column: 10},
			},
			ExceptionClass: "TypeError", ExceptionSubtype: "error",
		},
	})

	out := captureStream(t, &os.Stdout, func() {
		if err := runConsoleDefault(consoleCmd, []string{"9"}); err != nil {
			t.Fatalf("drill-down: %v", err)
		}
	})

	for _, want := range []string{"09 [", "stack:", "foo app.js:42:10", "exception: TypeError (error)"} {
		if !strings.Contains(out, want) {
			t.Errorf("drill-down text missing %q:\n%s", want, out)
		}
	}
}

func TestRunConsole_EmptyRangeIsExitZero(t *testing.T) {
	enableJSONOutput(t)
	// A seq range that holds nothing is a routine empty list with exit 0, not a
	// notice: sparse membership makes an empty range a normal result.
	mockConsoleDaemon(t, sparseConsoleEntries())
	_ = consoleCmd.PersistentFlags().Set("range", "321-420")
	t.Cleanup(func() { _ = consoleCmd.PersistentFlags().Set("range", "") })

	var err error
	out := captureStream(t, &os.Stdout, func() {
		err = runConsoleDefault(consoleCmd, nil)
	})
	if err != nil {
		t.Fatalf("empty range should not error: %v", err)
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("parse envelope: %v\n%s", err, out)
	}
	if resp["count"] != float64(0) {
		t.Errorf("empty range count = %v, want 0", resp["count"])
	}
	// Empty list, not JSON null: agents that require an array must not see null.
	entries, ok := resp["entries"].([]any)
	if !ok {
		t.Fatalf("empty range entries should be an array, got %T (%v)", resp["entries"], resp["entries"])
	}
	if len(entries) != 0 {
		t.Errorf("empty range entries length = %d, want 0", len(entries))
	}
}

func TestRunConsole_EmptyTypeFilterIsArrayNotNull(t *testing.T) {
	enableJSONOutput(t)
	// A type filter that matches nothing used to leave a nil slice, which JSON
	// encodes as "entries":null. Agents that always expect an array must get [].
	mockConsoleDaemon(t, sparseConsoleEntries())
	setConsoleStringSlice("type", []string{"nosuch"})
	t.Cleanup(func() { setConsoleStringSlice("type", nil) })

	var err error
	out := captureStream(t, &os.Stdout, func() {
		err = runConsoleDefault(consoleCmd, nil)
	})
	if err != nil {
		t.Fatalf("empty type filter should not error: %v", err)
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("parse envelope: %v\n%s", err, out)
	}
	if resp["count"] != float64(0) {
		t.Errorf("empty type filter count = %v, want 0", resp["count"])
	}
	entries, ok := resp["entries"].([]any)
	if !ok {
		t.Fatalf("empty type filter entries should be an array, got %T (%v)", resp["entries"], resp["entries"])
	}
	if len(entries) != 0 {
		t.Errorf("empty type filter entries length = %d, want 0", len(entries))
	}
}

func TestRunConsole_NonIntegerArgIsUnknownCommand(t *testing.T) {
	// A non-integer positional argument is not a drill-down address; it keeps the
	// unknown-command error. The branch returns before touching the daemon, so no
	// mock is needed.
	var err error
	out := captureStream(t, &os.Stderr, func() {
		err = runConsoleDefault(consoleCmd, []string{"bogus"})
	})
	if err == nil {
		t.Fatal("expected an error for a non-integer positional argument")
	}
	if !strings.Contains(out, `unknown command "bogus" for "webctl console"`) {
		t.Errorf("non-integer arg should yield the unknown-command error:\n%s", out)
	}
}

func TestConsoleCmd_AcceptsAtMostOneArg(t *testing.T) {
	// The drill-down address is a single positional token. Zero or one arg is
	// valid; a stray second token is a usage error rather than a silently
	// discarded arg.
	if err := consoleCmd.Args(consoleCmd, nil); err != nil {
		t.Errorf("zero args should be valid: %v", err)
	}
	if err := consoleCmd.Args(consoleCmd, []string{"42"}); err != nil {
		t.Errorf("one arg should be valid: %v", err)
	}
	if err := consoleCmd.Args(consoleCmd, []string{"42", "99"}); err == nil {
		t.Error("two args should be a usage error")
	}
}

func TestExecuteArgs_TypeFilterResetsBetweenCalls(t *testing.T) {
	// ExecuteArgs must clear StringSlice flags after each invocation so a REPL
	// sequence "console --type error" then bare "console" does not keep filtering.
	enableJSONOutput(t)
	mockConsoleDaemon(t, []ipc.ConsoleEntry{
		{Seq: 1, Type: "log", Text: "a"},
		{Seq: 2, Type: "error", Text: "b"},
	})

	out1 := captureStream(t, &os.Stdout, func() {
		ok, err := ExecuteArgs([]string{"console", "--json", "--type", "error"})
		if !ok || err != nil {
			t.Fatalf("first call: ok=%v err=%v", ok, err)
		}
	})
	var r1 map[string]any
	if err := json.Unmarshal([]byte(out1), &r1); err != nil {
		t.Fatalf("parse first output: %v\n%s", err, out1)
	}
	if r1["count"] != float64(1) {
		t.Fatalf("first count = %v, want 1", r1["count"])
	}

	out2 := captureStream(t, &os.Stdout, func() {
		ok, err := ExecuteArgs([]string{"console", "--json"})
		if !ok || err != nil {
			t.Fatalf("second call: ok=%v err=%v", ok, err)
		}
	})
	var r2 map[string]any
	if err := json.Unmarshal([]byte(out2), &r2); err != nil {
		t.Fatalf("parse second output: %v\n%s", err, out2)
	}
	if r2["count"] != float64(2) {
		t.Errorf("second count = %v, want 2 (type filter must not stick)", r2["count"])
	}
}
