# P-049: Testing find Command

- Status: In Progress
- Started: 2025-12-31

## Overview

Test the webctl find command which searches raw HTML content for text patterns and shows context around matches. Without flags, performs case-insensitive text search. Supports regex patterns and case-sensitive search. Output shows one line before and after each match, with matched text highlighted.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-find.sh
```

## Code References

- internal/cli/find.go

## Command Signature

```
webctl find <text>
```

Arguments:
- text: Text or pattern to search for (minimum 3 characters)

Flags:
- --regex, -E: Treat query as regex pattern
- --case-sensitive, -c: Case-sensitive search (plain text only, not with regex)
- --limit, -l: Limit number of matches (default: all)

Global flags:
- --json: JSON output format (includes CSS selector and XPath for each match)
- --no-color: Disable color output
- --debug: Enable debug output

## Test Checklist

Basic text search (case-insensitive):
- [ ] find "login" (finds "login", "Login", "LOGIN")
- [ ] find "button" (finds all button text)
- [ ] find "submit" (finds submit buttons/text)
- [ ] find "example" (finds example.com references)
- [ ] Verify case-insensitive by default

Case-sensitive search:
- [ ] find -c "Login" (finds only "Login", not "login")
- [ ] find --case-sensitive "Button" (exact case match)
- [ ] find -c "HTTP" (finds only uppercase)
- [ ] Verify case-sensitive flag works correctly

Regex pattern search:
- [ ] find -E "sign\s*up|register" (regex with alternation)
- [ ] find -E "button.*submit" (regex with wildcard)
- [ ] find -E "\d{3}-\d{4}" (phone number pattern)
- [ ] find -E "https?://[^\s]+" (URL pattern)
- [ ] find -E "^\s*<div" (line start pattern)

Limit results:
- [ ] find --limit 5 "button" (first 5 matches only)
- [ ] find -l 1 "login" (first match only)
- [ ] find --limit 10 "the" (limit common word)
- [ ] Verify limit respected in output

Minimum query length validation:
- [ ] find "ab" (should error: minimum 3 characters)
- [ ] find "a" (should error)
- [ ] find "abc" (should succeed, exactly 3 chars)
- [ ] Verify validation happens before daemon check

Context display:
- [ ] Verify one line before match shown
- [ ] Verify one line after match shown
- [ ] Verify matching line prefixed with ">"
- [ ] Verify matched text highlighted in yellow (text mode)

JSON output format:
- [ ] find --json "submit" (includes CSS selector)
- [ ] Verify JSON includes XPath for each match
- [ ] Verify JSON includes query field
- [ ] Verify JSON includes total count
- [ ] Verify matches array structure

Pages with different content types:
- [ ] Find text on simple HTML page (example.com)
- [ ] Find text on complex page (GitHub)
- [ ] Find text in table content
- [ ] Find text in form labels
- [ ] Find text in navigation menus
- [ ] Find text in headings
- [ ] Find text in paragraphs

Special characters in search:
- [ ] find "example.com" (with dot)
- [ ] find "C++" (with plus signs)
- [ ] find "user@example.com" (with @)
- [ ] find "$100" (with dollar sign)

Regex special cases:
- [ ] find -E "\\w+" (word characters)
- [ ] find -E "\\d+" (digits)
- [ ] find -E "\\s+" (whitespace)
- [ ] find -E "." (any character - should match a lot)
- [ ] find -E "^" (line start)
- [ ] find -E "$" (line end)

Error cases:
- [ ] find "ab" (query too short)
- [ ] find -E "[[invalid" (invalid regex pattern)
- [ ] find --case-sensitive -E "test" (invalid flag combination)
- [ ] find with daemon not running (error message)
- [ ] find with no active session (error message)

No matches found:
- [ ] find "xyznonexistenttext123" (no matches)
- [ ] Verify appropriate "no matches" message
- [ ] Verify total: 0 in JSON output

Many matches:
- [ ] find "the" (common word, many matches)
- [ ] find "a" (should error, too short, but if 3+ chars common word)
- [ ] find --limit 20 "and" (limit large result set)

Piping to other commands:
- [ ] find --json "submit" | jq -r '.matches[0].selector' (extract selector)
- [ ] Verify selector can be piped to click command
- [ ] find --json "input" | jq -r '.matches[0].xpath' (extract XPath)

Output format variations:
- [ ] Default text output with context
- [ ] --json output with selectors
- [ ] --no-color output (no highlighting)
- [ ] --debug verbose output

Search in different HTML structures:
- [ ] Find text in <title> tag
- [ ] Find text in <meta> description
- [ ] Find text in <script> tags
- [ ] Find text in <style> tags
- [ ] Find text in comments
- [ ] Find text in attributes

CLI vs REPL:
- [ ] CLI: webctl find "login"
- [ ] CLI: webctl find -E "sign\s*up"
- [ ] CLI: webctl find --case-sensitive "Login"
- [ ] CLI: webctl find --limit 5 "button"
- [ ] REPL: find "login"
- [ ] REPL: find -E "sign\s*up"
- [ ] REPL: find --case-sensitive "Login"
- [ ] REPL: find --limit 5 "button"

## Notes

- Searches raw HTML content, not rendered text
- Minimum query length of 3 characters enforced
- Case-insensitive by default
- Regex mode allows complex pattern matching
- Case-sensitive flag only works with plain text (not regex)
- Output shows context (one line before/after)
- JSON output includes CSS selector and XPath for automation
- Highlighted text in yellow in text mode
- Useful for finding elements to click, verifying content presence
- Can pipe JSON output to extract selectors for other commands

## Issues Discovered

(Issues will be documented here during testing)
