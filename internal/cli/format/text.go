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

// OutputOptions controls text formatting behavior.
type OutputOptions struct {
	UseColor    bool // Enable ANSI color codes
	ShowHeaders bool // Render request/response headers (network text mode)
}

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

// Console outputs console entries in text format.
func Console(w io.Writer, entries []ipc.ConsoleEntry, opts OutputOptions) error {
	for _, e := range entries {
		ts := time.UnixMilli(e.Timestamp).Local()
		timestamp := ts.Format("15:04:05")
		level := strings.ToUpper(e.Type)

		// Format: [HH:MM:SS] LEVEL Message
		if opts.UseColor {
			_, _ = fmt.Fprint(w, "[")
			colorFprint(w, color.Faint, timestamp)
			_, _ = fmt.Fprint(w, "] ")

			// Color the level based on type
			switch ipc.NormalizeConsoleType(e.Type) {
			case ipc.ConsoleTypeError:
				colorFprint(w, color.FgRed, level)
			case ipc.ConsoleTypeWarning:
				colorFprint(w, color.FgYellow, level)
			case ipc.ConsoleTypeInfo:
				colorFprint(w, color.FgCyan, level)
			default:
				_, _ = fmt.Fprint(w, level)
			}
			_, _ = fmt.Fprintf(w, " %s\n", e.Text)
		} else {
			_, _ = fmt.Fprintf(w, "[%s] %s %s\n", timestamp, level, e.Text)
		}

		// Source URL and line number indented below
		if e.URL != "" {
			if e.Line > 0 {
				_, _ = fmt.Fprintf(w, "  %s:%d\n", e.URL, e.Line)
			} else {
				_, _ = fmt.Fprintf(w, "  %s\n", e.URL)
			}
		}
	}
	return nil
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
func Network(w io.Writer, entries []ipc.NetworkEntry, opts OutputOptions) error {
	for _, e := range entries {
		// Format duration
		durationMs := int(e.Duration * 1000)

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
			if e.Error != "" {
				_, _ = fmt.Fprintf(w, "  error: %s\n", e.Error)
			}
			// A failed request has no response, but its request side is still
			// captured and diagnostic, so render it as the success path does.
			printNetworkRequestSide(w, e, opts)
			continue
		}

		// Main line: METHOD URL STATUS DURATION [TYPE] [SIZE]
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
		_, _ = fmt.Fprintln(w)

		// Subordinate detail: the request side (headers when --headers, then body)
		// followed by the response side, each part printed only when present.
		// Bodies arrive already bounded to --max-body-size, so they render verbatim
		// and a trailing marker flags any that were cut. A binary response body is
		// filed rather than stored on ResponseBody, so its saved path prints in
		// place of the payload.
		printNetworkRequestSide(w, e, opts)
		if opts.ShowHeaders {
			printNetworkHeaders(w, "response-headers:", e.ResponseHeaders)
		}
		if e.ResponseBody != "" {
			printNetworkBody(w, "response:", e.ResponseBody, e.ResponseBodyTruncated)
		} else if e.ResponseBodyPath != "" {
			_, _ = fmt.Fprintf(w, "  response: [binary saved to %s]\n", e.ResponseBodyPath)
		}
	}
	return nil
}

// printNetworkRequestSide renders the request half of an entry — headers when
// opts.ShowHeaders, then the request body — so the failed and success paths emit
// an identical request section. Each part prints only when present.
func printNetworkRequestSide(w io.Writer, e ipc.NetworkEntry, opts OutputOptions) {
	if opts.ShowHeaders {
		printNetworkHeaders(w, "request-headers:", e.RequestHeaders)
	}
	printNetworkBody(w, "request:", e.RequestBody, e.RequestBodyTruncated)
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
	_, _ = fmt.Fprintf(w, "  %s\n", label)
	for _, k := range keys {
		_, _ = fmt.Fprintf(w, "    %s: %s\n", k, headers[k])
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
		_, _ = fmt.Fprintf(w, "  %s %s\n", label, lines[0])
	} else {
		_, _ = fmt.Fprintf(w, "  %s\n", label)
		for _, line := range lines {
			_, _ = fmt.Fprintf(w, "    %s\n", line)
		}
	}
	if truncated {
		_, _ = fmt.Fprintf(w, "    … [truncated]\n")
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
