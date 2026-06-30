# Remove Redundant Raw Flag

## Goal

Remove the `--raw` flag from the commands where it merely duplicates `--json`, eliminating a second, confusing spelling for identical behavior. Leave `--raw` intact on the commands where it has a distinct meaning.

## Scope

In scope:

- Remove `--raw` from the console, network, and cookies commands: the flag declaration and all of its handling.
- On these three commands `--raw` means skip formatting and return raw JSON, which is identical to `--json`.

Out of scope:

- The css and html commands. Their `--raw` means skip CSS/HTML formatting and return unformatted source, a genuine and distinct feature. Do not touch it.
- Any other flag, behavior, or output change.

## Current State

Five commands declare a `--raw` flag, in two different meanings.

- internal/cli/console.go, internal/cli/network.go, internal/cli/cookies.go each declare `PersistentFlags().Bool("raw", false, "Skip formatting (return raw JSON)")` and read it with a parent-flag fallback, then route to the same JSON output that `--json` produces. For network this is confirmed: both `--raw` and `--json` route through `outputNetworkJSON`. Console and cookies follow the same pattern.
- internal/cli/css.go and internal/cli/html.go declare a `--raw` flag described as skip CSS/HTML formatting. This returns unformatted source instead of the pretty-printed output and is not equivalent to `--json`.
- internal/cli/cli_test.go sets the console `raw` flag in a test (around line 1069) and resets it in cleanup.

Removing `--raw` from the three JSON commands leaves `--json` as the single way to request JSON, with no behavior lost.

## Requirements

1. Remove the `--raw` flag and all of its handling from console, network, and cookies. After removal, passing `--raw` to these commands is an unknown-flag error, and `--json` is the only way to request JSON.
2. Before removing it from each command, confirm that command's `--raw` is truly equivalent to `--json` (routes to the same JSON output). If any one differs, stop and report rather than change behavior.
3. Do not modify `--raw` on css or html.
4. Update or remove any test that sets the removed flag, including the console `raw` reference in internal/cli/cli_test.go.
5. Update the human docs for the affected commands (docs/console.md, docs/network.md, docs/cookies.md) and any agent-help topic to drop `--raw`.

## Constraints

- Pure Go, no new dependencies. gofmt and go vet clean.
- Use the existing output helpers; do not change how JSON is produced, only how it is requested.
- Removing a flag also removes any per-command reset of it in the REPL flag-reset path; ensure no dangling reset remains.
- Follow the documentation conventions in AGENTS.md.

## Implementation Plan

1. Verify `--raw` equals `--json` on console, network, and cookies by reading each command's flag handling.
2. Remove the flag declaration and its read/branch in each of the three commands.
3. Remove the cli_test.go reference and any other test setup or assertion for the removed flag.
4. Update the three command docs and any agent-help topic.

## Implementation Guidance

This project lands before the network and console redesign projects, which edit the same command files. Doing the removal first keeps it from colliding with those larger changes.

The verification step is not a formality. The point of the project is that these three flags are pure duplicates; if one has quietly diverged, removing it would change behavior and must be surfaced, not absorbed.

## Acceptance Criteria

- console, network, and cookies reject `--raw` as an unknown flag.
- css and html still accept `--raw` with unchanged behavior.
- `--json` produces the same JSON these three commands previously produced via `--raw`.
- No test references the removed flag.
- The three command docs no longer mention `--raw`.
