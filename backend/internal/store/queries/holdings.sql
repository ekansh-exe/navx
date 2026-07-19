-- name: GetHoldingForUpdate :one
SELECT * FROM holdings WHERE user_id = $1 AND card_id = $2 FOR UPDATE;

-- name: ListHoldingsByUser :many
SELECT * FROM holdings WHERE user_id = $1;

-- name: UpsertHolding :one
-- first_bought_at is set once, the first time this holding's shares_owned
-- goes from nonexistent/zero to nonzero, and never overwritten afterward —
-- even across a full sell-to-zero and later rebuy — backing the HOLD_CARD
-- quest's "first buy" semantics (§7).
INSERT INTO holdings (user_id, card_id, shares_owned, avg_cost_basis, first_bought_at)
VALUES ($1, $2, $3, $4, CASE WHEN $3::BIGINT > 0 THEN now() ELSE NULL END)
ON CONFLICT (user_id, card_id) DO UPDATE
SET shares_owned = EXCLUDED.shares_owned,
    avg_cost_basis = EXCLUDED.avg_cost_basis,
    first_bought_at = COALESCE(holdings.first_bought_at, CASE WHEN EXCLUDED.shares_owned > 0 THEN now() ELSE NULL END)
RETURNING *;
