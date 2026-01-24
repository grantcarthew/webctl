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

## Current State

### Existing Implementation

IPC Protocol (internal/ipc/protocol.go:137-140):
- HTMLData has single HTML field (string) - no metadata support
- CSSData has ComputedMulti ([]map[string]string) - styles only, no metadata
- CSSData has Inline ([]string) - style attributes only, no metadata
- No ElementMeta or element identification structures exist

Daemon Handlers:
- handlers_css.go:104-169 handleCSSComputed returns []map[string]string
- handlers_css.go:256-310 handleCSSInline returns []string
- handlers_observation.go:307-382 handleHTML returns joined HTML with "--" separators

CLI Formatters (internal/cli/format/css.go):
- ComputedStylesMulti (line 23-37) outputs with "--" separators, no element IDs
- InlineStyles (line 46-64) outputs with "--" separators, no element IDs

CLI Commands:
- css.go:394-448 runCSSComputed calls ComputedStylesMulti formatter
- css.go:510-575 runCSSInline calls InlineStyles formatter
- html.go handles multi-element output with "--" separators from daemon

Tests:
- scripts/test/cli/test-observation.sh has tests for multi-element queries
- Tests verify presence of content but not element identification
- Tests at lines 54, 60, 140, 150 check multi-element CSS/HTML output

### Work Required

1. Protocol Changes (internal/ipc/protocol.go):
   - Add ElementMeta struct with Tag, ID, Class fields
   - Add ElementWithStyles struct combining ElementMeta + styles
   - Add ElementWithHTML struct combining ElementMeta + HTML
   - Update CSSData to include InlineMulti and ComputedMulti with metadata
   - Update HTMLData to include HTMLMulti with metadata

2. Daemon Handler Changes (internal/daemon/):
   - handlers_css.go handleCSSInline: Update JavaScript to extract tag/id/class, return ElementWithStyles array
   - handlers_css.go handleCSSComputed: Update JavaScript to extract tag/id/class, return ElementWithStyles array
   - handlers_observation.go handleHTML: Update JavaScript to extract tag/id/class, return ElementWithHTML array

3. CLI Formatter Changes (internal/cli/format/css.go):
   - Add formatElementIdentifier function (#id, .class:N, tag:N format)
   - Update ComputedStylesMulti to accept ElementWithStyles and output identifiers
   - Update InlineStyles to accept ElementWithStyles and output identifiers

4. CLI Command Changes:
   - css.go runCSSComputed: Use new ElementWithStyles from protocol
   - css.go runCSSInline: Use new ElementWithStyles from protocol
   - html.go: Add formatting for ElementWithHTML with identifiers

5. Test Updates:
   - Update test-observation.sh to verify element identification format
   - Add tests for edge cases (empty attrs, multiple classes, special chars)

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
- SVG className handling: Use el.getAttribute('class') for all elements (consistent across HTML/SVG)
- Special characters in id/class: Strip invalid characters, fall back to tag name if empty after stripping
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
- Strip invalid characters (keep only alphanumeric, hyphens, underscores)
- Fall back to tag name if result is empty after stripping
- Token-efficient and handles edge cases gracefully

### JavaScript Changes

CSS inline/computed handlers:
```javascript
const elements = document.querySelectorAll(selector);
return Array.from(elements).map((el, idx) => {
  // Get id (trim and check for non-empty)
  const id = (el.id || '').trim();

  // Get first class using getAttribute (works for HTML and SVG)
  const classAttr = el.getAttribute('class');
  const classes = (classAttr || '')
    .split(/\s+/)
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

  // Use getAttribute for consistent handling across HTML and SVG
  const classAttr = el.getAttribute('class');
  const classes = (classAttr || '')
    .split(/\s+/)
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
        // Sanitize ID: strip invalid CSS identifier characters
        sanitized := sanitizeIdentifier(meta.ID)
        if sanitized != "" {
            return "#" + sanitized
        }
    }
    if meta.Class != "" {
        // Sanitize class: strip invalid CSS identifier characters
        sanitized := sanitizeIdentifier(meta.Class)
        if sanitized != "" {
            return fmt.Sprintf(".%s:%d", sanitized, index+1)
        }
    }
    return fmt.Sprintf("%s:%d", meta.Tag, index+1)
}

// sanitizeIdentifier removes invalid CSS identifier characters.
// Keeps alphanumeric, hyphens, and underscores. Returns empty string if nothing remains.
func sanitizeIdentifier(s string) string {
    // Remove characters not valid in CSS identifiers
    var result strings.Builder
    for _, r := range s {
        if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
           (r >= '0' && r <= '9') || r == '-' || r == '_' {
            result.WriteRune(r)
        }
    }
    return result.String()
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
