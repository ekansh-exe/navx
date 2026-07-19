package ledger

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ekansh-exe/navx/internal/store/db"
)

// createTestBotUser inserts a BOT-type test user via raw SQL — CreateUser
// (users.sql) only ever creates HUMAN users (no user_type param), the same
// reason createTestCard in ledger_test.go bypasses sqlc for card creation.
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
		pool.Exec(cleanupCtx, "DELETE FROM transactions WHERE user_id = $1", id)
		pool.Exec(cleanupCtx, "DELETE FROM users WHERE id = $1", id)
	})
	return id
}

func TestRebalanceBotBalance_ResetsToTarget(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	q := db.New(pool)
	ctx := context.Background()

	// A bot that traded its way down to a much smaller balance.
	userID := createTestBotUser(t, pool, uniqueUsername("bot_low"), 250_000)

	user, err := l.RebalanceBotBalance(ctx, userID)
	if err != nil {
		t.Fatalf("RebalanceBotBalance: %v", err)
	}
	if user.CurrencyBalance != BotStartingBalance {
		t.Fatalf("balance = %d, want %d", user.CurrencyBalance, BotStartingBalance)
	}

	rows, err := q.ListTransactionsByUser(ctx, userID)
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected exactly 1 transaction row, got %d", len(rows))
	}
	if rows[0].Type != "REBALANCE" {
		t.Fatalf("transaction type = %q, want REBALANCE", rows[0].Type)
	}
	if want := BotStartingBalance - 250_000; rows[0].TotalCurrencyDelta != want {
		t.Fatalf("total_currency_delta = %d, want %d", rows[0].TotalCurrencyDelta, want)
	}
	if rows[0].ResultingBalance != BotStartingBalance {
		t.Fatalf("resulting_balance = %d, want %d", rows[0].ResultingBalance, BotStartingBalance)
	}
}

func TestRebalanceBotBalance_AlreadyAtTargetIsNoOp(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	q := db.New(pool)
	ctx := context.Background()

	userID := createTestBotUser(t, pool, uniqueUsername("bot_ok"), BotStartingBalance)

	user, err := l.RebalanceBotBalance(ctx, userID)
	if err != nil {
		t.Fatalf("RebalanceBotBalance: %v", err)
	}
	if user.CurrencyBalance != BotStartingBalance {
		t.Fatalf("balance = %d, want %d", user.CurrencyBalance, BotStartingBalance)
	}

	rows, err := q.ListTransactionsByUser(ctx, userID)
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected no transaction written for a no-op rebalance, got %d", len(rows))
	}
}
