package cli

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed agent-help/overview.md
var agentHelpOverview string

//go:embed agent-help/workflow.md
var agentHelpWorkflow string

//go:embed agent-help/observe.md
var agentHelpObserve string

//go:embed agent-help/interact.md
var agentHelpInteract string

//go:embed agent-help/wait.md
var agentHelpWait string

//go:embed agent-help/errors.md
var agentHelpErrors string

//go:embed agent-help/output.md
var agentHelpOutput string

//go:embed agent-help/serve.md
var agentHelpServe string

type helpTopic struct {
	name    string
	short   string
	content *string
}

// helpTopics lists the AI-agent help topics in display order.
var helpTopics = []helpTopic{
	{"agents", "Overview and command map for AI agents", &agentHelpOverview},
	{"workflow", "Common automation workflow patterns", &agentHelpWorkflow},
	{"observe", "Observation commands (html, css, console, network, cookies)", &agentHelpObserve},
	{"interact", "Interaction commands (click, type, key, select, scroll, focus)", &agentHelpInteract},
	{"wait", "Synchronization with the ready command", &agentHelpWait},
	{"errors", "Common errors and their solutions", &agentHelpErrors},
	{"output", "Output modes (stdout, save, JSON)", &agentHelpOutput},
	{"serve", "Local development server", &agentHelpServe},
}

// agentHelpTopicsBlock renders the topic list for the root `--help` template.
func agentHelpTopicsBlock() string {
	var b strings.Builder
	b.WriteString("AI agent help topics (use 'webctl help <topic>'):\n")
	for _, t := range helpTopics {
		b.WriteString(fmt.Sprintf("  %-11s %s\n", t.name, t.short))
	}
	b.WriteString("  all         All topics combined")
	return b.String()
}

// registerHelpTopics adds each agent help topic as a subcommand of the help
// command, plus an `all` subcommand that concatenates every topic.
func registerHelpTopics(helpCmd *cobra.Command) {
	for _, t := range helpTopics {
		t := t
		helpCmd.AddCommand(&cobra.Command{
			Use:   t.name,
			Short: t.short,
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println(*t.content)
			},
		})
	}

	helpCmd.AddCommand(&cobra.Command{
		Use:   "all",
		Short: "All agent help topics combined",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			for i, t := range helpTopics {
				if i > 0 {
					fmt.Println("---")
				}
				fmt.Println(*t.content)
			}
		},
	})
}
