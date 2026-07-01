# Buffer Sequence Index

## Goal

Give buffered event streams a stable, owned identifier so an individual entry can be addressed across separate daemon round-trips, independent of its position in the buffer. This is the foundation that lets a stateless CLI command list buffered entries, then fetch one specific entry by id in a later invocation.

## Scope

In scope:

- A monotonic sequence counter on the generic ring buffer, stamped onto each entry as it is pushed.
- A `seq` field on `ConsoleEntry` and `NetworkEntry`, carried through IPC and JSON.
- Resetting the counter when a buffer is cleared.
- Preserving each entry's `seq` across ring overwrite (eviction) and `RemoveIf` compaction.
- Applying the facility to both buffered streams (console and network), each of which has a consuming project.

Out of scope:

- Surfacing `seq` in command output. That belongs to the consuming projects: 05-network-output-redesign.md (network display and drill-down) and 06-console-index-and-drilldown.md (console display and drill-down).
- Any CLI flag, argument, or formatter changes.

## Current State

The daemon buffers events in a generic ring buffer.

- internal/daemon/buffer.go defines `RingBuffer[T any]`, a thread-safe fixed-capacity circular buffer guarded by a `sync.RWMutex`. Relevant methods: `Push` (appends, overwriting the oldest item when full), `All` (returns all items oldest-first, allocating a fresh slice), `Update` (newest-to-oldest in-place mutation), `RemoveIf` (removes matching items and compacts the survivors in place, preserving order), `Clear` (zeroes all items and resets `head`/`count`), `Len`, and `Cap`.
- internal/daemon/daemon.go declares two buffers: `consoleBuf *RingBuffer[ipc.ConsoleEntry]` and `networkBuf *RingBuffer[ipc.NetworkEntry]`, both constructed with `NewRingBuffer[...](cfg.BufferSize)`.
- internal/ipc/protocol.go defines `ConsoleEntry` (fields: SessionID, Type, Text, Args, Timestamp, URL, Line, Column) and `NetworkEntry` (a larger struct including RequestID, URL, Method, bodies, timing, initiator, and transport metadata). Both are serialized to JSON over the Unix-socket IPC and are the agent-facing data shape.
- internal/daemon/handlers_session.go handles the clear command, calling `Clear()` on the console buffer, the network buffer, or both, keyed on the target (`console`, `network`, `all`).

The buffer has no notion of entry identity today. Entries are addressable only by position in `All()`, which shifts when the ring overwrites the oldest entry or when `RemoveIf` compacts. There is no stable per-entry id.

## Requirements

1. The ring buffer assigns a stable, monotonically increasing sequence number to every entry at push time. The number is owned by the entry and does not change for the life of that entry in the buffer.
2. Sequence numbers start at 1 for the first entry pushed after construction or after a clear. The value 0 is reserved to mean unassigned (an entry that never passed through `Push`).
3. `ConsoleEntry` and `NetworkEntry` each carry the assigned sequence number in a `seq` field that is always present in JSON output (not omitted when zero), because it is a primary identifier an agent relies on.
4. Clearing a buffer resets its counter so the next pushed entry is again `seq` 1. Per-target clear semantics are unchanged: clearing console resets only the console counter, clearing network resets only the network counter.
5. An entry's `seq` survives ring overwrite. After the buffer wraps, the retained entries keep their original sequence numbers, so the lowest visible `seq` is greater than 1 and the visible range exposes how many entries were evicted.
6. An entry's `seq` survives `RemoveIf` compaction. Survivors keep their original numbers; compaction must not re-stamp them.
7. Sequence assignment is thread-safe, performed under the same lock that guards the rest of the push so concurrent pushes cannot collide or skip values.
8. A consumer holding the result of `All()` can determine the live sequence range (lowest and highest `seq` currently buffered) from the returned entries, which are ordered oldest-first. No separate lookup API is required, but if one is added it must report a clear found / not-found result suitable for an eviction-aware error message.

## Constraints

- Pure Go, no cgo, no new dependencies. Module Go version is pinned in go.mod (1.25.5 minimum).
- Format with gofmt; pass go vet.
- Preserve the existing thread-safety contract of `RingBuffer`. The counter increment and the entry stamp happen inside the existing write lock in `Push`.
- Do not special-case a concrete element type inside the buffer. The stamping mechanism must remain generic over the buffer's element type and apply equally to console and network entries.
- Do not change `All`, `Update`, or `RemoveIf` ordering semantics. They already preserve insertion order; `seq` must remain consistent with that order.

## Implementation Plan

1. Add a monotonic unsigned counter to `RingBuffer`. Increment it under the write lock in `Push` and stamp the resulting value onto the entry being stored.
2. Stamp the field through a construction-time function, not a type constraint. Give the buffer a `stamp func(*T, uint64)` supplied to `NewRingBuffer`; `Push` calls `b.stamp(&b.items[b.head], seq)` only when `stamp` is non-nil. Do not add a pointer-receiver setter constraint (for example `RingBuffer[T any, PT interface{ *T; SetSeq(uint64) }]`): that retroactively forbids `RingBuffer[int]`, which the existing buffer tests instantiate, and would force rewriting every generic test onto a bespoke element type. The function form keeps the buffer fully general and lets non-stamped element types pass a nil stamp; do not branch on the concrete type inside the buffer.
3. Add the `seq` field to `ConsoleEntry` and `NetworkEntry`. Construct the console and network buffers with stamp functions that set it (`func(e *ipc.ConsoleEntry, s uint64) { e.Seq = s }` and the network equivalent). Update the remaining `NewRingBuffer` call sites, including `RingBuffer[int]` in buffer_test.go, to pass a nil stamp.
4. Reset the counter in `Clear` so post-clear pushes restart at 1.
5. Confirm `RemoveIf` carries `seq` through compaction unchanged. Because it copies surviving entries by value into the rebuilt buffer, the stamped field travels with them; verify no path re-stamps a survivor.
6. Add unit tests in internal/daemon/buffer_test.go covering: monotonic assignment from 1; stability of `seq` across repeated `All()` calls; gaps in the visible range after the ring wraps; `seq` preservation through `RemoveIf`; and counter reset after `Clear`.

## Implementation Guidance

The stamp-on-push design keeps identity assignment in one place and guarantees every buffered entry has an id without each event handler having to manage a counter. Prefer it over assigning `seq` in the daemon event handlers, which would duplicate the counter per stream and risk drift.

For network entries specifically, stamping at `Push` gives redirect hops distinct sequence numbers even though they share a CDP `RequestID`, because each hop is a separate push. This is desirable: `seq` is the unambiguous address where `RequestID` is not.

An unsigned counter avoids any question of negative ids and comfortably covers a long-lived daemon session.

## Acceptance Criteria

- Every entry returned by `All()` on a non-empty buffer has a nonzero `seq`, and the values strictly increase in push order.
- Repeated `All()` calls return the same `seq` for the same entry; the value is not recomputed from position.
- After the buffer is filled beyond capacity, the lowest `seq` among retained entries is greater than 1 and the highest equals the total number of pushes, so the evicted count is visible from the range.
- After `Clear`, the next pushed entry has `seq` 1.
- After `RemoveIf` removes some entries, every survivor retains its original `seq`.
- `seq` appears in the JSON serialization of both `ConsoleEntry` and `NetworkEntry`.
