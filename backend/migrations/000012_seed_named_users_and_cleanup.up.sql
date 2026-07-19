-- Two independent things, both about making the leaderboard look real:
--
-- 1) Delete leftover test-fixture accounts. Every integration test in this
--    repo (internal/ledger, internal/leaderboard, internal/bots, internal/api,
--    internal/quests) runs its testPool()/testRedis() against the same
--    DATABASE_URL the dev server uses — there is no separate test database —
--    so every local `go test ./...` run has been inserting real HUMAN rows
--    into this database. That's the actual source of "everyone's net worth
--    looks the same" on the leaderboard (many converge on identical values
--    like 64154043 units), not bot balances: bots are already excluded from
--    ComputeLeaderboard (WHERE user_type = 'HUMAN').
--
--    The pattern below matches every username prefix actually used by
--    uniqueUsername()/literal test usernames across the test suite (verified
--    against this database: matches exactly 1,319 of 1,323 current HUMAN
--    rows). It deliberately does NOT match demo_trader, demo_user,
--    phase6test, or curluser_* — those don't fit any test-suite naming
--    pattern and appear to be real manual accounts, so an explicit exclusion
--    is added as a second line of defense even though the pattern alone
--    already spares them.
--
--    This is a one-time cleanup, not a repeatable migration step in the usual
--    sense — running it again after more test runs will leave newer fixture
--    rows behind. There's no reasonable down migration for this half: the
--    deleted rows aren't coming back, and re-inserting them would just be
--    recreating test pollution on purpose.
--
-- 2) Seed three named tribute/flavor leaderboard accounts (Bambani, Musky,
--    Windows) with fixed net worth. currency_balance is in the smallest unit
--    (frontend's formatCurrency divides by 100), so the displayed dollar
--    figures map to:
--      Bambani  $2,500,000 -> 250,000,000
--      Musky    $2,000,000 -> 200,000,000
--      Windows  $1,500,000 -> 150,000,000
--    password_hash is left NULL, same as bot accounts — these are static
--    display entries, not accounts meant to be logged into. They hold no
--    cards, so their net worth is exactly their currency_balance with no
--    holdings math involved.

DO $$
DECLARE
    test_user_ids UUID[];
BEGIN
    SELECT array_agg(id) INTO test_user_ids
    FROM users
    WHERE user_type = 'HUMAN'
      AND username NOT IN ('demo_trader', 'demo_user', 'phase6test')
      AND username !~ '^curluser_'
      AND username ~ '^(load|mix|mover|patient|whale|buyer|bystander|washer|seller|dup|greedy|invariant|launcher|poor|rich|replay|concurrent|vestcreator|cbmover|cbbystander|feeverify|washverify|usera|userb|trade|tester|lbtester)_[0-9a-zA-Z]+$|^u_[0-9a-f]+$|^leaderboard_test_[0-9]+$';

    IF test_user_ids IS NOT NULL THEN
        -- A test user that happened to launch a card (CARD_LAUNCH) is the
        -- creator on record; clear that reference first so the FK on
        -- cards.creator_user_id doesn't block deleting the user below. The
        -- card itself (and every human's holdings in it) is untouched.
        UPDATE cards SET creator_user_id = NULL WHERE creator_user_id = ANY(test_user_ids);

        DELETE FROM user_quests WHERE user_id = ANY(test_user_ids);
        DELETE FROM transactions WHERE user_id = ANY(test_user_ids);
        DELETE FROM holdings WHERE user_id = ANY(test_user_ids);
        DELETE FROM users WHERE id = ANY(test_user_ids);
    END IF;
END $$;

INSERT INTO users (username, user_type, password_hash, currency_balance) VALUES
    ('Bambani', 'HUMAN', NULL, 250000000),
    ('Musky', 'HUMAN', NULL, 200000000),
    ('Windows', 'HUMAN', NULL, 150000000)
ON CONFLICT (username) DO NOTHING;
