# P-020: HTML Command Implementation

- Status: Completed
- Started: 2025-12-28
- Completed: 2025-12-28

## Overview

**This is a breaking redesign and migration project.** Refactor the existing HTML command implementation to follow the new unified observation pattern defined in DR-025.

The current HTML command will be completely replaced with a new interface that includes:
- Default/show/save output modes (replacing current file-only behavior)
- Universal filtering flags (--select, --find, --raw, --json)
- Integrated text search (eliminating need for separate find command)

This migration brings the HTML command into alignment with the universal observation command pattern, providing consistent behavior with CSS, console, network, and cookies commands.

## Goals

1. Implement new HTML command interface per DR-025
2. Add default/show/save subcommands for output mode control
3. Add universal flags (--select, --find, --raw, --json)
4. Update CLI command file (internal/cli/html.go)
5. Update daemon handlers if needed (internal/daemon/handlers_html.go)
6. Update IPC protocol if needed (internal/ipc/protocol.go)
7. Add/update tests for new interface
8. Update CLI documentation

## Scope

In Scope:
- HTML command interface redesign (DR-025)
- Default behavior (save to temp with auto-generated filename)
- Show subcommand (output to stdout)
- Save subcommand (save to custom path)
- Universal flags (--select, --find, --raw, --json)
- Path handling (directory vs file detection)
- File naming pattern updates
- Integration tests for new interface
- Documentation updates

Out of Scope:
- Changes to other observation commands (covered in P-020 through P-023)
- HTML formatting implementation (already exists)
- CDP protocol changes (HTML retrieval already implemented)
- Find command removal (covered separately in DR-030 implementation)

## Success Criteria

- [x] Default (no subcommand) saves to temp with auto-generated filename
- [x] Show subcommand outputs to stdout
- [x] Save <path> subcommand saves to custom path
- [x] Directory paths auto-generate filenames
- [x] File paths use exact path
- [x] --select flag filters to element(s)
- [x] --find flag searches within HTML
- [x] --raw flag skips formatting
- [x] --json flag outputs JSON format
- [x] All existing tests pass
- [x] New tests cover all output modes and flags
- [x] Documentation updated (via command help text)
- [x] AGENTS.md updated with new HTML command pattern

## Deliverables

- Updated internal/cli/html.go (command implementation)
- Updated internal/daemon/handlers_html.go (if needed)
- Updated internal/ipc/protocol.go (if needed)
- New/updated tests in internal/cli/html_test.go
- Updated docs/cli/html.md (command documentation)
- Updated AGENTS.md (quick reference)

## Technical Approach

Command Structure:

Refactor HTML command to use Cobra subcommands:

```go
htmlCmd := &cobra.Command{
  Use:   "html",
  Short: "Extract HTML (default: save to temp)",
  RunE:  htmlDefaultHandler,  // Default: save to temp
}

htmlShowCmd := &cobra.Command{
  Use:   "show",
  Short: "Output HTML to stdout",
  RunE:  htmlShowHandler,
}

htmlSaveCmd := &cobra.Command{
  Use:   "save <path>",
  Short: "Save HTML to custom path",
  Args:  cobra.ExactArgs(1),
  RunE:  htmlSaveHandler,
}

htmlCmd.AddCommand(htmlShowCmd, htmlSaveCmd)
```

Universal Flags:

Add to root HTML command (inherited by subcommands):

```go
htmlCmd.Flags().StringP("select", "s", "", "Filter to element(s)")
htmlCmd.Flags().StringP("find", "f", "", "Search within HTML")
htmlCmd.Flags().Bool("raw", false, "Skip formatting")
// --json is global flag, already available
```

Default Handler (Save to Temp):

```go
func htmlDefaultHandler(cmd *cobra.Command, args []string) error {
  // Get HTML from daemon
  html := getHTMLFromDaemon(selector, find)

  // Auto-generate filename
  filename := generateHTMLFilename(title, selector)
  path := filepath.Join("/tmp/webctl-html", filename)

  // Save to temp
  writeHTMLToFile(path, html, raw)

  // Return JSON response
  return outputJSON(map[string]any{
    "ok": true,
    "path": path,
  })
}
```

Show Handler (Output to Stdout):

```go
func htmlShowHandler(cmd *cobra.Command, args []string) error {
  // Get HTML from daemon
  html := getHTMLFromDaemon(selector, find)

  // Format if needed
  if !raw {
    html = formatHTML(html)
  }

  // Output to stdout
  fmt.Println(html)
  return nil
}
```

Save Handler (Custom Path):

```go
func htmlSaveHandler(cmd *cobra.Command, args []string) error {
  path := args[0]

  // Get HTML from daemon
  html := getHTMLFromDaemon(selector, find)

  // Handle directory vs file
  if isDirectory(path) {
    filename := generateHTMLFilename(title, selector)
    path = filepath.Join(path, filename)
  }

  // Save to path
  writeHTMLToFile(path, html, raw)

  // Return JSON response
  return outputJSON(map[string]any{
    "ok": true,
    "path": path,
  })
}
```

Filename Generation:

```go
func generateHTMLFilename(title, selector string) string {
  timestamp := time.Now().Format("06-01-02-150405")

  identifier := "page"
  if selector != "" {
    identifier = sanitizeSelector(selector)
  } else if title != "" {
    identifier = sanitizeTitle(title)
  }

  return fmt.Sprintf("%s-%s.html", timestamp, identifier)
}
```

Integration with Existing Code:

- Reuse existing HTML extraction logic from daemon
- Reuse HTML formatting from DR-021 implementation
- Maintain existing CDP methods (DOM.getDocument, DOM.querySelectorAll, DOM.getOuterHTML)
- Update CLI command registration
- Maintain backward compatibility where possible in daemon handlers

Testing Strategy:

Following DR-004 testing approach with race detection and integration tests.

Unit Tests:
- Test filename generation
- Test path handling (directory vs file)
- Test flag parsing
- Test selector sanitization
- Run with -race flag for concurrency safety

Integration Tests:
- Test default behavior (save to temp)
- Test show subcommand (stdout output)
- Test save subcommand (custom path)
- Test --select flag (element filtering)
- Test --find flag (text search)
- Test --raw flag (unformatted output)
- Test --json flag (JSON output)
- Test directory path auto-generation
- Test file path exact usage
- Test error cases (invalid selector, no elements, etc.)
- Integration with real daemon/browser connection

Migration Considerations:

Breaking changes from previous interface:
- Default behavior changes (requires subcommand for custom path)
- --output flag removed (use save <path> instead)
- Selector moved to --select flag (was positional argument)

Migration path for existing scripts:
```bash
# Old
webctl html -o ./page.html

# New
webctl html save ./page.html
```

```bash
# Old
webctl html "#main"

# New
webctl html --select "#main"
```

## Dependencies

- DR-025: HTML Command Interface (design authority)
- DR-004: Testing Strategy (testing approach consistency)
- DR-021: HTML Formatter (existing formatting logic)
- Existing HTML command implementation (refactor base)
- Cobra library (subcommand support)

## Questions & Uncertainties

- Should we maintain any backward compatibility shims?
- How do we handle migration for existing scripts?
- Should we add deprecation warnings before removing old interface?

Note: Project is in early development, so clean break is preferred over backward compatibility.

## Notes

- This is one of five observation command implementation projects (P-019 through P-023)
- All observation commands follow the same universal pattern
- HTML command serves as reference implementation for the pattern
- Success of this project validates the universal pattern design

## Updates

- 2025-12-28: Project created
