# DR-030: Find Command Removal

- Date: 2025-12-28
- Status: Accepted
- Category: CLI

## Problem

The standalone find command creates confusion and inconsistency in the CLI interface. Current issues:

- find command only searches HTML, despite generic name suggesting broader capability
- Users expect to search console logs, network requests, CSS, and cookies
- Standalone command obscures the relationship between search and data source
- Inconsistent with universal --find flag pattern established for observation commands
- Creates duplicate functionality (find command vs --find flag)
- Generic name "find" doesn't indicate HTML-only searching

Users need text search across all observation data types (HTML, CSS, console, network, cookies), not just HTML. The standalone command model limits search to one data source and doesn't scale to multiple observation commands.

## Decision

Remove the standalone find command entirely. Replace with universal --find flag available on all observation commands:

```bash
# Old (standalone find - REMOVED)
webctl find "login"              # Only searches HTML

# New (universal --find flag)
webctl html --find "login"       # Search HTML
webctl css --find "button"       # Search CSS
webctl console --find "error"    # Search console logs
webctl network --find "api/"     # Search network requests
webctl cookies --find "session"  # Search cookies
```

The --find flag is available on all observation commands (html, css, console, network, cookies) and works across all output modes (default/show/save).

Breaking change: No deprecation period. The find command will be removed completely.

## Why

Universal Pattern Principle:

Text search is a universal filtering operation that applies to all observation data. Making it a flag rather than a standalone command:

- Works consistently across all data types
- Indicates what data source is being searched
- Integrates with other filters and output modes
- Scales to new observation commands

The --find flag follows the universal pattern principle: common operations should use consistent interfaces across all commands.

Eliminates Confusion:

The standalone find command only searches HTML, but users expect it to search other data too. Removing the command and providing --find on each observation command makes the search scope explicit:

- webctl html --find "text" - clearly searches HTML
- webctl console --find "text" - clearly searches logs
- webctl network --find "text" - clearly searches requests

No ambiguity about what data is being searched.

Better Discoverability:

Filter flags are listed in command help output. Users discovering the html command see --find as an available filter. The standalone command is isolated and less discoverable.

Composability with Other Filters:

The --find flag composes with other filters and output modes:

```bash
webctl html show --select "form" --find "password"
webctl console save ./errors.json --type error --find "TypeError"
webctl network --status 5xx --find "api/" --tail 20
```

Standalone command cannot compose with these other filters.

Consistent CLI Design:

All observation commands follow the same pattern: command + output mode + filters. Search is a filter operation, so it belongs as a flag, not a standalone command. This maintains consistent command structure.

Foundation for Future Commands:

New observation commands automatically get search capability by implementing the --find flag. Standalone command would require separate search commands for each data type.

Simpler Mental Model:

Users learn one pattern:
```bash
webctl <data-type> <output-mode> --find "search-text"
```

Rather than:
```bash
webctl find "text"           # For HTML only
webctl <data-type> --find    # For other data types?
```

Breaking Change Justified:

Project is in early development. No deprecation period needed. Clean break is better than maintaining compatibility with confusing design.

## Trade-offs

Accept:

- Breaking change from existing find command
- Users must update scripts using find command
- Slightly more verbose (webctl html --find vs webctl find)
- No standalone search command for quick HTML search
- Migration required for existing workflows

Gain:

- Search works across all data types (HTML, CSS, console, network, cookies)
- Clear indication of search scope in command
- Consistent flag-based filtering across all commands
- Composability with other filters and output modes
- Better discoverability through command help
- Simpler mental model (one pattern for all observation)
- Foundation for future observation commands
- Eliminates confusion about search scope

## Alternatives

Keep Find Command and Add --find Flag:

```bash
webctl find "text"               # HTML search (backward compatible)
webctl html --find "text"        # Also HTML search (new flag)
webctl console --find "text"     # Console search (new flag)
```

- Pro: No breaking changes
- Pro: Maintains backward compatibility
- Con: Duplicate functionality for HTML search
- Con: Two ways to do the same thing (confusing)
- Con: Still limited to HTML for standalone command
- Con: Inconsistent interface (command for HTML, flag for others)
- Rejected: Duplication and inconsistency worse than breaking change

Expand Find Command to Support All Data Types:

```bash
webctl find --html "text"        # Search HTML
webctl find --css "text"         # Search CSS
webctl find --console "text"     # Search console
webctl find --network "text"     # Search network
```

- Pro: Single search command for all data
- Pro: No breaking change to find command concept
- Con: Awkward flag-based data type selection
- Con: Doesn't compose with other filters (--type, --status, etc.)
- Con: Doesn't integrate with output modes (show/save)
- Con: Separate from observation command structure
- Rejected: Doesn't integrate with universal observation pattern

Find Subcommand on Each Observation Command:

```bash
webctl html find "text"          # Search HTML
webctl console find "text"       # Search console
webctl network find "text"       # Search network
```

- Pro: Clear search scope
- Pro: Grouped with data source command
- Con: Search is not an "output mode" like show/save
- Con: Doesn't compose with output modes
- Con: Subcommands should be operations, not filters
- Rejected: Search is a filter, not an operation

Deprecation Period for Find Command:

Keep find command with deprecation warning for 3-6 months, then remove.

- Pro: Gentler migration path
- Pro: Time for users to update scripts
- Con: Maintains confusing interface during deprecation
- Con: More work to implement deprecation warnings
- Con: Project is early development (few users affected)
- Rejected: Clean break is better for early development

Rename Find to html-find:

```bash
webctl html-find "text"          # Renamed to clarify scope
# Plus --find flags on other commands
```

- Pro: Makes HTML-only scope clear
- Pro: Smaller breaking change (just rename)
- Con: Still a standalone command for one data type
- Con: Inconsistent with flag-based filtering pattern
- Con: Clutters namespace
- Rejected: Doesn't achieve consistency with universal pattern

## Migration Guide

Old Pattern (DR-017):

```bash
# Search HTML for text
webctl find "login"
webctl find "password" -E  # Extended regex
webctl find "form" -c      # Case-sensitive
webctl find "button" --limit 10
```

New Pattern (DR-030):

```bash
# Search HTML for text
webctl html --find "login"
webctl html show --find "password"
webctl html save ./results.html --find "form"

# Search other data types (new capability)
webctl css --find "button"
webctl console --find "error"
webctl network --find "api/"
webctl cookies --find "session"
```

Migration Steps:

1. Identify all uses of `webctl find` in scripts
2. Replace with `webctl html --find` or `webctl html show --find`
3. Consider if other data types should be searched instead
4. Update to use appropriate observation command + --find flag

Example migrations:

```bash
# Before: Quick HTML search
webctl find "login"

# After: HTML search with stdout output
webctl html show --find "login"

# Before: Save search results
webctl find "form" > results.txt

# After: Save search results to file
webctl html save ./results.html --find "form"
```

## Command Removal Details

Files to Update:

- Remove internal/cli/find.go
- Remove find command from root command registration
- Remove find command from CLI documentation
- Update AGENTS.md to remove find references
- Update examples and tutorials

Implementation removed:

All find command code including:
- Command definition and flags
- HTML search implementation
- Output formatting
- Tests specific to find command

Note: HTML search functionality moves to --find flag implementation in html command. Code is refactored, not deleted.

## Replacement Functionality

The --find flag provides equivalent functionality with additional benefits:

HTML Search (find command replacement):

```bash
# Old
webctl find "text"

# New (equivalent)
webctl html --find "text"
webctl html show --find "text"  # Explicit stdout output
```

Extended Capabilities (new):

```bash
# Search CSS
webctl css --find "background"

# Search console logs
webctl console --find "TypeError"

# Search network requests
webctl network --find "api/user"

# Search cookies
webctl cookies --find "session"

# Combine with other filters
webctl console show --type error --find "undefined"
webctl network --status 5xx --find "api/" --tail 20
```

All observation commands support --find flag with:

- Case-insensitive search (default)
- Works across all output modes (default/show/save)
- Composable with command-specific filters
- Consistent behavior and syntax

## Updates

- 2025-12-28: Initial version (supersedes DR-017)
