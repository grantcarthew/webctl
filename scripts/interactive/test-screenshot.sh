#!/bin/bash
# Title: webctl screenshot command tests

set -e

# Color output helpers
title() { echo -e "\n\033[1;34m=== $1 ===\033[0m"; }
heading() { echo -e "\n\033[1;32m## $1\033[0m"; }
cmd() {
    echo -e "\n\033[0;33m$ $1\033[0m"
    if [[ "$OSTYPE" == "darwin"* ]]; then echo "$1" | pbcopy; else echo "$1" | xclip -selection clipboard; fi
    echo "(Command copied to clipboard - paste and execute)"
    read -p "Press Enter to continue..."
}

clear
title "webctl screenshot Command Test Suite"
echo "Project: P-039"
echo "Tests screenshot capture with viewport and full-page modes"
echo ""
echo "Prerequisites:"
echo "  - webctl must be built"
echo "  - Clipboard tool (pbcopy on macOS, xclip on Linux)"
echo "  - Daemon running (or start one)"
echo "  - Image viewer to verify screenshots"
echo ""
read -p "Press Enter to begin tests..."

# Setup
title "Setup"
echo "Ensure daemon is running and navigate to test page"
cmd "webctl start"

echo ""
echo "Start daemon if not running, then continue"
read -p "Press Enter when daemon ready..."

heading "Navigate to test page"
cmd "webctl navigate https://example.com --wait"

echo ""
echo "Wait for page to fully load"
read -p "Press Enter when page loaded..."

# Default mode tests
title "Default Mode (Viewport to Temp)"

heading "Take viewport screenshot to temp"
cmd "webctl screenshot"

echo ""
echo "Verify: File saved to /tmp/webctl-screenshots/"
echo "Verify: Auto-generated filename (YY-MM-DD-HHMMSS-{title}.png)"
echo "Verify: JSON response shows file path"
echo "Verify: PNG file created (open to verify)"
echo "Verify: Viewport size matches browser window"
read -p "Press Enter to continue..."

heading "Take another viewport screenshot"
cmd "webctl screenshot"

echo ""
echo "Verify: Different filename (new timestamp)"
echo "Verify: Both screenshots exist"
read -p "Press Enter to continue..."

# Full-page mode tests
title "Full-Page Mode"

heading "Take full-page screenshot to temp"
cmd "webctl screenshot --full-page"

echo ""
echo "Verify: File saved to /tmp/webctl-screenshots/"
echo "Verify: PNG file created"
echo "Verify: Full scrollable height captured (open to verify)"
echo "Verify: File size likely larger than viewport screenshot"
read -p "Press Enter to continue..."

heading "Compare viewport vs full-page"
echo "Open both screenshots and compare:"
echo "  - Viewport: only visible area"
echo "  - Full-page: entire scrollable page"
read -p "Press Enter when compared..."

# Custom output path tests
title "Save Mode (Custom Path)"

heading "Save to custom file path"
cmd "webctl screenshot save ./debug/page.png"

echo ""
echo "Verify: File saved to ./debug/page.png"
echo "Verify: Parent directory created if needed"
echo "Verify: .png extension in filename"
read -p "Press Enter to continue..."

heading "Save viewport screenshot to custom path"
cmd "webctl screenshot save ./test.png"

echo ""
echo "Verify: File saved to ./test.png"
read -p "Press Enter to continue..."

heading "Save full-page to custom path"
cmd "webctl screenshot save ./full.png --full-page"

echo ""
echo "Verify: Full-page screenshot saved to ./full.png"
echo "Verify: Larger file size than viewport"
read -p "Press Enter to continue..."

# Custom output path tests (directory)
title "Save Mode (Directory)"

heading "Save to directory with auto-filename (trailing slash = directory)"
cmd "webctl screenshot save ./screenshots/"

echo ""
echo "Verify: Auto-generated filename in ./screenshots/"
echo "Verify: Directory exists or was created"
echo "Note: Trailing slash (/) is REQUIRED for directory behavior"
echo "      Without slash, it would create a file named 'screenshots'"
read -p "Press Enter to continue..."

heading "Full-page to directory"
cmd "webctl screenshot save ./screenshots/ --full-page"

echo ""
echo "Verify: Full-page screenshot in ./screenshots/ with auto-filename"
read -p "Press Enter to continue..."

# Filename generation tests
title "Filename Generation Tests"

heading "Take screenshots to verify filename format"
cmd "webctl screenshot"

echo ""
echo "Verify filename format:"
echo "  - Timestamp: YY-MM-DD-HHMMSS"
echo "  - Title: normalized (lowercase, hyphens)"
echo "  - Special characters removed"
echo "  - Extension: .png"
read -p "Press Enter to continue..."

heading "Navigate to page with long title"
cmd "webctl navigate \"https://en.wikipedia.org/wiki/List_of_lists_of_lists\" --wait"

echo ""
echo "Wait for Wikipedia to load"
read -p "Press Enter when loaded..."

heading "Take screenshot of long title page"
cmd "webctl screenshot"

echo ""
echo "Verify: Title truncated to ~30 chars"
echo "Verify: Title normalized (lowercase, hyphens)"
read -p "Press Enter to continue..."

heading "Navigate to page with special characters in title"
cmd "webctl navigate \"https://github.com\" --wait"

echo ""
echo "Wait for GitHub to load"
read -p "Press Enter when loaded..."

heading "Take screenshot with special char title"
cmd "webctl screenshot"

echo ""
echo "Verify: Special characters removed/normalized"
echo "Verify: Multiple hyphens collapsed to single"
echo "Verify: Leading/trailing hyphens removed"
read -p "Press Enter to continue..."

# Different page types
title "Different Page Types"

heading "Screenshot simple page"
cmd "webctl navigate https://example.com --wait"
cmd "webctl screenshot"

echo ""
echo "Verify: Simple page screenshot created"
read -p "Press Enter to continue..."

heading "Screenshot complex page (GitHub)"
cmd "webctl navigate https://github.com --wait"
cmd "webctl screenshot"

echo ""
echo "Verify: Complex page with images captured"
read -p "Press Enter to continue..."

heading "Screenshot long scrollable page"
cmd "webctl navigate \"https://en.wikipedia.org/wiki/Wikipedia\" --wait"

echo ""
echo "Wait for Wikipedia to load"
read -p "Press Enter when loaded..."

heading "Compare viewport vs full-page on long page"
cmd "webctl screenshot save ./wiki-viewport.png"
cmd "webctl screenshot save --full-page ./wiki-full.png"

echo ""
echo "Open both files to compare:"
echo "  - ./wiki-viewport.png: visible area only"
echo "  - ./wiki-full.png: entire scrollable page"
read -p "Press Enter when compared..."

# Full-page edge cases
title "Full-Page Edge Cases"

heading "Full-page on short page (fits in viewport)"
cmd "webctl navigate https://example.com --wait"
cmd "webctl screenshot save --full-page ./example-full.png"
cmd "webctl screenshot save ./example-viewport.png"

echo ""
echo "Compare file sizes - should be similar (page fits in viewport)"
read -p "Press Enter to continue..."

heading "Full-page on very long page"
cmd "webctl navigate \"https://en.wikipedia.org/wiki/World_War_II\" --wait"

echo ""
echo "Wait for long Wikipedia article to load"
read -p "Press Enter when loaded..."

cmd "webctl screenshot save --full-page ./ww2-full.png"

echo ""
echo "Verify: Very large file created"
echo "Verify: Entire article captured when opened"
read -p "Press Enter to continue..."

# Output format tests
title "Output Format Tests"

heading "JSON output format"
cmd "webctl screenshot --json"

echo ""
echo "Verify: JSON formatted response with file path"
read -p "Press Enter to continue..."

heading "No color output"
cmd "webctl screenshot --no-color"

echo ""
echo "Verify: No ANSI color codes in output"
read -p "Press Enter to continue..."

heading "Debug verbose output"
cmd "webctl screenshot --debug"

echo ""
echo "Verify: Debug logging information shown"
read -p "Press Enter to continue..."

# Multiple screenshots
title "Multiple Screenshots in Sequence"

heading "Take 3 screenshots rapidly"
cmd "webctl screenshot"
cmd "webctl screenshot"
cmd "webctl screenshot"

echo ""
echo "Verify: 3 unique files with different timestamps"
echo "Verify: All screenshots saved successfully"
read -p "Press Enter to continue..."

# Error cases
title "Error Cases"

heading "Save to invalid path"
cmd "webctl screenshot save /root/invalid/path.png"

echo ""
echo "Verify: Permission denied or path error"
read -p "Press Enter to continue..."

heading "Save to read-only directory (if applicable)"
echo "Skip if no read-only directory available"
read -p "Press Enter to skip or test manually..."

# File operations
title "File Operations"

heading "Overwrite existing file"
cmd "webctl screenshot save ./overwrite-test.png"
cmd "webctl screenshot save ./overwrite-test.png"

echo ""
echo "Verify: File overwritten (second screenshot replaces first)"
read -p "Press Enter to continue..."

heading "Create nested directories"
cmd "webctl screenshot save ./deep/nested/dirs/screenshot.png"

echo ""
echo "Verify: All parent directories created automatically"
echo "Verify: Screenshot saved in nested location"
read -p "Press Enter to continue..."

# REPL mode tests
title "REPL Mode Tests"

heading "Test screenshot in REPL"
echo "Switch to daemon terminal and execute:"
cmd "screenshot"

echo ""
echo "Should work identically to CLI mode"
read -p "Press Enter when tested in REPL..."

heading "Test full-page screenshot in REPL"
echo "In REPL, try:"
cmd "screenshot --full-page"

echo ""
echo "Should capture full-page screenshot"
read -p "Press Enter when tested in REPL..."

heading "Test custom output in REPL"
echo "In REPL, try:"
cmd "screenshot save ./repl-screenshot.png"

echo ""
echo "Should save to custom path"
read -p "Press Enter when tested in REPL..."

# Common workflow patterns
title "Common Workflow Patterns"

heading "Capture after navigation"
cmd "webctl navigate https://example.com --wait && webctl screenshot"

echo ""
echo "Verify: Navigate and capture in one command"
read -p "Press Enter to continue..."

heading "Before/after comparison workflow"
echo "1. Take before screenshot"
cmd "webctl screenshot save ./before.png"

echo "2. Interact with page (manual or via evaluate/click)"
read -p "Manually interact with page, then press Enter..."

echo "3. Take after screenshot"
cmd "webctl screenshot save ./after.png"

echo ""
echo "Verify: Two screenshots showing before/after state"
read -p "Press Enter to continue..."

heading "Debug layout issues"
cmd "webctl screenshot save --full-page ./layout-debug.png"

echo ""
echo "Use case: Capture entire page to debug layout problems"
read -p "Press Enter to continue..."

heading "CI/CD artifact pattern"
cmd "webctl screenshot save ./artifacts/build-123-screenshot.png"

echo ""
echo "Use case: Save screenshots with build IDs for CI/CD"
read -p "Press Enter to continue..."

# Advanced scenarios
title "Advanced Scenarios"

heading "Screenshot after page interaction"
cmd "webctl navigate https://github.com --wait"

echo ""
echo "Wait for GitHub to load"
read -p "Press Enter when loaded..."

cmd "webctl screenshot save ./github-initial.png"

echo ""
echo "Screenshot of initial state captured"
read -p "Scroll or interact with page, then press Enter..."

cmd "webctl screenshot save ./github-after-interaction.png"

echo ""
echo "Screenshot after interaction captured"
read -p "Press Enter to continue..."

heading "Visual regression testing workflow"
echo "Workflow:"
echo "1. Navigate to page"
echo "2. Take baseline screenshot"
echo "3. Make changes (code/CSS)"
echo "4. Take new screenshot"
echo "5. Compare images manually or with diff tool"
read -p "Press Enter to continue..."

# Verify file permissions
title "Verify File Permissions"

heading "Check screenshot file permissions"
cmd "ls -l /tmp/webctl-screenshots/*.png | head -5"

echo ""
echo "Verify: Files have 0644 permissions (rw-r--r--)"
read -p "Press Enter to continue..."

# Cleanup
title "Cleanup"
echo "Clean up test files if desired:"
cmd "rm -f ./test.png ./full.png ./debug/*.png ./screenshots/*.png ./wiki-*.png ./example-*.png ./ww2-*.png ./overwrite-test.png ./deep/nested/dirs/screenshot.png ./before.png ./after.png ./repl-screenshot.png ./github-*.png ./artifacts/*.png"

echo ""
echo "Remove test files when ready"
read -p "Press Enter to finish..."

title "Test Suite Complete"
echo "All screenshot command tests finished"
echo ""
echo "Review checklist in docs/projects/p-039-testing-screenshot.md"
echo "Document any issues discovered during testing"
echo ""
echo "Note: Remember to verify screenshots visually by opening the PNG files"
