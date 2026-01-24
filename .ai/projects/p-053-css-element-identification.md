# p-053: CSS Element Identification

- Status: In Progress
- Started: 2026-01-24

## Overview

When CSS commands return results for multiple elements, there is no way to identify which element each result corresponds to. The output shows values separated by `--` markers but no element identification.

Example current output for `css inline "div"`:
```
(empty)
--
--active-panel-height: 0px;
--
(empty)
```

Users cannot determine which div has which inline style.

## Goals

1. Add element identification to multi-element CSS output
2. Apply consistent identification across all CSS subcommands that support multiple elements
3. Keep output concise and readable

## Scope

In Scope:
- `css inline <selector>` - multiple elements
- `css computed <selector>` - multiple elements
- `html --select <selector>` - multiple elements (also uses querySelectorAll with -- separators)
- `css matched <selector>` - single element (no change needed)
- Daemon-side changes to return element identification
- CLI formatter changes to display identification

Out of Scope:
- Other observation commands (console, network, cookies - don't return per-element data)
- JSON output format changes (only if not already structured with metadata)

## Success Criteria

- [ ] Multi-element CSS output includes element identification (#id, .class:N, or tag:N)
- [ ] Multi-element HTML output includes element identification
- [ ] Identification follows token-optimized format (no brackets, CSS selector notation)
- [ ] IPC protocol updated with ElementMeta struct and new response fields
- [ ] Text output shows identifiers on separate line before each element's data
- [ ] JSON output includes structured metadata (tag, id, class fields)
- [ ] Edge cases handled correctly (empty attrs, special chars, multiple classes)
- [ ] All affected commands updated: css inline, css computed, html --select
- [ ] Tests updated to verify element identification format
- [ ] Output remains readable and uncluttered

## Deliverables

- Updated `internal/daemon/handlers_css.go` - return element identification for inline/computed
- Updated `internal/daemon/handlers_observation.go` - return element identification for HTML queries
- Updated `internal/ipc/protocol.go` - add identification fields (ElementMeta struct)
- Updated `internal/cli/format/css.go` - display identification
- Updated `internal/cli/html.go` or formatter - display identification for multi-element HTML
- Updated command help text if output format changes
- Updated tests to verify element identification

## Technical Approach

Element identification format (token-optimized):

Identification strategy:
- If element has id attribute: `#id` (unique, no index needed)
- Else if element has class attribute: `.class:N` (first class, with index)
- Else: `tag:N` (tag name with index)

Index is 1-based, counting within the result set.

Example output:
```
#header
--active-header: true;
--
.panel:1
(empty)
--
.panel:2
--active-panel-height: 0px;
--
div:1
(empty)
```

Format rationale:
- No brackets - saves 2 tokens per element
- CSS selector notation - familiar and concise
- IDs don't need index - they're unique by definition
- Classes get index - multiple elements may share the class
- Plain tags get index - generic identification

## Decisions

- Include class/id if available: YES
- Format notation: Plain CSS selector notation (no brackets, most token-efficient)
- JSON output: Structured metadata fields only (not formatted strings)
- Multiple classes: Use first class only (token-efficient)
- Empty/whitespace attributes: Treat as missing, fall back to tag name
- Special characters: Use CSS.escape() for safe output
- Backward compatibility: Breaking IPC protocol changes acceptable (pre-1.0)
- Implementation approach: Implement first, then update tests

## Implementation Details

### Edge Case Handling

Multiple classes:
- Element with `class="panel active disabled"` → `.panel:N` (first class only)
- Reduces token usage, most classes are ordered by importance

Empty attributes:
- `id=""` or `class="   "` (whitespace only) → treated as missing
- Fall back to tag name: `tag:N`

Special characters in id/class:
- Use JavaScript `CSS.escape()` to safely escape special characters
- Example: `id="my:weird-id"` → `#my\:weird-id` (but rare in practice)
- Alternative: Strip invalid characters and fall back to tag if empty

### JavaScript Changes

CSS inline/computed handlers:
```javascript
const elements = document.querySelectorAll(selector);
return Array.from(elements).map((el, idx) => {
  // Get id (trim and check for non-empty)
  const id = (el.id || '').trim();

  // Get first class (split, filter empty, take first)
  const classes = (el.className || '')
    .split(' ')
    .map(c => c.trim())
    .filter(c => c.length > 0);
  const firstClass = classes.length > 0 ? classes[0] : null;

  return {
    tag: el.tagName.toLowerCase(),
    id: id || null,
    class: firstClass,
    styles: /* inline: el.getAttribute('style') || computed styles */
  };
});
```

HTML handler:
```javascript
const elements = document.querySelectorAll(selector);
return Array.from(elements).map((el, idx) => {
  const id = (el.id || '').trim();
  const classes = (el.className || '')
    .split(' ')
    .map(c => c.trim())
    .filter(c => c.length > 0);
  const firstClass = classes.length > 0 ? classes[0] : null;

  return {
    tag: el.tagName.toLowerCase(),
    id: id || null,
    class: firstClass,
    html: el.outerHTML
  };
});
```

### IPC Protocol Changes

New struct in protocol.go:
```go
// ElementMeta contains element identification metadata
type ElementMeta struct {
    Tag   string  `json:"tag"`            // lowercase tag name (div, span, etc)
    ID    string  `json:"id,omitempty"`   // id attribute value (if present)
    Class string  `json:"class,omitempty"` // first class name (if present)
}

// ElementWithStyles combines element metadata with styles
type ElementWithStyles struct {
    ElementMeta
    Styles map[string]string `json:"styles"` // for computed
    Inline string            `json:"inline"` // for inline
}

// ElementWithHTML combines element metadata with HTML
type ElementWithHTML struct {
    ElementMeta
    HTML string `json:"html"`
}
```

Update CSSData:
```go
type CSSData struct {
    CSS           string              `json:"css,omitempty"`
    Styles        map[string]string   `json:"styles,omitempty"`        // deprecated: single element
    ComputedMulti []ElementWithStyles `json:"computedMulti,omitempty"` // NEW: with metadata
    Value         string              `json:"value,omitempty"`
    InlineMulti   []ElementWithStyles `json:"inlineMulti,omitempty"`   // NEW: with metadata
    Inline        []string            `json:"inline,omitempty"`        // deprecated: backward compat
    Matched       []CSSMatchedRule    `json:"matched,omitempty"`
}
```

Update HTMLData:
```go
type HTMLData struct {
    HTML         string            `json:"html,omitempty"`         // single result or legacy
    HTMLMulti    []ElementWithHTML `json:"htmlMulti,omitempty"`    // NEW: multi-element with metadata
}
```

### CLI Formatter Logic

Text output format:
```go
func formatElementIdentifier(meta ElementMeta, index int) string {
    if meta.ID != "" {
        return "#" + meta.ID
    }
    if meta.Class != "" {
        return fmt.Sprintf(".%s:%d", meta.Class, index+1)
    }
    return fmt.Sprintf("%s:%d", meta.Tag, index+1)
}
```

JSON output format:
- Keep structured metadata as-is
- Do NOT add formatted identifier strings to JSON
- JSON consumers can format identifiers themselves if needed

### Implementation Order

1. Update protocol.go with new structs (ElementMeta, ElementWithStyles, ElementWithHTML)
2. Update handlers_css.go:
   - handleCSSInline: Return ElementWithStyles array with metadata
   - handleCSSComputed: Return ElementWithStyles array with metadata
3. Update handlers_observation.go:
   - handleHTML: Return ElementWithHTML array for multi-element queries
4. Update format/css.go:
   - ComputedStylesMulti: Accept ElementWithStyles, output with identifiers
   - InlineStyles: Accept ElementWithStyles, output with identifiers
5. Update html.go formatter or create new function for multi-element HTML
6. Update CLI commands to use new protocol fields
7. Update tests in scripts/test/cli/ to verify element identification
8. Update interactive test scripts if they check exact output
