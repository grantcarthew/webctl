# Context Directory

AI reference documentation and source code for the `webctl` project. This directory contains cloned repositories, fetched documentation, and local assets that provide context for AI agents working on this codebase.

## Structure

| Directory         | Description                                                           | Source                                                |
| ----------------- | --------------------------------------------------------------------- | ----------------------------------------------------- |
| rod               | Go library for CDP browser automation (reference implementation)      | <https://github.com/go-rod/rod>                       |
| devtools-protocol | Chrome DevTools Protocol specification and JSON schemas               | <https://github.com/ChromeDevTools/devtools-protocol> |
| docs              | Fetched CDP domain documentation (Runtime, Network, Page, DOM, Input) | See [docs/README.md](docs/README.md)                  |

## Rebuild

Run the upsert script to clone missing repos, pull existing ones, and refresh fetched docs:

```bash
./scripts/upsert-context
```
