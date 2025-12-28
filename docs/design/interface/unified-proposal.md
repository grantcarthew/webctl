# Unified Observation Command Proposal

## Core Principle
**Every observation command extracts data. Filtering and output destination are orthogonal concerns.**

## Unified Command Structure

```bash
webctl <WHAT> [scope] [filters...] [output-control]
```

Where:
- **WHAT** = html | css | console | network | cookies
- **scope** = command-specific selector/target
- **filters** = --find, --regex, --limit, etc. (universal)
- **output-control** = --output PATH or default smart behavior

## Proposed Commands

### HTML
```bash
# Basic extraction
webctl html                           # Full page → file
webctl html [selector]                # Element(s) → file
webctl html --stdout                  # Force stdout (may truncate)

# With filtering
webctl html --find "login"            # Find in HTML → file with matches highlighted
webctl html form --find "password"    # Find in specific selector

# Output control
webctl html -o page.html              # Custom path
webctl html --limit 5                 # First 5 elements (for selectors)
```

### CSS
```bash
# Basic extraction (collapse the subcommands)
webctl css                            # All stylesheets → file
webctl css [selector]                 # Computed styles for element → file or stdout?
webctl css [selector] --property bg   # Single property → stdout (always)

# With filtering
webctl css --find "background"        # Find CSS rules → stdout matches
webctl css button --find "hover"      # Find in element's styles

# Output control
webctl css -o styles.css              # Custom path
webctl css --stdout                   # Force stdout
```

### Console
```bash
# Basic extraction
webctl console                        # All logs → stdout (default)
webctl console --type error           # Filter by type

# With filtering
webctl console --find "TypeError"     # Text search in logs
webctl console --find "api" --type log # Combined filters

# Output control
webctl console -o console.json        # Save to file
webctl console --head 10              # First 10 entries
webctl console --tail 20              # Last 20 entries
```

### Network
```bash
# Basic extraction
webctl network                        # All requests → stdout (default)
webctl network --status 4xx           # Filter by status

# With filtering
webctl network --find "api/user"      # Text search in URLs/bodies
webctl network --find "error" --status 500 # Combined filters

# Output control
webctl network -o requests.json       # Save to file
webctl network --limit 50             # First 50 requests
```

### Cookies
```bash
# Basic extraction
webctl cookies                        # All cookies → stdout (default)

# With filtering (NEW)
webctl cookies --find "session"       # Search by name
webctl cookies --domain ".github.com" # Filter by domain

# Output control
webctl cookies -o cookies.json        # Save to file

# Mutations (keep as subcommands)
webctl cookies set <name> <value>
webctl cookies delete <name>
```

## Universal Flags (All Commands)

### Filtering
```bash
--find TEXT         # Text search (case-insensitive by default)
--find-regex REGEX  # Regex search
--case-sensitive    # Make --find case-sensitive
--limit N           # Limit to N results
--head N            # First N items
--tail N            # Last N items
--range N-M         # Items N through M
```

### Output Control
```bash
--output PATH, -o   # Save to file instead of stdout
--stdout            # Force stdout (even if normally file)
--raw               # Skip formatting/pretty-printing
```

### Format
```bash
--json              # JSON output (global flag)
```

## Smart Defaults

| Command | Default Output | Reason |
|---------|---------------|--------|
| `html` | File | Always large (10KB-1MB) |
| `css` (no selector) | File | Stylesheets are large (100KB+) |
| `css [selector]` | Stdout | Computed styles are medium (2-10KB) |
| `console` | Stdout | Usually small-medium, interactive |
| `network` | Stdout | Usually small-medium, interactive |
| `cookies` | Stdout | Always small (few KB) |

Override with `--output` or `--stdout` as needed.

## Remove `find` as Separate Command?

Instead of `webctl find <text>`, integrate into each command:

```bash
# Old way
webctl find "login"                   # Searches HTML only

# New way
webctl html --find "login"            # Search HTML
webctl css --find "button"            # Search CSS
webctl console --find "error"         # Search console
webctl network --find "api/auth"      # Search network
```

**Benefits:**
- Consistent filtering across all data sources
- No confusion about what find searches
- Can combine with other filters
- More discoverable

**Drawbacks:**
- Breaking change
- More typing for common HTML search case
- Could add alias: `webctl find = webctl html --find`

## Handling CSS Complexity

CSS currently has 4 operations. Proposed consolidation:

```bash
# Extraction (observation)
webctl css                 # All stylesheets → file
webctl css [selector]      # Computed styles → stdout
webctl css [sel] --prop bg # Single property → stdout

# Mutation (keep separate)
webctl css-inject <css>    # Or: webctl css --inject <css>
```

Or keep subcommands but make them consistent:
```bash
webctl css save [selector]     # → file (current)
webctl css get [selector]      # → stdout, all props OR single prop
webctl css inject <css>        # → mutation
```

## Migration Path

1. Add universal flags to all commands
2. Add `--stdout` flag to html/css
3. Add `--output` flag to console/network/cookies
4. Add `--find` flag to all commands
5. Deprecate standalone `find` command with warning
6. Remove after version or two

## Examples: Before vs After

### Before (Current)
```bash
# Searching different things requires different commands
webctl find "login"                    # Search HTML
# No way to search console
# No way to search network  
# No way to search cookies

# Getting data to files is inconsistent
webctl html > file.html                # Doesn't work, outputs path
webctl console > console.txt           # Works
webctl network > network.txt           # Works
```

### After (Proposed)
```bash
# Unified search across all data sources
webctl html --find "login"
webctl css --find "button"
webctl console --find "TypeError"
webctl network --find "api/auth"
webctl cookies --find "session"

# Unified output control
webctl html --stdout                   # Force to stdout
webctl console -o console.json         # Save to file
webctl network -o requests.json        # Save to file
webctl cookies -o cookies.json         # Save to file
```

## Open Questions

1. Should `css [selector]` default to stdout or file?
2. Keep `find` command as alias for `html --find`?
3. Should `--property` be part of `css` or separate `css get` subcommand?
4. What's the max size before stdout fails? Truncate or error?
5. Should we support `--format html|json|text` for flexible output formats?
