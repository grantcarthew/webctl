# DR-017: Find Command Interface

- Date: 2025-12-24
- Status: Proposed
- Category: CLI

## Problem

AI agents interacting with web pages need to locate specific content before clicking or selecting elements. Currently there is no way to search page content to discover what exists on a page. Agents must either know selectors in advance or inspect full HTML manually.

Requirements:

- Search raw HTML content for text patterns
- Show context around matches (line before and after)
- Support both plain text and regex search
- Provide selector/xpath for matched elements (for use with click, type commands)
- Visual output with coloured highlighting for human readability
- Prevent accidental broad searches with minimum query length

## Decision

Implement `webctl find <text>` command with the following interface:

```bash
webctl find <text> [flags]
```

Arguments:

text (required):

- Search query (minimum 3 characters)
- Plain text search by default
- Regex pattern when `-E` flag is used

Flags:

| Flag | Short | Type | Description |
|------|-------|------|-------------|
| --case-sensitive | -c | bool | Case-sensitive search (plain text only) |
| --regex | -E | bool | Treat query as regex pattern |
| --limit | -l | int | Limit number of matches (default: all) |
| --json | | bool | Output as JSON |

Exit codes:

| Code | Meaning |
|------|---------|
| 0 | Success (including no matches) |
| 1 | Error (invalid input, daemon not running, etc.) |

## Output Format

Human-readable output (default):

```
Match 1 of 3
  <div class="hero-text">Welcome to our website</div>
> <a href="/register" class="btn primary">Sign up today</a>
  <p>Already have an account?</p>
---
Match 2 of 3
  <p>Enter your details below</p>
> <button id="google-auth">Sign up with Google</button>
  <span>Or use email instead</span>
---
Match 3 of 3
  <!-- Footer navigation -->
> <a href="/register">Sign up</a> | <a href="/login">Login</a>
  <p>© 2024 Example Corp</p>
```

Visual formatting:

- Separator lines: Cyan colour
- Matched text within line: Yellow highlight
- Match indicator: `>` prefix on matching line
- Context lines: Two-space indent

No matches:

```
No matches found
```

JSON output (--json flag):

```json
{
  "ok": true,
  "query": "sign up",
  "total": 3,
  "matches": [
    {
      "index": 1,
      "context": {
        "before": "<div class=\"hero-text\">Welcome to our website</div>",
        "match": "<a href=\"/register\" class=\"btn primary\">Sign up today</a>",
        "after": "<p>Already have an account?</p>"
      },
      "selector": "a.btn.primary",
      "xpath": "/html/body/div[2]/a[1]"
    },
    {
      "index": 2,
      "context": {
        "before": "<p>Enter your details below</p>",
        "match": "<button id=\"google-auth\">Sign up with Google</button>",
        "after": "<span>Or use email instead</span>"
      },
      "selector": "#google-auth",
      "xpath": "/html/body/form/button[1]"
    }
  ]
}
```

Selector generation priority:

1. `#id` - Most specific, preferred when element has unique ID
2. `.class` - Use most specific class or combination
3. `tag.class` - Tag with class for disambiguation
4. Tag path - Fallback when no ID or class available

## Why

Raw HTML search:

Searching raw HTML rather than rendered text allows agents to find elements by their attributes, classes, IDs, and structure. This is more useful for automation than text-only search, as agents need to identify elements for subsequent commands like click or type.

Context lines:

Showing one line before and after each match provides structural context. Agents can understand where the match appears in the DOM hierarchy without loading full HTML. Similar to grep context output.

Minimum 3 characters:

Prevents accidental broad searches that would return excessive matches. Short queries like "a" or "id" would match nearly every line of HTML. Hard error (not warning) because proceeding with a 2-character search is rarely intentional.

Case-insensitive default:

HTML tag names and most text content are case-insensitive in practice. Users searching for "Login" should find "login" and "LOGIN". Case-sensitive flag available for precise matching when needed.

Regex for plain text only:

Regex patterns have their own case sensitivity via flags (e.g., `(?i)`). Applying `-c` to regex would be confusing and redundant.

Selector and XPath in JSON:

Including both selector and xpath allows agents to pipe results directly to other commands. Selector for CSS-based commands, xpath for more complex automation scenarios.

Coloured output:

Cyan separators and yellow highlighting make matches visually distinct in terminal output. Matches human expectations from tools like grep with colour.

## Trade-offs

Accept:

- 3-character minimum may frustrate users wanting to search for "id" or "a"
- Raw HTML search includes markup noise (tags, attributes)
- Selector generation may not always produce optimal selectors
- Large pages with many matches may produce verbose output
- Regex errors require user to understand regex syntax

Gain:

- Prevents accidental broad searches
- Agents can find elements by attributes and structure
- Direct integration with click/type commands via selector
- Human-readable output with clear visual hierarchy
- Flexible search with regex support
- Context aids understanding without full HTML inspection

## Alternatives

Text-only search (rendered content):

Search visible text content only, ignoring HTML markup.

- Pro: Cleaner results, no HTML noise
- Pro: Matches what users see on page
- Con: Cannot find elements by class, ID, or attributes
- Con: Less useful for automation (need selectors)
- Rejected: HTML structure essential for element identification

Warning instead of error for short queries:

Allow 1-2 character searches with a warning message.

- Pro: More flexible, trusts user intent
- Pro: Allows legitimate short searches
- Con: Easy to accidentally search "a" and get thousands of matches
- Con: Performance impact on large pages
- Rejected: Accidental broad searches more common than intentional short ones

No context lines:

Return only the matching line without before/after context.

- Pro: More compact output
- Pro: Simpler implementation
- Con: Loses structural context
- Con: Harder to understand where match appears in DOM
- Rejected: Context is valuable for understanding page structure

First match only:

Return only the first match instead of all matches.

- Pro: Simpler output
- Pro: Faster for large pages
- Con: May miss important matches
- Con: User must re-run with different queries to find all
- Rejected: All matches provide complete picture

Inline HTML in JSON (no file):

Similar to html command, could write results to file.

- Pro: Consistent with html command approach
- Pro: Handles very large result sets
- Con: Find results are typically small (context lines only)
- Con: Extra file I/O for simple searches
- Con: Breaks piping workflow
- Rejected: Find output is compact enough for inline JSON

## Usage Examples

Basic text search:

```bash
webctl find "login"
# Finds all occurrences of "login" in page HTML (case-insensitive)
```

Case-sensitive search:

```bash
webctl find -c "Login"
# Finds exact case match only
```

Regex search:

```bash
webctl find -E "sign\s*up|register"
# Finds "sign up", "signup", or "register"

webctl find -E "class=\"btn[^\"]*primary"
# Finds elements with btn and primary classes
```

Limit results:

```bash
webctl find --limit 5 "link"
# Returns first 5 matches only
```

JSON output for automation:

```bash
webctl find --json "submit" | jq -r '.matches[0].selector' | xargs webctl click
# Find submit button and click it
```

Combining flags:

```bash
webctl find -c --limit 10 "ERROR"
# Case-sensitive, limited to 10 matches
```

## Error Cases

Query too short:

```bash
webctl find "ab"
# Error: query must be at least 3 characters
# Exit code: 1
```

Invalid regex:

```bash
webctl find -E "[invalid"
# Error: invalid regex pattern: missing closing ]
# Exit code: 1
```

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

## Implementation Notes

CLI implementation:

- Parse text argument and flags with Cobra
- Validate minimum 3 characters before connecting to daemon
- If `-E` flag, validate regex compiles before sending request
- Connect to daemon via IPC
- Send request: `{"cmd": "find", "query": "login", "regex": false, "caseSensitive": false, "limit": 0}`
- Receive response with matches
- Format output based on --json flag
- Apply ANSI colours for human-readable output

Daemon implementation:

- Receive find request via IPC
- Verify active session exists
- Get full page HTML via DOM.getDocument + DOM.getOuterHTML
- Split HTML into lines
- Search each line for matches (text or regex)
- For each match, capture line before, match line, line after
- Generate selector and xpath for containing element
- Return matches to CLI

Selector generation:

- Parse HTML to find element containing matched text
- Check for id attribute first
- Check for unique class or class combination
- Fall back to tag path if needed
- Prefer shortest unique selector

Colour output:

- Use ANSI escape codes for terminal colours
- Cyan: `\033[36m` for separators
- Yellow: `\033[33m` for highlighted match text
- Reset: `\033[0m` after coloured sections
- Detect if stdout is TTY, disable colours if not (for piping)

Line handling:

- Split HTML on newline characters
- Preserve original line content (no trimming)
- Handle edge cases: match on first line (no before), match on last line (no after)
- Empty context lines shown as empty (not omitted)

## Testing Strategy

Unit tests:

- Query validation (minimum length)
- Regex compilation and error handling
- Flag parsing
- Colour formatting functions
- Selector generation logic

Integration tests:

- Start daemon, navigate to page, search for text
- Verify match count and context lines
- Test case-sensitive flag
- Test regex patterns
- Test limit flag
- Test JSON output format
- Test selector accuracy (can be used with click)
- Test no matches returns exit 0
- Test short query returns exit 1
- Test invalid regex returns exit 1
- Test colour output disabled when piping

## Session Context

Find command operates on the active session. When multiple browser tabs are open, the command searches HTML from the currently active tab.

Session selection via `webctl target` command allows choosing which tab to search.

## Future Enhancements

XPath-only search:

```bash
webctl find --xpath "//button[@type='submit']"
```

Search using XPath expressions instead of text. Deferred as text search covers primary use case.

Attribute search:

```bash
webctl find --attr "data-testid=login-btn"
```

Search by specific attribute values. Deferred as regex can accomplish this.

Interactive mode:

```bash
webctl find -i "login"
# Navigate through matches with n/p keys
```

Interactive navigation through matches. Deferred as JSON output enables scripted workflows.

## Updates

- 2025-12-24: Initial version
