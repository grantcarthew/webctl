# P-020: CSS Command Implementation

- Status: Proposed
- Started: (not yet started)
- Completed: (not yet completed)

## Overview

**This is a breaking redesign and migration project.** Refactor the existing CSS command implementation to follow the new unified observation pattern defined in DR-026.

The current CSS command (with save/computed/get/inject subcommands) will be restructured to:
- Use universal pattern for stylesheet extraction (default/show/save)
- Retain CSS-specific operations as separate subcommands (computed, get, inject)
- Add universal filtering flags (--select, --find, --raw, --json)

This migration brings CSS command into alignment with the universal pattern while maintaining specialized CSS operations that are unique to styling inspection and modification.

## Goals

1. Implement new CSS command interface per DR-026
2. Add default/show/save subcommands for stylesheet extraction
3. Add universal flags (--select, --find, --raw, --json)
4. Retain CSS-specific subcommands (computed, get, inject)
5. Update CLI command file (internal/cli/css.go)
6. Update daemon handlers if needed (internal/daemon/handlers_css.go)
7. Update IPC protocol if needed (internal/ipc/protocol.go)
8. Add/update tests for new interface
9. Update CLI documentation

## Scope

In Scope:
- CSS command interface redesign (DR-026)
- Default behavior for stylesheet extraction (save to temp)
- Show subcommand for stylesheet output (stdout)
- Save subcommand for custom path
- Universal flags for observation (--select, --find, --raw, --json)
- Retain computed subcommand (computed styles to stdout)
- Retain get subcommand (single property to stdout)
- Retain inject subcommand (CSS injection)
- Path handling (directory vs file detection)
- File naming pattern updates
- Integration tests for new interface
- Documentation updates

Out of Scope:
- Changes to other observation commands (covered in P-019, P-021, P-022, P-023)
- CSS parsing/formatting changes
- CDP protocol changes
- New CSS-specific operations beyond existing computed/get/inject

## Success Criteria

- [ ] Default (no subcommand) saves all stylesheets to temp
- [ ] Show subcommand outputs all stylesheets to stdout
- [ ] Save <path> subcommand saves to custom path
- [ ] Directory paths auto-generate filenames
- [ ] File paths use exact path
- [ ] --select flag returns computed styles for element
- [ ] --find flag searches within CSS
- [ ] --raw flag skips formatting
- [ ] --json flag outputs JSON format
- [ ] computed subcommand works (all computed styles to stdout)
- [ ] get subcommand works (single property to stdout)
- [ ] inject subcommand works (CSS injection)
- [ ] All existing tests pass
- [ ] New tests cover all output modes and flags
- [ ] Documentation updated
- [ ] AGENTS.md updated with new CSS command pattern

## Deliverables

- Updated internal/cli/css.go (command implementation)
- Updated internal/daemon/handlers_css.go (if needed)
- Updated internal/ipc/protocol.go (if needed)
- New/updated tests in internal/cli/css_test.go
- Updated docs/cli/css.md (command documentation)
- Updated AGENTS.md (quick reference)

## Technical Approach

Command Structure:

Refactor CSS command with both universal pattern and specific subcommands:

```go
cssCmd := &cobra.Command{
  Use:   "css",
  Short: "Extract CSS (default: save all stylesheets to temp)",
  RunE:  cssDefaultHandler,  // Default: save to temp
}

// Universal pattern subcommands
cssShowCmd := &cobra.Command{
  Use:   "show",
  Short: "Output CSS to stdout",
  RunE:  cssShowHandler,
}

cssSaveCmd := &cobra.Command{
  Use:   "save <path>",
  Short: "Save CSS to custom path",
  Args:  cobra.ExactArgs(1),
  RunE:  cssSaveHandler,
}

// CSS-specific operation subcommands
cssComputedCmd := &cobra.Command{
  Use:   "computed <selector>",
  Short: "Get computed styles for element (stdout)",
  Args:  cobra.ExactArgs(1),
  RunE:  cssComputedHandler,
}

cssGetCmd := &cobra.Command{
  Use:   "get <selector> <property>",
  Short: "Get single property value (stdout)",
  Args:  cobra.ExactArgs(2),
  RunE:  cssGetHandler,
}

cssInjectCmd := &cobra.Command{
  Use:   "inject <css>",
  Short: "Inject CSS into page",
  RunE:  cssInjectHandler,
}

cssCmd.AddCommand(cssShowCmd, cssSaveCmd, cssComputedCmd, cssGetCmd, cssInjectCmd)
```

Universal Flags:

Add to root CSS command (inherited by default/show/save):

```go
cssCmd.Flags().StringP("select", "s", "", "Filter to element's computed styles")
cssCmd.Flags().StringP("find", "f", "", "Search within CSS")
cssCmd.Flags().Bool("raw", false, "Skip formatting")
// --json is global flag
```

Default Handler (Save All Stylesheets to Temp):

```go
func cssDefaultHandler(cmd *cobra.Command, args []string) error {
  selector, _ := cmd.Flags().GetString("select")

  var css string
  if selector != "" {
    // Get computed styles for element
    css = getComputedStyles(selector)
  } else {
    // Get all stylesheets
    css = getAllStylesheets()
  }

  // Apply find filter if needed
  if find != "" {
    css = filterCSS(css, find)
  }

  // Auto-generate filename
  filename := generateCSSFilename(title, selector)
  path := filepath.Join("/tmp/webctl-css", filename)

  // Save to temp
  writeCSSToFile(path, css, raw)

  return outputJSON(map[string]any{
    "ok": true,
    "path": path,
  })
}
```

CSS-Specific Subcommands:

Retain existing computed/get/inject logic:

```go
func cssComputedHandler(cmd *cobra.Command, args []string) error {
  selector := args[0]
  styles := getComputedStyles(selector)

  // Always stdout, formatted as property: value
  for prop, value := range styles {
    fmt.Printf("%s: %s\n", prop, value)
  }
  return nil
}

func cssGetHandler(cmd *cobra.Command, args []string) error {
  selector := args[0]
  property := args[1]
  value := getSingleProperty(selector, property)

  // Plain text output (just the value)
  fmt.Println(value)
  return nil
}

func cssInjectHandler(cmd *cobra.Command, args []string) error {
  css := args[0]
  if filePath := cmd.Flag("file").Value.String(); filePath != "" {
    css = readFile(filePath)
  }

  injectCSS(css)
  fmt.Println("OK")
  return nil
}
```

Integration with Existing Code:

- Refactor existing CSS save subcommand into universal pattern
- Keep computed/get/inject subcommands as-is (they already follow correct pattern)
- Maintain existing CDP methods for CSS extraction
- Update CLI command registration
- Reuse CSS formatting logic

Testing Strategy:

Following DR-004 testing approach with race detection and integration tests.

Unit Tests:
- Test filename generation
- Test path handling (directory vs file)
- Test flag parsing
- Test selector sanitization
- Run with -race flag for concurrency safety

Integration Tests:
- Test default behavior (save all stylesheets to temp)
- Test show subcommand (stdout output)
- Test save subcommand (custom path)
- Test --select flag (computed styles)
- Test --find flag (CSS search)
- Test --raw flag (unformatted output)
- Test --json flag (JSON output)
- Test computed subcommand (all computed styles)
- Test get subcommand (single property)
- Test inject subcommand (CSS injection)
- Test directory path auto-generation
- Test file path exact usage
- Test error cases (invalid selector, property, etc.)

Migration Considerations:

Breaking changes from DR-023:
- css save removed as subcommand (now default/show/save pattern)
- Default behavior changes (save to temp instead of subcommand)
- --output flag removed (use save <path> instead)

Migration path:
```bash
# Old
webctl css save -o ./styles.css

# New
webctl css save ./styles.css
```

```bash
# Old
webctl css save "#header"  # Computed styles to file

# New
webctl css --select "#header"  # Computed styles to temp
webctl css save ./header.css --select "#header"  # To custom path
```

Subcommands computed/get/inject remain unchanged.

## Dependencies

- DR-026: CSS Command Interface (design authority)
- DR-004: Testing Strategy (testing approach consistency)
- DR-023: CSS Command Architecture (existing implementation)
- Existing CSS command code (refactor base)
- Cobra library (subcommand support)

## Questions & Uncertainties

- Should --select flag work with computed/get/inject subcommands (currently they have positional selector)?
- How to handle conflict between universal pattern and specific subcommands (flag inheritance)?
- Should we maintain backward compatibility for css save subcommand?

Note: Project is in early development, so clean break is preferred.

## Notes

- CSS command is unique: universal pattern + specific subcommands
- Demonstrates how specialized operations coexist with universal pattern
- Success validates the pattern can handle commands with both observation and operation needs

## Updates

- 2025-12-28: Project created
