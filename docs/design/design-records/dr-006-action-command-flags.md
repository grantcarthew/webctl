# DR-006: Action Command Flags

- Date: 2025-12-14
- Status: Accepted
- Category: CLI

## Problem

Action commands (navigate, reload, click, type, select, scroll, etc.) are often used to trigger specific browser interactions that agents want to debug. A common workflow pattern is:

1. Clear event buffers to remove old noise
2. Perform the action
3. Observe results in console/network buffers

Currently this requires two separate commands:

```bash
webctl clear console
webctl click "#submit-button"
webctl console
```

This creates several issues:

- More commands for agents to execute
- Potential race condition between clear and action (though unlikely in practice)
- Intent is split across multiple commands
- Verbose for common workflow

Additionally, agents often want to isolate events from specific actions to debug failures. The ability to clear buffers atomically before an action makes this workflow explicit and reliable.

## Decision

Add `--clear[=TARGET]` flag to all action commands:

```bash
webctl navigate URL --clear
webctl click SELECTOR --clear=console
webctl reload --clear=network
```

Flag behavior:

- `--clear` without value: clears all buffers (console and network)
- `--clear=console`: clears only console buffer
- `--clear=network`: clears only network buffer
- `--clear=all`: explicitly clears all buffers (same as bare `--clear`)

Clearing happens atomically BEFORE the action executes.

Note: Clearing always clears all entries regardless of session. This provides a clean slate for debugging. See DR-010 for session management details.

Action commands that support `--clear`:

- navigate
- reload
- back
- forward
- click
- type
- select
- scroll

Observation commands (console, network, screenshot, html, eval, cookies) do NOT support `--clear` as they are read-only operations.

## Why

Workflow convenience for common debugging pattern:

The pattern "clear buffers, perform action, check results" is fundamental to agent debugging workflows. Making this atomic and convenient reduces friction and makes agent scripts more readable.

Explicit intent:

```bash
webctl click "#submit" --clear=console
```

This single command clearly expresses: "clear console, then click, so I can see only logs from this click."

No race conditions:

While unlikely in practice, clearing and acting in a single atomic operation guarantees no events slip through between clear and action.

Maintains standalone clear command:

`webctl clear` remains useful for explicit buffer management independent of actions. The `--clear` flag is a convenience, not a replacement.

## Trade-offs

Accept:

- Increased complexity: all action commands now support this flag
- Two ways to clear: standalone `webctl clear` and `--clear` flag
- Implementation overhead: each action command must handle clearing before execution

Gain:

- One command instead of two for common workflow
- Atomic operation (clear then act)
- Self-documenting intent
- Cleaner agent scripts
- Explicit isolation of events from specific actions

## Alternatives

Keep clear as separate command only:

```bash
webctl clear console
webctl click "#button"
```

- Pro: Single responsibility, simpler command interface
- Pro: Only one way to clear buffers
- Con: More verbose for common case
- Con: Separate commands instead of atomic operation
- Rejected: Common workflow should be convenient

Add clear subcommand to actions:

```bash
webctl click "#button" clear
```

- Pro: Groups clear with action
- Con: Unclear semantics (clear before or after?)
- Con: Inconsistent with Unix flag patterns
- Rejected: Flags are more natural for modifiers

Add both --clear-before and --clear-after:

```bash
webctl click "#button" --clear-before=console --clear-after=all
```

- Pro: Maximum flexibility
- Con: Overly complex for marginal benefit
- Con: Clear-after is rarely useful (just use separate clear command)
- Rejected: --clear-before is sufficient, "before" is implied

## Usage Examples

Isolate console logs from a navigation:

```bash
webctl navigate https://example.com --clear
webctl console  # Only logs from this page load
```

Debug a click interaction:

```bash
webctl navigate https://example.com
webctl click "#submit" --clear=console
webctl console  # Only logs from the click
```

Check network requests from reload:

```bash
webctl reload --clear=network
webctl network  # Only requests from the reload
```

Comprehensive debugging with fresh buffers:

```bash
webctl navigate https://example.com --clear
webctl click "#tab1"
webctl click "#tab2" --clear  # Fresh start for this interaction
webctl console
webctl network
```

Combined with explicit clear for complex workflows:

```bash
webctl navigate https://example.com
webctl click "#load-data"
webctl clear console  # Explicit clear between actions
webctl click "#submit"
webctl console
```

## Implementation Notes

Action command implementation must:

1. Parse `--clear` flag value (defaults to "all")
2. Send clear request to daemon BEFORE sending action request
3. Handle errors from clear operation
4. Proceed with action only if clear succeeds (or fail gracefully)

Daemon implementation:

- No changes needed - existing clear handler supports targeted clearing
- IPC protocol already supports `{"cmd": "clear", "target": "console|network|all"}`

CLI implementation pattern (pseudocode):

```
if --clear flag present:
    target = flag value or "all"
    send clear request to daemon
    if clear fails:
        return error
perform action command
```

## Updates

- 2025-12-14: Initial version
- 2025-12-16: Clarified that clear always clears all sessions (see DR-010)
