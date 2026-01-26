# Local Development Server

Start development server with hot reload.

## Auto-Start

```
webctl serve
```

Automatically starts daemon and browser if not running.

## Static Mode

Serve files from directory:

```
webctl serve
webctl serve ./public
webctl serve ./dist
webctl serve . --port 3000
webctl serve ./public --host 0.0.0.0
```

## Proxy Mode

Proxy to backend server:

```
webctl serve --proxy localhost:8080
webctl serve --proxy http://localhost:3000
webctl serve --proxy http://api.local:8080 --port 3001
```

## Watch Paths

```
webctl serve ./public --watch src/,assets/
webctl serve ./dist --ignore "*.tmp,*.log"
```

## Common Patterns

Local development:

```
webctl serve ./dist
webctl console --type error
webctl network --failed
```

Backend integration:

```
webctl serve --proxy localhost:3000
webctl ready --network-idle
webctl html --select "#app"
```

Stop server:

```
Ctrl+C
webctl stop
```
