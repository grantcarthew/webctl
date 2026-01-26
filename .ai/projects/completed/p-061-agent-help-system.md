# P-061: Agent Help System

Project: Implement token-efficient help system for AI agents

Status: Completed
Started: 2026-01-25
Completed: 2026-01-26
Active: No

## Overview

Create a comprehensive help system optimized for AI agent token efficiency. The help content uses minimal formatting, focuses on command examples, and provides all necessary information in a compact format.

## Context

AI agents using webctl need quick reference documentation that:
- Minimizes token usage (no bold, italic, emojis, excessive nesting)
- Provides complete command syntax and flags
- Shows practical examples without verbose explanations
- Is embedded in the binary (works without source tree)

## Design Principles

Token-efficient markdown (from ~/.ai/environment.md):

Do not use:
- Bold or italic formatting
- Horizontal rules
- Emojis
- HTML comments
- Image embeds
- Multiple consecutive blank lines
- Nested lists beyond 3 levels
- Task lists
- Heading depth beyond ###

Keep:
- Headings for structure
- Single blank lines between sections
- Inline code and code blocks
- Tables for structured data
- Lists (ordered and unordered)
- Callout prefixes (Note:, Warning:, Important:) without bold

## Implementation Details

### Architecture

Help content flow:
1. Markdown files authored in `.ai/agent-help/` (source of truth)
2. Copied to `internal/cli/agent-help/` for embedding
3. Embedded in binary via `//go:embed` directive
4. Accessed via `webctl help agents [topic]` command

### File Structure

```
.ai/agent-help/
  overview.md          # Main help (COMPLETED)
  workflow.md          # Common patterns (TODO)
  observe.md           # Observation commands (TODO)
  interact.md          # Interaction commands (TODO)
  wait.md              # Ready command modes (TODO)
  debug.md             # Debugging help (TODO)
  errors.md            # Common errors (TODO)
  output.md            # Output modes (TODO)
  filters.md           # Filter flags (TODO)
  selectors.md         # CSS selectors (TODO)

internal/cli/agent-help/
  README.md            # Documents the copy workflow
  overview.md          # Embedded copy
  [other files...]     # Embedded copies

internal/cli/
  help_agents.go       # Help command implementation
  root.go              # Modified to register help subcommand
```

### Code References

#### help_agents.go (internal/cli/help_agents.go)

```go
package cli

import (
	_ "embed"
	"fmt"

	"github.com/spf13/cobra"
)

//go:embed agent-help/overview.md
var agentHelpOverview string

// Add more embed directives here as topics are created:
// //go:embed agent-help/workflow.md
// var agentHelpWorkflow string

var helpAgentsCmd = &cobra.Command{
	Use:   "agents [topic]",
	Short: "Token-efficient help for AI agents",
	Long:  "Displays token-efficient help documentation designed for AI agent consumption.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runHelpAgents,
}

func runHelpAgents(cmd *cobra.Command, args []string) error {
	// If no topic specified, show overview
	if len(args) == 0 {
		fmt.Println(agentHelpOverview)
		return nil
	}

	topic := args[0]

	// Map topics to their content
	topics := map[string]string{
		// Add new topics here as they are created:
		// "workflow":  agentHelpWorkflow,
		// "observe":   agentHelpObserve,
		// etc.
	}

	content, exists := topics[topic]
	if !exists {
		return fmt.Errorf("unknown help topic: %s\n\nAvailable topics:\n  workflow, observe, interact, wait, debug, errors, output, filters, selectors", topic)
	}

	fmt.Println(content)
	return nil
}
```

#### root.go modifications (internal/cli/root.go)

Lines 77-83 - Update Long description:
```go
var rootCmd = &cobra.Command{
	Use:   "webctl",
	Short: "Browser automation CLI for AI agents",
	Long: `webctl captures DevTools data (console logs, network requests, JS errors) via a persistent daemon that buffers CDP events.

For AI agents, see: webctl help agents`,
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
}
```

Lines 86-98 - Register help subcommand:
```go
func init() {
	rootCmd.PersistentFlags().BoolVar(&Debug, "debug", false, "Enable verbose debug output")
	rootCmd.PersistentFlags().BoolVar(&JSONOutput, "json", false, "Output in JSON format (default is text)")
	rootCmd.PersistentFlags().BoolVar(&NoColor, "no-color", false, "Disable color output")
	rootCmd.SetVersionTemplate(`webctl version {{.Version}}
Repository: https://github.com/grantcarthew/webctl
Report issues: https://github.com/grantcarthew/webctl/issues/new
`)

	// Add agents subcommand to help
	// This makes "webctl help agents" work properly
	rootCmd.InitDefaultHelpCmd()
	helpCmd, _, _ := rootCmd.Find([]string{"help"})
	if helpCmd != nil {
		helpCmd.AddCommand(helpAgentsCmd)
	}
}
```

## Completed: overview.md

File: `.ai/agent-help/overview.md`

### Content Structure

1. Title and key principles (2 lines)
   - "stdout is token efficient"
   - "Use --json for structured data selection"

2. Quick Start (32 line code block)
   - Complete workflow from start to stop
   - Shows all major command types

3. Global Flags (4 flags)
   - --debug, --json, --no-color

4. Command Flags (single flat list, ~100 lines)
   - All commands with their flags
   - No nested categories
   - Grouped by command similarity

5. Help Topics (index of additional help)
   - Lists all available topics

### Key Features

- No introductory prose
- All commands in code blocks
- Flags shown with syntax examples
- Total size: ~117 lines (very token efficient)

## Remaining Topics

### 1. workflow.md

Purpose: Common automation patterns

Content structure:
```
# Agent Workflow Patterns

## Lifecycle

(code block: start, stop, status commands)

## Basic Navigation

(code block: navigate, ready, screenshot sequence)

## Form Interaction

(code block: type, click, ready pattern)

## Data Extraction

(code block: navigate, ready, html/console/network patterns)

## Error Checking

(code block: console --type error, network --status 4xx patterns)

## Multi-page Workflows

(code block: navigate, extract, navigate, extract patterns)
```

Estimated lines: 80-100

### 2. observe.md

Purpose: All observation commands with complete flag reference

Content structure:
```
# Observation Commands

Commands extract data from browser. Default output: stdout.

## html

(code block: all html variations)

## css

(code block: all css variations)

## console

(code block: all console variations)

## network

(code block: all network variations)

## cookies

(code block: all cookies variations)

## screenshot

(code block: screenshot variations)

## eval

(code block: eval variations)
```

Estimated lines: 120-150

Design records to reference:
- DR-025: HTML Command Interface
- DR-026: CSS Command Interface
- DR-027: Console Command Interface
- DR-028: Network Command Interface
- DR-029: Cookies Command Interface
- DR-011: Screenshot Command
- DR-014: Eval Command Interface

### 3. interact.md

Purpose: All interaction commands

Content structure:
```
# Interaction Commands

Commands modify browser state or simulate user actions.

## click

(code block: click examples)

## type

(code block: type with --key, --clear examples)

## key

(code block: key with modifiers)

## select

(code block: select dropdown)

## scroll

(code block: scroll to element, position, offset)

## focus

(code block: focus element)

## Common Patterns

(code block: form filling, keyboard navigation)
```

Estimated lines: 80-100

Design records to reference:
- DR-013: Navigation & Interaction Commands

### 4. wait.md

Purpose: Ready command comprehensive guide

Content structure:
```
# Wait Strategies

## Page Load Mode

(code block: ready examples)

## Selector Mode

(code block: ready with selector)

## Network Idle Mode

(code block: ready --network-idle)

## Eval Mode

(code block: ready --eval)

## When to Use Each Mode

Table showing use cases

## Chaining Waits

(code block: multiple ready commands)
```

Estimated lines: 70-90

Design records to reference:
- DR-020: Ready Command Extensions

### 5. debug.md

Purpose: Debugging and troubleshooting

Content structure:
```
# Debugging

## Debug Flag

(code block: webctl --debug command examples)

Debug output shows:
- Request parameters
- Response timing
- Filter operations
- File I/O

## Common Issues

Start fails:
(code block: error and solution)

Navigation timeout:
(code block: error and solution)

Element not found:
(code block: error and solution)

## Checking State

(code block: status, console, network commands for debugging)
```

Estimated lines: 60-80

### 6. errors.md

Purpose: Error messages and solutions

Content structure:
```
# Common Errors

## Daemon Errors

Error: daemon not running
(solution)

Error: daemon is already running
(solution)

## Navigation Errors

Error: net::ERR_NAME_NOT_RESOLVED
(solution)

Error: timeout waiting for page load
(solution)

## Interaction Errors

Error: element not found
(solution)

Error: no elements found
(solution)

## State Errors

Error: no active session
(solution)

Error: no previous page in history
(solution)
```

Estimated lines: 80-100

Grep codebase for error messages:
```bash
grep -r "return outputError" internal/cli/*.go
grep -r "return outputNotice" internal/cli/*.go
```

### 7. output.md

Purpose: Output modes and file handling

Content structure:
```
# Output Modes

## Default: stdout

(code block: command | jq examples)

## Save Mode

(code block: command save patterns)

## Path Conventions

Trailing slash:
(examples)

Exact path:
(examples)

Auto-generated filenames:
(examples)

## JSON Output

(code block: --json examples)

## Raw Output

(code block: --raw examples)
```

Estimated lines: 60-80

### 8. filters.md

Purpose: All filter flags explained

Content structure:
```
# Filter Flags

## Universal Filters

--select, -s
(examples across html, css)

--find, -f
(examples across html, console, network, cookies)

--raw
(examples)

## HTML/CSS Filters

--before, -B
--after, -A
--context, -C
(examples)

## Console Filters

--type
--head
--tail
--range
(examples)

## Network Filters

--type
--method
--status
--url
--mime
--min-duration
--min-size
--failed
(examples)

## Cookies Filters

--domain
--name
(examples)
```

Estimated lines: 100-120

### 9. selectors.md

Purpose: CSS selector reference

Content structure:
```
# CSS Selectors

## Basic Selectors

Tag: div
Class: .class-name
ID: #element-id
Attribute: [name=value]

(examples)

## Combinators

Descendant: div p
Child: div > p
Sibling: div + p
General sibling: div ~ p

(examples)

## Pseudo-classes

:first-child
:last-child
:nth-child(n)
:enabled
:disabled
:checked

(examples)

## Attribute Selectors

[attr]
[attr=value]
[attr^=value]
[attr$=value]
[attr*=value]

(examples)

## Common Patterns

Forms:
(examples)

Navigation:
(examples)

Data attributes:
(examples)
```

Estimated lines: 80-100

## Workflow for Adding New Topics

### 1. Create markdown file

```bash
# Create in source location
cat > .ai/agent-help/workflow.md << 'EOF'
# Agent Workflow Patterns

[content here]
EOF
```

### 2. Copy to embedding location

```bash
cp .ai/agent-help/workflow.md internal/cli/agent-help/
```

### 3. Add embed directive

Edit `internal/cli/help_agents.go`:

```go
//go:embed agent-help/workflow.md
var agentHelpWorkflow string
```

### 4. Add to topics map

Edit `runHelpAgents()` function:

```go
topics := map[string]string{
	"workflow":  agentHelpWorkflow,
	// ... other topics
}
```

### 5. Build and test

```bash
go build -o webctl ./cmd/webctl
./webctl help agents workflow
```

### 6. Update error message

If adding new topic, update the error message in `runHelpAgents()` to include it in available topics list.

## Testing Checklist

For each new topic:

- [ ] File created in `.ai/agent-help/`
- [ ] File copied to `internal/cli/agent-help/`
- [ ] Embed directive added
- [ ] Topic added to map
- [ ] Error message updated
- [ ] Binary builds without errors
- [ ] `webctl help agents` lists the topic
- [ ] `webctl help agents <topic>` displays content
- [ ] Content follows token-efficiency rules
- [ ] All commands in content are verified against actual implementation
- [ ] No markdown violations (check with linter if available)

## Design Record References

Core architecture:
- DR-001: Core Architecture
- DR-002: CLI Browser Commands

Commands:
- DR-013: Navigation & Interaction Commands
- DR-014: Eval Command Interface
- DR-020: Ready Command Extensions
- DR-025: HTML Command Interface
- DR-026: CSS Command Interface
- DR-027: Console Command Interface
- DR-028: Network Command Interface
- DR-029: Cookies Command Interface

Output:
- DR-018: Text Output Format
- DR-019: CLI Terminal Colors

## Success Criteria

- [ ] All 10 help topics implemented
- [ ] All content follows token-efficiency rules
- [ ] All commands verified against implementation
- [ ] Help system accessible via `webctl help agents [topic]`
- [ ] Documentation includes workflow for future additions
- [ ] No markdown formatting violations
- [ ] Total token count for all topics < 5000 tokens

## Current Status

Completed:
- ✅ overview.md (117 lines)
- ✅ Help command infrastructure
- ✅ Embedding system
- ✅ Copy workflow documented

Remaining (in priority order):
1. workflow.md - Most immediately useful for agents
2. observe.md - Comprehensive observation command reference
3. interact.md - Comprehensive interaction command reference
4. wait.md - Ready command deep dive
5. errors.md - Troubleshooting reference
6. debug.md - Debug flag usage
7. output.md - Output mode details
8. filters.md - Filter flag reference
9. selectors.md - CSS selector guide

## Notes

- Keep each topic self-contained
- Cross-reference sparingly (adds tokens)
- Focus on examples over explanations
- Verify all command syntax against actual code before writing
- Test each topic after creation
- Update this document as work progresses
