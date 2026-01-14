# P-014: Terminal Colors

- Status: Completed
- Started: 2025-12-24
- Completed: 2025-12-24

## Overview

Add ANSI color support to webctl's terminal output to improve readability and usability. Colors will highlight important information like errors, status codes, log levels, and the REPL prompt, making it easier for users to quickly scan and understand command output.

## Goals

1. Implement color support using github.com/fatih/color library
2. Apply semantic colors to all text output formats (console, network, status, errors, target, REPL)
3. Respect standard color disable mechanisms (NO_COLOR env var, --no-color flag)
4. Ensure colors are only used when appropriate (TTY detection, not in JSON mode)
5. Create a consistent and readable color scheme across all outputs

## Scope

In Scope:

- Color support in all text formatters (internal/cli/format/)
- Color detection logic (TTY, NO_COLOR, --no-color flag)
- REPL prompt colorization
- Error message colorization
- Console log level colors
- Network request method and status code colors
- Status message colors
- Target session list colors

Out of Scope:

- JSON output colorization (JSON must remain parseable)
- Custom color configuration (user-defined color schemes)
- Color themes or alternate palettes

## Success Criteria

- [x] github.com/fatih/color dependency added to go.mod
- [x] Color detection respects priority: --json > --no-color > NO_COLOR > TTY
- [x] Console output colorized (timestamps, log levels, sources)
- [x] Network output colorized (methods, status codes, URLs, durations)
- [x] Status output colorized (state messages, session markers)
- [x] Error messages colorized (Error prefix in red)
- [x] Target output colorized (active markers, IDs, URLs)
- [x] REPL prompt colorized (webctl/URL/prompt character) - FIXED: Replaced liner with readline
- [x] All existing tests pass
- [ ] Manual testing confirms colors appear correctly in terminal (including REPL)

## Deliverables

- Updated internal/cli/format/text.go with color support
- Updated internal/daemon/repl.go with colored prompt
- Updated internal/cli/root.go with --no-color flag and color detection logic
- DR documenting color implementation decisions
- Updated go.mod with fatih/color dependency

## Color Scheme

Errors and Warnings:

- Error prefix: Red
- Warning level: Yellow

Success and Status:

- OK/Success: Green
- Info: Cyan

Console Logs:

- Timestamps: Dim/Gray
- ERROR: Red
- WARNING: Yellow
- INFO: Cyan
- LOG: Default

Network Requests:

- GET: Green
- POST: Blue
- PUT: Yellow
- DELETE: Red
- Status 2xx: Green
- Status 3xx: Cyan
- Status 4xx: Yellow
- Status 5xx: Red

REPL Prompt:

- "webctl": Blue
- "[" and "]": Default
- URL/domain: Cyan
- ">": Bold white

## Color Detection Priority

1. If --json flag is set: No colors (JSON must be parseable)
2. Else if --no-color flag is set: No colors
3. Else if NO_COLOR env var is set: No colors
4. Else if stdout is a TTY: Use colors
5. Else: No colors (piped/redirected output)

## Implementation Notes

- Use github.com/fatih/color for ANSI color code generation
- Color instances should be created once and reused for performance
- All color application should check the UseColor flag from OutputOptions
- REPL color detection should follow the same rules (always TTY but respect flags)
- Ensure color codes don't break existing output parsing or tests

## Known Issues

### REPL Prompt Color Bug - RESOLVED

**Status**: FIXED (2025-12-24)

**Original Issue**: The liner library (github.com/peterh/liner v1.2.2) explicitly rejected prompts containing control characters (including ANSI escape codes) in its validation code:
```go
for _, r := range prompt {
    if unicode.Is(unicode.C, r) {
        return "", ErrInvalidPrompt
    }
}
```

**Solution Implemented**: Replaced `github.com/peterh/liner` with `github.com/chzyer/readline` (v1.5.1), which explicitly supports ANSI escape sequences in prompts. From readline's documentation: "prompt supports ANSI escape sequence, so we can color some characters even in windows."

**Changes Made**:
- Updated go.mod to use chzyer/readline instead of peterh/liner
- Modified internal/daemon/repl.go to use readline API
- Re-enabled shouldUseREPLColor() to return true
- All tests pass

**Testing Required**: Manual verification of colored prompts in REPL

## Progress Notes

2025-12-24 (Morning): Implementation complete except for REPL prompt coloring. All tests passing. Colors working for all CLI commands (status, console, network, etc.). REPL prompt bug discovered and documented.

2025-12-24 (Evening): REPL prompt bug resolved by replacing liner with readline library. All tests passing. Ready for manual testing and completion.
