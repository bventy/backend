#!/bin/bash
echo "Stopping any existing server on port 8082..."
fuser -k -9 8082/tcp > /dev/null 2>&1 || true

echo "Starting Bventy Backend..."
go run cmd/api/main.go
