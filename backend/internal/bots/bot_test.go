package bots

import (
	"context"
	"testing"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/ledger"
	"github.com/ekansh-exe/navx/internal/store/db"
)

// TestBotTick_ExecutesRandomWalkerTrade confirms a bot's tick actually goes
// through Ledger.ExecuteTrade and lands real rows in the ledger — not a
// bypass. The random walker always buys when it owns nothing, so across a
// few ticks at least one BUY (+ its linked FEE row) should appear.
func TestBotTick_ExecutesRandomWalkerTrade(t *testing.T) {
	pool := testPool(t)
	ledgerSvc := ledger.New(pool)
	q := db.New(pool)
	ctx := context.Background()

	createTestCard(t, pool, uniqueSymbol("BOTRW"), domain.SupplyModelFixed, ptrInt64(1_000_000), 10000, 10, 1_000_000, domain.CardStatusActive)
	// fetchSnapshots sees every active card in the shared test DB (all the
	// seeded companies too), and the random walker picks uniformly among
	// them — so this needs enough balance to afford whichever real card it
	// lands on, not just the one created above.
	botUserID := createTestBotUser(t, pool, uniqueUsername("bot_test_rw"), 10_000_000_000)

	bot := newBot(botUserID, "bot_test_rw", PersonaRandomWalker, ledgerSvc, q)
	for i := 0; i < 5; i++ {
		bot.tick(ctx)
	}

	txns, err := q.ListTransactionsByUser(ctx, botUserID)
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	if len(txns) == 0 {
		t.Fatal("expected at least one trade to have executed across 5 ticks")
	}
	for _, txn := range txns {
		if txn.Type != "BUY" && txn.Type != "SELL" && txn.Type != "FEE" {
			t.Fatalf("unexpected transaction type %q", txn.Type)
		}
	}
}

// TestBotTick_GracefullySkipsOnInsufficientBalance forces every possible
// buy to be rejected (zero balance) and confirms tick() neither panics nor
// writes a partial transaction — it just skips, exactly like a human's
// rejected HTTP request would.
func TestBotTick_GracefullySkipsOnInsufficientBalance(t *testing.T) {
	pool := testPool(t)
	ledgerSvc := ledger.New(pool)
	q := db.New(pool)
	ctx := context.Background()

	createTestCard(t, pool, uniqueSymbol("BOTPOOR"), domain.SupplyModelFixed, ptrInt64(1_000_000), 1000, 100_000, 1_000_000, domain.CardStatusActive)
	botUserID := createTestBotUser(t, pool, uniqueUsername("bot_test_poor"), 0)

	bot := newBot(botUserID, "bot_test_poor", PersonaRandomWalker, ledgerSvc, q)
	for i := 0; i < 5; i++ {
		bot.tick(ctx) // must not panic
	}

	txns, err := q.ListTransactionsByUser(ctx, botUserID)
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	if len(txns) != 0 {
		t.Fatalf("expected no transactions for a zero-balance bot, got %d", len(txns))
	}
}

func TestIsSkippableTradeError(t *testing.T) {
	skippable := []error{
		ledger.ErrCardNotTradable,
		ledger.ErrCircuitBreakerActive,
		ledger.ErrPositionCapExceeded,
		ledger.ErrInsufficientSupply,
		ledger.ErrInsufficientBalance,
		ledger.ErrInsufficientShares,
	}
	for _, err := range skippable {
		if !isSkippableTradeError(err) {
			t.Errorf("isSkippableTradeError(%v) = false, want true", err)
		}
	}
	if isSkippableTradeError(ledger.ErrCardNotFound) {
		t.Error("isSkippableTradeError(ErrCardNotFound) = true, want false (not in the skippable set)")
	}
}
