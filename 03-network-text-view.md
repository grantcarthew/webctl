# Project: Surface Captured Network Data Missing From the Text View

## Goal

The `webctl network` text formatter renders only a fraction of the data the daemon already captures per request. Investigate every captured-but-unshown field and surface the ones that help a human or agent reading text output, with the failure reason as the clear priority: a failed request currently prints a bare status of 0 with no explanation.

## Scope

In scope:

- The `Network` text formatter in `internal/cli/format/text.go` and its tests in `internal/cli/format/format_test.go`.
- Deciding which already-captured `ipc.NetworkEntry` fields belong in the text view and how to render them without breaking the scannable one-entry-per-request layout.

Out of scope:

- Capturing new data from CDP. That is a separate project (`project-network-cdp-capture.md`).
- JSON output. It already serialises the full `NetworkEntry` and needs no change.
- Changing the `ipc.NetworkEntry` struct or any daemon handler.

## Current State

`ipc.NetworkEntry` (`internal/ipc/protocol.go`) carries these fields, all populated by the daemon during capture: `SessionID`, `RequestID`, `URL`, `Method`, `Type`, `Status`, `StatusText`, `MimeType`, `RequestTime`, `ResponseTime`, `Duration`, `Size`, `RequestHeaders`, `ResponseHeaders`, `RequestBody` (+`RequestBodyTruncated`), `ResponseBody` (+`ResponseBodyTruncated`), `ResponseBodyPath`, `Failed`, `Error`.

The `Network` function in `internal/cli/format/text.go` renders only:

- Main line: `METHOD URL STATUS DURATIONms` (status and method colourised on a TTY).
- Request body, when present, via `printNetworkBody`.
- Response body inline, or `response: [binary saved to <path>]` when the body was filed as binary.

Fields captured but never shown in text mode: `Type`, `StatusText`, `MimeType`, `Size`, `RequestHeaders`, `ResponseHeaders`, `Failed`, `Error`.

The most damaging gap is failure reporting. When a request fails (`loadingFailed`), the daemon sets `Failed = true`, leaves `Status` at 0, and stores a reason in `Error` (`"canceled"` or the CDP `errorText`). The text formatter has no failure branch, so a failed request prints as `GET https://example.com/x 0 0ms` — indistinguishable from a malformed entry and giving the reader no reason for the failure.

Design context from `AGENTS.md`: JSON output is the primary interface for AI agents; text output is a TTY convenience for humans. Text is therefore expected to be a curated subset, not a full dump. This project decides which subset, and treats the failure reason as non-negotiable.

## Requirements

1. A failed request must render its failure reason in text mode rather than a bare status of 0. Failed entries must be visually distinguishable from successful ones.
2. Assess each captured-but-unshown field (`Type`, `StatusText`, `MimeType`, `Size`, `RequestHeaders`, `ResponseHeaders`) for value in the text view. Surface the fields that aid comprehension at a glance; keep the per-entry output scannable.
3. Headers are high-volume. If surfaced at all, they must not clutter the default view. Decide between omitting them, summarising them, or gating them behind an existing or new verbosity flag, and record the decision.
4. Record the rationale for every field: shown by default, shown conditionally, or deliberately omitted. The outcome is a documented, intentional text view, not an accidental one.
5. Tests cover the failure-reason rendering and every newly surfaced field.

## Constraints

- Pure Go, gofmt clean, `go vet` clean. No cgo.
- Do not change JSON output or the `ipc.NetworkEntry` struct.
- Use the existing colour helpers (`colorFprint`, `colorFprintf`) and the `OutputOptions.UseColor` gate already used in `Network`; do not write raw ANSI codes.
- Preserve the existing one-entry-per-request shape. Multi-line detail (such as headers) must read as clearly subordinate to the main line, matching the existing body indentation style.

## Implementation Plan

1. Read the full `Network` function and `printNetworkBody` in `internal/cli/format/text.go` and the existing tests in `format_test.go` to internalise the current layout and colour conventions.
2. Enumerate the captured-but-unshown fields and classify each as default, conditional, or omitted, with a one-line reason. Resolve this classification before writing code.
3. Implement the failure branch first: detect `Failed`, render the reason from `Error`, and make failed entries visually distinct (for example a red status token plus the reason). Confirm a zero-status non-failed entry, if such a thing exists, is not mistaken for a failure.
4. Implement the remaining surfaced fields per the classification, keeping the main line compact and pushing any verbose detail to indented subordinate lines.
5. Add tests for the failure path and each newly surfaced field, asserting on rendered substrings as the existing network tests do.
6. Update the `observe` agent-help topic (`internal/cli/agent-help/observe.md`) if the text-view behaviour it describes changes.

## Acceptance Criteria

- A failed or canceled request rendered in text mode shows its failure reason and is visually distinguishable from a successful request.
- Every field surfaced by this project appears in text output and is covered by a test asserting its presence.
- The classification of each captured field as shown, conditional, or omitted is recorded in the formatter (comments) or the project's closing notes.
