-- name: GetCardByID :one
SELECT * FROM cards WHERE id = $1;

-- name: GetCardForUpdate :one
SELECT * FROM cards WHERE id = $1 FOR UPDATE;

-- name: UpdateCardAfterTrade :one
UPDATE cards
SET circulating_supply = $2,
    current_price = $3
WHERE id = $1
RETURNING *;
