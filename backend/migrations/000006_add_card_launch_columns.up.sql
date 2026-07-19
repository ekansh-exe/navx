ALTER TABLE cards ADD COLUMN description TEXT;
ALTER TABLE cards ADD COLUMN image_url TEXT;
ALTER TABLE cards ADD COLUMN creator_retained_shares_sold BIGINT NOT NULL DEFAULT 0;
ALTER TABLE cards ADD CONSTRAINT cards_creator_retained_shares_sold_check
    CHECK (creator_retained_shares_sold >= 0 AND creator_retained_shares_sold <= creator_retained_shares);
