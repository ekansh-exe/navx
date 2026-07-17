-- name: GetHoldingForUpdate :one
SELECT * FROM holdings WHERE user_id = $1 AND card_id = $2 FOR UPDATE;

-- name: UpsertHolding :one
INSERT INTO holdings (user_id, card_id, shares_owned, avg_cost_basis)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, card_id) DO UPDATE
SET shares_owned = EXCLUDED.shares_owned,
    avg_cost_basis = EXCLUDED.avg_cost_basis
RETURNING *;
