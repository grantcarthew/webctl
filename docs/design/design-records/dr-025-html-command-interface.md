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

Redesign HTML command to follow the universal observation command pattern:

```bash
# Universal pattern
webctl html                # Save to temp with auto-generated name
webctl html show           # Output to stdout
webctl html save <path>    # Save to custom path

# Universal flags
--select, -s SELECTOR      # Filter to element(s)
--find, -f TEXT            # Search within HTML
--raw                      # Skip formatting
--json                     # JSON output
```

The HTML command uses ONLY the universal pattern with no HTML-specific subcommands. All functionality is provided through the base pattern and universal flags.

Complete specification: docs/design/interface/html.md

## Why

Universal Pattern Consistency:

The universal pattern (default/show/save) provides predictable behavior across all observation commands. Users learn once, apply everywhere. This reduces cognitive load and makes the CLI more intuitive.

Default to Temp File:

Saving to temp by default preserves output for later analysis while keeping stdout clean. The temp file location is returned in JSON response, allowing users to read the file when needed. This matches screenshot command behavior.

Show Subcommand for stdout:

Explicit show subcommand makes intent clear. Users who want stdout output request it explicitly. This prevents accidental flooding of terminal with large HTML documents while still allowing quick inspection when desired.

Save Subcommand with Path Argument:

Using save <path> instead of --output flag makes the command structure more natural and consistent. The path argument is required, making it clear that users must specify where to save. Directory paths auto-generate filenames, file paths use exact names.

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
- Saves full page HTML to /tmp/webctl-html/
- Auto-generates filename: YY-MM-DD-HHMMSS-{title}.html
- Returns JSON with file path
- Most common use case for large HTML documents

Show subcommand:
- Outputs HTML to stdout
- Useful for piping to other tools
- Quick inspection without file management
- Works with --select for focused output

Save subcommand:
- Requires path argument
- Directory: auto-generates filename in that directory
- File: saves to exact path
- Creates parent directories if needed

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

Directory vs File Paths:

If path is directory:
- Auto-generate filename using pattern above
- Save in specified directory
- Example: webctl html save ./output/ → ./output/25-12-28-143052-example-domain.html

If path is file:
- Use exact path as-is
- Example: webctl html save ./page.html → ./page.html

## Usage Examples

Default behavior (save to temp):

```bash
webctl html
# {"ok": true, "path": "/tmp/webctl-html/25-12-28-143052-example-domain.html"}

webctl html --select "#main"
# {"ok": true, "path": "/tmp/webctl-html/25-12-28-143115-main.html"}
```

Show to stdout:

```bash
webctl html show
# <html>...</html>

webctl html show --select ".content"
# <div class="content">...</div>

webctl html show --find "login"
# <html>...<form class="login">...</html>
```

Save to custom path:

```bash
webctl html save ./page.html
# {"ok": true, "path": "./page.html"}

webctl html save ./output/
# {"ok": true, "path": "./output/25-12-28-143052-example-domain.html"}

webctl html save ./debug/content.html --select ".content"
# {"ok": true, "path": "./debug/content.html"}
```

Combining flags:

```bash
webctl html show --select "form" --find "password"
# Forms containing "password"

webctl html --select ".card" --find "product"
# All product cards saved to temp

webctl html save ./results.html --select "#results" --raw
# Raw HTML saved to custom path
```

## Breaking Changes

From DR-012 (HTML Command Interface):

1. Changed: Default behavior now saves to temp instead of requiring path
2. Added: show subcommand for stdout output
3. Added: save subcommand for explicit path specification
4. Added: --find flag for text search (integrates find command)
5. Changed: --output flag replaced by save <path> subcommand
6. Added: --raw flag for unformatted output

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

New pattern (DR-025):
```bash
webctl html                    # Save to temp (same)
webctl html save ./page.html   # Save to custom path (changed)
webctl html --select "#main"   # Save element to temp (changed)
```

Old pattern (find command):
```bash
webctl find "login"            # Search HTML for text
```

New pattern (--find flag):
```bash
webctl html --find "login"     # Search HTML for text
webctl html show --find "login"  # Show results on stdout
```

## Updates

- 2025-12-28: Initial version (supersedes DR-012 and DR-021)
