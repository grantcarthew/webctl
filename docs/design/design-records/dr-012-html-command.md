# DR-012: HTML Command Interface

- Date: 2025-12-17
- Status: Accepted
- Category: CLI

## Problem

AI agents debugging web applications need to inspect HTML structure to understand page composition, diagnose rendering issues, verify DOM manipulation, and extract content. While browser DevTools provides this capability interactively, agents need programmatic access to full page HTML or specific element HTML.

Requirements:

- Extract full page HTML for comprehensive page analysis
- Extract specific element HTML via CSS selectors for focused debugging
- Return HTML in a format agents can read and analyze incrementally
- Support multiple element matches when selector is ambiguous
- Work with active session in multi-tab scenarios
- Handle large HTML documents efficiently (typical pages are hundreds/thousands of lines)

## Decision

Implement `webctl html [selector]` command with the following interface:

```bash
webctl html [selector] [flags]
```

Arguments:

selector (optional):
- CSS selector to target specific element(s)
- Examples: `.content`, `#main`, `div.card`, `section > p`
- If omitted, returns full page HTML
- CSS selectors only (no XPath)

Flags:

| Flag | Short | Type | Description |
|------|-------|------|-------------|
| --output | -o | string | Save to specified path instead of temp directory |

Output format:

JSON response with file path:

```json
{
  "ok": true,
  "path": "/tmp/webctl-html/25-12-17-214523-example-domain.html"
}
```

File location: `/tmp/webctl-html/` (separate from screenshots)

Filename pattern: `YY-MM-DD-HHMMSS-{normalized-title}.html`

Uses same title normalization as screenshot command for consistency.

HTML content:

Full page (no selector):
```html
<!DOCTYPE html>
<html lang="en">
<head>
  <title>Example Domain</title>
  ...
</head>
<body>
  ...
</body>
</html>
```

Single element match:
```html
<div class="content">
  <p>Text content</p>
</div>
```

Multiple element matches:
```html
<!-- Element 1 of 3: div.card -->
<div class="card">...</div>

<!-- Element 2 of 3: div.card -->
<div class="card">...</div>

<!-- Element 3 of 3: div.card -->
<div class="card">...</div>
```

HTML type: Outer HTML only (includes the element tags, not just inner content).

## Why

File-based output for HTML:

HTML pages are typically very large (hundreds to thousands of lines). File-based output allows:

- Incremental reading with Read tool offset/limit parameters
- Agents can read first 50 lines to understand structure
- Come back later to read specific sections
- No need to load entire HTML into memory at once
- Similar pattern to screenshot command (file-based outputs)

Agents cannot include multi-thousand line HTML in JSON responses efficiently. File-based approach matches screenshot design philosophy.

Separate /tmp/webctl-html/ directory:

Keeping HTML separate from screenshots (/tmp/webctl-screenshots/) avoids breaking changes to screenshot command while maintaining clear organization. File extensions make type obvious (.html vs .png).

Multiple element matches with separators:

When selector matches multiple elements, include all matches with HTML comment separators. This allows:

- Agents see all matching elements (not just first)
- Comments provide count and context
- Agents can read file incrementally to process each match
- No ambiguity about which element was selected

Alternative of returning only first match would hide useful debugging information.

CSS selectors only:

CSS selectors cover 95% of use cases and match browser DevTools behavior. XPath is more powerful but complex and less familiar to most users. Keeping interface simple reduces learning curve.

Outer HTML only:

Always include element tags in output. This provides complete context for debugging - agents see full element with attributes, classes, IDs. Inner HTML only would lose critical structural information.

--output flag for agents:

Like screenshot command, --output flag allows agents to save HTML to working directory where they can read without approval. Default /tmp location works for interactive human use, --output enables agent workflows.

Session context:

HTML command operates on active session (same as screenshot, console, network). Consistent behavior across observation commands. User can switch sessions with `webctl target` command.

## Trade-offs

Accept:

- Temp files require eventual OS cleanup (disk space usage)
- File-based output requires two operations (capture + read)
- Multiple matches may produce large files
- Title normalization may produce non-unique filenames (timestamp provides uniqueness)
- CSS selectors only (no XPath support)
- Outer HTML only (no inner HTML option)

Gain:

- Efficient handling of large HTML documents
- Incremental reading capability for agents
- Consistent JSON interface across all commands
- Files persist for multi-step agent workflows
- Multiple matches provide complete debugging context
- Automatic chronological organization
- Agent-friendly with --output flag

## Alternatives

Inline JSON with HTML content:

```json
{"ok": true, "html": "<!DOCTYPE html>..."}
```

- Pro: Single operation, no file I/O
- Pro: Works over network protocols easily
- Con: Large JSON payloads for typical pages (10-100KB+)
- Con: Cannot read incrementally with offset/limit
- Con: Entire HTML must fit in memory
- Con: Not practical for large pages
- Rejected: File-based approach more efficient for large documents

First match only for selectors:

When selector matches multiple elements, return only first match.

- Pro: Simpler output, single element
- Pro: Matches querySelector() behavior
- Con: Hides useful debugging information
- Con: Agents cannot see all matches
- Con: May miss relevant elements
- Rejected: Complete information more valuable for debugging

Unified /tmp/webctl/ directory:

Use single directory for all file outputs (screenshots and HTML).

- Pro: Single location to find all webctl outputs
- Pro: Simpler directory structure
- Con: Breaking change to screenshot command
- Con: Mixed file types in one directory
- Rejected: Avoid breaking changes, file extensions already indicate type

Include selector in filename:

```bash
25-12-17-214523-example-domain-div-card.html
```

- Pro: Filename indicates what was captured
- Pro: More descriptive
- Con: Longer filenames
- Con: Selector normalization complexity (. # > special chars)
- Con: Full page would need special handling
- Rejected: Added complexity not worth marginal benefit

Support XPath selectors:

```bash
webctl html "/html/body/div[1]/section"
```

- Pro: More powerful selection capabilities
- Pro: Can select by position/attributes
- Con: Less familiar to most users
- Con: More complex implementation
- Con: Different CDP methods required
- Rejected: CSS selectors sufficient for 95% of use cases

Inner HTML option:

Add `--inner` flag to return element contents only (no tags).

- Pro: Flexibility for different use cases
- Con: Outer HTML almost always more useful for debugging
- Con: Additional flag complexity
- Con: Less clear what was captured
- Rejected: Outer HTML provides complete context

Error on multiple matches:

Require selectors to match exactly one element.

- Pro: Forces precise selectors
- Pro: No ambiguity
- Con: Frustrating user experience
- Con: Selector precision not always needed
- Con: Extra work to disambiguate
- Rejected: Returning all matches more helpful

## Usage Examples

Full page HTML:

```bash
webctl html
# {"ok": true, "path": "/tmp/webctl-html/25-12-17-214523-example-domain.html"}
```

Specific element:

```bash
webctl html ".main-content"
# {"ok": true, "path": "/tmp/webctl-html/25-12-17-214524-example-domain.html"}

webctl html "#header"
# {"ok": true, "path": "/tmp/webctl-html/25-12-17-214525-example-domain.html"}
```

Complex selector:

```bash
webctl html "div.card > h2"
# {"ok": true, "path": "/tmp/webctl-html/25-12-17-214526-example-domain.html"}
```

Custom output path (agent workflow):

```bash
webctl html --output ./page.html
# {"ok": true, "path": "./page.html"}

webctl html ".content" -o ./debug/content.html
# {"ok": true, "path": "./debug/content.html"}
```

Agent workflow - capture and analyze:

```bash
webctl navigate https://example.com
webctl html --output ./page.html
# Agent reads ./page.html incrementally to analyze structure
```

Multi-step debugging:

```bash
webctl html
# Inspect full page structure
webctl html "nav.primary"
# Focus on navigation element
webctl html "nav.primary a"
# Examine all navigation links
```

## Implementation Notes

CLI implementation:

- Parse selector and flags with Cobra
- Connect to daemon via IPC
- Send request: `{"cmd": "html", "selector": ".content"}`
- Receive response with HTML string
- Generate filename using timestamp and normalized title
- Create /tmp/webctl-html/ directory if not exists
- Write HTML to file
- Return JSON response with file path

Daemon implementation:

- Receive html request via IPC
- Verify active session exists (return error if not)
- Use active session ID for CDP command routing
- For full page (no selector):
  - Call DOM.getDocument to get root node
  - Call DOM.getOuterHTML with document node ID
- For selector:
  - Call DOM.getDocument to get root node
  - Call DOM.querySelectorAll with selector (not querySelector - need all matches)
  - If no matches, return error
  - For each match, call DOM.getOuterHTML
  - If multiple matches, concatenate with HTML comment separators
- Return HTML string to CLI

Title normalization:

- Use same normalizeTitle() function as screenshot command
- Consistent filenames across commands
- Same 30 character limit and hyphenation rules

Filename generation:

- Format: YY-MM-DD-HHMMSS-{normalized-title}.html
- Use local time for timestamp (user's timezone)
- Ensure /tmp/webctl-html/ exists before writing
- No collision handling needed (timestamp provides uniqueness)

Custom output path:

- When --output specified, skip temp directory logic
- Use provided path directly
- Create parent directories if needed
- Validate path is writable (return error if not)

Error cases:

Daemon not running:

```json
{"ok": false, "error": "daemon not running. Start with: webctl start"}
```

No active session:

```json
{
  "ok": false,
  "error": "no active session - use 'webctl target <id>' to select",
  "sessions": [...]
}
```

Selector matches no elements:

```json
{"ok": false, "error": "selector '.nonexistent' matched no elements"}
```

Invalid selector syntax:

```json
{"ok": false, "error": "invalid CSS selector syntax: '.class['"}
```

CDP failure:

```json
{"ok": false, "error": "failed to get HTML: connection timeout"}
```

File write failure:

```json
{"ok": false, "error": "failed to write HTML: permission denied"}
```

## CDP Methods

DOM.getDocument:

Gets the root DOM node for the active session.

```json
{
  "method": "DOM.getDocument",
  "params": {
    "depth": -1,
    "pierce": false
  },
  "sessionId": "9A3E8D71..."
}
```

Returns document node with nodeId.

DOM.querySelectorAll:

Finds all elements matching CSS selector.

```json
{
  "method": "DOM.querySelectorAll",
  "params": {
    "nodeId": 1,
    "selector": ".content"
  },
  "sessionId": "9A3E8D71..."
}
```

Returns array of nodeIds for matching elements.

DOM.getOuterHTML:

Gets outer HTML for a specific node.

```json
{
  "method": "DOM.getOuterHTML",
  "params": {
    "nodeId": 42
  },
  "sessionId": "9A3E8D71..."
}
```

Returns outerHTML string for the node.

Sequence for full page:
1. DOM.getDocument → get root nodeId
2. DOM.getOuterHTML with root nodeId → get full page HTML

Sequence for selector:
1. DOM.getDocument → get root nodeId
2. DOM.querySelectorAll with selector → get array of nodeIds
3. For each nodeId: DOM.getOuterHTML → get element HTML
4. Concatenate with comment separators if multiple matches

## Testing Strategy

Unit tests:

- Selector parsing and validation
- Flag parsing (--output)
- Error message formatting
- Filename generation using mock executor

Integration tests:

- Start daemon, navigate to page, extract full HTML
- Verify HTML file created in /tmp/webctl-html/
- Verify filename matches pattern
- Verify HTML is valid and complete
- Test selector extraction for single element
- Test selector extraction for multiple elements (verify separators)
- Test --output creates file at custom path
- Test error when selector matches nothing
- Test error when daemon not running
- Test error when no active session
- Verify HTML uses active session (multi-tab scenario)

## Session Context

HTML command operates on the active session. When multiple browser tabs are open, the command extracts HTML from the currently active tab.

Session selection via `webctl target` command allows choosing which tab to inspect. See DR-010 for session management details.

HTML command does not support --clear flag as it is an observation command (read-only operation). See DR-006 for --clear flag scope.

## Future Enhancements

Potential future additions (deferred from initial implementation):

XPath selector support:

```bash
webctl html --xpath "/html/body/div[1]"
```

More powerful selection capabilities. Deferred as CSS selectors cover most use cases.

Inner HTML option:

```bash
webctl html ".content" --inner
```

Extract element contents only (no tags). Deferred as outer HTML provides more context.

Computed styles option:

```bash
webctl html ".button" --with-styles
```

Include computed CSS styles with HTML. Deferred as separate concern from HTML structure.

Pretty-print option:

```bash
webctl html --pretty
```

Format HTML for readability. Deferred as browser HTML is usually already formatted.

## Updates

- 2025-12-17: Initial version
