-- Revert bot bankrolls to the original 1,000,000-unit seed (migration 000008),
-- but only for bots still sitting at exactly the raised default, so a bot that
-- has since traded away from it is left untouched.
UPDATE users
SET currency_balance = 1000000
WHERE user_type = 'BOT'
  AND currency_balance = 50000000;
