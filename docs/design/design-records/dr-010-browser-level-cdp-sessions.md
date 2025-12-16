# DR-010: Browser-Level CDP Sessions

- Date: 2025-12-16
- Status: Accepted
- Category: Architecture

## Problem

The daemon currently connects to a page-level CDP WebSocket. When cross-origin navigation occurs, Chrome's Site Isolation may create a new renderer process and target, causing the existing CDP connection to become stale. Network and console events stop being captured after navigation to a different origin.

Mature CDP tools (Puppeteer, chromedp) solve this by connecting to the browser-level WebSocket and using `Target.setAutoAttach` to automatically track page targets as they come and go. Each page target gets a unique session ID, and the connection remains stable across navigations.

Additionally, users may open multiple browser tabs. The current architecture has no concept of multiple pages or sessions.

## Decision

Connect to the browser-level CDP WebSocket and implement session-based target management:

1. Connect to browser WebSocket (from `/json/version` endpoint) instead of page target
2. Use `Target.setAutoAttach` with `flatten: true` for flat session mode
3. Track multiple page sessions with one designated as "active"
4. Tag buffer entries with session ID and filter by active session
5. Provide `webctl target` command for session listing and switching
6. Show session context in REPL prompt

## Why

- Browser-level connection is stable across page navigations
- Auto-attach handles new targets automatically without reconnection
- Session tracking enables future multi-page debugging workflows
- Matches architecture used by Puppeteer and chromedp (proven patterns)
- Fixes the cross-origin navigation bug at its root cause

## Trade-offs

Accept:

- More complex CDP message handling (session IDs on requests/events)
- Session lifecycle management (attach/detach handling)
- Additional state tracking in daemon

Gain:

- Reliable event capture across all navigation types
- Multi-page awareness (foundation for future features)
- Predictable behavior matching user expectations
- Alignment with industry-standard CDP patterns

## Alternatives

Re-enable Network domain on navigation:

- Subscribe to `Page.frameNavigated`, call `Network.enable` again
- Pro: Minimal code change
- Con: Does not address root cause (page target may be replaced entirely)
- Con: Race conditions between navigation and re-enable
- Rejected: Tested and did not reliably fix the issue

Keep page-level connection with reconnection logic:

- Detect stale connection, reconnect to new page target
- Pro: Less architectural change
- Con: Complex reconnection logic with race conditions
- Con: Gap in event capture during reconnection
- Rejected: Browser-level connection is cleaner

---

## Session Data Structure

Minimal session tracking with entries tagged by session ID:

```
PageSession:
  SessionID  string    # CDP session identifier
  TargetID   string    # CDP target identifier
  URL        string    # Current page URL
  Title      string    # Current page title
```

Buffer entries include session ID:

```
ConsoleEntry:
  SessionID  string    # Which session produced this entry
  Type       string
  Text       string
  Timestamp  int64
  ...existing fields...

NetworkEntry:
  SessionID  string    # Which session produced this entry
  RequestID  string
  URL        string
  ...existing fields...
```

Single global buffer with session filtering, not per-session buffers. This bounds memory usage regardless of session count.

---

## Active Session Behaviour

First session becomes active:

- When daemon starts and first page target attaches, it becomes active
- User is debugging that page by default

New sessions do not change active:

- User opens new tab or page opens popup → new session tracked but active unchanged
- User remains in control of which page they're debugging

Active session detaches triggers auto-switch:

- When active session detaches (tab closed, cross-origin navigation)
- Most recently attached remaining session becomes active
- If no sessions remain, enter "no active session" state
- On auto-switch, REPL displays notification message

No active session state:

- Commands return error with available session list
- User must select a session with `webctl target <query>`

---

## Session Commands

The `target` command handles both listing and switching:

List sessions (no argument):

```bash
webctl target
```

Response:

```json
{
  "ok": true,
  "activeSession": "9A3E8D71...",
  "sessions": [
    {"id": "9A3E8D71...", "title": "Example Domain", "url": "http://example.com", "active": true},
    {"id": "B2C4E5F6...", "title": "Other Page", "url": "https://other.com", "active": false}
  ]
}
```

Switch session (with query):

```bash
webctl target 9A3E
webctl target example
```

Query matching algorithm:

1. Try match as session ID prefix (case-sensitive)
2. If no match, try as title substring (case-insensitive)
3. If ambiguous, return error with matching sessions
4. If no match, return error

Ambiguous match response:

```json
{
  "ok": false,
  "error": "ambiguous query 'e', matches multiple sessions",
  "matches": [
    {"id": "9A3E8D71...", "title": "Example Domain"},
    {"id": "B2C4E5F6...", "title": "Example Other"}
  ]
}
```

---

## REPL Prompt Format

Prompt shows active session context:

Single session:

```
webctl [Example Domain]>
```

Multiple sessions (shows count):

```
webctl [Example Domain](2)>
```

Title truncation:

- Maximum 30 characters
- Longer titles truncated with "..."
- Example: `webctl [JSONPlaceholder - Free Fak...]>`

Session change notification:

- When active session changes due to detach, REPL prints message
- Example: `[Session changed: now debugging "Other Page"]`

---

## Buffer Entry Filtering

Default behaviour filters to active session:

- `webctl console` returns only active session's entries
- `webctl network` returns only active session's entries
- Entries tagged with session ID enable this filtering

Entries discarded on session detach:

- When a session detaches, its entries are purged from buffers
- Keeps buffer focused on relevant data

Clear command scope:

- `webctl clear` clears all entries from all sessions
- `webctl clear console` clears all console entries from all sessions
- `webctl clear network` clears all network entries from all sessions
- Simple mental model: clear means clear everything
- Provides clean slate for debugging without session complexity

---

## No Active Session Error

When no active session exists and user runs a command:

```json
{
  "ok": false,
  "error": "no active session - use 'webctl target <id>' to select",
  "sessions": [
    {"id": "9A3E8D71...", "title": "Example Domain", "url": "http://example.com"},
    {"id": "B2C4E5F6...", "title": "Other Page", "url": "https://other.com"}
  ]
}
```

Provides session list inline so user can immediately select.

---

## Status Command Enhancement

The `webctl status` command includes session information:

```json
{
  "ok": true,
  "running": true,
  "pid": 12345,
  "activeSession": {
    "id": "9A3E8D71...",
    "title": "Example Domain",
    "url": "http://example.com"
  },
  "sessions": [
    {"id": "9A3E8D71...", "title": "Example Domain", "url": "http://example.com", "active": true},
    {"id": "B2C4E5F6...", "title": "Other Page", "url": "https://other.com", "active": false}
  ]
}
```

---

## CDP Implementation Notes

Browser WebSocket URL:

- Obtained from `/json/version` endpoint (already available via `FetchVersion`)
- Field: `VersionInfo.WebSocketURL`

Target.setAutoAttach parameters:

```json
{
  "autoAttach": true,
  "flatten": true,
  "waitForDebuggerOnStart": true
}
```

- `flatten: true` enables flat session mode (single WebSocket, session IDs route messages)
- `waitForDebuggerOnStart: true` pauses new targets until we're ready

Events to handle:

- `Target.attachedToTarget` - new session, store in map, enable domains if page type
- `Target.detachedFromTarget` - remove session, purge entries, update active if needed
- `Target.targetInfoChanged` - update URL/title for session

Per-session domain setup (on attach for page targets):

1. Send `Runtime.enable` with session ID
2. Send `Network.enable` with session ID
3. Send `Page.enable` with session ID
4. Send `Runtime.runIfWaitingForDebugger` to unpause

Message routing:

- Requests include `sessionId` field for session-specific commands
- Events include `sessionId` field identifying source session
- Events without session ID are browser-level (Target.* events)

---

## Updates

- 2025-12-16: Initial design
