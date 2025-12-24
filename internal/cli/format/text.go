package format

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/grantcarthew/webctl/internal/ipc"
	"golang.org/x/term"
)

// OutputOptions controls text formatting behavior.
type OutputOptions struct {
	UseColor bool // Enable ANSI color codes
}

// DefaultOptions returns default output options based on TTY detection.
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
func ActionError(w io.Writer, msg string) error {
	_, err := fmt.Fprintf(w, "Error: %s\n", msg)
	return err
}

// Status outputs daemon status in text format.
func Status(w io.Writer, data ipc.StatusData, opts OutputOptions) error {
	// Not running state
	if !data.Running {
		_, err := fmt.Fprintln(w, "Not running")
		return err
	}

	// Running but no browser
	if data.ActiveSession == nil && len(data.Sessions) == 0 {
		fmt.Fprintln(w, "No browser")
		if data.PID > 0 {
			fmt.Fprintf(w, "pid: %d\n", data.PID)
		}
		return nil
	}

	// Running but no active session (browser connected but no pages)
	if data.ActiveSession == nil {
		fmt.Fprintln(w, "No session")
		if data.PID > 0 {
			fmt.Fprintf(w, "pid: %d\n", data.PID)
		}
		return nil
	}

	// All systems operational
	fmt.Fprintln(w, "OK")
	if data.PID > 0 {
		fmt.Fprintf(w, "pid: %d\n", data.PID)
	}

	// Show sessions
	if len(data.Sessions) > 0 {
		fmt.Fprintln(w, "sessions:")
		for _, session := range data.Sessions {
			prefix := "  "
			if session.Active {
				prefix = "  * "
			} else {
				prefix = "    "
			}
			fmt.Fprintf(w, "%s%s\n", prefix, session.URL)
		}
	}

	return nil
}

// Console outputs console entries in text format.
func Console(w io.Writer, entries []ipc.ConsoleEntry, opts OutputOptions) error {
	for _, e := range entries {
		ts := time.UnixMilli(e.Timestamp).Local()
		timestamp := ts.Format("15:04:05")

		// Format: [HH:MM:SS] LEVEL Message
		fmt.Fprintf(w, "[%s] %s %s\n", timestamp, strings.ToUpper(e.Type), e.Text)

		// Source URL and line number indented below
		if e.URL != "" {
			if e.Line > 0 {
				fmt.Fprintf(w, "  %s:%d\n", e.URL, e.Line)
			} else {
				fmt.Fprintf(w, "  %s\n", e.URL)
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
		fmt.Fprintf(w, "%s %s %d %dms\n", e.Method, e.URL, e.Status, durationMs)

		// Request body (if present and non-empty)
		if e.Body != "" && e.Method != "GET" {
			// Try to parse as JSON to detect if it's request/response
			// For now, just show bodies indented
			bodyLines := strings.Split(strings.TrimSpace(e.Body), "\n")
			for _, line := range bodyLines {
				fmt.Fprintf(w, "  %s\n", line)
			}
		}
	}
	return nil
}

// Cookies outputs cookies in text format (semicolon-separated attributes).
func Cookies(w io.Writer, cookies []ipc.Cookie, opts OutputOptions) error {
	for _, c := range cookies {
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

		fmt.Fprintln(w, strings.Join(parts, "; "))
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
		prefix := "  "
		if session.ID == data.ActiveSession {
			prefix = "* "
		}

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

		fmt.Fprintf(w, "%s%s - %s [%s]\n", prefix, session.URL, title, displayID)
	}
	return nil
}

// TargetError outputs target error with session/match information.
func TargetError(w io.Writer, errorMsg string, sessions []ipc.PageSession, matches []ipc.PageSession, opts OutputOptions) error {
	fmt.Fprintf(w, "Error: %s\n", errorMsg)

	if len(sessions) > 0 {
		fmt.Fprintln(w, "Available sessions:")
		for _, session := range sessions {
			// Truncate ID to 8 chars
			displayID := session.ID
			if len(displayID) > 8 {
				displayID = displayID[:8]
			}
			fmt.Fprintf(w, "  %s - %s\n", displayID, session.Title)
		}
	}

	if len(matches) > 0 {
		fmt.Fprintln(w, "Matching sessions:")
		for _, session := range matches {
			// Truncate ID to 8 chars
			displayID := session.ID
			if len(displayID) > 8 {
				displayID = displayID[:8]
			}
			fmt.Fprintf(w, "  %s - %s\n", displayID, session.Title)
		}
	}

	return nil
}
