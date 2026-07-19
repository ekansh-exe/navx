DELETE FROM holdings
USING users u
WHERE holdings.user_id = u.id
  AND u.username IN ('bot_newsreactive_1', 'bot_newsreactive_2');
