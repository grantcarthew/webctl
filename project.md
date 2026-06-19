# Project: webctl markdown command

## Goal

Add a `markdown` command (alias `md`) to webctl that returns the current browser page rendered as Markdown, so an agent can read page content without HTML tags. Conversion happens CLI-side by reusing the existing `html` retrieval path; the daemon is not modified.

## Scope

In scope:

- A new `markdown` CLI command with alias `md`, available both as `webctl markdown` and inside the REPL.
- A `markdown save [path]` subcommand mirroring `html save` / `css save`.
- A new `internal/markdownformat` package that converts HTML to Markdown.
- A shared save helper that removes the duplication across all save commands (`html save`, `css save`, `console save`, `network save`) and is used by `markdown save` from the start.
- Selector targeting and text search that match the `html` command exactly.

Out of scope:

- Any readability or article-extraction step. Page structure varies too much for a reliable heuristic; the agent scopes content with `--select` instead.
- Daemon-side or IPC-protocol changes. The command reuses the existing `html` IPC request.
- A new daemon handler or new IPC request/response types.
- Per-command documentation files under `docs/` (none exist for `html` or `css`; do not add one for `markdown`).

## Current State

webctl is a Go browser-automation CLI with a daemon plus stateless-command model. The daemon holds a live CDP connection; short-lived CLI commands talk to it over a Unix socket via IPC, then exit. See AGENTS.md for architecture, conventions, and the output-helper contract.

Relevant existing pieces:

- `internal/cli/html.go` defines the `html` command and its `save` subcommand. `getHTMLFromDaemon(cmd)` sends `ipc.Request{Cmd: "html", Params: ipc.HTMLParams{Selector: ...}}` through the executor, receives `ipc.HTMLData`, optionally prettifies via `internal/htmlformat`, and applies a `--find` filter. The daemon side (`handleHTML` in `internal/daemon/handlers_observation.go`) retrieves the live `documentElement.outerHTML` over CDP and supports `--select`. This path is reused as-is; markdown needs the raw (unformatted) outerHTML, not the htmlformat-prettified output.
- `--find` on `html` is a line-based text filter (`filterHTMLByText` in `internal/cli/html.go`), applied after formatting, with `-B`/`-A`/`-C` context flags. It operates on plain lines, so it applies unchanged to Markdown output.
- `runHTMLSave` (`internal/cli/html.go`) and `runCSSSave` (`internal/cli/css.go`) are near-identical. Each: checks the daemon is running, calls its content getter, maps `ErrNoMatches` / `ErrNoElements` (`css` adds `ErrNoRules`) to `outputNotice`, then resolves an output path in three modes — no path (auto-generate into a temp dir), a path ending in a separator (treat as directory, auto-generate filename, `MkdirAll`), or a plain path (exact file). Filenames follow `YY-MM-DD-HHMMSS-{identifier}.{ext}`, where `identifier` is the sanitized selector if given, otherwise the normalized page title fetched via a `status` IPC call, otherwise a fixed fallback word. Temp dirs are `/tmp/webctl-html` and `/tmp/webctl-css`. On success, text mode prints the path via `format.FilePath`; JSON mode returns `{"ok": true, "path": ...}`. Beyond the timer label, content getter, temp dir, and file extension, the two functions also diverge in two filename behaviors: the no-title fallback identifier (`html` uses `page`, `css` uses `untitled`) and the `status`-lookup error policy (`html` ignores a failed lookup and uses its fallback; `css` returns the error and aborts the save). The shared helper removes both divergences by adopting the unified filename scheme defined in Requirements. `console save` (`internal/cli/console.go`) and `network save` (`internal/cli/network.go`) duplicate the same three-mode path resolution and filename generation, differing only in using a fixed identifier (`console`, `network`) rather than a selector/title lookup; they fold into the same helper.
- Commands register in `init()` via `rootCmd.AddCommand` and are assigned a help group in `commandGroups` (`internal/cli/root.go`); `html` and `css` share the observation group.
- No command in the repository currently uses cobra `Aliases`; short forms today come only from `Execute`'s unique-prefix expansion. `md` is not a prefix of `markdown`, so it must be a real cobra alias — the first in the repo.
- Output helpers (`outputSuccess`, `outputError`, `outputNotice`, `outputJSON`) live in `internal/cli/root.go`. JSON output is the primary agent interface; text output is a TTY convenience.
- The agent-help topic `internal/cli/agent-help/observe.md` lists the observation commands (html, css, console, network, cookies). The human-facing command list lives in the root `README.md`: the Commands table Observation row (README.md:49) and the Observation feature bullet (README.md:33). `docs/README.md` is a short pointer with no command list and is not edited by this project.

snag (`~/Projects/snag`) is an existing tool by the same author that performs the same HTML-to-Markdown conversion. Matching its library and plugin set keeps output identical across the two tools.

## References

- `~/Projects/snag/formats.go` — reference converter setup. Constructs `converter.NewConverter` with `base`, `commonmark`, `table`, and `strikethrough` plugins, then calls `markdownConverter.ConvertString(html)`.
- `~/Projects/snag/go.mod` — pins `github.com/JohannesKaufmann/html-to-markdown/v2 v2.5.0`.
- Plugin import paths: `github.com/JohannesKaufmann/html-to-markdown/v2/converter`, `.../plugin/base`, `.../plugin/commonmark`, `.../plugin/table`, `.../plugin/strikethrough`.
- `internal/cli/html.go` — the command, save subcommand, content getter, and `--find` filter to mirror.
- `internal/cli/css.go`, `internal/cli/console.go`, `internal/cli/network.go` — the other save implementations to fold into the shared helper.

## Requirements

1. `internal/markdownformat` package exposing a single conversion entry point that takes an HTML string and returns Markdown plus an error. It wraps html-to-markdown/v2 with the base, commonmark, table, and strikethrough plugins — the same set snag uses — so output matches snag.
2. A `markdown` command (alias `md`) that retrieves the live page HTML via the existing `html` IPC path, converts it to Markdown, and writes it to stdout. It must work both as `webctl markdown` / `webctl md` and inside the REPL.
3. The command accepts `-s` / `--select` to scope conversion to a CSS selector, with identical behavior to `html --select` (it flows through the same IPC request; multi-match elements are concatenated before conversion).
4. The command accepts `-f` / `--find` with `-B` / `-A` / `-C` context flags, reusing the same line-based filter as `html`, applied to the Markdown output.
5. JSON mode (`--json`) returns the Markdown under a `markdown` field via the standard success envelope. Text mode prints raw Markdown.
6. A `markdown save [path]` subcommand with the same three path modes, auto-filename pattern, and success output as `html save`. Temp dir is `/tmp/webctl-markdown`; extension is `.md`.
7. A shared save helper that every save command uses — `html save`, `css save`, `console save`, `network save`, and `markdown save` — with five per-command variation points: timer label, content getter, temp dir, file extension, and identifier source. After this change, no per-command save function contains duplicated path-resolution, filename-generation, or file-writing logic. The helper generates filenames as `YY-MM-DD-HHMMSS-mmm[-identifier].{ext}`: a millisecond-precision timestamp (so two saves within the same second no longer overwrite each other) followed by an optional identifier. The identifier source is one of two kinds: the content commands (`html`, `css`, `markdown`) resolve it as the sanitized selector if given, otherwise the normalized page title, otherwise omitted — including when the `status` lookup fails, in which case the save still succeeds; the buffer commands (`console`, `network`) supply a fixed word (`console`, `network`). This replaces the previous per-command `page`/`untitled` fallback words and the `css` abort-on-status-error behavior.
8. The command is registered and assigned to the same help group as `html` and `css`.
9. Unit tests for `internal/markdownformat` covering table, nested-list, and strikethrough fidelity.
10. A CLI bash test for `markdown` mirroring the existing `html` CLI test, exercising default output, `--select`, `--find`, and `save`.
11. The agent-help `observe.md` topic and the root `README.md` command list include `markdown` — both the Commands table Observation row and the Observation feature bullet.

## Constraints

- Pure Go, no cgo. html-to-markdown/v2 is pure Go and is the only new dependency permitted by this project. Pin the same major/minor as snag (v2.5.0 or compatible).
- Use the output helpers in `internal/cli/root.go`; do not write to stdout/stderr directly.
- Feed the converter the raw `outerHTML`, not htmlformat-prettified HTML.
- Do not add a daemon handler, IPC request type, or IPC response type. The command must reuse `Cmd: "html"`.
- Follow AGENTS.md: gofmt clean, `go vet` clean, idiomatic error returns, no panics in library code.
- The save-helper refactor changes the generated filename shape for every save command (ms-precision timestamp; `html`/`css` also drop the `page`/`untitled` fallback word); this is an intentional change. The existing html, css, console, and network save tests must still pass unchanged — they assert the file extension and that a file is written, not the exact filename. Preserve all other per-command save behavior: path modes, error/sentinel handling, and success output.

## Implementation Plan

1. Add the html-to-markdown/v2 dependency and create `internal/markdownformat` with the conversion function. Construct the converter once (package-level) with the four plugins, following snag's setup. Add unit tests for table, nested-list, and strikethrough output.
2. Extract the shared save logic into a new helper (e.g. `internal/cli/save.go`). Model the variation points as data: a timer label, a content getter `func(*cobra.Command) (string, error)`, a temp directory, a file extension, and an identifier source (a selector/title resolver for the content commands, a fixed string for the buffer commands). The helper owns daemon-running check, sentinel-to-notice mapping (`ErrNoMatches`, `ErrNoElements`, `ErrNoRules`), three-mode path resolution, filename generation, file writing, and JSON/text success output. It generates the unified filename `YY-MM-DD-HHMMSS-mmm[-identifier].{ext}` (ms-precision timestamp, identifier omitted only when a content command has no selector and no title). Update the `Save:` example lines in the `html`, `css`, `console`, and `network` command help text to the new shape.
3. Rewrite the `html`, `css`, `console`, and `network` save functions to delegate to the shared helper. Run the existing html, css, console, and network tests to confirm no behavior change beyond the intended filename shape.
4. Create `internal/cli/markdown.go`: define the `markdown` command with `Aliases: []string{"md"}`, the `-s/--select` and `-f/--find` (+ `-B/-A/-C`) flags, and a `getMarkdownFromDaemon` that calls the existing html IPC path for raw outerHTML, converts via `internal/markdownformat`, then applies the shared `--find` line filter. Default run writes Markdown to stdout (text) or the `markdown` field (JSON).
5. Add the `markdown save` subcommand wired to the shared save helper with the markdown content getter, `/tmp/webctl-markdown`, and `.md`.
6. Register the command in `init()` and add it to `commandGroups` in the same group as `html` and `css`.
7. Verify the `md` alias dispatches correctly through the REPL command path, not only the top-level CLI.
8. Update `internal/cli/agent-help/observe.md` and the root `README.md` command list (Commands table Observation row and Observation feature bullet) to include `markdown`.
9. Add the CLI bash test mirroring the html test. Run `./test-runner quick`, then `./test-runner ci`.

## Implementation Guidance

- Prefer reusing `getHTMLFromDaemon`'s request-building over duplicating IPC marshalling; the markdown getter differs only in skipping htmlformat and converting instead. If `getHTMLFromDaemon` cannot return raw HTML without prettifying, factor out the raw-fetch step rather than copying it.
- Keep the shared save helper concrete and small. It exists to remove the existing duplication across the current save commands (`html`, `css`, `console`, `network`) plus `markdown`, not to anticipate future ones. Do not generalize beyond what those five need.
- The `md` alias is the repo's first cobra alias. Confirm both `Execute`'s prefix expansion and the REPL still resolve commands correctly after adding it.

## Acceptance Criteria

- `webctl md` and `webctl markdown` both emit Markdown for the current page; the same works inside the REPL.
- `webctl md --select <sel>` converts only the selected subtree; output matches converting the equivalent `webctl html --select <sel>` result.
- `webctl md --find <text>` filters the Markdown by line with `-B/-A/-C` behaving as on `html`.
- `webctl md --json` returns the Markdown under a `markdown` field in the success envelope.
- `webctl md save`, `webctl md save <file>`, and `webctl md save <dir>/` produce `.md` files using the established naming and the `/tmp/webctl-markdown` temp dir, with the same success output as `html save`.
- All save commands (`html save`, `css save`, `console save`, `network save`, `markdown save`) retain their path modes, error handling, and success output, share one save implementation, and emit the unified filename `YY-MM-DD-HHMMSS-mmm[-identifier].{ext}`; no save path-resolution or filename logic is duplicated across them.
- Converting a page with a table, a nested list, and strikethrough yields Markdown matching snag's output for the same HTML.
- `markdown` appears in `webctl --help` grouped with `html` and `css`, in `observe.md`, and in the root `README.md` command list (Commands table Observation row and Observation feature bullet).
