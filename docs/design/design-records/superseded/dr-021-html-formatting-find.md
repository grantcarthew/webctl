# DR-021: HTML Formatting for Find Command

- Date: 2025-12-25
- Status: Superseded by [DR-025](../dr-025-html-command-interface.md)
- Implemented: 2025-12-26
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

Implement custom HTML pretty-printer that formats HTML before find command searches it, and also provide formatted output for the html command by default.

Implementation:

1. Add `internal/htmlformat/format.go` with `Format(html string) (string, error)` function
2. Use `golang.org/x/net/html` tokenizer (standard library package)
3. Format output with 2-space indentation
4. Preserve raw content in `<pre>` and `<textarea>` tags
5. Update `handleFind()` to format HTML before searching
6. Update `html` command to format by default with `--raw` flag to disable
7. Collapse multiple consecutive spaces in text content

The formatter is available as a shared package that both daemon and CLI can use.

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

Location: `internal/htmlformat/format.go` (shared package)

Core function:

```go
Format(html string) (string, error)
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

```go
// After getting HTML from page:
html := htmlResp.OuterHTML

// Format before searching:
formattedHTML, err := htmlformat.Format(html)
if err != nil {
    // Fall back to raw HTML if formatting fails
    formattedHTML = html
}

// Search formatted HTML:
matches, err := d.searchHTML(formattedHTML, params)
```

Modify `runHTML()` in `internal/cli/html.go`:

```go
// Format HTML unless --raw flag is set
htmlOutput := data.HTML
if !rawOutput {
    formatted, err := htmlformat.Format(data.HTML)
    if err != nil {
        // Fall back to raw HTML if formatting fails
    } else {
        htmlOutput = formatted
    }
}

// Write formatted HTML to file
os.WriteFile(outputPath, []byte(htmlOutput), 0644)
```

Testing:

Unit tests in `internal/htmlformat/format_test.go`:

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

Files created/modified:

- Created: `internal/htmlformat/format.go` (169 lines)
- Created: `internal/htmlformat/format_test.go` (942 lines, 46 comprehensive tests)
- Modified: `internal/daemon/handlers_observation.go` (add formatting call in `handleFind()`)
- Modified: `internal/cli/html.go` (add `--raw` flag and formatting logic)
- Modified: `internal/cli/cli_test.go` (update test to expect formatted output)
- Updated: `go.mod` (added `golang.org/x/net v0.48.0`)

Edge cases tested (46 total): script/style tags, HTML entities, ALL 14 void elements, deep nesting (50+ levels), malformed HTML, inline SVG, MathML, long attributes, Unicode/emoji, data attributes, template tags, mixed line endings, CDATA sections, unquoted attributes, CVE-2025-22872 related cases, boolean attributes, custom elements, and more.

Test/Code Ratio: 5.6:1 - Production-grade comprehensive coverage.

## Updates

- 2025-12-25: Initial version
- 2025-12-26: Implemented and accepted
  - Created htmlformat package in `internal/htmlformat/`
  - Implemented Format() function with comprehensive test coverage (46 tests, all passing)
  - **Critical fix:** Added script and style tag preservation (prevents breaking JavaScript/CSS)
  - Integrated into find command for formatted search
  - Added --raw flag to html command for optional unformatted output
  - Added collapseSpaces() helper to normalize whitespace in text nodes
  - Extensive edge case testing: HTML entities, ALL void elements, deep nesting (50+), SVG, MathML, Unicode, malformed HTML, CDATA, unquoted attributes, CVE-2025-22872 related cases, and more
  - Research-driven edge case discovery using Kagi search: found 15+ additional critical edge cases
  - Test/Code Ratio: 5.6:1 (942 test lines / 169 production lines) - Production-grade coverage
  - All existing tests pass with updated expectations
