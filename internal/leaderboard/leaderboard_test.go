package leaderboard

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/ekansh-exe/navx/internal/store/db"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect to test db: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// testRedis mirrors testPool's skip-if-unavailable pattern for the DB.
func testRedis(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("REDIS_URL")
	if addr == "" {
		addr = "redis://localhost:6379"
	}
	opts, err := redis.ParseURL(addr)
	if err != nil {
		t.Fatalf("parse REDIS_URL: %v", err)
	}
	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("redis not reachable, skipping integration test")
	}
	t.Cleanup(func() {
		client.Del(context.Background(), CacheKey)
		client.Close()
	})
	return client
}

func uniqueUsername(prefix string) string {
	return prefix + "_" + uuid.NewString()[:8]
}

func createTestHumanUser(t *testing.T, pool *pgxpool.Pool, username string, balance int64) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	var id uuid.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO users (username, user_type, currency_balance) VALUES ($1, 'HUMAN', $2) RETURNING id`,
		username, balance,
	).Scan(&id)
	if err != nil {
		t.Fatalf("create test human user: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		pool.Exec(cleanupCtx, "DELETE FROM holdings WHERE user_id = $1", id)
		pool.Exec(cleanupCtx, "DELETE FROM transactions WHERE user_id = $1", id)
		pool.Exec(cleanupCtx, "DELETE FROM users WHERE id = $1", id)
	})
	return id
}

func createTestBotUser(t *testing.T, pool *pgxpool.Pool, username string, balance int64) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	var id uuid.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO users (username, user_type, currency_balance) VALUES ($1, 'BOT', $2) RETURNING id`,
		username, balance,
	).Scan(&id)
	if err != nil {
		t.Fatalf("create test bot user: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		pool.Exec(cleanupCtx, "DELETE FROM users WHERE id = $1", id)
	})
	return id
}

func createTestCardWithPrice(t *testing.T, pool *pgxpool.Pool, symbol string, price int64) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO cards (card_type, symbol, name, supply_model, total_supply, circulating_supply, base_price, scale, current_price, status)
		VALUES ('SYSTEM_COMPANY', $1, $1, 'FIXED', 1000000, 1000, 10, 1000000, $2, 'ACTIVE')
		RETURNING id`,
		symbol, price,
	).Scan(&id)
	if err != nil {
		t.Fatalf("create test card: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		pool.Exec(cleanupCtx, "DELETE FROM holdings WHERE card_id = $1", id)
		pool.Exec(cleanupCtx, "DELETE FROM cards WHERE id = $1", id)
	})
	return id
}

func seedHolding(t *testing.T, pool *pgxpool.Pool, userID, cardID uuid.UUID, shares int64) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		"INSERT INTO holdings (user_id, card_id, shares_owned, avg_cost_basis) VALUES ($1,$2,$3,0)",
		userID, cardID, shares,
	); err != nil {
		t.Fatalf("seed holding: %v", err)
	}
}

// --- buildEntries: pure, no DB/Redis ---

func TestBuildEntries_RanksAndDiffsAgainstPrevious(t *testing.T) {
	userA, userB, userC := uuid.New(), uuid.New(), uuid.New()
	rows := []db.ComputeLeaderboardRow{
		{UserID: userA, Username: "alice", NetWorth: 500},
		{UserID: userB, Username: "bob", NetWorth: 300},
		{UserID: userC, Username: "carol", NetWorth: 100},
	}
	prev := []Entry{
		{Rank: 1, UserID: userA, Username: "alice", NetWorth: 400},
		{Rank: 2, UserID: userB, Username: "bob", NetWorth: 300},
		// carol wasn't present last refresh.
	}

	got := buildEntries(rows, prev)
	if len(got) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(got))
	}
	if got[0].Rank != 1 || got[0].UserID != userA {
		t.Fatalf("entries[0] = %+v, want rank 1 / alice", got[0])
	}
	if got[0].ChangeFromLastRefresh == nil || *got[0].ChangeFromLastRefresh != 100 {
		t.Fatalf("alice's change = %v, want 100 (500-400)", got[0].ChangeFromLastRefresh)
	}
	if got[1].ChangeFromLastRefresh == nil || *got[1].ChangeFromLastRefresh != 0 {
		t.Fatalf("bob's change = %v, want 0 (300-300)", got[1].ChangeFromLastRefresh)
	}
	if got[2].ChangeFromLastRefresh != nil {
		t.Fatalf("carol's change = %v, want nil (not present last refresh)", *got[2].ChangeFromLastRefresh)
	}
}

// --- refreshOnce / ReadCached: DB + Redis integration ---

func TestRefreshOnce_RanksCorrectlyAndExcludesBots(t *testing.T) {
	pool := testPool(t)
	redisClient := testRedis(t)
	queries := db.New(pool)
	ctx := context.Background()

	cardID := createTestCardWithPrice(t, pool, "LBTEST1_"+uuid.NewString()[:6], 100)

	// This dev DB accumulates leftover test users from other packages' test
	// runs (some with balances in the tens of millions) — use balances far
	// beyond any plausible pollution so these two are guaranteed to land in
	// ComputeLeaderboard's top-100 LIMIT regardless of what else is in there.
	const richBalance, poorBalance = int64(900_000_000_000), int64(800_000_000_000)
	richUser := createTestHumanUser(t, pool, uniqueUsername("rich"), richBalance)
	seedHolding(t, pool, richUser, cardID, 100) // +10,000 -> net worth richBalance+10,000

	poorUser := createTestHumanUser(t, pool, uniqueUsername("poor"), poorBalance)

	botUser := createTestBotUser(t, pool, uniqueUsername("bot_lb"), 999_999_999)

	refreshOnce(ctx, queries, redisClient)

	entries, err := ReadCached(ctx, redisClient)
	if err != nil {
		t.Fatalf("ReadCached: %v", err)
	}

	var richEntry, poorEntry *Entry
	for i := range entries {
		if entries[i].UserID == richUser {
			richEntry = &entries[i]
		}
		if entries[i].UserID == poorUser {
			poorEntry = &entries[i]
		}
		if entries[i].UserID == botUser {
			t.Fatalf("bot user %s appeared in the leaderboard — bots must be excluded", botUser)
		}
	}
	if richEntry == nil || poorEntry == nil {
		t.Fatal("expected both seeded human users to appear in the leaderboard")
	}
	if want := richBalance + 10_000; richEntry.NetWorth != want {
		t.Fatalf("rich user net_worth = %d, want %d (balance + 100*100 holdings)", richEntry.NetWorth, want)
	}
	if richEntry.Rank >= poorEntry.Rank {
		t.Fatalf("rich user rank (%d) should be better (lower) than poor user rank (%d)", richEntry.Rank, poorEntry.Rank)
	}
}

func TestRefreshOnce_SecondCallProducesChangeFromLastRefresh(t *testing.T) {
	pool := testPool(t)
	redisClient := testRedis(t)
	queries := db.New(pool)
	ctx := context.Background()

	// Large balance for the same reason as the ranking test above: guarantee
	// top-100 inclusion regardless of other users already in this dev DB.
	userID := createTestHumanUser(t, pool, uniqueUsername("mover"), 700_000_000_000)

	refreshOnce(ctx, queries, redisClient)
	first, err := ReadCached(ctx, redisClient)
	if err != nil {
		t.Fatalf("ReadCached (first): %v", err)
	}
	for _, e := range first {
		if e.UserID == userID && e.ChangeFromLastRefresh != nil {
			t.Fatalf("expected nil change on the very first appearance, got %v", *e.ChangeFromLastRefresh)
		}
	}

	if _, err := pool.Exec(ctx, "UPDATE users SET currency_balance = currency_balance + 5000 WHERE id = $1", userID); err != nil {
		t.Fatalf("bump balance: %v", err)
	}

	refreshOnce(ctx, queries, redisClient)
	second, err := ReadCached(ctx, redisClient)
	if err != nil {
		t.Fatalf("ReadCached (second): %v", err)
	}
	found := false
	for _, e := range second {
		if e.UserID == userID {
			found = true
			if e.ChangeFromLastRefresh == nil || *e.ChangeFromLastRefresh != 5000 {
				t.Fatalf("change_from_last_refresh = %v, want 5000", e.ChangeFromLastRefresh)
			}
		}
	}
	if !found {
		t.Fatal("user missing from second refresh")
	}
}

func TestReadCached_EmptyWhenKeyAbsent(t *testing.T) {
	redisClient := testRedis(t)
	ctx := context.Background()
	redisClient.Del(ctx, CacheKey)

	entries, err := ReadCached(ctx, redisClient)
	if err != nil {
		t.Fatalf("ReadCached: %v", err)
	}
	if entries == nil || len(entries) != 0 {
		t.Fatalf("entries = %v, want an empty (non-nil) slice", entries)
	}
}
