# Console Command Design - LOCKED

## Universal Pattern

```bash
# Default: save all logs to temp file
webctl console
# → /tmp/webctl-console/25-12-28-HHMMSS-console.json

# Show: output all logs to stdout
webctl console show

# Save: save all logs to custom path
webctl console save <path>
# If <path> is a directory, auto-generate filename
webctl console save ./output/
# → ./output/25-12-28-HHMMSS-console.json
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
# All logs to temp
webctl console
# → /tmp/webctl-console/25-12-28-HHMMSS-console.json

# All logs to stdout
webctl console show

# Filter by type
webctl console show --type error
webctl console show --type error,warn
webctl console show --type error --type warn

# Search within logs
webctl console show --find "TypeError"
webctl console --find "undefined"
# → /tmp/webctl-console/... (filtered logs)

# Combine filters
webctl console show --type error --find "TypeError"
webctl console save ./errors.json --type error --find "undefined"

# Limit results
webctl console show --head 10
webctl console show --tail 20
webctl console show --range 10-30

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
- Default saves to temp (preserves logs for analysis)
- `show` for interactive debugging
- `save <path>` for CI/CD or archival

**Console-specific flags:**
- `--type` - Essential for filtering error vs log vs warn
- `--head/tail/range` - Common for large log volumes
- These are filter flags, not operations, so they apply to all output modes

**No specific subcommands:**
- Console doesn't need operations like CSS's `computed/get/inject`
- Filtering and output control covers all use cases
