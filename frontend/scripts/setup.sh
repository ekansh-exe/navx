#!/usr/bin/env bash
# Checks frontend dependencies, installs node_modules if missing.
# frontend/.env is optional -- api/client.ts and ws/WebSocketProvider.tsx
# already default to http://localhost:8080 / ws://localhost:8080 with no
# .env file present at all.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FRONTEND_DIR="$(dirname "$SCRIPT_DIR")"
cd "$FRONTEND_DIR"

MISSING=()

echo "== Checking frontend dependencies =="
for cmd in node npm; do
  if command -v "$cmd" >/dev/null 2>&1; then
    echo "  [ok] $cmd ($($cmd --version))"
  else
    echo "  [MISSING] $cmd -- install Node.js from https://nodejs.org/"
    MISSING+=("$cmd")
  fi
done

if [ ${#MISSING[@]} -gt 0 ]; then
  echo
  echo "Install the missing tools above, then re-run this script."
  exit 1
fi

echo
echo "== Dependencies =="
if [ -d node_modules ]; then
  echo "  [ok] node_modules already installed"
else
  echo "  Installing (npm install)..."
  npm install
fi

echo
echo "Frontend is ready. Start it with: ./scripts/start.sh (or 'npm run dev' from frontend/)"
