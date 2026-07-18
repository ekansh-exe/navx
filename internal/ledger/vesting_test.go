package ledger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/store/db"
)

func TestUnlockedRetainedShares(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	const retained = int64(1000)

	tests := []struct {
		name string
		now  time.Time
		want int64
	}{
		{"at launch, nothing unlocked", createdAt, 0},
		{"halfway through vesting", createdAt.Add(VestingPeriod / 2), 500},
		{"exactly at vesting end", createdAt.Add(VestingPeriod), 1000},
		{"well past vesting end", createdAt.Add(VestingPeriod * 10), 1000},
		{"clock skew: now before createdAt", createdAt.Add(-time.Hour), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unlockedRetainedShares(retained, createdAt, tt.now)
			if got != tt.want {
				t.Errorf("unlockedRetainedShares = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCreatorSellLimit(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("nothing unlocked yet, no prior sales", func(t *testing.T) {
		card := db.Card{CreatorRetainedShares: 1000, CreatorRetainedSharesSold: 0, CreatedAt: createdAt}
		maxSellable, unlocked := creatorSellLimit(card, 1000, createdAt)
		if maxSellable != 0 || unlocked != 0 {
			t.Fatalf("maxSellable=%d unlocked=%d, want 0,0", maxSellable, unlocked)
		}
	})

	t.Run("half vested, none sold yet, owns exactly the retained amount", func(t *testing.T) {
		card := db.Card{CreatorRetainedShares: 1000, CreatorRetainedSharesSold: 0, CreatedAt: createdAt}
		now := createdAt.Add(VestingPeriod / 2)
		maxSellable, unlocked := creatorSellLimit(card, 1000, now)
		if maxSellable != 500 || unlocked != 500 {
			t.Fatalf("maxSellable=%d unlocked=%d, want 500,500", maxSellable, unlocked)
		}
	})

	t.Run("half vested, already sold the unlocked portion", func(t *testing.T) {
		card := db.Card{CreatorRetainedShares: 1000, CreatorRetainedSharesSold: 500, CreatedAt: createdAt}
		now := createdAt.Add(VestingPeriod / 2)
		maxSellable, unlocked := creatorSellLimit(card, 500, now)
		if maxSellable != 0 || unlocked != 0 {
			t.Fatalf("maxSellable=%d unlocked=%d, want 0,0 (already sold everything unlocked so far)", maxSellable, unlocked)
		}
	})

	t.Run("fully vested, some retained shares already sold, plus extra bought shares", func(t *testing.T) {
		// Owns 1200: 800 remaining retained (1000-200 sold) + 400 bought freely.
		card := db.Card{CreatorRetainedShares: 1000, CreatorRetainedSharesSold: 200, CreatedAt: createdAt}
		now := createdAt.Add(VestingPeriod * 2)
		maxSellable, unlocked := creatorSellLimit(card, 1200, now)
		// unlocked-from-restricted = full 1000 - 200 sold = 800; freely
		// sellable = owned(1200) - remainingRestricted(800) = 400.
		if unlocked != 800 {
			t.Fatalf("unlocked = %d, want 800", unlocked)
		}
		if maxSellable != 1200 {
			t.Fatalf("maxSellable = %d, want 1200 (800 restricted-now-unlocked + 400 freely bought)", maxSellable)
		}
	})

	t.Run("restricted pool fully sold, owns only freely-bought shares", func(t *testing.T) {
		card := db.Card{CreatorRetainedShares: 1000, CreatorRetainedSharesSold: 1000, CreatedAt: createdAt}
		now := createdAt // no vesting progress at all, but it shouldn't matter — pool already exhausted
		maxSellable, unlocked := creatorSellLimit(card, 300, now)
		if unlocked != 0 {
			t.Fatalf("unlocked = %d, want 0", unlocked)
		}
		if maxSellable != 300 {
			t.Fatalf("maxSellable = %d, want 300 (all freely bought, unaffected by vesting)", maxSellable)
		}
	})
}

// TestExecuteTrade_CreatorVestingBlocksEarlySale confirms a card creator
// cannot sell any of their retained shares immediately after launch (§4.3).
func TestExecuteTrade_CreatorVestingBlocksEarlySale(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	ctx := context.Background()

	creatorID := createTestUser(t, pool, uniqueUsername("vestcreator"))
	launchResult, err := l.LaunchCard(ctx, LaunchCardParams{
		CreatorUserID: creatorID, Symbol: uniqueSymbol("VEST"), Name: "Vesting Test Co.",
		TotalSupply: 100_000, RetainedPercent: 0.2, IdempotencyKey: uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("LaunchCard: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM holdings WHERE card_id = $1", launchResult.Card.ID)
		pool.Exec(context.Background(), "DELETE FROM transactions WHERE card_id = $1", launchResult.Card.ID)
		pool.Exec(context.Background(), "DELETE FROM cards WHERE id = $1", launchResult.Card.ID)
	})

	_, err = l.ExecuteTrade(ctx, TradeParams{
		UserID: creatorID, CardID: launchResult.Card.ID, Type: domain.TransactionTypeSell,
		Shares: 1, IdempotencyKey: uuid.NewString(),
	})
	if !errors.Is(err, ErrRetainedSharesLocked) {
		t.Fatalf("expected ErrRetainedSharesLocked immediately after launch, got %v", err)
	}
}

// TestExecuteTrade_CreatorCanSellAfterVestingPeriod confirms the creator
// can freely sell retained shares once the vesting period has fully
// elapsed (simulated by backdating created_at, same technique as Phase 1's
// day-simulation tests).
func TestExecuteTrade_CreatorCanSellAfterVestingPeriod(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	ctx := context.Background()

	creatorID := createTestUser(t, pool, uniqueUsername("vestcreator2"))
	launchResult, err := l.LaunchCard(ctx, LaunchCardParams{
		CreatorUserID: creatorID, Symbol: uniqueSymbol("VESTOK"), Name: "Vesting Test Co 2.",
		TotalSupply: 100_000, RetainedPercent: 0.2, IdempotencyKey: uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("LaunchCard: %v", err)
	}
	cardID := launchResult.Card.ID
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM holdings WHERE card_id = $1", cardID)
		pool.Exec(context.Background(), "DELETE FROM transactions WHERE card_id = $1", cardID)
		pool.Exec(context.Background(), "DELETE FROM cards WHERE id = $1", cardID)
	})

	backdated := time.Now().UTC().Add(-VestingPeriod - 24*time.Hour)
	if _, err := pool.Exec(ctx, "UPDATE cards SET created_at = $1 WHERE id = $2", backdated, cardID); err != nil {
		t.Fatalf("backdate card: %v", err)
	}

	result, err := l.ExecuteTrade(ctx, TradeParams{
		UserID: creatorID, CardID: cardID, Type: domain.TransactionTypeSell,
		Shares: launchResult.Card.CreatorRetainedShares, IdempotencyKey: uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("expected the fully-vested creator to sell all retained shares, got %v", err)
	}
	if result.Card.CreatorRetainedSharesSold != launchResult.Card.CreatorRetainedShares {
		t.Fatalf("creator_retained_shares_sold = %d, want %d", result.Card.CreatorRetainedSharesSold, launchResult.Card.CreatorRetainedShares)
	}
}

// TestExecuteTrade_NonCreatorUnaffectedByVesting confirms vesting only
// restricts the card's own creator — any other user can buy and sell
// freely regardless of how recently the card launched.
func TestExecuteTrade_NonCreatorUnaffectedByVesting(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	ctx := context.Background()

	creatorID := createTestUser(t, pool, uniqueUsername("vestcreator3"))
	launchResult, err := l.LaunchCard(ctx, LaunchCardParams{
		CreatorUserID: creatorID, Symbol: uniqueSymbol("VESTOTHER"), Name: "Vesting Test Co 3.",
		TotalSupply: 100_000, RetainedPercent: 0.2, IdempotencyKey: uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("LaunchCard: %v", err)
	}
	cardID := launchResult.Card.ID
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM holdings WHERE card_id = $1", cardID)
		pool.Exec(context.Background(), "DELETE FROM transactions WHERE card_id = $1", cardID)
		pool.Exec(context.Background(), "DELETE FROM cards WHERE id = $1", cardID)
	})

	otherUserID := createTestUser(t, pool, uniqueUsername("buyer"))
	if _, err := l.ExecuteTrade(ctx, TradeParams{
		UserID: otherUserID, CardID: cardID, Type: domain.TransactionTypeBuy,
		Shares: 50, IdempotencyKey: uuid.NewString(),
	}); err != nil {
		t.Fatalf("other user buy: %v", err)
	}
	// Immediately sell right back — no vesting restriction applies since
	// this isn't the creator.
	if _, err := l.ExecuteTrade(ctx, TradeParams{
		UserID: otherUserID, CardID: cardID, Type: domain.TransactionTypeSell,
		Shares: 50, IdempotencyKey: uuid.NewString(),
	}); err != nil {
		t.Fatalf("expected non-creator sell to succeed immediately, got %v", err)
	}
}
