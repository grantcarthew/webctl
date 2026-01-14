# DR-018: Text Output Format

- Date: 2025-12-24
- Status: Accepted
- Category: CLI

## Problem

The current default output format is JSON. While JSON is machine-parseable, it has significant overhead for AI agent consumption:

- Structural tokens (braces, brackets, quotes, colons, commas) add no information value
- Field names repeat for every entry, multiplying token cost
- A simple error message like `{"ok": false, "error": "daemon not running"}` uses ~10-12 tokens when the actual information is ~3-4 tokens

AI agents are the primary consumers of webctl output. Token efficiency directly impacts agent context window usage and API costs.

Requirements:

- Default output format optimized for token efficiency
- Human readable without sacrificing efficiency
- Structured data uses visual structure (indentation, position) instead of repeated labels
- JSON available via `--json` flag for programmatic parsing when needed

## Decision

Change default output format from JSON to text across all commands. Add `--json` flag to all commands that produce output.

Text output principles:

- Use visual structure (indentation, separators, position) to convey meaning
- Minimize repeated labels and structural tokens
- Presence of data implies state (no need to say "running" when showing pid)
- Heading as status where applicable
- Native format for captured data (JSON bodies stay JSON)

## Output Specifications

### status

All systems operational:

```
OK
pid: 12345
port: 9222
sessions:
  * https://example.com
    https://google.com
    https://github.com/user/repo
```

The `*` indicates active session.

No session:

```
No session
pid: 12345
port: 9222
```

No browser:

```
No browser
pid: 12345
```

Not running:

```
Not running
```

### find

```
2 matches
---
<div class="hero">Welcome</div>
  <a href="/register">Sign up today</a>
<p>Already have an account?</p>
---
<p>Or continue with</p>
  <button id="google-auth">Sign up with Google</button>
<span class="divider">or</span>
```

Structure:

- Match count as heading
- `---` separator before each match
- Context: line before, matching line (indented), line after
- No closing separator

No matches:

```
No matches
```

### console

```
[12:34:56] ERROR Failed to load resource: net::ERR_CONNECTION_REFUSED
  https://example.com/app.js:142
[12:34:57] WARN Deprecated API: document.write
  https://example.com/legacy.js:89
[12:34:58] LOG User clicked button
  https://example.com/main.js:23
```

Structure:

- Timestamp, level, message on main line
- Source URL and line number indented below

### network

```
GET https://example.com/api/users 200 45ms
POST https://example.com/api/login 401 123ms
  request: {"email":"user@example.com","password":"secret"}
  response: {"error":"invalid credentials"}
GET https://example.com/style.css 200 12ms
```

Structure:

- Method, URL, status code, duration on main line
- Request/response bodies indented (when present)
- Bodies shown as native JSON (captured data)
- No separators between entries

### screenshot

```
/tmp/webctl-screenshots/25-12-24-125634-example-domain.png
```

Just the file path.

### html

```
/tmp/webctl-html/25-12-24-125634-example-domain.html
```

Just the file path.

### eval

Raw JS return value:

```
42
```

```
hello world
```

```
{"name":"John","age":30}
```

```
null
```

```
undefined
```

### cookies

```
session_id=abc123; domain=example.com; path=/; secure; httponly
user_pref=dark; domain=example.com; path=/
tracking=xyz789; domain=.example.com; path=/; expires=2025-12-31
```

One cookie per line, semicolon-separated attributes.

### Action Commands

Commands: navigate, reload, back, forward, click, type, select, scroll, wait-for, start, stop, browser, clear

Success:

```
OK
```

Error:

```
Error: Element not found: .nonexistent
```

```
Error: Navigation timeout
```

```
Error: Daemon already running
```

## Why

Token efficiency:

JSON output for a simple status check might be 50+ tokens. The equivalent text output is 10-15 tokens. For complex outputs like network logs with dozens of entries, the savings multiply significantly.

Visual structure over labels:

Indentation and position convey structure without tokens. An indented line is clearly a child/detail of the line above. Position in a sequence (before/match/after) is self-evident from the order.

Heading as status:

Instead of `status: OK` followed by data, just `OK` followed by data. The heading tells you the state, the data that follows is contextual. Saves the `status: ` label tokens.

Native format for captured data:

Network request/response bodies are JSON because that is what the server sent. Converting them would lose fidelity and add complexity. The text format wraps native data, not replaces it.

Human readability:

The text format is actually more readable than JSON for humans too. No escaping, no structural noise, clean visual hierarchy.

## Trade-offs

Accept:

- Text output is not directly parseable by standard JSON tools
- Agents must understand visual structure conventions
- Different commands have different output structures
- Need to implement `--json` flag across all commands

Gain:

- 50-80% reduction in token usage for typical outputs
- Cleaner human-readable output
- Visual structure matches conceptual structure
- JSON still available when programmatic parsing needed

## Alternatives

Keep JSON as default:

- Pro: Consistent, parseable, familiar
- Pro: No migration effort
- Con: High token overhead
- Con: AI agents waste context on structural tokens
- Rejected: Token efficiency is critical for AI agent workflows

YAML output:

- Pro: Less verbose than JSON
- Pro: Still structured and parseable
- Con: Still has structural overhead (keys, colons, dashes)
- Con: Indentation sensitivity can cause issues
- Rejected: Custom text format is more token-efficient

Line-based key-value only:

```
method: GET
url: https://example.com
status: 200
```

- Pro: Consistent structure
- Pro: Easy to parse
- Con: Labels repeat for every field, every entry
- Con: Not as token-efficient as visual structure
- Rejected: Visual structure (indentation, position) more efficient

## Implementation Notes

CLI changes:

- Add `--json` flag to all output-producing commands
- Default to text output when `--json` not specified
- JSON output preserves current structure for backwards compatibility
- Text output implemented per-command based on specifications above

Colour handling:

- Text output may include ANSI colours for terminal display
- Detect if stdout is TTY; disable colours when piping
- Colours used: cyan for separators (find), standard terminal colours otherwise

Error format:

- All errors use `Error: <message>` format
- Sentence case for readability
- Exit code 1 for errors, 0 for success

## Testing Strategy

Unit tests:

- Text formatting functions for each command
- Colour stripping for non-TTY output
- Edge cases (empty results, single item, many items)

Integration tests:

- Verify text output matches specification
- Verify `--json` produces valid JSON
- Verify colour codes present in TTY, absent in pipe

## Updates

- 2025-12-24: Initial version
