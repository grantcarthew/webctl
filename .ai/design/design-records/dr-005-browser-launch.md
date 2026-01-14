# DR-005: Browser Launch Configuration

- Date: 2025-12-12
- Updated: 2025-12-23
- Status: Accepted
- Category: Browser

## Problem

Launching Chrome for CDP automation requires careful selection of command-line flags. Too few flags results in unwanted dialogs, throttled processes, and unreliable CDP communication. Too many flags masks real-world browser behaviour that users may need to debug.

Additionally, users need control over the browser profile directory for different use cases:

- Isolated testing (temp directory)
- Debugging with user's extensions/settings (default profile)
- Custom profile directory

## Decision

### Supported Platforms

macOS and Linux only. Windows is not supported.

### Chrome Launch Flags

Flags are grouped by purpose. Each flag has been individually reviewed for necessity.

#### Required for CDP

```
--remote-debugging-port=PORT     # Required for CDP connection
```

#### Prevent Dialogs

```
--no-first-run                   # Skip first-run wizards
--no-default-browser-check       # Skip "set as default" prompt
--disable-prompt-on-repost       # Skip POST resubmission dialog
```

#### Prevent Throttling (Critical for CDP)

These flags are essential for reliable CDP operation. Without them, Chrome throttles background/occluded tabs at the OS level, causing CDP calls like `DOM.getDocument` to timeout.

```
--disable-background-timer-throttling      # Don't throttle JS timers in background tabs
--disable-backgrounding-occluded-windows   # Don't treat occluded window as background
--disable-renderer-backgrounding           # Don't lower process priority for background tabs
--disable-hang-monitor                     # Don't show "page unresponsive" dialogs
--disable-ipc-flooding-protection          # Don't rate-limit IPC (CDP sends many messages)
```

#### Reduce Background Noise

```
--disable-background-networking  # Disable update checks, safe browsing, etc.
--disable-sync                   # No Google account sync
--disable-breakpad               # No crash dump collection
--disable-client-side-phishing-detection  # No phishing scans
```

#### Container/CI Compatibility

```
--disable-dev-shm-usage          # Use regular memory instead of /dev/shm
```

#### Screenshots

```
--force-color-profile=srgb       # Consistent colours across displays
```

#### Conditional Flags

```
--disable-popup-blocking         # Only in headed mode (popups invisible in headless)
--headless                       # When headless mode requested
--user-data-dir=PATH             # When custom profile specified
```

#### Platform-Specific

```
# macOS
--use-mock-keychain              # Avoid Keychain permission dialogs

# Linux
--password-store=basic           # Avoid Gnome Keyring/KWallet dialogs
```

#### Start Page

```
about:blank                      # Clean starting state
```

### User Data Directory

Three modes via `--user-data-dir` option:

| Value | Behaviour |
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
- First-time profile initialisation
- Extension loading

### Graceful Shutdown

Send `SIGINT` first, allowing Chrome to save state. Fall back to `SIGKILL` if the process doesn't exit.

## Why

### Throttling Flags

The `--disable-background-*` and `--disable-renderer-backgrounding` flags were initially rejected as "automation-only" features. Testing revealed they are essential for CDP reliability:

1. Chrome throttles JS timers in background/occluded tabs
2. Chrome lowers process priority for non-foreground tabs
3. CDP calls to throttled tabs timeout (10+ seconds for `DOM.getDocument`)
4. These flags don't affect page behaviour, only Chrome's process management

### Minimal Other Flags

- Users may need to debug popup behaviour (conditional flag)
- Users may need to debug translation issues (no translate flag)
- Users may need built-in extensions like PDF viewer (no disable-component-extensions)
- Each remaining flag is justified by avoiding a specific dialog/prompt or enabling CDP

### Platform-Specific Flags

- macOS Keychain dialogs block headless execution
- Linux keyring dialogs block headless execution
- These don't affect debugging behaviour

### User Data Directory Options

- Temp dir (default) provides isolation between sessions
- "default" enables debugging with user's extensions/settings
- Custom path supports reproducible test scenarios

### 30-Second Timeout

- 10 seconds proved too short for some environments
- 30 seconds is generous without being excessive
- Provides clear error on actual failures

## Flags Not Used

| Flag | Reason Not Used |
|------|-----------------|
| `--enable-automation` | Shows "Chrome is being controlled" banner, affects screenshots |
| `--mute-audio` | User may need to debug audio issues |
| `--disable-extensions` | User may need extensions for debugging |
| `--disable-features=TranslateUI` | User may need to debug translation issues |
| `--disable-component-extensions-with-background-pages` | User may need PDF viewer, etc. |
| `--disable-default-apps` | Minimal impact, let Chrome behave naturally |

## Trade-offs

Accept:

- Some background network activity (safe browsing, etc.) - mitigated by --disable-background-networking
- Translation popups may appear on foreign-language pages
- Default Chrome apps installed

Gain:

- Browser behaviour matches production
- Extensions work when using default profile
- Audio, popups, translation, and other features available for debugging
- Reliable CDP communication via anti-throttling flags

## References

- [Chrome Flags for Tooling](https://github.com/GoogleChrome/chrome-launcher/blob/main/docs/chrome-flags-for-tools.md) - Google's official documentation
- BUG-003 investigation (P-007) - Identified throttling as cause of CDP timeouts
