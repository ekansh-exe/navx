package ledger

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/engine"
	"github.com/ekansh-exe/navx/internal/store/db"
)

// initialBalance mirrors the users.currency_balance DEFAULT in migration
// 000001 — the starting balance isn't itself a ledger transaction (there's
// no transaction type for it in §2's enum), so the invariant this package
// enforces is: balance == initialBalance + sum(transaction deltas).
const initialBalance int64 = 100000

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

func createTestUser(t *testing.T, pool *pgxpool.Pool, username string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	q := db.New(pool)
	hash := "test-hash"
	user, err := q.CreateUser(ctx, db.CreateUserParams{
		Username:     username,
		PasswordHash: &hash,
	})
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM transactions WHERE user_id = $1", user.ID)
		pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", user.ID)
	})
	return user.ID
}

// uniqueUsername stays within the 3-32 char bound auth.validateUsername
// enforces (uuid.NewString() alone is 36 chars).
func uniqueUsername(prefix string) string {
	return prefix + "_" + uuid.NewString()[:8]
}

func uniqueSymbol(prefix string) string {
	return prefix + "_" + uuid.NewString()[:8]
}

// createTestCard inserts a dedicated, isolated test card (never a shared
// seeded company) so trade tests can't interfere with each other or with
// other tests running concurrently. current_price is computed from the
// same curve params the card is given (matching the production invariant
// that current_price == SpotPrice(circulating_supply, curve params) —
// migration 000005 backfills exactly this for seeded companies) rather than
// a disconnected placeholder, since Phase 5's circuit breaker now relies on
// current_price being a real pre-trade baseline.
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

func ptrInt64(v int64) *int64 { return &v }

// assertInvariant checks §2's non-negotiable invariant: balance always
// equals initialBalance plus the sum of the user's transaction deltas, and
// is never negative.
func assertInvariant(t *testing.T, q *db.Queries, userID uuid.UUID, balance int64) {
	t.Helper()
	sum, err := q.SumTransactionDeltasByUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("sum transaction deltas: %v", err)
	}
	if want := initialBalance + sum; balance != want {
		t.Fatalf("invariant violated: currency_balance=%d, but initialBalance+sum(deltas)=%d", balance, want)
	}
	if balance < 0 {
		t.Fatalf("invariant violated: balance is negative: %d", balance)
	}
}

// TestNextLoginState covers the streak/grant decision logic directly with
// arbitrary simulated dates — real UTC "today" can't be fast-forwarded in a
// database-backed test, so the multi-day streak behavior (§7) is verified
// here as a pure function instead.
func TestNextLoginState(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	yesterday := now.AddDate(0, 0, -1)
	twoDaysAgo := now.AddDate(0, 0, -2)
	sameDayEarlier := time.Date(2026, 7, 15, 0, 1, 0, 0, time.UTC)
	tomorrow := now.AddDate(0, 0, 1) // clock-skew edge case

	tests := []struct {
		name           string
		lastLoginAt    *time.Time
		currentStreak  int32
		wantStreak     int32
		wantAlreadyDid bool
	}{
		{"never logged in before", nil, 0, 1, false},
		{"first login today, already granted", &sameDayEarlier, 1, 1, true},
		{"consecutive day continues streak", &yesterday, 3, 4, false},
		{"gap of two days resets streak", &twoDaysAgo, 5, 1, false},
		{"clock skew: last login appears to be tomorrow resets streak", &tomorrow, 5, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			streak, alreadyGranted := nextLoginState(tt.lastLoginAt, tt.currentStreak, now)
			if streak != tt.wantStreak {
				t.Errorf("streak = %d, want %d", streak, tt.wantStreak)
			}
			if alreadyGranted != tt.wantAlreadyDid {
				t.Errorf("alreadyGranted = %v, want %v", alreadyGranted, tt.wantAlreadyDid)
			}
		})
	}
}

// TestGrantDailyReward_SingleGrantAndSameDayIdempotency exercises the real
// DB-backed path: a fresh grant updates balance/ledger correctly, and a
// same-day repeat is a true no-op (§7's idempotency requirement).
func TestGrantDailyReward_SingleGrantAndSameDayIdempotency(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	q := db.New(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool, uniqueUsername("invariant"))

	user, granted, err := l.GrantDailyReward(ctx, userID)
	if err != nil {
		t.Fatalf("grant daily reward: %v", err)
	}
	if !granted {
		t.Fatal("expected the first grant to succeed")
	}
	if want := initialBalance + DailyRewardAmount; user.CurrencyBalance != want {
		t.Fatalf("balance = %d, want %d", user.CurrencyBalance, want)
	}
	if user.LoginStreakCount != 1 {
		t.Fatalf("streak = %d, want 1", user.LoginStreakCount)
	}
	assertInvariant(t, q, userID, user.CurrencyBalance)

	// Repeat calls the same day must be true no-ops.
	for i := 0; i < 3; i++ {
		again, granted, err := l.GrantDailyReward(ctx, userID)
		if err != nil {
			t.Fatalf("repeat %d: grant daily reward: %v", i, err)
		}
		if granted {
			t.Fatalf("repeat %d: expected same-day repeat to not grant", i)
		}
		if again.CurrencyBalance != user.CurrencyBalance {
			t.Fatalf("repeat %d: balance changed on idempotent repeat: %d -> %d", i, user.CurrencyBalance, again.CurrencyBalance)
		}
		assertInvariant(t, q, userID, again.CurrencyBalance)
	}
}

// TestGrantDailyReward_ConcurrentCallsGrantExactlyOnce fires many
// simultaneous GrantDailyReward calls for one fresh user and asserts the row
// lock (§4.2) serializes them into exactly one grant, not one per goroutine.
func TestGrantDailyReward_ConcurrentCallsGrantExactlyOnce(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool, uniqueUsername("concurrent"))

	const n = 20
	var wg sync.WaitGroup
	var mu sync.Mutex
	grantedCount := 0
	errs := make([]error, n)

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			_, granted, err := l.GrantDailyReward(ctx, userID)
			if err != nil {
				errs[i] = err
				return
			}
			if granted {
				mu.Lock()
				grantedCount++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}

	if grantedCount != 1 {
		t.Fatalf("expected exactly 1 grant among %d concurrent calls, got %d", n, grantedCount)
	}

	q := db.New(pool)
	user, err := q.GetUserByID(ctx, userID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if want := initialBalance + DailyRewardAmount; user.CurrencyBalance != want {
		t.Fatalf("balance = %d, want %d (exactly one grant)", user.CurrencyBalance, want)
	}
	assertInvariant(t, q, userID, user.CurrencyBalance)
}
