-- name: ComputeLeaderboard :many
SELECT u.id AS user_id, u.username,
       CAST(u.currency_balance + COALESCE(SUM(h.shares_owned * c.current_price), 0) AS BIGINT) AS net_worth
FROM users u
LEFT JOIN holdings h ON h.user_id = u.id
LEFT JOIN cards c ON c.id = h.card_id
WHERE u.user_type = 'HUMAN'
GROUP BY u.id, u.username, u.currency_balance
ORDER BY net_worth DESC
LIMIT 100;
