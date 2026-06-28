# Project: Capture Additional Network Telemetry the CDP Provides

## Goal

webctl stores only a subset of the per-request data the Chrome DevTools Protocol Network domain exposes. Investigate the additional fields available in the CDP network events and capture the ones that add real diagnostic value for an agent debugging network behaviour (remote endpoint, protocol, cache origin, connection reuse, timing breakdown, request initiator, redirect chain).

## Scope

In scope:

- Investigating the CDP Network domain event payloads webctl already subscribes to and identifying fields not currently parsed.
- Extending `ipc.NetworkEntry` and the daemon event handlers in `internal/daemon/events.go` to capture the high-value fields, and exposing them through JSON output.

Out of scope:

- The text formatter. Rendering new fields in text mode is the separate `project-network-text-view.md`. New fields appearing in JSON output is sufficient here.
- Request and response body capture. Already implemented.
- Subscribing to entirely new CDP domains. Stay within the Network domain events already handled unless a needed field strictly requires one more event in that same domain.

## Current State

Capture lives in `internal/daemon/events.go`. The daemon subscribes to CDP Network events and builds `ipc.NetworkEntry` values held in a ring buffer. What is parsed today:

- `Network.requestWillBeSent` (`createRequestEntry`): `Type`, `RequestTime` (from `wallTime`), `Request.Headers`. The request body path uses `hasPostData` plus an off-read-loop `Network.getRequestPostData` fetch.
- `Network.responseReceived` (`updateResponseEvent`): `Response.Status`, `Response.StatusText`, `Response.MimeType`, `Response.Headers`. The local parse struct discards every other `Response` field.
- `Network.loadingFinished` (`handleLoadingFinished`): `encodedDataLength` into `Size`, then an async `Network.getResponseBody` fetch.
- `Network.loadingFailed` (`handleLoadingFailed`): sets `Failed` and `Error`.

The CDP `Network.Response` object carries many fields webctl ignores: `remoteIPAddress`, `remotePort`, `protocol`, `fromDiskCache`, `fromServiceWorker`, `fromPrefetchCache`, `connectionReused`, `connectionId`, `securityState`, and a `timing` (`ResourceTiming`) breakdown with `dnsStart`/`dnsEnd`, `connectStart`/`connectEnd`, `sslStart`/`sslEnd`, `sendStart`/`sendEnd`, and `receiveHeadersEnd`. `Network.requestWillBeSent` additionally carries `initiator` (type plus stack), `redirectResponse` (the redirect chain), `Request.urlFragment`, and `documentURL`.

`ipc.NetworkEntry` has no fields for any of the above. Today the daemon derives `Duration` as a wall-clock delta between request and response timestamps; the CDP `timing` breakdown would allow a far more precise and granular picture (DNS, connect, TLS, TTFB).

## References

- CDP Network domain reference: https://chromedevtools.github.io/devtools-protocol/tot/Network/ — authoritative field list for `Request`, `Response`, `ResourceTiming`, `Initiator`, and the event payloads. Confirm exact field names and presence conditions against this before adding struct fields.

## Requirements

1. Enumerate the fields present in the CDP Network event payloads that webctl does not currently capture, with a one-line value assessment for each (diagnostic usefulness against noise and storage cost). The enumeration is a deliverable, not just an input.
2. Capture the high-value subset. At minimum: remote endpoint (`remoteIPAddress`, `remotePort`), `protocol`, cache origin (`fromDiskCache`, `fromServiceWorker`), the `timing` breakdown, and the request `initiator`. Decide and record a verdict on the remainder (`connectionReused`, `connectionId`, `securityState`, `redirectResponse`, `fromPrefetchCache`, `urlFragment`, `documentURL`).
3. Extend `ipc.NetworkEntry` with the chosen fields using `json:",omitempty"` tags and name-leading doc comments, matching the conventions of the existing body fields.
4. Populate the new fields in the appropriate handlers in `internal/daemon/events.go`, parsing them from the existing event payloads wherever possible rather than issuing new CDP round-trips.
5. Tests cover parsing and population of the new fields. Where an integration test (`internal/daemon/integration_test.go`) can assert a field against real Chrome traffic, add the assertion.

## Constraints

- Pure Go, no cgo, gofmt clean, `go vet` clean.
- Every new `NetworkEntry` field uses `json:",omitempty"` and a name-leading doc comment.
- Do not rename, remove, or change the JSON tags of existing fields.
- CDP calls inside event handlers run on the read loop and block waiting for a response on that same loop. A synchronous CDP call in a handler deadlocks the daemon (see the existing comment in `handleLoadingFinished`). Prefer parsing fields already present in the event payload. If any new field genuinely requires a fresh CDP call, fetch it asynchronously off the read loop, as the body fetches already do.
- Capturing a field must not block or slow the hot path of event handling for the common case.

## Implementation Plan

1. Against the CDP Network reference and a live capture, confirm the exact field names and the events that carry them. A live capture is available by running the daemon and inspecting raw event payloads.
2. Produce the enumeration and value assessment from Requirement 1. Resolve the keep/reject verdict for every field before changing code.
3. Extend the local parse structs in `updateResponseEvent` and `createRequestEntry` to read the chosen fields. Most live directly on the existing `Response` and `requestWillBeSent` payloads and need no extra CDP call.
4. Add the corresponding fields to `ipc.NetworkEntry` and assign them in the handlers. For the timing breakdown, decide whether to store the raw `ResourceTiming` offsets or derived phase durations, and record the rationale.
5. Add unit tests for the new parsing and population. Extend the network integration test to assert the high-confidence fields (for example `protocol` and `remoteIPAddress`) against real traffic.
6. If any field needs an off-read-loop CDP fetch, model it on the existing async body-fetch pattern, including its timeout and the in-place buffer update.

## Implementation Guidance

- Prefer fields already delivered in the subscribed events over new CDP method calls. The richest source is the `Response` object in `Network.responseReceived`, most of which is currently discarded.
- The `timing` breakdown is the highest-value addition for debugging slow requests, but its offsets are relative to a `requestTime` baseline. Decide on a single, documented representation rather than exposing raw CDP offsets that callers must interpret.

## Acceptance Criteria

- The chosen CDP fields appear in `webctl network --json` output, populated from real traffic.
- The network integration test asserts at least one newly captured field against live Chrome traffic.
- The enumeration of all considered fields, each marked captured or rejected with a reason, is recorded in the project's closing notes or as comments at the capture site.
- No existing `NetworkEntry` JSON field name or behaviour changes.

## Follow-ups

Once this project is complete, the newly captured fields land in `webctl network --json` but not in the text formatter; this project deliberately leaves text rendering out of scope. Create a new project to close that gap across both output modes.

- Surface the new fields in the text view. This is a repeat of `03-network-text-view.md`: classify each field captured here as shown by default, shown conditionally (for example behind `--headers` or a new flag), or deliberately omitted, with a recorded reason. Likely text candidates are the `timing` breakdown, `remoteIPAddress`/`remotePort`, and `protocol`; low-value-in-text fields such as `connectionId` or `securityState` likely stay JSON-only. The classification block in the `Network` doc comment in `internal/cli/format/text.go` is the seam to extend.
- Audit JSON completeness. Confirm every field this project captures is serialised, and check for any remaining captured-but-unexposed data before declaring the network output complete.
- Update `internal/cli/agent-help/observe.md` for any new text-view behaviour or flags, as `03` did.
