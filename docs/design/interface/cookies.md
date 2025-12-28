# Cookies Command Design - LOCKED

## Universal Pattern (Observation)

```bash
# Default: save all cookies to temp file
webctl cookies
# → /tmp/webctl-cookies/25-12-28-HHMMSS-cookies.json

# Show: output all cookies to stdout
webctl cookies show

# Save: save all cookies to custom path
webctl cookies save <path>
# If <path> is a directory, auto-generate filename
webctl cookies save ./output/
# → ./output/25-12-28-HHMMSS-cookies.json
```

## Universal Flags

```bash
--find, -f TEXT          # Search within cookie names and values
--raw                    # Skip formatting/pretty-printing
--json                   # JSON output (global flag)
```

## Cookies-Specific Flags

These filters are specific to cookies:

```bash
--domain DOMAIN          # Filter by domain
--name NAME              # Filter by name (exact match)
```

## Cookies-Specific Subcommands (Mutations)

These are mutations, separate from observation:

```bash
# Set a cookie
webctl cookies set <name> <value>
  --domain DOMAIN        # Cookie domain (defaults to current page domain)
  --path PATH            # Cookie path (defaults to "/")
  --secure               # Require HTTPS connection
  --httponly             # Prevent JavaScript access (document.cookie)
  --max-age SECONDS      # Expiry in seconds from now (0 = session cookie)
  --samesite POLICY      # SameSite policy: Strict, Lax, or None

# Delete a cookie
webctl cookies delete <name>
  --domain DOMAIN        # Cookie domain (required if multiple cookies match)
```

## Examples

### Observation

```bash
# All cookies to temp
webctl cookies
# → /tmp/webctl-cookies/25-12-28-HHMMSS-cookies.json

# All cookies to stdout
webctl cookies show

# Filter by domain
webctl cookies show --domain ".github.com"
webctl cookies show --domain "example.com"

# Search cookies
webctl cookies show --find "session"
webctl cookies --find "token"
# → /tmp/webctl-cookies/... (filtered cookies)

# Save filtered cookies
webctl cookies save ./auth-cookies.json --find "auth"
webctl cookies save ./github-cookies.json --domain ".github.com"

# Combine filters
webctl cookies show --domain "example.com" --find "session"
```

### Mutations

```bash
# Set session cookie
webctl cookies set session abc123

# Set persistent cookie (1 hour)
webctl cookies set remember_me yes --max-age 3600

# Set secure cookie
webctl cookies set auth_token xyz789 --secure --httponly

# Set cookie with domain
webctl cookies set tracking id123 --domain .example.com

# Set cookie with SameSite
webctl cookies set csrf_token xyz --samesite Strict

# Delete cookie
webctl cookies delete session

# Delete cookie with domain (if ambiguous)
webctl cookies delete session --domain api.example.com
```

## Output Format

**Text mode:**
- Formatted table with name, value, domain, path, expiry, flags

**JSON mode:**
- Array of cookie objects
- Each cookie includes: name, value, domain, path, expires, httpOnly,
  secure, sameSite, size, session

## Design Rationale

**Universal pattern for observation:**
- Consistent with html, css, console, network
- Default saves to temp (preserves cookies for analysis)
- `show` for interactive debugging
- `save <path>` for archival or transfer

**Cookies-specific filters:**
- `--domain` - Essential for multi-domain cookie filtering
- `--name` - Exact name matching
- `--find` - Fuzzy search across names and values

**Mutation subcommands:**
- `set` and `delete` are mutations, not observations
- Keep as separate subcommands (established pattern)
- Don't mix with observation pattern (save/show)
- Each has specific flags for cookie attributes

**Separation of concerns:**
- Observation (default/show/save) - read-only operations
- Mutations (set/delete) - write operations
- Clear distinction prevents confusion
