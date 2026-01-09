# webctl Test Data

This directory contains test resources for webctl development and testing.

## Files

### Frontend Test Page

**index.html** - Comprehensive test page for browser automation testing

Features:
- 🎨 Modern responsive design with gradient background
- ⏰ Live clock that updates every second
- 📊 Page load counter (tests hot reload functionality)
- 🎯 Console test buttons (log, warn, error, info)
- 🌐 Network request test button (AJAX to JSONPlaceholder API)
- 📺 Console output mirror showing what's logged
- 🏷️ Feature showcase badges

Usage:
```bash
# Serve the test page
webctl serve testdata

# Or from within testdata
cd testdata
webctl serve
```

The page will automatically load in the browser and you can:
- Click console buttons to test `webctl console` command
- Click network button to test `webctl network` command
- Edit the HTML to test hot reload
- Use `webctl html`, `webctl css` commands on it

### Backend Test Server

**backend.go** - Simple HTTP server for proxy testing

**start-backend.sh** - Helper script to start the backend

Features:
- Runs on port 3000 by default (configurable)
- Multiple test endpoints with different responses
- CORS enabled for cross-origin testing
- JSON responses for all endpoints

Endpoints:
```
GET  /                - Server info (version, time, path)
GET  /api/hello       - Hello message
GET  /api/users       - User list (3 mock users)
GET  /api/echo        - Echo request details (method, headers, query)
GET  /status/200      - 200 OK response
GET  /status/400      - 400 Bad Request
GET  /status/404      - 404 Not Found
GET  /status/500      - 500 Internal Server Error
GET  /delay           - Delayed response (2 seconds)
```

Usage:
```bash
# Start backend on default port 3000
cd testdata
./start-backend.sh

# Start on custom port
./start-backend.sh 8080

# Or run directly
go run backend.go 3000
```

Then test proxy mode:
```bash
# In another terminal
webctl serve --proxy localhost:3000

# The browser will proxy requests to the backend
# Visit http://localhost:XXXX/api/hello in browser
```

## Testing Workflows

### Test Hot Reload
1. Start serve: `cd testdata && webctl serve`
2. Edit index.html (change title or content)
3. Watch browser auto-reload

### Test Console Capture
1. Start serve: `cd testdata && webctl serve`
2. Click console test buttons on the page
3. Run: `webctl console`
4. Verify captured logs

### Test Network Monitoring
1. Start serve: `cd testdata && webctl serve`
2. Click "Network Request" button on the page
3. Run: `webctl network`
4. Verify captured API request

### Test Proxy Mode
1. Start backend: `cd testdata && ./start-backend.sh`
2. Start proxy: `webctl serve --proxy localhost:3000`
3. Visit http://localhost:XXXX/api/hello in browser
4. Verify backend response shows through proxy

## Integration with Test Scripts

These resources are used by test scripts:
- `scripts/interactive/test-serve.sh` - Uses both frontend and backend
- `scripts/interactive/test-console.sh` - Uses frontend console buttons
- `scripts/interactive/test-network.sh` - Uses frontend network button

All test scripts reference these files using:
```bash
$(git rev-parse --show-toplevel)/testdata
```

This ensures they work from any directory in the repository.
