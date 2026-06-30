# Console Capture Enrichment

## Goal

Capture the console diagnostic detail that Chrome DevTools Protocol exposes but webctl currently discards: full stack traces with function names, exception class, richer argument values, and browser Log-domain entries with network correlation. The result is console data genuinely useful for debugging, available in JSON to every consumer regardless of how it is displayed.

## Scope

In scope:

- Capture the full stack trace (all call frames with function name, url, line, column) for console API calls and for exceptions, replacing the current top-frame-only capture.
- Capture the exception class and subtype for thrown exceptions.
- Capture argument fidelity: for non-primitive console arguments, capture a representation that does not collapse to null.
- Enable the Log domain and capture its entries (source, level, text, url, line, stack trace, network request id, worker id).
- Extend `ConsoleEntry` and its JSON with the new fields.

Out of scope:

- Console presentation: list display, the sequence index, and drill-down belong to 06-console-index-and-drilldown.md. This project captures data; it does not change the formatter or command surface.
- The `seq` field, which is added by 03-buffer-sequence-index.md.
- Any network command or capture changes.

## Dependencies

None blocking. Shares the `ConsoleEntry` struct with 03-buffer-sequence-index.md (which adds `seq`). The two projects both extend `ConsoleEntry` additively; land them in sequence rather than in parallel to avoid a struct merge conflict.

## References

- Chrome DevTools Protocol, Runtime domain (consoleAPICalled, exceptionThrown, RemoteObject, StackTrace): https://chromedevtools.github.io/devtools-protocol/tot/Runtime/
- Chrome DevTools Protocol, Log domain (enable, entryAdded, LogEntry): https://chromedevtools.github.io/devtools-protocol/tot/Log/

## Current State

Console capture lives in internal/daemon/events.go.

- parseConsoleEvent handles Runtime.consoleAPICalled. It reads `type`, `timestamp`, and `args` (each as `{type, value}`; string values pass through, non-strings are `json.Marshal`ed; `Text` becomes the first arg). It reads only `stackTrace.callFrames[0]` and stores that frame's url, line, and column. It does not read `functionName` on any frame.
- parseExceptionEvent handles Runtime.exceptionThrown. It reads `exceptionDetails.text`, `url`, `lineNumber`, `columnNumber`, and `exception.description` (preferred over text). It does not read the exception stack trace or the exception class name.
- Domains enabled at startup are Runtime.enable, Page.enable, and DOM.enable. The Log domain is not enabled, so Log.entryAdded is never received. That event carries an entire class of browser messages that consoleAPICalled never delivers: deprecation warnings, security and CSP violations, blocked or failed network resources, interventions. It also carries `networkRequestId`, which links a log entry to a specific network request.
- ConsoleEntry (internal/ipc/protocol.go) has SessionID, Type, Text, Args, Timestamp, URL, Line, Column. Project 03 adds `seq`.
- consoleBuf is a `RingBuffer[ipc.ConsoleEntry]` in the daemon, pushed from the event handlers.

Two losses are notable. First, a deep error stack is reduced to a single location with no function name, so the call chain behind an error is gone. Second, a logged object's RemoteObject usually carries no `value` (it has objectId, className, description, and a shallow preview instead), so `console.log(someObject)` currently records null or an empty value rather than anything meaningful.

## Requirements

1. Full stack trace. For both console API calls and exceptions, capture every call frame in the order CDP provides, each with function name, url, line, and column. Keep populating the existing top-level URL, Line, and Column (from the first frame) for backward compatibility, and add the full frame list as a new field.
2. Function name. Capture the function name for each frame. An anonymous frame (empty function name) is represented as such, not dropped.
3. Exception detail. For exceptions, capture the exception class name and subtype in addition to the existing description text, and capture the exception stack trace via requirement 1.
4. Argument fidelity. For non-primitive console arguments, capture a representation that preserves meaning: the RemoteObject description and, where present, a shallow property preview. Primitive arguments keep their current value rendering.
5. Log domain. Enable the Log domain for the relevant session(s) and capture Log.entryAdded entries into the console stream, preserving source, level, text, url, line, stack trace, network request id, and worker id. Preserve the network request id verbatim so a consumer can correlate the entry to the network buffer.
6. Schema. Extend `ConsoleEntry` and its JSON with the new fields, named clearly, using omitempty for fields that are frequently absent so ordinary entries stay compact.
7. Backward compatibility. The existing fields keep their current meaning. All new data is additive.

## Constraints

- Pure Go, no cgo, no new dependencies. gofmt and go vet clean.
- Enable the Log domain through the same domain-enable path the daemon already uses for Runtime, Page, and DOM. Do not invent a separate enabling mechanism.
- Do not perform per-argument round trips (for example Runtime.getProperties) to expand objects. Use the description and preview that CDP already delivers inline, so capture stays off the critical path.
- Capture runs in the daemon event loop. Keep parsing allocations modest and do not block the read loop on extra CDP calls; mirror the existing handler patterns.
- Entries are stored by value in the ring buffer. New fields, including the frame list, travel by value with the entry.

## Implementation Plan

1. Extend the parseConsoleEvent parameter struct to read all call frames including function name. Build a slice of frames on the entry, and keep the first frame populating the existing URL, Line, and Column.
2. Extend parseExceptionEvent to read `exceptionDetails.stackTrace.callFrames` and `exception.className` and subtype.
3. Extend argument parsing so a non-primitive RemoteObject contributes its description and shallow preview rather than a null value.
4. Add a Log domain enable alongside the existing domain enables, and add a Log.entryAdded handler that builds console entries carrying source, level, network request id, and the other Log fields, pushing them to consoleBuf.
5. Add the new fields to `ConsoleEntry` with JSON tags.
6. Add tests covering: a captured stack deeper than one frame; function names captured; an exception class captured; an object argument yielding a non-empty representation; a Log.entryAdded entry captured with its source and network request id.

## Implementation Guidance

Represent a call frame as a small struct (function name, url, line, column) and store a slice of them on the entry. This mirrors what CDP returns; do not flatten the stack into a string.

Treat Log-domain entries as first-class console entries distinguished by a source label, not as a separate or lesser stream. The categories they carry (deprecation, CSP and security violations, blocked requests) are exactly what an agent debugging a page needs, and they never arrive through consoleAPICalled.

The network request id is the bridge to the network buffer. It pairs with the sequence-index work: a later consumer can resolve it to a network entry. Preserve the raw id here; resolving or displaying it is a consumer concern.

Keep object-argument capture shallow. The goal is to stop recording null for an object, not to serialize an object graph. The description plus a shallow preview is sufficient.

## Acceptance Criteria

- A thrown error's console entry carries the full call stack with function names, not just the top location.
- console.trace produces a multi-frame stack in the captured entry.
- An uncaught exception entry includes the exception class name (for example TypeError).
- console.log of an object records a non-empty representation rather than null.
- Messages surfaced through the Log domain (for example a deprecation or CSP violation) appear as console entries with a source, and network-related ones carry the network request id.
- The new fields appear in console JSON; the pre-existing fields are unchanged.
