package quests

import (
	"context"
	"testing"

	"github.com/ekansh-exe/navx/internal/domain"
)

// TestRecordTrade_ProgressIncrementsAndCompletes covers §7's "make 3 trades
// today": progress should go 0→1→2→3, granting the reward as a
// QUEST_REWARD transaction exactly once it hits target_value, and a 4th
// trade the same day must not grant it again.
func TestRecordTrade_ProgressIncrementsAndCompletes(t *testing.T) {
	pool := testPool(t)
	svc := New(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool, "HUMAN", 100_000)
	questID := questIDForType(t, pool, string(domain.QuestTypeMakeTrades))

	for i, wantProgress := range []int32{1, 2, 3} {
		if err := svc.RecordTrade(ctx, userID); err != nil {
			t.Fatalf("RecordTrade #%d: %v", i+1, err)
		}
		progress, completed, _, found := getUserQuestRow(t, pool, userID, questID)
		if !found {
			t.Fatalf("expected a user_quests row after RecordTrade #%d", i+1)
		}
		if progress != wantProgress {
			t.Fatalf("after trade #%d: progress = %d, want %d", i+1, progress, wantProgress)
		}
		wantCompleted := wantProgress == 3
		if completed != wantCompleted {
			t.Fatalf("after trade #%d: completed = %v, want %v", i+1, completed, wantCompleted)
		}
	}

	if n := countTransactions(t, pool, userID, "QUEST_REWARD"); n != 1 {
		t.Fatalf("QUEST_REWARD transactions after completing = %d, want 1", n)
	}

	// A 4th trade the same day must not grant the reward again.
	if err := svc.RecordTrade(ctx, userID); err != nil {
		t.Fatalf("RecordTrade #4: %v", err)
	}
	progress, _, _, _ := getUserQuestRow(t, pool, userID, questID)
	if progress != 3 {
		t.Fatalf("progress after 4th trade = %d, want unchanged at 3", progress)
	}
	if n := countTransactions(t, pool, userID, "QUEST_REWARD"); n != 1 {
		t.Fatalf("QUEST_REWARD transactions after a 4th trade = %d, want still 1 (no double grant)", n)
	}
}

// TestRecordTrade_RewardTransactionShape confirms the granted reward is a
// real QUEST_REWARD transaction with the seeded reward amount, not a
// separate ad-hoc balance update.
func TestRecordTrade_RewardTransactionShape(t *testing.T) {
	pool := testPool(t)
	svc := New(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool, "HUMAN", 100_000)
	for i := 0; i < 3; i++ {
		if err := svc.RecordTrade(ctx, userID); err != nil {
			t.Fatalf("RecordTrade #%d: %v", i+1, err)
		}
	}

	var txType string
	var delta int64
	err := pool.QueryRow(context.Background(),
		`SELECT type, total_currency_delta FROM transactions WHERE user_id = $1 AND type = 'QUEST_REWARD'`, userID,
	).Scan(&txType, &delta)
	if err != nil {
		t.Fatalf("query quest reward transaction: %v", err)
	}
	if txType != "QUEST_REWARD" {
		t.Fatalf("transaction type = %q, want QUEST_REWARD", txType)
	}
	if delta != 100 {
		t.Fatalf("reward delta = %d, want 100 (seeded MAKE_TRADES reward)", delta)
	}
}
