# p-053: CSS Element Identification

- Status: Proposed
- Started:

## Overview

When CSS commands return results for multiple elements, there is no way to identify which element each result corresponds to. The output shows values separated by `--` markers but no element identification.

Example current output for `css inline "div"`:
```
(empty)
--
--active-panel-height: 0px;
--
(empty)
```

Users cannot determine which div has which inline style.

## Goals

1. Add element identification to multi-element CSS output
2. Apply consistent identification across all CSS subcommands that support multiple elements
3. Keep output concise and readable

## Scope

In Scope:
- `css inline <selector>` - multiple elements
- `css computed <selector>` - multiple elements
- `css matched <selector>` - single element (no change needed)
- Daemon-side changes to return element identification
- CLI formatter changes to display identification

Out of Scope:
- Other observation commands (html, console, network, cookies)
- JSON output format changes (may already include sufficient data)

## Success Criteria

- [ ] Multi-element CSS output includes element identification
- [ ] Identification is concise (index, tag, or generated selector)
- [ ] Output remains readable and not cluttered
- [ ] All CSS subcommands with multi-element support are updated

## Deliverables

- Updated `internal/daemon/handlers_css.go` - return element identification
- Updated `internal/ipc/protocol.go` - add identification fields if needed
- Updated `internal/cli/format/css.go` - display identification
- Updated command help text if output format changes

## Technical Approach

Options for element identification:

1. Index only: `[1]`, `[2]`, `[3]`
2. Tag + index: `div[1]`, `div[2]`, `div[3]`
3. Generated selector: `div.panel`, `div#header`, `div:nth-child(2)`

Recommendation: Option 2 (tag + index) provides useful context without complexity of generating unique selectors.

Example output:
```
[div:1] (empty)
--
[div:2] --active-panel-height: 0px;
--
[div:3] (empty)
```

## Questions

- Should the identification include class/id if available?
- What format works best for scripting (grep, awk)?
- Should JSON output include the same identification or use structured data?
