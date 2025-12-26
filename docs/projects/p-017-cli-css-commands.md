# P-017: CSS Commands

- Status: Proposed
- Started: TBD
- Completed: TBD

## Overview

Add CSS extraction, inspection, and manipulation commands to webctl, enabling developers to debug styles, extract stylesheets, get computed CSS properties, and inject custom CSS for testing. This complements the existing HTML extraction and provides comprehensive access to page styling information.

## Goals

1. Implement CSS extraction to file (like html command)
2. Implement computed styles inspection via CDP
3. Implement single property getter for quick checks
4. Implement CSS injection for testing and modifications
5. Support CSS formatting with raw option
6. Provide both file output (large data) and stdout (quick checks)

## Scope

In Scope:

- `webctl css save [selector]` - Extract and save CSS to file
  - All stylesheets (inline, style tags, linked) when no selector
  - Computed styles for element when selector provided
  - Formatting by default, --raw flag for unformatted
  - --output flag for custom file path
- `webctl css computed <selector>` - Get computed styles to stdout
  - All CSS properties for matched element
  - Text format (default) and JSON format
- `webctl css get <selector> <property>` - Get single CSS property to stdout
  - Quick property checks for scripting
  - Returns computed value
- `webctl css inject <css>` - Inject CSS into page
  - Inline CSS string
  - --file flag to inject from file
  - Useful for testing, hiding elements, visual modifications
- CSS formatting (pretty-print with indentation)
- Integration with existing file naming patterns (like html command)

Out of Scope:

- CSS modification/editing (can inject new CSS but not edit existing)
- CSS validation or linting
- SASS/LESS compilation
- CSS source maps
- Specific framework handling (CSS-in-JS, styled-components)
- CSS coverage analysis
- Screenshot comparison after CSS changes

## Success Criteria

- [ ] Can extract all page CSS to a formatted file
- [ ] Can extract computed styles for a selector to file
- [ ] Can get all computed styles for element to stdout
- [ ] Can get single CSS property value to stdout
- [ ] Can inject CSS string into page
- [ ] Can inject CSS from file into page
- [ ] CSS formatting works (indentation, line breaks)
- [ ] --raw flag outputs unformatted CSS
- [ ] --output flag saves to custom path
- [ ] File naming matches html command pattern
- [ ] Works with --json flag for programmatic use
- [ ] Error handling for invalid selectors
- [ ] Error handling for non-existent properties

## Deliverables

- `internal/cli/css.go` - CSS command implementation with subcommands
  - save subcommand
  - computed subcommand
  - get subcommand
  - inject subcommand
- `internal/cli/format/css.go` - CSS text formatters
  - Computed styles formatter
  - Property value formatter
- `internal/cssformat/format.go` - CSS formatting/pretty-printing
  - Parse and format CSS rules
  - Indentation and line breaks
- `internal/daemon/handlers_css.go` - CSS command handlers
  - Extract all stylesheets via CDP
  - Get computed styles via CDP Runtime.evaluate
  - Inject CSS via CDP Page.addStyleTag
- `docs/cli/css.md` - User documentation for css commands
- DR-023: CSS Command Architecture
- Tests for CSS extraction, formatting, and injection
- Updated AGENTS.md

## Technical Approach

High-level implementation strategy:

1. CSS Extraction (save subcommand)
   - Use CDP CSS.getAllStyleSheets() to get all CSS
   - Or use Runtime.evaluate with document.styleSheets iteration
   - For selector: use getComputedStyle(element) via Runtime.evaluate
   - Format CSS with cssformat package
   - Save to temp directory with naming pattern like html command

2. Computed Styles (computed subcommand)
   - Use CDP Runtime.evaluate: `window.getComputedStyle(element)`
   - Parse CSSStyleDeclaration object
   - Output all properties to stdout in text or JSON format

3. Single Property (get subcommand)
   - Use CDP Runtime.evaluate: `getComputedStyle(element).getPropertyValue(property)`
   - Output value to stdout
   - Simple text output for scripting

4. CSS Injection (inject subcommand)
   - Use CDP Page.addStyleTag with content or file
   - Append style tag to document head
   - Return success confirmation

5. CSS Formatting
   - Parse CSS into rules, selectors, properties
   - Pretty-print with configurable indentation
   - Handle @media queries, @keyframes, etc.
   - --raw flag bypasses formatting

6. Command Structure
   - Parent command: css
   - Subcommands: save, computed, get, inject
   - Flags: --output, --raw, --json, --file

## Questions & Uncertainties

- Should css save without selector get ALL stylesheets or just inline styles?
- How to handle cross-origin stylesheets (CORS restrictions)?
- Should we support CSS minification (opposite of formatting)?
- Should css computed support multiple selectors?
- What CSS properties to include (all 300+, or common subset)?
- Should we filter out default browser styles (like user agent stylesheet)?
- How to handle !important rules in output?
- Should injection be temporary (page reload removes) or persistent?

## Testing Strategy

- Unit tests for CSS formatting
- Integration tests for CSS extraction via CDP
- Integration tests for computed styles retrieval
- Integration tests for CSS injection
- Test file naming and path handling
- Test error cases (invalid selector, missing property)
- Manual testing with real websites
- Test formatting output readability

## Notes

Key insight: CSS debugging is complementary to HTML inspection. While html command shows structure, css commands show styling. Together they provide complete page analysis.

The subcommand structure (save, computed, get, inject) clearly separates different use cases:
- save: large output, saved to file
- computed: medium output, to stdout for inspection
- get: small output, to stdout for quick checks/scripting
- inject: modification, for testing

Following the html command pattern for file output ensures consistency and familiarity.

CSS formatting improves readability for debugging, similar to HTML formatting in the html command.
