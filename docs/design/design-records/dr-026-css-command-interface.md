# DR-026: CSS Command Interface

- Date: 2025-12-28
- Status: Accepted
- Category: CLI

## Problem

The current CSS command implementation has inconsistent output patterns and doesn't follow the unified observation command structure established for other commands. Current issues:

- Four subcommands (save/computed/get/inject) with mixed output destinations
- save uses file output, computed/get use stdout (inconsistent within same command)
- No integration with universal filtering patterns (--find flag)
- Doesn't match the pattern established by html, console, network commands
- Output mode controlled by subcommand choice rather than explicit mode selection

Users need CSS commands to follow the universal observation pattern while preserving CSS-specific operations (computed styles, property retrieval, injection).

## Decision

Redesign CSS command to follow universal observation pattern for stylesheet extraction, with CSS-specific subcommands for unique operations:

```bash
# Universal pattern (stylesheet extraction)
webctl css                  # Output to stdout (Unix convention)
webctl css save [path]      # Save to file (temp if no path, custom if path given)

# Universal flags (apply to universal pattern)
--select, -s SELECTOR       # Filter to element's computed styles
--find, -f TEXT             # Search within CSS
--raw                       # Skip formatting
--json                      # JSON output

# CSS-specific subcommands (unique operations)
webctl css computed <selector>           # All computed styles → stdout
webctl css get <selector> <property>     # Single property → stdout
webctl css inject <css>                  # Inject CSS (mutation)
webctl css inject --file <path>          # Inject from file
```

The universal pattern applies to stylesheet extraction. CSS-specific subcommands provide operations unique to CSS that don't apply to other observation commands.

Complete specification: docs/design/interface/css.md

## Why

Universal Pattern for Stylesheets:

The default/show/save pattern applies to extracting all page stylesheets, providing consistent behavior with html, console, network, and cookies commands. Users get predictable output mode control.

CSS-Specific Subcommands Retained:

The computed, get, and inject operations are unique to CSS and don't map to the universal pattern. These operations have different purposes than stylesheet extraction:

- computed: Quick style inspection for debugging (always stdout)
- get: Single property lookup for scripting (always stdout, plain text)
- inject: Runtime CSS modification (mutation, not observation)

Keeping these as separate subcommands preserves their specialized behavior while maintaining the universal pattern for stylesheet extraction.

Select Flag for Computed Styles:

When --select is used with the universal pattern, it returns computed styles for the selected element instead of all stylesheets. This provides two ways to get computed styles:

1. Universal pattern: webctl css --select ".button" (saves to file)
2. Specific subcommand: webctl css computed ".button" (outputs to stdout)

The flag approach integrates with output modes (can save or show), while the subcommand is a shortcut for stdout output.

Find Flag for CSS Search:

The --find flag enables text search within CSS, matching the pattern established for HTML, console, and network. Users can filter CSS before output, useful for finding specific properties or selectors.

Subcommand vs Flag Decision:

Universal patterns (output modes, filtering) → consistent flags across all commands
Command-specific operations (computed, get, inject) → unique subcommands for CSS only

This principle maintains consistency while allowing specialized functionality where needed.

## Trade-offs

Accept:

- More complex command structure (universal pattern + specific subcommands)
- Two ways to get computed styles (--select flag vs computed subcommand)
- Breaking changes from DR-023 CSS command
- Users must learn both universal pattern and CSS-specific operations
- Implementation complexity for dual behavior modes

Gain:

- Consistent universal pattern across all observation commands
- Preserved CSS-specific operations that users need
- Flexible output modes for stylesheet extraction
- Integrated text search for CSS filtering
- Clear separation: observation (universal pattern) vs operations (subcommands)
- Foundation matches other commands while allowing CSS specialization
- Better discoverability through consistent flags

## Alternatives

Keep DR-023 Design (Four Independent Subcommands):

```bash
webctl css save [selector]
webctl css computed <selector>
webctl css get <selector> <property>
webctl css inject <css>
```

- Pro: No breaking changes from DR-023
- Pro: Each subcommand has clear purpose
- Con: Inconsistent with html/console/network commands
- Con: No universal pattern implementation
- Con: save subcommand doesn't match universal behavior
- Rejected: Fails to achieve cross-command consistency

Flatten All to Flags:

```bash
webctl css [selector]                    # Save to temp
webctl css [selector] --stdout           # Output to stdout
webctl css [selector] --property color   # Get single property
webctl css --inject "..."                # Inject CSS
```

- Pro: Single command with flags
- Pro: Simpler command structure
- Con: Flag overload for different operations
- Con: Mixing observation and mutation in unclear way
- Con: Hard to extend with new operations
- Rejected: Subcommands provide clearer intent separation

Remove Computed/Get Subcommands:

```bash
webctl css                    # Only stylesheet extraction
webctl css --select ".btn"    # Only way to get computed styles
webctl css inject <css>       # Only injection
```

- Pro: Simpler, fewer subcommands
- Pro: Forces universal pattern usage
- Con: Loses convenient computed/get shortcuts
- Con: Less ergonomic for common debugging tasks
- Con: No plain-text single property output
- Rejected: Computed/get shortcuts are valuable for users

Separate Top-Level Commands:

```bash
webctl css [options]                    # Stylesheet extraction
webctl css-computed <selector>          # Computed styles
webctl css-get <selector> <property>    # Single property
webctl css-inject <css>                 # Injection
```

- Pro: Very clear separation
- Pro: No subcommand complexity
- Con: Clutters top-level namespace with four commands
- Con: Less discoverable (no grouping)
- Con: Breaks CSS functionality into unrelated pieces
- Rejected: Subcommands group CSS functionality better

Make Computed/Get Output Modes:

```bash
webctl css show                         # All stylesheets
webctl css computed <selector>          # Computed styles mode
webctl css get <selector> <property>    # Single property mode
```

- Pro: Everything uses similar structure
- Pro: More consistent naming
- Con: Computed/get aren't really "output modes"
- Con: Confuses observation modes with operations
- Con: Doesn't fit the universal pattern semantics
- Rejected: Output modes should be about destination, not operation type

## Structure

Universal Pattern (Stylesheet Extraction):

Default (no subcommand):
- Outputs all page stylesheets to stdout (Unix convention)
- Formatted by default (use --raw for minified)
- Useful for piping to other tools or quick inspection

Save subcommand:
- Optional path argument
- No path: saves to /tmp/webctl-css/ with auto-generated filename
- Directory: auto-generates filename in that directory
- File: saves to exact path
- Creates parent directories if needed

Universal Flags (Apply to Universal Pattern):

--select, -s SELECTOR:
- Changes behavior to return computed styles for element
- Single match: that element's computed styles
- Multiple matches: computed styles for all (with separators)
- Works across all output modes

--find, -f TEXT:
- Search for text within CSS content
- Filters rules/properties containing search text
- Works across all output modes

--raw:
- Skips CSS formatting/pretty-printing
- Returns minified CSS as-is from browser

--json:
- Global flag for JSON output format
- Provides structured data instead of text

CSS-Specific Subcommands:

computed <selector>:
- Returns all computed CSS properties for element
- Always outputs to stdout (not file)
- Text format: property: value (one per line)
- JSON format: {"property": "value", ...}
- Shortcut for quick style inspection

get <selector> <property>:
- Returns single CSS property value
- Always outputs to stdout (not file)
- Plain text output (just the value)
- Scriptable for automation
- Fast property checks

inject <css>:
- Injects CSS into page (runtime modification)
- Inline: webctl css inject "body { background: red; }"
- File: webctl css inject --file ./custom.css
- Returns success confirmation
- Temporary (removed on page reload)

## Usage Examples

Default behavior (stdout):

```bash
webctl css
# body { margin: 0; ... }

webctl css --find "background"
# .header { background: #fff; ... }
# (filtered CSS with background rules)

webctl css --raw
# body{margin:0;padding:0}...
```

Save to file:

```bash
webctl css save
# {"ok": true, "path": "/tmp/webctl-css/25-12-28-143052-example-domain.css"}

webctl css save ./styles.css
# {"ok": true, "path": "./styles.css"}

webctl css save ./output/
# {"ok": true, "path": "./output/25-12-28-143052-example-domain.css"}
```

Computed styles via universal pattern:

```bash
webctl css --select ".button"
# display: flex;
# background-color: rgb(0, 113, 227);
# ...

webctl css save --select ".button"
# {"ok": true, "path": "/tmp/webctl-css/25-12-28-143120-button.css"}
# (computed styles saved to file)

webctl css save ./button-styles.css --select ".button"
# {"ok": true, "path": "./button-styles.css"}
```

CSS-specific subcommands:

```bash
# Quick computed styles inspection
webctl css computed ".button"
# display: flex
# background-color: rgb(0, 113, 227)
# padding: 10px 20px
# ...

# Single property lookup
webctl css get ".button" background-color
# rgb(0, 113, 227)

webctl css get "#header" display
# flex

# Scriptable property check
if [ "$(webctl css get '.modal' display)" = "none" ]; then
  echo "Modal is hidden"
fi

# Inject CSS
webctl css inject "body { background: red !important; }"
# OK

webctl css inject --file ./dark-mode.css
# OK
```

Combining universal flags:

```bash
webctl css --select ".button" --find "hover"
# (computed styles for .button, filtered for "hover")

webctl css save ./results.css --find "media" --raw
# (all CSS with media queries, unformatted)
```

## Breaking Changes

From DR-023 (CSS Command Architecture):

1. Changed: Default behavior now outputs to stdout (Unix convention)
2. Removed: show subcommand (not needed - stdout is default)
3. Changed: save subcommand now takes optional path (temp if no path)
4. Added: --select flag for computed styles (alternative to computed subcommand)
5. Added: --find flag for CSS text search
6. Added: --raw flag for unformatted output
7. Retained: computed, get, inject subcommands (behavior unchanged)

Migration Guide:

Old pattern (DR-023):
```bash
webctl css save                      # All stylesheets to temp
webctl css save -o ./styles.css      # All stylesheets to custom path
webctl css save "#header"            # Computed styles to file
webctl css computed ".button"        # Computed styles to stdout
webctl css get ".button" color       # Single property to stdout
webctl css inject "..."              # Inject CSS
```

New pattern (DR-026 after P-051):
```bash
webctl css                           # Output to stdout (changed)
webctl css save                      # Save to temp (changed)
webctl css save ./styles.css         # Save to custom path (changed)
webctl css --select "#header"        # Computed styles to stdout (changed)
webctl css computed ".button"        # Computed styles to stdout (same)
webctl css get ".button" color       # Single property to stdout (same)
webctl css inject "..."              # Inject CSS (same)
```

## Updates

- 2026-01-09: Updated to stdout default, removed show subcommand (P-051)
- 2025-12-28: Initial version (supersedes DR-023)
