# HTML Command Design - LOCKED

## Universal Pattern

```bash
# Default: save full page to temp file
webctl html
# → /tmp/webctl-html/25-12-28-HHMMSS-page-title.html

# Show: output full page to stdout
webctl html show

# Save: save full page to custom path
webctl html save <path>
# If <path> is a directory, auto-generate filename
webctl html save ./output/
# → ./output/25-12-28-HHMMSS-page-title.html
```

## Universal Flags

```bash
--select, -s SELECTOR    # Filter to element(s)
--find, -f TEXT          # Search within HTML
--raw                    # Skip formatting/pretty-printing
--json                   # JSON output (global flag)
```

## Examples with Flags

```bash
# Get specific element
webctl html --select "#main"
# → /tmp/webctl-html/25-12-28-HHMMSS-main.html

webctl html show --select "#main"
# → stdout (just #main element)

webctl html save ./main.html --select "#main"
# → ./main.html (just #main element)

# Search within HTML
webctl html --find "login"
# → /tmp/webctl-html/... (HTML with matches highlighted/filtered)

webctl html show --find "login"
# → stdout (HTML with matches)

# Combine filters
webctl html show --select "form" --find "password"
# → stdout (forms containing "password")

# Multiple elements (selector matches multiple)
webctl html --select ".card"
# → /tmp/webctl-html/... (all .card elements with separators)
```

## Output Format

**Without selector:**
- Full page HTML (document.documentElement)
- Formatted by default, use `--raw` for unformatted

**With selector (single match):**
- Outer HTML of matched element

**With selector (multiple matches):**
- All matched elements with HTML comment separators:
```html
<!-- Element 1 of 3: .card -->
<div class="card">...</div>

<!-- Element 2 of 3: .card -->
<div class="card">...</div>

<!-- Element 3 of 3: .card -->
<div class="card">...</div>
```

**With find:**
- Filtered/highlighted matches (format TBD)

## HTML-Specific Subcommands

None. HTML uses only the universal pattern.

## Design Rationale

**Universal pattern only:**
- HTML extraction doesn't need special operations like CSS does
- The universal pattern (default/show/save) covers all HTML use cases
- Filtering via `--select` and `--find` provides necessary control

**No inject subcommand:**
- Unlike CSS, injecting HTML is complex and dangerous (XSS, DOM manipulation)
- Can be done via `eval` command if needed: `webctl eval "document.body.innerHTML = '...'"`
- Not common enough to warrant a dedicated subcommand
