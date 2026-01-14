# DR-027: Console Command Interface

- Date: 2025-12-28
- Status: Accepted
- Category: CLI

## Problem

The current console command only outputs to stdout, making it inconsistent with other observation commands that support multiple output modes. Current limitations:

- No option to save console logs to file for later analysis
- Cannot preserve logs for archival or CI/CD pipelines
- Inconsistent with html, css, network commands that support file output
- Missing universal pattern (default/show/save) for output control
- No integrated text search for filtering log messages

Users need console logs to follow the universal observation pattern with file output options while maintaining console-specific filtering capabilities.

## Decision

Redesign console command to follow the universal observation pattern with console-specific filter flags:

```bash
# Universal pattern
webctl console              # Output all logs to stdout (Unix convention)
webctl console save [path]  # Save to file (temp if no path, custom if path given)

# Universal flags
--find, -f TEXT             # Search within log messages
--raw                       # Skip formatting
--json                      # JSON output

# Console-specific filter flags
--type TYPE                 # Filter by type: log, warn, error, debug, info
--head N                    # First N entries
--tail N                    # Last N entries
--range N-M                 # Entries N through M
```

The console command outputs to stdout by default (Unix convention), with a save subcommand for file output. Filtering is provided through console-specific flags that apply to all output modes.

Complete specification: .ai/design/interface/console.md

## Why

Unix Convention (stdout by default):

Following Unix philosophy, observation commands output to stdout by default. This enables:
- Piping to other tools (grep, less, jq)
- Quick inspection without file management
- Consistent with standard CLI tools

Save Subcommand for Files:

When file output is needed, the save subcommand provides flexibility:
- `console save` - saves to temp directory with auto-generated filename
- `console save ./logs.json` - saves to custom path (file)
- `console save ./output/` - saves to directory with auto-generated filename (trailing slash required)
- Trailing slash convention (like rsync): path with `/` suffix is directory, without is file

Console-Specific Filter Flags:

The --type, --head, --tail, and --range flags provide filtering specific to console log entries. These filters apply to all output modes, allowing users to filter before output regardless of destination.

Type filtering is essential for debugging (show only errors/warnings). Range filters handle large log volumes. These are console-specific needs that don't apply universally.

No Console-Specific Subcommands:

Console logging doesn't require special operations like CSS does (computed/get/inject) or cookies does (set/delete). All console functionality is observation and filtering, which maps perfectly to the universal pattern with filter flags.

Text Search Integration:

The --find flag enables searching within log messages, matching the pattern for HTML and CSS. Users can filter logs by content, useful for finding specific errors or debugging messages.

## Trade-offs

Accept:

- Breaking change from current stdout-only behavior
- Default to file may surprise users expecting stdout
- Temp files require eventual cleanup
- More complex command structure with subcommands
- Users must learn new pattern for familiar command
- Additional flags for filtering increase surface area

Gain:

- Consistent interface across all observation commands
- Log preservation for debugging and analysis
- Flexible output modes for different use cases
- File output for CI/CD and automation
- Integrated text search for log filtering
- Filter flags work across all output modes
- Foundation matches other observation commands
- Predictable behavior pattern (learn once, use everywhere)

## Alternatives

Keep Current Stdout-Only Behavior:

```bash
webctl console    # Always stdout
```

- Pro: No breaking changes, existing scripts work
- Pro: Simple single behavior
- Pro: Matches current user expectations
- Con: No way to save logs to file
- Con: Inconsistent with html/css/network commands
- Con: Cannot preserve logs for later analysis
- Rejected: Fails to provide file output capability and consistency

Add File Output Flag:

```bash
webctl console               # Stdout (current behavior)
webctl console -o <path>     # Save to file (new option)
```

- Pro: Minimal breaking change
- Pro: Adds file capability
- Con: Inconsistent with universal pattern
- Con: Stdout-first doesn't match other observation commands
- Con: Doesn't establish predictable pattern
- Rejected: Partial solution that doesn't achieve consistency

Mirror HTML/CSS Default Behavior:

```bash
webctl console               # Save to temp (like html/css)
webctl console -o <path>     # Save to custom path
# No stdout option
```

- Pro: Matches html/css pattern
- Pro: File-first approach
- Con: No stdout option breaks interactive debugging
- Con: Users expect console logs on screen
- Rejected: Console logs often needed interactively, must support stdout

Always Require Output Mode:

```bash
webctl console show          # Must specify show for stdout
webctl console save <path>   # Must specify save for file
# No default behavior
```

- Pro: Explicit intent required
- Pro: No assumptions about user preference
- Con: Verbose for common case
- Con: Extra typing for every console command
- Con: Less ergonomic than sensible default
- Rejected: Default to temp provides better ergonomics

Separate Commands for Output Modes:

```bash
webctl console-show          # Stdout
webctl console-save <path>   # File output
```

- Pro: Very explicit
- Pro: No subcommand complexity
- Con: Clutters command namespace
- Con: Two commands instead of one
- Con: Less discoverable
- Rejected: Subcommands group functionality better

## Structure

Output Modes:

Default (no subcommand):
- Outputs console logs to stdout (Unix convention)
- Formatted table with timestamp, type, message
- Color-coded by type unless --raw or piped
- Useful for piping to other tools

Save subcommand:
- Optional path argument
- No path: saves to /tmp/webctl-console/ with auto-generated filename
- Path with trailing slash (path/): auto-generates filename in that directory
- Path without trailing slash (path): saves to exact file path
- Creates parent directories if needed
- Trailing slash convention follows Unix tools like rsync

Universal Flags:

--find, -f TEXT:
- Search for text within log messages
- Filters logs containing search text
- Works across all output modes
- Case-insensitive search

--raw:
- Skips formatting/pretty-printing
- Returns logs in raw format
- Useful for machine processing

--json:
- Global flag for JSON output format
- Array of console entry objects
- Each entry: timestamp, type, message, args, stackTrace, url, lineNumber

Console-Specific Filter Flags:

--type TYPE:
- Filter by log type: log, warn, error, debug, info
- Repeatable: --type error --type warn
- CSV-supported: --type error,warn
- Multiple values are OR-combined

--head N:
- Return first N log entries
- Applied after other filters
- Mutually exclusive with --tail and --range

--tail N:
- Return last N log entries
- Applied after other filters
- Mutually exclusive with --head and --range
- Most common for recent errors

--range N-M:
- Return entries N through M (inclusive)
- Applied after other filters
- Mutually exclusive with --head and --tail

## Usage Examples

Default behavior (stdout):

```bash
webctl console
# 14:30:52 | log   | Page loaded
# 14:30:53 | warn  | Deprecated API call
# 14:30:54 | error | TypeError: undefined

webctl console --type error
# 14:30:54 | error | TypeError: undefined
# 14:31:02 | error | Failed to fetch

webctl console --find "TypeError"
# 14:30:54 | error | TypeError: undefined
# 14:31:15 | error | TypeError: Cannot read property
```

Save to file:

```bash
webctl console save
# {"ok": true, "path": "/tmp/webctl-console/25-12-28-143052-console.json"}

webctl console save ./logs/debug.json
# {"ok": true, "path": "./logs/debug.json"}

webctl console save ./output/
# {"ok": true, "path": "./output/25-12-28-143052-console.json"}

webctl console save ./errors.json --type error
# {"ok": true, "path": "./errors.json"}
```

Console-specific filters:

```bash
# Type filtering
webctl console --type error,warn
webctl console --type error --type warn
webctl console save --type error  # Save errors to temp

# Search within logs
webctl console --find "undefined"
webctl console --type error --find "TypeError"

# Limit results
webctl console --head 10
webctl console --tail 20
webctl console --range 10-30

# Combined filtering
webctl console save ./recent-errors.json --type error --tail 50
webctl console --type error --find "fetch" --tail 20
```

JSON output:

```bash
webctl console --json
# {
#   "ok": true,
#   "logs": [
#     {
#       "timestamp": "2025-12-28T14:30:52Z",
#       "type": "log",
#       "message": "Page loaded",
#       "args": [...],
#       "url": "https://example.com",
#       "lineNumber": 42
#     },
#     ...
#   ]
# }

webctl console save ./logs.json --json
# {"ok": true, "path": "./logs.json"}
```

## File Naming

Auto-generated Filenames:

Pattern: /tmp/webctl-console/YY-MM-DD-HHMMSS-console.json

Default extension: .json (logs are structured data)

Example filenames:
- 25-12-28-143052-console.json
- 25-12-28-143115-console.json
- 25-12-28-143120-console.json

Identifier: Fixed to "console" (no variation needed)

## Output Format

Text Mode (default):

Formatted table with color-coding:
```
14:30:52 | log   | Page loaded
14:30:53 | warn  | Deprecated API call
14:30:54 | error | TypeError: undefined
```

Color scheme:
- log: default color
- warn: yellow
- error: red
- debug: blue
- info: cyan

Use --raw to disable formatting and colors.

JSON Mode (--json flag):

Array of console entry objects:
```json
{
  "ok": true,
  "logs": [
    {
      "timestamp": "2025-12-28T14:30:52Z",
      "type": "log",
      "message": "Page loaded",
      "args": [],
      "url": "https://example.com/page.js",
      "lineNumber": 42,
      "stackTrace": null
    }
  ]
}
```

## Breaking Changes

From DR-007 (Console Command Interface):

1. Changed: Default behavior now outputs to stdout (Unix convention)
2. Removed: show subcommand (not needed - stdout is default)
3. Changed: save subcommand now takes optional path (temp if no path)
4. Added: --find flag for text search within logs
5. Retained: --type, --head, --tail, --range filters (behavior unchanged)
6. Retained: Color-coded output for default mode

Migration Guide:

Old pattern (DR-007):
```bash
webctl console                    # Stdout with all logs
webctl console --type error       # Stdout with errors only
```

New pattern (DR-027 after P-051):
```bash
webctl console                    # Output to stdout (same as before)
webctl console --type error       # Stdout with errors only (same)
webctl console save               # Save to temp (new)
webctl console save ./logs.json   # Save to custom path (new feature)
```

The default stdout behavior is preserved. Use `webctl console save` when file output is needed.

## Updates

- 2026-01-09: Updated to stdout default, removed show subcommand (P-051)
- 2025-12-28: Initial version (supersedes DR-007)
