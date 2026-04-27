# Agent Help Files

These help files are embedded in the webctl binary at build time via `//go:embed` directives in `internal/cli/help_agents.go`.

To update help content:
1. Edit the relevant `.md` file in this directory
2. Rebuild: `go build ./cmd/webctl`

Topics are exposed via `webctl help <topic>` and `webctl help all`.
