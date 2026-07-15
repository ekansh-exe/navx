# NavXchange — Full Build Specification

**Purpose of this document:** This is a complete, ordered engineering spec meant to be handed to an AI coding agent (Opus 4.8) in phases. Each phase is scoped to be buildable and testable independently. Do not skip the "Correctness & Concurrency" sections — this app is fundamentally a financial ledger with a game skin on it, and most "fun bugs" in games like this (duplicated currency, negative balances, race-condition exploits) come from treating it like a normal CRUD app instead.

**Branding:** Product name is **NavXchange**. Shorthand/logo mark used throughout the UI (header, favicon, loading states, watermark on charts) is **NavX**. Use "NavXchange" in full on the landing page, auth screens, and page titles; use the "NavX" mark anywhere space is tight (nav bar logo, mobile header, browser tab icon).

**Stack decision:**
- Backend: Go (chi or Echo router, sqlc or GORM for DB access, PostgreSQL, Redis for caching/pubsub)
- Frontend: React + TypeScript + Vite, TailwindCSS, WebSockets for live price feed
- Real-time: WebSocket server for price ticks, leaderboard updates, news feed
- Deployment target: containerized (Docker), assume Postgres + Redis + Go binary + static frontend

---

## 1. Core Concept Recap

Players log in (real account, not anonymous) and start with a fixed amount of virtual currency. They buy/sell shares in **companies** whose prices move based on supply/demand, algorithmic drift, and randomness. The market is anchored by **30 fixed, system-seeded companies** (mostly commodity-sector: food, oil & gas, semiconductors, etc. — see §5), with the top 5 by market cap also tracked as a composite index card, **NAV5** (an S&P500-style index of the market). On top of this fixed roster, players who cross a currency threshold can still publish their own user-generated card (see §6), choosing its supply model and retained ownership stake. The system must resist "infinite money glitches" (self-trading, wash trading, pump-and-dump exploits by whales), and needs daily-engagement hooks (leaderboard, in-game newspaper, login rewards — including a flat +5 currency grant every day the player logs in).

---

## 2. Domain Model

### Entities

**User**
- id, username, display_name, created_at
- currency_balance (int64, stored in smallest unit — e.g. "cents" of the in-game currency, never float)
- total_net_worth (derived: cash + holdings valued at current price — recomputed, not stored as source of truth)
- last_login_at, login_streak_count

**Card** (covers system companies, the NAV5 index, and user-created cards — one table, distinguished by `card_type`)
- id, creator_user_id (nullable — system-issued companies and the index have no creator)
- card_type: `SYSTEM_COMPANY` | `INDEX` | `USER_CREATED`
- sector (nullable for user-created — e.g. `FOOD`, `OIL_GAS`, `SEMICONDUCTOR`, `METALS`, `AGRICULTURE`; see §5 for the seed list)
- name, symbol, description, image_url
- supply_model: `FIXED` | `UNLIMITED`
- total_supply (nullable if UNLIMITED — see §4.4 for how unlimited actually works)
- creator_retained_shares (shares creator keeps at launch — 0 for system/index cards)
- creator_retained_percent (for display)
- current_price (denormalized cache, source of truth is price engine)
- created_at, status: `ACTIVE` | `DELISTED` | `FROZEN`

**IndexComponent** (only relevant for the NAV5 card)
- index_card_id, component_card_id, weight
- Recomputed on a schedule as the top-5-by-market-cap set changes (see §5.2) — membership isn't static forever, but changes should be infrequent and announced via the newspaper, not silent.

**Holding**
- user_id, card_id, shares_owned (int64), avg_cost_basis
- Composite primary key (user_id, card_id)

**Transaction** (append-only ledger — this is the actual source of truth)
- id, user_id, card_id, type: `BUY` | `SELL` | `CARD_LAUNCH` | `DAILY_REWARD` | `FEE`
- shares, price_per_share, total_currency_delta, resulting_balance
- created_at, idempotency_key

**PriceTick**
- card_id, price, volume, timestamp (for charting)

**NewsEvent**
- id, headline, body, category, related_card_id (nullable), created_at

### Non-negotiable invariant
> `user.currency_balance` must always equal the sum of currency-affecting transactions for that user, and must never go negative. Every trade is one atomic DB transaction. No exceptions, no "we'll fix the balance later" logic.

---

## 3. Database Schema (PostgreSQL, illustrative)

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT UNIQUE NOT NULL,
    currency_balance BIGINT NOT NULL DEFAULT 100000 CHECK (currency_balance >= 0),
    login_streak_count INT NOT NULL DEFAULT 0,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE cards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_user_id UUID REFERENCES users(id),
    card_type TEXT NOT NULL CHECK (card_type IN ('SYSTEM_COMPANY','INDEX','USER_CREATED')),
    sector TEXT,
    symbol TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    supply_model TEXT NOT NULL CHECK (supply_model IN ('FIXED','UNLIMITED')),
    total_supply BIGINT,
    circulating_supply BIGINT NOT NULL DEFAULT 0,
    creator_retained_shares BIGINT NOT NULL DEFAULT 0,
    current_price BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE index_components (
    index_card_id UUID REFERENCES cards(id),
    component_card_id UUID REFERENCES cards(id),
    weight NUMERIC NOT NULL,
    PRIMARY KEY (index_card_id, component_card_id)
);

CREATE TABLE holdings (
    user_id UUID REFERENCES users(id),
    card_id UUID REFERENCES cards(id),
    shares_owned BIGINT NOT NULL CHECK (shares_owned >= 0),
    avg_cost_basis BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, card_id)
);

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    card_id UUID REFERENCES cards(id),
    type TEXT NOT NULL,
    shares BIGINT,
    price_per_share BIGINT,
    total_currency_delta BIGINT NOT NULL,
    resulting_balance BIGINT NOT NULL,
    idempotency_key TEXT UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE price_ticks (
    card_id UUID REFERENCES cards(id),
    price BIGINT NOT NULL,
    volume BIGINT NOT NULL DEFAULT 0,
    ts TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_price_ticks_card_ts ON price_ticks(card_id, ts DESC);

CREATE TABLE news_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    headline TEXT NOT NULL,
    body TEXT,
    category TEXT,
    related_card_id UUID REFERENCES cards(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

All currency and share fields are `BIGINT`, never floating point. Fractional pricing is handled by storing prices in a fixed smallest unit (e.g. 1 currency = 10000 units) to avoid float drift entirely.

---

## 4. Market Simulation Engine

### 4.1 Pricing model — bonding curve, not order book (recommended)

An order-matching order book is more "realistic" but is dramatically harder to make exploit-free and is overkill for this game. Recommend an **automated market maker (AMM) style bonding curve** per card instead — this makes price a pure function of circulating supply plus a demand/drift modifier, which is much easier to reason about and test.

```
price(card) = base_price * curve_multiplier(circulating_supply) * demand_modifier * drift_factor
```

- `curve_multiplier`: monotonically increasing function of circulating supply (e.g. square-root or logistic curve) — this is what makes buying push price up and selling push it down, automatically, with no order book needed.
- `demand_modifier`: rolling factor based on recent net buy volume (e.g. EMA of last N trades' net direction), decays over time.
- `drift_factor`: small bounded random walk per tick (server-side cron, e.g. every 30s), clamped to a max % move per tick to prevent discontinuous jumps.

### 4.2 Trade execution (this is the critical-correctness part)

Every BUY/SELL must:
1. Run inside a single DB transaction with `SELECT ... FOR UPDATE` locking the card row (and user row) to serialize concurrent trades on the same card.
2. Recompute the execution price **at execution time**, not at quote time (quote-then-confirm is fine for UX, but re-derive price server-side — never trust a client-submitted price).
3. Apply slippage: large orders move the price *during* execution (integrate the curve across the trade size, don't apply a single price to the whole order) — this alone kills most "buy a ton at old low price" exploits.
4. Deduct a small transaction fee (e.g. 0.5–1%) on both buy and sell — this is your primary anti-wash-trading lever, since round-tripping (buy then immediately sell) becomes a guaranteed small loss.
5. Write the `Transaction` row and update `resulting_balance` inside the same DB transaction, using an idempotency key from the client to make retries safe.

### 4.3 Anti-whale / anti-exploit checklist

- **Slippage on size** (§4.2.3) — the more you buy in one trade, the worse your average price. This is the single most important anti-glitch mechanism.
- **Per-user, per-card position caps** (e.g. no single user can own more than X% of circulating supply) — prevents total market cornering.
- **Transaction fees** on both sides — prevents zero-cost round-tripping.
- **Self-trade / wash-trade detection**: flag rapid buy-sell pairs by the same user on the same card within a short window; either reject or apply a punitive fee multiplier.
- **Circuit breaker**: if a card's price moves more than X% within a rolling window, temporarily halt trading on it (classic stock-exchange pattern) — prevents flash-manipulation.
- **Creator self-dealing limits**: a card creator's `creator_retained_shares` should vest or be capped from immediate resale (e.g. can't dump >Y% of retained shares in the first 24h) — otherwise creators mint a card, pump it with alt accounts, dump on real players.
- **New-account trade-size ramps**: new accounts get lower max order sizes for their first N trades — slows down bot/multi-account exploitation.

### 4.4 Unlimited supply cards

For `UNLIMITED` cards, "unlimited" should still mean *mintable-on-buy, burnable-on-sell* (like an AMM), not literally infinite pre-existing supply — otherwise price math breaks. Model it as: buying mints new shares into circulation at the current curve price; selling burns them. This is effectively how crypto AMMs work and keeps the same pricing math from §4.1 valid for both supply models.

---

## 5. Seeded Companies & the NAV5 Index

### 5.1 The 30 fixed companies

At launch (Phase 0/1 data seed), the database is pre-populated with exactly 30 `SYSTEM_COMPANY` cards. These are fictional companies (avoid real brand names/tickers for legal-safety reasons — invented names in real sectors is the right pattern), spread across commodity-leaning sectors, e.g.:

- **Food/Agriculture** (~6): grain, livestock, coffee/cocoa, fishing, packaged food, agricultural equipment
- **Oil & Gas** (~6): upstream drilling, refining, pipeline/midstream, LNG, oilfield services, a national-style oil company
- **Semiconductors** (~5): chip fabrication, chip design, semiconductor equipment, memory, a consumer-electronics company that consumes chips
- **Metals & Mining** (~4): gold/precious metals, industrial metals (copper/steel), rare earth/lithium, mining equipment
- **Utilities/Energy** (~4): power generation, renewables, water utility
- **Shipping/Logistics** (~3): container shipping, freight/logistics, ports
- **Misc commodities** (~2): textiles/cotton, timber/lumber

Exact naming, symbols, and starting prices are a content task, not an architecture task — Opus 4.8 should generate 30 fictional company names/symbols fitting these sectors as part of the Phase 1 seed migration, each with a starting `current_price` and an initial `circulating_supply` sized so the bonding curve (§4.1) starts in a reasonable range.

### 5.2 NAV5 — the top-5 index

`NAV5` is a single `INDEX`-type card with no independent bonding curve of its own — its price is *derived*, not traded against directly on a curve:

```
NAV5_price = Σ (component.current_price * component.weight)
```

- Membership = the 5 `SYSTEM_COMPANY` cards with the highest market cap (`current_price * circulating_supply`), recomputed on a schedule (e.g. daily), not every tick — frequent membership churn would be confusing and hard to explain to players.
- Weight = market-cap-weighted (like a real cap-weighted index), normalized so weights sum to 1.
- Players **can** buy/sell NAV5 shares — treat it as its own mintable/burnable AMM-style asset (like an UNLIMITED-supply card, §4.4) whose *reference* price tracks the formula above, so it still needs its own slippage/fee logic even though its price target is derived from components.
- When component membership changes (a company enters/exits the top 5), auto-generate a `NewsEvent` ("Reshuffle: X enters NAV5, Y drops out").

### 5.3 User-created cards remain a separate, additional feature

Everything in the original card-creation design still applies **on top of** the fixed 30 + NAV5 — user-created cards get `card_type = USER_CREATED` and follow the flow below.

---

## 6. Card Creation Flow (user-generated cards only — system companies are seeded, not created this way)

1. Player crosses currency threshold → "Launch a Card" unlocked in UI.
2. Player sets: name, symbol, image, supply model (FIXED/UNLIMITED), total supply (if fixed), retained percentage.
3. Server validates: retained percentage within allowed max (e.g. cap at 40% to prevent instant rug-pulls), symbol uniqueness, and a currency cost to launch (sink to control spam).
4. Server creates the card (`card_type = USER_CREATED`), mints `creator_retained_shares` to the creator's holdings, sets remaining supply as purchasable from the curve.
5. Vesting schedule applied to retained shares (§4.3).
6. A `NewsEvent` is auto-generated ("X launched a new card: SYMBOL").

---

## 7. Retention Mechanics

- **Daily login reward**: flat **+5 currency** every day the player logs in (simple, predictable — no streak-multiplier complexity requested, so keep it exactly flat unless you decide otherwise). Must be idempotent per calendar day per user (one grant per day, keyed off `last_login_at`'s date vs today's date in the user's timezone or server UTC — pick one and be consistent).
- `login_streak_count` can still be tracked for display/leaderboard-of-streaks purposes even though it doesn't currently affect the reward amount.
- **Daily quests**: e.g. "make 3 trades today", "hold a card for 24h" — small currency rewards.
- Reward issuance must go through the same `Transaction` ledger as trades — never a separate ad-hoc balance update path. One code path for "money moves," always.

---

## 8. Leaderboard

- Rank by `net_worth = currency_balance + Σ(holding.shares_owned * card.current_price)`.
- Compute leaderboard on a scheduled job (e.g. every 60s) into a cached Redis sorted set — do **not** compute it live on every request by joining holdings × prices for all users, it won't scale and isn't necessary for correctness (a few seconds of staleness is fine for a leaderboard).

## 9. Newspaper / News Feed

Event-driven, generated server-side from real game events, not free-text LLM generation (keeps it deterministic and bug-free):
- Card launches (system NAV5 reshuffles, and user card launches)
- Large price moves (>X% in a tick)
- New richest player
- Circuit breaker triggers
- Daily "market summary" digest
- **Fictional geopolitical/commodity events** ("oil war", "floods", "drought", "trade embargo") that plausibly move a sector's prices — these should be tied into the pricing engine's `demand_modifier` (§4.1), not purely cosmetic: a flood headline about a food-producing fictional country should actually nudge the relevant food-sector companies' prices, so the newspaper feels causal rather than decorative.

### 9.1 Fictional country names

To avoid depicting real countries in war/disaster/crisis news, use a fixed mapping table of fictional near-homophones, seeded once and reused consistently (a player should learn "Eran = the real-world-analog oil-heavy fictional country" over time):

| Fictional name | Real-world analog (internal reference only, never shown to players) |
|---|---|
| Endia | India |
| Use (USE) | USA |
| Eran | Iran |
| Chinar | China |
| Rusko | Russia |
| Brazoria | Brazil |
| Sadia | Saudi Arabia |
| Nigera | Nigeria |
| Kanadia | Canada |
| Straya | Australia |

Extend this list as needed (aim for ~15–20 entries to comfortably cover news variety). Store it as static seed data (a simple table or even a Go constants file), and have the news-generation job pick a fictional country + event type (war, flood, drought, embargo, strike, discovery) + affected sector, then compose a headline from a template bank plus that pick — e.g. `"{event} in {country} rattles {sector} markets"` — rather than freeform generation, so headlines stay deterministic, testable, and never accidentally reference something real-world-sensitive.

---

## 10. API Surface (REST + WebSocket)

```
POST   /api/auth/register
POST   /api/auth/login
GET    /api/users/me
GET    /api/users/{id}

GET    /api/cards
GET    /api/cards/{id}
POST   /api/cards                 (launch new card)
GET    /api/cards/{id}/price-history

POST   /api/trades/quote          (non-binding price preview)
POST   /api/trades/execute        (idempotency-key required)

GET    /api/leaderboard

GET    /api/news

WS     /ws/prices                 (subscribe to live price ticks)
WS     /ws/news                   (subscribe to live news)
```

---

## 11. Go Backend Architecture

```
/cmd/server            - main entrypoint
/internal/domain       - core types (User, Card, Holding, Transaction) - no framework deps
/internal/engine       - pricing engine, curve math, drift ticker (pure functions, heavily unit tested)
/internal/store        - Postgres access (sqlc-generated or hand-written, all trade logic uses explicit transactions)
/internal/api          - HTTP handlers, request/response DTOs, validation
/internal/ws           - WebSocket hub (price + news broadcast)
/internal/jobs         - cron-style workers: drift ticker, leaderboard refresh, vesting unlocks
/migrations            - SQL migrations
```

**Concurrency notes for Go specifically:**
- The price drift ticker and any background jobs should run as separate goroutines started from `main`, coordinated with `context.Context` for clean shutdown.
- Never share mutable state (like an in-memory "current price") across goroutines without a mutex or channel — either keep price authoritative in Postgres (simplest, recommended) or use a single goroutine "owning" each card's price with channel-based requests if you need in-memory speed.
- Use `database/sql` transactions with `SELECT ... FOR UPDATE` for every trade — this is Go's tool for the row-locking described in §4.2.

---

## 12. Frontend (React + TS)

**Pages:** Login/Register, Home/Market overview (30 companies + NAV5 pinned at top), Card detail (chart + buy/sell panel), Portfolio, Leaderboard, Newspaper, Card launch wizard (user-created cards), Profile.

**Design direction** (per your "not cluttered, not too minimalist" brief): a data-dense but breathable dashboard feel — think a trading app crossed with a playful game skin. Card grid with live-updating sparkline charts, a persistent portfolio-value ticker in the header, and the newspaper rendered as a distinct visual section (not just another list) to make it feel like a feature rather than a log. The **NavX** mark anchors the top-left of the nav bar on every page; NAV5 gets visually distinguished from the 30 individual companies (e.g. a distinct card treatment or a pinned "index" row) since it behaves differently (derived price, no direct curve).

**Auth:** standard email/username + password login and registration (session or JWT-based — either is fine for a game like this; JWT is simpler to reason about statelessly across the WebSocket connections too). On successful login, trigger the daily-reward check (§7) server-side as part of the login response, not as a separate client-initiated call — keeps the "one grant per day" logic in one place and unreachable by replaying a stray API call.

**State/data:**
- React Query (or SWR) for REST data + caching
- A WebSocket context provider feeding live price ticks into subscribed components (card detail, market grid)
- Keep optimistic UI *off* for trade execution — always wait for server confirmation before updating displayed balance, given money is involved; optimistic updates here are a common source of "ghost balance" bugs.

---

## 13. Testing Strategy (do not skip — this is where "no bugs" actually gets enforced)

- **Unit tests** on the pricing engine curve math in isolation (pure functions — test edge cases: zero supply, max supply, extreme buy sizes).
- **Concurrency/load tests**: spin up many simultaneous buy/sell requests against the same card and assert the final `circulating_supply` and all user balances reconcile exactly against the transaction ledger — this is the test that catches race-condition money bugs before players do.
- **Property-based tests**: assert the ledger invariant from §2 holds after any sequence of random valid operations (property: sum of all transaction deltas per user == their current balance, always).
- **Integration tests** on the full trade API path including fee deduction and slippage.
- **Exploit simulation tests**: explicitly write tests that try to wash-trade, self-deal as a creator, and rapid-fire trade past position caps — assert they're blocked or penalized as designed in §4.3.

---

## 14. Build Order (phases to hand to Opus 4.8, in sequence)

1. **Phase 0 — Schema & skeleton**: migrations (including the 30-company + NAV5 seed data from §5), domain types, empty Go server with health check, empty React app shell with NavXchange/NavX branding placeholders.
2. **Phase 1 — Core ledger & auth**: registration/login, user creation, currency balance, daily +5 login reward, transaction ledger, the invariant tests from §13. Trading not wired up yet — just prove money can't leak and login/reward logic is correct and idempotent.
3. **Phase 2 — Pricing engine**: bonding curve math as pure functions + unit tests, no API yet.
4. **Phase 3 — Trading API**: buy/sell endpoints wired to the engine + ledger, with row-locking and idempotency. Load-test this before moving on.
5. **Phase 4 — Card launch flow**: creation, vesting, retained-share logic.
6. **Phase 5 — Anti-exploit layer**: fees, slippage sizing, position caps, wash-trade detection, circuit breakers.
7. **Phase 6 — Leaderboard + news**: scheduled jobs, WebSocket broadcast.
8. **Phase 7 — Retention mechanics**: daily rewards, streaks, quests.
9. **Phase 8 — Frontend build-out**: all pages against the now-stable API.
10. **Phase 9 — Exploit simulation test pass + load test pass** before calling it done.

Hand this document to Opus 4.8 one phase at a time rather than all at once — ask it to fully implement and test each phase before moving to the next. This keeps each unit reviewable and makes it far more likely you end up with something correct.
