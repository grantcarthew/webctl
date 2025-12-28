# CSS Command Design - LOCKED

## Universal Pattern

```bash
# Default: save all stylesheets to temp file
webctl css
# → /tmp/webctl-css/25-12-28-HHMMSS-page-title.css

# Show: output all stylesheets to stdout
webctl css show

# Save: save all stylesheets to custom path
webctl css save <path>
# If <path> is a directory, auto-generate filename
webctl css save ./output/
# → ./output/25-12-28-HHMMSS-page-title.css
```

## Universal Flags (apply to default, show, save)

```bash
--select, -s SELECTOR    # Filter to element's computed styles
--find, -f TEXT          # Search within CSS
--raw                    # Skip formatting/pretty-printing
--json                   # JSON output (global flag)
```

## Examples with Flags

```bash
# Get computed styles for element
webctl css --select ".button"
# → /tmp/webctl-css/25-12-28-HHMMSS-button.css (computed styles)

webctl css show --select ".button"
# → stdout (computed styles)

webctl css save ./button.css --select ".button"
# → ./button.css (computed styles)

# Search within CSS
webctl css --find "background"
# → /tmp/webctl-css/... (filtered CSS with matches)

webctl css show --find "background"
# → stdout (filtered CSS with matches)

# Combine filters
webctl css show --select ".button" --find "hover"
# → stdout (computed styles for .button, filtered for "hover")
```

## CSS-Specific Subcommands

These are unique operations that don't apply to other commands.

```bash
# Get all computed styles for element (always stdout)
webctl css computed <selector>
# Example: webctl css computed ".button"
# Output: All CSS properties for .button

# Get single property value (always stdout, plain text)
webctl css get <selector> <property>
# Example: webctl css get ".button" background-color
# Output: rgb(0, 113, 227)

# Inject CSS into page (mutation)
webctl css inject <css>
webctl css inject --file <path>
# Example: webctl css inject "body { background: red; }"
```

## Design Rationale

**Universal pattern:**
- Consistent with html, console, network, cookies
- Default saves to temp (most common use case)
- `show` for stdout, `save <path>` for custom location
- Filters work across all output modes

**CSS-specific subcommands:**
- `computed` - Quick inspection of element styles
- `get` - Scriptable single property lookup
- `inject` - Runtime CSS modification
- These operations are unique to CSS and don't apply to other commands

**Principle:**
- Universal patterns → consistent across commands
- Command-specific operations → unique subcommands
