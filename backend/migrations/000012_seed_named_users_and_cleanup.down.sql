-- Only reverses the named-user seed (half 2 of the up migration). The
-- test-fixture cleanup (half 1) has no meaningful down: those rows were
-- pollution, not data worth restoring.
DELETE FROM users WHERE username IN ('Bambani', 'Musky', 'Windows') AND user_type = 'HUMAN';
