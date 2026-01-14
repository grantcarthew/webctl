# P-013: Find Command

- Status: Completed
- Started: 2025-12-25
- Completed: 2025-12-25

## Overview

Implement `webctl find` command for searching raw HTML content. Enables AI agents and users to locate elements on a page before interacting with them.

## Goals

1. Search raw HTML for text patterns
2. Show context around matches (line before, match, line after)
3. Support plain text and regex search modes
4. Provide selectors for matched elements in JSON output

## Scope

In Scope:

- Text search through raw HTML
- Regex search with `-E` flag
- Case-sensitive option with `-c` flag (plain text only)
- Limit results with `--limit` flag
- Context display (before/match/after lines)
- JSON output with selector and xpath

Out of Scope:

- XPath search expressions
- Attribute-specific search
- Interactive match navigation

## Success Criteria

- [x] `webctl find <text>` searches page HTML
- [x] Minimum 3 character query enforced
- [x] Case-insensitive by default
- [x] `-E` flag enables regex mode
- [x] `-c` flag enables case-sensitive (plain text only)
- [x] `--limit N` limits results
- [x] Text output shows context with indented match line
- [x] JSON output includes selector and xpath
- [x] Exit 0 for success (including no matches)
- [x] Exit 1 for errors (short query, bad regex, etc.)

## Deliverables

- `webctl find` command implementation
- Unit tests for query validation and formatting
- Integration tests for search functionality
- DR-017: Find Command Interface (completed)

## Technical Approach

1. Add find command to CLI with Cobra
2. Implement query validation (minimum length, regex compilation)
3. Daemon handler to fetch HTML and search
4. Line-based search with context capture
5. Selector generation for matched elements
6. Text and JSON output formatters

## Dependencies

- P-012: Text Output Format (find uses text output)
- DR-017: Find Command Interface (defines specification)
- DR-018: Text Output Format (defines output format)

## Design Decisions

- DR-017: Find Command Interface
- DR-018: Text Output Format
