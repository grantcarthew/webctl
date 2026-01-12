# P-052: CSS Command Redesign

- Status: Completed
- Started: 2026-01-12
- Completed: 2026-01-12

## Overview

Redesign the CSS command to provide consistent, useful functionality. The current implementation has conceptual issues: `--select` duplicates `computed` and doesn't filter stylesheets like `html --select` filters HTML. This project restructures CSS command to have clear, distinct operations.

## Goals

1. Make `--select` filter stylesheet rules by selector (consistent with `html --select`)
2. Update `computed` to support multiple elements with `--` separators
3. Add `matched` subcommand to show applied rules from stylesheets
4. Add `inline` subcommand to show inline style attributes
5. Ensure consistent output separators across all CSS operations

## Scope

In Scope:
- Redesign `css --select` to filter stylesheet content by selector
- Update `css computed` to use querySelectorAll and add `--` separators
- Implement `css matched` using CDP CSS.getMatchedStylesForNode
- Implement `css inline` using CDP CSS.getInlineStylesForNode or JS
- CSS rule parsing for `--select` functionality
- Update help text and documentation

Out of Scope:
- Changes to other observation commands (html, console, network, cookies)
- New output formats beyond current text/JSON

## Success Criteria

- [x] `css --select "h1"` returns CSS rules where selector matches/contains "h1"
- [x] `css computed "h1"` returns computed styles for all matching elements with `--` separators
- [x] `css matched "#main"` returns applied stylesheet rules for element
- [x] `css inline "#main"` returns inline style attribute content
- [x] All subcommands work with `save` mode
- [x] `--find` continues to work as text filter on all modes
- [x] Help text accurately describes each operation

## Deliverables

- Updated `internal/cli/css.go` with new command structure
- Updated `internal/daemon/handlers_css.go` with new handlers
- CSS rule parsing logic for `--select` filtering
- Updated CLI help text
- Test script `scripts/interactive/test-css.sh` updated

## Technical Approach

Command Structure:
```
css                              # All stylesheets
css --select <sel>               # Filter to rules matching selector
css --find <text>                # Text search (line-based)
css save                         # Save stylesheets

css computed <selector>          # Computed styles for element(s)
css get <selector> <property>    # Single property value
css matched <selector>           # Applied rules from stylesheets
css inline <selector>            # Inline style attribute
```

Implementation Order:
1. `css inline` - Easiest, proves the pattern
2. `css computed` update - Similar to html --select change
3. `css matched` - Uses existing CDP API
4. `css --select` - Requires CSS parsing, most complex

CSS Parsing Options:
- Go CSS parser library (github.com/nicholaides/css or similar)
- Simple regex-based rule extraction for common cases
- Evaluate trade-offs during implementation

## Questions and Uncertainties

- Which CSS parser library is best for Go? Need to research options.
- How should `--select` handle complex selectors like `div h1` or `.container h1`?
- Should `matched` show specificity information?
- What format for `inline` output - raw attribute string or parsed properties?

## Dependencies

- Builds on P-021 (CSS Command Implementation)
- Related to recent `--` separator work in html.go and css.go
