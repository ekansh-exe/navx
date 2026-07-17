-- name: CreateTransaction :one
INSERT INTO transactions (
    user_id, card_id, type, shares, price_per_share,
    total_currency_delta, resulting_balance, idempotency_key, related_transaction_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetTransactionByIdempotencyKey :one
SELECT * FROM transactions WHERE idempotency_key = $1;

-- name: GetRelatedFeeTransaction :one
SELECT * FROM transactions WHERE related_transaction_id = $1 AND type = 'FEE';

-- name: SumTransactionDeltasByUser :one
SELECT COALESCE(SUM(total_currency_delta), 0)::bigint AS total
FROM transactions
WHERE user_id = $1;

-- name: ListTransactionsByUser :many
SELECT * FROM transactions WHERE user_id = $1 ORDER BY created_at ASC;
