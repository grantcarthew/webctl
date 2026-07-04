# Console Capture Enrichment

## Goal

Capture the console diagnostic detail that Chrome DevTools Protocol exposes but webctl currently discards: full stack traces with function names, exception class, richer argument values, and browser Log-domain entries with network correlation. The result is console data genuinely useful for debugging, available in JSON to every consumer regardless of how it is displayed.

## Scope

In scope:

- Capture the full stack trace (all call frames with function name, url, line, column) for console API calls and for exceptions, replacing the current top-frame-only capture. Include the asynchronous continuation: enable async stack depth so CDP attaches the `StackTrace.parent` chain, and capture that chain so the logical call path through promises, timers, and event handlers survives.
- Capture the exception class and subtype for thrown exceptions.
- Capture argument fidelity: represent each console argument as a structured value (type, subtype, primitive value, non-primitive description, shallow preview) instead of a string, so non-primitives no longer collapse to null.
- Enable the Log domain and capture its entries (source, severity mapped onto the shared `Type`, text, url, line, stack trace, network request id, worker id).
- Redesign `ConsoleEntry` and its JSON around the enriched, structured fields.

Out of scope:

- Console presentation: list display, the sequence index, and drill-down belong to 06-console-index-and-drilldown.md. This project captures data; it does not change the formatter or command surface.
- The `seq` field, added by 03-buffer-sequence-index.md (already landed; preserve it).
- Any network command or capture changes.

## Dependencies

None blocking. 03-buffer-sequence-index.md has landed; `ConsoleEntry` already carries `seq`. This project reshapes the other `ConsoleEntry` fields and is free to change their types; preserve the `seq` field and its semantics.

## References

- Chrome DevTools Protocol, Runtime domain (consoleAPICalled, exceptionThrown, RemoteObject, StackTrace): https://chromedevtools.github.io/devtools-protocol/tot/Runtime/
- Chrome DevTools Protocol, Log domain (enable, entryAdded, LogEntry): https://chromedevtools.github.io/devtools-protocol/tot/Log/

## Current State

Console capture lives in internal/daemon/events.go.

- parseConsoleEvent handles Runtime.consoleAPICalled. It reads `type`, `timestamp`, and `args` (each as `{type, value}`; string values pass through, non-strings are `json.Marshal`ed; `Text` becomes the first arg). It reads only `stackTrace.callFrames[0]` and stores that frame's url, line, and column. It does not read `functionName` on any frame.
- parseExceptionEvent handles Runtime.exceptionThrown. It reads `exceptionDetails.text`, `url`, `lineNumber`, `columnNumber`, and `exception.description` (preferred over text). It does not read the exception stack trace or the exception class name.
- Domains enabled at startup are Runtime.enable, Page.enable, and DOM.enable. The Log domain is not enabled, so Log.entryAdded is never received. That event carries an entire class of browser messages that consoleAPICalled never delivers: deprecation warnings, security and CSP violations, blocked or failed network resources, interventions. It also carries `networkRequestId`, which links a log entry to a specific network request.
- ConsoleEntry (internal/ipc/protocol.go) has Seq (from project 03, already landed), SessionID, Type, Text, Args (currently `[]string`), Timestamp, URL, Line, Column.
- consoleBuf is a `RingBuffer[ipc.ConsoleEntry]` in the daemon, pushed from the event handlers.

Two losses are notable. First, a deep error stack is reduced to a single location with no function name, so the call chain behind an error is gone. Second, a logged object's RemoteObject usually carries no `value` (it has objectId, className, description, and a shallow preview instead), so `console.log(someObject)` currently records null or an empty value rather than anything meaningful.

## Requirements

1. Full stack trace. For both console API calls and exceptions, capture every call frame in the order CDP provides, each with function name, url, line, and column. Capture the asynchronous ancestry too: CDP delivers it on the nested `StackTrace.parent` chain, and only when async stack depth is enabled (see requirement 8), so an error thrown inside a promise, timer, or event handler retains its logical call chain rather than the shallow microtask stack. Walk the parent chain and represent the frames as a single ordered slice, marking each async boundary with the parent group's description (for example `Promise.then`) so the transition between synchronous and asynchronous frames is not lost. Keep populating the top-level URL, Line, and Column (from the first frame) as a convenience summary locator, and add the full frame list as a new field.
2. Function name. Capture the function name for each frame. An anonymous frame (empty function name) is represented as such, not dropped.
3. Exception detail. For exceptions, capture the exception class name and subtype in addition to the existing description text, and capture the exception stack trace via requirement 1.
4. Argument fidelity. Represent each console argument as a structured value mirroring the CDP RemoteObject: its type and subtype, the verbatim value for primitives, and for non-primitives the RemoteObject description plus a shallow property preview. Replace the string-based `Args []string` with a slice of this struct; do not stringify arguments. Derive the existing `Text` from the first argument's rendering (its value or description) so a message string remains available.
5. Log domain. Enable the Log domain for the relevant session(s) and capture Log.entryAdded entries into the console stream. Map the entry's `level` onto the shared `Type` field (verbose maps to debug; info, warning, and error pass through) so `--type` filtering and level display work uniformly across both event streams, and record the Log `source` (security, network, deprecation, ...) as a new field that distinguishes these entries. Preserve text, url, line, stack trace, network request id, and worker id, keeping the network request id verbatim so a consumer can correlate the entry to the network buffer.
6. Schema. Redesign `ConsoleEntry` and its JSON around the captured detail: a structured argument slice, a call-frame slice, exception class and subtype, and the Log-domain fields. Name fields clearly and use omitempty for fields that are frequently absent so ordinary entries stay compact.
7. Schema redesign, not additive. There is no on-the-wire compatibility requirement; existing field types and shapes may change. Where an existing field is retained (`Text`, top-level `URL`/`Line`/`Column`), it is kept as a derived convenience for summary display, not for compatibility. Preserve only `seq`, which project 03 owns.
8. Async stack depth. Enable `Runtime.setAsyncCallStackDepth` (depth 32) per session through the same domain-enable path used for Runtime, Page, and DOM, so CDP populates the `StackTrace.parent` chain that requirement 1 captures. This is a one-time per-session enable, not a per-event round trip, so it does not violate the off-the-critical-path constraint.

## Constraints

- Pure Go, no cgo, no new dependencies. gofmt and go vet clean.
- Enable the Log domain and async stack depth through the same domain-enable path the daemon already uses for Runtime, Page, and DOM. Do not invent a separate enabling mechanism.
- Do not perform per-argument round trips (for example Runtime.getProperties) to expand objects. Use the description and preview that CDP already delivers inline, so capture stays off the critical path.
- Capture runs in the daemon event loop. Keep parsing allocations modest and do not block the read loop on extra CDP calls; mirror the existing handler patterns.
- Entries are stored by value in the ring buffer. New fields, including the frame list, travel by value with the entry.

## Implementation Plan

1. Add `Runtime.setAsyncCallStackDepth` (depth 32) alongside the existing domain enables so CDP attaches the `StackTrace.parent` chain to console and exception events.
2. Extend the parseConsoleEvent parameter struct to read all call frames including function name, plus the nested `parent` StackTrace and its `description`. Flatten the callFrames and their parent chain into a single ordered frame slice, tagging each async boundary with the parent description, and keep the first frame populating the existing URL, Line, and Column.
3. Extend parseExceptionEvent to read `exceptionDetails.stackTrace.callFrames` and its `parent` chain (via the same frame-flattening helper as step 2), plus `exception.className` and subtype.
4. Replace `Args []string` with a structured argument slice. For each argument capture the RemoteObject type and subtype, the verbatim value for primitives, and the description plus shallow preview for non-primitives. Derive `Text` from the first argument's rendering.
5. Add a Log domain enable alongside the existing domain enables, and add a Log.entryAdded handler that builds console entries carrying source, the severity mapped onto Type, network request id, and the other Log fields, pushing them to consoleBuf.
6. Redefine the `ConsoleEntry` fields (structured Args, call-frame slice, exception class and subtype, Log fields) with JSON tags, preserving `seq`.
7. Add tests covering: a captured stack deeper than one frame; an async stack that carries frames from beyond the immediate microtask boundary; function names captured; an exception class captured; an object argument yielding a non-empty representation; a Log.entryAdded entry captured with its source and network request id.

## Implementation Guidance

Represent a call frame as a small struct (function name, url, line, column) and store a slice of them on the entry. This mirrors what CDP returns; do not flatten the stack into a string. The async parent chain flattens into the same slice: walk `StackTrace.parent` after the immediate `callFrames`, and carry the parent group's description on the boundary frame (a dedicated field on the frame struct, left empty for synchronous frames) so the async transition is visible without a separate nested shape.

Treat Log-domain entries as first-class console entries distinguished by a source label, not as a separate or lesser stream. The categories they carry (deprecation, CSP and security violations, blocked requests) are exactly what an agent debugging a page needs, and they never arrive through consoleAPICalled.

The network request id is the bridge to the network buffer. It pairs with the sequence-index work: a later consumer can resolve it to a network entry. Preserve the raw id here; resolving or displaying it is a consumer concern.

Keep object-argument capture shallow. The goal is to stop recording null for an object, not to serialize an object graph. The description plus a shallow preview is sufficient.

## Acceptance Criteria

- A thrown error's console entry carries the full call stack with function names, not just the top location.
- An error thrown inside an async callback (promise, timer, or event handler) carries frames from beyond the immediate microtask boundary, with the async transition marked, rather than a shallow synchronous stack.
- console.trace produces a multi-frame stack in the captured entry.
- An uncaught exception entry includes the exception class name (for example TypeError).
- console.log of an object records a non-empty representation rather than null.
- Messages surfaced through the Log domain (for example a deprecation or CSP violation) appear as console entries with a source and a severity in `Type` (so `console --type error` selects an error-level Log entry), and network-related ones carry the network request id.
- The enriched fields appear in console JSON, arguments serialize as structured values (not strings), and `seq` is preserved.
