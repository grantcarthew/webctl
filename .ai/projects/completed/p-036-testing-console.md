# P-036: Testing console Command

- Status: Completed
- Started: 2025-12-31
- Completed: 2026-01-14

## Overview

Test the webctl console command which extracts console logs from the current page. This command outputs to stdout by default, with a save subcommand for file output.

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
webctl console [save [path]] [--find text] [--type type] [--head N] [--tail N] [--range N-M] [--raw]
```

Subcommands:
- (default): Output console logs to stdout
- save: Save to /tmp/webctl-console/ with auto-generated filename
- save <path>: Save to file or directory/ (trailing slash = directory)

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

Default mode (stdout):
- [ ] console (all logs to stdout)
- [ ] console --type error (only errors to stdout)
- [ ] console --type error,warn (multiple types CSV)
- [ ] console --type error --type warn (multiple types repeatable)
- [ ] console --find "TypeError" (search and output)
- [ ] Verify formatted text output to stdout
- [ ] Verify no file created

Save mode (file output):
- [ ] console save (all logs to temp)
- [ ] console save --type error (only errors to temp)
- [ ] console save --find "undefined" (search and save)
- [ ] Verify file saved to /tmp/webctl-console/
- [ ] Verify auto-generated filename format (YY-MM-DD-HHMMSS-console.json)
- [ ] Verify JSON response with file path
- [ ] Verify JSON file structure (ok, logs, count)

Save mode (custom path):
- [ ] console save ./logs/debug.json (save to file)
- [ ] console save ./output/ (save to dir with auto-filename, creates dir)
- [ ] console save ./output (save to file named "output", NOT a directory)
- [ ] console save ./errors.json --type error --tail 50
- [ ] Verify trailing slash behavior

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
- [ ] CLI: webctl console save
- [ ] CLI: webctl console save ./logs.json
- [ ] CLI: webctl console --type error --tail 10
- [ ] REPL: console
- [ ] REPL: console save
- [ ] REPL: console save ./logs.json
- [ ] REPL: console --type error --tail 10

## Notes

- Default mode outputs to stdout (Unix convention)
- Save mode saves to temp or custom path
- Trailing slash convention: path/ = directory (auto-filename), path = file (like rsync)
- Type filter supports multiple types via CSV or repeatable flags
- Find flag searches within log message text (case insensitive)
- Head/tail/range flags mutually exclusive
- Raw flag outputs JSON instead of formatted text
- Auto-generated filenames use timestamp with "console" identifier
- Saved files contain JSON with ok, logs array, and count
- Console logs captured from browser's console API (log, warn, error, info, debug)

## Issues Discovered

(Issues will be documented here during testing)
