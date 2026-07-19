-- name: GetOrCreateUserQuestForUpdate :one
-- Upserts a fresh (progress=0) row if none exists, and locks the row
-- either way — INSERT ... ON CONFLICT DO UPDATE takes a row lock on
-- conflict just like a plain UPDATE would, so this doubles as both
-- "get or create" and "lock for the rest of this transaction."
INSERT INTO user_quests (user_id, quest_id, progress, reset_at)
VALUES ($1, $2, 0, $3)
ON CONFLICT (user_id, quest_id) DO UPDATE
SET user_id = user_quests.user_id
RETURNING *;

-- name: SetUserQuestState :one
UPDATE user_quests
SET progress = $3, completed_at = $4, reset_at = $5
WHERE user_id = $1 AND quest_id = $2
RETURNING *;

-- name: ListUserQuestsForUser :many
SELECT
    q.id AS quest_id,
    q.title,
    q.description,
    q.type,
    q.target_value,
    q.reward_currency,
    q.reset_time,
    uq.progress AS progress,
    uq.completed_at AS completed_at,
    uq.reset_at AS reset_at
FROM quests q
LEFT JOIN user_quests uq ON uq.quest_id = q.id AND uq.user_id = $1
ORDER BY q.created_at;

-- name: ListHumanUsersWithQualifyingHolding :many
-- Users eligible right now for the HOLD_CARD quest: a HUMAN user with a
-- nonzero position first bought at or before the 24h cutoff.
SELECT DISTINCT h.user_id
FROM holdings h
JOIN users u ON u.id = h.user_id
WHERE u.user_type = 'HUMAN'
  AND h.shares_owned > 0
  AND h.first_bought_at IS NOT NULL
  AND h.first_bought_at <= $1;

-- name: ResetDueUserQuests :execrows
UPDATE user_quests
SET progress = 0, completed_at = NULL, reset_at = $2
WHERE reset_at <= $1;
