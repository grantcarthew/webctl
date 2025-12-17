# DR-011: Screenshot Command Interface

- Date: 2025-12-17
- Status: Accepted
- Category: CLI

## Problem

AI agents debugging web applications need to capture visual state of pages for analysis, bug reporting, and documentation. Screenshots provide context that console logs and network requests cannot - visual layout issues, rendering bugs, UI state, and user-visible errors.

Requirements:

- Capture viewport screenshots for current page state
- Capture full-page screenshots for complete page analysis
- Return file path in JSON for agent processing
- Support custom output paths for organized storage
- Work with active session in multi-tab scenarios
- High-quality output for debugging (sufficient resolution)

## Decision

Implement `webctl screenshot` command with the following interface:

```bash
webctl screenshot [flags]
```

Flags:

| Flag | Short | Type | Description |
|------|-------|------|-------------|
| --full-page | | bool | Capture entire scrollable page instead of viewport |
| --output | -o | string | Save to specified path instead of temp directory |

Output format:

JSON response with file path:

```json
{
  "ok": true,
  "path": "/tmp/webctl-screenshots/25-12-17-212345-example-domain.png"
}
```

Default file location: `/tmp/webctl-screenshots/`

Filename pattern: `YY-MM-DD-HHMMSS-{normalized-title}.png`

Title normalization algorithm:

1. Extract page title from active session
2. Trim leading and trailing whitespace
3. Limit to 30 characters maximum
4. Convert all non-alphanumeric characters to hyphens
5. Replace multiple consecutive hyphens with single hyphen
6. Remove leading and trailing hyphens
7. Convert to lowercase for consistent sorting

Examples:

- "Example Domain" → `example-domain`
- "React App - Development Server!" → `react-app-development-server`
- "JSONPlaceholder - Free Fake REST API for Testing" → `jsonplaceholder-free-fake-res` (30 chars)
- "   Lots   of---Spaces!!!   " → `lots-of-spaces`

Image format: PNG only (lossless, suitable for debugging)

## Why

Save to temp directory by default:

Agents benefit from automatic file management. The temp directory approach:

- Provides predictable location agents can rely on
- Uses OS-managed cleanup (no manual file management needed)
- Avoids cluttering working directory
- Returns path in JSON for easy agent consumption
- Consistent with tool's agent-first design philosophy

JSON output with file path:

Unlike binary stdout (typical Unix approach), returning JSON with file path:

- Maintains consistent JSON output format across all commands
- Simplifies error handling (errors use standard JSON error format)
- Easier for agents to parse (no binary data handling needed)
- File persists for later inspection or multi-step workflows
- Aligns with agent-focused design (agents prefer structured data)

Filename with timestamp and title:

Chronological sorting (YY-MM-DD-HHMMSS prefix) allows easy navigation:

- `ls -la` shows screenshots in time order automatically
- Debugging sessions produce naturally organized files
- Human-readable timestamps aid manual inspection

Including normalized page title:

- Provides context when browsing temp directory
- Makes files self-documenting
- Helps identify screenshots without opening them

30 character title limit:

- Prevents excessive filename length
- Balances context with filesystem limits
- Most page titles frontload important information

Full-page screenshot support:

Different debugging scenarios require different capture modes:

- Viewport: "What does the user see right now?"
- Full-page: "What is the complete rendered output?"

Supporting both enables comprehensive debugging workflows.

PNG format only:

PNG is lossless and suitable for debugging scenarios:

- No compression artifacts (JPEG quality issues)
- Supports transparency (important for overlay debugging)
- Standard format universally supported
- File size acceptable for debugging use (not production asset)

JPEG would only benefit file size at cost of quality and complexity.

Custom output path support:

While temp directory covers most use cases, --output enables:

- Organized test artifacts in CI/CD pipelines
- Permanent documentation screenshots
- Integration with existing tooling/workflows

Optional flag keeps simple case simple while enabling advanced use.

## Trade-offs

Accept:

- Temp files require eventual OS cleanup (disk space usage)
- Title normalization may produce non-unique filenames (timestamp provides uniqueness)
- JSON response requires parsing (not direct binary pipe like traditional tools)
- PNG-only limits format flexibility
- Filename pattern is opinionated (some users may prefer different schemes)

Gain:

- Consistent JSON interface across all commands
- No special error handling for binary output modes
- Files persist for multi-step agent workflows
- Automatic chronological organization
- Predictable, documented behavior
- Agent-friendly structured responses
- High-quality lossless screenshots

## Alternatives

Binary PNG to stdout:

```bash
webctl screenshot > page.png
```

- Pro: Traditional Unix approach, familiar pattern
- Pro: No temp file management needed
- Con: Errors cannot use JSON format on stdout (must use stderr)
- Con: Requires special error handling (binary vs text output)
- Con: Less convenient for agents (must handle binary data)
- Con: File does not persist (must redirect or lose data)
- Rejected: JSON-first approach better for agent workflows

JSON with base64-encoded image:

```json
{"ok": true, "data": "iVBORw0KGgoAAAANS..."}
```

- Pro: Single JSON response, no file I/O
- Pro: Works over network protocols easily
- Con: Large JSON payloads (base64 encoding overhead)
- Con: Agent must decode and write file anyway
- Con: Not practical for full-page screenshots (multi-MB payloads)
- Rejected: File-based approach more efficient

Save to working directory by default:

```bash
webctl screenshot  # Creates screenshot-*.png in current directory
```

- Pro: User can immediately see the file
- Pro: No temp directory needed
- Con: Clutters working directory with debugging artifacts
- Con: Requires cleanup from user
- Con: May conflict with existing files
- Rejected: Temp directory cleaner for debugging workflows

Include session ID in filename:

```bash
25-12-17-212345-9a3e8d71-example-domain.png
```

- Pro: Guarantees filename uniqueness across sessions
- Pro: Identifies which tab produced screenshot
- Con: Longer, less readable filenames
- Con: Session ID not meaningful to humans
- Con: Timestamp already provides uniqueness
- Rejected: Timestamp sufficient, session ID adds noise

Support JPEG format:

```bash
webctl screenshot --format jpeg --quality 80
```

- Pro: Smaller file sizes
- Pro: Quality control for bandwidth-constrained scenarios
- Con: Lossy compression inappropriate for debugging
- Con: Additional flags and complexity
- Con: No meaningful benefit for local debugging
- Rejected: PNG sufficient for debugging use case

Implement selector-based screenshots now:

```bash
webctl screenshot --selector ".main-content"
```

- Pro: Captures specific elements only
- Pro: Smaller images, focused debugging
- Con: Adds complexity (element selection, bounding box calculation)
- Con: Unclear if agents would use this feature
- Con: Can defer to future if demand emerges
- Rejected: Defer until use cases are clear

Configurable temp directory:

```bash
webctl screenshot  # Uses config setting for temp location
```

- Pro: User control over file location
- Pro: Can use faster storage or specific mount points
- Con: Adds configuration complexity
- Con: Breaking predictability (agents must check config)
- Con: /tmp is standard and appropriate
- Rejected: Standard /tmp sufficient

## Usage Examples

Basic viewport screenshot:

```bash
webctl screenshot
# {"ok": true, "path": "/tmp/webctl-screenshots/25-12-17-212345-example-domain.png"}
```

Full-page screenshot:

```bash
webctl screenshot --full-page
# {"ok": true, "path": "/tmp/webctl-screenshots/25-12-17-212346-example-domain.png"}
```

Custom output path:

```bash
webctl screenshot --output ./docs/screenshot.png
# {"ok": true, "path": "./docs/screenshot.png"}

webctl screenshot -o /tmp/debug/issue-123.png
# {"ok": true, "path": "/tmp/debug/issue-123.png"}
```

Agent workflow capturing visual state:

```bash
webctl navigate https://example.com
webctl screenshot --full-page
# Agent parses JSON, extracts path, analyzes image or includes in report
```

Debugging visual rendering:

```bash
webctl navigate https://myapp.local
webctl click "#theme-toggle"
webctl screenshot
# Compare before/after screenshots for theme switching bug
```

CI/CD test artifact:

```bash
webctl navigate https://staging.example.com
webctl screenshot --full-page --output ./test-results/homepage-${CI_BUILD_ID}.png
# Organized artifacts in test results directory
```

Multi-session debugging:

```bash
webctl target "Admin Panel"
webctl screenshot --output ./admin-screenshot.png

webctl target "User Dashboard"
webctl screenshot --output ./dashboard-screenshot.png
# Capture different tabs explicitly
```

## Implementation Notes

CLI implementation:

- Parse flags with Cobra
- Connect to daemon via IPC
- Send request: `{"cmd": "screenshot", "fullPage": false}`
- Receive response with PNG data
- Generate filename using timestamp and normalized title
- Create /tmp/webctl-screenshots/ directory if not exists
- Write PNG data to file
- Return JSON response with file path

Daemon implementation:

- Receive screenshot request via IPC
- Verify active session exists (return error if not)
- Use active session ID for CDP command routing
- Call Page.captureScreenshot via CDP with session ID
- For full-page: capture entire scrollable content
- For viewport: capture current visible area only
- Return PNG binary data to CLI

Title normalization:

- Get page title from active session (already tracked)
- Apply normalization algorithm as specified
- Handle edge cases (empty title, all non-alphanumeric)
- Fallback to "untitled" if normalized title is empty

Filename generation:

- Format: YY-MM-DD-HHMMSS-{normalized-title}.png
- Use local time for timestamp (user's timezone)
- Ensure /tmp/webctl-screenshots/ exists before writing
- No collision handling needed (timestamp provides uniqueness)

Custom output path:

- When --output specified, skip temp directory logic
- Use provided path directly
- Create parent directories if needed
- Validate path is writable (return error if not)

Error cases:

Daemon not running:

```json
{"ok": false, "error": "daemon not running. Start with: webctl start"}
```

No active session:

```json
{
  "ok": false,
  "error": "no active session - use 'webctl target <id>' to select",
  "sessions": [...]
}
```

CDP failure:

```json
{"ok": false, "error": "failed to capture screenshot: connection timeout"}
```

File write failure:

```json
{"ok": false, "error": "failed to write screenshot: permission denied"}
```

## CDP Methods

Page.captureScreenshot:

Used for both viewport and full-page screenshots.

Viewport screenshot:

```json
{
  "method": "Page.captureScreenshot",
  "params": {
    "format": "png"
  },
  "sessionId": "9A3E8D71..."
}
```

Full-page screenshot:

Implementation requires capturing entire scrollable content. Typical approach:

1. Query document dimensions (DOM.getDocument)
2. Get current viewport size
3. Set viewport to document dimensions (Emulation.setDeviceMetricsOverride)
4. Capture screenshot (Page.captureScreenshot)
5. Restore original viewport size

Alternative approach: Use Page.captureScreenshot with captureBeyondViewport parameter if browser supports it.

Note: Implementation details delegated to code. DR documents the decision to support full-page capture, not the exact CDP sequence.

## Testing Strategy

Unit tests:

- Title normalization algorithm (various inputs, edge cases)
- Filename generation (timestamp format, title integration)
- Flag parsing (--full-page, --output)
- Error message formatting

Integration tests:

- Start daemon, navigate to page, capture screenshot
- Verify PNG file created in /tmp/webctl-screenshots/
- Verify filename matches pattern
- Verify file is valid PNG (can be opened)
- Test --full-page captures more than viewport
- Test --output creates file at custom path
- Test error when daemon not running
- Test error when no active session
- Verify screenshot uses active session (multi-tab scenario)

## Session Context

Screenshot command operates on the active session. When multiple browser tabs are open, the screenshot captures the currently active tab.

Session selection via `webctl target` command allows choosing which tab to screenshot. See DR-010 for session management details.

Screenshot command does not support --clear flag as it is an observation command (read-only operation). See DR-006 for --clear flag scope.

## Future Enhancements

Potential future additions (deferred from initial implementation):

Selector-based screenshots:

```bash
webctl screenshot --selector ".main-content"
```

Capture specific elements only. Requires element selection and bounding box calculation. Deferred until use cases emerge.

JPEG format support:

```bash
webctl screenshot --format jpeg --quality 80
```

Smaller file sizes for bandwidth-constrained scenarios. Deferred as PNG is sufficient for debugging.

Viewport size control:

```bash
webctl screenshot --viewport 1920x1080
```

Capture at specific viewport dimensions. Deferred as browser window size is appropriate default.

## Updates

- 2025-12-17: Initial version
