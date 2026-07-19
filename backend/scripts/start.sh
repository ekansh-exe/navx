#!/usr/bin/env bash
# Starts the backend API on PORT (default 8080). Assumes ./scripts/setup.sh
# has already been run at least once (Postgres/Redis reachable, migrations
# applied) -- this script does not re-check any of that.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(dirname "$SCRIPT_DIR")"
cd "$BACKEND_DIR"

if [ ! -f .env ]; then
  echo "backend/.env is missing -- run ./scripts/setup.sh first." >&2
  exit 1
fi

# shellcheck disable=SC1091
set -a; source .env; set +a

echo "Starting backend on :${PORT:-8080} ..."
exec go run ./cmd/server
