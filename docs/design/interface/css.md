# CSS Command Design - LOCKED

## Universal Pattern

```bash
# Default: output all stylesheets to stdout
webctl css

# Save: save all stylesheets to file
webctl css save           # Save to temp file
webctl css save <path>    # Save to custom path

# Path conventions (trailing slash required for directories):
webctl css save ./styles.css   # File: saves to ./styles.css
webctl css save ./output/      # Directory: auto-generates filename
# → ./output/25-12-28-HHMMSS-page-title.css
webctl css save ./output       # File: saves to ./output (not a directory!)
```

## Universal Flags (apply to default and save)

```bash
--select, -s SELECTOR    # Filter to element's computed styles
--find, -f TEXT          # Search within CSS
--raw                    # Skip formatting/pretty-printing
--json                   # JSON output (global flag)
```

## Examples with Flags

```bash
# Get computed styles for element to stdout
webctl css --select ".button"

# Save computed styles to file
webctl css save ./button.css --select ".button"
# → ./button.css (computed styles)

# Search within CSS
webctl css --find "background"
# → stdout (filtered CSS with matches)

webctl css save --find "background"
# → /tmp/webctl-css/... (filtered CSS with matches)

# Combine filters
webctl css --select ".button" --find "hover"
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
# NOTE: NOT IMPLEMENTED - Removed in 2025-12-29
# Reason: Used non-existent CDP method (Page.addStyleTag).
# Alternative: Use `webctl eval` with JavaScript to inject styles:
#   webctl eval "const s=document.createElement('style');s.textContent='...';document.head.appendChild(s)"
webctl css inject <css>
webctl css inject --file <path>
# Example: webctl css inject "body { background: red; }"
```

## Design Rationale

**Universal pattern:**
- Consistent with html, console, network, cookies
- Default outputs to stdout (Unix convention)
- `save` for file output (temp or custom path)
- Filters work across all output modes

**CSS-specific subcommands:**
- `computed` - Quick inspection of element styles
- `get` - Scriptable single property lookup
- `inject` - ~~Runtime CSS modification~~ NOT IMPLEMENTED (see note above)
- These operations are unique to CSS and don't apply to other commands

**Principle:**
- Universal patterns → consistent across commands
- Command-specific operations → unique subcommands
