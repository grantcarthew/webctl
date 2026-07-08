# Console Index and Drill-down

## Goal

Give the console command the same scan-then-drill-down workflow as the network command: a lean indexed list by default, and a per-entry drill-down that reveals the enriched payload (full stack trace, all arguments, exception and Log-domain detail). An agent scans the console stream, then fetches the full detail for one entry by its index.

## Scope

In scope:

- Indexed console list: each entry prefixed with its `seq`, one summary line per entry.
- Drill-down: `console <n>` returns the single entry with that `seq`, rendered in full.
- JSON: `console <n>` returns the one entry in the standard envelope; the default `console --json` stays full fidelity. The console JSON envelope keys its array `entries` with a `count`, matching the network command; the current `logs` key is retired across the list, drill-down, and save outputs.
- Seq-based `--range`: `console --range START-END` selects entries by inclusive `seq` membership, matching the network command; `--head` and `--tail` remain entry counts.

Out of scope:

- The `seq` facility, added by 03-buffer-sequence-index.md.
- The enriched capture (stack traces, function names, exception class, argument previews, Log-domain fields), added by 04-console-capture-enrichment.md. This project displays those fields; it does not capture them.
- Any detail dial. Console is two-level (list versus drill-down); there is no `--detail` flag and no middle tier.
- Network command changes.

## Dependencies

Requires 03-buffer-sequence-index.md (every `ConsoleEntry` carries a stable `seq`, and the live `seq` range is derivable from the buffered entries) and 04-console-capture-enrichment.md (the enriched fields this project surfaces on drill-down). Assumes 02-remove-redundant-raw-flag.md has already removed `--raw` from the console command.

This project mirrors the conventions established by 05-network-output-redesign.md: bare zero-padded `seq` display with no brackets, seven-space subordinate indent, bare-integer drill-down, inclusive seq-membership `--range`, the eviction-aware error wording, and the single-entry JSON envelope keyed `entries`. Match them so the two commands behave identically.

## Current State

The console command lives in internal/cli/console.go and its text formatter in internal/cli/format/text.go.

- console.go default command rejects any positional argument with an unknown-command error (`cobra.MaximumNArgs(1)`), filters with `--find/-f` and `--type`, and windows with `--head/--tail/--range`. The `--raw` flag is removed by project 02.
- internal/cli/format/text.go `Console` renders each entry as `[timestamp] LEVEL text`, then the source `url:line` on a two-space-indented line. It does not render `Args`, function names, stack frames, or any of the enriched fields.
- After projects 03 and 04, `ConsoleEntry` (internal/ipc/protocol.go) carries `seq` plus the enriched fields: the full stack frame list with function names, exception class and subtype, argument descriptions and previews, and Log-domain source, level, and network request id.
- consoleBuf is a `RingBuffer[ipc.ConsoleEntry]` in the daemon; the console IPC command returns its entries.

The default list is already compact, but with enrichment in place the per-entry detail (full stacks, full arguments, Log correlation) is now substantial and does not belong inline in a scan. It is the payload to reveal on drill-down.

## Requirements

1. Indexed list. The default console output shows one summary line per entry, prefixed with the entry's `seq`, zero-padded to a minimum width of two and growing naturally beyond, with no brackets. The summary line carries the `[HH:MM:SS]` wall-clock timestamp, the level, the top stack frame (function name and url:line:column when present), and the entry's `Text` as the message. Retain the timestamp: unlike the network main line, console has no duration or timing block, so the timestamp is its only wall-clock signal and the primary key for correlating a log against a network entry, screenshot, or external log. Source the message from `Text`, not from `Args[0]`: console API calls set `Text` to the rendered first argument, while exceptions and Log-domain entries carry no `Args` and populate only `Text`, so `Text` is the one message source that is present for every entry kind. `Args` stays a drill-down-only payload. Any retained subordinate line indents seven spaces, matching the network command.
2. Lean by default. The default list does not render the full stack trace, the full argument set, or Log-domain detail beyond what fits the summary line. These form the payload shown only on drill-down.
3. Drill-down. `console <n>` returns the single entry whose `seq` is the bare integer `n`, rendered in full: the complete stack trace with function names, all arguments including object previews, exception class and subtype, and Log source, level, and network request id when present. Drill-down is an identity lookup, not a search: it resolves `n` against the active session's full unfiltered entry set and ignores the filter flags (`--find`, `--type`) and the `--head`/`--tail`/`--range` limiting, so a live entry is never hidden by a narrowing flag, matching the network command. A non-existent or evicted `n` errors with the live `seq` range derived from that same unfiltered set, matching the network wording. A non-integer positional argument keeps the existing unknown-command error.
4. JSON. The console JSON envelope keys its array `entries` alongside a `count`, matching the network command and the underlying `ConsoleData` shape (both IPC data types already serialize as `entries`; `logs` is a CLI-only rename applied at the final marshal). Retire the `logs` key across the list, drill-down, and save outputs and emit `entries` instead. `console <n>` in JSON returns that one entry in this envelope (an `entries` array of length one with `count` 1). The default `console --json` is unchanged in fidelity — complete entries, every field, never reduced by the list-versus-drill-down distinction — but its array key becomes `entries`. This is a deliberate breaking change to the current `console --json` and `console save` output shape.
5. Seq-based range. `console --range START-END` selects entries by inclusive `seq` membership, matching the network command: the held seqs may be sparse, so the endpoints need not be present and the flag returns whatever held seqs fall inside `[START, END]`, empty when none do. An empty range is an empty list with exit 0, exactly like the network command; console must not treat it as a notice. Retire the `ErrNoEntriesInRange` sentinel from the console and save paths, since sparse seq membership makes an empty range a routine result rather than an error. Replace the current position-based windowing in `applyConsoleLimiting` so the range addresses the same seq values the list now displays. `--head` and `--tail` stay entry counts over the seq-ordered list.
6. Consistency. The seq format, subordinate indent, drill-down syntax, JSON envelope shape, `--range` semantics, and eviction-error wording match the network command so a user or agent learns one pattern for both.

## Constraints

- Pure Go, no cgo, no new dependencies. gofmt and go vet clean.
- Use the output helpers in internal/cli/root.go; do not write to stdout/stderr directly.
- Command abbreviation must keep working: the bare-integer drill-down argument must not be mistaken for a subcommand.
- Reset any new per-command flag state between REPL invocations, consistent with existing per-command flags.
- Keep the JSON envelope shape identical to the network command's list and drill-down responses, including the `entries` array key. Migrating console from `logs` to `entries` breaks the current `console --json` and `console save` key; update the existing `logs`-key assertions in internal/cli/cli_test.go to `entries` as part of this change.
- Follow the documentation conventions in AGENTS.md. Create docs/console.md (it does not exist yet) to document the indexed list, drill-down, and seq-based `--range`.

## Implementation Plan

1. Prefix each console summary line with its `seq` (from the entry, not its slice position), zero-padded, no brackets; adjust the `Console` formatter.
2. Keep the default list to one summary line per entry (plus a consistent subordinate source location if retained), and ensure the enriched payload is not rendered in the list.
3. Accept an optional bare-integer positional argument on the default command. On an integer, resolve the entry with that `seq` against the active session's full unfiltered set (add a `fetchConsoleEntries` helper mirroring the network command's `fetchNetworkEntries`, distinct from the filtered `getConsoleFromDaemon` list path) and render it in full; ignore the filter and `--head`/`--tail`/`--range` flags on this path; on a miss, error with the live `seq` range derived from that same unfiltered set; preserve the unknown-command error for non-integers.
4. Render the full entry for drill-down: the stack frames (one per line, function then location), all arguments including previews, exception class and subtype, and Log source, level, and network request id.
5. Convert `console --range` in `applyConsoleLimiting` from 1-indexed position windowing to inclusive `seq` membership, reusing the network command's `parseSeqRange` and membership approach; keep `--head` and `--tail` as entry counts. Remove the `ErrNoEntriesInRange` branch so an empty range returns an empty list with exit 0, and drop its now-dead handling from the console default and save paths.
6. Migrate the console JSON envelope from `logs` to `entries` across the list, drill-down, and save paths, and return the single-entry envelope (`entries` length one, `count` 1) for `console <n>` in JSON.
7. Create docs/console.md (it does not exist yet) to document the indexed list, drill-down, and the seq-based `--range`, update the `logs`-key assertions in internal/cli/cli_test.go to `entries`, and add tests for indexed display, drill-down hit and evicted-miss, seq-range selection, and the JSON single-entry response.

## Implementation Guidance

Mirror the network command's helpers and conventions rather than inventing new ones. The two commands should feel identical in their list and drill-down behavior, differing only in content: console has no bodies and no dial; it shows stacks and arguments instead.

The summary line's top frame with its function name is console's analog of the network main line. Lead with the most useful locator: the function name plus url:line.

For the drill-down stack, render one frame per line as function name then location. That matches how developers read a stack and keeps deep stacks legible.

When a Log-domain entry carries a network request id, surface it in the drill-down as the correlating network identity. Displaying the raw id is sufficient here; resolving it to the network entry's `seq` is a possible later enhancement.

## Acceptance Criteria

- `console` shows one indexed summary line per entry, and the index is a bare zero-padded number that matches the integer drill-down accepts.
- The default list does not show full stack traces or full argument dumps.
- `console <n>` shows the full entry: the complete stack with function names, all arguments, exception class, and Log source and network request id when present.
- `console <n>` on an evicted or unknown index errors with the live `seq` range.
- `console --range START-END` selects entries by inclusive `seq` membership, addressing the same numbers shown in the list.
- `console <n>` in JSON returns the single entry in the standard `entries` envelope; `console --json` and `console save` emit `entries` (not `logs`) and remain full fidelity.
- The display conventions match the network command: seq format, indent, drill-down syntax, `--range` semantics, JSON envelope key, and eviction wording.
- docs/console.md documents the indexed list, drill-down, and seq-based `--range`.
