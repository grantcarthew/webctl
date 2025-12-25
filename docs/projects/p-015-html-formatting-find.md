# P-015: HTML Formatting for Find and HTML Commands

- Status: Proposed
- Started: -
- Completed: -

## Overview

Implement custom HTML pretty-printer for both the find and html commands. Modern web frameworks often output minified/single-line HTML, making find output unreadable and html output difficult to inspect.

This project implements a lightweight HTML formatter (approximately 150-200 lines) using golang.org/x/net/html tokenizer. The formatter is used internally by find (always) and by html command (by default, with --raw flag to disable).

## Goals

1. Create custom HTML formatter in daemon
2. Integrate formatter into find command pipeline (always format before search)
3. Update html command to pretty-print by default (add --raw flag for unformatted)
4. Maintain command performance (formatting adds minimal overhead)
5. Ensure formatted output preserves searchability of tags, attributes, and text
6. Write comprehensive tests for formatter and integration

## Scope

In Scope:

- Custom HTML formatter implementation in internal/daemon/htmlformat.go
- Integration with find command (always format before search)
- Integration with html command (format by default, --raw flag to disable)
- Unit tests for formatter edge cases
- Integration tests with minified HTML
- Preservation of preformatted content (pre, textarea tags)
- 2-space indentation with proper nesting

Out of Scope:

- Customizable indentation (hardcoded to 2 spaces)
- Line number output in formatted HTML
- Configuration options for formatting behavior
- Performance optimization beyond basic implementation

## Success Criteria

- [ ] Created internal/daemon/htmlformat.go with Format() function
- [ ] Format() correctly indents nested HTML elements
- [ ] Preserves content in pre and textarea tags (no reformatting)
- [ ] handleFind() formats HTML before searching
- [ ] Find command works with minified HTML (tested with real website)
- [ ] html command pretty-prints by default
- [ ] html command --raw flag returns unformatted HTML
- [ ] Unit tests cover: minified HTML, nested elements, self-closing tags, comments, doctypes, pre/textarea preservation
- [ ] Integration test: navigate to minified page, search text, verify readable output
- [ ] Integration test: html command with/without --raw flag
- [ ] All existing tests still pass
- [ ] Code reviewed for edge cases and error handling

## Deliverables

- internal/daemon/htmlformat.go (approximately 150-200 lines)
- internal/daemon/htmlformat_test.go (approximately 100-150 lines)
- Updated internal/daemon/handlers_observation.go (add formatting call in handleFind)
- Updated internal/cli/html.go (add --raw flag, format by default)
- DR-021: HTML Formatting for Find Command (created)
- Integration tests in daemon integration test suite

## Technical Approach

Implementation steps:

1. Create htmlformat package:
   - Use golang.org/x/net/html tokenizer
   - Track indentation level (starts at 0)
   - Process tokens: StartTag (indent, write, increase level), EndTag (decrease level, indent, write), Text (write trimmed), etc.
   - Special handling for raw tags (pre, textarea): preserve whitespace

2. Integrate into find command:
   - Modify handleFind() in handlers_observation.go
   - After getting HTML from page, call htmlformat.Format()
   - Search formatted HTML instead of raw HTML
   - Return results as normal

3. Integrate into html command:
   - Modify runHTML() in cli/html.go
   - Add --raw flag (boolean, default false)
   - If --raw is false, call htmlformat.Format() before writing to file
   - If --raw is true, write unformatted HTML (current behavior)

4. Testing strategy:
   - Unit tests: various HTML structures, edge cases
   - Integration test: real minified HTML from test page
   - Performance test: ensure formatting doesn't significantly slow find

Reference implementation:

Used github.com/yosssi/gohtml (context/gohtml/) as implementation guide:

- parser.go (103 lines): Tokenizer usage pattern
- tag_element.go (143 lines): Indentation logic
- text_element.go (43 lines): Text node handling
- utils.go (101 lines): Buffer formatting

Simplified by removing:

- Writer wrapper (33 lines)
- Inline tag condensing (approximately 50 lines of complex logic)
- Configurable options (InlineTags map, Condense flag, InlineTagMaxLength)
- Line number formatting

## Dependencies

- DR-021: HTML Formatting for Find Command (design decision)
- P-013: Find Command (completed - this enhances it)
- golang.org/x/net/html package (standard library)

## Design Decisions

- DR-021: HTML Formatting for Find Command

## Testing Strategy

Unit tests (htmlformat_test.go):

Test minified HTML:

```
Input: <!DOCTYPE html><html><head><title>Test</title></head><body><div><p>Text</p></div></body></html>
Expected: Properly indented output with each tag on separate line
```

Test nested elements:

```
Input: <div><ul><li>Item 1</li><li>Item 2</li></ul></div>
Expected: Correct indentation levels for each nesting
```

Test pre tag preservation:

```
Input: <pre>  Line 1
  Line 2</pre>
Expected: Whitespace preserved exactly
```

Test self-closing tags:

```
Input: <img src="test.jpg"/><br/>
Expected: Properly formatted on separate lines
```

Test comments and doctypes:

```
Input: <!DOCTYPE html><!-- comment --><html></html>
Expected: Doctype and comment formatted correctly
```

Integration tests (daemon/integration_test.go):

1. Create test HTML file with minified content containing search term
2. Start daemon, navigate to test page
3. Execute find command for term in minified section
4. Verify output shows properly formatted context lines
5. Verify selector generation still works

## Research

Code review of github.com/yosssi/gohtml:

Location: /home/grant/Projects/webctl/context/gohtml/

Analysis:

- Production code: 519 lines total
- Test code: 535 lines
- Last commit: October 2020 (5+ years unmaintained)
- License: MIT
- Uses golang.org/x/net/html tokenizer (same approach we'll use)
- No unsafe package usage
- Well-structured but includes features we don't need

Key files reviewed:

- parser.go: Token processing loop, raw tag detection
- tag_element.go: Element rendering with indentation
- text_element.go: Text node trimming and handling
- utils.go: Formatted buffer with indentation tracking
- consts.go: Default indent string (2 spaces)

Decision: Build our own simplified version rather than depend on unmaintained library

## Files to Modify

Create:

- internal/daemon/htmlformat.go
- internal/daemon/htmlformat_test.go

Modify:

- internal/daemon/handlers_observation.go (handleFind function)
- go.mod (if golang.org/x/net not already present)

## Notes

Performance considerations:

- Formatting adds processing time to find command
- Tokenizer is efficient (standard library implementation)
- Formatting happens in daemon (not exposed to user)
- Trade-off: slight performance cost for much better UX

Future enhancements (out of scope for this project):

- Add --pretty flag to html command
- Configurable indentation size
- Option to disable formatting for performance
- Line numbers in formatted output

## Updates

- 2025-12-25: Initial project definition
