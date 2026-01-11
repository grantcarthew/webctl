# Console Command Design - LOCKED

## Universal Pattern

```bash
# Default: output all logs to stdout
webctl console

# Save: save all logs to file
webctl console save           # Save to temp file
webctl console save <path>    # Save to custom path

# Path conventions (trailing slash required for directories):
webctl console save ./logs.json   # File: saves to ./logs.json
webctl console save ./output/     # Directory: auto-generates filename
# → ./output/25-12-28-HHMMSS-console.json
webctl console save ./output      # File: saves to ./output (not a directory!)
```

## Universal Flags

```bash
--find, -f TEXT          # Search within log messages
--raw                    # Skip formatting/pretty-printing
--json                   # JSON output (global flag)
```

## Console-Specific Flags

These filters are specific to console log entries:

```bash
--type TYPE              # Filter by type: log, warn, error, debug, info
                         # Repeatable: --type error --type warn
                         # CSV-supported: --type error,warn
--head N                 # Return first N entries
--tail N                 # Return last N entries
--range N-M              # Return entries N through M
```

Note: `--head`, `--tail`, and `--range` are mutually exclusive.

## Examples

```bash
# All logs to stdout
webctl console

# All logs to temp file
webctl console save
# → /tmp/webctl-console/25-12-28-HHMMSS-console.json

# Filter by type
webctl console --type error
webctl console --type error,warn
webctl console --type error --type warn

# Search within logs
webctl console --find "TypeError"
webctl console save --find "undefined"
# → /tmp/webctl-console/... (filtered logs)

# Combine filters
webctl console --type error --find "TypeError"
webctl console save ./errors.json --type error --find "undefined"

# Limit results
webctl console --head 10
webctl console --tail 20
webctl console --range 10-30

# Save filtered logs
webctl console save ./recent-errors.json --type error --tail 50
```

## Output Format

**Text mode:**
- Formatted table with timestamp, type, and message
- Color-coded by type (unless `--raw` or `--no-color`)

**JSON mode:**
- Array of console entry objects
- Each entry includes: timestamp, type, message, args, stackTrace, url, lineNumber

## Console-Specific Subcommands

None. Console uses only the universal pattern with console-specific filter flags.

## Design Rationale

**Universal pattern:**
- Consistent with html, css, network, cookies
- Default outputs to stdout (Unix convention)
- `save` for file output (temp or custom path)

**Console-specific flags:**
- `--type` - Essential for filtering error vs log vs warn
- `--head/tail/range` - Common for large log volumes
- These are filter flags, not operations, so they apply to all output modes

**No specific subcommands:**
- Console doesn't need operations like CSS's `computed/get/inject`
- Filtering and output control covers all use cases
