#!/usr/bin/env bash
# Checks backend dependencies, creates backend/.env if missing, creates the
# navx Postgres role/db if reachable-but-absent, and runs migrations.
# Does NOT start or manage Postgres/Redis themselves — this repo assumes a
# native (non-Docker) Postgres and Redis are already running; see the
# printed instructions below if they aren't.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(dirname "$SCRIPT_DIR")"
cd "$BACKEND_DIR"

MISSING=()
FAILED=0

echo "== Checking backend dependencies =="

check_cmd() {
  local name="$1" hint="$2"
  if command -v "$name" >/dev/null 2>&1; then
    echo "  [ok] $name"
  else
    echo "  [MISSING] $name -- $hint"
    MISSING+=("$name")
  fi
}

check_cmd go "install from https://go.dev/dl/ (this repo uses go.mod's declared version)"
check_cmd psql "Postgres client -- install the postgresql (client) package for your OS"
check_cmd migrate "golang-migrate CLI -- go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"

if [ ${#MISSING[@]} -gt 0 ]; then
  echo
  echo "Install the missing tools above, then re-run this script."
  exit 1
fi

echo
echo "== .env =="
if [ -f .env ]; then
  echo "  [ok] backend/.env already exists (leaving it as-is)"
else
  cp .env.example .env
  echo "  [created] backend/.env from .env.example -- edit JWT_SECRET before anything but local dev"
fi

# shellcheck disable=SC1091
set -a; source .env; set +a

echo
echo "== Postgres =="
if psql "$DATABASE_URL" -c '\q' >/dev/null 2>&1; then
  echo "  [ok] reachable at DATABASE_URL, and the navx role/db already exist"
else
  echo "  Not connectable with the configured DATABASE_URL yet. Trying to create the role/db..."
  # Best-effort: works if the current OS user already has sufficient
  # privilege on this Postgres instance (e.g. the initdb superuser).
  # Falls through to the manual instructions below if not.
  if createuser -h localhost -p 5432 navx 2>/dev/null; then
    echo "  [created] role 'navx'"
    psql -h localhost -p 5432 -d postgres -c "ALTER ROLE navx WITH PASSWORD 'navx' LOGIN;" >/dev/null 2>&1
  fi
  if createdb -h localhost -p 5432 -O navx navx 2>/dev/null; then
    echo "  [created] database 'navx'"
  fi
  if psql "$DATABASE_URL" -c '\q' >/dev/null 2>&1; then
    echo "  [ok] navx role/db now reachable"
  else
    echo "  [FAIL] still can't connect. This machine needs a running native Postgres first:"
    echo "    initdb -D ~/pgdata && pg_ctl -D ~/pgdata -l ~/pgdata/log start"
    echo "    createuser -s navx && createdb -O navx navx"
    echo "    psql -d navx -c \"ALTER ROLE navx WITH PASSWORD 'navx';\""
    echo "  (or point DATABASE_URL in backend/.env at whatever Postgres you do have)"
    FAILED=1
  fi
fi

echo
echo "== Redis =="
if command -v redis-cli >/dev/null 2>&1; then
  if redis-cli -u "$REDIS_URL" ping 2>/dev/null | grep -q PONG; then
    echo "  [ok] reachable at REDIS_URL"
  else
    echo "  [FAIL] redis-cli can't reach REDIS_URL. Start it natively:"
    echo "    redis-server --daemonize yes"
    FAILED=1
  fi
else
  # No redis-cli on PATH -- fall back to a raw TCP probe so this doesn't
  # falsely report failure on a box that just lacks the CLI tool.
  HOST_PORT=$(echo "$REDIS_URL" | sed -E 's#redis://##; s#.*@##')
  HOST="${HOST_PORT%%:*}"
  PORT="${HOST_PORT##*:}"
  if timeout 2 bash -c "echo > /dev/tcp/$HOST/$PORT" 2>/dev/null; then
    echo "  [ok] something is listening on $HOST:$PORT (install redis-cli for a real PING check)"
  else
    echo "  [FAIL] nothing listening on $HOST:$PORT. Start Redis natively:"
    echo "    redis-server --daemonize yes"
    FAILED=1
  fi
fi

if [ "$FAILED" -eq 1 ]; then
  echo
  echo "Fix the failures above, then re-run this script."
  exit 1
fi

echo
echo "== Migrations =="
migrate -path migrations -database "$DATABASE_URL" up
echo "  [ok] migrations applied"

echo
echo "Backend is ready. Start it with: ./scripts/start.sh (or 'make run' from backend/)"
