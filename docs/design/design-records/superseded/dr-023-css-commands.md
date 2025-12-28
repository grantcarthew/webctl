# DR-023: CSS Command Architecture

- Date: 2025-12-26
- Status: Superseded by [DR-026](../dr-026-css-command-interface.md)
- Category: CLI

## Problem

Developers debugging web applications need access to CSS information beyond what the html command provides. Current limitations:

- Cannot extract computed styles for elements
- Cannot get specific CSS property values for scripting
- Cannot extract all stylesheets from a page
- Cannot inject custom CSS for testing or modifications
- Must use browser DevTools for style inspection

CSS debugging is essential for:
- Understanding why elements render incorrectly
- Verifying responsive design styles
- Debugging specificity conflicts
- Testing style changes
- Scraping site styling

## Decision

Implement a css command with four subcommands that provide comprehensive CSS access:

Command Structure:

```bash
# Extract CSS to file (large output)
webctl css save [selector] [--output PATH] [--raw]

# Get computed styles to stdout (medium output)
webctl css computed <selector> [--json]

# Get single property to stdout (quick check)
webctl css get <selector> <property>

# Inject CSS into page (modification)
webctl css inject <css> [--file PATH]
```

Subcommands:

1. save - Extract and save CSS to file
2. computed - Get all computed styles for element
3. get - Get single CSS property value
4. inject - Add CSS to page

## Why

Subcommand Structure:

Clear separation of concerns based on output size and use case:
- save: Large data (all stylesheets) needs files
- computed: Medium data (all properties) suitable for stdout
- get: Small data (single property) perfect for scripting
- inject: Modification operation, distinct from extraction

Follows established patterns:
- cookies has subcommands (set, delete)
- Extensible for future CSS operations

File Output for Large Data:

Following html command pattern:
- CSS files can be large (hundreds of KB)
- File output allows incremental reading
- Temp directory with consistent naming
- --output flag for custom paths

Stdout for Quick Checks:

Computed styles and single properties to stdout:
- Fast feedback for debugging
- Scriptable (pipe to other tools)
- No file management needed
- JSON format available for parsing

CSS Formatting:

Pretty-print by default (like html command):
- Readable output for debugging
- Proper indentation and line breaks
- --raw flag for unformatted/minified

CSS Injection:

Enables testing and modifications:
- Hide elements for screenshots
- Test responsive breakpoints
- Override vendor styles
- Visual debugging

## Trade-offs

Accept:

- Four subcommands (more complex than single command)
- Inconsistent output (files vs stdout) based on subcommand
- CSS formatting requires parsing (additional complexity)
- Cross-origin stylesheets may be blocked by CORS
- Computed styles only (cannot get cascade/specificity info)
- Injection is temporary (page reload removes)

Gain:

- Clear command intent (save vs get vs inject)
- Efficient output handling (large to file, small to stdout)
- Scriptable property checks
- Complete CSS debugging workflow
- Consistent with html command patterns
- Room for future expansion

## Alternatives

Single Command with Flags:

```bash
webctl css [selector]                    # Save to file
webctl css [selector] --stdout           # Output to stdout
webctl css [selector] --property color   # Get property
webctl css --inject "..."                # Inject CSS
```

- Pro: Simpler command structure
- Pro: Fewer commands to learn
- Con: Flag overload for different operations
- Con: Unclear intent (save vs get vs inject)
- Con: Hard to extend with new operations
- Rejected: Subcommands are clearer

Mirror HTML Command Exactly:

```bash
webctl css [selector]                    # Always save to file
```

- Pro: Perfectly consistent with html
- Pro: Simple single command
- Con: Getting single property requires file I/O
- Con: No way to inject CSS
- Con: Computed styles need separate command anyway
- Rejected: Too limited for CSS use cases

Separate Top-Level Commands:

```bash
webctl css-save [selector]
webctl css-computed <selector>
webctl css-get <selector> <property>
webctl css-inject <css>
```

- Pro: Very explicit
- Pro: No subcommand complexity
- Con: Clutters top-level namespace
- Con: Four new commands vs one
- Con: Less discoverable (no grouping)
- Rejected: Subcommands group related functionality better

Property-First Syntax:

```bash
webctl css background-color "#header"
```

- Pro: Shorter for single properties
- Con: Awkward for selectors with spaces
- Con: Cannot distinguish property from selector
- Con: No clear save/inject operations
- Rejected: Too ambiguous

## Structure

Subcommand Details:

save - Extract CSS to file:
- No selector: All stylesheets (inline + style tags + linked)
- With selector: Computed styles for matched element
- Default path: /tmp/webctl-css/YY-MM-DD-HHMMSS-{title}.css
- Formatted by default, --raw for unformatted
- --output flag for custom path

computed - Get computed styles to stdout:
- Requires selector argument
- Returns all CSS properties for element
- Text format: property: value (one per line)
- JSON format: {"property": "value", ...}
- Useful for debugging specific elements

get - Get single property to stdout:
- Requires selector and property arguments
- Returns computed value for that property
- Plain text output (just the value)
- Scriptable for automation
- Fast property checks

inject - Inject CSS into page:
- Inline: `webctl css inject "body { background: red; }"`
- File: `webctl css inject --file ./custom.css`
- Adds style tag to document head
- Temporary (removed on page reload)
- Returns success confirmation

## Implementation Details

CSS Extraction Methods:

All Stylesheets (save without selector):
```javascript
// Via CDP Runtime.evaluate
Array.from(document.styleSheets).map(sheet => {
  try {
    return Array.from(sheet.cssRules).map(rule => rule.cssText).join('\n');
  } catch (e) {
    return '/* Cross-origin stylesheet - cannot access */';
  }
}).join('\n\n');
```

Computed Styles (save with selector, computed subcommand):
```javascript
// Via CDP Runtime.evaluate
const element = document.querySelector(selector);
const styles = window.getComputedStyle(element);
const result = {};
for (let i = 0; i < styles.length; i++) {
  const prop = styles[i];
  result[prop] = styles.getPropertyValue(prop);
}
return result;
```

Single Property (get subcommand):
```javascript
// Via CDP Runtime.evaluate
const element = document.querySelector(selector);
window.getComputedStyle(element).getPropertyValue(property);
```

CSS Injection:
```go
// Via CDP Page.addStyleTag
params := map[string]any{
  "content": cssContent,
}
// or
params := map[string]any{
  "url": "file://" + filepath.Abs(cssFile),
}
```

## File Naming

Following html command pattern:

```
/tmp/webctl-css/
  25-12-26-143052-tesla-com.css           # All CSS
  25-12-26-143115-header.css              # Computed styles for #header
  25-12-26-143120-button.css              # Computed styles for .button
```

Format: `YY-MM-DD-HHMMSS-{identifier}.css`

Identifier:
- For all CSS: sanitized page title or domain
- For selector: sanitized selector (remove special chars)

## CSS Formatting

Parse CSS into structured format:

```css
/* Input (minified) */
body{margin:0;padding:0;background:#fff}.header{display:flex}

/* Output (formatted) */
body {
  margin: 0;
  padding: 0;
  background: #fff;
}

.header {
  display: flex;
}
```

Formatting rules:
- One rule per block
- Opening brace on same line as selector
- One property per line
- 2-space indentation
- Closing brace on new line
- Blank line between rules

Computed styles formatting:

```css
/* Text format */
display: flex
background-color: rgb(255, 255, 255)
width: 1200px
margin: 0px

/* JSON format */
{
  "display": "flex",
  "background-color": "rgb(255, 255, 255)",
  "width": "1200px",
  "margin": "0px"
}
```

## Output Examples

save - All CSS:
```bash
$ webctl css save
/tmp/webctl-css/25-12-26-143052-tesla-com.css

$ cat /tmp/webctl-css/25-12-26-143052-tesla-com.css
body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto;
  background: #000;
  color: #fff;
}

.header {
  display: flex;
  justify-content: space-between;
  padding: 20px;
}
...
```

save - Computed styles for selector:
```bash
$ webctl css save "#header" --output ./header-styles.css
./header-styles.css

$ cat ./header-styles.css
display: flex;
justify-content: space-between;
align-items: center;
width: 1200px;
height: 80px;
background-color: rgb(0, 0, 0);
...
```

computed - All properties:
```bash
$ webctl css computed ".button"
display: inline-block
padding: 10px 20px
background-color: rgb(0, 113, 227)
color: rgb(255, 255, 255)
border-radius: 4px
...

$ webctl css computed ".button" --json
{
  "ok": true,
  "styles": {
    "display": "inline-block",
    "padding": "10px 20px",
    "background-color": "rgb(0, 113, 227)",
    "color": "rgb(255, 255, 255)",
    "border-radius": "4px",
    ...
  }
}
```

get - Single property:
```bash
$ webctl css get "#header" background-color
rgb(0, 0, 0)

$ webctl css get ".button" display
inline-block

# Scriptable
$ if [ "$(webctl css get '.modal' display)" = "none" ]; then
    echo "Modal is hidden"
  fi
```

inject - Add CSS:
```bash
$ webctl css inject "body { background: red !important; }"
OK

$ webctl css inject --file ./dark-mode.css
OK

# Useful for testing
$ webctl css inject ".ads { display: none !important; }"
$ webctl screenshot  # Screenshot without ads
```

## Error Handling

Invalid selector:
```
Error: selector '.missing' matched no elements
```

Invalid property:
```
Error: property 'invalid-prop' does not exist
```

Cross-origin stylesheet:
```
/* Stylesheet from https://cdn.example.com - blocked by CORS */
```

Write permission error:
```
Error: failed to write CSS: permission denied: /path/to/file.css
```

## Integration with Existing Commands

Workflow examples:

Debug element styling:
```bash
webctl navigate example.com
webctl html "#header" --output ./header.html    # Get structure
webctl css computed "#header" --output ./header.css  # Get styles
```

Test CSS changes:
```bash
webctl css inject ".button { background: green; }"
webctl screenshot --output ./test.png
```

Scrape site styles:
```bash
webctl css save --output ./site-styles.css
webctl html --output ./site-structure.html
```

Script element visibility:
```bash
if [ "$(webctl css get '.modal' display)" = "none" ]; then
  echo "Modal hidden, clicking trigger..."
  webctl click "#show-modal"
fi
```

## Updates

None yet.
