# P-018: Observation Commands Interface Redesign

- Status: Proposed
- Started: (not yet started)
- Completed: (not yet completed)

## Overview

Redesign all observation command interfaces (html, css, console, network, cookies) to follow a unified pattern. This eliminates current inconsistencies where commands have different output destinations, different filtering capabilities, and different subcommand structures.

Current state: Commands are inconsistent
- html always outputs to file
- css has 4 subcommands with mixed output (file/stdout)
- console/network output to stdout only
- cookies has no filtering
- find is a standalone command that only searches HTML

Goal: Unified pattern across all observation commands with consistent output control, filtering, and command-specific extensions.

## Goals

1. Document unified observation command pattern in design records
2. Create individual DRs for each command superseding old designs
3. Create implementation projects for each command refactor
4. Establish design principle: universal patterns vs command-specific subcommands
5. Remove standalone find command, integrate as --find flag

## Scope

In Scope:
- HTML command interface (supersedes DR-012, DR-021)
- CSS command interface (supersedes DR-023)
- Console command interface (supersedes DR-007)
- Network command interface (supersedes DR-009)
- Cookies command interface (supersedes DR-015)
- Remove standalone find command (supersedes DR-017)
- Screenshot command alignment (follows pattern without show subcommand)

Out of Scope:
- Eval command (different purpose - JS execution, not observation)
- Target command (session management, not observation)
- Status command (daemon status, not page observation)
- Navigation/interaction commands (click, type, etc. - not observation)
- Implementation of the new interfaces (covered in separate projects)

## Design Principles

Universal Pattern Principle:
- Universal patterns (output modes, filtering) → consistent across all commands
- Command-specific operations → unique subcommands for that command only

Output Pattern:
```bash
webctl <cmd>              # Default: save to temp with auto-generated name
webctl <cmd> show         # Explicit: output to stdout
webctl <cmd> save <path>  # Explicit: save to custom path (required arg, not -o flag)
```

Path Handling:
- If path is a file → save to that file
- If path is a directory → save with auto-generated filename in that directory
- Auto-generated pattern: /tmp/webctl-<cmd>/YY-MM-DD-HHMMSS-{identifier}.{ext}

Universal Flags (all commands):
```bash
--select, -s SELECTOR    # Filter to element/selector (html, css only)
--find, -f TEXT          # Text search/filter (all commands)
--raw                    # Skip formatting/pretty-printing
--json                   # JSON output (global flag)
```

Command-Specific:
- Filters as flags (--type, --status, --method, --domain, etc.)
- Operations as subcommands (css computed, css get, css inject, cookies set, cookies delete)

## Success Criteria

- [ ] Created DR-024: HTML Command Interface
- [ ] Created DR-025: CSS Command Interface
- [ ] Created DR-026: Console Command Interface
- [ ] Created DR-027: Network Command Interface
- [ ] Created DR-028: Cookies Command Interface
- [ ] Created DR-029: Find Command Removal
- [ ] Updated all superseded DRs (DR-007, DR-009, DR-012, DR-015, DR-017, DR-021, DR-023) to "Superseded" status
- [ ] Moved superseded DRs to design-records/superseded/
- [ ] Created P-019: HTML Command Implementation
- [ ] Created P-020: CSS Command Implementation
- [ ] Created P-021: Console Command Implementation
- [ ] Created P-022: Network Command Implementation
- [ ] Created P-023: Cookies Command Implementation
- [ ] Updated AGENTS.md with new command patterns

## Deliverables

Design Records:
- docs/design/design-records/dr-024-html-command-interface.md
- docs/design/design-records/dr-025-css-command-interface.md
- docs/design/design-records/dr-026-console-command-interface.md
- docs/design/design-records/dr-027-network-command-interface.md
- docs/design/design-records/dr-028-cookies-command-interface.md
- docs/design/design-records/dr-029-find-command-removal.md

Superseded DRs (moved to superseded/):
- docs/design/design-records/superseded/dr-007-console-command.md
- docs/design/design-records/superseded/dr-009-network-command.md
- docs/design/design-records/superseded/dr-012-html-command.md
- docs/design/design-records/superseded/dr-015-cookies-command.md
- docs/design/design-records/superseded/dr-017-find-command.md
- docs/design/design-records/superseded/dr-021-html-formatting-find.md
- docs/design/design-records/superseded/dr-023-css-commands.md

Implementation Projects:
- docs/projects/p-019-html-command-implementation.md
- docs/projects/p-020-css-command-implementation.md
- docs/projects/p-021-console-command-implementation.md
- docs/projects/p-022-network-command-implementation.md
- docs/projects/p-023-cookies-command-implementation.md

## Detailed Command Specifications

Reference documents in docs/design/interface/:
- html.md - Complete HTML command specification
- css.md - Complete CSS command specification
- console.md - Complete Console command specification
- network.md - Complete Network command specification
- cookies.md - Complete Cookies command specification

### HTML Command (DR-024)

Universal Pattern Only - No HTML-specific subcommands

```bash
# Default: save to temp
webctl html
# → /tmp/webctl-html/25-12-28-HHMMSS-page-title.html

# Show: output to stdout
webctl html show

# Save: save to custom path
webctl html save <path>
webctl html save ./output/              # Directory: auto-generate filename
webctl html save ./page.html            # File: use exact path
```

Universal Flags:
- --select, -s SELECTOR - Filter to element(s), outer HTML
- --find, -f TEXT - Search within HTML
- --raw - Skip formatting
- --json - JSON output

Examples:
```bash
webctl html --select "#main"
webctl html show --select "#main" --find "login"
webctl html save ./output/ --select "form"
```

Supersedes: DR-012 (HTML Command Interface), DR-021 (HTML Formatting Find)

Reference: docs/design/interface/html.md

### CSS Command (DR-025)

Universal Pattern + CSS-Specific Subcommands

```bash
# Universal pattern (all stylesheets)
webctl css                              # → temp
webctl css show                         # → stdout
webctl css save <path>                  # → custom path

# With filters
webctl css --select ".button"           # Computed styles for element
webctl css show --find "background"     # Search in CSS
```

Universal Flags:
- --select, -s SELECTOR - Filter to element's computed styles
- --find, -f TEXT - Search within CSS
- --raw - Skip formatting
- --json - JSON output

CSS-Specific Subcommands:
```bash
webctl css computed <selector>          # All computed styles → stdout
webctl css get <selector> <property>    # Single property value → stdout
webctl css inject <css>                 # Inject CSS (mutation)
webctl css inject --file <path>
```

Rationale for subcommands:
- computed/get/inject are CSS-specific operations
- Don't apply to html, console, network, or cookies
- Keep as separate subcommands per design principle

Examples:
```bash
webctl css save ./styles.css
webctl css show --select ".button"
webctl css computed ".button"
webctl css get ".button" background-color
webctl css inject "body { background: red; }"
```

Supersedes: DR-023 (CSS Commands)

Reference: docs/design/interface/css.md

### Console Command (DR-026)

Universal Pattern + Console-Specific Filter Flags

```bash
# Universal pattern
webctl console                          # → temp
webctl console show                     # → stdout
webctl console save <path>              # → custom path
```

Universal Flags:
- --find, -f TEXT - Search within log messages
- --raw - Skip formatting
- --json - JSON output

Console-Specific Flags:
- --type TYPE - Filter by type: log, warn, error, debug, info (repeatable, CSV)
- --head N - First N entries
- --tail N - Last N entries
- --range N-M - Entries N through M (mutually exclusive with head/tail)

Examples:
```bash
webctl console show --type error
webctl console show --find "TypeError" --type error
webctl console save ./errors.json --type error --tail 50
webctl console show --head 20
```

Supersedes: DR-007 (Console Command)

Reference: docs/design/interface/console.md

### Network Command (DR-027)

Universal Pattern + Network-Specific Filter Flags

```bash
# Universal pattern
webctl network                          # → temp
webctl network show                     # → stdout
webctl network save <path>              # → custom path
```

Universal Flags:
- --find, -f TEXT - Search in URLs/bodies
- --raw - Skip formatting
- --json - JSON output

Network-Specific Flags:
- --type TYPE - CDP resource type: xhr, fetch, document, script, etc.
- --method METHOD - HTTP method: GET, POST, PUT, DELETE, etc.
- --status CODE - Status code or range: 200, 4xx, 5xx, 200-299
- --url PATTERN - URL regex pattern
- --mime TYPE - MIME type: application/json, text/html, etc.
- --min-duration DURATION - Minimum duration: 1s, 500ms
- --min-size BYTES - Minimum response size
- --failed - Only failed requests
- --head N / --tail N / --range N-M - Limit results

All repeatable/CSV-supported except --url, --min-duration, --min-size, --failed

Examples:
```bash
webctl network show --status 4xx,5xx
webctl network show --find "api/" --method POST
webctl network save ./api-errors.json --status 5xx --url "api/"
webctl network show --min-duration 1s --tail 20
```

Supersedes: DR-009 (Network Command)

Reference: docs/design/interface/network.md

### Cookies Command (DR-028)

Universal Pattern + Mutation Subcommands

```bash
# Universal pattern (observation)
webctl cookies                          # → temp
webctl cookies show                     # → stdout
webctl cookies save <path>              # → custom path
```

Universal Flags:
- --find, -f TEXT - Search in cookie names/values
- --raw - Skip formatting
- --json - JSON output

Cookies-Specific Flags:
- --domain DOMAIN - Filter by domain
- --name NAME - Filter by name

Cookies-Specific Subcommands (mutations):
```bash
webctl cookies set <name> <value>
  --domain, --path, --secure, --httponly, --max-age, --samesite

webctl cookies delete <name>
  --domain (required if ambiguous)
```

Rationale:
- set/delete are mutations, not observations
- Keep separate from observation pattern (default/show/save)

Examples:
```bash
webctl cookies show --domain ".github.com"
webctl cookies --find "session"
webctl cookies save ./auth-cookies.json --find "token"
webctl cookies set session abc123 --secure --httponly
webctl cookies delete session
```

Supersedes: DR-015 (Cookies Command)

Reference: docs/design/interface/cookies.md

### Find Command Removal (DR-029)

Remove standalone find command completely.

Current: webctl find <text> - searches HTML only
New: --find flag available on all observation commands

Rationale:
- Standalone find only searches HTML, causing confusion
- Users want to search console logs, network requests, CSS, cookies
- Universal --find flag provides filtering across all data sources
- More discoverable and consistent

Migration:
```bash
# Old
webctl find "login"

# New
webctl html --find "login"              # Search HTML
webctl css --find "button"              # Search CSS
webctl console --find "error"           # Search logs
webctl network --find "api/"            # Search requests
webctl cookies --find "session"         # Search cookies
```

Breaking Change:
- Remove webctl find command entirely
- No deprecation period (project in early development)
- Update documentation and examples

Supersedes: DR-017 (Find Command)

### Screenshot Command Alignment

Screenshot follows the pattern but without show subcommand (binary data):

```bash
webctl screenshot                       # → temp
webctl screenshot save <path>           # → custom path
# No show subcommand (PNG binary, not suitable for stdout)
```

Flags:
- --full-page - Capture full scrollable page
- --output removed (use save <path> instead)

No new DR needed - update existing screenshot implementation to match pattern.

## Implementation Project Templates

Each implementation project (P-019 through P-023) should include:

Goals:
1. Implement new command interface per DR-NNN
2. Update CLI command file (internal/cli/<cmd>.go)
3. Update daemon handlers if needed (internal/daemon/handlers_<cmd>.go)
4. Update IPC protocol if needed (internal/ipc/protocol.go)
5. Add/update tests
6. Update CLI documentation

Success Criteria:
- [ ] Default (no subcommand) saves to temp with auto-generated filename
- [ ] show subcommand outputs to stdout
- [ ] save <path> subcommand saves to custom path
- [ ] Directory paths auto-generate filenames
- [ ] Universal flags work (--find, --raw, --json)
- [ ] Command-specific flags/subcommands work
- [ ] All tests pass
- [ ] Documentation updated

Deliverables:
- Updated internal/cli/<cmd>.go
- Updated tests
- Updated documentation

Technical Approach:
- Refactor command to use Cobra subcommands (show, save)
- Default RunE function implements save-to-temp behavior
- Add universal flags to root command
- Add command-specific flags/subcommands as needed
- Update output formatting to support text and JSON modes
- Handle directory vs file path detection for save subcommand
- Maintain backward compatibility where possible (flag values, output format)

## Technical Approach

For creating DRs:

1. Use DR template from docs/design/dr-writing-guide.md
2. Each DR documents one command's interface
3. Include Problem, Decision, Why, Trade-offs, Alternatives sections
4. Reference docs/design/interface/<cmd>.md for complete specification
5. Mark as "Accepted" status (design is locked in)
6. No bold formatting (per DR guide)

For superseding old DRs:

1. Update old DR header: Status: Superseded by DR-NNN
2. Move to docs/design/design-records/superseded/
3. Update link to use ../dr-NNN-title.md (up one directory)
4. Update design-records/README.md index

For creating implementation projects:

1. Use Project template from docs/projects/p-writing-guide.md
2. Reference the corresponding DR
3. Include specific deliverables (files to modify)
4. Clear success criteria (testable outcomes)
5. Status: Proposed

## Dependencies

None - this is a foundational redesign.

## Questions & Uncertainties

None - design is locked in via interface documents.

## Notes

Design conversation preserved in:
- docs/design/interface/command-comparison.md
- docs/design/interface/observation-commands-analysis.md
- docs/design/interface/unified-proposal.md

These documents capture the analysis and decision-making process that led to the locked interface designs.

Context usage concern: Individual DRs chosen over single comprehensive DR to minimize AI agent context requirements when working on specific commands.

## Updates

- 2025-12-28: Project created with complete specifications
