#!/bin/bash
# Start the test backend server

PORT="${1:-3000}"

cd "$(dirname "$0")"

echo "Starting test backend on port $PORT..."
go run backend.go "$PORT"
