# webctl network

Inspect the network requests captured for the active page. The default view is a compact indexed list; full request and response bodies are a deliberate drill-down rather than the default.

## Synopsis

```bash
webctl network                       # Indexed list, transport detail, no bodies
webctl network --detail summary      # One line per entry
webctl network --detail full         # List with request and response bodies
webctl network <n>                   # Drill into one entry by its seq
webctl network <n> --schema          # Preview an entry's JSON body shape
webctl network --json                # Full-fidelity JSON (untruncated)
webctl network save [path]           # Save the full JSON envelope to a file
```

## Description

The daemon buffers Chrome DevTools Protocol network events as they fire. Each buffered entry is assigned a stable sequence number (`seq`) when it is captured. The `network` command returns the active page session's entries, addressed by that `seq`.

On a content-heavy page the old always-bodies output ran into thousands of lines, almost all of it script, stylesheet, and document source. The redesigned default renders one indexed line per entry plus a transport detail block and no bodies, so an agent can scan the traffic, then fetch the payload for one specific entry.

The intended workflow is two steps:

1. List the traffic (optionally narrowed with filters or `--find`) to find the entry worth looking at.
2. Drill into that entry by its `seq` (`webctl network <n>`) to see its bodies, or preview a JSON body's shape with `--schema`.

## Detail levels

The `--detail` dial controls how much of each entry the text view renders. It applies to text output only; it is ignored in JSON mode and by `--schema`.

| Level | Renders |
|-------|---------|
| `summary` | One line per entry: the main line only. No transport block, no headers, no bodies. |
| `standard` | Main line plus the transport detail block (remote, timing, initiator). No bodies. This is the default. |
| `full` | Standard plus the request and response bodies, bounded by `--max-body-size`. |

A failed entry always shows its `FAILED` token and its failure reason at every level, because the reason is the point of a failed entry. Its transport block still follows the standard rule and its request body follows the full rule.

`--headers` adds request and response header blocks at the standard and full levels. At summary it is silently ignored.

## Indexed output

Every entry line begins with its `seq`, zero-padded to a minimum of two digits and growing naturally beyond (01, 09, 10, 99, 100, and up), with no surrounding brackets, followed by the main line:

```
01 GET https://example.com/ 200 45ms document 12.4KB
       remote: 93.184.216.34:443 h2 conn:1186
       timing: dns 12ms connect 30ms tls 20ms wait 40ms
02 GET https://example.com/app.js 200 8ms script 3.4KB
       remote: 93.184.216.34:443 h2 conn:1186
```

The displayed index is the same bare integer that drill-down accepts, so `webctl network 2` fetches the second entry above.

## Drill-down

```bash
webctl network 42
```

`webctl network <n>` returns the single entry whose `seq` is the integer `n`, rendered with its request and response bodies regardless of `--detail`, because asking for one entry by number is the explicit request to see its content. Those bodies are unbounded by default; the text `--detail full` cap of 102400 does not apply unless `--max-body-size` is set. `--headers` still adds headers.

Drill-down is an identity lookup, not a search. It addresses the active session's full unfiltered set and ignores the filter flags (`--find`, `--type`, `--method`, `--status`, `--url`, `--mime`, `--min-duration`, `--min-size`, `--failed`) and the `--head`/`--tail`/`--range` limiting, so a live entry is never hidden by a narrowing flag.

In JSON mode drill-down returns that one entry in the standard envelope shape (an `entries` array of length one with `count` 1), so a parser handles the one-entry and many-entry responses identically.

A `seq` the active session does not hold returns an error naming the lowest and highest `seq` currently held, as orientation:

```
entry 42 not in buffer (holds seq 318-425; run network to list)
```

Those bounds are orientation only; the held seqs may be sparse, so a value between them is not guaranteed present. Recover by re-running `network`, not by guessing another number. When the session holds no entries at all, the error reports the empty buffer instead:

```
entry 42 not in buffer (buffer empty)
```

## Schema preview

```bash
webctl network 42 --schema
```

`--schema` returns a token-efficient key skeleton of entry `n`'s JSON response body, so an agent can learn a body's shape without pulling the whole payload. It requires an entry index; `webctl network --schema` without one is an error. It resolves `n` through the same exact-membership lookup as drill-down, so a miss returns the same eviction-aware error rather than an empty schema.

The skeleton mirrors the body's structure with each leaf value replaced by its JSON type name. Arrays collapse to a single representative element whose object is the union of keys seen across all elements, so a heterogeneous array does not hide fields:

```json
{"ok":true,"schema":{"count":"number","vehicles":[{"id":"number","name":"string","options":["string"]}]}}
```

The schema is read from the full stored response body before any `--max-body-size` truncation, since a truncated body is not parseable JSON.

A non-JSON response body (HTML, script, binary, empty) is not an error. It returns the same envelope on stdout with exit 0, the schema `null` and a notice naming why:

```json
{"ok":true,"schema":null,"notice":"response body is not JSON (text/html)"}
```

Both outcomes share one envelope, one stream, and one exit code, so a single parser branches on the `schema` field: `null` means see the `notice`.

## Filtering and limiting

All filters are AND-combined. StringSlice flags support CSV (`--status 4xx,5xx`) and repeatable (`--status 4xx --status 5xx`) syntax.

| Flag | Description |
|------|-------------|
| `--find`, `-f` | Search for text within URLs and bodies. Narrows the list; because the default level renders no bodies, the matched body is seen by drilling into the entry. |
| `--type` | CDP resource type: `xhr`, `fetch`, `document`, `script`, `stylesheet`, `image`, `font`, `websocket`, `media`, `manifest`, `texttrack`, `eventsource`, `prefetch`, `other`. |
| `--method` | HTTP method: `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS`. |
| `--status` | Status code or range: `200`, `4xx`, `5xx`, `200-299`. |
| `--url` | URL regex pattern (Go regexp syntax). |
| `--mime` | MIME type: `application/json`, `text/html`, `image/png`. |
| `--min-duration` | Minimum request duration: `1s`, `500ms`, `100ms`. |
| `--min-size` | Minimum response size in bytes. |
| `--failed` | Show only failed requests (network errors, CORS, and so on). |
| `--head N` | Return the first N entries (a count over the seq-ordered list). |
| `--tail N` | Return the last N entries (a count over the seq-ordered list). |
| `--range START-END` | Keep entries whose `seq` is in `[START, END]` inclusive. |

`--head`, `--tail`, and `--range` are mutually exclusive.

`--range` selects entries by inclusive `seq` membership, matching the displayed indices. The held seqs are sparse, so the endpoints need not be present; the range names bounds and returns whatever held seqs fall inside them, empty when none do. For example, `webctl network --range 318-425` returns every held entry whose `seq` is between 318 and 425. This differs from `--head`/`--tail`, which remain entry counts rather than index references.

### The `--find` workflow

`--find` matches both URLs and body text, but the default level renders no bodies, so a body match narrows the list without showing the matched text:

```bash
webctl network --find "checkout"     # Narrow to entries matching "checkout"
webctl network 37                    # Drill into the interesting one to see its body
```

This is the intended two-step flow: `--find` surfaces the entries worth looking at, then drill-down shows the payload. The redesign deliberately does not promote matched bodies back into the list, which would rebuild the wall of text it removes.

## Body size control

`--max-body-size` accepts three regimes:

- A positive N caps bodies at N bytes.
- `0` suppresses all body content.
- `-1` means unlimited.

When the flag is not set, the default depends on the output mode:

| Mode | Unset default |
|------|---------------|
| `--detail full` text list | `102400` |
| JSON | unlimited |
| Drill-down (`network <n>`) text | unlimited |
| Save | unlimited (a saved file is a full-fidelity archive) |

An explicitly set value is honored in every mode. There is no single universal cap: the 102400 default applies only to the `--detail full` text list.

## JSON output

`webctl network --json` always returns complete entries: every field and every body, untruncated by default. The detail dial never reduces JSON output. An agent shapes its payload by querying — filters, `--head`/`--tail`/`--range`, drill-down — not by a verbosity flag. An explicit `--max-body-size` is still honored.

## Save mode

```bash
webctl network save                          # Save to temp dir with auto-filename
webctl network save ./logs/requests.json     # Save to a file
webctl network save ./output/                # Save to a directory (auto-filename)
webctl network save --status 5xx --tail 50   # Save a filtered subset
```

`network save` writes the full JSON envelope with untruncated bodies by default. The filter and limiting flags apply; the `--detail` dial and `--schema` do not. An explicit `--max-body-size` is honored.

## Flags

| Flag | Description |
|------|-------------|
| `--detail <level>` | Text detail level: `summary`, `standard`, or `full` (default `standard`). Text only. |
| `--schema` | Preview an entry's JSON response body as a key skeleton. Requires an entry index. |
| `--headers` | Show request and response headers (standard and full levels). |
| `--max-body-size <n>` | Body byte cap: `102400` for the `--detail full` text list, unlimited for JSON, drill-down, and save; `0` suppresses; `-1` unlimited. |
| `--find`, `-f <text>` | Search URLs and bodies. |
| `--type`, `--method`, `--status`, `--url`, `--mime`, `--min-duration`, `--min-size`, `--failed` | Filters (see above). |
| `--head`, `--tail`, `--range` | Limiting (see above; mutually exclusive). |
| `--json` | Emit full-fidelity JSON. |

## Error cases

- `No matches found` — the `--find` text is not in any request.
- `entry <n> not in buffer (...)` — drill-down or `--schema` to a `seq` the active session does not hold.
- `network --schema requires an entry index` — `--schema` used without an entry index.
- `daemon not running` — start the daemon first with `webctl start`.

## See also

- `webctl console` — inspect captured console logs.
- `webctl clear` — reset the capture buffers.
- `webctl start` — start the daemon and begin capturing.
