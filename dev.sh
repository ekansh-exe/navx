#!/usr/bin/env bash
# Single command to run NavXchange full-stack locally, no Docker.
# Runs both projects' setup scripts (dependency checks, .env creation,
# migrations), then starts backend (:8080) and frontend (:5173) together.
# Ctrl+C stops both cleanly.
set -uo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=== NavXchange: full-stack dev (no Docker) ==="
echo

echo "--- Backend setup ---"
if ! "$ROOT_DIR/backend/scripts/setup.sh"; then
  echo
  echo "Backend setup failed -- fix the issues above before continuing." >&2
  exit 1
fi

echo
echo "--- Frontend setup ---"
if ! "$ROOT_DIR/frontend/scripts/setup.sh"; then
  echo
  echo "Frontend setup failed -- fix the issues above before continuing." >&2
  exit 1
fi

PIDS=()
cleanup() {
  echo
  echo "Shutting down..."
  for pid in "${PIDS[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null
}
trap cleanup INT TERM EXIT

echo
echo "--- Starting backend (:8080) ---"
"$ROOT_DIR/backend/scripts/start.sh" &
PIDS+=($!)

echo "Waiting for backend health check..."
BACKEND_UP=0
for _ in $(seq 1 30); do
  if curl -sf http://localhost:8080/health >/dev/null 2>&1; then
    BACKEND_UP=1
    break
  fi
  sleep 1
done

if [ "$BACKEND_UP" -eq 0 ]; then
  echo "Backend never became healthy on :8080 -- check the output above." >&2
  exit 1
fi
echo "Backend is up."

echo
echo "--- Starting frontend (:5173) ---"
"$ROOT_DIR/frontend/scripts/start.sh" &
PIDS+=($!)

echo
echo "Backend:  http://localhost:8080"
echo "Frontend: http://localhost:5173"
echo "Press Ctrl+C to stop both."
echo

wait
