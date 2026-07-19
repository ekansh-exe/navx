ALTER TABLE transactions DROP CONSTRAINT transactions_type_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_type_check
    CHECK (type IN ('BUY','SELL','CARD_LAUNCH','DAILY_REWARD','FEE','REBALANCE'));

-- Bot accounts (§4.5): real users.user_type='BOT' rows that trade through the
-- exact same ExecuteTrade path as humans. password_hash is NULL — bots never
-- log in. currency_balance matches ledger.BotStartingBalance, the target the
-- nightly rebalance job resets toward.
INSERT INTO users (username, user_type, password_hash, currency_balance) VALUES
    ('bot_momentum_1', 'BOT', NULL, 1000000),
    ('bot_momentum_2', 'BOT', NULL, 1000000),
    ('bot_contrarian_1', 'BOT', NULL, 1000000),
    ('bot_contrarian_2', 'BOT', NULL, 1000000),
    ('bot_randomwalker_1', 'BOT', NULL, 1000000),
    ('bot_randomwalker_2', 'BOT', NULL, 1000000),
    ('bot_newsreactive_1', 'BOT', NULL, 1000000),
    ('bot_newsreactive_2', 'BOT', NULL, 1000000),
    ('bot_indextracker_1', 'BOT', NULL, 1000000),
    ('bot_indextracker_2', 'BOT', NULL, 1000000);
