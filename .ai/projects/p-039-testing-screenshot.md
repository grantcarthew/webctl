# p-039: Testing screenshot Command

- Status: In Progress
- Started: 2025-12-31

## Overview

Test the webctl screenshot command which captures PNG screenshots of the current page. This command supports viewport or full-page capture with custom output paths.

## Test Script

Run the interactive test script:

```bash
./scripts/interactive/test-screenshot.sh
```

## Code References

- internal/cli/screenshot.go
- internal/cli/format (file path formatting)

## Command Signature

```
webctl screenshot [--full-page] [--output path]
```

Flags:
- --full-page: Capture entire scrollable page instead of viewport only
- --output, -o <path>: Save to specified path instead of temp directory
- --debug: Enable debug output (global flag)
- --json: Output in JSON format (global flag)
- --no-color: Disable colored output (global flag)

## Test Checklist

Default mode (viewport to temp):
- [ ] screenshot (viewport to temp)
- [ ] Verify file saved to /tmp/webctl-screenshots/
- [ ] Verify auto-generated filename format (YY-MM-DD-HHMMSS-{title}.png)
- [ ] Verify JSON response with file path
- [ ] Verify PNG file created
- [ ] Verify viewport size matches browser window

Full-page mode (to temp):
- [ ] screenshot --full-page (entire page to temp)
- [ ] Verify file saved to /tmp/webctl-screenshots/
- [ ] Verify PNG file created
- [ ] Verify full scrollable height captured
- [ ] Compare viewport vs full-page file sizes

Custom output path (file):
- [ ] screenshot -o ./debug/page.png
- [ ] screenshot --output ./test.png
- [ ] screenshot --full-page -o ./full.png
- [ ] Verify file saved to exact path
- [ ] Verify parent directories created
- [ ] Verify file extension (.png)

Custom output path (directory):
- [ ] screenshot -o ./output/
- [ ] Verify auto-generated filename in directory
- [ ] Verify directory exists or created

Filename generation:
- [ ] Verify timestamp format (YY-MM-DD-HHMMSS)
- [ ] Verify title normalization (lowercase, hyphens)
- [ ] Verify title truncation (30 chars max)
- [ ] Verify special characters removed
- [ ] Verify "untitled" fallback for empty titles
- [ ] Verify multiple hyphens collapsed to single
- [ ] Verify leading/trailing hyphens removed

Full-page flag:
- [ ] Viewport screenshot (default)
- [ ] Full-page screenshot (--full-page)
- [ ] Compare dimensions
- [ ] Test on short page (fits in viewport)
- [ ] Test on long page (requires scrolling)
- [ ] Test on wide page (horizontal scroll)

Output formats:
- [ ] Default JSON response (file path)
- [ ] --json output format
- [ ] --no-color output
- [ ] --debug verbose output

Multiple screenshots:
- [ ] Take multiple screenshots in sequence
- [ ] Verify unique filenames (timestamp)
- [ ] Verify all files saved

Different pages:
- [ ] Screenshot simple page
- [ ] Screenshot complex page with images
- [ ] Screenshot page with forms
- [ ] Screenshot page with animations
- [ ] Screenshot page after navigation
- [ ] Screenshot page after interaction (click)

Error cases:
- [ ] Failed to capture screenshot (CDP error)
- [ ] Failed to write screenshot (permission denied)
- [ ] Invalid output path
- [ ] No active session (error)
- [ ] Daemon not running

Edge cases:
- [ ] Empty page
- [ ] Very long page
- [ ] Very wide page
- [ ] Page with fixed elements
- [ ] Page with sticky headers
- [ ] Page with modal overlays
- [ ] Page with lazy-loaded content

File operations:
- [ ] Output to existing file (overwrite)
- [ ] Output to new file
- [ ] Output to non-existent directory (create)
- [ ] Output to read-only directory (error)
- [ ] Verify file permissions (0644)

CLI vs REPL:
- [ ] CLI: webctl screenshot
- [ ] CLI: webctl screenshot --full-page
- [ ] CLI: webctl screenshot -o ./page.png
- [ ] CLI: webctl screenshot --full-page -o ./full.png
- [ ] REPL: screenshot
- [ ] REPL: screenshot --full-page
- [ ] REPL: screenshot -o ./page.png

Common patterns:
- [ ] Capture after navigation (navigate + screenshot)
- [ ] Before/after comparison (screenshot, interact, screenshot)
- [ ] Multi-tab capture (target + screenshot)
- [ ] CI/CD artifacts (screenshot with build ID in path)
- [ ] Debug layout issues (full-page screenshot)
- [ ] Visual regression testing workflow

## Notes

- Default mode captures current viewport only
- Full-page mode captures entire scrollable page (may be very large)
- Output format is PNG (lossless) for accurate debugging
- Auto-generated filenames include timestamp and normalized page title
- Temp directory is /tmp/webctl-screenshots/
- Title normalization removes special chars, converts to lowercase, uses hyphens
- Screenshots saved with 0644 permissions
- Parent directories created automatically if needed
- Large full-page screenshots may take time to capture and write
- Screenshots useful for visual debugging, regression testing, documentation
- Works with multi-tab workflows via target command
- Can capture after page interactions for state verification

## Issues Discovered

(Issues will be documented here during testing)
