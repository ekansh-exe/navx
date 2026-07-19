package ledger

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/engine"
	"github.com/ekansh-exe/navx/internal/store/db"
)

func TestLaunchCard_HappyPath(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	q := db.New(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool, uniqueUsername("launcher"))
	symbol := uniqueSymbol("LNCH")
	desc := "a test card"

	const totalSupply = int64(1_000_000)
	const retainedPercent = 0.25

	result, err := l.LaunchCard(ctx, LaunchCardParams{
		CreatorUserID:   userID,
		Symbol:          symbol,
		Name:            "Test Launch Co.",
		Description:     &desc,
		TotalSupply:     totalSupply,
		RetainedPercent: retainedPercent,
		IdempotencyKey:  uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("LaunchCard: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM holdings WHERE card_id = $1", result.Card.ID)
		pool.Exec(context.Background(), "DELETE FROM transactions WHERE card_id = $1", result.Card.ID)
		pool.Exec(context.Background(), "DELETE FROM cards WHERE id = $1", result.Card.ID)
	})

	wantRetained := int64(totalSupply * 25 / 100)
	if result.Card.CreatorRetainedShares != wantRetained {
		t.Fatalf("creator_retained_shares = %d, want %d", result.Card.CreatorRetainedShares, wantRetained)
	}
	if result.Card.CirculatingSupply != wantRetained {
		t.Fatalf("circulating_supply = %d, want %d (only the retained shares exist at launch)", result.Card.CirculatingSupply, wantRetained)
	}
	if result.Card.Type != domain.CardTypeUserCreated {
		t.Fatalf("card_type = %s, want USER_CREATED", result.Card.Type)
	}
	if result.Card.CreatorUserID == nil || *result.Card.CreatorUserID != userID {
		t.Fatalf("creator_user_id not set correctly")
	}
	wantPrice, err := engine.SpotPrice(wantRetained, engine.CurveParams{BasePrice: launchBasePrice, Scale: float64(totalSupply), DemandModifier: 1, DriftFactor: 1})
	if err != nil {
		t.Fatalf("engine.SpotPrice: %v", err)
	}
	if result.Card.CurrentPrice != wantPrice {
		t.Fatalf("current_price = %d, want %d", result.Card.CurrentPrice, wantPrice)
	}

	if result.Transaction.TotalCurrencyDelta != -LaunchCost {
		t.Fatalf("launch transaction delta = %d, want %d", result.Transaction.TotalCurrencyDelta, -LaunchCost)
	}
	wantBalance := initialBalance - LaunchCost
	if result.User.CurrencyBalance != wantBalance {
		t.Fatalf("balance = %d, want %d", result.User.CurrencyBalance, wantBalance)
	}
	assertInvariant(t, q, userID, result.User.CurrencyBalance)

	holding, err := q.GetHoldingForUpdate(ctx, db.GetHoldingForUpdateParams{UserID: userID, CardID: result.Card.ID})
	if err != nil {
		t.Fatalf("get holding: %v", err)
	}
	if holding.SharesOwned != wantRetained {
		t.Fatalf("holding shares_owned = %d, want %d", holding.SharesOwned, wantRetained)
	}
	if holding.AvgCostBasis != 0 {
		t.Fatalf("holding avg_cost_basis = %d, want 0 (granted, not purchased)", holding.AvgCostBasis)
	}
}

func TestLaunchCard_BelowThreshold(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool, uniqueUsername("poor"))
	if _, err := pool.Exec(ctx, "UPDATE users SET currency_balance = $1 WHERE id = $2", LaunchThreshold-1, userID); err != nil {
		t.Fatalf("lower balance: %v", err)
	}

	_, err := l.LaunchCard(ctx, LaunchCardParams{
		CreatorUserID: userID, Symbol: uniqueSymbol("POORL"), Name: "x",
		TotalSupply: 1000, RetainedPercent: 0.1, IdempotencyKey: uuid.NewString(),
	})
	if !errors.Is(err, ErrBelowLaunchThreshold) {
		t.Fatalf("expected ErrBelowLaunchThreshold, got %v", err)
	}
}

func TestLaunchCard_RetainedPercentTooHigh(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool, uniqueUsername("greedy"))
	_, err := l.LaunchCard(ctx, LaunchCardParams{
		CreatorUserID: userID, Symbol: uniqueSymbol("GREEDY"), Name: "x",
		TotalSupply: 1000, RetainedPercent: 0.41, IdempotencyKey: uuid.NewString(),
	})
	if !errors.Is(err, ErrInvalidRetainedPercent) {
		t.Fatalf("expected ErrInvalidRetainedPercent, got %v", err)
	}
}

func TestLaunchCard_InvalidTotalSupply(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool, uniqueUsername("u"))
	for _, ts := range []int64{0, -100} {
		_, err := l.LaunchCard(ctx, LaunchCardParams{
			CreatorUserID: userID, Symbol: uniqueSymbol("BADTS"), Name: "x",
			TotalSupply: ts, RetainedPercent: 0.1, IdempotencyKey: uuid.NewString(),
		})
		if !errors.Is(err, ErrInvalidTotalSupply) {
			t.Fatalf("total_supply=%d: expected ErrInvalidTotalSupply, got %v", ts, err)
		}
	}
}

func TestLaunchCard_DuplicateSymbol(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool, uniqueUsername("dup"))
	symbol := uniqueSymbol("DUPSYM")

	first, err := l.LaunchCard(ctx, LaunchCardParams{
		CreatorUserID: userID, Symbol: symbol, Name: "First",
		TotalSupply: 1000, RetainedPercent: 0.1, IdempotencyKey: uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("first launch: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM holdings WHERE card_id = $1", first.Card.ID)
		pool.Exec(context.Background(), "DELETE FROM transactions WHERE card_id = $1", first.Card.ID)
		pool.Exec(context.Background(), "DELETE FROM cards WHERE id = $1", first.Card.ID)
	})

	_, err = l.LaunchCard(ctx, LaunchCardParams{
		CreatorUserID: userID, Symbol: symbol, Name: "Second",
		TotalSupply: 1000, RetainedPercent: 0.1, IdempotencyKey: uuid.NewString(),
	})
	if !errors.Is(err, ErrSymbolTaken) {
		t.Fatalf("expected ErrSymbolTaken, got %v", err)
	}
}

func TestLaunchCard_IdempotentReplay(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool, uniqueUsername("replay"))
	key := uuid.NewString()
	params := LaunchCardParams{
		CreatorUserID: userID, Symbol: uniqueSymbol("REPL"), Name: "x",
		TotalSupply: 1000, RetainedPercent: 0.1, IdempotencyKey: key,
	}

	first, err := l.LaunchCard(ctx, params)
	if err != nil {
		t.Fatalf("first launch: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM holdings WHERE card_id = $1", first.Card.ID)
		pool.Exec(context.Background(), "DELETE FROM transactions WHERE card_id = $1", first.Card.ID)
		pool.Exec(context.Background(), "DELETE FROM cards WHERE id = $1", first.Card.ID)
	})

	second, err := l.LaunchCard(ctx, params)
	if err != nil {
		t.Fatalf("replay launch: %v", err)
	}
	if second.Card.ID != first.Card.ID {
		t.Fatalf("replay created a different card: %s vs %s", second.Card.ID, first.Card.ID)
	}
	if second.User.CurrencyBalance != first.User.CurrencyBalance {
		t.Fatalf("replay changed balance: %d -> %d (double-charged)", first.User.CurrencyBalance, second.User.CurrencyBalance)
	}
}
