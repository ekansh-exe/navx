-- Raise bot bankrolls to match the increased ledger.BotStartingBalance.
--
-- The original seed (migration 000008) funded each bot with 1,000,000 units.
-- Against the sqrt bonding curve that is far too little to move any seeded
-- card's rounded integer price: a ~1% slice of a million-share card priced in
-- the thousands costs many times that balance, so bots skipped nearly every
-- buy with "insufficient balance" and prices never moved. This resets every
-- bot to the new target so their trades can actually shift prices.
--
-- Applied only to bots still sitting at exactly the old default, so it never
-- clobbers a bot that has since traded away from it (the rebalance job will
-- carry those to the new target on its next run regardless). Bots are excluded
-- from the leaderboard (§8 ranks HUMAN users only), so this changes no player
-- standings.
UPDATE users
SET currency_balance = 50000000
WHERE user_type = 'BOT'
  AND currency_balance = 1000000;
