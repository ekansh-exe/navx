-- Phase 9 (§7): daily quests on top of the daily login reward from Phase 1.

ALTER TABLE transactions DROP CONSTRAINT transactions_type_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_type_check
    CHECK (type IN ('BUY','SELL','CARD_LAUNCH','DAILY_REWARD','FEE','REBALANCE','QUEST_REWARD'));

-- first_bought_at backs the HOLD_CARD quest ("hold any card for 24h"): set
-- once, the first time a holding goes from nonexistent/zero to a nonzero
-- position, and never overwritten afterward — see UpsertHolding below.
ALTER TABLE holdings ADD COLUMN first_bought_at TIMESTAMPTZ;

CREATE TABLE quests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL CHECK (type IN ('MAKE_TRADES', 'HOLD_CARD', 'REACH_RANK')),
    target_value INT NOT NULL,
    reward_currency INT NOT NULL,
    reset_time TEXT NOT NULL CHECK (reset_time IN ('DAILY', 'WEEKLY')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user_quests (
    user_id UUID NOT NULL REFERENCES users(id),
    quest_id UUID NOT NULL REFERENCES quests(id),
    progress INT NOT NULL DEFAULT 0,
    completed_at TIMESTAMPTZ,
    reset_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, quest_id)
);

INSERT INTO quests (title, description, type, target_value, reward_currency, reset_time) VALUES
    ('Make 3 trades today', 'Execute 3 buy or sell trades today.', 'MAKE_TRADES', 3, 100, 'DAILY'),
    ('Hold any card for 24 hours', 'Keep a nonzero position in any card for a full 24 hours.', 'HOLD_CARD', 1, 150, 'DAILY'),
    ('Reach rank 50 on the leaderboard', 'Reach rank 50 or better on the leaderboard.', 'REACH_RANK', 1, 200, 'DAILY');
