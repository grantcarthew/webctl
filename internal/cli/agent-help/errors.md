# Common Errors

Error messages and solutions.

## Daemon Errors

```
Error: daemon not running. Start with: webctl start
Solution: webctl start
```

```
Error: daemon not running or not responding
Solution: webctl stop && webctl start
```

## Element Errors

```
Error: No elements found
Solution: Check selector syntax
webctl html --select "button"  # Verify element exists
```

```
Error: selector '.missing' matched no elements
Solution: Inspect page HTML to confirm selector
webctl html | grep "class"
```

## Navigation Errors

```
Error: No previous page
Solution: Cannot go back, no history
```

```
Error: No next page
Solution: Cannot go forward, no history
```

```
Error: net::ERR_NAME_NOT_RESOLVED
Solution: Check URL, domain does not exist
```

```
Error: net::ERR_CONNECTION_REFUSED
Solution: Server not responding, check if running
```

```
Error: timeout waiting for page load
Solution: Increase timeout
webctl ready --timeout 120s
```

## Data Errors

```
Error: No matches found
Solution: --find text not present, adjust search term
```

```
Error: No entries in range
Solution: Range exceeds available entries
webctl console --tail 10  # Check entry count first
```

```
Error: No rules found
Solution: No CSS rules match selector
```

```
Error: Property not found
Solution: CSS property does not exist on element
```

## Scroll Errors

```
Error: invalid --to coordinates
Solution: Use x,y format
webctl scroll --to 0,500
```

```
Error: provide a selector, --to x,y, or --by x,y
Solution: Specify scroll mode
webctl scroll "#footer"
webctl scroll --to 0,0
webctl scroll --by 0,100
```

## File Errors

```
Error: failed to create directory
Solution: Check permissions
```

```
Error: directory does not exist
Solution: Create directory first
mkdir -p ./output
webctl html save ./output/
```
