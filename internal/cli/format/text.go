package format

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/grantcarthew/webctl/internal/ipc"
	"golang.org/x/term"
)

// Color helper functions that respect color.NoColor flag
func colorFprint(w io.Writer, c color.Attribute, s string) {
	_, _ = color.New(c).Fprint(w, s)
}

func colorFprintf(w io.Writer, c color.Attribute, format string, args ...interface{}) {
	_, _ = color.New(c).Fprintf(w, format, args...)
}

// DetailLevel controls how much of each network entry the text formatter renders.
type DetailLevel int

const (
	// DetailSummary renders only the main line per entry (no transport block, no
	// headers, no bodies). A failed entry still shows its failure reason.
	DetailSummary DetailLevel = iota
	// DetailStandard adds the transport detail block (remote, timing, initiator)
	// and, when ShowHeaders is set, the header blocks. This is the default.
	DetailStandard
	// DetailFull adds the request and response bodies, bounded by --max-body-size.
	DetailFull
)

// OutputOptions controls text formatting behavior.
type OutputOptions struct {
	UseColor    bool        // Enable ANSI color codes
	ShowHeaders bool        // Render request/response headers (network text mode)
	Detail      DetailLevel // Network detail level (summary/standard/full)
}

// Network subordinate-line indentation. Detail lines read as children of their
// seq-prefixed entry; nested lines (header entries, multi-line body content) sit
// one level deeper.
const (
	netIndent  = "       "   // 7 spaces: subordinate detail lines
	netIndent2 = "         " // 9 spaces: nested lines under a detail label
)

// NewOutputOptions returns output options based on flags and environment.
// Priority: jsonOutput > noColorFlag > NO_COLOR env > TTY detection.
func NewOutputOptions(jsonOutput bool, noColorFlag bool) OutputOptions {
	// JSON output never has colors
	if jsonOutput {
		return OutputOptions{UseColor: false}
	}

	// --no-color flag disables colors
	if noColorFlag {
		return OutputOptions{UseColor: false}
	}

	// NO_COLOR environment variable disables colors
	if os.Getenv("NO_COLOR") != "" {
		return OutputOptions{UseColor: false}
	}

	// Enable colors if stdout is a TTY
	return OutputOptions{
		UseColor: term.IsTerminal(int(os.Stdout.Fd())),
	}
}

// DefaultOptions returns default output options based on TTY detection.
// Deprecated: Use NewOutputOptions instead for proper color detection.
func DefaultOptions() OutputOptions {
	return OutputOptions{
		UseColor: term.IsTerminal(int(os.Stdout.Fd())),
	}
}

// ActionSuccess outputs "OK" for successful action commands.
func ActionSuccess(w io.Writer) error {
	_, err := fmt.Fprintln(w, "OK")
	return err
}

// ActionError outputs "Error: <message>" for failed action commands.
func ActionError(w io.Writer, msg string, opts OutputOptions) error {
	if opts.UseColor {
		colorFprint(w, color.FgRed, "Error:")
		_, err := fmt.Fprintf(w, " %s\n", msg)
		return err
	}
	_, err := fmt.Fprintf(w, "Error: %s\n", msg)
	return err
}

// formatHTTPStatus outputs an HTTP status code with appropriate coloring.
// Format: " (status)" - e.g., " (200)", " (404)"
func formatHTTPStatus(w io.Writer, status int, opts OutputOptions) {
	if opts.UseColor {
		_, _ = fmt.Fprint(w, " (")
		switch {
		case status >= 200 && status < 300:
			colorFprintf(w, color.FgGreen, "%d", status)
		case status >= 300 && status < 400:
			colorFprintf(w, color.FgCyan, "%d", status)
		case status >= 400 && status < 500:
			colorFprintf(w, color.FgYellow, "%d", status)
		case status >= 500:
			colorFprintf(w, color.FgRed, "%d", status)
		default:
			_, _ = fmt.Fprintf(w, "%d", status)
		}
		_, _ = fmt.Fprint(w, ")")
	} else {
		_, _ = fmt.Fprintf(w, " (%d)", status)
	}
}

// Status outputs daemon status in text format.
func Status(w io.Writer, data ipc.StatusData, opts OutputOptions) error {
	// Not running state
	if !data.Running {
		if opts.UseColor {
			colorFprint(w, color.FgYellow, "Not running (start with: webctl start)\n")
		} else {
			_, _ = fmt.Fprintln(w, "Not running (start with: webctl start)")
		}
		return nil
	}

	// Running but no browser
	if data.ActiveSession == nil && len(data.Sessions) == 0 {
		if opts.UseColor {
			colorFprint(w, color.FgYellow, "No browser\n")
		} else {
			_, _ = fmt.Fprintln(w, "No browser")
		}
		if data.PID > 0 {
			_, _ = fmt.Fprintf(w, "pid: %d\n", data.PID)
		}
		return nil
	}

	// Running but no active session (browser connected but no pages)
	if data.ActiveSession == nil {
		if opts.UseColor {
			colorFprint(w, color.FgYellow, "No session\n")
		} else {
			_, _ = fmt.Fprintln(w, "No session")
		}
		if data.PID > 0 {
			_, _ = fmt.Fprintf(w, "pid: %d\n", data.PID)
		}
		return nil
	}

	// All systems operational
	if opts.UseColor {
		colorFprint(w, color.FgGreen, "OK\n")
	} else {
		_, _ = fmt.Fprintln(w, "OK")
	}
	if data.PID > 0 {
		_, _ = fmt.Fprintf(w, "pid: %d\n", data.PID)
	}

	// Show sessions
	if len(data.Sessions) > 0 {
		_, _ = fmt.Fprintln(w, "sessions:")
		for _, session := range data.Sessions {
			if session.Active {
				if opts.UseColor {
					_, _ = fmt.Fprint(w, "  ")
					colorFprint(w, color.FgCyan, "* ")
					_, _ = fmt.Fprint(w, session.URL)
				} else {
					_, _ = fmt.Fprintf(w, "  * %s", session.URL)
				}
			} else {
				_, _ = fmt.Fprintf(w, "    %s", session.URL)
			}
			// Append HTTP status if available
			if session.Status > 0 {
				formatHTTPStatus(w, session.Status, opts)
			}
			_, _ = fmt.Fprintln(w)
		}
	}

	return nil
}

// Console renders the indexed console list: one summary line per entry, prefixed
// with the entry's seq (its drill-down address). The line carries the wall-clock
// timestamp, the level, the top stack frame, and the first line of the message.
// The enriched payload (full multi-line text, complete stack, all arguments, and
// exception or Log-domain detail) is reserved for drill-down (ConsoleDetail).
func Console(w io.Writer, entries []ipc.ConsoleEntry, opts OutputOptions) error {
	for _, e := range entries {
		writeConsoleSummaryLine(w, e, opts)
	}
	return nil
}

// ConsoleDetail renders a single console entry in full for drill-down: the
// summary line, then the complete multi-line message, stack, arguments, and any
// exception or Log-domain correlation on seven-space subordinate lines, matching
// the network drill-down layout.
func ConsoleDetail(w io.Writer, e ipc.ConsoleEntry, opts OutputOptions) error {
	writeConsoleSummaryLine(w, e, opts)

	// The summary line already carries the first line of Text; a multi-line
	// message repeats in full here so nothing is lost off the index. Strip
	// trailing newlines first so a terminal newline (common on exception
	// descriptions) does not invent an empty subordinate line; a sole trailing
	// newline then leaves no second line at all and the block stays closed.
	// Internal blank lines are preserved.
	text := strings.TrimRight(e.Text, "\r\n")
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		_, _ = fmt.Fprintf(w, "%smessage:\n", netIndent)
		for _, line := range strings.Split(text, "\n") {
			_, _ = fmt.Fprintf(w, "%s%s\n", netIndent2, strings.TrimRight(line, "\r"))
		}
	}

	printConsoleStack(w, e.Stack)
	printConsoleArgs(w, e.Args)

	if e.ExceptionClass != "" {
		if e.ExceptionSubtype != "" {
			_, _ = fmt.Fprintf(w, "%sexception: %s (%s)\n", netIndent, e.ExceptionClass, e.ExceptionSubtype)
		} else {
			_, _ = fmt.Fprintf(w, "%sexception: %s\n", netIndent, e.ExceptionClass)
		}
	}

	// Log-domain correlation: the entry's origin, and the network request id that
	// ties a network-source log to its network buffer entry.
	if e.Source != "" {
		_, _ = fmt.Fprintf(w, "%ssource: %s\n", netIndent, e.Source)
	}
	if e.NetworkRequestID != "" {
		_, _ = fmt.Fprintf(w, "%snetwork-request: %s\n", netIndent, e.NetworkRequestID)
	}
	if e.WorkerID != "" {
		_, _ = fmt.Fprintf(w, "%sworker: %s\n", netIndent, e.WorkerID)
	}
	return nil
}

// writeConsoleSummaryLine writes the one-line index entry shared by the list and
// the drill-down header: "SEQ [HH:MM:SS] LEVEL frame message", where frame is the
// top stack locator and message is the first line of Text. Absent components are
// omitted rather than padded.
func writeConsoleSummaryLine(w io.Writer, e ipc.ConsoleEntry, opts OutputOptions) {
	ts := time.UnixMilli(e.Timestamp).Local().Format("15:04:05")
	level := strings.ToUpper(e.Type)
	frame := consoleTopFrame(e)
	msg := firstLine(e.Text)

	// Seq prefix, zero-padded to a minimum of two digits and growing naturally
	// beyond, with no surrounding brackets, so it matches the drill-down integer.
	_, _ = fmt.Fprintf(w, "%02d ", e.Seq)

	if opts.UseColor {
		_, _ = fmt.Fprint(w, "[")
		colorFprint(w, color.Faint, ts)
		_, _ = fmt.Fprint(w, "] ")
		printConsoleLevel(w, e.Type, level)
	} else {
		_, _ = fmt.Fprintf(w, "[%s] %s", ts, level)
	}

	if frame != "" {
		_, _ = fmt.Fprintf(w, " %s", frame)
	}
	if msg != "" {
		_, _ = fmt.Fprintf(w, " %s", msg)
	}
	_, _ = fmt.Fprintln(w)
}

// printConsoleLevel writes the severity level, colourised by type on a TTY.
func printConsoleLevel(w io.Writer, rawType, level string) {
	switch ipc.NormalizeConsoleType(rawType) {
	case ipc.ConsoleTypeError:
		colorFprint(w, color.FgRed, level)
	case ipc.ConsoleTypeWarning:
		colorFprint(w, color.FgYellow, level)
	case ipc.ConsoleTypeInfo:
		colorFprint(w, color.FgCyan, level)
	default:
		_, _ = fmt.Fprint(w, level)
	}
}

// consoleTopFrame returns the summary locator for an entry: the top stack frame's
// function name and location, falling back to the convenience url:line:column
// when no stack was captured. Empty when the entry carries no location at all
// (exceptions and Log-domain entries frequently do).
func consoleTopFrame(e ipc.ConsoleEntry) string {
	function := ""
	url, line, column := e.URL, e.Line, e.Column
	if len(e.Stack) > 0 {
		top := e.Stack[0]
		function, url, line, column = top.Function, top.URL, top.Line, top.Column
	}
	loc := consoleLocation(url, line, column)
	switch {
	case function != "" && loc != "":
		return function + " " + loc
	case function != "":
		return function
	default:
		return loc
	}
}

// consoleLocation formats a url:line:column locator. CDP lines and columns are
// both 0-based, so 0 is a real first-line / first-column position and must render
// rather than being treated as "absent". Returns empty when no URL was captured;
// never returns a bare URL without line and column.
func consoleLocation(url string, line, column int) string {
	if url == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d:%d", url, line, column)
}

// printConsoleStack renders the full call stack, one frame per line as function
// then location, at nine-space indent under a "stack:" label. An asynchronous
// continuation boundary prints its parent group description on its own line so
// the sync/async split stays legible. Nothing prints for an empty stack.
func printConsoleStack(w io.Writer, stack []ipc.ConsoleFrame) {
	if len(stack) == 0 {
		return
	}
	_, _ = fmt.Fprintf(w, "%sstack:\n", netIndent)
	for _, f := range stack {
		if f.Async != "" {
			_, _ = fmt.Fprintf(w, "%sasync %s\n", netIndent2, f.Async)
		}
		function := f.Function
		if function == "" {
			function = "<anonymous>"
		}
		if loc := consoleLocation(f.URL, f.Line, f.Column); loc != "" {
			_, _ = fmt.Fprintf(w, "%s%s %s\n", netIndent2, function, loc)
		} else {
			_, _ = fmt.Fprintf(w, "%s%s\n", netIndent2, function)
		}
	}
}

// printConsoleArgs renders every captured console argument at nine-space indent
// under an "args:" label. A primitive shows its type and verbatim value; a
// non-primitive shows its type, description, and shallow property preview.
// Nothing prints when no arguments were captured (exceptions and Log-domain
// entries carry none).
func printConsoleArgs(w io.Writer, args []ipc.ConsoleArg) {
	if len(args) == 0 {
		return
	}
	_, _ = fmt.Fprintf(w, "%sargs:\n", netIndent)
	for i, a := range args {
		_, _ = fmt.Fprintf(w, "%s[%d] %s\n", netIndent2, i, formatConsoleArg(a))
	}
}

// formatConsoleArg renders one argument as a single line. A primitive is its type
// followed by the verbatim JSON value; a non-primitive is its type (refined by
// subtype), its description, and a brace-wrapped preview of its shallow
// properties when one was captured.
func formatConsoleArg(a ipc.ConsoleArg) string {
	typ := a.Type
	if a.Subtype != "" {
		typ = a.Type + "/" + a.Subtype
	}

	if len(a.Value) > 0 {
		return strings.TrimSpace(fmt.Sprintf("%s %s", typ, string(a.Value)))
	}

	parts := []string{typ}
	if a.Description != "" {
		parts = append(parts, a.Description)
	}
	if len(a.Preview) > 0 {
		props := make([]string, 0, len(a.Preview))
		for _, p := range a.Preview {
			if p.Value != "" {
				props = append(props, fmt.Sprintf("%s: %s", p.Name, p.Value))
			} else {
				props = append(props, p.Name)
			}
		}
		parts = append(parts, "{ "+strings.Join(props, ", ")+" }")
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// firstLine returns the first physical line of s with any trailing carriage
// return stripped, so a multi-line message contributes only one line to the
// index.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimRight(s, "\r")
}

// formatBytes renders a byte count in human-readable form (B, KB, MB, GB, ...)
// on a 1024 base, with one decimal place above bytes.
func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(n)/float64(div), "KMGTPE"[exp])
}

// printNetworkMethod writes the HTTP method, colourised by method on a TTY.
func printNetworkMethod(w io.Writer, method string, opts OutputOptions) {
	if !opts.UseColor {
		_, _ = fmt.Fprint(w, method)
		return
	}
	switch method {
	case "GET":
		colorFprint(w, color.FgGreen, method)
	case "POST":
		colorFprint(w, color.FgBlue, method)
	case "PUT", "PATCH":
		colorFprint(w, color.FgYellow, method)
	case "DELETE":
		colorFprint(w, color.FgRed, method)
	default:
		_, _ = fmt.Fprint(w, method)
	}
}

// printNetworkStatus writes the HTTP status code, colourised by category on a TTY.
func printNetworkStatus(w io.Writer, status int, opts OutputOptions) {
	if !opts.UseColor {
		_, _ = fmt.Fprintf(w, "%d", status)
		return
	}
	switch {
	case status >= 200 && status < 300:
		colorFprintf(w, color.FgGreen, "%d", status)
	case status >= 300 && status < 400:
		colorFprintf(w, color.FgCyan, "%d", status)
	case status >= 400 && status < 500:
		colorFprintf(w, color.FgYellow, "%d", status)
	case status >= 500:
		colorFprintf(w, color.FgRed, "%d", status)
	default:
		_, _ = fmt.Fprintf(w, "%d", status)
	}
}

// Network outputs network entries in text format.
//
// Text is a curated human convenience; JSON carries the full NetworkEntry. The
// captured fields are classified for the text view as follows:
//   - Method, URL, Status, Duration: shown on the main line (the core request line).
//   - Failed/Error: shown — a failed entry renders a FAILED token plus its reason.
//   - Type: shown — short resource category (xhr, document, image, ...).
//   - Size: shown when > 0 — human-readable response size.
//   - MimeType: omitted — overlaps Type and the printed body; the exact
//     Content-Type stays in JSON and response headers.
//   - StatusText: omitted — redundant with the numeric status, which is already
//     shown and colourised; a failure's reason comes from Error, not StatusText.
//   - RequestHeaders/ResponseHeaders: conditional — high-volume, so kept out of
//     the default view and shown only when opts.ShowHeaders (the --headers flag),
//     letting an agent get headers in compact text form without the full JSON.
//   - RemoteIPAddress/RemotePort/Protocol/ConnectionID: shown on a subordinate
//     "remote:" line when captured — the endpoint actually contacted, the
//     negotiated wire protocol, and the connection id (conn:N) that exposes
//     HTTP/2 multiplexing and keep-alive reuse across entries.
//   - SecurityState: shown on the same "remote:" line only when not "secure",
//     so a non-secure posture (insecure, neutral, unknown) stands out as a
//     signal while the common secure case stays silent.
//   - Timing: shown on a subordinate "timing:" line as integer-millisecond
//     phase durations (dns, connect, tls, send, wait) so a slow request reveals
//     where the time went. Phases under half a millisecond are dropped.
//   - Initiator: shown on a subordinate "initiator:" line as "type url:line"
//     when a location was captured (parser and script initiators), naming what
//     triggered the request. The bare "other" initiator is omitted as noise.
//   - FromDiskCache/FromServiceWorker/FromPrefetchCache: shown as a single
//     self-describing main-line token (disk, service-worker, prefetch) naming
//     which cache served the response. The origins are mutually exclusive.
func Network(w io.Writer, entries []ipc.NetworkEntry, opts OutputOptions) error {
	for _, e := range entries {
		// Format duration
		durationMs := int(e.Duration * 1000)

		// Each entry line is prefixed with its own seq (the drill-down address),
		// zero-padded to a minimum of two digits and growing naturally beyond.
		_, _ = fmt.Fprintf(w, "%02d ", e.Seq)

		// A failed request (loadingFailed) carries no status, so render a distinct
		// FAILED token plus the captured reason instead of a bare status of 0. The
		// branch keys on Failed, not status == 0, so a genuine zero-status success
		// is never mistaken for a failure.
		if e.Failed {
			printNetworkMethod(w, e.Method, opts)
			_, _ = fmt.Fprintf(w, " %s ", e.URL)
			if opts.UseColor {
				colorFprint(w, color.FgRed, "FAILED")
			} else {
				_, _ = fmt.Fprint(w, "FAILED")
			}
			_, _ = fmt.Fprintf(w, " %dms", durationMs)
			if e.Type != "" {
				_, _ = fmt.Fprintf(w, " %s", e.Type)
			}
			_, _ = fmt.Fprintln(w)
			// The failure reason is the point of a failed entry, so it renders at
			// every detail level, before the transport/initiator metadata.
			if e.Error != "" {
				_, _ = fmt.Fprintf(w, "%serror: %s\n", netIndent, e.Error)
			}
			if opts.Detail >= DetailStandard {
				printNetworkRemote(w, e)
				printNetworkTiming(w, e)
				printNetworkInitiator(w, e)
				if opts.ShowHeaders {
					printNetworkHeaders(w, "request-headers:", e.RequestHeaders)
				}
			}
			// A failed request has no response, but its request body is still
			// captured and diagnostic, so it renders with the bodies at full.
			if opts.Detail >= DetailFull {
				printNetworkBody(w, "request:", e.RequestBody, e.RequestBodyTruncated)
			}
			continue
		}

		// Main line: METHOD URL STATUS DURATION [TYPE] [SIZE] [(CACHE)]
		printNetworkMethod(w, e.Method, opts)
		_, _ = fmt.Fprintf(w, " %s ", e.URL)
		printNetworkStatus(w, e.Status, opts)
		_, _ = fmt.Fprintf(w, " %dms", durationMs)
		if e.Type != "" {
			_, _ = fmt.Fprintf(w, " %s", e.Type)
		}
		if e.Size > 0 {
			_, _ = fmt.Fprintf(w, " %s", formatBytes(e.Size))
		}
		if tok := networkCacheToken(e); tok != "" {
			_, _ = fmt.Fprintf(w, " (%s)", tok)
		}
		_, _ = fmt.Fprintln(w)

		// Transport detail block: shown at standard and full, hidden at summary.
		if opts.Detail >= DetailStandard {
			printNetworkRemote(w, e)
			printNetworkTiming(w, e)
			printNetworkInitiator(w, e)
			if opts.ShowHeaders {
				printNetworkHeaders(w, "request-headers:", e.RequestHeaders)
			}
		}

		// Bodies: shown only at full. They arrive already bounded to
		// --max-body-size, so they render verbatim and a trailing marker flags any
		// that were cut. A binary response body is filed rather than stored on
		// ResponseBody, so its saved path prints in place of the payload.
		if opts.Detail >= DetailFull {
			printNetworkBody(w, "request:", e.RequestBody, e.RequestBodyTruncated)
		}
		if opts.Detail >= DetailStandard && opts.ShowHeaders {
			printNetworkHeaders(w, "response-headers:", e.ResponseHeaders)
		}
		if opts.Detail >= DetailFull {
			if e.ResponseBody != "" {
				printNetworkBody(w, "response:", e.ResponseBody, e.ResponseBodyTruncated)
			} else if e.ResponseBodyPath != "" {
				_, _ = fmt.Fprintf(w, "%sresponse: [binary saved to %s]\n", netIndent, e.ResponseBodyPath)
			}
		}
	}
	return nil
}

// networkCacheToken returns the self-describing cache-origin token for an entry,
// or "" when the response came over the network. The origins are mutually
// exclusive; the token names which cache answered so a human need not drop to
// JSON to tell a stale disk cache from a service-worker interception.
func networkCacheToken(e ipc.NetworkEntry) string {
	switch {
	case e.FromServiceWorker:
		return "service-worker"
	case e.FromDiskCache:
		return "disk"
	case e.FromPrefetchCache:
		return "prefetch"
	default:
		return ""
	}
}

// printNetworkRemote renders the transport line for an entry: the remote
// endpoint and the negotiated protocol. It prints only when the protocol or
// address was captured, so request-only and failed entries (which carry no
// response) stay quiet.
func printNetworkRemote(w io.Writer, e ipc.NetworkEntry) {
	if e.Protocol == "" && e.RemoteIPAddress == "" {
		return
	}
	parts := make([]string, 0, 4)
	if e.RemoteIPAddress != "" {
		if e.RemotePort > 0 {
			parts = append(parts, fmt.Sprintf("%s:%d", e.RemoteIPAddress, e.RemotePort))
		} else {
			parts = append(parts, e.RemoteIPAddress)
		}
	}
	if e.Protocol != "" {
		parts = append(parts, e.Protocol)
	}
	if e.ConnectionID > 0 {
		parts = append(parts, fmt.Sprintf("conn:%d", int64(e.ConnectionID)))
	}
	// Only surface a non-secure posture; "secure" is the norm on HTTPS and would
	// be noise on nearly every row, so its absence here means the request is fine.
	if e.SecurityState != "" && e.SecurityState != "secure" {
		parts = append(parts, e.SecurityState)
	}
	_, _ = fmt.Fprintf(w, "%sremote: %s\n", netIndent, strings.Join(parts, " "))
}

// printNetworkTiming renders the per-phase latency line for an entry. Phases are
// listed in request order and shown as integer milliseconds; a phase under half
// a millisecond rounds to zero and is dropped, so negligible phases (usually the
// request send) fall away and the line carries only meaningful time. Nothing
// prints when no timing was captured or every phase is negligible.
func printNetworkTiming(w io.Writer, e ipc.NetworkEntry) {
	if e.Timing == nil {
		return
	}
	phases := []struct {
		name string
		ms   float64
	}{
		{"dns", e.Timing.DNSMs},
		{"connect", e.Timing.ConnectMs},
		{"tls", e.Timing.TLSMs},
		{"send", e.Timing.SendMs},
		{"wait", e.Timing.WaitMs},
	}
	parts := make([]string, 0, len(phases))
	for _, p := range phases {
		if rounded := int(p.ms + 0.5); rounded >= 1 {
			parts = append(parts, fmt.Sprintf("%s %dms", p.name, rounded))
		}
	}
	if len(parts) == 0 {
		return
	}
	_, _ = fmt.Fprintf(w, "%stiming: %s\n", netIndent, strings.Join(parts, " "))
}

// printNetworkInitiator renders what triggered the request as a subordinate
// "initiator:" line: the initiator type and the source location that issued it,
// in the same url:line form the console formatter uses. It prints only when a
// location was captured (the parser and script cases) and stays silent for the
// bare "other" initiator, which carries no location and would only add noise.
func printNetworkInitiator(w io.Writer, e ipc.NetworkEntry) {
	if e.Initiator == nil || e.Initiator.URL == "" {
		return
	}
	// Mirror the console formatter: append the line only when it is meaningful,
	// since CDP line numbers are 0-based and line 0 would render a bare ":0".
	if e.Initiator.Line > 0 {
		_, _ = fmt.Fprintf(w, "%sinitiator: %s %s:%d\n", netIndent, e.Initiator.Type, e.Initiator.URL, e.Initiator.Line)
	} else {
		_, _ = fmt.Fprintf(w, "%sinitiator: %s %s\n", netIndent, e.Initiator.Type, e.Initiator.URL)
	}
}

// printNetworkHeaders renders a labeled header map as indented subordinate
// lines, keys sorted for stable output. Nothing prints for an empty map.
func printNetworkHeaders(w io.Writer, label string, headers map[string]string) {
	if len(headers) == 0 {
		return
	}
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	_, _ = fmt.Fprintf(w, "%s%s\n", netIndent, label)
	for _, k := range keys {
		_, _ = fmt.Fprintf(w, "%s%s: %s\n", netIndent2, k, headers[k])
	}
}

// printNetworkBody renders a labeled network body. A single-line body follows
// the label on the same line; a multi-line body prints the bare label line then
// each body line indented four spaces. When truncated is set, a trailing marker
// line signals that --max-body-size cut the body, so a text reader is not misled
// by a payload that silently ends. Nothing prints for an empty body.
func printNetworkBody(w io.Writer, label, body string, truncated bool) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	lines := strings.Split(body, "\n")
	if len(lines) == 1 {
		_, _ = fmt.Fprintf(w, "%s%s %s\n", netIndent, label, lines[0])
	} else {
		_, _ = fmt.Fprintf(w, "%s%s\n", netIndent, label)
		for _, line := range lines {
			_, _ = fmt.Fprintf(w, "%s%s\n", netIndent2, line)
		}
	}
	if truncated {
		_, _ = fmt.Fprintf(w, "%s… [truncated]\n", netIndent2)
	}
}

// Cookies outputs cookies in text format (semicolon-separated attributes).
func Cookies(w io.Writer, cookies []ipc.Cookie, opts OutputOptions) error {
	for _, c := range cookies {
		if opts.UseColor {
			// Cookie name in cyan
			colorFprint(w, color.FgCyan, c.Name)
			_, _ = fmt.Fprint(w, "=")
			// Cookie value in default color
			_, _ = fmt.Fprint(w, c.Value)

			// Attributes in dim gray
			if c.Domain != "" {
				_, _ = fmt.Fprint(w, "; ")
				colorFprintf(w, color.Faint, "domain=%s", c.Domain)
			}
			if c.Path != "" {
				_, _ = fmt.Fprint(w, "; ")
				colorFprintf(w, color.Faint, "path=%s", c.Path)
			}
			if c.Secure {
				_, _ = fmt.Fprint(w, "; ")
				colorFprint(w, color.Faint, "secure")
			}
			if c.HTTPOnly {
				_, _ = fmt.Fprint(w, "; ")
				colorFprint(w, color.Faint, "httponly")
			}
			if !c.Session && c.Expires > 0 {
				expiresTime := time.Unix(int64(c.Expires), 0)
				_, _ = fmt.Fprint(w, "; ")
				colorFprintf(w, color.Faint, "expires=%s", expiresTime.Format("2006-01-02"))
			}
			if c.SameSite != "" {
				_, _ = fmt.Fprint(w, "; ")
				colorFprintf(w, color.Faint, "samesite=%s", c.SameSite)
			}
			_, _ = fmt.Fprintln(w)
		} else {
			// No color - original behavior
			parts := []string{
				fmt.Sprintf("%s=%s", c.Name, c.Value),
			}

			if c.Domain != "" {
				parts = append(parts, fmt.Sprintf("domain=%s", c.Domain))
			}
			if c.Path != "" {
				parts = append(parts, fmt.Sprintf("path=%s", c.Path))
			}
			if c.Secure {
				parts = append(parts, "secure")
			}
			if c.HTTPOnly {
				parts = append(parts, "httponly")
			}
			if !c.Session && c.Expires > 0 {
				expiresTime := time.Unix(int64(c.Expires), 0)
				parts = append(parts, fmt.Sprintf("expires=%s", expiresTime.Format("2006-01-02")))
			}
			if c.SameSite != "" {
				parts = append(parts, fmt.Sprintf("samesite=%s", c.SameSite))
			}

			_, _ = fmt.Fprintln(w, strings.Join(parts, "; "))
		}
	}
	return nil
}

// FilePath outputs a file path (for screenshot, html commands).
func FilePath(w io.Writer, path string) error {
	_, err := fmt.Fprintln(w, path)
	return err
}

// EvalResult outputs the raw JavaScript return value.
func EvalResult(w io.Writer, data ipc.EvalData) error {
	if !data.HasValue {
		_, err := fmt.Fprintln(w, "undefined")
		return err
	}

	// Format value based on type
	switch v := data.Value.(type) {
	case nil:
		_, err := fmt.Fprintln(w, "null")
		return err
	case string:
		_, err := fmt.Fprintln(w, v)
		return err
	case map[string]interface{}, []interface{}:
		// JSON objects/arrays - compact format
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(w, string(jsonBytes))
		return err
	default:
		// Numbers, booleans, etc.
		_, err := fmt.Fprintf(w, "%v\n", v)
		return err
	}
}

// Tab outputs the tab list in text format.
func Tab(w io.Writer, data ipc.TabData, opts OutputOptions) error {
	for _, session := range data.Sessions {
		isActive := session.ID == data.ActiveSession

		// Truncate ID to 8 chars
		displayID := session.ID
		if len(displayID) > 8 {
			displayID = displayID[:8]
		}

		// Truncate title to 40 chars
		title := strings.TrimSpace(session.Title)
		if len(title) > 40 {
			title = title[:37] + "..."
		}

		if opts.UseColor {
			if isActive {
				colorFprint(w, color.FgCyan, "* ")
			} else {
				_, _ = fmt.Fprint(w, "  ")
			}
			_, _ = fmt.Fprintf(w, "%s - %s [", session.URL, title)
			colorFprint(w, color.FgCyan, displayID)
			_, _ = fmt.Fprintln(w, "]")
		} else {
			prefix := "  "
			if isActive {
				prefix = "* "
			}
			_, _ = fmt.Fprintf(w, "%s%s - %s [%s]\n", prefix, session.URL, title, displayID)
		}
	}
	return nil
}

// TabError outputs a tab error with session/match information.
func TabError(w io.Writer, errorMsg string, sessions []ipc.PageSession, matches []ipc.PageSession, opts OutputOptions) error {
	if opts.UseColor {
		colorFprint(w, color.FgRed, "Error:")
		_, _ = fmt.Fprintf(w, " %s\n", errorMsg)
	} else {
		_, _ = fmt.Fprintf(w, "Error: %s\n", errorMsg)
	}

	if len(sessions) > 0 {
		_, _ = fmt.Fprintln(w, "Available tabs:")
		for _, session := range sessions {
			displayID := session.ID
			if len(displayID) > 8 {
				displayID = displayID[:8]
			}
			_, _ = fmt.Fprintf(w, "  %s - %s\n", displayID, session.Title)
		}
	}

	if len(matches) > 0 {
		_, _ = fmt.Fprintln(w, "Matching tabs:")
		for _, session := range matches {
			displayID := session.ID
			if len(displayID) > 8 {
				displayID = displayID[:8]
			}
			_, _ = fmt.Fprintf(w, "  %s - %s\n", displayID, session.Title)
		}
	}

	return nil
}

// Find outputs find results in text format with colored highlighting.
