DELETE FROM index_components WHERE index_card_id IN (SELECT id FROM cards WHERE symbol = 'NAV5');
DELETE FROM cards WHERE card_type IN ('SYSTEM_COMPANY', 'INDEX');
