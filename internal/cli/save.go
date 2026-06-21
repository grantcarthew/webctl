package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grantcarthew/webctl/internal/cli/format"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

// saveSpec captures the per-command variation points of the shared save flow.
// Everything else — daemon check, sentinel mapping, three-mode path resolution,
// filename generation, file writing, and success output — is uniform and lives
// in runSave.
type saveSpec struct {
	// timerLabel names the operation for debug timing (e.g. "html save").
	timerLabel string
	// tempDir is the directory used when no path argument is given.
	tempDir string
	// ext is the file extension without a leading dot (e.g. "html", "json").
	ext string
	// produce returns the exact bytes to write to disk. For content commands
	// this is the content string; for buffer commands it is the marshalled JSON
	// envelope. Sentinel errors from this function map to notices, not errors.
	produce func(*cobra.Command) (string, error)
	// identifier resolves the optional filename identifier segment. Returns an
	// empty string to omit the segment entirely.
	identifier func(*cobra.Command) string
}

// runSave executes the shared save flow for a string-content command.
func runSave(cmd *cobra.Command, args []string, spec saveSpec) error {
	t := startTimer(spec.timerLabel)
	defer t.log()

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	content, err := spec.produce(cmd)
	if err != nil {
		if notice, ok := saveSentinelNotice(err); ok {
			return notice
		}
		return outputError(err.Error())
	}

	outputPath, err := resolveSavePath(cmd, args, spec)
	if err != nil {
		return outputError(err.Error())
	}

	if err := writeSaveFile(outputPath, content); err != nil {
		return outputError(err.Error())
	}

	if JSONOutput {
		return outputJSON(os.Stdout, map[string]any{
			"ok":   true,
			"path": outputPath,
		})
	}

	return format.FilePath(os.Stdout, outputPath)
}

// saveSentinelNotice maps the informational sentinels a save command can return
// to their notice message. The union covers every covered command; a sentinel a
// given command never returns simply never fires for it. Returns false for any
// non-sentinel error so the caller surfaces it via outputError.
func saveSentinelNotice(err error) (error, bool) {
	switch {
	case errors.Is(err, ErrNoMatches):
		return outputNotice("No matches found"), true
	case errors.Is(err, ErrNoElements):
		return outputNotice("No elements found"), true
	case errors.Is(err, ErrNoRules):
		return outputNotice("No rules found"), true
	case errors.Is(err, ErrNoEntriesInRange):
		return outputNotice("No entries in range"), true
	}
	return nil, false
}

// resolveSavePath resolves the destination path across the three save modes:
// no argument (temp dir + auto filename), trailing-separator argument (treat as
// directory + auto filename), or a plain argument (exact file path).
func resolveSavePath(cmd *cobra.Command, args []string, spec saveSpec) (string, error) {
	if len(args) == 0 {
		return filepath.Join(spec.tempDir, spec.filename(cmd)), nil
	}

	path := args[0]
	if strings.HasSuffix(path, string(os.PathSeparator)) || strings.HasSuffix(path, "/") {
		if err := os.MkdirAll(path, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory: %v", err)
		}
		return filepath.Join(path, spec.filename(cmd)), nil
	}

	return path, nil
}

// filename generates the unified save filename:
// YY-MM-DD-HHMMSS-mmm[-identifier].{ext}. The millisecond segment ensures two
// saves within the same second do not collide. The identifier segment is
// omitted when empty.
func (s saveSpec) filename(cmd *cobra.Command) string {
	now := time.Now()
	base := fmt.Sprintf("%s-%03d", now.Format("06-01-02-150405"), now.Nanosecond()/int(time.Millisecond))

	id := ""
	if s.identifier != nil {
		id = s.identifier(cmd)
	}
	if id != "" {
		return fmt.Sprintf("%s-%s.%s", base, id, s.ext)
	}
	return fmt.Sprintf("%s.%s", base, s.ext)
}

// marshalSaveEnvelope marshals a buffer command's JSON envelope into the string
// payload the save helper writes to disk, preserving the indented file format.
func marshalSaveEnvelope(data map[string]any) (string, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal save data: %v", err)
	}
	return string(jsonBytes), nil
}

// writeSaveFile writes content to path, creating parent directories as needed.
func writeSaveFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}
	debugFile("wrote", path, len(content))
	return nil
}

// fixedIdentifier returns an identifier source that always yields word. Used by
// the buffer commands (console, network, cookies).
func fixedIdentifier(word string) func(*cobra.Command) string {
	return func(*cobra.Command) string { return word }
}

// selectorOrTitleIdentifier resolves the identifier for content commands
// (html, css, markdown): the sanitized --select value if given, otherwise the
// normalized page title, otherwise empty. A failed title lookup yields an empty
// identifier so the save still succeeds.
func selectorOrTitleIdentifier(cmd *cobra.Command) string {
	if selector := saveSelectorFlag(cmd); selector != "" {
		return sanitizeSelector(selector)
	}
	return pageTitleIdentifier()
}

// pageTitleIdentifier fetches the active session title via a status request and
// normalizes it for filename use. Returns an empty string when the daemon,
// request, or session has no usable title.
func pageTitleIdentifier() string {
	exec, err := execFactory.NewExecutor()
	if err != nil {
		return ""
	}
	defer func() { _ = exec.Close() }()

	resp, err := exec.Execute(ipc.Request{Cmd: "status"})
	if err != nil || !resp.OK {
		return ""
	}

	var status ipc.StatusData
	if err := json.Unmarshal(resp.Data, &status); err != nil {
		return ""
	}
	if status.ActiveSession != nil && status.ActiveSession.Title != "" {
		return normalizeTitle(status.ActiveSession.Title)
	}
	return ""
}

// saveSelectorFlag reads the --select flag from a save subcommand, falling back
// to the parent command's persistent flag.
func saveSelectorFlag(cmd *cobra.Command) string {
	selector, _ := cmd.Flags().GetString("select")
	if selector == "" && cmd.Parent() != nil {
		selector, _ = cmd.Parent().PersistentFlags().GetString("select")
	}
	return selector
}
