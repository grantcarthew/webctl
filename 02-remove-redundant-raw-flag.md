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
- Each of these three commands also hand-types a `--raw` line inside its `Long` help string (console.go:30, network.go:33, cookies.go:31, all reading `  --raw             Skip formatting (return raw JSON)` under "Universal flags"). This string is what `webctl <command> --help` prints. Cobra does not generate it from the flag set, so removing the flag declaration does not remove this line.
- internal/cli/css.go and internal/cli/html.go declare a `--raw` flag described as skip CSS/HTML formatting. This returns unformatted source instead of the pretty-printed output and is not equivalent to `--json`.
- internal/cli/cli_test.go sets the console `raw` flag in a test (around line 1069) and resets it in cleanup.
- scripts/interactive/test-console.sh, test-network.sh, and test-cookies.sh each invoke `--raw` on their command (test-console.sh lines 298 and 305, test-network.sh lines 450 and 457, test-cookies.sh lines 174 and 181). test-css.sh and test-html.sh also use `--raw`, but theirs is the distinct CSS/HTML feature and stays.

Removing `--raw` from the three JSON commands leaves `--json` as the single way to request JSON, with no behavior lost.

## Requirements

1. Remove the `--raw` flag and all of its handling from console, network, and cookies. This includes the flag declaration, the read/branch, and the hand-typed `--raw` line in each command's `Long` help string (console.go:30, network.go:33, cookies.go:31). After removal, passing `--raw` to these commands is an unknown-flag error, `webctl <command> --help` no longer lists it, and `--json` is the only way to request JSON.
2. Before removing it from each command, confirm that command's `--raw` is truly equivalent to `--json` (routes to the same JSON output). If any one differs, stop and report rather than change behavior.
3. Do not modify `--raw` on css or html.
4. Update or remove any test that uses the removed flag:
   - internal/cli/cli_test.go: the console `raw` reference around line 1069. Switch it to JSON mode (the test parses JSON output) rather than deleting the coverage.
   - scripts/interactive/test-console.sh, test-network.sh, test-cookies.sh: replace each `--raw` invocation with `--json`, the surviving equivalent.
   - Do not touch the `--raw` invocations in scripts/interactive/test-css.sh and test-html.sh.
5. Update the documentation that presents `--raw` for these commands. There are no per-command human docs for console, network, or cookies in `docs/`. The surfaces are the in-command `Long` help strings (covered by Requirement 1) and the agent-help markdown:
   - internal/cli/agent-help/observe.md: remove the `webctl console --raw`, `webctl network --raw`, and `webctl cookies --raw` example lines. Leave the html and css `--raw` examples.
   - internal/cli/agent-help/output.md: line 34 currently reads "--raw applies to html, css, console, network, cookies (not markdown)." Correct it so `--raw` is listed only for html and css.

## Constraints

- Pure Go, no new dependencies. gofmt and go vet clean.
- Use the existing output helpers; do not change how JSON is produced, only how it is requested.
- Removing a flag also removes any per-command reset of it in the REPL flag-reset path; ensure no dangling reset remains.
- Follow the documentation conventions in AGENTS.md.

## Implementation Plan

1. Verify `--raw` equals `--json` on console, network, and cookies by reading each command's flag handling.
2. Remove the flag declaration, its read/branch, and the `--raw` line in the `Long` help string in each of the three commands.
3. Update the test surface per Requirement 4: switch the cli_test.go console reference to JSON mode, and replace `--raw` with `--json` in the three interactive scripts.
4. Update the agent-help topics observe.md and output.md (see Requirement 5).

## Implementation Guidance

This project lands before the network and console redesign projects, which edit the same command files. Doing the removal first keeps it from colliding with those larger changes.

The verification step is not a formality. The point of the project is that these three flags are pure duplicates; if one has quietly diverged, removing it would change behavior and must be surfaced, not absorbed.

## Acceptance Criteria

- console, network, and cookies reject `--raw` as an unknown flag.
- `webctl console --help`, `webctl network --help`, and `webctl cookies --help` no longer list `--raw`.
- css and html still accept `--raw` with unchanged behavior.
- `--json` produces the same JSON these three commands previously produced via `--raw`.
- No test references the removed flag.
- The agent-help topics no longer present `--raw` for console, network, or cookies: observe.md drops the three example lines and output.md lists `--raw` only for html and css.
