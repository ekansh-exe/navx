package bots

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/engine"
)

// These fixtures mirror internal/ledger/ledger_test.go's exactly (testPool,
// createTestCard's current-price-via-engine.SpotPrice approach) since
// package-private test helpers can't be shared across packages.

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

func uniqueUsername(prefix string) string {
	return prefix + "_" + uuid.NewString()[:8]
}

func uniqueSymbol(prefix string) string {
	return prefix + "_" + uuid.NewString()[:8]
}

func ptrInt64(v int64) *int64 { return &v }

// createTestBotUser inserts a BOT-type test user via raw SQL — CreateUser
// (users.sql) only ever creates HUMAN users.
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
		pool.Exec(cleanupCtx, "DELETE FROM holdings WHERE user_id = $1", id)
		pool.Exec(cleanupCtx, "DELETE FROM transactions WHERE user_id = $1", id)
		pool.Exec(cleanupCtx, "DELETE FROM users WHERE id = $1", id)
	})
	return id
}

// createTestCard inserts a dedicated, isolated test card — current_price is
// computed from the same curve params the card is given (matching the
// production invariant current_price == SpotPrice(circulating_supply,
// curve params)), same as ledger_test.go's helper.
func createTestCard(t *testing.T, pool *pgxpool.Pool, symbol string, supplyModel domain.SupplyModel, totalSupply *int64, circulatingSupply int64, basePrice, scale float64, status domain.CardStatus) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	currentPrice, err := engine.SpotPrice(circulatingSupply, engine.CurveParams{BasePrice: basePrice, Scale: scale, DemandModifier: 1, DriftFactor: 1})
	if err != nil {
		t.Fatalf("compute starting price: %v", err)
	}
	var id uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO cards (card_type, symbol, name, supply_model, total_supply, circulating_supply, base_price, scale, current_price, status)
		VALUES ('SYSTEM_COMPANY', $1, $1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		symbol, string(supplyModel), totalSupply, circulatingSupply, basePrice, scale, currentPrice, string(status),
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
