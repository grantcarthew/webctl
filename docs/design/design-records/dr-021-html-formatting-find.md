# DR-021: HTML Formatting for Find Command

- Date: 2025-12-25
- Status: Proposed
- Category: CLI

## Problem

The find command searches raw HTML by splitting on newlines and matching patterns. This works well for formatted HTML, but modern web frameworks often output minified or single-line HTML where multiple elements are concatenated without line breaks.

When searching minified HTML:

- Match lines are extremely long (thousands of characters)
- Context lines (before/after) are equally long
- Output is unreadable
- Cannot identify which specific element contains the match
- Selector generation is unreliable

Example minified HTML:

```html
<!DOCTYPE html><html><head><title>App</title></head><body><div class="container"><button class="btn-primary">Sign up</button><p>Terms apply</p></div></body></html>
```

Searching for "Sign up" returns the entire body as one line.

Requirements:

- Make find output readable for minified HTML
- Preserve ability to search all HTML content (tags, attributes, text)
- Maintain accurate selector generation
- Keep implementation simple and maintainable

## Decision

Implement custom HTML pretty-printer in webctl daemon (approximately 150-200 lines) that formats HTML before find command searches it.

Implementation:

1. Add `internal/daemon/htmlformat.go` with `Format(html string) string` function
2. Use `golang.org/x/net/html` tokenizer (standard library package)
3. Format output with 2-space indentation
4. Preserve raw content in `<pre>` and `<textarea>` tags
5. Update `handleFind()` to format HTML before searching

The formatter runs inside the daemon, so formatted HTML is never exposed to the user (only search results are returned).

## Why

Custom implementation vs third-party library:

We evaluated `github.com/yosssi/gohtml` (519 lines, MIT licensed):

- Well-structured, uses `golang.org/x/net/html`
- Good test coverage (535 lines of tests)
- But: Last commit October 2020 (5+ years unmaintained)
- Has 4 open issues, no recent maintenance

Building our own provides:

- Control over code (can fix bugs, add features)
- No dependency on unmaintained library
- Simpler implementation for our specific needs (approximately 150-200 lines vs 519)
- Can optimize for daemon context (no need for Writer wrapper, configurable options, inline tag condensing)

We use gohtml source as implementation guide:

- Parser approach using tokenizer
- Handling of raw/preformatted tags
- Indentation strategy

But simplify by removing:

- Writer wrapper
- Inline tag condensing logic
- Configurable indent strings
- Line number formatting

## Trade-offs

Accept:

- Must maintain HTML formatting code (approximately 150-200 lines)
- Need to handle edge cases in HTML parsing
- Formatting adds processing time to find command
- Must write tests for formatter

Gain:

- Find command works with minified HTML
- Readable output shows structure clearly
- No unmaintained dependency
- Can customize formatting for our needs
- Full control over implementation

## Alternatives

Use gohtml library directly:

- Pro: Ready to use, tested implementation
- Pro: Handles edge cases
- Con: Unmaintained for 5+ years
- Con: 519 lines for features we don't need
- Rejected: Taking dependency on unmaintained code risks future compatibility

Keep current line-based search, truncate long lines:

- Pro: Simplest change (approximately 20 lines)
- Pro: No HTML parsing needed
- Con: Truncation may cut important context
- Con: Still hard to read minified HTML
- Con: Doesn't solve root problem
- Rejected: Band-aid solution, doesn't provide good UX

Search using DOM tree traversal instead of text:

- Pro: Most accurate element matching
- Pro: Perfect selector generation
- Con: Cannot search attributes/classes by text
- Con: Complex implementation
- Con: Changes search semantics (DR-017 specifies raw HTML search)
- Rejected: Violates design decision in DR-017

## Implementation Notes

File structure:

Location: `internal/daemon/htmlformat.go`

Core function:

```
Format(html string) string
```

Algorithm:

1. Parse HTML using `html.NewTokenizer(strings.NewReader(html))`
2. Track indentation level (starts at 0)
3. For each token:
   - StartTagToken: Write opening tag with indent, increase level
   - EndTagToken: Decrease level, write closing tag with indent
   - TextToken: Write trimmed text (skip if whitespace-only)
   - SelfClosingTagToken: Write with current indent
   - CommentToken: Write with current indent
   - DoctypeToken: Write with current indent
4. Handle raw tags (`<pre>`, `<textarea>`): preserve formatting, don't trim whitespace

Integration:

Modify `handleFind()` in `internal/daemon/handlers_observation.go`:

```
// After getting HTML from page:
html := htmlResp.OuterHTML

// Format before searching:
formattedHTML := htmlformat.Format(html)

// Search formatted HTML:
matches, err := d.searchHTML(formattedHTML, params)
```

Testing:

Unit tests in `internal/daemon/htmlformat_test.go`:

- Test minified HTML formatting
- Test preservation of `<pre>` and `<textarea>` content
- Test nested element indentation
- Test self-closing tags
- Test comments and doctypes
- Test empty elements
- Test malformed HTML handling

Integration test:

- Navigate to page with minified HTML
- Search for text in minified section
- Verify readable output with proper context

## References

Research:

- Reviewed `github.com/yosssi/gohtml` source code in `/home/grant/Projects/webctl/context/gohtml/`
- Key files analyzed:
  - `parser.go` (103 lines): Tokenizer usage pattern
  - `tag_element.go` (143 lines): Element rendering with indentation
  - `text_element.go` (43 lines): Text node handling
  - `utils.go` (101 lines): Formatting buffer implementation

Files to modify:

- Create: `internal/daemon/htmlformat.go` (approximately 150-200 lines)
- Create: `internal/daemon/htmlformat_test.go` (approximately 100-150 lines)
- Modify: `internal/daemon/handlers_observation.go` (add formatting call in `handleFind()`)
- Update: `go.mod` (add `golang.org/x/net/html` if not present)

## Updates

- 2025-12-25: Initial version
