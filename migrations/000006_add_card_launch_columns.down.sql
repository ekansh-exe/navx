ALTER TABLE cards DROP CONSTRAINT cards_creator_retained_shares_sold_check;
ALTER TABLE cards DROP COLUMN creator_retained_shares_sold;
ALTER TABLE cards DROP COLUMN image_url;
ALTER TABLE cards DROP COLUMN description;
