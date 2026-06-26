# Capture Request POST Data

## Goal

Capture and surface the body of outgoing network requests (POST, PUT, PATCH, and any request with a payload) so agents can inspect what the page sent, not just what it received. Today webctl records request URL, method, and headers but discards the request body entirely; the only body it stores is the response body. Close that gap end to end: capture in the daemon, carry through the IPC contract, and render in both JSON and text output.

## Scope

In scope:
- Capturing request body data from CDP for every observed request that has one.
- Extending the network entry contract with a request-body field (and any truncation/oversize signalling that mirrors the existing response-body handling).
- Surfacing the request body in `webctl network` JSON output, text output, the `--find` search, and `webctl network save`.
- Setting CDP's `maxPostDataSize` on `Network.enable` so typical bodies arrive inline, with a fallback fetch for bodies that exceed the inline cap.
- Fixing the text formatter, which currently labels the response body as the request body.
- Updating affected docs and agent-help.

Out of scope:
- Response-body capture and storage. It already works; do not change its behaviour.
- Request/response interception or modification (`Fetch` domain, `Network.setRequestInterception`). This project observes; it does not mutate traffic.
- Capturing multipart file contents. CDP deliberately omits files from multipart request bodies (see References). Capture what CDP provides; do not attempt to reconstruct uploaded file payloads.
- New top-level commands or flags beyond what is needed to expose the request body. Reuse existing filtering and truncation surfaces.

## Current State

webctl is a daemon-plus-stateless-command browser-automation CLI in pure Go (module github.com/grantcarthew/webctl, Go 1.25.5 minimum, no cgo). The daemon holds a CDP WebSocket, subscribes to Network events, and buffers them in a ring buffer; stateless commands query the buffer over a Unix socket. See AGENTS.md for the full architecture and conventions.

Network capture today, by file:

- internal/daemon/events.go
  - `Network.requestWillBeSent` subscription calls `parseRequestEvent`, which reads only `request.url`, `request.method`, `request.headers`, plus `wallTime` and `type`. It ignores `request.postData` and `request.hasPostData`. The parsed entry is appended to `d.networkBuf`.
  - `Network.responseReceived` (`updateResponseEvent`) fills status, statusText, mimeType, response headers, and timing.
  - `Network.loadingFinished` (`handleLoadingFinished`) records size and fetches the response body. It does this in a goroutine and documents why: synchronous CDP calls inside an event handler deadlock, because the response to the call travels back through the same read loop that is currently blocked in the handler. Any new CDP call made from an event handler must follow this same off-the-read-loop pattern.
  - `Network.loadingFailed` (`handleLoadingFailed`) marks failures.
  - Response bodies are stored as text on the entry, or, for binary MIME types, written to a file via `saveBinaryBody` with the path stored on the entry (`internal/daemon/helpers.go`, `getBodiesDir` under `$XDG_STATE_HOME/webctl/bodies`).

- internal/ipc/protocol.go
  - `NetworkEntry` is the wire contract. It carries request fields (`Method`, `RequestHeaders`, `RequestTime`) and response fields (`Status`, `ResponseHeaders`, `Body`, `BodyTruncated`, `BodyPath`, `Size`, ...). `Body` is the response body. There is no request-body field.

- internal/daemon/session.go, daemon.go, handlers_observation.go, handlers_navigation.go
  - `Network.enable` is sent at three sites: `daemon.go` (auto-attach path), `handlers_observation.go` (lazy enable on first observe), and `handlers_navigation.go` (navigation path). All three pass `nil` params, so no `maxPostDataSize` is set. The at-most-once-per-session enable is guarded by `SessionManager.ClaimNetworkEnable` / `ClearNetworkEnabled`.

- internal/cli/network.go
  - Filters (`--method`, `--status`, `--type`, `--url`, `--mime`, `--failed`, head/tail/range), `--find` substring search (currently searches `entry.Body` only), `--max-body-size` truncation (default 102400 bytes) applied to `Body` in both JSON and raw paths, and `webctl network save`.

- internal/cli/format/text.go
  - `Network` renders each entry. Lines under the comment "Request body (if present and non-empty)" actually print `e.Body`, the response body, for non-GET methods. This is mislabeled and is the only place a "request body" is implied today.

- Docs and help
  - internal/cli/agent-help/observe.md documents `webctl network` usage. docs/ has no per-command network page (README.md, serve.md, start.md, testing.md only).

## References

- Chrome DevTools Protocol, Network domain (tip-of-tree): https://chromedevtools.github.io/devtools-protocol/tot/Network/
  - `Network.requestWillBeSent` → `request` (type `Network.Request`) provides `postData` (string; omitted when the body is too long), `hasPostData` (bool; set even when `postData` is omitted), and `postDataEntries` (experimental array of `PostDataEntry{bytes}`).
  - `Network.enable` parameter `maxPostDataSize` (integer): the longest post-body size in bytes that is included inline in `requestWillBeSent`. With it unset, `postData` is excluded from the event regardless of size and only `hasPostData` is set; the parameter is what opts bodies (up to its byte cap) into the event. This is why setting it (Req 2) is required, not an optimisation.
  - `Network.getRequestPostData(requestId)` → `{ postData: string }`. Returns an error when no data was sent with the request. For multipart requests it returns the body with form fields and boundaries intact, omitting only the uploaded file contents (a partial, non-empty body). The result is a plain string (not base64-encoded), so binary payloads may be lossy.
- AGENTS.md: repository architecture, build, testing, and CLI/daemon contracts.
- Existing precedent for off-read-loop CDP calls and body storage: internal/daemon/events.go (`handleLoadingFinished`) and internal/daemon/helpers.go (`saveBinaryBody`, `getBodiesDir`).

## Requirements

1. Capture request body data. When a request has a body, the daemon records it on the corresponding network entry. The capture must cover both the inline case (body present in `requestWillBeSent`) and the omitted-but-present case (`hasPostData` true, `postData` absent), fetching the latter via `Network.getRequestPostData`. A redirect chain reuses one `requestId` across hops, and each hop is a separate buffer entry, so the fetched body must land on the entry that originated the request rather than on a later hop that happens to share the `requestId`. The inline case is already correct because each entry receives its own event's `postData`; the omitted-body fallback must target the specific entry awaiting a body (see Implementation Plan step 3).

2. Set `maxPostDataSize` on every `Network.enable`. All three enable sites pass a consistent `maxPostDataSize` so typical request bodies (for example JSON API payloads) arrive inline without an extra round trip. Choose a cap consistent with the existing default body limit. The omitted-body fallback in requirement 1 must still exist for bodies larger than the cap.

3. Extend the IPC contract. Add `RequestBody` (JSON `requestBody`) and `RequestBodyTruncated` (JSON `requestBodyTruncated`) to `NetworkEntry`, distinct from the response `Body`. Keep the contract flat; do not nest request and response into sub-objects. This matches the existing flat naming (`requestHeaders`, `responseHeaders`, `requestTime`, `responseTime`, `body`, `bodyTruncated`) and avoids a breaking rewrite of every consumer. Both new fields use `omitempty` so they are absent from JSON when there is no request body, consistent with the other optional fields. Existing response-body fields keep their current names and meaning.

4. Off-read-loop capture. Any `Network.getRequestPostData` call made from an event handler runs off the CDP read loop, following the existing `handleLoadingFinished` pattern. No synchronous CDP call may be issued from inside an event handler.

5. Surface in JSON output. `webctl network` JSON includes the request body for entries that have one, as the flat `requestBody` field, with `requestBodyTruncated` set when `--max-body-size` truncates it. The examples below illustrate placement of the new `requestBody` and `requestBodyTruncated` fields; they are abridged. Always-present fields keep emitting (for example `requestTime` and `failed` have no `omitempty` and appear on every entry), and the existing field set is unchanged:

```json
{
  "requestId": "1000.42",
  "url": "https://api.example.com/login",
  "method": "POST",
  "type": "Fetch",
  "status": 200,
  "mimeType": "application/json",
  "requestHeaders": { "content-type": "application/json" },
  "requestBody": "{\"username\":\"grant\",\"password\":\"hunter2\"}",
  "responseHeaders": { "content-type": "application/json" },
  "body": "{\"token\":\"abc123\",\"expires\":3600}",
  "duration": 0.142
}
```

A request body truncated by `--max-body-size` carries the flag:

```json
{
  "requestId": "1000.57",
  "url": "https://api.example.com/upload",
  "method": "PUT",
  "requestBody": "{\"chunk\":\"AAAAAAAAAAAAAAAA",
  "requestBodyTruncated": true,
  "status": 201
}
```

6. Surface in text output. The text formatter labels the request body `request:` and the response body `response:`, request line first, each indented two spaces, with no arrow or other glyph prefix. Print the request line whenever a request body is present. Response-body display is otherwise unchanged from current behaviour (the existing method/empty gating stays); only its label is corrected. A single-line body follows the label on the same line after one space. A multi-line body prints the bare label line, then each body line indented four spaces. The exact form:

```
POST https://api.example.com/login 200 142ms
  request: {"username":"grant","password":"hunter2"}
  response: {"token":"abc123","expires":3600}
```

This replaces the current block that prints the response body under request-body intent.

7. Extend `--find`. The substring search matches against the request body in addition to the response body, so an agent can locate a request by its payload.

8. Preserve `webctl network save`. Saved output includes the request body (this follows automatically from the contract change, but verify it).

9. Multipart and binary behaviour is explicit. Store whatever CDP returns verbatim in `requestBody`, without fabricating, sniffing content types, or substituting placeholder text. For a multipart upload this is a partial body: CDP supplies the form fields and boundaries and omits only the uploaded file contents, so the partial body itself makes the omission visible. A body that is not valid text is stored without corrupting the buffer or the JSON encoding. `requestBody` is left empty only when CDP genuinely returns no data. Document the multipart-file limitation where request-body capture is described.

10. Documentation. Update internal/cli/agent-help/observe.md (and any other agent-help topic that describes network fields) to mention request-body capture, the inline-vs-fetched distinction, and the multipart-file limitation. If a per-field reference exists for network output, update it to list the new field.

11. Tests. Add unit coverage for parsing a `requestWillBeSent` event that carries `postData` and one that carries `hasPostData` without `postData`. Cover truncation of the request body at the CLI layer and inclusion of the request body in `--find`. Integration coverage (gated by `testing.Short()`) should assert that a real POST with a JSON body is captured and visible in `webctl network`.

## Constraints

- Pure Go, standard library and the existing go.mod dependency set only. No cgo, no new dependencies.
- Format with gofmt; pass `go vet` and staticcheck. Use the output helpers in internal/cli/root.go rather than writing to stdout/stderr directly.
- Idiomatic Go: explicit error returns, no panics in library code, exported identifiers documented with a comment beginning with the name.
- Do not regress the daemon-plus-stateless-command model, the at-most-once `Network.enable` guarantee, or response-body capture.
- Do not block the CDP read loop. CDP calls from event handlers must run asynchronously.
- Run `./test-runner quick` before pushing and `./test-runner ci` because this touches command surface, the daemon, and the IPC contract.
- Do not add Co-Authored-By trailers to commits.

## Implementation Plan

1. Extend the contract. Add the request-body field(s) to `NetworkEntry` in internal/ipc/protocol.go, mirroring the optionality and JSON-tag conventions of the existing response-body fields. Keep the response `Body` semantics unchanged.

2. Capture inline bodies. In `parseRequestEvent`, read `request.postData` and `request.hasPostData`. When `postData` is present, populate the new field on the entry as it is appended to the buffer.

3. Capture omitted bodies. When `hasPostData` is true but `postData` is absent, mark the just-pushed entry as awaiting its body (for example a transient `hasPostData`/pending flag on the entry), then fetch the body via `Network.getRequestPostData` using the event's session and request id, off the read loop, following the `handleLoadingFinished` goroutine pattern. When the body arrives, update the entry that is awaiting it — match on `requestId` and the awaiting/empty-`requestBody` condition, not merely the newest entry sharing the `requestId` — so a redirect hop cannot steal the body. Treat the "no data was sent" error as a non-error (nothing to store; clear the awaiting marker).

4. Set the inline cap. Add `maxPostDataSize` to the `Network.enable` params at all three enable sites. Prefer a single shared params value or helper so the three sites cannot drift. Size it to inline typical bodies while still relying on step 3 for larger ones.

5. Truncate at the CLI. Apply the existing `--max-body-size` truncation to the request body wherever it is applied to the response body in internal/cli/network.go (JSON and raw paths), setting the matching truncation flag.

6. Search the request body. Extend the `--find` predicate in internal/cli/network.go to also match the request body.

7. Render in text. In internal/cli/format/text.go, print the request body for requests that have one, labeled distinctly from the response body, and correct the existing mislabeled block.

8. Update docs and agent-help. Reflect request-body capture, the inline-vs-fetched behaviour, and the multipart-file limitation.

9. Tests. Add the unit and integration coverage from requirement 11. Run `./test-runner quick`, then `./test-runner ci`.

## Implementation Guidance

- The response-body path is the template for almost every decision here: how bodies are fetched off the read loop, how truncation and oversize are signalled, and how optional fields are tagged. Match it rather than inventing a parallel mechanism, so the two bodies behave predictably and read symmetrically. One place must differ: the response fetch updates the newest entry sharing the `requestId`, which is correct for a response (the final hop) but wrong for a request body (the originating hop). Use the awaiting-entry match from Implementation Plan step 3 for the request-body fallback instead of copying the newest-first matcher.
- Prefer not to write request bodies to files. Response bodies go to files only for binary MIME types; request bodies that webctl can observe are predominantly form-encoded or JSON text, and CDP already omits multipart file contents. Storing the request body as truncated text on the entry keeps the model simple. If a request body is not valid UTF-8, store it without corrupting the buffer or the JSON encoding rather than diverting to file storage.
- Truncation is settled: both bodies share the single `--max-body-size` threshold (no second flag), and the request body reports its own truncation via `requestBodyTruncated`. Apply it exactly where the response body's truncation is applied.
- Keep the three `Network.enable` sites consistent. The cleanest result is one place that builds the enable params; three hand-copied literals will drift.

## Acceptance Criteria

- A POST or PUT request issued by the page appears in `webctl network --json` with its request body populated, separate from the response `Body` field.
- A request whose body exceeds the inline `maxPostDataSize` still has its body captured (via the fallback fetch) and visible in output.
- `webctl network` text output shows the request body for a request that has one, and no longer prints the response body under request-body intent.
- `webctl network --find <substring-of-payload>` matches a request by its request body.
- `webctl network save` output contains the request body.
- `--max-body-size` truncates the request body and the entry signals the truncation.
- A multipart upload is captured without error; `requestBody` holds the partial body CDP provides (form fields and boundaries present, uploaded file contents absent), not a placeholder or an empty value.
- internal/cli/agent-help/observe.md documents request-body capture and the multipart-file limitation.
- `./test-runner ci` passes, including new unit tests for `postData` and `hasPostData` parsing and an integration test asserting a captured POST body.
