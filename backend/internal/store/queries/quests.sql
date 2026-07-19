-- name: ListQuests :many
SELECT * FROM quests ORDER BY created_at;

-- name: GetQuestByType :one
SELECT * FROM quests WHERE type = $1;
