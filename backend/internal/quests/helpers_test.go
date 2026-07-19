package quests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
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

// testRedis mirrors testPool's skip-if-unavailable pattern for Redis.
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
	t.Cleanup(func() { client.Close() })
	return client
}

func uniqueUsername(prefix string) string {
	return prefix + "_" + uuid.NewString()[:8]
}

func uniqueSymbol(prefix string) string {
	return prefix + "_" + uuid.NewString()[:8]
}

func createTestUser(t *testing.T, pool *pgxpool.Pool, userType string, balance int64) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	var id uuid.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO users (username, user_type, currency_balance) VALUES ($1, $2, $3) RETURNING id`,
		uniqueUsername("quest_test"), userType, balance,
	).Scan(&id)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		pool.Exec(cleanupCtx, "DELETE FROM user_quests WHERE user_id = $1", id)
		pool.Exec(cleanupCtx, "DELETE FROM holdings WHERE user_id = $1", id)
		pool.Exec(cleanupCtx, "DELETE FROM transactions WHERE user_id = $1", id)
		pool.Exec(cleanupCtx, "DELETE FROM users WHERE id = $1", id)
	})
	return id
}

func createTestCard(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO cards (card_type, symbol, name, supply_model, circulating_supply, base_price, scale, current_price, status)
		VALUES ('SYSTEM_COMPANY', $1, $1, 'FIXED', 100000, 10, 1, 1000, 'ACTIVE')
		RETURNING id`,
		uniqueSymbol("QST"),
	).Scan(&id)
	if err != nil {
		t.Fatalf("create test card: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		pool.Exec(cleanupCtx, "DELETE FROM holdings WHERE card_id = $1", id)
		pool.Exec(cleanupCtx, "DELETE FROM transactions WHERE card_id = $1", id)
		pool.Exec(cleanupCtx, "DELETE FROM cards WHERE id = $1", id)
	})
	return id
}

// createTestHolding inserts a holding row directly with an explicit
// first_bought_at (rather than going through ExecuteTrade), so HOLD_CARD
// tests can simulate a position bought arbitrarily far in the past.
func createTestHolding(t *testing.T, pool *pgxpool.Pool, userID, cardID uuid.UUID, sharesOwned int64, firstBoughtAt time.Time) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx,
		`INSERT INTO holdings (user_id, card_id, shares_owned, avg_cost_basis, first_bought_at) VALUES ($1, $2, $3, 0, $4)`,
		userID, cardID, sharesOwned, firstBoughtAt,
	)
	if err != nil {
		t.Fatalf("create test holding: %v", err)
	}
}

func countTransactions(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, txType string) int {
	t.Helper()
	ctx := context.Background()
	var n int
	err := pool.QueryRow(ctx, `SELECT count(*) FROM transactions WHERE user_id = $1 AND type = $2`, userID, txType).Scan(&n)
	if err != nil {
		t.Fatalf("count transactions: %v", err)
	}
	return n
}

func questIDForType(t *testing.T, pool *pgxpool.Pool, questType string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	var id uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM quests WHERE type = $1`, questType).Scan(&id); err != nil {
		t.Fatalf("look up quest id for type %q: %v", questType, err)
	}
	return id
}

func getUserQuestRow(t *testing.T, pool *pgxpool.Pool, userID, questID uuid.UUID) (progress int32, completed bool, resetAt time.Time, found bool) {
	t.Helper()
	ctx := context.Background()
	var completedAt *time.Time
	err := pool.QueryRow(ctx,
		`SELECT progress, completed_at, reset_at FROM user_quests WHERE user_id = $1 AND quest_id = $2`,
		userID, questID,
	).Scan(&progress, &completedAt, &resetAt)
	if err != nil {
		return 0, false, time.Time{}, false
	}
	return progress, completedAt != nil, resetAt, true
}
