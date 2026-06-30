# Network Output Redesign

## Goal

Reduce the default payload of the `network` command and make full request/response bodies a deliberate drill-down rather than the default. An AI agent should be able to scan the traffic as a compact indexed list, then fetch the payload for one specific entry, and preview the shape of a JSON response body before pulling it.

## Scope

In scope:

- A text-only detail dial (`--detail summary|standard|full`) controlling how much of each entry renders.
- A new default text view: an indexed list with the transport detail block, no bodies.
- Indexed display of every entry using its owned sequence number.
- Per-entry drill-down by sequence number (`network <n>`), including an eviction-aware error.
- JSON output as full fidelity: always complete, untruncated by default.
- A `--max-body-size` unlimited sentinel and output-mode-dependent defaults.
- A `--schema` flag that returns a token-efficient key skeleton of an entry's JSON response body.

Out of scope:

- The sequence-number facility itself. This project depends on 03-buffer-sequence-index.md, which adds a stable `seq` field to every network entry. Do not implement sequence assignment here; consume it.
- Removing the `--raw` flag. That is handled by 02-remove-redundant-raw-flag.md, which runs first. This project assumes `--raw` is already gone from the network command and adds no handling for it.
- Console command changes.
- Changes to the `network save` subcommand beyond keeping its flags consistent. Save continues to write the full JSON envelope; the detail dial does not apply to it.

## Dependencies

Requires 03-buffer-sequence-index.md. That project gives every `NetworkEntry` a stable, monotonic `seq` assigned when it is buffered, reset on `clear`, and surviving ring eviction and compaction. This project assumes `seq` is present on each entry returned from the daemon and uses it as the entry's address. The lowest and highest `seq` among the entries currently returned define the live range used in drill-down error messages.

Assumes 02-remove-redundant-raw-flag.md has already removed `--raw` from the network command.

## Current State

The network command lives in internal/cli/network.go and its text formatter in internal/cli/format/text.go.

- internal/cli/network.go defines `networkCmd` (default behavior) and the `save` subcommand. `runNetworkDefault` rejects any positional argument with an unknown-command error, then fetches entries via `getNetworkFromDaemon`, and renders either JSON (`outputNetworkJSON`) or text (`format.Network`). `getNetworkFromDaemon` reads filter flags (with parent-flag fallback), sends the `network` IPC request, unmarshals `ipc.NetworkData`, applies filters, `--find`, and head/tail/range limiting.
- Body handling: `resolveMaxBodySize` reads `--max-body-size`, distinguishing an unset flag from an explicit value via `Changed`, falling back to `ipc.DefaultMaxBodySize` (102400). `applyBodyTruncation` bounds request and response bodies to that size and sets per-entry truncation flags. Both the text and JSON paths call it before rendering. `truncateBody` cuts on a UTF-8 rune boundary.
- internal/cli/format/text.go `Network` renders each entry: a main line (method, URL, status, duration, type, size, cache token), then subordinate lines from `printNetworkRemote`, `printNetworkTiming`, `printNetworkInitiator`, the request side (`printNetworkRequestSide`: optional headers then request body), and the response side (optional response headers then response body or a saved-binary path). Bodies are printed already bounded; a trailing marker flags truncation. Subordinate lines are indented two spaces today.
- Filter and output flags on `networkCmd` (persistent, inherited by `save`): `--find/-f`, `--type`, `--method`, `--status`, `--url`, `--mime`, `--min-duration`, `--min-size`, `--failed`, `--headers`, `--max-body-size`, `--head`, `--tail`, `--range`. `--head/--tail/--range` are mutually exclusive. The `--raw` flag is removed by project 02 before this work.
- internal/ipc/protocol.go defines `NetworkEntry`, `NetworkData`, and `DefaultMaxBodySize`. JSON is the primary interface for agents; text is a TTY convenience.
- Output helpers in internal/cli/root.go (`outputSuccess`, `outputError`, `outputNotice`, `outputHint`, `outputJSON`) must be used rather than writing to stdout/stderr directly. Global flags `--json`, `--debug`, `--no-color` bind to package vars and are reset per REPL invocation.

The default text output is dominated by response bodies. On a content-heavy page it runs into the thousands of lines, almost all of it script, stylesheet, and document source that an agent rarely needs up front.

## Requirements

1. Detail dial. Add `--detail` with values `summary`, `standard`, `full`, defaulting to `standard`. It controls text rendering only and is ignored in JSON mode and by `--schema`.
   - summary: one line per entry (the main line only). No transport detail block, no headers, no bodies.
   - standard: the main line plus the transport detail block (remote, timing, initiator). No bodies. This is the default.
   - full: standard plus the request and response bodies, bounded by `--max-body-size`.
2. Headers orthogonality. `--headers` adds request and response header blocks at the standard and full levels. At summary it is silently ignored: no hint, no error, no promotion to a higher level.
3. Indexed display. Every entry line begins with its `seq`, zero-padded to a minimum width of two digits and growing naturally beyond (01, 09, 10, 99, 100, and up), with no surrounding brackets, followed by the existing main line. Subordinate detail lines indent seven spaces so they read as children of their entry.
4. Drill-down. `network <n>` returns the single entry whose `seq` is the bare integer `n`. In text it renders that entry with its bodies regardless of `--detail` (drilling in is the request to see the payload); `--headers` still adds headers. In JSON it returns that one entry in the standard envelope shape (an `entries` array of length one with `count` 1). A non-existent or evicted `n` returns an error that states the live `seq` range currently held, for example: entry 42 not in buffer (holds 318-425). The bare integer is a positional argument; a non-integer positional argument keeps the existing unknown-command error, and `save` remains a subcommand.
5. JSON full fidelity. JSON output always returns complete entries: every field and every body, untruncated by default. The detail dial never reduces JSON. An agent shapes its payload by querying (filters, head/tail/range, drill-down), not by a verbosity flag.
6. Body size control. `--max-body-size` accepts three regimes: a positive N caps bodies at N bytes; 0 suppresses all body content; -1 means unlimited. Defaults depend on output mode when the flag is not set: text defaults to 102400, JSON defaults to unlimited. An explicitly set value is honored in both modes. The bound applies wherever bodies render: the full text level, JSON, and save.
7. Schema preview. Add `--schema`. It requires an index; `network --schema` without one is an error directing the user to supply an entry. `network <n> --schema` parses entry n's response body as JSON and returns a key skeleton:
   - Representation: a nested structure mirroring the body, with each leaf value replaced by its JSON type name as a string, for example {"count":"number","vehicles":[{"id":"number","name":"string","options":["string"]}]}.
   - Arrays collapse to a single representative element whose object is the union of keys seen across all elements, so a heterogeneous array does not hide fields.
   - It reads the full stored response body before any `--max-body-size` truncation, since a truncated body is not parseable JSON.
   - A non-JSON response body (HTML, script, binary, empty) returns a notice explaining the body is not JSON, not an error.
   - Output is wrapped in the standard JSON envelope as {"ok":true,"schema":{...}}. `--schema` is self-contained: it does not require `--json`, its output is JSON-shaped regardless, and it short-circuits the detail dial.

## Constraints

- Pure Go, no cgo, no new dependencies. gofmt and go vet clean.
- Use the output helpers in internal/cli/root.go; do not write to stdout/stderr directly. Preserve the `printedError` no-double-print contract.
- JSON is the primary agent interface; keep the envelope shape consistent across the list, drill-down, and schema responses so a single parser handles all three.
- Command abbreviation must keep working: `Execute` expands unique command prefixes, so the bare-integer drill-down argument must not be mistaken for a subcommand, and `save` must still resolve.
- Reset any new per-command flag state between REPL invocations, consistent with how existing per-command flags are reset, to avoid flag bleed in the REPL.
- Follow the documentation conventions in AGENTS.md. Update docs/network.md to reflect the new flags, the detail levels, the indexed output, drill-down, and schema.

## Implementation Plan

1. Add the `--detail` flag with validation of its three values and a default of standard. Thread the resolved level into the text formatter.
2. Restructure `format.Network` to honor the level: summary prints only the main line; standard adds the transport detail block; full adds bodies. Move body rendering behind the full level. Change the subordinate indent to seven spaces.
3. Prefix each entry's main line with its zero-padded `seq` and no brackets. Confirm the value comes from the entry, not from its position in the slice.
4. Make `--max-body-size` mode-aware: introduce the -1 unlimited sentinel, and when the flag is unset choose the text default (102400) or the JSON default (unlimited) based on output mode. Keep 0 as suppress. Apply the bound in the text full level, JSON, and save; ensure unlimited skips truncation entirely.
5. Make JSON output return complete, untruncated entries by default while still honoring an explicit `--max-body-size`.
6. Accept an optional bare-integer positional argument on the default command for drill-down. On an integer, fetch the entry with that `seq`; render it with bodies in text or as a single-entry envelope in JSON; on a miss, error with the live `seq` range derived from the available entries. Preserve the unknown-command error for non-integer arguments and the `save` subcommand.
7. Add `--schema`. Require an index, parse the full response body as JSON, build the union-collapsed key skeleton with type-name leaves, wrap it in the envelope, and emit a notice for non-JSON bodies.
8. Update docs/network.md and any agent-help topic that documents network flags or output.
9. Extend internal/cli/network_test.go and format tests to cover the detail levels, indexed display, drill-down hit and evicted-miss, the max-body-size regimes, schema on JSON and non-JSON bodies, and schema without an index.

## Implementation Guidance

The three detail levels map onto rendering that already exists. Standard is close to today's output with bodies suppressed (the transport detail block is already produced by `printNetworkRemote`, `printNetworkTiming`, `printNetworkInitiator`). Summary is the main line alone. Full adds the body rendering that is currently always on. Implement the level as a single gate over the detail block and the body block rather than three separate code paths.

The display index is decoration around an existing value. The input is always the bare integer (`network 42`); the listing shows the same bare zero-padded number so input and output match. Do not print brackets; they are shell globs and would mislead a user into typing something that does not parse.

Drill-down is the single-entry payload view. It should show bodies even at the default level, because asking for one entry by number is the explicit request to see its content. Keep its JSON shape identical to the list (an `entries` array) so the agent does not branch on a different schema for one-versus-many.

Schema is a preview that saves the agent from pulling a large body to learn its shape. Type-name leaves plus union-collapsed arrays give enough structure to write a precise jq expression against the full body fetched separately. Keep the skeleton minimal: structure and type names only, no example values, no counts.

For mode-dependent `--max-body-size` defaults, the existing `resolveMaxBodySize` already distinguishes an explicitly set flag from an unset one via `Changed`. Extend that to select the text or JSON default when unset, rather than a single constant.

## Acceptance Criteria

- Running `network` on a multi-entry page shows one indexed line per entry plus the transport detail block, and no request or response bodies.
- `network --detail summary` shows exactly one line per entry: no detail block, no bodies, and `--headers` has no effect.
- `network --detail full` shows request and response bodies, bounded by the 102400-byte text default unless `--max-body-size` overrides it.
- Each displayed index is a bare zero-padded number with no brackets, and it matches the integer accepted by drill-down.
- `network <n>` returns only the entry with `seq` n, rendered with its bodies; a drill-down to an evicted or unknown n errors with a message naming the live `seq` range.
- `network --json` returns complete, untruncated bodies, and `--detail` has no effect on JSON.
- `--max-body-size 0` suppresses bodies, `-1` returns them unbounded, and a positive N caps them at N bytes, in both text full level and JSON.
- `network <n> --schema` returns {"ok":true,"schema":{...}} with a union-collapsed, type-name key skeleton of the response body; a non-JSON body returns a notice; `network --schema` without an index errors.
- docs/network.md documents the detail levels, indexed output, drill-down, schema, and the `--max-body-size` regimes.
