-- The news-reactive persona (§4.5's last bullet) sells affected-sector
-- cards on negative headlines (flood/drought/war/embargo), but SELL
-- requires pre-existing holdings — there's no short-selling primitive
-- (ledger.ErrInsufficientShares). Without a starting position, a sector
-- the bot has never bought into (e.g. FOOD, since it only ever buys on
-- DISCOVERY-affected sectors) could never actually see a negative-news
-- sell. Seed a modest reserve — 3% of each SYSTEM_COMPANY card's
-- circulating supply — for both news-reactive bot accounts, comfortably
-- above any single trade's size (sizeTrade tops out at 0.8% of supply) so
-- it depletes gradually across many reactions rather than in one trade.
-- ON CONFLICT DO NOTHING: a news-reactive bot may already hold a card by the
-- time this seed runs (organic trading before migration, or a re-run against
-- a live/partially-migrated environment) — skip rather than fail so this
-- stays a safe top-up, not a hard reset of existing positions.
INSERT INTO holdings (user_id, card_id, shares_owned, avg_cost_basis)
SELECT u.id, c.id, ROUND(c.circulating_supply * 0.03)::BIGINT, c.current_price
FROM users u
CROSS JOIN cards c
WHERE u.username IN ('bot_newsreactive_1', 'bot_newsreactive_2')
  AND c.card_type = 'SYSTEM_COMPANY'
ON CONFLICT (user_id, card_id) DO NOTHING;
