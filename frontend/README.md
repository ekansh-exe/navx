# NavXchange — frontend

React SPA for NavXchange: the trading dashboard, card detail pages with live
price charts, portfolio, leaderboard, news feed, and daily quests.

**Live:** https://navx-nine.vercel.app

## Stack

React 19 · Vite · TypeScript · TanStack Query · Zustand · react-router-dom ·
Tailwind CSS v4 · Radix UI · Recharts · Sonner (toasts)

## Running locally

Requires the backend running (see `backend/README.md`, or just run
`./dev.sh` from the repo root for both at once).

```bash
./scripts/setup.sh   # checks deps, npm install
npm run dev           # http://localhost:5173
```

`.env` is optional — with none present, the app defaults to
`http://localhost:8080` / `ws://localhost:8080`, which matches the backend's
own local default. Copy `.env.example` only if you need to point at
something else.

## Structure

```
src/api/         REST calls, one file per resource
src/ws/          WebSocketProvider (manages all 3 channels + reconnect/backoff)
                 + priceTickStore (high-frequency price state, outside React)
src/stores/      Zustand stores (auth, leaderboard)
src/hooks/       React Query hooks wrapping src/api
src/pages/       Route-level views
src/components/  Presentational + feature components, grouped by page
src/types/       API/WS payload shapes, hand-kept in sync with the backend DTOs
```

Live price ticks bypass React state entirely (`src/ws/priceTickStore.ts`) —
only the specific card component currently on screen re-renders per tick,
since the backend warns of high message volume across ~30 cards trading
continuously.

## Build

```bash
npm run build   # tsc -b && vite build -> dist/
```

Deployed on Vercel with **Root Directory** set to `frontend` (this is a
monorepo) and a SPA rewrite (`vercel.json`) so client-side routes don't 404
on refresh.
