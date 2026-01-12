# P-035: Debug Output

- Status: Proposed
- Started:
- Completed:

## Overview

Add comprehensive debug output throughout the CLI codebase. Currently, debug messages are sparse and inconsistent. The `--debug` flag should provide useful diagnostic information for troubleshooting and development.

## Goals

1. Add consistent debug output to all command execution paths
2. Log IPC requests and responses
3. Include timing information for operations
4. Show filter/selector parameters being applied
5. Log file operations (paths, sizes)
6. Establish a consistent debug message format

## Scope

In Scope:

- All CLI commands (html, css, console, network, cookies, navigate, etc.)
- IPC request/response logging
- Filter and selector parameter logging
- File I/O operations
- Timing for key operations

Out of Scope:

- Daemon-side debug output (separate project if needed)
- Log levels beyond debug (info, warn, error hierarchy)
- Log file output (debug goes to stderr only)

## Success Criteria

- [ ] Every command produces debug output when `--debug` is set
- [ ] IPC requests show command and parameters
- [ ] IPC responses show status and data size
- [ ] Filter operations log what was filtered and result counts
- [ ] File saves log path and bytes written
- [ ] Debug format is consistent across all commands
- [ ] Debug output does not appear without `--debug` flag

## Deliverables

- Updated CLI command files with debug statements
- Consistent debug message format documented in code comments
- Updated tests verify debug output where appropriate

## Technical Approach

1. Define debug message categories:
   - `[REQUEST]` - IPC requests
   - `[RESPONSE]` - IPC responses
   - `[FILTER]` - Filter/selector operations
   - `[FILE]` - File I/O operations
   - `[TIMING]` - Performance timing

2. Add debug statements at key points:
   - Before IPC request: parameters being sent
   - After IPC response: status, data size
   - After filtering: input count, output count, filter criteria
   - After file write: path, bytes written

3. Consider adding timing wrapper for operations

## Questions and Uncertainties

- Should timing be opt-in (e.g., `--debug --timing`) or always included with debug?
- What level of detail for IPC data? Full JSON or just size/summary?
- Should debug output be structured (parseable) or human-readable?
