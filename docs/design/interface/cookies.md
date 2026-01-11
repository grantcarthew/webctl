# Cookies Command Design - LOCKED

## Universal Pattern (Observation)

```bash
# Default: output all cookies to stdout
webctl cookies

# Save: save all cookies to file
webctl cookies save           # Save to temp file
webctl cookies save <path>    # Save to custom path

# Path conventions (trailing slash required for directories):
webctl cookies save ./cookies.json   # File: saves to ./cookies.json
webctl cookies save ./output/        # Directory: auto-generates filename
# → ./output/25-12-28-HHMMSS-cookies.json
webctl cookies save ./output         # File: saves to ./output (not a directory!)
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
# All cookies to stdout
webctl cookies

# All cookies to temp file
webctl cookies save
# → /tmp/webctl-cookies/25-12-28-HHMMSS-cookies.json

# Filter by domain
webctl cookies --domain ".github.com"
webctl cookies --domain "example.com"

# Search cookies
webctl cookies --find "session"
webctl cookies save --find "token"
# → /tmp/webctl-cookies/... (filtered cookies)

# Save filtered cookies
webctl cookies save ./auth-cookies.json --find "auth"
webctl cookies save ./github-cookies.json --domain ".github.com"

# Combine filters
webctl cookies --domain "example.com" --find "session"
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
- Default outputs to stdout (Unix convention)
- `save` for file output (temp or custom path)

**Cookies-specific filters:**
- `--domain` - Essential for multi-domain cookie filtering
- `--name` - Exact name matching
- `--find` - Fuzzy search across names and values

**Mutation subcommands:**
- `set` and `delete` are mutations, not observations
- Keep as separate subcommands (established pattern)
- Don't mix with observation pattern (default/save)
- Each has specific flags for cookie attributes

**Separation of concerns:**
- Observation (default/save) - read-only operations
- Mutations (set/delete) - write operations
- Clear distinction prevents confusion
