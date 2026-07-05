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
- Changes to the `network save` subcommand beyond keeping its flags consistent. Save continues to write the full JSON envelope with untruncated bodies by default; the detail dial does not apply to it.

## Dependencies

Requires 03-buffer-sequence-index.md. That project gives every `NetworkEntry` a stable, monotonic `seq` assigned when it is buffered, reset on `clear`, and surviving ring eviction and compaction. This project assumes `seq` is present on each entry returned from the daemon and uses it as the entry's address within the active session. `seq` is assigned from a single counter shared across all page sessions, while the `network` command returns only the active session's entries, so the seq values one session holds are a subsequence of the global order and need not be contiguous: background-tab traffic interleaves other seqs, and `clear` or a tab close removes interior entries. Drill-down therefore resolves an entry by exact membership in the active session's full unfiltered set, not by a range test, and derives any error bounds from that same set before filter or head/tail/range narrowing.

Assumes 02-remove-redundant-raw-flag.md has already removed `--raw` from the network command.

## Current State

The network command lives in internal/cli/network.go and its text formatter in internal/cli/format/text.go.

- internal/cli/network.go defines `networkCmd` (default behavior) and the `save` subcommand. `runNetworkDefault` rejects any positional argument with an unknown-command error, then fetches entries via `getNetworkFromDaemon`, and renders either JSON (`outputNetworkJSON`) or text (`format.Network`). `getNetworkFromDaemon` reads filter flags (with parent-flag fallback), sends the `network` IPC request, unmarshals `ipc.NetworkData`, applies filters, `--find`, and head/tail/range limiting.
- Body handling: `resolveMaxBodySize` reads `--max-body-size`, distinguishing an unset flag from an explicit value via `Changed`, falling back to `ipc.DefaultMaxBodySize` (102400). `applyBodyTruncation` bounds request and response bodies to that size and sets per-entry truncation flags. Both the text and JSON paths call it before rendering. `truncateBody` cuts on a UTF-8 rune boundary.
- internal/cli/format/text.go `Network` renders each entry: a main line (method, URL, status, duration, type, size, cache token), then subordinate lines from `printNetworkRemote`, `printNetworkTiming`, `printNetworkInitiator`, the request side (`printNetworkRequestSide`: optional headers then request body), and the response side (optional response headers then response body or a saved-binary path). Bodies are printed already bounded; a trailing marker flags truncation. Subordinate lines are indented two spaces today.
- Filter and output flags on `networkCmd` (persistent, inherited by `save`): `--find/-f`, `--type`, `--method`, `--status`, `--url`, `--mime`, `--min-duration`, `--min-size`, `--failed`, `--headers`, `--max-body-size`, `--head`, `--tail`, `--range`. `--head/--tail/--range` are mutually exclusive. The `--raw` flag is removed by project 02 before this work.
- internal/ipc/protocol.go defines `NetworkEntry`, `NetworkData`, and `DefaultMaxBodySize`. JSON is the primary interface for agents; text is a TTY convenience.
- The daemon's `network` handler (internal/daemon/handlers_observation.go) returns only the active session's entries (`SessionID == activeID`), and `seq` is assigned from one counter shared across every session (internal/daemon/buffer.go). The active session's held seqs are consequently sparse whenever another tab has interleaved traffic, and `purgeSessionEntries` drops a whole session's entries on tab close. Drill-down and its error bounds are computed over the active session's returned set, so `network <n>` cannot address an entry outside the active session even though `seq` is globally unique.
- Output helpers in internal/cli/root.go (`outputSuccess`, `outputError`, `outputNotice`, `outputHint`, `outputJSON`) must be used rather than writing to stdout/stderr directly. Global flags `--json`, `--debug`, `--no-color` bind to package vars and are reset per REPL invocation.

The default text output is dominated by response bodies. On a content-heavy page it runs into the thousands of lines, almost all of it script, stylesheet, and document source that an agent rarely needs up front.

## Requirements

1. Detail dial. Add `--detail` with values `summary`, `standard`, `full`, defaulting to `standard`. It controls text rendering only and is ignored in JSON mode and by `--schema`.
   - summary: one line per entry (the main line only). No transport detail block, no headers, no bodies.
   - standard: the main line plus the transport detail block (remote, timing, initiator). No bodies. This is the default.
   - full: standard plus the request and response bodies, bounded by `--max-body-size`.
2. Headers orthogonality. `--headers` adds request and response header blocks at the standard and full levels. At summary it is silently ignored: no hint, no error, no promotion to a higher level.
3. Indexed display. Every entry line begins with its `seq`, zero-padded to a minimum width of two digits and growing naturally beyond (01, 09, 10, 99, 100, and up), with no surrounding brackets, followed by the existing main line. Subordinate detail lines indent seven spaces so they read as children of their entry.
4. Drill-down. `network <n>` returns the single entry whose `seq` is the bare integer `n`. Drilling in is an identity lookup, not a search: it addresses the active session's full unfiltered set (the same scope the list uses) and ignores the filter flags (`--find`, `--type`, `--method`, `--status`, `--url`, `--mime`, `--min-duration`, `--min-size`, `--failed`) and the head/tail/range limiting, so a live entry in the active session is never hidden by a narrowing flag. In text it renders that entry with its bodies regardless of `--detail` (drilling in is the request to see the payload), and those bodies are unbounded by default: the payload view is complete, not capped at the 102400 text default. An explicit `--max-body-size` still applies, so a caller who wants a cap can ask for one. `--headers` still adds headers. In JSON it returns that one entry in the standard envelope shape (an `entries` array of length one with `count` 1). A miss — `n` is not among the active session's held seqs — returns an error. The lookup is an exact-membership test, not a range check: because the held seqs may be sparse, `n` falling between the lowest and highest held seq does not mean it is present, and a held seq must never be reported missing. The error names the lowest and highest seq the active session currently holds as orientation and directs the reader to re-list, for example: entry 42 not in buffer (holds seq 318-425; run network to list). Those bounds are orientation only; they do not promise every value between them is present, and the reader recovers by re-running network, not by guessing another number. When the active session holds no entries at all (a fresh daemon, after `clear`, or a session with no traffic), there is no bound to name, so the error reports the empty buffer instead, for example: entry 42 not in buffer (buffer empty). The bare integer is a positional argument; a non-integer positional argument keeps the existing unknown-command error, and `save` remains a subcommand.
5. JSON full fidelity. JSON output always returns complete entries: every field and every body, untruncated by default. The detail dial never reduces JSON. An agent shapes its payload by querying (filters, head/tail/range, drill-down), not by a verbosity flag.
6. Body size control. `--max-body-size` accepts three regimes: a positive N caps bodies at N bytes; 0 suppresses all body content; -1 means unlimited. Defaults depend on output mode when the flag is not set: the `--detail full` text list defaults to 102400, JSON defaults to unlimited, text drill-down (`network <n>`) defaults to unlimited, and save defaults to unlimited (a saved file is a full-fidelity archive). An explicitly set value is honored in every mode. The bound applies wherever bodies render: the full text list level, text drill-down, JSON, and save.
7. Schema preview. Add `--schema`. It requires an index; `network --schema` without one is an error directing the user to supply an entry. `network <n> --schema` parses entry n's response body as JSON and returns a key skeleton:
   - Representation: a nested structure mirroring the body, with each leaf value replaced by its JSON type name as a string, for example {"count":"number","vehicles":[{"id":"number","name":"string","options":["string"]}]}.
   - Arrays collapse to a single representative element whose object is the union of keys seen across all elements, so a heterogeneous array does not hide fields.
   - It reads the full stored response body before any `--max-body-size` truncation, since a truncated body is not parseable JSON.
   - A parsed body is wrapped in the standard JSON envelope as {"ok":true,"schema":{...}} on stdout with exit 0.
   - A non-JSON response body (HTML, script, binary, empty) is not an error: it returns the same envelope on stdout with exit 0, the schema null and a notice naming why, as {"ok":true,"schema":null,"notice":"response body is not JSON (text/html)"}. Both outcomes share one envelope, one stream, and one exit code so a single parser branches on the `schema` field (null means see `notice`) rather than on stream, exit code, or output mode.
   - `--schema` is self-contained: it does not require `--json`, its output is JSON-shaped regardless, and it short-circuits the detail dial. Do not route the non-JSON case through `outputNotice`, whose stderr, non-zero-exit, mode-dependent shape would break this contract.

## Constraints

- Pure Go, no cgo, no new dependencies. gofmt and go vet clean.
- Use the output helpers in internal/cli/root.go; do not write to stdout/stderr directly. Preserve the `printedError` no-double-print contract.
- JSON is the primary agent interface; keep the envelope shape consistent across the list, drill-down, and schema responses so a single parser handles all three.
- Command abbreviation must keep working: `Execute` expands unique command prefixes, so the bare-integer drill-down argument must not be mistaken for a subcommand, and `save` must still resolve.
- Reset any new per-command flag state between REPL invocations, consistent with how existing per-command flags are reset, to avoid flag bleed in the REPL.
- Follow the documentation conventions in AGENTS.md. Create docs/network.md (it does not exist yet; only serve.md and start.md have per-command docs) covering the new flags, the detail levels, the indexed output, drill-down, and schema. No network agent-help topic exists, so none needs updating.

## Implementation Plan

1. Add the `--detail` flag with validation of its three values and a default of standard. Thread the resolved level into the text formatter.
2. Restructure `format.Network` to honor the level: summary prints only the main line; standard adds the transport detail block; full adds bodies. Move body rendering behind the full level. Change the subordinate indent to seven spaces.
3. Prefix each entry's main line with its zero-padded `seq` and no brackets. Confirm the value comes from the entry, not from its position in the slice.
4. Make `--max-body-size` mode-aware: introduce the -1 unlimited sentinel, and when the flag is unset choose the default by output mode — the `--detail full` text list 102400; JSON and save unlimited (the text drill-down default, also unlimited, is wired with the drill-down path in step 6). Keep 0 as suppress. Apply the bound in the text full list level, text drill-down, JSON, and save; ensure unlimited skips truncation entirely.
5. Make JSON output return complete, untruncated entries by default while still honoring an explicit `--max-body-size`.
6. Accept an optional bare-integer positional argument on the default command for drill-down. On an integer, fetch the active session's full unfiltered set (bypassing the CLI filter flags and head/tail/range limiting), locate the entry whose `seq` equals `n` by exact membership, and render it with bodies in text or as a single-entry envelope in JSON; on a miss, error with the lowest and highest held seq as orientation (or the empty-buffer message), directing the reader to re-list. Do not treat `lowest <= n <= highest` as present — the held seqs may be sparse, so test membership rather than range. Text drill-down renders its bodies unbounded when `--max-body-size` is unset, so the payload view is complete; an explicit `--max-body-size` still caps it. Preserve the unknown-command error for non-integer arguments and the `save` subcommand.
7. Add `--schema`. Require an index, parse the full response body as JSON, build the union-collapsed key skeleton with type-name leaves, wrap it in the envelope, and emit a notice for non-JSON bodies.
8. Create docs/network.md documenting the network flags and output. No network-specific agent-help topic exists today, so there is none to update.
9. Extend internal/cli/network_test.go and format tests to cover the detail levels, indexed display, drill-down hit and miss, the max-body-size regimes, schema on JSON and non-JSON bodies, and schema without an index. The miss coverage must include a sparse held set (a seq between the lowest and highest that is absent) to prove the lookup tests membership rather than range, and the empty-buffer message.

## Implementation Guidance

The three detail levels map onto rendering that already exists. Standard is close to today's output with bodies suppressed (the transport detail block is already produced by `printNetworkRemote`, `printNetworkTiming`, `printNetworkInitiator`). Summary is the main line alone. Full adds the body rendering that is currently always on. Implement the level as a single gate over the detail block and the body block rather than three separate code paths.

The display index is decoration around an existing value. The input is always the bare integer (`network 42`); the listing shows the same bare zero-padded number so input and output match. Do not print brackets; they are shell globs and would mislead a user into typing something that does not parse.

Drill-down is the single-entry payload view. It should show bodies even at the default level, because asking for one entry by number is the explicit request to see its content. Keep its JSON shape identical to the list (an `entries` array) so the agent does not branch on a different schema for one-versus-many.

Schema is a preview that saves the agent from pulling a large body to learn its shape. Type-name leaves plus union-collapsed arrays give enough structure to write a precise jq expression against the full body fetched separately. Keep the skeleton minimal: structure and type names only, no example values, no counts.

For mode-dependent `--max-body-size` defaults, the existing `resolveMaxBodySize` already distinguishes an explicitly set flag from an unset one via `Changed`. Extend that to select the mode default when unset (the `--detail full` text list 102400; JSON, save, and text drill-down unlimited), rather than a single constant. The list and drill-down text paths share text rendering but not the same unset default, so the caller has to tell the resolver which it is — the resolver cannot read it off the flags alone.

## Acceptance Criteria

- Running `network` on a multi-entry page shows one indexed line per entry plus the transport detail block, and no request or response bodies.
- `network --detail summary` shows exactly one line per entry: no detail block, no bodies, and `--headers` has no effect.
- `network --detail full` shows request and response bodies, bounded by the 102400-byte text default unless `--max-body-size` overrides it.
- Each displayed index is a bare zero-padded number with no brackets, and it matches the integer accepted by drill-down.
- `network <n>` returns only the entry with `seq` n, rendered with its bodies unbounded by default (the text 102400 cap does not apply unless `--max-body-size` is set); a drill-down to a seq the active session does not hold errors with a message naming the lowest and highest held seq (or reporting an empty buffer), and a seq the session does hold is never reported missing even when the held seqs are sparse.
- `network --json` returns complete, untruncated bodies, and `--detail` has no effect on JSON.
- `--max-body-size 0` suppresses bodies, `-1` returns them unbounded, and a positive N caps them at N bytes, in both text full level and JSON.
- `network <n> --schema` returns {"ok":true,"schema":{...}} with a union-collapsed, type-name key skeleton of the response body; a non-JSON body returns {"ok":true,"schema":null,"notice":...} on stdout with exit 0; `network --schema` without an index errors.
- docs/network.md documents the detail levels, indexed output, drill-down, schema, and the `--max-body-size` regimes.
