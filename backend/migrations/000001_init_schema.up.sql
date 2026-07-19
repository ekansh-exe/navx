CREATE EXTENSION IF NOT EXISTS pgcrypto;

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
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','DELISTED','FROZEN')),
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
    type TEXT NOT NULL CHECK (type IN ('BUY','SELL','CARD_LAUNCH','DAILY_REWARD','FEE')),
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
