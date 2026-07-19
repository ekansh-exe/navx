ALTER TABLE cards ADD COLUMN base_price DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE cards ADD COLUMN scale DOUBLE PRECISION NOT NULL DEFAULT 1;
ALTER TABLE cards ADD CONSTRAINT cards_circulating_supply_check CHECK (circulating_supply >= 0);

ALTER TABLE transactions ADD COLUMN related_transaction_id UUID REFERENCES transactions(id);

-- Anchor scale: FIXED cards to their total_supply, the UNLIMITED NAV5 card
-- (no total_supply) to a fixed reference constant.
UPDATE cards SET scale = total_supply WHERE supply_model = 'FIXED';
UPDATE cards SET scale = 1000000 WHERE supply_model = 'UNLIMITED';

-- Backfill base_price so SpotPrice(circulating_supply, {base_price, scale, 1, 1})
-- reproduces today's already-seeded current_price exactly, avoiding a price
-- discontinuity the moment trading goes live.
UPDATE cards SET base_price = current_price / SQRT((circulating_supply + 1)::float8 / scale);
