# Observation Commands Detailed Comparison

| Command | Data Size | Structure | Current Output | Filtering | Primary Use Case |
|---------|-----------|-----------|----------------|-----------|------------------|
| **html [selector]** | Large (10KB-1MB) | Document | File only | Selector only | Inspect page structure |
| **css save** | Large (100KB+) | Document | File only | Selector (computed) | Extract stylesheets |
| **css computed** | Medium (2-10KB) | Key-value pairs | Stdout only | N/A | Debug element styles |
| **css get** | Tiny (<100 bytes) | Scalar value | Stdout only | N/A | Script automation |
| **console** | Variable (0-100s) | Records (logs) | Stdout only | --type, --head, --tail, --range | Debug JS errors |
| **network** | Variable (0-100s) | Records (requests) | Stdout only | --type, --method, --status, --url, --mime, etc. | Debug API calls |
| **cookies** | Small (0-50) | Records (cookies) | Stdout only | None | Inspect auth state |
| **find <text>** | Small (matches) | Records (matches) | Stdout only | Built-in (IS the filter) | Search page content |
| **eval <expr>** | Variable | Any JSON value | Stdout only | N/A | Query page state |

## Key Observations

### Inconsistencies
1. **html** - File only, no stdout option
2. **css** - Three different commands with three different outputs
3. **console/network** - Extensive filtering but no text search
4. **cookies** - No filtering at all
5. **find** - Only searches HTML, not CSS/console/network

### Pattern Violations
- HTML is always file, but CSS computed (similar data) is stdout
- Network has extensive filtering, cookies has none
- Find can't be used on console/network/css (where it would be useful)

### Missing Capabilities
- Can't search console logs for error patterns
- Can't search network requests for specific responses
- Can't search cookies by value patterns
- Can't output HTML to stdout for piping
- Can't save console/network to file for later analysis

## Potential Unified Pattern

### Option 1: All Commands Support Both Outputs
```bash
# Default to stdout (small) or file (large)
webctl html                    # → file (large)
webctl console                 # → stdout (small-medium)

# Force stdout with flag
webctl html --stdout           # → stdout

# Force file with flag  
webctl console -o logs.json    # → file

# All support filtering
webctl html --find "login"
webctl console --find "error"
webctl network --find "api/user"
webctl css --find "background"
```

### Option 2: Separate Subcommands for Output Type
```bash
# View commands (stdout)
webctl html view [selector]
webctl console view [--filter]
webctl network view [--filter]

# Save commands (file)
# Note: Use trailing slash for directories (path/) vs files (path)
webctl html save [selector] [path]      # path or path/
webctl console save [path]               # path or path/
webctl network save [path]               # path or path/

# Find commands (stdout, filtered)
webctl html find <text>
webctl console find <text>
webctl network find <text>
```

### Option 3: Smart Defaults + Universal Flags
```bash
# Smart defaults based on size
webctl html              # → file (always large)
webctl console           # → stdout (usually small)
webctl network           # → stdout (usually small)

# Universal override flags (all commands)
--output, -o PATH        # Save to file
--stdout                 # Force stdout (fail if too large?)

# Universal filter flags (all commands)
--find TEXT              # Text search
--regex PATTERN          # Regex search
--limit N                # Limit results
--head N / --tail N      # Range selection
```

## Questions to Answer

1. Should all commands support both stdout and file output?
2. Should filtering be unified across all commands?
3. Should `find` become a universal flag instead of separate command?
4. How do we handle size limits for stdout?
5. Should we maintain backward compatibility?
