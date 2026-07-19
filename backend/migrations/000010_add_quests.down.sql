DROP TABLE IF EXISTS user_quests;
DROP TABLE IF EXISTS quests;

ALTER TABLE holdings DROP COLUMN IF EXISTS first_bought_at;

ALTER TABLE transactions DROP CONSTRAINT transactions_type_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_type_check
    CHECK (type IN ('BUY','SELL','CARD_LAUNCH','DAILY_REWARD','FEE','REBALANCE'));
