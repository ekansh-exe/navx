# NavXchange: backend

Go API, trading engine, market-making bots, news generator, and WebSocket
hub. Everything that touches currency goes through `internal/ledger`: bots
and the HTTP handlers call the exact same `ExecuteTrade`, so there's no
special-cased path for either.

## Stack

Go 1.25 · chi (routing) · pgx/v5 (Postgres) · go-redis v9 · golang-jwt ·
golang-migrate (embedded, runs at boot) · coder/websocket

## Pricing mechanics

Each card follows a square-root bonding curve:

```
price = base_price * sqrt((circulating_supply + 1) / scale)
```

Buying increases supply (price up), selling decreases it (price down). A
trade's cost is the curve integrated across the shares traded, not price ×
shares, so a large order visibly slips worse than a small one at the same
starting price.

On top of that:

- **Fee**: 1% of trade value, minimum 1 currency unit, on every buy and sell.
- **Position cap**: a user can't hold more than 25% of a card's circulating
  supply (buy-side only; selling is never blocked by this).
- **Wash-trade deterrent**: reversing your own last trade on the same card
  within 5 minutes multiplies that leg's fee ×5 instead of rejecting it.
- **Circuit breaker**: if a card's price moves >15% within a 1-minute
  window, trading halts for 30 seconds.
- **Creator vesting**: a card's launch creator has a retained allocation
  that unlocks gradually rather than being sellable immediately.

## Package overview

| Package | Responsibility |
|---|---|
| `internal/engine` | Pure bonding-curve math (no DB, no side effects) |
| `internal/ledger` | The only package that mutates currency: trades, card launches, daily rewards, bot rebalancing |
| `internal/bots` | 10 bot accounts (momentum / contrarian / random-walker / news-reactive / index-tracker), each its own goroutine |
| `internal/news` | Generates sector-tagged headlines on a timer |
| `internal/leaderboard` | Computes + caches (Redis) the top-100 net-worth ranking, plus the GOAT tribute row |
| `internal/quests` | Daily quests (trade count / hold a position / reach a rank) |
| `internal/auth` | Registration, login, JWT issuance |
| `internal/ws` | The WebSocket hub: fans out price ticks, news, and leaderboard updates |
| `internal/api` | Thin HTTP handlers; all logic lives in the packages above |
| `internal/migrate` | Applies `migrations/*.sql` (embedded in the binary) at startup |

## Running locally

Requires a native (non-Docker) Postgres and Redis already running, plus Go
and the `migrate` CLI (only needed for the manual workflow below; the
server applies migrations itself on boot).

```bash
./scripts/setup.sh   # checks deps, creates .env from .env.example, creates the DB role/db
make run              # or: go run ./cmd/server
```

Or just run `./dev.sh` from the repo root, which does this for both projects
at once.

Configuration is via environment variables. `setup.sh` creates `backend/.env`
for you locally; production values are set directly on the host, never
committed.

## API surface

- `POST /api/auth/register`, `POST /api/auth/login`
- `GET /api/cards`, `GET /api/cards/{id}`, `POST /api/cards` (launch, auth)
- `POST /api/trades/quote`, `POST /api/trades/execute` (auth)
- `GET /api/news`, `GET /api/leaderboard`, `GET /api/quests` (auth)
- `GET /api/users/me` (auth)
- `WS /ws/prices`, `WS /ws/news`, `WS /ws/leaderboard`

## Migrations

Migrations live in `migrations/*.sql` and are embedded into the binary
(`migrations/embed.go`); the server applies any pending ones automatically
on every boot (`internal/migrate`), so a fresh deploy never needs a human to
run `migrate` by hand. The CLI workflow (`make migrate-up` / `migrate-down`)
still works unchanged for manual use against the same files on disk.

## Tests

```bash
go test ./...
```

Most packages have DB/Redis integration tests that skip automatically if a
local database/cache isn't reachable.
