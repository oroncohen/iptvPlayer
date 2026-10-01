#!/bin/bash
# Builds the frontend + Go backend, then starts IPTV Player and opens it in
# your browser.
set -euo pipefail
cd "$(dirname "$0")"
PORT="${1:-4568}"
GO="$(pwd)/.gotoolchain/go/bin/go"

if [ ! -x "$GO" ]; then
  echo "Go toolchain not found at $GO -- see .gotoolchain/README (or just install Go normally and use 'go' instead)." >&2
  exit 1
fi

if [ -d frontend ]; then
  if [ ! -d frontend/node_modules ]; then
    echo "Installing frontend dependencies (first run)..."
    (cd frontend && npm install)
  fi
  echo "Building frontend..."
  (cd frontend && npm run build)
fi

echo "Building server..."
(cd goserver && "$GO" build -o ../iptvplayer .)

echo "Starting IPTV Player on http://localhost:$PORT ..."
./iptvplayer "$PORT" &
sleep 1
open "http://localhost:$PORT"
wait
