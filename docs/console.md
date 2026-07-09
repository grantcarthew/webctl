# webctl console

Inspect the console messages captured for the active page. The default view is a compact indexed list; the full per-entry detail (complete stack trace, all arguments, exception and Log-domain fields) is a deliberate drill-down rather than the default.

## Synopsis

```bash
webctl console                       # Indexed list, one summary line per entry
webctl console --type error          # Only errors
webctl console --find "undefined"    # Narrow to entries matching the text
webctl console <n>                   # Drill into one entry by its seq
webctl console --json                # Full-fidelity JSON (every field)
webctl console save [path]           # Save the full JSON envelope to a file
```

## Description

The daemon buffers Chrome DevTools Protocol console events as they fire: `console.*` API calls, uncaught exceptions, and Log-domain entries (network, security, deprecation, and so on). Each buffered entry is assigned a stable sequence number (`seq`) when it is captured. The `console` command returns the active page session's entries, addressed by that `seq`.

With deep capture in place, a single entry now carries a full stack trace, every argument with object previews, exception class and subtype, and Log-domain correlation. That detail does not belong inline in a scan, so the redesigned default renders one indexed summary line per entry and reserves the payload for drill-down.

The intended workflow is two steps:

1. List the messages (optionally narrowed with filters or `--find`) to find the entry worth looking at.
2. Drill into that entry by its `seq` (`webctl console <n>`) to see its full stack, arguments, and detail.

Unlike `network`, console has no detail dial. It is strictly two-level: the indexed list, and the full drill-down. There is no middle tier and no `--detail` flag.

## Indexed output

Every entry is one physical line, prefixed with its `seq`, zero-padded to a minimum of two digits and growing naturally beyond (01, 09, 10, 99, 100, and up), with no surrounding brackets. The line carries the wall-clock timestamp, the level, the top stack frame (function name and `url:line:column` when present), and the entry's message:

```
01 [15:04:05] LOG init app.js:12:4 booting
02 [15:04:06] ERROR handleClick app.js:42:10 TypeError: undefined is not a function
03 [15:04:06] WARNING deprecated.js:8:0 componentWillMount is deprecated
```

The timestamp is retained deliberately: console has no duration or timing block, so `[HH:MM:SS]` is its only wall-clock signal and the primary key for correlating a log against a network entry, a screenshot, or an external log.

The message is sourced from the entry's `Text`, which is present for every entry kind (console API calls, exceptions, and Log-domain entries alike). `Text` is stored verbatim and is frequently multi-line: an exception's `Text` is its full stack-dump description, and a multi-line `console.log` keeps its newlines. So the summary line shows only the first line of `Text`; the full multi-line message appears on drill-down. The line is not otherwise width-truncated.

The displayed index is the same bare integer that drill-down accepts, so `webctl console 2` fetches the second entry above.

## Drill-down

```bash
webctl console 42
```

`webctl console <n>` returns the single entry whose `seq` is the integer `n`, rendered in full. Below the summary line, on seven-space subordinate lines, it shows:

- the complete multi-line message, when `Text` spans more than one line;
- the full call stack, one frame per line as function name then location, including asynchronous continuation boundaries;
- every argument, with object descriptions and shallow property previews;
- the exception class and subtype, for an uncaught exception;
- the Log-domain source, the network request id, and the worker id, when the entry originates from the Log domain.

```
07 [15:04:06] ERROR handleClick app.js:42:10 TypeError: undefined is not a function
       stack:
         handleClick app.js:42:10
         dispatch app.js:9:3
       exception: TypeError (error)
```

Drill-down is an identity lookup, not a search. It addresses the active session's full unfiltered set and ignores the filter flags (`--find`, `--type`) and the `--head`/`--tail`/`--range` limiting, so a live entry is never hidden by a narrowing flag.

In JSON mode drill-down returns that one entry in the standard envelope shape (an `entries` array of length one with `count` 1), so a parser handles the one-entry and many-entry responses identically.

A `seq` the active session does not hold returns an error naming the lowest and highest `seq` currently held, as orientation:

```
entry 42 not in buffer (holds seq 318-425; run console to list)
```

Those bounds are orientation only; the held seqs may be sparse, so a value between them is not guaranteed present. Recover by re-running `console`, not by guessing another number. When the session holds no entries at all, the error reports the empty buffer instead:

```
entry 42 not in buffer (buffer empty)
```

## Filtering and limiting

| Flag | Description |
|------|-------------|
| `--find`, `-f` | Search for text within message text. Narrows the list. |
| `--type` | Filter by level: `log`, `warn`, `error`, `debug`, `info`. Repeatable and CSV-supported. |
| `--head N` | Return the first N entries (a count over the seq-ordered list). |
| `--tail N` | Return the last N entries (a count over the seq-ordered list). |
| `--range START-END` | Keep entries whose `seq` is in `[START, END]` inclusive. |

`--head`, `--tail`, and `--range` are mutually exclusive.

`--range` selects entries by inclusive `seq` membership, matching the displayed indices. The held seqs are sparse, so the endpoints need not be present; the range names bounds and returns whatever held seqs fall inside them, empty when none do. For example, `webctl console --range 318-425` returns every held entry whose `seq` is between 318 and 425. An empty range is an empty list with exit 0, not an error. This differs from `--head`/`--tail`, which remain entry counts rather than index references.

## JSON output

`webctl console --json` always returns complete entries: every field on every entry, never reduced by the list-versus-drill-down distinction. The envelope keys the array `entries` alongside a `count`, matching the network command:

```json
{"ok":true,"entries":[{"seq":1,"type":"log","text":"booting","timestamp":1700000000000}],"count":1}
```

An agent shapes its payload by querying — filters, `--head`/`--tail`/`--range`, drill-down — not by a verbosity flag.

Note: the JSON array key is `entries`, not `logs`. Earlier versions emitted `logs`; that key is retired across the list, drill-down, and save outputs.

## Save mode

```bash
webctl console save                          # Save to temp dir with auto-filename
webctl console save ./logs/debug.json        # Save to a file
webctl console save ./output/                # Save to a directory (auto-filename)
webctl console save --type error --tail 50   # Save a filtered subset
```

`console save` writes the full JSON envelope, keyed `entries`, with the filter and limiting flags applied. A saved file is a full-fidelity archive.

## Flags

| Flag | Description |
|------|-------------|
| `--find`, `-f <text>` | Search message text. |
| `--type <level>` | Filter by level (repeatable, CSV-supported). |
| `--head`, `--tail`, `--range` | Limiting (see above; mutually exclusive). |
| `--json` | Emit full-fidelity JSON. |

## Error cases

- `No matches found` — the `--find` text is not in any message.
- `entry <n> not in buffer (...)` — drill-down to a `seq` the active session does not hold.
- `unknown command "<arg>" for "webctl console"` — a non-integer positional argument.
- `daemon not running` — start the daemon first with `webctl start`.

## See also

- `webctl network` — inspect captured network requests.
- `webctl clear` — reset the capture buffers.
- `webctl start` — start the daemon and begin capturing.
