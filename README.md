# NavXchange

A fictional commodity/stock trading game. Users trade shares in ~30 seeded
"companies" and an index (**NAV5**) whose prices move on a live bonding
curve, driven by real trades, a fleet of always-on trading bots, and
randomly generated news events that shock whole sectors at once.

**Live:** https://navx-nine.vercel.app

## Features

- **Bonding-curve pricing**: every card's price is a deterministic function
  of its circulating supply (`price = base_price * sqrt(supply / scale)`), so
  buying pushes a price up and selling pushes it down, with real slippage
  integrated across the trade size rather than a flat per-share quote.
- **Market-making bots**: 10 bot accounts across 5 personas (momentum,
  contrarian, random-walker, news-reactive, index-tracker) trade continuously
  through the exact same execution path a human's trade takes, so the market
  stays active even with no one watching.
- **News events**: a periodic job generates fictional geopolitical/commodity
  headlines (floods, wars, embargoes, discoveries...) tied to specific
  sectors; news-reactive bots trade off them, and every event streams live to
  connected clients.
- **Anti-exploit guardrails**: a 25%-of-supply position cap per user per
  card, a circuit breaker that halts trading after a >15% move in a minute,
  and a punitive fee multiplier on rapid buy/sell round-trips.
- **Live leaderboard**: top 100 users by net worth, refreshed on a schedule
  and pushed live over WebSocket, with a permanent tribute row pinned above
  rank 1.
- **Daily quests & rewards**: daily login bonus with streak tracking, plus
  rotating daily quests (trade count, hold a position, reach a rank).
- **Real-time throughout**: `/ws/prices`, `/ws/news`, and `/ws/leaderboard`
  push every price tick, headline, and leaderboard refresh to the browser
  with no polling.

See [backend/README.md](backend/README.md) and
[frontend/README.md](frontend/README.md) for the details of each half.

## Repo layout

```
backend/    Go API + trading engine + bots + WebSocket hub
frontend/   React + Vite SPA
dev.sh      one-command local dev (both, no Docker)
```

## Running it locally

You need a local Postgres and Redis already running (native install, not
Docker; see `backend/scripts/setup.sh` for exact instructions if you don't
have them) plus Go and Node installed. Then, from the repo root:

```bash
./dev.sh
```

This runs each project's setup script (installs nothing for you, but checks
dependencies, creates `.env` from `.env.example` if missing, and applies
migrations), then starts:

- backend on `:8080`, waiting for its `/health` check to go green
- frontend on `:5173`

`Ctrl+C` stops both. Open `http://localhost:5173`.

Prefer to run each side yourself? See the "Running locally" section in
[backend/README.md](backend/README.md) and
[frontend/README.md](frontend/README.md).

## Deployment

Backend runs as a Docker container on **Render**, backed by **Neon**
(Postgres) and **Upstash** (Redis, TLS). Migrations are embedded in the Go
binary and applied automatically on every boot, with no manual migration
step required against production. Frontend is a static Vite build on **Vercel**.
