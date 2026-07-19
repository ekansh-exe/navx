package quests

import (
	"context"
	"testing"
	"time"

	"github.com/ekansh-exe/navx/internal/domain"
)

// TestCheckHoldCardQuests_GrantsAfter24h covers §7's "hold a card for 24
// hours": a HUMAN user whose position was first bought more than 24h ago
// and who still holds shares should be granted the reward, exactly once.
func TestCheckHoldCardQuests_GrantsAfter24h(t *testing.T) {
	pool := testPool(t)
	svc := New(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool, "HUMAN", 100_000)
	cardID := createTestCard(t, pool)
	createTestHolding(t, pool, userID, cardID, 10, time.Now().UTC().Add(-25*time.Hour))

	if err := svc.CheckHoldCardQuests(ctx); err != nil {
		t.Fatalf("CheckHoldCardQuests: %v", err)
	}

	questID := questIDForType(t, pool, string(domain.QuestTypeHoldCard))
	_, completed, _, found := getUserQuestRow(t, pool, userID, questID)
	if !found || !completed {
		t.Fatal("expected the HOLD_CARD quest to be completed after a qualifying 25h-old position")
	}
	if n := countTransactions(t, pool, userID, "QUEST_REWARD"); n != 1 {
		t.Fatalf("QUEST_REWARD transactions = %d, want 1", n)
	}

	// Running the check again must not double-grant.
	if err := svc.CheckHoldCardQuests(ctx); err != nil {
		t.Fatalf("second CheckHoldCardQuests: %v", err)
	}
	if n := countTransactions(t, pool, userID, "QUEST_REWARD"); n != 1 {
		t.Fatalf("QUEST_REWARD transactions after a second check = %d, want still 1", n)
	}
}

// TestCheckHoldCardQuests_SkipsUnder24h confirms a position bought less
// than 24h ago is not yet eligible.
func TestCheckHoldCardQuests_SkipsUnder24h(t *testing.T) {
	pool := testPool(t)
	svc := New(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool, "HUMAN", 100_000)
	cardID := createTestCard(t, pool)
	createTestHolding(t, pool, userID, cardID, 10, time.Now().UTC().Add(-1*time.Hour))

	if err := svc.CheckHoldCardQuests(ctx); err != nil {
		t.Fatalf("CheckHoldCardQuests: %v", err)
	}
	if n := countTransactions(t, pool, userID, "QUEST_REWARD"); n != 0 {
		t.Fatalf("QUEST_REWARD transactions for a 1h-old position = %d, want 0", n)
	}
}

// TestCheckHoldCardQuests_ExcludesBots mirrors §4.5/§8's bot-exclusion
// precedent — a BOT user with an otherwise-qualifying position should never
// be granted a quest reward.
func TestCheckHoldCardQuests_ExcludesBots(t *testing.T) {
	pool := testPool(t)
	svc := New(pool)
	ctx := context.Background()

	botID := createTestUser(t, pool, "BOT", 100_000)
	cardID := createTestCard(t, pool)
	createTestHolding(t, pool, botID, cardID, 10, time.Now().UTC().Add(-48*time.Hour))

	if err := svc.CheckHoldCardQuests(ctx); err != nil {
		t.Fatalf("CheckHoldCardQuests: %v", err)
	}
	if n := countTransactions(t, pool, botID, "QUEST_REWARD"); n != 0 {
		t.Fatalf("QUEST_REWARD transactions for a bot user = %d, want 0", n)
	}
}
