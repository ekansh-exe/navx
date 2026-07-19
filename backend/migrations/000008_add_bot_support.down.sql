DELETE FROM transactions WHERE user_id IN (SELECT id FROM users WHERE user_type = 'BOT');
DELETE FROM holdings WHERE user_id IN (SELECT id FROM users WHERE user_type = 'BOT');
DELETE FROM users WHERE user_type = 'BOT';

ALTER TABLE transactions DROP CONSTRAINT transactions_type_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_type_check
    CHECK (type IN ('BUY','SELL','CARD_LAUNCH','DAILY_REWARD','FEE'));
