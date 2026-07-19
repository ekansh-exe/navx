-- name: CreateUser :one
INSERT INTO users (username, password_hash)
VALUES ($1, $2)
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: ListUsersByType :many
SELECT * FROM users WHERE user_type = $1 ORDER BY username;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserForUpdate :one
SELECT * FROM users WHERE id = $1 FOR UPDATE;

-- name: ApplyDailyReward :one
UPDATE users
SET currency_balance = currency_balance + $2,
    last_login_at = $3,
    login_streak_count = $4
WHERE id = $1
RETURNING *;

-- name: ApplyBalanceDelta :one
UPDATE users
SET currency_balance = currency_balance + $2
WHERE id = $1
RETURNING *;
