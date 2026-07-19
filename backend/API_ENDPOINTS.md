# NavXchange API — Example Requests & Responses

Reference for every endpoint in SPEC.md §10, with real example payloads taken from the
current implementation's Go DTOs (`internal/api/dto.go`) — not hypothetical. Endpoints
that don't exist yet are clearly marked **NOT YET IMPLEMENTED** with a proposed shape so
frontend work can start against a stable contract before the backend catches up.

## Conventions

- **Base URL**: `http://localhost:8080` in local dev (configurable via `PORT`).
- **Auth**: routes marked 🔒 require `Authorization: Bearer <token>`, a JWT returned by
  `POST /api/auth/login`. Missing/invalid token → `401` with the standard error shape.
- **Content-Type**: `application/json` on every request body and response.
- **Field naming**: `snake_case` everywhere in JSON, regardless of the Go struct's
  PascalCase field names.
- **Currency values** (`currency_balance`, `current_price`, `total_currency_delta`,
  trade costs, quest rewards, etc.) are **integers in the smallest currency unit**
  (1 currency = 100 units — i.e. treat them like cents). A card priced at `2200` is
  22.00 currency; a balance of `100000` is 1,000.00 currency. Never render these as
  the raw integer.
- **Timestamps**: RFC 3339 / ISO 8601 UTC, e.g. `"2026-07-18T14:52:41.37709Z"`.
- **IDs**: UUIDv4 strings throughout.
- **Error shape** (used by every endpoint, any non-2xx status):
  ```json
  { "error": "human-readable message" }
  ```

### Error → status code reference

| Error message | HTTP status | Endpoints it can come from |
|---|---|---|
| `username already taken` | 409 | register |
| `invalid username or password` | 401 | login |
| `username must be 3-32 characters` | 400 | register |
| `password must be 8-72 characters` | 400 | register |
| `shares must be a positive number` | 400 | quote, trade execute |
| `trade type must be BUY or SELL` | 400 | quote, trade execute |
| `card not found` | 404 | quote, trade execute |
| `card is not currently active` | 409 | trade execute (BUY only — sells are always allowed) |
| `not enough remaining supply for this card` | 409 | trade execute |
| `cannot sell more shares than you own` | 409 | trade execute |
| `insufficient balance for this trade` | 409 | trade execute |
| `idempotency key was already used for a different trade` | 409 | trade execute |
| `cannot sell that many retained shares yet — still vesting` | 409 | trade execute (card creators only) |
| `trade would exceed the maximum position size for this card` | 409 | trade execute |
| `trading on this card is temporarily halted after a large price move` | 409 | trade execute |
| `total_supply must be a positive number` | 400 | launch card |
| `retained_percent must be between 0 and the maximum allowed` | 400 | launch card |
| `symbol is already in use` | 409 | launch card |
| `currency balance is below the card-launch threshold` | 409 | launch card |
| `missing authenticated user` | 401 | any 🔒 route with no/bad token |
| anything else | 500 | `"internal error"` — generic, no internals leaked |

---

## 1. Auth

### `POST /api/auth/register`

Request:
```json
{
  "username": "trader_jane",
  "password": "correct-horse-battery-staple"
}
```

Response `201 Created`:
```json
{
  "id": "3f2a1c9e-4b7d-4e21-9c2a-1a2b3c4d5e6f",
  "username": "trader_jane",
  "user_type": "HUMAN",
  "currency_balance": 100000,
  "login_streak_count": 0,
  "last_login_at": null,
  "created_at": "2026-07-18T14:52:41.377090Z"
}
```

Notes for UI:
- Starting balance is always `100000` (1,000.00 currency).
- Registration does **not** grant the daily login reward or a token — the client must
  call `/api/auth/login` immediately after to get a session.
- Duplicate username → `409 {"error": "username already taken"}`.

### `POST /api/auth/login`

Request:
```json
{
  "username": "trader_jane",
  "password": "correct-horse-battery-staple"
}
```

Response `200 OK`:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIzZjJhMWM5ZS00YjdkLTRlMjEtOWMyYS0xYTJiM2M0ZDVlNmYiLCJleHAiOjE3NTMyNzc5NjF9.dQw4w9WgXcQ...",
  "user": {
    "id": "3f2a1c9e-4b7d-4e21-9c2a-1a2b3c4d5e6f",
    "username": "trader_jane",
    "user_type": "HUMAN",
    "currency_balance": 100005,
    "login_streak_count": 1,
    "last_login_at": "2026-07-18T14:52:41.377090Z",
    "created_at": "2026-07-18T09:00:00.000000Z"
  },
  "reward_granted": true,
  "reward_amount": 5
}
```

Notes for UI:
- `reward_granted` is only `true` the **first** login of a UTC calendar day — show a
  "+5 daily reward!" toast only when this is `true`. `reward_amount` is `0`/omitted-in-
  spirit when not granted (the field is always present but only meaningful when
  `reward_granted` is `true`).
- `user.currency_balance` already reflects the reward if one was granted — no need to
  add `reward_amount` to a previously-cached balance.
- Store `token` (e.g. in memory + refresh via re-login, or secure storage) and send as
  `Authorization: Bearer <token>` on every 🔒 route. Tokens expire (24h by default,
  server-configured) — a `401` on any 🔒 route means "log in again," not a bug.
- Wrong password or unknown username both return the **same** error (no user
  enumeration): `401 {"error": "invalid username or password"}`.

### `GET /api/users/me` 🔒

Response `200 OK`:
```json
{
  "id": "3f2a1c9e-4b7d-4e21-9c2a-1a2b3c4d5e6f",
  "username": "trader_jane",
  "user_type": "HUMAN",
  "currency_balance": 100005,
  "login_streak_count": 1,
  "last_login_at": "2026-07-18T14:52:41.377090Z",
  "created_at": "2026-07-18T09:00:00.000000Z"
}
```

Use this to refresh the header/wallet display after any action that might change balance
without returning a full user object itself (rare — most mutating endpoints already
return the updated user).

### `GET /api/users/{id}` — **NOT YET IMPLEMENTED**

Listed in SPEC.md §10 but no handler exists yet. Proposed shape — a public profile view,
deliberately **omitting** `currency_balance` (a live balance isn't public information;
net worth is only surfaced indirectly via the leaderboard):

```json
{
  "id": "3f2a1c9e-4b7d-4e21-9c2a-1a2b3c4d5e6f",
  "username": "trader_jane",
  "user_type": "HUMAN",
  "login_streak_count": 4,
  "created_at": "2026-07-18T09:00:00.000000Z"
}
```
`404 {"error": "user not found"}` for an unknown id. Treat this shape as provisional
until the backend lands it.

---

## 2. Cards

### `GET /api/cards` — **NOT YET IMPLEMENTED**

Proposed: a public, unauthenticated list of active cards (mirrors the `cardDTO` shape
already used elsewhere, e.g. in trade responses), likely paginated like `/api/news`
(`?limit=&offset=`) once built.

```json
{
  "cards": [
    {
      "id": "8b35acdf-a0a2-4781-af66-4973396ca4e2",
      "creator_user_id": null,
      "symbol": "HSTD",
      "name": "Homestead Foods",
      "description": null,
      "image_url": null,
      "card_type": "SYSTEM_COMPANY",
      "supply_model": "FIXED",
      "total_supply": 2200000,
      "circulating_supply": 1000000,
      "creator_retained_shares": 0,
      "creator_retained_shares_sold": 0,
      "current_price": 2600,
      "status": "ACTIVE",
      "created_at": "2026-01-01T00:00:00Z"
    },
    {
      "id": "3edbdac1-f3f7-4df4-acdd-171e959d58b3",
      "creator_user_id": "3f2a1c9e-4b7d-4e21-9c2a-1a2b3c4d5e6f",
      "symbol": "JANE1",
      "name": "Jane's Startup",
      "description": "A user-launched card.",
      "image_url": "https://cdn.example.com/jane1.png",
      "card_type": "USER_CREATED",
      "supply_model": "FIXED",
      "total_supply": 1000000,
      "circulating_supply": 250000,
      "creator_retained_shares": 200000,
      "creator_retained_shares_sold": 0,
      "current_price": 1050,
      "status": "ACTIVE",
      "created_at": "2026-07-15T10:00:00Z"
    }
  ],
  "limit": 20,
  "offset": 0
}
```

UI notes:
- `card_type` drives iconography/badges: `SYSTEM_COMPANY` (the 30 fixed companies),
  `INDEX` (NAV5, the only index card — `sector` is always absent/null for it),
  `USER_CREATED` (player-launched — show `creator_user_id`'s username via a lookup,
  and consider surfacing `creator_retained_shares_sold` / `creator_retained_shares` as
  a vesting progress bar).
- `supply_model: "UNLIMITED"` cards have `total_supply: null` — don't render "X of null"
  in a supply bar; show circulating supply alone or an "∞" style indicator.
- `status`: `ACTIVE` (tradable), `DELISTED`, `FROZEN` — disable the buy/sell UI for
  anything other than `ACTIVE` (sells remain allowed server-side even when not
  `ACTIVE`, per §4.2, so don't disable the sell button, only buy).

### `GET /api/cards/{id}` — **NOT YET IMPLEMENTED**

Proposed: a single `cardDTO`, same shape as one element of the list above.
`404 {"error": "card not found"}` for an unknown id.

### `POST /api/cards` 🔒 (launch a new card)

Request:
```json
{
  "symbol": "JANE1",
  "name": "Jane's Startup",
  "description": "A user-launched card.",
  "image_url": "https://cdn.example.com/jane1.png",
  "total_supply": 1000000,
  "retained_percent": 20.0,
  "idempotency_key": "9c8b7a6f-5e4d-3c2b-1a0f-9e8d7c6b5a4f"
}
```

Response `201 Created`:
```json
{
  "card": {
    "id": "3edbdac1-f3f7-4df4-acdd-171e959d58b3",
    "creator_user_id": "3f2a1c9e-4b7d-4e21-9c2a-1a2b3c4d5e6f",
    "symbol": "JANE1",
    "name": "Jane's Startup",
    "description": "A user-launched card.",
    "image_url": "https://cdn.example.com/jane1.png",
    "card_type": "USER_CREATED",
    "supply_model": "FIXED",
    "total_supply": 1000000,
    "circulating_supply": 200000,
    "creator_retained_shares": 200000,
    "creator_retained_shares_sold": 0,
    "current_price": 1000,
    "status": "ACTIVE",
    "created_at": "2026-07-18T14:52:41.377090Z"
  },
  "transaction": {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "type": "CARD_LAUNCH",
    "card_id": "3edbdac1-f3f7-4df4-acdd-171e959d58b3",
    "shares": 200000,
    "price_per_share": null,
    "total_currency_delta": -10000,
    "resulting_balance": 90005,
    "created_at": "2026-07-18T14:52:41.377090Z"
  },
  "user": {
    "id": "3f2a1c9e-4b7d-4e21-9c2a-1a2b3c4d5e6f",
    "username": "trader_jane",
    "user_type": "HUMAN",
    "currency_balance": 90005,
    "login_streak_count": 1,
    "last_login_at": "2026-07-18T14:52:41.377090Z",
    "created_at": "2026-07-18T09:00:00.000000Z"
  }
}
```

UI notes:
- Launching a card costs a flat **100.00 currency** (`total_currency_delta: -10000`),
  deducted immediately — show this as an upfront cost in the launch form, not just a
  "retained shares" tradeoff.
- `transaction.shares` on a `CARD_LAUNCH` transaction is the creator's **retained**
  share count (not total_supply) — here `20% of 1,000,000 = 200,000`.
- `retained_percent` is a **float 0–100** (a UI percent slider maps directly, no /100
  conversion needed) — server enforces a maximum (currently returns `409` if you're
  below a currency threshold, `400` if the percent is out of range).
- `idempotency_key` — generate a fresh UUID client-side per *attempt*; reusing the
  same key on retry-after-network-error safely returns the original result instead of
  double-launching.
- Card creators can't immediately dump `creator_retained_shares` — see `creator_retained_shares_sold`
  climb over time as vesting unlocks; selling too much too soon returns
  `409 {"error": "cannot sell that many retained shares yet — still vesting"}` on a
  subsequent `/api/trades/execute` SELL, not on launch itself.

### `GET /api/cards/{id}/price-history` — **NOT YET IMPLEMENTED**

Backs charting. Proposed shape, based on the existing (currently unused by any reader)
`price_ticks` table — one row per recorded tick, `card_id` omitted per-row since it's
implied by the URL:

```json
{
  "card_id": "8b35acdf-a0a2-4781-af66-4973396ca4e2",
  "symbol": "HSTD",
  "ticks": [
    { "price": 2550, "volume": 1200, "ts": "2026-07-18T14:00:00Z" },
    { "price": 2580, "volume": 3400, "ts": "2026-07-18T14:00:30Z" },
    { "price": 2600, "volume": 900,  "ts": "2026-07-18T14:01:00Z" }
  ],
  "limit": 500,
  "offset": 0
}
```
`ticks` ordered oldest→newest (chart-ready left-to-right). `volume` is shares traded in
that tick's window, not currency. Expect a `from`/`to` or `interval` query param design
once implemented — not finalized yet.

---

## 3. Trades

### `POST /api/trades/quote` 🔒

Non-binding preview — **no mutation**, no `idempotency_key`, safe to call on every
keystroke of a share-quantity input (debounce recommended anyway).

Request:
```json
{
  "card_id": "8b35acdf-a0a2-4781-af66-4973396ca4e2",
  "type": "BUY",
  "shares": 50
}
```

Response `200 OK`:
```json
{
  "card": {
    "id": "8b35acdf-a0a2-4781-af66-4973396ca4e2",
    "creator_user_id": null,
    "symbol": "HSTD",
    "name": "Homestead Foods",
    "description": null,
    "image_url": null,
    "card_type": "SYSTEM_COMPANY",
    "supply_model": "FIXED",
    "total_supply": 2200000,
    "circulating_supply": 1000000,
    "creator_retained_shares": 0,
    "creator_retained_shares_sold": 0,
    "current_price": 2600,
    "status": "ACTIVE",
    "created_at": "2026-01-01T00:00:00Z"
  },
  "type": "BUY",
  "shares": 50,
  "estimated_cost": 130250,
  "estimated_fee": 1303,
  "estimated_price_per_share": 2605
}
```

UI notes:
- `estimated_cost` sign convention: **positive = buyer would pay, negative = seller
  would receive** — a SELL quote's `estimated_cost` will be a negative number; render
  `Math.abs()` of it as "you'll receive ~X.XX".
- `estimated_price_per_share` is the *average* price across the whole order (slippage
  already integrated) — it will differ from `card.current_price` for larger orders;
  show both so the slippage is visible ("current price 26.00, your avg 26.05").
- `estimated_fee` is already included in `estimated_cost` for a BUY (total you'll be
  charged); for a SELL it's deducted from what you receive. Show it as a separate
  line item regardless, for transparency.
- This is a **preview only** — the real price is re-derived server-side at execute
  time, so the actual numbers on `/api/trades/execute` can differ slightly if the
  market moved between quote and execute. Don't treat the quote as a locked price.

### `POST /api/trades/execute` 🔒

Request:
```json
{
  "card_id": "8b35acdf-a0a2-4781-af66-4973396ca4e2",
  "type": "BUY",
  "shares": 50,
  "idempotency_key": "b2c3d4e5-f6a7-8901-bcde-f21234567891"
}
```

Response `200 OK`:
```json
{
  "transaction": {
    "id": "c3d4e5f6-a7b8-9012-cdef-321234567892",
    "type": "BUY",
    "card_id": "8b35acdf-a0a2-4781-af66-4973396ca4e2",
    "shares": 50,
    "price_per_share": 2605,
    "total_currency_delta": -130250,
    "resulting_balance": 99735750,
    "created_at": "2026-07-18T14:53:00.000000Z"
  },
  "fee_transaction": {
    "id": "d4e5f6a7-b8c9-0123-def0-4321234567893",
    "type": "FEE",
    "card_id": "8b35acdf-a0a2-4781-af66-4973396ca4e2",
    "shares": null,
    "price_per_share": null,
    "total_currency_delta": -1303,
    "resulting_balance": 99734447,
    "created_at": "2026-07-18T14:53:00.000000Z"
  },
  "user": {
    "id": "3f2a1c9e-4b7d-4e21-9c2a-1a2b3c4d5e6f",
    "username": "trader_jane",
    "user_type": "HUMAN",
    "currency_balance": 99734447,
    "login_streak_count": 1,
    "last_login_at": "2026-07-18T14:52:41.377090Z",
    "created_at": "2026-07-18T09:00:00.000000Z"
  },
  "card": {
    "id": "8b35acdf-a0a2-4781-af66-4973396ca4e2",
    "creator_user_id": null,
    "symbol": "HSTD",
    "name": "Homestead Foods",
    "description": null,
    "image_url": null,
    "card_type": "SYSTEM_COMPANY",
    "supply_model": "FIXED",
    "total_supply": 2200000,
    "circulating_supply": 1000050,
    "creator_retained_shares": 0,
    "creator_retained_shares_sold": 0,
    "current_price": 2606,
    "status": "ACTIVE",
    "created_at": "2026-01-01T00:00:00Z"
  }
}
```

UI notes:
- **Two transactions always come back together**: `transaction` (the BUY/SELL itself)
  and `fee_transaction` (always `type: "FEE"`, linked via `related_transaction_id` on
  the fee row pointing back at `transaction.id`, though that field isn't rendered
  above since `transactionDTO` doesn't expose it — treat them as a pair from this one
  response). Show them as one combined line in a trade-history list ("BUY 50 HSTD @
  26.05 — fee 13.03").
- `transaction.total_currency_delta` and `fee_transaction.total_currency_delta` are
  both negative for a BUY (money leaving the account); for a SELL, `transaction`'s
  delta is positive (money arriving) while `fee_transaction`'s is still negative (fee
  always costs you, both directions).
- `user.currency_balance` in this response is **already final** — apply it directly to
  the wallet display, don't sum deltas yourself.
- `card.current_price` and `card.circulating_supply` reflect the *post-trade* state —
  refresh any card-detail view open elsewhere in the UI with this fresh data rather
  than re-fetching.
- `idempotency_key` must be a **fresh UUID per distinct trade intent** — reuse (e.g. on
  a retried request after a timeout) safely replays the original result instead of
  double-executing; reusing the same key for a *different* trade (different shares/
  type/card) returns `409 {"error": "idempotency key was already used for a different trade"}`.
- A rejected trade (insufficient balance, circuit breaker, position cap, etc.) returns
  `409` with the specific message from the error table above — surface these directly
  to the user rather than a generic "trade failed."

---

## 4. Leaderboard

### `GET /api/leaderboard`

Public, unauthenticated. Always serves whatever's cached from the last scheduled
refresh (~every 60s) — never computed live, so don't expect it to reflect a trade that
executed 2 seconds ago.

Response `200 OK`:
```json
{
  "leaderboard": [
    {
      "rank": 1,
      "user_id": "3f2a1c9e-4b7d-4e21-9c2a-1a2b3c4d5e6f",
      "username": "trader_jane",
      "net_worth": 1245000,
      "change_from_last_refresh": 15000
    },
    {
      "rank": 2,
      "user_id": "5a6b7c8d-9e0f-1a2b-3c4d-5e6f7a8b9c0d",
      "username": "market_mike",
      "net_worth": 998000,
      "change_from_last_refresh": -3200
    },
    {
      "rank": 3,
      "user_id": "6b7c8d9e-0f1a-2b3c-4d5e-6f7a8b9c0d1e",
      "username": "newcomer_nia",
      "net_worth": 500000
    }
  ]
}
```

UI notes:
- `rank` is **1-indexed**, already sorted — render in array order, don't re-sort.
- `change_from_last_refresh` is **omitted entirely** (not `null` — the JSON key is
  absent) on a user's very first appearance in a cached leaderboard, or if there's no
  prior snapshot yet. Treat "key absent" as "no change data available yet" (e.g. show
  a neutral dash, not a 0 or down-arrow).
- Only up to the top 100 by net worth appear; there is no pagination — a user outside
  the top 100 simply won't be in this list (there's no "your rank" lookup endpoint
  today; consider that a gap if you need "you are #142" in the UI).
- `user_type = BOT` accounts are excluded server-side — every entry here is a real
  player.
- `net_worth` = currency balance + current market value of all holdings, in the same
  smallest-unit integer convention as everything else.

---

## 5. News

### `GET /api/news`

Public, unauthenticated. Query params: `?limit=20&offset=0` (limit defaults to 20,
capped at 100; offset defaults to 0, uncapped).

Response `200 OK`:
```json
{
  "news": [
    {
      "id": "419b21e7-f665-4ce3-bdf0-98c9bb9b6206",
      "headline": "Drought in Brittania affects Agriculture and Food markets",
      "body": null,
      "category": "DROUGHT",
      "related_card_id": null,
      "created_at": "2026-07-18T14:46:21.000000Z"
    },
    {
      "id": "c413927e-6054-4556-aec3-63b574bc9f36",
      "headline": "Embargo in Sadia affects Oil Gas and Metals markets",
      "body": null,
      "category": "EMBARGO",
      "related_card_id": null,
      "created_at": "2026-07-18T14:46:31.000000Z"
    },
    {
      "id": "0b280bd1-89ce-4a57-b909-29fb1ca41f4f",
      "headline": "Discovery in Mexiga affects Semiconductor, Metals and Misc Commodities markets",
      "body": null,
      "category": "DISCOVERY",
      "related_card_id": null,
      "created_at": "2026-07-18T14:46:51.000000Z"
    }
  ],
  "limit": 20,
  "offset": 0
}
```

UI notes:
- Ordered **most recent first** — render top-to-bottom as-is for a newspaper feed.
- `category` is the raw event type (`FLOOD`, `DROUGHT`, `WAR`, `EMBARGO`, `STRIKE`,
  `DISCOVERY`, plus non-macro categories like `CARD_LAUNCH`/`CIRCUIT_BREAKER`/etc. as
  those event sources come online) — use it to pick an icon, not `headline` text
  matching.
- `body` and `related_card_id` are **usually null today** — the current generator only
  populates `headline`+`category` for macro/sector events (a flood affects a sector,
  not one specific card). Build the UI to handle both present and absent gracefully;
  don't assume `related_card_id` is always there even for card-specific news types.
- These are the exact same headlines the news-reactive market bots react to — a food
  sector headline and a subsequent price dip on food-sector cards are causally linked,
  not decorative, so it's reasonable to link a headline to "affected sector" cards in
  the UI (parse `headline`'s "affects X markets" clause, or wait for a future
  `affected_sector` field — not present today).

---

## 6. Quests *(bonus — not in SPEC.md §10, added in Phase 9, but live today)*

### `GET /api/quests` 🔒

Response `200 OK`:
```json
{
  "quests": [
    {
      "id": "7f8e9d0c-1b2a-3c4d-5e6f-7a8b9c0d1e2f",
      "title": "Make 3 trades today",
      "progress": 2,
      "target_value": 3,
      "reward_currency": 100,
      "completed": false,
      "reset_at": "2026-07-19T00:00:00Z"
    },
    {
      "id": "8a9b0c1d-2e3f-4a5b-6c7d-8e9f0a1b2c3d",
      "title": "Hold any card for 24 hours",
      "progress": 1,
      "target_value": 1,
      "reward_currency": 150,
      "completed": true,
      "reset_at": "2026-07-19T00:00:00Z"
    },
    {
      "id": "9b0c1d2e-3f4a-5b6c-7d8e-9f0a1b2c3d4e",
      "title": "Reach rank 50 on the leaderboard",
      "progress": 0,
      "target_value": 1,
      "reward_currency": 200,
      "completed": false,
      "reset_at": "2026-07-19T00:00:00Z"
    }
  ]
}
```

UI notes:
- `progress`/`target_value` drive a progress bar (`2/3`, `1/1`, `0/1`). All three
  quests reset daily at `reset_at` (UTC midnight) regardless of completion — show a
  countdown to `reset_at`, not a static "resets daily" label.
- `completed: true` → show a checkmark/claimed state; the reward has **already** been
  credited automatically (as a `QUEST_REWARD` transaction — see below) the moment the
  condition was met. There is no separate "claim" action/endpoint — completion and
  payout happen atomically server-side.
- A completed quest's `progress` stays pinned at `target_value` until the next
  `reset_at` — a 4th trade past the "make 3 trades" target does not increment past 3
  or grant a second reward.
- Quest-reward transactions appear in a user's transaction history (were you to build
  a trade-history view) with `type: "QUEST_REWARD"` — same shape as any other
  `transactionDTO`, e.g.:
  ```json
  {
    "id": "a0b1c2d3-e4f5-6789-0abc-def123456789",
    "type": "QUEST_REWARD",
    "card_id": null,
    "shares": null,
    "price_per_share": null,
    "total_currency_delta": 100,
    "resulting_balance": 100105,
    "created_at": "2026-07-18T15:10:00.000000Z"
  }
  ```
  (No dedicated GET-transactions endpoint exists yet either — this shape is shown for
  completeness in case one is added.)

---

## 7. WebSocket

⚠️ **Not implemented yet.** `internal/ws` doesn't exist in the codebase as of this
writing — SPEC.md §10 lists `WS /ws/prices` and `WS /ws/news`, but there's no server
code behind them. Everything below is a **proposed** message format, designed to be
consistent with the REST DTOs above (same field names/casing/units) so a frontend
built against this doc needs no rework once the real thing lands — but treat every
detail here as subject to change until `internal/ws` actually exists.

### Connection & envelope convention

Proposed: every message on every WS channel is a small JSON envelope with a `type`
discriminator and a `data` payload, so a single `onmessage` handler can route by type:

```json
{ "type": "<message_type>", "data": { ... } }
```

### `WS /ws/prices` — subscribe to live price ticks

One message per executed trade (human or bot) or drift-ticker tick, for every actively
traded card (not just ones the client is "watching" — proposed client-side filtering
by `card_id`/`symbol`, no server-side subscribe-to-specific-card protocol assumed yet).

```json
{
  "type": "price_tick",
  "data": {
    "card_id": "8b35acdf-a0a2-4781-af66-4973396ca4e2",
    "symbol": "HSTD",
    "price": 2606,
    "previous_price": 2600,
    "volume": 50,
    "ts": "2026-07-18T14:53:00.000000Z"
  }
}
```

UI notes:
- `price`/`previous_price` are both included so a client can flash green/red on the
  ticker without keeping its own prior-price cache.
- `volume` is shares moved in *this* tick (matches the trade/drift event that caused
  it), not a rolling total.
- Expect very high message frequency across all cards combined (every bot trade is a
  tick) — batch/throttle UI re-renders (e.g. requestAnimationFrame) rather than
  re-rendering per message for cards not currently visible.

### `WS /ws/news` — subscribe to live news

One message per headline the moment it's generated — same shape as a `newsEventDTO`
element from `GET /api/news`, wrapped in the envelope:

```json
{
  "type": "news_published",
  "data": {
    "id": "419b21e7-f665-4ce3-bdf0-98c9bb9b6206",
    "headline": "Drought in Brittania affects Agriculture and Food markets",
    "body": null,
    "category": "DROUGHT",
    "related_card_id": null,
    "created_at": "2026-07-18T14:46:21.000000Z"
  }
}
```

UI notes: prepend to whatever list `GET /api/news` populated on initial load — same DTO
shape, so no separate parsing path needed for "live" vs "page load" news items.

### Leaderboard live update — *proposed extension beyond SPEC.md's WS surface*

SPEC.md §10 only lists `/ws/prices` and `/ws/news` — there's no leaderboard WS channel
in the spec. Since this was asked for, here's a plausible design if one were added
(e.g. a third channel `/ws/leaderboard`, or folded into `/ws/news` with its own `type`):

```json
{
  "type": "leaderboard_update",
  "data": {
    "leaderboard": [
      { "rank": 1, "user_id": "3f2a1c9e-4b7d-4e21-9c2a-1a2b3c4d5e6f", "username": "trader_jane", "net_worth": 1245000, "change_from_last_refresh": 15000 },
      { "rank": 2, "user_id": "5a6b7c8d-9e0f-1a2b-3c4d-5e6f7a8b9c0d", "username": "market_mike", "net_worth": 998000, "change_from_last_refresh": -3200 }
    ]
  }
}
```
Same array shape as `GET /api/leaderboard`'s `leaderboard` field — pushed wholesale on
each ~60s refresh rather than as a diff, matching how the REST endpoint already works
(simplest to implement, and the payload is small — top 100 rows). Treat this section as
a suggestion, not a commitment, until it's actually built.
