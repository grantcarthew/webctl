#!/bin/bash
# Test HTML extraction timing after navigation

# Start daemon in background
./webctl start --debug > /tmp/webctl_test.log 2>&1 &
DAEMON_PID=$!

# Wait for daemon to start
sleep 2

# Navigate
echo "=== Navigate ==="
./webctl navigate https://example.com/ 2>&1 | grep -v DEBUG

# Wait a moment
sleep 1

# Get HTML and show timing
echo "=== HTML ==="
./webctl html 2>&1 | tee /tmp/html_output.txt

# Show debug logs with timestamps
echo ""
echo "=== Debug timing from logs ==="
grep "\[DEBUG\].*html:" /tmp/webctl_test.log | tail -10

# Cleanup
./webctl stop > /dev/null 2>&1
kill $DAEMON_PID 2>/dev/null
