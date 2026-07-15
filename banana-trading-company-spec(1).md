# NavXchange — Full Build Specification

**Purpose of this document:** This is a complete, ordered engineering spec meant to be handed to an AI coding agent (Opus 4.8) in phases. Each phase is scoped to be buildable and testable independently. Do not skip the "Correctness & Concurrency" sections — this app is fundamentally a financial ledger with a game skin on it, and most "fun bugs" in games like this (duplicated currency, negative balances, race-condition exploits) come from treating it like a normal CRUD app instead.

**Branding:** Product name is **NavXchange**. Shorthand/logo mark used throughout the UI (header, favicon, loading states, watermark on charts) is **NavX**. Use "NavXchange" in full on the landing page, auth screens, and page titles; use the "NavX" mark anywhere space is tight (nav bar logo, mobile header, browser tab icon).

**Build strategy:** Monolith first, extract services later. Everything below is written so the logic (domain model, pricing engine, bots, anti-exploit rules) is identical either way — what changes is only how it's packaged. §11 covers both: the monolith structure to build first (get it *working*), and the later extraction into microservices (§11.5) once the monolith is solid and demoable. Don't start the extraction until the monolith passes everything in §13.

**Stack decision:**
- Backend: Go — a single binary to start (internal packages, not separate processes — see §11.1), PostgreSQL, Redis for caching
- Frontend: React + TypeScript + Vite, TailwindCSS, WebSockets for live price feed
- Real-time: WebSocket hub inside the same Go binary for price ticks, leaderboard updates, news feed
- Autonomous market activity: bot personas running as background goroutines that call the same internal trade-execution function real requests use — see §4.5
- Deployment target: containerized (a single Docker image + Postgres + Redis to start; see §11.5 for the later multi-container step)

---

## 1. Core Concept Recap

Players log in (real account, not anonymous) and start with a fixed amount of virtual currency. They buy/sell shares in **companies** whose prices move based on supply/demand, algorithmic drift, and randomness. The market is anchored by **30 fixed, system-seeded companies** (mostly commodity-sector: food, oil & gas, semiconductors, etc. — see §5), with the top 5 by market cap also tracked as a composite index card, **NAV5** (an S&P500-style index of the market). On top of this fixed roster, players who cross a currency threshold can still publish their own user-generated card (see §6), choosing its supply model and retained ownership stake. The system must resist "infinite money glitches" (self-trading, wash trading, pump-and-dump exploits by whales), and needs daily-engagement hooks (leaderboard, in-game newspaper, login rewards — including a flat +5 currency grant every day the player logs in).

---

## 2. Domain Model

### Entities

**User**
- id, username, display_name, created_at
- user_type: `HUMAN` | `BOT` — bot accounts are real rows in this same table and trade through the exact same trading API and ledger as humans (see §4.5); this is what keeps bot activity honest and testable rather than a special-cased shortcut
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
    user_type TEXT NOT NULL DEFAULT 'HUMAN' CHECK (user_type IN ('HUMAN','BOT')),
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

### 4.5 Market bots — keeping the market alive with nobody playing

A pure server-side `drift_factor` random walk (§4.1) is enough to keep prices *ticking*, but it doesn't produce believable volume, doesn't interact with the anti-exploit systems, and doesn't give you anything interesting to point at in an interview. Instead, run a pool of bot personas (`user_type = BOT` accounts) that place trades through the **same internal trade-execution function** that the HTTP trading handler calls — not a raw database write, not a bypass of any check. In the monolith this is just a Go function call from a background goroutine instead of an HTTP request; if you later extract a Bot Service (§11.5), that internal call becomes a real HTTP/gRPC call to the Ledger & Market Service, but the important property — bots get zero special treatment — holds either way. This is the key design decision: every anti-exploit mechanism in §4.3 is automatically exercised and tested by your own bots before a real player ever hits it.

**Bot personas** (run several concurrently, each with different behavior, so price action looks organic rather than uniform):
- **Momentum trader**: buys cards that have been trending up over the last N ticks, sells ones trending down — amplifies moves, creates trends.
- **Contrarian / mean-reversion trader**: buys dips, sells rallies — dampens moves, creates support/resistance-like behavior.
- **Random walker**: small random buy/sell orders at random intervals, sector-agnostic — pure background noise.
- **News-reactive trader**: subscribes to the same news events (§9) as the newspaper does, and nudges trades toward/away from the sector a headline affects — this is what makes "flood in Endia" actually feel connected to price action instead of just decorative copy.
- **Index-tracking bot**: keeps NAV5's actual tradable liquidity healthy by periodically rebalancing toward the index's derived price (§5.2).

**Operational rules:**
- Bots trade on their own schedule (e.g. a ticker every few seconds per bot, jittered so they don't all fire in lockstep) via background goroutines started from `main`, not tied to whether any human is currently connected, so the market keeps moving 24/7.
- Bots are still subject to every position cap, fee, and circuit breaker in §4.3. If a bot's strategy would trip a circuit breaker, it backs off like a real trader would — this is a good source of realistic-looking "cooldown" behavior after a big move.
- Bot balances should be periodically reset/rebalanced (e.g. a nightly job) so a bot's strategy going wrong can't permanently distort a card's price or drain toward one side — treat this as its own scheduled goroutine, logged clearly so you can show "here's how I kept the simulation self-correcting" as a talking point.
- Exclude `user_type = BOT` rows from the human leaderboard (§8) by default — a leaderboard topped by bots isn't the point. You can optionally expose a separate "market activity" view that's honest about bots being bots, if you want the transparency.

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
POST   /api/trades/execute        (idempotency-key required — used identically by human clients and, once bots are extracted per §11.5, the bot service)

GET    /api/leaderboard

GET    /api/news

WS     /ws/prices                 (subscribe to live price ticks)
WS     /ws/news                   (subscribe to live news)
```

In the monolith stage, one Go server handles all of this directly. If you later extract services per §11.5, an API Gateway takes over routing these same paths to whichever internal service owns them — the frontend's contract doesn't change either way.

---

## 11. Backend Architecture — Monolith First, Then Extract

Build this as one Go binary first. Get the whole game correct and playable before introducing any distributed-systems complexity — that's what actually protects the "I don't want any bugs" goal. The microservices story (§11.5) is real and worth doing, but only *after* §11.1–§11.4 are working end to end.

### 11.1 Monolith package layout

```
/cmd/server              - main entrypoint: starts the HTTP server, WebSocket hub, and all background goroutines (drift ticker, bot personas, leaderboard refresh, vesting unlocks, daily bot rebalancing)
/internal/domain         - core types (User, Card, Holding, Transaction) - no framework deps
/internal/auth           - registration, login, JWT issuance/validation
/internal/ledger         - the trade-execution function, ledger invariant logic, row-locking — this package is the one every other package calls into for anything money-related, never the other way around
/internal/engine         - pricing engine, curve math, drift ticker (pure functions, heavily unit tested)
/internal/bots           - bot personas (§4.5), each calling /internal/ledger's trade-execution function directly, not over HTTP
/internal/news           - news generation, fictional-country template bank (§9)
/internal/leaderboard    - net-worth ranking computation, Redis caching
/internal/store          - Postgres access (sqlc-generated or hand-written, all trade logic uses explicit transactions)
/internal/api            - HTTP handlers, request/response DTOs, validation — thin: handlers call into the packages above, they don't contain business logic themselves
/internal/ws             - WebSocket hub (price + news broadcast)
/migrations              - SQL migrations
```

The `/internal/api` handlers and the `/internal/bots` goroutines should call the exact same `/internal/ledger` functions — that's what preserves the "bots get zero special treatment" property (§4.5) even without a network hop between them. Keep that boundary clean from day one; it's what makes the later extraction (§11.5) mostly a packaging exercise instead of a rewrite.

### 11.2 Concurrency notes for Go specifically

- The price drift ticker and bot goroutines run as separate goroutines started from `main`, coordinated with `context.Context` for clean shutdown.
- Never share mutable state (like an in-memory "current price") across goroutines without a mutex or channel — keep price authoritative in Postgres (simplest, recommended).
- Use `database/sql` transactions with `SELECT ... FOR UPDATE` for every trade — this is Go's tool for the row-locking described in §4.2, and it applies identically whether the caller is a human's HTTP request or a bot's goroutine.

### 11.3 Auth, still inside the monolith

Keep it simple here too: one `users` table (§3) holds both credentials and wallet data for now — don't pre-split identity from wallet until you're actually extracting Auth into its own service (§11.5), where that split starts to earn its keep.

### 11.4 Deployment (monolith stage)

One Docker image (the Go binary), plus Postgres and Redis as separate containers in a small `docker-compose.yml`. This is what you deploy first, and it's a completely legitimate, fully-functional resume project on its own — a working real-time trading game with a correct financial ledger and autonomous market bots is a strong showcase even as a single service.

### 11.5 Later: extracting microservices (do this after §13's tests pass on the monolith)

This is the natural "part 2" of the project once the game works — pulling clean internal package boundaries out into real services, which is a much lower-risk way to end up with genuine microservices than starting distributed. The one hard rule guiding what gets split: **anything that touches money stays together.** Splitting the ledger, holdings, and pricing across services would mean a single trade needs a distributed transaction (2PC or a saga) to stay correct — exactly the complexity this whole spec is trying to avoid. "I built it as a modular monolith, then extracted these two services once the boundaries were proven, and deliberately kept the ledger together because splitting it would require distributed transactions" is a genuinely strong thing to say in an interview — probably stronger than "I built 7 microservices," and it's also just more likely to actually work.

**Suggested extraction order** (each one is optional and independent — do as many or as few as you want):

1. **Auth Service** — the easiest first cut. `/internal/auth` becomes its own process with its own `credentials` table; the main app (now "Ledger & Market Service") validates JWTs it trusts but no longer issues.
2. **Bot Service** — also low-risk to extract, since `/internal/bots` already only calls the ledger's trade-execution function; extracting it means that call becomes a real HTTP/gRPC call to the Ledger & Market Service's public trading API instead of an in-process function call. This is a good one to point at: "bots are just another API client, with zero special access."
3. **News Service** and **Leaderboard Service** — both natural read-side extractions once you want to introduce an event bus (NATS JetStream is a reasonable choice) so they react to `TradeExecuted`/`CardLaunched` events instead of polling the Ledger & Market database directly.
4. **Realtime Gateway** — split the WebSocket hub out last, once there's an event bus for it to subscribe to instead of living inside the monolith.

**Data ownership once extracted:** each service that needs persistence gets its own Postgres database (or schema). No service reaches directly into another service's database — cross-service data needs go through the event bus or a synchronous API call. Add an API Gateway in front of everything once there's more than one backend service, so the frontend keeps talking to one stable surface regardless of how many processes are behind it.

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
- **Bot-as-client tests**: since bots call the same trade-execution function real requests use (§4.5), a good chunk of load/concurrency testing is just "run the bot goroutines against a test environment and watch the invariant checks" — this doubles as both a bot-behavior test and an extra layer of exploit/concurrency testing for free.
- **Cross-service contract tests** (once you've done any extraction from §11.5): test against the event schema (`TradeExecuted`, `PriceTick`, etc.) and the API contract directly, so a change in the Ledger & Market Service that silently breaks the News or Leaderboard Service's assumptions gets caught before deploy, not after. Not applicable until you've actually extracted something.

---

## 14. Build Order (phases to hand to Opus 4.8, in sequence)

### Part A — the monolith (build this first, all the way through)

1. **Phase 0 — Schema & skeleton**: migrations (including the 30-company + NAV5 seed data from §5), domain types, an empty Go server with a health check, an empty React app shell with NavXchange/NavX branding placeholders. *(This is the phase you've already completed.)*
2. **Phase 1 — Core ledger & auth**: registration/login/JWT (`/internal/auth`), currency balance, daily +5 login reward, transaction ledger (`/internal/ledger`), the invariant tests from §13. Trading not wired up yet — just prove money can't leak and login/reward logic is correct and idempotent.
3. **Phase 2 — Pricing engine**: bonding curve math as pure functions + unit tests, inside `/internal/engine`, no API yet.
4. **Phase 3 — Trading API**: buy/sell endpoints wired to the engine + ledger, with row-locking and idempotency. Load-test this before moving on — it's the highest-risk part of the whole system.
5. **Phase 4 — Card launch flow**: creation, vesting, retained-share logic.
6. **Phase 5 — Anti-exploit layer**: fees, slippage sizing, position caps, wash-trade detection, circuit breakers.
7. **Phase 6 — Market bots**: the personas from §4.5 as background goroutines calling `/internal/ledger`'s trade-execution function directly, plus the nightly rebalancing job. This phase should also work as a stress test of Phases 3–5 — if a bot can find an exploit, so could a player.
8. **Phase 7 — News + Leaderboard**: `/internal/news` and `/internal/leaderboard`, scheduled jobs, wired into the WebSocket hub.
9. **Phase 8 — Realtime Gateway (in-process)**: WebSocket hub (`/internal/ws`) broadcasting price ticks, news, and leaderboard updates to connected clients.
10. **Phase 9 — Retention mechanics**: daily quests on top of the daily login reward already built in Phase 1.
11. **Phase 10 — Frontend build-out**: all pages against the now-stable API and WebSocket endpoints.
12. **Phase 11 — Full-system pass**: exploit simulation tests, load tests, and a full `docker-compose up` (Go binary + Postgres + Redis) before calling the monolith done. **This is a complete, working, deployable game — treat it as done-done, not "done for now."** Ship it, put it on your resume, then decide if you want to keep going.

### Part B — service extraction (optional, only after Part A is fully working and tested)

13. **Phase 12 — Extract Auth**: pull `/internal/auth` into its own service per §11.5's extraction order, with its own database, communicating via validated JWTs.
14. **Phase 13 — Extract Bot Service**: pull `/internal/bots` into its own service per §11.5, switching its internal function calls to real HTTP/gRPC calls against the Ledger & Market Service's public trading API.
15. **Phase 14 — Introduce an event bus + extract News/Leaderboard**: add NATS JetStream, publish `TradeExecuted`/`CardLaunched`/`CircuitBreakerTripped` events from the Ledger & Market Service, extract News and Leaderboard to subscribe to them instead of polling.
16. **Phase 15 — Extract the Realtime Gateway and add an API Gateway**: WebSocket hub becomes its own service subscribing to the event bus; an API Gateway goes in front of everything so the frontend's contract stays unchanged.
17. **Phase 16 — Cross-service contract tests + full multi-container `docker-compose up`** covering every extracted service.

Hand this document to Opus 4.8 one phase at a time rather than all at once — ask it to fully implement and test each phase before moving to the next. Don't start Part B until Part A's Phase 11 is genuinely passing; extracting services from something that doesn't work yet just gives you a distributed version of the same bugs.
