# DR-025: HTML Command Interface

- Date: 2025-12-28
- Status: Accepted
- Category: CLI

## Problem

The current HTML command implementation is inconsistent with other observation commands, creating confusion and limiting functionality. Current issues:

- Always outputs to file (no stdout option for quick inspection)
- No text search/filtering capability
- Inconsistent with other observation commands (console, network)
- Missing universal patterns that users expect
- No integrated find functionality (requires separate find command)

Users need a unified interface across all observation commands with consistent output modes, filtering capabilities, and predictable behavior patterns.

## Decision

Redesign HTML command to follow Unix convention (stdout by default):

```bash
# Pattern
webctl html                # Output to stdout (Unix convention)
webctl html save [path]    # Save to file (temp if no path, custom if path given)

# Universal flags
--select, -s SELECTOR      # Filter to element(s)
--find, -f TEXT            # Search within HTML
--before, -B N             # Show N lines before each match (requires --find)
--after, -A N              # Show N lines after each match (requires --find)
--context, -C N            # Show N lines before and after each match (requires --find)
--raw                      # Skip formatting
--json                     # JSON output
```

The HTML command outputs to stdout by default (Unix convention), with a save subcommand for file output. The `show` subcommand is not needed - stdout is the default.

Complete specification: .ai/design/interface/html.md

## Why

Unix Convention (stdout by default):

Following Unix philosophy, observation commands output to stdout by default. This enables:
- Piping to other tools (grep, less, jq)
- Quick inspection without file management
- Consistent with standard CLI tools

Save Subcommand for Files:

When file output is needed, the save subcommand provides flexibility:
- `html save` - saves to temp directory with auto-generated filename
- `html save ./page.html` - saves to custom path
- Directory paths auto-generate filenames, file paths use exact names

Universal Flags for Filtering:

The --select and --find flags provide filtering capabilities that work across all output modes. Users can filter HTML before output, whether saving to file or showing on stdout. This eliminates the need for separate find command.

No HTML-Specific Subcommands:

HTML extraction doesn't require special operations like CSS does (computed/get/inject). The universal pattern covers all HTML use cases. Keeping the interface minimal reduces complexity.

Text Search Integration:

The --find flag replaces the standalone find command for HTML search. Having search as a flag rather than a separate command is more discoverable and consistent with filtering patterns in console and network commands.

## Trade-offs

Accept:

- Breaking change from current html command behavior
- Three-way output mode choice (default/show/save) adds complexity
- Temp files require eventual cleanup
- Users must learn new subcommand structure
- Standalone find command will be removed
- Migration required for existing scripts

Gain:

- Consistent interface across all observation commands
- Predictable behavior pattern (learn once, use everywhere)
- Integrated text search without separate command
- Flexible output modes for different use cases
- Better discoverability (flags show filtering options)
- Cleaner command structure (no special cases)
- Foundation for future observation commands

## Alternatives

Keep Current File-Only Behavior:

```bash
webctl html [selector]    # Always saves to file
```

- Pro: No breaking changes, existing scripts work
- Pro: Simple single behavior
- Con: Inconsistent with console/network commands
- Con: No stdout option for quick inspection
- Con: No filtering capabilities
- Con: Doesn't address core consistency problem
- Rejected: Fails to solve the fundamental inconsistency issue

Add Stdout Flag to Current Design:

```bash
webctl html [selector]           # Save to file
webctl html [selector] --stdout  # Output to stdout
```

- Pro: Minimal change from current behavior
- Pro: Adds stdout capability
- Con: Still inconsistent with other commands
- Con: Flags for output modes less clear than subcommands
- Con: Doesn't establish universal pattern
- Rejected: Partial solution that doesn't achieve consistency goals

Mirror Console Command Exactly:

```bash
webctl html                 # Always stdout
webctl html --output <path> # Save to file
```

- Pro: Matches current console pattern
- Pro: Simple stdout-first design
- Con: Large HTML to stdout by default is problematic
- Con: Different from screenshot (file-first) pattern
- Rejected: HTML is more similar to screenshot (large output) than console

Separate Commands for Different Output Modes:

```bash
webctl html-save [selector]   # Save to file
webctl html-show [selector]   # Output to stdout
```

- Pro: Very explicit intent
- Pro: No subcommand complexity
- Con: Clutters command namespace
- Con: Two commands instead of one
- Con: Less discoverable
- Rejected: Subcommands group related functionality better

Use Output Flag Instead of Save Subcommand:

```bash
webctl html               # Save to temp
webctl html show          # Output to stdout
webctl html -o <path>     # Save to custom path
```

- Pro: Shorter for custom paths
- Pro: Matches some CLI conventions
- Con: Inconsistent flag vs subcommand mixing
- Con: -o could be confused with global flags
- Con: Less clear than explicit save subcommand
- Rejected: save <path> pattern is clearer and more consistent

## Structure

Output Modes:

Default (no subcommand):
- Outputs HTML to stdout (Unix convention)
- Useful for piping to other tools
- Quick inspection without file management
- Works with --select for focused output

Save subcommand:
- Optional path argument
- No path: saves to /tmp/webctl-html/ with auto-generated filename
- Path with trailing slash (path/): auto-generates filename in that directory
- Path without trailing slash (path): saves to exact file path
- Creates parent directories if needed
- Trailing slash convention follows Unix tools like rsync

Universal Flags:

--select, -s SELECTOR:
- Filter to specific element(s) matching CSS selector
- Single match: returns that element's outer HTML
- Multiple matches: returns all with HTML comment separators
- Works across all output modes

--find, -f TEXT:
- Search for text within HTML
- Filters/highlights matches
- Replaces standalone find command
- Works across all output modes
- Case-insensitive matching

--before, -B N:
- Show N lines before each matching line
- Requires --find flag
- Similar to ripgrep's -B flag

--after, -A N:
- Show N lines after each matching line
- Requires --find flag
- Similar to ripgrep's -A flag

--context, -C N:
- Show N lines before and after each matching line
- Requires --find flag
- Equivalent to -B N -A N
- Similar to ripgrep's -C flag
- Overlapping context regions are merged

--raw:
- Skips HTML formatting/pretty-printing
- Returns HTML exactly as received from browser
- Useful for exact byte-for-byte output

--json:
- Global flag for JSON output format
- Provides structured data instead of text

## Path Handling

Auto-generated Filenames:

Pattern: /tmp/webctl-html/YY-MM-DD-HHMMSS-{identifier}.html

Identifier selection:
- Full page: sanitized page title or domain
- With selector: sanitized selector or element identifier
- Falls back to "page" if no suitable identifier

Example filenames:
- 25-12-28-143052-example-domain.html
- 25-12-28-143115-main.html
- 25-12-28-143120-navigation.html

Directory vs File Paths (Trailing Slash Convention):

Trailing slash convention follows Unix tools like rsync:

If path ends with `/`:
- Treated as directory
- Auto-generates filename using pattern above
- Creates directory if it doesn't exist
- Example: webctl html save ./output/ → ./output/25-12-28-143052-example-domain.html

If path does NOT end with `/`:
- Treated as file path
- Uses exact path as-is
- Example: webctl html save ./page.html → ./page.html
- Example: webctl html save ./output → ./output (creates file named "output")

This makes behavior predictable and independent of filesystem state.

## Usage Examples

Default behavior (stdout):

```bash
webctl html
# <html>...</html>

webctl html --select "#main"
# <div id="main">...</div>

webctl html --find "login"
# Lines containing "login"

webctl html --find "login" -C 3
# Lines containing "login" with 3 lines of context

webctl html --find "error" -B 5 -A 2
# Lines containing "error" with 5 lines before and 2 after
```

Save to file:

```bash
webctl html save
# /tmp/webctl-html/25-12-28-143052-example-domain.html

webctl html save ./page.html
# ./page.html

webctl html save ./output/
# ./output/25-12-28-143052-example-domain.html

webctl html save ./debug/content.html --select ".content"
# ./debug/content.html
```

Combining flags:

```bash
webctl html --select "form" --find "password"
# Forms containing "password" (to stdout)

webctl html save --select ".card" --find "product"
# All product cards saved to temp

webctl html save ./results.html --select "#results" --raw
# Raw HTML saved to custom path
```

## Breaking Changes

From DR-012 (HTML Command Interface):

1. Changed: Default behavior now outputs to stdout (Unix convention)
2. Removed: show subcommand (not needed - stdout is default)
3. Changed: save subcommand now takes optional path (temp if no path)
4. Added: --find flag for text search (integrates find command)
5. Added: --raw flag for unformatted output

From DR-021 (HTML Formatting Find):

1. Integrated: Find functionality moved to --find flag
2. Changed: Standalone find command being removed (see DR-030)

Migration Guide:

Old pattern (DR-012):
```bash
webctl html                    # Save to temp
webctl html -o ./page.html     # Save to custom path
webctl html "#main"            # Save element to temp
```

New pattern (DR-025 after P-051):
```bash
webctl html                    # Output to stdout (changed)
webctl html save               # Save to temp (changed)
webctl html save ./page.html   # Save to custom path (changed)
webctl html --select "#main"   # Output element to stdout (changed)
```

Old pattern (find command):
```bash
webctl find "login"            # Search HTML for text
```

New pattern (--find flag):
```bash
webctl html --find "login"     # Search HTML for text (to stdout)
webctl html save --find "login"  # Save matching lines to temp
```

## Updates

- 2026-01-12: Added context flags (-A, -B, -C) for --find (P-034)
- 2025-01-09: Updated to stdout default, removed show subcommand (P-051)
- 2025-12-28: Initial version (supersedes DR-012 and DR-021)
