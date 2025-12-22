# DR-014: Eval Command Interface

- Date: 2025-12-22
- Status: Accepted
- Category: CLI

## Problem

AI agents debugging web applications need to execute arbitrary JavaScript in the browser context to inspect state, call functions, extract data, or manipulate the page for testing. While the `html` command provides DOM inspection, agents need programmatic JavaScript execution for dynamic analysis.

Requirements:

- Execute JavaScript expressions in the current page context
- Handle both synchronous and asynchronous (Promise-based) expressions
- Return serializable results as JSON
- Support complex return values (objects, arrays, primitives)
- Handle errors gracefully with clear messages
- Work with active session in multi-tab scenarios

## Decision

Implement `webctl eval <expression>` command with the following interface:

```bash
webctl eval <expression>
```

Arguments:

expression (required):
- JavaScript expression to evaluate
- Can be any valid JavaScript expression
- Examples: `document.title`, `1+1`, `fetch('/api').then(r => r.json())`

Flags:

| Flag | Short | Type | Description |
|------|-------|------|-------------|
| --timeout | -t | duration | Timeout for async expressions (default: 30s) |

Output format:

Successful evaluation:

```json
{
  "ok": true,
  "value": "Example Domain"
}
```

```json
{
  "ok": true,
  "value": {"data": "fetched", "count": 42}
}
```

```json
{
  "ok": true,
  "value": null
}
```

Expression types and their output:

| Expression | Result |
|------------|--------|
| `1 + 1` | `{"ok": true, "value": 2}` |
| `document.title` | `{"ok": true, "value": "Page Title"}` |
| `[1, 2, 3]` | `{"ok": true, "value": [1, 2, 3]}` |
| `{a: 1, b: 2}` | `{"ok": true, "value": {"a": 1, "b": 2}}` |
| `null` | `{"ok": true, "value": null}` |
| `undefined` | `{"ok": true}` (no value field) |
| `Promise.resolve(42)` | `{"ok": true, "value": 42}` |
| `document.querySelector('.x')` | `{"ok": true}` (DOM nodes not serializable) |

## Why

Single expression argument:

Keep interface minimal - just the expression to evaluate. More complex JavaScript should be in files or passed via here-docs. The expression is everything after `eval` command.

Async expression support:

Modern web apps use Promises extensively. Using `awaitPromise: true` in CDP allows natural async code like `fetch().then()` without wrapper boilerplate. Agents can write idiomatic JavaScript.

JSON serializable results only:

CDP's `returnByValue: true` converts results to JSON. Non-serializable values (DOM nodes, functions, circular references) return undefined or error. This matches what agents can actually use - they cannot manipulate remote object references.

Undefined handling:

When expression evaluates to `undefined`, omit the `value` field rather than including `"value": null`. This distinguishes between explicit null and undefined returns.

Timeout flag:

Async expressions may hang indefinitely. Default 30 second timeout prevents blocking agents. Flag allows adjustment for known slow operations.

## Trade-offs

Accept:

- Non-serializable results (DOM nodes, functions) return undefined
- Circular references cause serialization errors
- No access to remote object references (only JSON values)
- Expression must be valid JavaScript (no syntax validation before execution)
- Multi-statement code requires IIFE wrapper

Gain:

- Simple, minimal interface
- Natural async/Promise support
- Direct JSON results for agent consumption
- Consistent with CDP's built-in serialization
- Timeout prevents hanging

## Alternatives

File-based JavaScript input:

```bash
webctl eval --file script.js
```

- Pro: Supports complex multi-line scripts
- Pro: Avoids shell escaping issues
- Con: Extra file management overhead
- Con: Simple expressions become two-step process
- Rejected: Single expression covers 90% of use cases, can add file support later

Remote object references:

Return object IDs for non-serializable results, allow subsequent operations.

```bash
webctl eval "document.body"
# {"ok": true, "objectId": "1234567890"}
webctl eval --object 1234567890 ".innerHTML"
```

- Pro: Can work with DOM nodes
- Pro: More powerful
- Con: Complex state management
- Con: Object references can become stale
- Con: Agents rarely need DOM manipulation (use click/type commands)
- Rejected: Complexity not justified for debugging use case

REPL mode:

Multi-statement interactive JavaScript REPL.

```bash
webctl eval --repl
> const x = 1
> x + 2
3
```

- Pro: Interactive debugging
- Pro: Variable persistence across statements
- Con: Significant implementation complexity
- Con: Agents don't benefit from interactive mode
- Con: State management issues
- Rejected: Not needed for agent workflows

## Usage Examples

Simple expressions:

```bash
webctl eval "1 + 1"
# {"ok": true, "value": 2}

webctl eval "document.title"
# {"ok": true, "value": "Example Domain"}

webctl eval "window.location.href"
# {"ok": true, "value": "https://example.com/"}
```

Object and array results:

```bash
webctl eval "[1, 2, 3].map(x => x * 2)"
# {"ok": true, "value": [2, 4, 6]}

webctl eval "({name: 'test', count: 42})"
# {"ok": true, "value": {"name": "test", "count": 42}}
```

Async expressions:

```bash
webctl eval "fetch('/api/data').then(r => r.json())"
# {"ok": true, "value": {"data": "response"}}

webctl eval "new Promise(r => setTimeout(() => r('done'), 1000))"
# {"ok": true, "value": "done"}
```

DOM inspection (limited):

```bash
webctl eval "document.querySelectorAll('a').length"
# {"ok": true, "value": 15}

webctl eval "document.querySelector('#main').textContent.trim()"
# {"ok": true, "value": "Main content text"}
```

Multi-statement with IIFE:

```bash
webctl eval "(function() { const x = 1; const y = 2; return x + y; })()"
# {"ok": true, "value": 3}
```

With timeout:

```bash
webctl eval --timeout 60s "slowOperation()"
```

## Implementation Notes

CLI implementation:

- Parse expression from args (join all args with space for shell-friendly use)
- Parse timeout flag (default 30s)
- Connect to daemon via IPC
- Send request: `{"cmd": "eval", "params": {"expression": "...", "timeout": 30000}}`
- Receive response with value
- Output JSON result

Daemon implementation:

- Receive eval request via IPC
- Verify active session exists (return error if not)
- Use active session ID for CDP command routing
- Call Runtime.evaluate with:
  - expression: the JavaScript expression
  - awaitPromise: true (handle async)
  - returnByValue: true (serialize result)
  - timeout: from params (default 30000ms)
- Check for exceptionDetails in response (indicates error)
- Extract result.value for successful evaluation
- Return value or error to CLI

Error cases:

JavaScript syntax error:

```json
{"ok": false, "error": "SyntaxError: Unexpected token ')'"}
```

JavaScript runtime error:

```json
{"ok": false, "error": "ReferenceError: undefinedVar is not defined"}
```

Serialization failure:

```json
{"ok": false, "error": "failed to serialize result: circular reference"}
```

Timeout:

```json
{"ok": false, "error": "evaluation timed out after 30s"}
```

Daemon not running:

```json
{"ok": false, "error": "daemon not running. Start with: webctl start"}
```

No active session:

```json
{"ok": false, "error": "no active session - use 'webctl target <id>' to select"}
```

## CDP Methods

Runtime.evaluate:

Evaluates JavaScript in the page context.

```json
{
  "method": "Runtime.evaluate",
  "params": {
    "expression": "document.title",
    "awaitPromise": true,
    "returnByValue": true,
    "timeout": 30000
  },
  "sessionId": "9A3E8D71..."
}
```

Response (success):

```json
{
  "result": {
    "type": "string",
    "value": "Example Domain"
  }
}
```

Response (error):

```json
{
  "result": {...},
  "exceptionDetails": {
    "text": "Uncaught",
    "exception": {
      "type": "object",
      "subtype": "error",
      "description": "ReferenceError: x is not defined"
    }
  }
}
```

## Testing Strategy

Unit tests:

- Expression parsing from args
- Timeout flag parsing
- Error message formatting
- Response JSON formatting

Integration tests:

- Start daemon, navigate to page, evaluate simple expression
- Verify numeric result
- Verify string result
- Verify object result
- Verify array result
- Verify null result
- Verify undefined result (no value field)
- Test async Promise expression
- Test async fetch expression (with test server)
- Test JavaScript syntax error handling
- Test JavaScript runtime error handling
- Test timeout handling
- Test error when daemon not running
- Test error when no active session

## Session Context

Eval command operates on the active session. When multiple browser tabs are open, the command executes JavaScript in the currently active tab.

Session selection via `webctl target` command allows choosing which tab to inspect. See DR-010 for session management details.

Eval command does not support --clear flag as it is an observation command (read-only operation with respect to buffers). See DR-006 for --clear flag scope.

## Updates

- 2025-12-22: Initial version
