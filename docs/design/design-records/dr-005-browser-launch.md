# DR-005: Browser Launch Configuration

- Date: 2025-12-12
- Status: Accepted
- Category: Browser

## Problem

Launching Chrome for CDP automation requires careful selection of command-line flags. Too few flags results in unwanted dialogs and prompts. Too many flags masks real-world browser behavior that users may need to debug.

Additionally, users need control over the browser profile directory for different use cases:
- Isolated testing (temp directory)
- Debugging with user's extensions/settings (default profile)
- Custom profile directory

## Decision

### Supported Platforms

macOS and Linux only. Windows is not supported.

### Chrome Launch Flags

Use a minimal set of flags focused on avoiding dialogs while preserving real browser behavior:

```
--remote-debugging-port=PORT     # Required for CDP
--no-first-run                   # Skip first-run dialogs
--no-default-browser-check       # Skip "set as default" prompt
--disable-background-networking  # Reduce network noise
--disable-sync                   # No Google account sync
--disable-popup-blocking         # Allow popups for debugging
--headless                       # When headless mode requested
about:blank                      # Start page
```

Platform-specific flags:
- macOS: `--use-mock-keychain` (avoid Keychain permission dialogs)
- Linux: `--password-store=basic` (avoid Gnome Keyring/KWallet dialogs)

### User Data Directory

Three modes via `--user-data-dir` option:

| Value | Behavior |
|-------|----------|
| Empty (default) | Create temporary directory, cleaned up on close |
| `default` | Use user's Chrome profile (no flag sent to Chrome) |
| Any path | Use specified directory |

### Headless Mode

Use `--headless` without the `=new` suffix:
- Chrome 112-131: Uses legacy headless mode
- Chrome 132+: Uses new headless mode automatically

The `--headless=new` flag is no longer needed as Chrome 132 removed the old headless mode.

### Startup Timeout

30 seconds to wait for CDP endpoint to respond. This accommodates:
- Slow machines
- First-time profile initialization
- Extension loading

### Graceful Shutdown

Send `SIGINT` first, allowing Chrome to save state. Fall back to `SIGKILL` if the process doesn't exit.

## Why

Minimal flags:
- Users may need to debug popup behavior, audio, or other features
- Disabling too much masks real-world behavior
- Each flag is justified by avoiding a specific dialog/prompt

Platform-specific flags:
- macOS Keychain dialogs block headless execution
- Linux keyring dialogs block headless execution
- These don't affect debugging behavior

User data directory options:
- Temp dir (default) provides isolation between sessions
- "default" enables debugging with user's extensions/settings
- Custom path supports reproducible test scenarios

30-second timeout:
- 10 seconds proved too short for some environments
- 30 seconds is generous without being excessive
- Provides clear error on actual failures

## Flags Not Used

| Flag | Reason Not Used |
|------|-----------------|
| `--enable-automation` | Shows "Chrome is being controlled" banner, affects screenshots |
| `--mute-audio` | User may need to debug audio issues |
| `--disable-extensions` | User may need extensions for debugging |
| `--metrics-recording-only` | Not necessary, already disabled sync |
| `--safebrowsing-disable-auto-update` | Flag was removed in Nov 2017 |
| `--disable-translate` | Now use `--disable-features=Translate` if needed |
| Throttling flags | Would mask real-world tab behavior |

## Comparison with Rod

Rod uses many more flags for automation reliability. webctl uses fewer because:
- webctl is for debugging real pages, not automation
- Users need to see real browser behavior
- Fewer flags means less deviation from production

## Trade-offs

Accept:
- Some background network activity (safe browsing, etc.)
- Potential for some Chrome prompts in edge cases

Gain:
- Browser behavior matches production
- Extensions work when using default profile
- Audio, popups, and other features available for debugging

## Alternatives

Use Rod's full flag set:
- Pro: Maximum automation reliability
- Con: Hides real browser behavior
- Rejected: webctl is for debugging, not automation

No platform-specific flags:
- Pro: Simpler code
- Con: Blocking dialogs on macOS/Linux
- Rejected: Dialogs break headless mode

Configurable timeout:
- Pro: Flexibility
- Con: Additional option to explain
- Rejected: 30 seconds works for all reasonable cases

Windows support:
- Pro: Larger user base
- Con: Different process management, path handling, no SIGINT
- Con: Additional testing burden
- Rejected: Focus on Unix-like systems where webctl is most likely to be used
