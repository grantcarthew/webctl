# p-061: CLI Observation Tests

- Status: Pending
- Started:
- Design Record: .ai/design/design-records/dr-032-test-framework-architecture.md

## Overview

Create automated tests for the webctl observation commands. These tests verify data capture and output functionality for html, css, console, network, cookies, and screenshot commands.

## Goals

1. Test html command with various selectors and output modes
2. Test css command with selectors and property queries
3. Test console command with type filtering
4. Test network command with status and method filtering
5. Test cookies command with domain filtering and set/delete
6. Test screenshot command with save functionality
7. Test error conditions for each command

## Scope

In Scope:

- scripts/test/cli/test-observation.sh
- Tests for: html, css, console, network, cookies, screenshot commands
- Output modes: stdout, save to file, save to temp
- Filtering options (--select, --find, --type, --status, etc.)
- Error handling: invalid selectors, no data

Out of Scope:

- Interaction command tests (p-062)
- JSON output mode testing (future)
- Performance benchmarks

## Success Criteria

- [ ] scripts/test/cli/test-observation.sh created
- [ ] Tests pass with ./test-runner cli observation
- [ ] html command tests: basic output, selector, find, save
- [ ] css command tests: basic output, selector, computed, get, matched
- [ ] console command tests: basic output, type filter
- [ ] network command tests: basic output, status filter, method filter
- [ ] cookies command tests: basic output, set, delete
- [ ] screenshot command tests: save to file

## Deliverables

- scripts/test/cli/test-observation.sh

## Technical Approach

Test structure:

- Source shared modules (test-framework.sh, assertions.sh, setup.sh)
- Start daemon and test server for all tests
- Navigate to appropriate test pages before each observation test
- Use run_test wrapper for consistent output capture
- Test text output (default)

Key test scenarios:

1. html - navigate to page, verify output contains expected content
2. html --select - verify filtered output
3. css - verify stylesheet output
4. css computed - verify computed styles for element
5. console - navigate to console-types.html, verify log capture
6. network - trigger requests, verify capture
7. cookies - set cookie, read, delete, verify
8. screenshot save - verify file created

## Current State

### Test Framework

- test-runner at project root dispatches to scripts/test/cli/ directory
- scripts/bash_modules/ contains shared modules
- scripts/test/cli/test-navigation.sh demonstrates navigation test patterns

### Test Pages

Available test pages in testdata/pages/:

- navigation.html - basic HTML structure
- forms.html - form elements
- cookies.html - cookie operations
- console-types.html - various console log types
- network-requests.html - triggers network requests
- css-showcase.html - CSS styling examples

## Dependencies

- p-055: Test Framework Bash Modules (completed)
- p-056: Test Library (completed)
- p-057: Test Runner (completed)
- p-058: Test Pages (completed)
- p-060: CLI Navigation Tests (completed)

## Notes

- Observation tests require a running daemon and test server
- Some tests need specific page content (console-types for console logs)
- Screenshot tests should verify file exists and has content
- Clean up saved files in cleanup trap
