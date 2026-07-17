package ledger

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/google/uuid"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/engine"
	"github.com/ekansh-exe/navx/internal/store/db"
)

func TestExecuteTrade_BuyHappyPath(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	q := db.New(pool)
	ctx := context.Background()

	const startSupply, basePrice, scale = int64(1000), 10.0, 1_000_000.0
	cardID := createTestCard(t, pool, uniqueSymbol("BUY"), domain.SupplyModelFixed, ptrInt64(1_000_000), startSupply, basePrice, scale, domain.CardStatusActive)
	userID := createTestUser(t, pool, uniqueUsername("buyer"))

	const shares = int64(100)
	params := engine.CurveParams{BasePrice: basePrice, Scale: scale, DemandModifier: 1, DriftFactor: 1}
	wantCost, err := engine.ExecutionCost(startSupply, shares, params)
	if err != nil {
		t.Fatalf("engine.ExecutionCost: %v", err)
	}
	wantFee := computeFee(wantCost)

	result, err := l.ExecuteTrade(ctx, TradeParams{
		UserID: userID, CardID: cardID, Type: domain.TransactionTypeBuy, Shares: shares, IdempotencyKey: uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("ExecuteTrade: %v", err)
	}

	if result.Transaction.TotalCurrencyDelta != -wantCost {
		t.Fatalf("trade delta = %d, want %d", result.Transaction.TotalCurrencyDelta, -wantCost)
	}
	if result.FeeTransaction.TotalCurrencyDelta != -wantFee {
		t.Fatalf("fee delta = %d, want %d", result.FeeTransaction.TotalCurrencyDelta, -wantFee)
	}
	if result.FeeTransaction.RelatedTransactionID == nil || *result.FeeTransaction.RelatedTransactionID != result.Transaction.ID {
		t.Fatalf("fee transaction's RelatedTransactionID doesn't point at the trade transaction")
	}
	wantBalance := initialBalance - wantCost - wantFee
	if result.User.CurrencyBalance != wantBalance {
		t.Fatalf("balance = %d, want %d", result.User.CurrencyBalance, wantBalance)
	}
	if result.Card.CirculatingSupply != startSupply+shares {
		t.Fatalf("circulating supply = %d, want %d", result.Card.CirculatingSupply, startSupply+shares)
	}

	holding, err := q.GetHoldingForUpdate(ctx, db.GetHoldingForUpdateParams{UserID: userID, CardID: cardID})
	if err != nil {
		t.Fatalf("get holding: %v", err)
	}
	if holding.SharesOwned != shares {
		t.Fatalf("shares_owned = %d, want %d", holding.SharesOwned, shares)
	}
	wantAvgCost := int64(math.Round(float64(wantCost+wantFee) / float64(shares)))
	if holding.AvgCostBasis != wantAvgCost {
		t.Fatalf("avg_cost_basis = %d, want %d", holding.AvgCostBasis, wantAvgCost)
	}

	assertInvariant(t, q, userID, result.User.CurrencyBalance)
}

func TestExecuteTrade_SellHappyPath(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	q := db.New(pool)
	ctx := context.Background()

	const startSupply, basePrice, scale = int64(1000), 10.0, 1_000_000.0
	cardID := createTestCard(t, pool, uniqueSymbol("SELL"), domain.SupplyModelFixed, ptrInt64(1_000_000), startSupply, basePrice, scale, domain.CardStatusActive)
	userID := createTestUser(t, pool, uniqueUsername("seller"))

	const buyShares = int64(200)
	buyResult, err := l.ExecuteTrade(ctx, TradeParams{
		UserID: userID, CardID: cardID, Type: domain.TransactionTypeBuy, Shares: buyShares, IdempotencyKey: uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("setup buy: %v", err)
	}
	holdingAfterBuy, err := q.GetHoldingForUpdate(ctx, db.GetHoldingForUpdateParams{UserID: userID, CardID: cardID})
	if err != nil {
		t.Fatalf("get holding after buy: %v", err)
	}

	const sellShares = int64(80)
	supplyAfterBuy := startSupply + buyShares
	params := engine.CurveParams{BasePrice: basePrice, Scale: scale, DemandModifier: 1, DriftFactor: 1}
	wantCost, err := engine.ExecutionCost(supplyAfterBuy, -sellShares, params)
	if err != nil {
		t.Fatalf("engine.ExecutionCost: %v", err)
	}
	wantFee := computeFee(wantCost)

	result, err := l.ExecuteTrade(ctx, TradeParams{
		UserID: userID, CardID: cardID, Type: domain.TransactionTypeSell, Shares: sellShares, IdempotencyKey: uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("ExecuteTrade sell: %v", err)
	}

	if result.Transaction.TotalCurrencyDelta != -wantCost {
		t.Fatalf("sell trade delta = %d, want %d", result.Transaction.TotalCurrencyDelta, -wantCost)
	}
	if wantCost >= 0 {
		t.Fatalf("expected a sell to have a negative engine cost (proceeds), got %d", wantCost)
	}
	wantBalance := buyResult.User.CurrencyBalance - wantCost - wantFee
	if result.User.CurrencyBalance != wantBalance {
		t.Fatalf("balance = %d, want %d", result.User.CurrencyBalance, wantBalance)
	}
	if result.Card.CirculatingSupply != supplyAfterBuy-sellShares {
		t.Fatalf("circulating supply = %d, want %d", result.Card.CirculatingSupply, supplyAfterBuy-sellShares)
	}

	holding, err := q.GetHoldingForUpdate(ctx, db.GetHoldingForUpdateParams{UserID: userID, CardID: cardID})
	if err != nil {
		t.Fatalf("get holding: %v", err)
	}
	if holding.SharesOwned != buyShares-sellShares {
		t.Fatalf("shares_owned = %d, want %d", holding.SharesOwned, buyShares-sellShares)
	}
	// Selling never changes the remaining shares' cost basis.
	if holding.AvgCostBasis != holdingAfterBuy.AvgCostBasis {
		t.Fatalf("avg_cost_basis changed on sell: %d -> %d", holdingAfterBuy.AvgCostBasis, holding.AvgCostBasis)
	}

	assertInvariant(t, q, userID, result.User.CurrencyBalance)
}

func TestExecuteTrade_InvalidShares(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	ctx := context.Background()

	cardID := createTestCard(t, pool, uniqueSymbol("INVSHR"), domain.SupplyModelFixed, ptrInt64(1_000_000), 1000, 10, 1_000_000, domain.CardStatusActive)
	userID := createTestUser(t, pool, uniqueUsername("u"))

	for _, shares := range []int64{0, -5} {
		_, err := l.ExecuteTrade(ctx, TradeParams{UserID: userID, CardID: cardID, Type: domain.TransactionTypeBuy, Shares: shares, IdempotencyKey: uuid.NewString()})
		if !errors.Is(err, ErrInvalidShares) {
			t.Fatalf("shares=%d: expected ErrInvalidShares, got %v", shares, err)
		}
	}
}

func TestExecuteTrade_InvalidType(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	ctx := context.Background()

	cardID := createTestCard(t, pool, uniqueSymbol("INVTYP"), domain.SupplyModelFixed, ptrInt64(1_000_000), 1000, 10, 1_000_000, domain.CardStatusActive)
	userID := createTestUser(t, pool, uniqueUsername("u"))

	_, err := l.ExecuteTrade(ctx, TradeParams{UserID: userID, CardID: cardID, Type: domain.TransactionTypeFee, Shares: 10, IdempotencyKey: uuid.NewString()})
	if !errors.Is(err, ErrInvalidTradeType) {
		t.Fatalf("expected ErrInvalidTradeType, got %v", err)
	}
}

func TestExecuteTrade_CardNotFound(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool, uniqueUsername("u"))
	_, err := l.ExecuteTrade(ctx, TradeParams{UserID: userID, CardID: uuid.New(), Type: domain.TransactionTypeBuy, Shares: 10, IdempotencyKey: uuid.NewString()})
	if !errors.Is(err, ErrCardNotFound) {
		t.Fatalf("expected ErrCardNotFound, got %v", err)
	}
}

func TestExecuteTrade_InsufficientBalance(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	ctx := context.Background()

	// A high base_price makes even a modest buy exceed the default starting balance.
	cardID := createTestCard(t, pool, uniqueSymbol("EXPNSV"), domain.SupplyModelFixed, ptrInt64(1_000_000_000), 1000, 1_000_000, 1_000_000, domain.CardStatusActive)
	userID := createTestUser(t, pool, uniqueUsername("poor"))

	_, err := l.ExecuteTrade(ctx, TradeParams{UserID: userID, CardID: cardID, Type: domain.TransactionTypeBuy, Shares: 100, IdempotencyKey: uuid.NewString()})
	if !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("expected ErrInsufficientBalance, got %v", err)
	}
}

func TestExecuteTrade_InsufficientShares(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	ctx := context.Background()

	cardID := createTestCard(t, pool, uniqueSymbol("NOSHR"), domain.SupplyModelFixed, ptrInt64(1_000_000), 1000, 10, 1_000_000, domain.CardStatusActive)
	userID := createTestUser(t, pool, uniqueUsername("u"))

	// Never bought anything, so any sell must fail.
	_, err := l.ExecuteTrade(ctx, TradeParams{UserID: userID, CardID: cardID, Type: domain.TransactionTypeSell, Shares: 1, IdempotencyKey: uuid.NewString()})
	if !errors.Is(err, ErrInsufficientShares) {
		t.Fatalf("expected ErrInsufficientShares, got %v", err)
	}
}

func TestExecuteTrade_InsufficientSupply(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	ctx := context.Background()

	const startSupply, total = int64(1000), int64(1100)
	cardID := createTestCard(t, pool, uniqueSymbol("TIGHT"), domain.SupplyModelFixed, ptrInt64(total), startSupply, 1, 1_000_000, domain.CardStatusActive)
	userID := createTestUser(t, pool, uniqueUsername("u"))

	// Only 100 shares remain (1100-1000); buying 101 must be rejected before
	// balance is even considered.
	_, err := l.ExecuteTrade(ctx, TradeParams{UserID: userID, CardID: cardID, Type: domain.TransactionTypeBuy, Shares: total - startSupply + 1, IdempotencyKey: uuid.NewString()})
	if !errors.Is(err, ErrInsufficientSupply) {
		t.Fatalf("expected ErrInsufficientSupply, got %v", err)
	}
}

func TestExecuteTrade_BuyBlockedButSellAllowedWhenNotActive(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	ctx := context.Background()

	cardID := createTestCard(t, pool, uniqueSymbol("FROZEN"), domain.SupplyModelFixed, ptrInt64(1_000_000), 1000, 10, 1_000_000, domain.CardStatusActive)
	userID := createTestUser(t, pool, uniqueUsername("u"))

	// Buy while still ACTIVE, then freeze the card.
	if _, err := l.ExecuteTrade(ctx, TradeParams{UserID: userID, CardID: cardID, Type: domain.TransactionTypeBuy, Shares: 50, IdempotencyKey: uuid.NewString()}); err != nil {
		t.Fatalf("setup buy: %v", err)
	}
	if _, err := pool.Exec(ctx, "UPDATE cards SET status = 'FROZEN' WHERE id = $1", cardID); err != nil {
		t.Fatalf("freeze card: %v", err)
	}

	if _, err := l.ExecuteTrade(ctx, TradeParams{UserID: userID, CardID: cardID, Type: domain.TransactionTypeBuy, Shares: 10, IdempotencyKey: uuid.NewString()}); !errors.Is(err, ErrCardNotTradable) {
		t.Fatalf("expected ErrCardNotTradable for BUY on a FROZEN card, got %v", err)
	}
	if _, err := l.ExecuteTrade(ctx, TradeParams{UserID: userID, CardID: cardID, Type: domain.TransactionTypeSell, Shares: 10, IdempotencyKey: uuid.NewString()}); err != nil {
		t.Fatalf("expected SELL to succeed on a FROZEN card (never trap a position), got %v", err)
	}
}

func TestExecuteTrade_IdempotentReplay(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	ctx := context.Background()

	cardID := createTestCard(t, pool, uniqueSymbol("IDEMP"), domain.SupplyModelFixed, ptrInt64(1_000_000), 1000, 10, 1_000_000, domain.CardStatusActive)
	userID := createTestUser(t, pool, uniqueUsername("u"))
	key := uuid.NewString()

	first, err := l.ExecuteTrade(ctx, TradeParams{UserID: userID, CardID: cardID, Type: domain.TransactionTypeBuy, Shares: 30, IdempotencyKey: key})
	if err != nil {
		t.Fatalf("first execute: %v", err)
	}

	second, err := l.ExecuteTrade(ctx, TradeParams{UserID: userID, CardID: cardID, Type: domain.TransactionTypeBuy, Shares: 30, IdempotencyKey: key})
	if err != nil {
		t.Fatalf("replay execute: %v", err)
	}

	if second.Transaction.ID != first.Transaction.ID {
		t.Fatalf("replay returned a different transaction: %s vs %s", second.Transaction.ID, first.Transaction.ID)
	}
	if second.User.CurrencyBalance != first.User.CurrencyBalance {
		t.Fatalf("replay changed the balance: %d -> %d (double-charged)", first.User.CurrencyBalance, second.User.CurrencyBalance)
	}
	if second.Card.CirculatingSupply != first.Card.CirculatingSupply {
		t.Fatalf("replay changed circulating supply: %d -> %d (double-applied)", first.Card.CirculatingSupply, second.Card.CirculatingSupply)
	}
}

func TestExecuteTrade_IdempotencyKeyMismatch(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	ctx := context.Background()

	cardID := createTestCard(t, pool, uniqueSymbol("MISMTCH"), domain.SupplyModelFixed, ptrInt64(1_000_000), 1000, 10, 1_000_000, domain.CardStatusActive)
	userID := createTestUser(t, pool, uniqueUsername("u"))
	key := uuid.NewString()

	if _, err := l.ExecuteTrade(ctx, TradeParams{UserID: userID, CardID: cardID, Type: domain.TransactionTypeBuy, Shares: 30, IdempotencyKey: key}); err != nil {
		t.Fatalf("first execute: %v", err)
	}

	// Same key, different share count — a genuinely different trade.
	_, err := l.ExecuteTrade(ctx, TradeParams{UserID: userID, CardID: cardID, Type: domain.TransactionTypeBuy, Shares: 999, IdempotencyKey: key})
	if !errors.Is(err, ErrIdempotencyKeyMismatch) {
		t.Fatalf("expected ErrIdempotencyKeyMismatch, got %v", err)
	}
}

// TestExecuteTrade_RoundTripSymmetry buys N then sells N back at the
// resulting supply. Supply returns to exactly its starting value (integer
// arithmetic, no rounding involved), and the net notional cancels to within
// a small tolerance — each leg's cost independently rounds to the nearest
// currency unit, so exact cancellation isn't guaranteed, only near-zero.
func TestExecuteTrade_RoundTripSymmetry(t *testing.T) {
	pool := testPool(t)
	l := New(pool)
	ctx := context.Background()

	const startSupply = int64(50_000)
	cardID := createTestCard(t, pool, uniqueSymbol("RNDTRIP"), domain.SupplyModelFixed, ptrInt64(1_000_000), startSupply, 10, 1_000_000, domain.CardStatusActive)
	userID := createTestUser(t, pool, uniqueUsername("u"))

	const shares = int64(500)
	buy, err := l.ExecuteTrade(ctx, TradeParams{UserID: userID, CardID: cardID, Type: domain.TransactionTypeBuy, Shares: shares, IdempotencyKey: uuid.NewString()})
	if err != nil {
		t.Fatalf("buy: %v", err)
	}
	sell, err := l.ExecuteTrade(ctx, TradeParams{UserID: userID, CardID: cardID, Type: domain.TransactionTypeSell, Shares: shares, IdempotencyKey: uuid.NewString()})
	if err != nil {
		t.Fatalf("sell: %v", err)
	}

	if sell.Card.CirculatingSupply != startSupply {
		t.Fatalf("circulating supply after round trip = %d, want %d", sell.Card.CirculatingSupply, startSupply)
	}

	netNotional := buy.Transaction.TotalCurrencyDelta + sell.Transaction.TotalCurrencyDelta
	if abs(netNotional) > 1 {
		t.Fatalf("round-trip net notional = %d, want within 1 of 0 (buy+sell should nearly cancel)", netNotional)
	}

	wantBalanceDrop := -buy.FeeTransaction.TotalCurrencyDelta - sell.FeeTransaction.TotalCurrencyDelta
	actualBalanceDrop := initialBalance - sell.User.CurrencyBalance
	if abs(actualBalanceDrop-wantBalanceDrop) > 1 {
		t.Fatalf("round trip cost %d, want within 1 of the two fees (%d) — should lose only fee friction", actualBalanceDrop, wantBalanceDrop)
	}
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
