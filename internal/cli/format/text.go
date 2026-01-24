package format

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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
	UseColor bool // Enable ANSI color codes
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

// Network outputs network entries in text format.
func Network(w io.Writer, entries []ipc.NetworkEntry, opts OutputOptions) error {
	for _, e := range entries {
		// Format duration
		durationMs := int(e.Duration * 1000)

		// Main line: METHOD URL STATUS DURATION
		if opts.UseColor {
			// Color the HTTP method
			switch e.Method {
			case "GET":
				colorFprint(w, color.FgGreen, e.Method)
			case "POST":
				colorFprint(w, color.FgBlue, e.Method)
			case "PUT", "PATCH":
				colorFprint(w, color.FgYellow, e.Method)
			case "DELETE":
				colorFprint(w, color.FgRed, e.Method)
			default:
				_, _ = fmt.Fprint(w, e.Method)
			}

			_, _ = fmt.Fprintf(w, " %s ", e.URL)

			// Color the status code by category
			if e.Status >= 200 && e.Status < 300 {
				colorFprintf(w, color.FgGreen, "%d", e.Status)
			} else if e.Status >= 300 && e.Status < 400 {
				colorFprintf(w, color.FgCyan, "%d", e.Status)
			} else if e.Status >= 400 && e.Status < 500 {
				colorFprintf(w, color.FgYellow, "%d", e.Status)
			} else if e.Status >= 500 {
				colorFprintf(w, color.FgRed, "%d", e.Status)
			} else {
				_, _ = fmt.Fprintf(w, "%d", e.Status)
			}

			_, _ = fmt.Fprintf(w, " %dms\n", durationMs)
		} else {
			_, _ = fmt.Fprintf(w, "%s %s %d %dms\n", e.Method, e.URL, e.Status, durationMs)
		}

		// Request body (if present and non-empty)
		if e.Body != "" && e.Method != "GET" {
			// Try to parse as JSON to detect if it's request/response
			// For now, just show bodies indented
			bodyLines := strings.Split(strings.TrimSpace(e.Body), "\n")
			for _, line := range bodyLines {
				_, _ = fmt.Fprintf(w, "  %s\n", line)
			}
		}
	}
	return nil
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

// Target outputs page sessions list in text format.
func Target(w io.Writer, data ipc.TargetData, opts OutputOptions) error {
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

// TargetError outputs target error with session/match information.
func TargetError(w io.Writer, errorMsg string, sessions []ipc.PageSession, matches []ipc.PageSession, opts OutputOptions) error {
	if opts.UseColor {
		colorFprint(w, color.FgRed, "Error:")
		_, _ = fmt.Fprintf(w, " %s\n", errorMsg)
	} else {
		_, _ = fmt.Fprintf(w, "Error: %s\n", errorMsg)
	}

	if len(sessions) > 0 {
		_, _ = fmt.Fprintln(w, "Available sessions:")
		for _, session := range sessions {
			// Truncate ID to 8 chars
			displayID := session.ID
			if len(displayID) > 8 {
				displayID = displayID[:8]
			}
			_, _ = fmt.Fprintf(w, "  %s - %s\n", displayID, session.Title)
		}
	}

	if len(matches) > 0 {
		_, _ = fmt.Fprintln(w, "Matching sessions:")
		for _, session := range matches {
			// Truncate ID to 8 chars
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
