# p-046: Testing eval Command

- Status: Superseded
- Started: 2025-12-31
- Superseded: 2026-01-24
- Superseded By: p-062 CLI Interaction Tests (automated test framework)

## Overview

Test the webctl eval command which evaluates JavaScript expressions in the browser context. Supports both synchronous and asynchronous (Promise-based) expressions. Results are automatically serialized to JSON. Non-serializable values (DOM nodes, functions, circular references) return undefined.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-eval.sh
```

## Code References

- internal/cli/eval.go

## Command Signature

```
webctl eval <expression>
```

Arguments:
- expression: JavaScript expression to evaluate (supports multiple args joined with spaces)

Flags:
- --timeout, -t: Timeout for async expressions (default 60s, accepts Go duration format)

Global flags:
- --json: JSON output format
- --no-color: Disable color output
- --debug: Enable debug output

## Test Checklist

Simple expressions:
- [ ] eval "1 + 1" (returns 2)
- [ ] eval "document.title" (returns page title)
- [ ] eval "window.location.href" (returns current URL)
- [ ] eval "Date.now()" (returns timestamp)
- [ ] eval "Math.random()" (returns random number)
- [ ] eval "true && false" (returns false)

Object and array results:
- [ ] eval "[1, 2, 3].map(x => x * 2)" (returns [2, 4, 6])
- [ ] eval "({name: 'test', count: 42})" (returns object)
- [ ] eval "Array.from(document.querySelectorAll('a')).map(a => a.href)" (returns array of URLs)
- [ ] eval "[1, 2, 3, 4, 5]" (returns array)
- [ ] eval "Object.keys(window).length" (returns number)

DOM inspection (values only, not nodes):
- [ ] eval "document.querySelectorAll('a').length" (count elements)
- [ ] eval "document.querySelector('#main').textContent.trim()" (get text content)
- [ ] eval "document.querySelector('input').value" (get input value)
- [ ] eval "getComputedStyle(document.body).backgroundColor" (get computed style)
- [ ] eval "document.querySelector('div') !== null" (check existence)

Async/Promise expressions:
- [ ] eval "fetch('/api/data').then(r => r.json())" (fetch API)
- [ ] eval "new Promise(r => setTimeout(() => r('done'), 1000))" (promise with delay)
- [ ] eval "Promise.resolve('immediate')" (resolved promise)
- [ ] eval "Promise.reject('error')" (rejected promise, should error)

Check element existence:
- [ ] eval "document.querySelector('.dashboard') !== null" (check presence)
- [ ] eval "!!document.getElementById('loaded-indicator')" (boolean check)
- [ ] eval "document.querySelector('nonexistent') === null" (check absence)

Get application state:
- [ ] eval "window.__APP_STATE__" (React/Redux state if present)
- [ ] eval "localStorage.getItem('user')" (local storage)
- [ ] eval "sessionStorage.getItem('token')" (session storage)
- [ ] eval "document.cookie" (cookies)

Multi-statement with IIFE:
- [ ] eval "(function() { const x = 1; const y = 2; return x + y; })()" (returns 3)
- [ ] eval "(() => { let sum = 0; for(let i=0; i<10; i++) sum += i; return sum; })()" (returns 45)
- [ ] eval "(function() { return 'nested function'; })()" (returns string)

Modify page state:
- [ ] eval "document.body.style.background = 'red'" (changes background)
- [ ] eval "localStorage.setItem('debug', 'true')" (sets local storage)
- [ ] eval "document.title = 'New Title'" (changes title)
- [ ] eval "window.testVar = 123" (sets window variable)

Timeout handling:
- [ ] eval --timeout 5s "new Promise(r => setTimeout(() => r('done'), 1000))" (should succeed)
- [ ] eval --timeout 1s "new Promise(r => setTimeout(() => r('done'), 5000))" (should timeout)
- [ ] eval -t 10s "fetch('https://httpbin.org/delay/2').then(r => r.json())" (custom timeout)

Return values:
- [ ] Verify undefined expression returns {"ok": true} (no value field)
- [ ] Verify expression with value returns {"ok": true, "value": ...}
- [ ] Verify null returned as {"ok": true, "value": null}
- [ ] Verify false returned as {"ok": true, "value": false}

Error cases:
- [ ] eval "invalid syntax {{" (SyntaxError)
- [ ] eval "undefinedVariable" (ReferenceError)
- [ ] eval "throw new Error('test')" (runtime error)
- [ ] eval with daemon not running (error message)
- [ ] eval "new Promise(() => {})" with short timeout (timeout error)

Output formats:
- [ ] Default text output (raw value)
- [ ] --json output format (includes ok and value)
- [ ] --no-color output
- [ ] --debug verbose output
- [ ] Error output format

Multi-arg expressions:
- [ ] eval 1 + 1 (joined as "1 + 1")
- [ ] eval document.title (joined as "document.title")
- [ ] Verify spaces preserved in multi-arg mode

CLI vs REPL:
- [ ] CLI: webctl eval "document.title"
- [ ] CLI: webctl eval "1 + 1"
- [ ] CLI: webctl eval "[1,2,3].map(x => x * 2)"
- [ ] REPL: eval "document.title"
- [ ] REPL: eval "1 + 1"
- [ ] REPL: eval "[1,2,3].map(x => x * 2)"

## Notes

- Expressions can be synchronous or asynchronous (Promises auto-awaited)
- Non-serializable values (DOM nodes, functions, circular references) return undefined
- Default timeout is 60 seconds for async expressions
- Multi-arg support allows shell-friendly use without quotes in some cases
- JSON mode outputs structured response with ok and value fields
- Text mode outputs raw value only
- Useful for checking page state, extracting data, or debugging

## Issues Discovered

(Issues will be documented here during testing)
