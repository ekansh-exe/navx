package quests

import (
	"context"
	"testing"
)

// TestListUserQuests_DefaultsThenReflectsProgress confirms a brand-new
// user (no user_quests rows at all) sees all seeded quests at progress 0,
// not completed — and that progress becomes visible immediately after
// RecordTrade.
func TestListUserQuests_DefaultsThenReflectsProgress(t *testing.T) {
	pool := testPool(t)
	svc := New(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool, "HUMAN", 100_000)

	views, err := svc.ListUserQuests(ctx, userID)
	if err != nil {
		t.Fatalf("ListUserQuests: %v", err)
	}
	if len(views) != 3 {
		t.Fatalf("len(views) = %d, want 3 (the seeded quests)", len(views))
	}
	for _, v := range views {
		if v.Progress != 0 || v.Completed {
			t.Fatalf("quest %q for a brand-new user: progress=%d completed=%v, want 0/false", v.Title, v.Progress, v.Completed)
		}
		if v.ResetAt.IsZero() {
			t.Fatalf("quest %q has a zero reset_at, want a computed next-midnight default", v.Title)
		}
	}

	if err := svc.RecordTrade(ctx, userID); err != nil {
		t.Fatalf("RecordTrade: %v", err)
	}

	views, err = svc.ListUserQuests(ctx, userID)
	if err != nil {
		t.Fatalf("ListUserQuests after a trade: %v", err)
	}
	found := false
	for _, v := range views {
		if v.Title == "Make 3 trades today" {
			found = true
			if v.Progress != 1 {
				t.Fatalf("MAKE_TRADES progress after one trade = %d, want 1", v.Progress)
			}
			if v.TargetValue != 3 || v.RewardCurrency != 100 {
				t.Fatalf("MAKE_TRADES target/reward = %d/%d, want 3/100", v.TargetValue, v.RewardCurrency)
			}
		}
	}
	if !found {
		t.Fatal(`expected "Make 3 trades today" among the listed quests`)
	}
}
