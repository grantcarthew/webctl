# p-048: Testing clear Command

- Status: Superseded
- Started: 2025-12-31
- Superseded: 2026-01-24
- Superseded By: p-062 CLI Interaction Tests (automated test framework)

## Overview

Test the webctl clear command which clears event buffers (console and/or network). Can clear both buffers or target a specific one. Useful for resetting state between test scenarios or focusing on new events.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-clear.sh
```

## Code References

- internal/cli/clear.go

## Command Signature

```
webctl clear [console|network]
```

Arguments:
- target: Optional target buffer to clear ("console" or "network"). If omitted, clears all buffers.

No flags for this command.

Global flags:
- --json: JSON output format
- --no-color: Disable color output
- --debug: Enable debug output

## Test Checklist

Clear all buffers:
- [ ] clear (no argument, clears both console and network)
- [ ] Verify console events cleared
- [ ] Verify network events cleared
- [ ] Verify subsequent console returns empty
- [ ] Verify subsequent network returns empty

Clear console buffer only:
- [ ] clear console
- [ ] Verify console events cleared
- [ ] Verify network events NOT cleared
- [ ] console after clear (should be empty)
- [ ] network after clear (should still have events if any existed)

Clear network buffer only:
- [ ] clear network
- [ ] Verify network events cleared
- [ ] Verify console events NOT cleared
- [ ] network after clear (should be empty)
- [ ] console after clear (should still have events if any existed)

Workflow with console:
- [ ] Navigate to page with console logs
- [ ] console (verify events present)
- [ ] clear console
- [ ] console (verify empty)
- [ ] Generate new console log
- [ ] console (verify only new event)

Workflow with network:
- [ ] Navigate to page with network requests
- [ ] network (verify requests present)
- [ ] clear network
- [ ] network (verify empty)
- [ ] Navigate to new page
- [ ] network (verify only new requests)

Workflow with both buffers:
- [ ] Generate console logs and network requests
- [ ] console and network (verify both have events)
- [ ] clear (no argument)
- [ ] console and network (verify both empty)
- [ ] Generate new events
- [ ] Verify only new events present

Clear between test scenarios:
- [ ] Test scenario 1: generate events
- [ ] Observe events
- [ ] clear
- [ ] Test scenario 2: generate different events
- [ ] Verify only scenario 2 events present

Multiple clears in sequence:
- [ ] clear console
- [ ] clear network
- [ ] clear console (again, should be idempotent)
- [ ] clear network (again, should be idempotent)
- [ ] clear (all)

Empty buffer clears:
- [ ] clear when buffers already empty (should succeed)
- [ ] clear console when console buffer empty
- [ ] clear network when network buffer empty

Error cases:
- [ ] clear invalid-target (error: invalid target)
- [ ] clear Console (case mismatch, should error)
- [ ] clear NETWORK (case mismatch, should error)
- [ ] clear "" (empty string, should work as clear all)
- [ ] clear with daemon not running (error message)

Output formats:
- [ ] Default text output (just OK)
- [ ] --json output with message (all buffers cleared)
- [ ] --json output with message (console buffer cleared)
- [ ] --json output with message (network buffer cleared)
- [ ] --no-color output
- [ ] --debug verbose output

Verify independence of buffers:
- [ ] Generate console events only
- [ ] clear network (should not affect console)
- [ ] console (verify events still present)
- [ ] Generate network requests only
- [ ] clear console (should not affect network)
- [ ] network (verify requests still present)

CLI vs REPL:
- [ ] CLI: webctl clear
- [ ] CLI: webctl clear console
- [ ] CLI: webctl clear network
- [ ] REPL: clear
- [ ] REPL: clear console
- [ ] REPL: clear network

Integration with observation commands:
- [ ] console before clear (has events)
- [ ] clear console
- [ ] console after clear (empty)
- [ ] network before clear (has requests)
- [ ] clear network
- [ ] network after clear (empty)

## Notes

- Clears event buffers maintained by daemon, not browser console
- No confirmation required (immediate execution)
- Idempotent operation (can clear multiple times safely)
- Useful for test isolation and focusing on new events
- Does not clear browser's actual console or network panel
- JSON output includes helpful message about what was cleared
- Target argument is case-sensitive (must be lowercase "console" or "network")

## Issues Discovered

(Issues will be documented here during testing)
