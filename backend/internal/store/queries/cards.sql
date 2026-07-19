-- name: GetCardByID :one
SELECT * FROM cards WHERE id = $1;

-- name: ListActiveCards :many
SELECT * FROM cards WHERE status = 'ACTIVE' ORDER BY symbol;

-- name: GetCardForUpdate :one
SELECT * FROM cards WHERE id = $1 FOR UPDATE;

-- name: UpdateCardAfterTrade :one
UPDATE cards
SET circulating_supply = $2,
    current_price = $3,
    creator_retained_shares_sold = creator_retained_shares_sold + $4,
    circuit_breaker_window_started_at = $5,
    circuit_breaker_window_start_price = $6,
    circuit_breaker_halted_until = $7
WHERE id = $1
RETURNING *;

-- name: CreateUserCard :one
INSERT INTO cards (
    card_type, creator_user_id, symbol, name, description, image_url,
    supply_model, total_supply, circulating_supply, creator_retained_shares,
    base_price, scale, current_price, status
)
VALUES (
    'USER_CREATED', $1, $2, $3, $4, $5,
    $6, $7, $8, $8,
    $9, $10, $11, 'ACTIVE'
)
RETURNING *;
