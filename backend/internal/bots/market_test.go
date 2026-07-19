package bots

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/store/db"
)

func TestAppendHistory_CapsAtWindow(t *testing.T) {
	history := make(map[uuid.UUID][]int64)
	cardID := uuid.New()
	for i := int64(1); i <= int64(historyWindow)+3; i++ {
		appendHistory(history, cardID, i*10)
	}
	got := history[cardID]
	if len(got) != historyWindow {
		t.Fatalf("history length = %d, want %d", len(got), historyWindow)
	}
	wantFirst := int64(4) * 10 // values 10..80 inserted, only the last 5 (40..80) survive
	if got[0] != wantFirst {
		t.Fatalf("got[0] = %d, want %d", got[0], wantFirst)
	}
	if got[len(got)-1] != 80 {
		t.Fatalf("got[last] = %d, want 80", got[len(got)-1])
	}
}

func TestFetchSnapshots_IncludesHoldingsAndHistory(t *testing.T) {
	pool := testPool(t)
	q := db.New(pool)
	ctx := context.Background()

	cardID := createTestCard(t, pool, uniqueSymbol("BOTMKT"), domain.SupplyModelFixed, ptrInt64(1_000_000), 1000, 10, 1_000_000, domain.CardStatusActive)
	botUserID := createTestBotUser(t, pool, uniqueUsername("bot_test_mkt"), 1_000_000)

	if _, err := pool.Exec(ctx, "INSERT INTO holdings (user_id, card_id, shares_owned, avg_cost_basis) VALUES ($1,$2,42,10)", botUserID, cardID); err != nil {
		t.Fatalf("seed holding: %v", err)
	}

	history := make(map[uuid.UUID][]int64)
	snapshots, err := fetchSnapshots(ctx, q, botUserID, history)
	if err != nil {
		t.Fatalf("fetchSnapshots: %v", err)
	}

	found := findSnapshot(snapshots, cardID)
	if found == nil {
		t.Fatal("test card not found among active-card snapshots")
	}
	if found.SharesOwned != 42 {
		t.Fatalf("SharesOwned = %d, want 42", found.SharesOwned)
	}
	if len(found.History) != 1 || found.History[0] != found.CurrentPrice {
		t.Fatalf("History = %v, want a single entry matching CurrentPrice %d", found.History, found.CurrentPrice)
	}

	snapshots2, err := fetchSnapshots(ctx, q, botUserID, history)
	if err != nil {
		t.Fatalf("fetchSnapshots (2nd): %v", err)
	}
	found2 := findSnapshot(snapshots2, cardID)
	if found2 == nil || len(found2.History) != 2 {
		t.Fatalf("History length after 2nd fetch = %v, want length 2", found2)
	}
}

func findSnapshot(snapshots []MarketSnapshot, cardID uuid.UUID) *MarketSnapshot {
	for i := range snapshots {
		if snapshots[i].CardID == cardID {
			return &snapshots[i]
		}
	}
	return nil
}

func TestComputeDerivedPrices_WeightedSum(t *testing.T) {
	pool := testPool(t)
	q := db.New(pool)
	ctx := context.Background()

	comp1 := createTestCard(t, pool, uniqueSymbol("COMPA"), domain.SupplyModelFixed, ptrInt64(1_000_000), 1000, 100, 1_000_000, domain.CardStatusActive)
	comp2 := createTestCard(t, pool, uniqueSymbol("COMPB"), domain.SupplyModelFixed, ptrInt64(1_000_000), 1000, 100, 1_000_000, domain.CardStatusActive)
	if _, err := pool.Exec(ctx, "UPDATE cards SET current_price = 100 WHERE id = $1", comp1); err != nil {
		t.Fatalf("set comp1 price: %v", err)
	}
	if _, err := pool.Exec(ctx, "UPDATE cards SET current_price = 200 WHERE id = $1", comp2); err != nil {
		t.Fatalf("set comp2 price: %v", err)
	}

	indexCard := createTestCard(t, pool, uniqueSymbol("IDXTST"), domain.SupplyModelUnlimited, nil, 1000, 100, 1_000_000, domain.CardStatusActive)
	if _, err := pool.Exec(ctx, "UPDATE cards SET card_type = 'INDEX' WHERE id = $1", indexCard); err != nil {
		t.Fatalf("set card_type: %v", err)
	}

	if _, err := pool.Exec(ctx,
		"INSERT INTO index_components (index_card_id, component_card_id, weight) VALUES ($1,$2,0.6), ($1,$3,0.4)",
		indexCard, comp1, comp2,
	); err != nil {
		t.Fatalf("insert index components: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM index_components WHERE index_card_id = $1", indexCard)
	})

	derived, err := computeDerivedPrices(ctx, q, []uuid.UUID{indexCard})
	if err != nil {
		t.Fatalf("computeDerivedPrices: %v", err)
	}
	const want = int64(0.6*100 + 0.4*200) // 140
	if derived[indexCard] != want {
		t.Fatalf("derived price = %d, want %d", derived[indexCard], want)
	}
}
