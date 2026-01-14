# DR-019: Terminal Colors

- Date: 2025-12-24
- Status: Accepted
- Category: CLI

## Problem

webctl's text output is currently monochrome, making it difficult for users to quickly scan and identify important information like errors, warnings, HTTP status codes, and log levels. In a terminal-based workflow, color is a critical tool for improving usability and reducing cognitive load when reading command output.

Users need:

- Quick visual identification of errors vs successes
- Easy scanning of console logs by severity level
- Clear distinction between different HTTP methods and status codes
- Visual feedback on the current context in the REPL prompt

Additionally, some users and environments require the ability to disable colors (CI/CD pipelines, accessibility needs, personal preference).

## Decision

Implement ANSI color support for all text output using the github.com/fatih/color library with a semantic color scheme and multiple disable mechanisms.

Color detection follows this priority order:

1. If --json flag is set: No colors (JSON must be parseable)
2. Else if --no-color flag is set: No colors
3. Else if NO_COLOR env var is set: No colors (standard convention)
4. Else if stdout is a TTY: Use colors
5. Else: No colors (piped/redirected output)

## Why

ANSI colors significantly improve terminal UX:

- Errors in red immediately draw attention to problems
- HTTP status codes colored by category (2xx green, 5xx red) provide instant feedback
- Colored log levels (ERROR red, WARNING yellow) make scanning logs faster
- REPL prompt colors help distinguish the prompt from output

Using github.com/fatih/color provides:

- Cross-platform ANSI color support (Windows, Linux, macOS)
- Automatic color disabling on non-TTY outputs
- Simple API for applying colors
- Well-tested and widely used in Go CLI tools

The priority order ensures:

- JSON output is never corrupted with ANSI codes
- Users have both global (NO_COLOR) and per-command (--no-color) control
- Colors appear automatically in interactive terminals
- Piped output remains clean for further processing

## Trade-offs

Accept:

- Additional dependency (github.com/fatih/color)
- Slightly more complex output logic (color detection)
- Need to test output in both colored and non-colored modes
- ANSI codes add bytes to output (negligible for human-readable text)

Gain:

- Significantly improved readability and usability
- Faster visual scanning of output
- Better error visibility
- Standard behavior matching other CLI tools (git, ls, ripgrep, etc.)
- Accessibility through multiple disable mechanisms

## Alternatives

Use standard library only (manual ANSI codes):

- Pro: No dependency
- Pro: Full control over color codes
- Con: Manual handling of Windows compatibility
- Con: Need to implement color detection logic ourselves
- Con: More code to maintain and test
- Rejected: fatih/color provides better cross-platform support and is well-tested

Use a different color library (e.g., aurora, lipgloss):

- Pro: aurora is zero-allocation
- Pro: lipgloss has advanced styling features
- Con: aurora doesn't handle NO_COLOR automatically
- Con: lipgloss is heavier and designed for TUI apps, not simple CLI output
- Rejected: fatih/color is the right balance of features and simplicity

Add --color=auto|always|never flag instead of --no-color:

- Pro: More flexible (can force colors on)
- Con: More complex for users
- Con: --color=always is rarely needed (edge case for piping to less -R)
- Rejected: --no-color is simpler and covers the main use case

No color support:

- Pro: Simpler code
- Con: Poor UX for terminal users
- Con: Doesn't match modern CLI tool expectations
- Rejected: Colors are a standard feature of quality CLI tools

## Color Scheme

Errors and Warnings:

- Error prefix: Red (FgRed)
- WARNING level: Yellow (FgYellow)

Success and Status:

- OK/Success states: Green (FgGreen)
- INFO level: Cyan (FgCyan)

Console Logs:

- Timestamps: Dim/Faint (Faint attribute)
- ERROR: Red (FgRed)
- WARNING: Yellow (FgYellow)
- INFO: Cyan (FgCyan)
- LOG: Default (no color)

Network Requests:

- GET method: Green (FgGreen)
- POST method: Blue (FgBlue)
- PUT method: Yellow (FgYellow)
- DELETE method: Red (FgRed)
- Status 2xx: Green (FgGreen)
- Status 3xx: Cyan (FgCyan)
- Status 4xx: Yellow (FgYellow)
- Status 5xx: Red (FgRed)

REPL Prompt (webctl [example.com]>):

- "webctl" text: Blue (FgBlue)
- "[" and "]" brackets: Default (no color)
- URL/domain: Cyan (FgCyan)
- ">" prompt character: Bold white (FgWhite + Bold)

Status Output:

- Active session marker "*": Green (FgGreen) or Cyan (FgCyan)
- "Not running": Yellow (FgYellow)
- "No browser": Yellow (FgYellow)
- "No session": Yellow (FgYellow)

## Implementation Notes

Color Detection:

- Add --no-color flag to root command persistent flags
- Check NO_COLOR environment variable using os.Getenv
- Use term.IsTerminal for TTY detection (already in use)
- Update OutputOptions.UseColor logic to follow priority order

Color Application:

- Create color instances once (package-level or in a struct) for performance
- Apply colors in format package functions (Console, Network, Status, etc.)
- Only apply colors when opts.UseColor is true
- REPL prompt should follow same color detection rules

Testing:

- Existing tests should continue to work (they likely don't use TTY)
- Add tests for color detection logic
- Manual testing required for visual verification of colors
- Test with NO_COLOR set and --no-color flag

## Usage Examples

Default behavior (TTY detected):

```bash
webctl console
# Output with colored log levels and timestamps
```

Disable colors for one command:

```bash
webctl console --no-color
# Output without colors
```

Disable colors globally:

```bash
export NO_COLOR=1
webctl console
# Output without colors
```

JSON output (never colored):

```bash
webctl console --json
# JSON output, no ANSI codes
```

REPL with colored prompt:

```bash
webctl start
# Shows: webctl [example.com]> with colors
```

## Updates

- 2025-12-24: Initial implementation completed. During implementation, discovered that peterh/liner library explicitly rejects prompts containing control characters (ANSI escape codes). Replaced liner with chzyer/readline (v1.5.1) which supports ANSI escape sequences in prompts. Updated DR-008 to reflect this change. All color functionality implemented and tested.
