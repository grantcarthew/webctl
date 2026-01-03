# P-036: Testing console Command

- Status: In Progress
- Started: 2025-12-31

## Overview

Test the webctl console command which extracts console logs from the current page. This command supports three modes (default/show/save) with comprehensive filtering by type, text search, and range limiting.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-console.sh
```

## Code References

- internal/cli/console.go
- internal/cli/format (console formatting)

## Command Signature

```
webctl console [show|save <path>] [--find text] [--type type] [--head N] [--tail N] [--range N-M] [--raw]
```

Subcommands:
- (default): Save to /tmp/webctl-console/ with auto-generated filename
- show: Output console logs to stdout
- save <path>: Save to custom path

Universal flags (work with default/show/save modes):
- --find, -f <text>: Search for text within log messages
- --raw: Skip formatting (return raw JSON)
- --json: Output in JSON format (global flag)

Console-specific filter flags:
- --type <TYPE>: Filter by log type (log, warn, error, debug, info) - repeatable, CSV-supported
- --head <N>: Return first N entries
- --tail <N>: Return last N entries
- --range <N-M>: Return entries N through M (mutually exclusive with head/tail)

## Test Checklist

Default mode (save to temp):
- [ ] console (all logs to temp)
- [ ] console --type error (only errors to temp)
- [ ] console --find "undefined" (search and save)
- [ ] Verify file saved to /tmp/webctl-console/
- [ ] Verify auto-generated filename format (YY-MM-DD-HHMMSS-console.json)
- [ ] Verify JSON response with file path
- [ ] Verify JSON file structure (ok, logs, count)

Show mode (stdout):
- [ ] console show (all logs to stdout)
- [ ] console show --type error (only errors)
- [ ] console show --type error,warn (multiple types CSV)
- [ ] console show --type error --type warn (multiple types repeatable)
- [ ] console show --find "TypeError" (search and show)
- [ ] Verify formatted text output to stdout
- [ ] Verify no file created

Save mode (custom path):
- [ ] console save ./logs/debug.json (save to file)
- [ ] console save ./output/ (save to dir with auto-filename)
- [ ] console save ./errors.json --type error --tail 50
- [ ] Verify file saved to custom path

Type filter:
- [ ] --type log (log entries)
- [ ] --type warn (warning entries)
- [ ] --type error (error entries)
- [ ] --type debug (debug entries)
- [ ] --type info (info entries)
- [ ] --type error,warn (CSV format)
- [ ] --type error --type warn (repeatable format)
- [ ] Case insensitivity of type filter
- [ ] Invalid type (no matches)

Find flag:
- [ ] --find with simple text
- [ ] --find case insensitive matching
- [ ] --find with no matches (error)
- [ ] --find combined with --type

Head flag:
- [ ] --head 10 (first 10 entries)
- [ ] --head 1 (first entry)
- [ ] --head 100 when fewer entries exist
- [ ] --head with --type filter
- [ ] --head with --find filter

Tail flag:
- [ ] --tail 20 (last 20 entries)
- [ ] --tail 1 (last entry)
- [ ] --tail 100 when fewer entries exist
- [ ] --tail with --type filter
- [ ] --tail with --find filter

Range flag:
- [ ] --range 10-20 (entries 10 through 20)
- [ ] --range 0-10 (first 10 entries)
- [ ] --range 100-200 when fewer entries exist
- [ ] --range with invalid format (error)
- [ ] --range START-END where START >= END
- [ ] --range with --type filter
- [ ] --range with --find filter

Mutual exclusivity:
- [ ] --head and --tail together (error)
- [ ] --head and --range together (error)
- [ ] --tail and --range together (error)

Raw flag:
- [ ] --raw output (JSON format)
- [ ] Compare raw vs formatted output
- [ ] --raw with show mode
- [ ] --raw with filters

Combination tests:
- [ ] --type and --find together
- [ ] --type and --tail together
- [ ] --find and --head together
- [ ] Multiple filters applied in sequence

Output formats:
- [ ] Default JSON response (file path)
- [ ] Show mode text format (timestamp, type, message)
- [ ] --json with show mode
- [ ] --raw output format
- [ ] --no-color output
- [ ] --debug verbose output

Error cases:
- [ ] Find text not in logs (no matches error)
- [ ] Save to invalid path
- [ ] Invalid range format
- [ ] Mutually exclusive flags used together
- [ ] Daemon not running

CLI vs REPL:
- [ ] CLI: webctl console
- [ ] CLI: webctl console show
- [ ] CLI: webctl console save ./logs.json
- [ ] CLI: webctl console show --type error --tail 10
- [ ] REPL: console
- [ ] REPL: console show
- [ ] REPL: console save ./logs.json
- [ ] REPL: console show --type error --tail 10

## Notes

- Default mode saves to temp for quick debugging
- Show mode useful for real-time monitoring and piping
- Type filter supports multiple types via CSV or repeatable flags
- Find flag searches within log message text (case insensitive)
- Head/tail/range flags mutually exclusive
- Raw flag outputs JSON instead of formatted text
- Auto-generated filenames use timestamp with "console" identifier
- Saved files contain JSON with ok, logs array, and count
- Console logs captured from browser's console API (log, warn, error, info, debug)

## Issues Discovered

(Issues will be documented here during testing)
