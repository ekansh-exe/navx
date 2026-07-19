ALTER TABLE transactions DROP COLUMN related_transaction_id;
ALTER TABLE cards DROP CONSTRAINT cards_circulating_supply_check;
ALTER TABLE cards DROP COLUMN scale;
ALTER TABLE cards DROP COLUMN base_price;
