-- name: CreateNewsEvent :one
INSERT INTO news_events (headline, body, category, related_card_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListNewsEvents :many
SELECT * FROM news_events ORDER BY created_at DESC LIMIT $1 OFFSET $2;
