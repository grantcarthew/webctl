# Agent Help Files

These help files are embedded in the webctl binary at build time.

Source of truth: `.ai/agent-help/`

To update help content:
1. Edit files in `.ai/agent-help/`
2. Copy to this directory: `cp .ai/agent-help/*.md internal/cli/agent-help/`
3. Rebuild: `go build ./cmd/webctl`

Note: This directory contains copies for embedding. Always edit the originals in `.ai/agent-help/`.
