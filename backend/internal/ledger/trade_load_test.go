package ledger

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/store/db"
)

// queryTotalDeltaSum sums total_currency_delta across every transaction
// (trade rows AND fee rows) for the given users, straight from the DB. This
// backs an aggregate cross-check independent of the per-user assertInvariant
// loop: since there's no modeled AMM reserve account, a buy's cost (not
// just its fee) leaves the tracked set of user balances with no offsetting
// credit anywhere in the system — the market itself isn't a real account —
// so the only valid "total" invariant is the literal sum of every recorded
// delta, not "fees are the only sink" (that assumption is wrong and was
// caught by this test failing on first run).
func queryTotalDeltaSum(t *testing.T, pool *pgxpool.Pool, userIDs []uuid.UUID) int64 {
	t.Helper()
	var sum int64
	if err := pool.QueryRow(context.Background(),
		"SELECT COALESCE(SUM(total_currency_delta), 0) FROM transactions WHERE user_id = ANY($1)",
		userIDs,
	).Scan(&sum); err != nil {
		t.Fatalf("sum deltas: %v", err)
	}
	return sum
}

// TestExecuteTrade_ConcurrentBuysReconcile is §13's explicit "spin up many
// simultaneous buy/sell requests against the same card and assert the final
// circulating_supply and all user balances reconcile exactly against the
// transaction ledger" test — the highest-priority test in this phase, given
// the spec calls trade execution "the highest-risk part of the whole system."
func TestExecuteTrade_ConcurrentBuysReconcile(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	q := db.New(pool)
	ctx := context.Background()

	const startSupply, basePrice, scale = int64(100_000), 5.0, 1_000_000.0
	cardID := createTestCard(t, pool, uniqueSymbol("LOAD"), domain.SupplyModelUnlimited, nil, startSupply, basePrice, scale, domain.CardStatusActive)

	const numUsers = 40
	const sharesPerBuy = int64(50)
	userIDs := make([]uuid.UUID, numUsers)
	for i := range userIDs {
		userIDs[i] = createTestUser(t, pool, uniqueUsername("load"))
	}

	var wg sync.WaitGroup
	errs := make([]error, numUsers)
	wg.Add(numUsers)
	for i, userID := range userIDs {
		go func(i int, userID uuid.UUID) {
			defer wg.Done()
			_, err := l.ExecuteTrade(ctx, TradeParams{
				UserID: userID, CardID: cardID, Type: domain.TransactionTypeBuy,
				Shares: sharesPerBuy, IdempotencyKey: uuid.NewString(),
			})
			errs[i] = err
		}(i, userID)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("user %d: ExecuteTrade: %v", i, err)
		}
	}

	card, err := q.GetCardByID(ctx, cardID)
	if err != nil {
		t.Fatalf("get card: %v", err)
	}
	wantSupply := startSupply + int64(numUsers)*sharesPerBuy
	if card.CirculatingSupply != wantSupply {
		t.Fatalf("circulating supply = %d, want %d", card.CirculatingSupply, wantSupply)
	}

	var totalFinalBalance int64
	for _, userID := range userIDs {
		user, err := q.GetUserByID(ctx, userID)
		if err != nil {
			t.Fatalf("get user: %v", err)
		}
		assertInvariant(t, q, userID, user.CurrencyBalance)
		totalFinalBalance += user.CurrencyBalance
	}

	wantTotalBalance := int64(numUsers)*initialBalance + queryTotalDeltaSum(t, pool, userIDs)
	if totalFinalBalance != wantTotalBalance {
		t.Fatalf("sum of final balances = %d, want %d (initial + sum of every recorded delta)", totalFinalBalance, wantTotalBalance)
	}
}

// TestExecuteTrade_ConcurrentMixedBuySellReconcile mixes concurrent buys and
// sells against the same card (after sequentially seeding every user with a
// starting position to sell from) — verifying the lock ordering and
// holdings bookkeeping reconcile correctly under two-directional contention,
// not just same-direction contention.
func TestExecuteTrade_ConcurrentMixedBuySellReconcile(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	q := db.New(pool)
	ctx := context.Background()

	const startSupply, basePrice, scale = int64(100_000), 5.0, 1_000_000.0
	cardID := createTestCard(t, pool, uniqueSymbol("MIXLOAD"), domain.SupplyModelUnlimited, nil, startSupply, basePrice, scale, domain.CardStatusActive)

	const numUsers = 40
	const seedShares = int64(200)
	userIDs := make([]uuid.UUID, numUsers)
	for i := range userIDs {
		userIDs[i] = createTestUser(t, pool, uniqueUsername("mix"))
		if _, err := l.ExecuteTrade(ctx, TradeParams{
			UserID: userIDs[i], CardID: cardID, Type: domain.TransactionTypeBuy,
			Shares: seedShares, IdempotencyKey: uuid.NewString(),
		}); err != nil {
			t.Fatalf("seed buy for user %d: %v", i, err)
		}
	}
	supplyAfterSeed := startSupply + int64(numUsers)*seedShares

	const tradeShares = int64(30)
	var wg sync.WaitGroup
	errs := make([]error, numUsers)
	wg.Add(numUsers)
	for i, userID := range userIDs {
		go func(i int, userID uuid.UUID) {
			defer wg.Done()
			tradeType := domain.TransactionTypeBuy
			if i%2 == 0 {
				tradeType = domain.TransactionTypeSell
			}
			_, err := l.ExecuteTrade(ctx, TradeParams{
				UserID: userID, CardID: cardID, Type: tradeType,
				Shares: tradeShares, IdempotencyKey: uuid.NewString(),
			})
			errs[i] = err
		}(i, userID)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("user %d: ExecuteTrade: %v", i, err)
		}
	}

	card, err := q.GetCardByID(ctx, cardID)
	if err != nil {
		t.Fatalf("get card: %v", err)
	}
	// Equal split of buys/sells (numUsers even) at equal size nets to zero
	// additional supply change — exact integer arithmetic, no rounding.
	if card.CirculatingSupply != supplyAfterSeed {
		t.Fatalf("circulating supply = %d, want %d (equal buys/sells should net to zero change)", card.CirculatingSupply, supplyAfterSeed)
	}

	var totalFinalBalance int64
	for _, userID := range userIDs {
		user, err := q.GetUserByID(ctx, userID)
		if err != nil {
			t.Fatalf("get user: %v", err)
		}
		assertInvariant(t, q, userID, user.CurrencyBalance)
		totalFinalBalance += user.CurrencyBalance
	}

	wantTotalBalance := int64(numUsers)*initialBalance + queryTotalDeltaSum(t, pool, userIDs)
	if totalFinalBalance != wantTotalBalance {
		t.Fatalf("sum of final balances = %d, want %d (initial + sum of every recorded delta)", totalFinalBalance, wantTotalBalance)
	}
}
