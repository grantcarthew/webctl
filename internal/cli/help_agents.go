package cli

import (
	_ "embed"
	"fmt"

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

	// Special case: "all" concatenates all topics
	if topic == "all" {
		fmt.Println(agentHelpOverview)
		fmt.Println("---")
		fmt.Println(agentHelpWorkflow)
		fmt.Println("---")
		fmt.Println(agentHelpObserve)
		fmt.Println("---")
		fmt.Println(agentHelpInteract)
		fmt.Println("---")
		fmt.Println(agentHelpWait)
		fmt.Println("---")
		fmt.Println(agentHelpErrors)
		fmt.Println("---")
		fmt.Println(agentHelpOutput)
		fmt.Println("---")
		fmt.Println(agentHelpServe)
		return nil
	}

	// Map topics to their content (will add more as we create them)
	topics := map[string]string{
		"workflow": agentHelpWorkflow,
		"observe":  agentHelpObserve,
		"interact": agentHelpInteract,
		"wait":     agentHelpWait,
		"errors":   agentHelpErrors,
		"output":   agentHelpOutput,
		"serve":    agentHelpServe,
	}

	content, exists := topics[topic]
	if !exists {
		return fmt.Errorf("unknown help topic: %s\n\nAvailable topics:\n  workflow, observe, interact, wait, errors, output, serve, all", topic)
	}

	fmt.Println(content)
	return nil
}
