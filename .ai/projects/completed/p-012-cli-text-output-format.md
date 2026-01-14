# P-012: Text Output Format

- Status: Completed
- Started: 2025-12-24
- Completed: 2025-12-24

## Overview

Change default CLI output format from JSON to text across all commands. The text format is optimized for token efficiency, reducing AI agent context window usage by 50-80% compared to JSON.

## Goals

1. Implement text output as default for all commands
2. Add `--json` flag to all output-producing commands
3. Maintain backwards compatibility via `--json` flag
4. Reduce token usage for typical command outputs

## Scope

In Scope:

- Text output formatting for all commands per DR-018 specification
- `--json` flag implementation across all commands
- TTY detection for colour output
- Error format standardization (`Error: <message>`)

Out of Scope:

- New commands (covered by separate projects)
- Changes to JSON output structure
- Configuration file for default format preference

## Success Criteria

- [x] All commands output text by default
- [x] All commands support `--json` flag
- [x] Text output matches DR-018 specification
- [x] Colours disabled when output is piped
- [x] Existing JSON consumers can use `--json` flag

## Deliverables

- Updated CLI commands with text output
- `--json` flag on all output-producing commands
- DR-018: Text Output Format (completed)

## Technical Approach

1. Create text formatter functions for each output type
2. Add `--json` global flag or per-command flag
3. Detect TTY for colour handling
4. Update each command to use text formatter by default
5. Test both text and JSON output modes

## Commands to Update

Action commands (OK/Error pattern):

- navigate, reload, back, forward
- click, type, select, scroll
- wait-for
- start, stop, browser, clear

Structured output commands:

- status
- console
- network
- cookies

File path commands:

- screenshot
- html

Raw value commands:

- eval

## Dependencies

- DR-018: Text Output Format (defines specifications)

## Design Decisions

- DR-018: Text Output Format
