# Output Modes

Commands: html, css, console, network, cookies

## Default: stdout

```
webctl <command>
```

## Save Mode

```
webctl <command> save
webctl <command> save ./file.ext
webctl <command> save ./dir/
```

## Path Conventions

```
./dir/       Trailing slash = directory (auto-generates filename)
./file.ext   No slash = exact file path
(none)       No path = temp directory (/tmp/webctl-<type>/)
```

## Output Types

```
webctl --json <command>
webctl <command> --raw
```

## Screenshot

Binary output, always saves to file:

```
webctl screenshot save
webctl screenshot save ./page.png
webctl screenshot save ./output/
```
