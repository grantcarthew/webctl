# p-050: Testing target Command

- Status: Superseded
- Started: 2025-12-31
- Superseded: 2026-01-24
- Superseded By: p-062 CLI Interaction Tests (automated test framework)

## Overview

Test the webctl target command which lists all page sessions or switches to a specific session. Without arguments, lists all active sessions with their IDs, titles, and URLs. With a query argument, switches to the matching session. Query matching supports session ID prefix (case-sensitive) or title substring (case-insensitive).

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-target.sh
```

## Code References

- internal/cli/target.go

## Command Signature

```
webctl target [query]
```

Arguments:
- query: Optional query to match session (ID prefix or title substring)

No flags for this command.

Global flags:
- --json: JSON output format
- --no-color: Disable color output
- --debug: Enable debug output

## Test Checklist

List all sessions (no query):
- [ ] target (list all sessions)
- [ ] Verify shows session IDs (truncated to 8 chars)
- [ ] Verify shows titles (truncated to 40 chars)
- [ ] Verify shows URLs (full)
- [ ] Verify active session marked/indicated
- [ ] Verify multiple sessions listed if multiple open

Switch by session ID prefix:
- [ ] target 9A3E (switch by ID prefix, case-sensitive)
- [ ] target 1234 (4-digit prefix)
- [ ] target AB (2-character prefix)
- [ ] Verify case-sensitive matching for IDs
- [ ] Verify switches to correct session

Switch by title substring:
- [ ] target example (match "Example Domain" title)
- [ ] target github (match GitHub page title)
- [ ] target "New Tab" (match new tab title)
- [ ] Verify case-insensitive matching for titles
- [ ] Verify switches to correct session

Multi-tab scenario:
- [ ] Open multiple tabs/pages
- [ ] target (list all sessions)
- [ ] target to switch between them
- [ ] Verify each switch successful
- [ ] Verify commands execute in correct session after switch

Single session scenario:
- [ ] Start with single session only
- [ ] target (should list one session)
- [ ] target with ID of current session (should still work)

Session ID handling:
- [ ] Verify full session IDs in JSON output
- [ ] Verify truncated IDs in text output
- [ ] Verify truncation indicator if ID > 8 chars
- [ ] Verify unique prefixes work for switching

Title handling:
- [ ] Verify long titles truncated to 40 chars
- [ ] Verify truncation indicator if title > 40 chars
- [ ] Verify empty titles handled gracefully
- [ ] Verify special characters in titles displayed correctly

Query matching behavior:
- [ ] Unique match: switches to that session
- [ ] Multiple matches: shows error with list of matches
- [ ] No matches: shows error with list of all sessions
- [ ] Partial ID match (unique prefix)
- [ ] Partial title match (case-insensitive)

Error cases - multiple matches:
- [ ] target "page" (if multiple pages match)
- [ ] Verify error message
- [ ] Verify shows matching sessions in error
- [ ] Verify suggests more specific query

Error cases - no matches:
- [ ] target "nonexistent-xyz" (no match)
- [ ] Verify error message
- [ ] Verify shows all sessions in error
- [ ] Verify helpful guidance provided

Error cases - daemon:
- [ ] target with daemon not running (error message)
- [ ] target with no active sessions (error message)

Active session indication:
- [ ] Verify current/active session clearly marked in list
- [ ] Verify active field in JSON output
- [ ] After switching, verify new session becomes active

Navigate and target workflow:
- [ ] navigate example.com (creates session 1)
- [ ] Open new tab/page somehow (creates session 2)
- [ ] target (list both)
- [ ] target to switch to session 1
- [ ] Verify commands execute in session 1
- [ ] target to switch to session 2
- [ ] Verify commands execute in session 2

Output formats:
- [ ] Default text output (formatted table/list)
- [ ] --json output (full session data with IDs, titles, URLs, active flag)
- [ ] --no-color output
- [ ] --debug verbose output

JSON output structure:
- [ ] Verify activeSession field
- [ ] Verify sessions array
- [ ] Verify each session has id, title, url, active fields
- [ ] Verify truncated vs full IDs in JSON

Session persistence:
- [ ] Create multiple sessions
- [ ] Switch between them
- [ ] Verify sessions persist until closed
- [ ] Verify daemon tracks all sessions correctly

Edge cases:
- [ ] Empty query string (should list all, same as no argument)
- [ ] Query matching multiple sessions (error with list)
- [ ] Query with special regex characters
- [ ] Very long session ID
- [ ] Very long title
- [ ] URL with special characters

CLI vs REPL:
- [ ] CLI: webctl target
- [ ] CLI: webctl target 9A3E
- [ ] CLI: webctl target example
- [ ] REPL: target
- [ ] REPL: target 9A3E
- [ ] REPL: target example

## Notes

- Lists all active page sessions when called without arguments
- Query can be session ID prefix (case-sensitive) or title substring (case-insensitive)
- Session IDs truncated to 8 characters in text output (full in JSON)
- Titles truncated to 40 characters in text output
- Active session indicated in output
- Multiple matches show error with list of matching sessions
- No matches show error with list of all sessions
- Useful for multi-tab testing and session management
- Currently may need manual multi-tab creation (check if implemented)

## Issues Discovered

(Issues will be documented here during testing)

Note: Multi-tab/window creation may not be fully implemented yet. Test what's currently available and document any limitations.
