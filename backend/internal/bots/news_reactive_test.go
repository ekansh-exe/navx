package bots

import (
	"context"
	"testing"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/ledger"
	"github.com/ekansh-exe/navx/internal/store/db"
)

func createTestNewsEvent(t *testing.T, q *db.Queries, headline, category string) {
	t.Helper()
	if _, err := q.CreateNewsEvent(context.Background(), db.CreateNewsEventParams{
		Headline: headline,
		Category: &category,
	}); err != nil {
		t.Fatalf("create test news event: %v", err)
	}
}

// TestBotTick_NewsReactive_SellsOnNegativeSectorNews confirms a flood
// headline naming FOOD produces a real SELL transaction on a FOOD-sector
// card, drawing down the bot's pre-existing holding — §4.5's "sell
// affected-sector cards" for negative events.
func TestBotTick_NewsReactive_SellsOnNegativeSectorNews(t *testing.T) {
	pool := testPool(t)
	ledgerSvc := ledger.New(pool)
	q := db.New(pool)
	ctx := context.Background()

	foodCardID := createTestCard(t, pool, uniqueSymbol("NRFOOD"), domain.SupplyModelFixed, ptrInt64(1_000_000), 500_000, 20, 1_000_000, domain.CardStatusActive)
	if _, err := pool.Exec(ctx, `UPDATE cards SET sector = 'FOOD' WHERE id = $1`, foodCardID); err != nil {
		t.Fatalf("set card sector: %v", err)
	}
	botUserID := createTestBotUser(t, pool, uniqueUsername("bot_test_nr_sell"), 10_000_000_000)
	createTestHolding(t, pool, botUserID, foodCardID, 5000, 2000)

	bot := newBot(botUserID, "bot_test_nr_sell", PersonaNewsReactive, ledgerSvc, q)
	bot.tick(ctx) // seeds the news cursor before any headline exists

	createTestNewsEvent(t, q, "Flood in Endia affects Food markets", "FLOOD")

	// The shared test DB also has the seeded FOOD/AGRICULTURE companies
	// (migration 000002), so a flood headline queues trades for those too
	// (skipped as zero-share sells for this fresh bot) ahead of or behind
	// our own card in the queue — loop generously to drain past them.
	var sawSell bool
	for i := 0; i < 40 && !sawSell; i++ {
		bot.tick(ctx)
		txns, err := q.ListTransactionsByUser(ctx, botUserID)
		if err != nil {
			t.Fatalf("list transactions: %v", err)
		}
		for _, txn := range txns {
			if txn.Type == "SELL" && txn.CardID != nil && *txn.CardID == foodCardID {
				sawSell = true
			}
		}
	}
	if !sawSell {
		t.Fatal("expected a SELL transaction on the food card after a flood headline, got none")
	}
}

// TestBotTick_NewsReactive_BuysOnPositiveSectorNews confirms a discovery
// headline naming SEMICONDUCTOR produces a real BUY transaction — §4.5's
// "buy affected-sector cards" for positive events.
func TestBotTick_NewsReactive_BuysOnPositiveSectorNews(t *testing.T) {
	pool := testPool(t)
	ledgerSvc := ledger.New(pool)
	q := db.New(pool)
	ctx := context.Background()

	chipCardID := createTestCard(t, pool, uniqueSymbol("NRCHIP"), domain.SupplyModelFixed, ptrInt64(1_000_000), 500_000, 20, 1_000_000, domain.CardStatusActive)
	// Overwrite the card's sector — createTestCard doesn't take one.
	if _, err := pool.Exec(ctx, `UPDATE cards SET sector = 'SEMICONDUCTOR' WHERE id = $1`, chipCardID); err != nil {
		t.Fatalf("set card sector: %v", err)
	}
	botUserID := createTestBotUser(t, pool, uniqueUsername("bot_test_nr_buy"), 10_000_000_000)

	bot := newBot(botUserID, "bot_test_nr_buy", PersonaNewsReactive, ledgerSvc, q)
	bot.tick(ctx)

	createTestNewsEvent(t, q, "Discovery in Straya affects Semiconductor markets", "DISCOVERY")

	// Same reasoning as the sell test above — the seeded SEMICONDUCTOR/
	// METALS/MISC_COMMODITIES companies also queue BUYs on this headline.
	var sawBuy bool
	for i := 0; i < 40 && !sawBuy; i++ {
		bot.tick(ctx)
		txns, err := q.ListTransactionsByUser(ctx, botUserID)
		if err != nil {
			t.Fatalf("list transactions: %v", err)
		}
		for _, txn := range txns {
			if txn.Type == "BUY" && txn.CardID != nil && *txn.CardID == chipCardID {
				sawBuy = true
			}
		}
	}
	if !sawBuy {
		t.Fatal("expected a BUY transaction on the semiconductor card after a discovery headline, got none")
	}
}

// TestBotTick_NewsReactive_IgnoresPreExistingBacklog confirms a headline
// created before the bot ever polled isn't treated as new — a
// freshly-started bot shouldn't replay the entire news history.
func TestBotTick_NewsReactive_IgnoresPreExistingBacklog(t *testing.T) {
	pool := testPool(t)
	ledgerSvc := ledger.New(pool)
	q := db.New(pool)
	ctx := context.Background()

	foodCardID := createTestCard(t, pool, uniqueSymbol("NRBACK"), domain.SupplyModelFixed, ptrInt64(1_000_000), 500_000, 20, 1_000_000, domain.CardStatusActive)
	if _, err := pool.Exec(ctx, `UPDATE cards SET sector = 'FOOD' WHERE id = $1`, foodCardID); err != nil {
		t.Fatalf("set card sector: %v", err)
	}
	botUserID := createTestBotUser(t, pool, uniqueUsername("bot_test_nr_backlog"), 10_000_000_000)
	createTestHolding(t, pool, botUserID, foodCardID, 5000, 2000)

	createTestNewsEvent(t, q, "Flood in Endia affects Food markets", "FLOOD")

	bot := newBot(botUserID, "bot_test_nr_backlog", PersonaNewsReactive, ledgerSvc, q)
	for i := 0; i < 5; i++ {
		bot.tick(ctx)
	}

	txns, err := q.ListTransactionsByUser(ctx, botUserID)
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	if len(txns) != 0 {
		t.Fatalf("expected no transactions from pre-existing news backlog, got %d", len(txns))
	}
}

// TestBotTick_NewsReactive_IgnoresStrikeEvent confirms STRIKE headlines
// (not classified positive or negative) never queue a trade.
func TestBotTick_NewsReactive_IgnoresStrikeEvent(t *testing.T) {
	pool := testPool(t)
	ledgerSvc := ledger.New(pool)
	q := db.New(pool)
	ctx := context.Background()

	shippingCardID := createTestCard(t, pool, uniqueSymbol("NRSTRK"), domain.SupplyModelFixed, ptrInt64(1_000_000), 500_000, 20, 1_000_000, domain.CardStatusActive)
	if _, err := pool.Exec(ctx, `UPDATE cards SET sector = 'SHIPPING' WHERE id = $1`, shippingCardID); err != nil {
		t.Fatalf("set card sector: %v", err)
	}
	botUserID := createTestBotUser(t, pool, uniqueUsername("bot_test_nr_strike"), 10_000_000_000)
	createTestHolding(t, pool, botUserID, shippingCardID, 5000, 2000)

	bot := newBot(botUserID, "bot_test_nr_strike", PersonaNewsReactive, ledgerSvc, q)
	bot.tick(ctx)

	createTestNewsEvent(t, q, "Strike in Kanadia affects Shipping markets", "STRIKE")

	for i := 0; i < 5; i++ {
		bot.tick(ctx)
	}

	txns, err := q.ListTransactionsByUser(ctx, botUserID)
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	if len(txns) != 0 {
		t.Fatalf("expected no transactions for a STRIKE headline, got %d", len(txns))
	}
}

// TestBotTick_NewsReactive_GracefullySkipsWithoutInventory confirms a
// negative headline against a sector the bot holds nothing in never
// panics and produces no transaction — the same "log and skip" contract
// every other persona's rejected trade already gets.
func TestBotTick_NewsReactive_GracefullySkipsWithoutInventory(t *testing.T) {
	pool := testPool(t)
	ledgerSvc := ledger.New(pool)
	q := db.New(pool)
	ctx := context.Background()

	foodCardID := createTestCard(t, pool, uniqueSymbol("NRNOINV"), domain.SupplyModelFixed, ptrInt64(1_000_000), 500_000, 20, 1_000_000, domain.CardStatusActive)
	if _, err := pool.Exec(ctx, `UPDATE cards SET sector = 'FOOD' WHERE id = $1`, foodCardID); err != nil {
		t.Fatalf("set card sector: %v", err)
	}
	botUserID := createTestBotUser(t, pool, uniqueUsername("bot_test_nr_noinv"), 10_000_000_000)
	// Deliberately no holding created — this bot owns nothing in this card.

	bot := newBot(botUserID, "bot_test_nr_noinv", PersonaNewsReactive, ledgerSvc, q)
	bot.tick(ctx)

	createTestNewsEvent(t, q, "Flood in Endia affects Food markets", "FLOOD")

	for i := 0; i < 5; i++ {
		bot.tick(ctx) // must not panic
	}

	txns, err := q.ListTransactionsByUser(ctx, botUserID)
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	for _, txn := range txns {
		if txn.Type == "SELL" && txn.CardID != nil && *txn.CardID == foodCardID {
			t.Fatalf("expected no SELL against a card the bot holds nothing in")
		}
	}
}
