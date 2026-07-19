package quests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ekansh-exe/navx/internal/domain"
)

// insertUserQuest writes a user_quests row directly with arbitrary state,
// so reset tests can simulate "yesterday's" progress without waiting for
// real time to pass.
func insertUserQuest(t *testing.T, pool *pgxpool.Pool, userID, questID uuid.UUID, progress int32, completedAt *time.Time, resetAt time.Time) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx,
		`INSERT INTO user_quests (user_id, quest_id, progress, completed_at, reset_at) VALUES ($1, $2, $3, $4, $5)`,
		userID, questID, progress, completedAt, resetAt,
	)
	if err != nil {
		t.Fatalf("insert test user_quests row: %v", err)
	}
}

// TestResetDueQuests_ResetsRegardlessOfCompletion covers the reset-scope
// decision (§7 extension): a DAILY quest's progress resets for every user
// once its period elapses, whether or not it was completed — "make 3
// trades today" is a fresh target every day, not just for people who
// finished it.
func TestResetDueQuests_ResetsRegardlessOfCompletion(t *testing.T) {
	pool := testPool(t)
	svc := New(pool)
	ctx := context.Background()

	completedUser := createTestUser(t, pool, "HUMAN", 100_000)
	partialUser := createTestUser(t, pool, "HUMAN", 100_000)
	notYetDueUser := createTestUser(t, pool, "HUMAN", 100_000)

	questID := questIDForType(t, pool, string(domain.QuestTypeMakeTrades))

	yesterday := time.Now().UTC().Add(-1 * time.Hour) // already past — due for reset
	tomorrow := time.Now().UTC().Add(24 * time.Hour)  // not due yet
	now := time.Now().UTC()

	insertUserQuest(t, pool, completedUser, questID, 3, &now, yesterday)
	insertUserQuest(t, pool, partialUser, questID, 2, nil, yesterday)
	insertUserQuest(t, pool, notYetDueUser, questID, 2, nil, tomorrow)

	if _, err := svc.ResetDueQuests(ctx, time.Now().UTC()); err != nil {
		t.Fatalf("ResetDueQuests: %v", err)
	}

	progress, completed, resetAt, found := getUserQuestRow(t, pool, completedUser, questID)
	if !found || progress != 0 || completed || !resetAt.After(time.Now().UTC()) {
		t.Fatalf("completed user after reset: progress=%d completed=%v resetAt=%v, want 0/false/future", progress, completed, resetAt)
	}

	progress, completed, resetAt, found = getUserQuestRow(t, pool, partialUser, questID)
	if !found || progress != 0 || completed || !resetAt.After(time.Now().UTC()) {
		t.Fatalf("partial-progress user after reset: progress=%d completed=%v resetAt=%v, want 0/false/future", progress, completed, resetAt)
	}

	progress, completed, resetAt, found = getUserQuestRow(t, pool, notYetDueUser, questID)
	// Postgres timestamptz has microsecond precision; tomorrow (in-memory)
	// carries nanoseconds, so allow sub-microsecond rounding slop rather
	// than requiring bit-for-bit equality after the DB round trip.
	if !found || progress != 2 || completed || resetAt.Sub(tomorrow).Abs() > time.Microsecond {
		t.Fatalf("not-yet-due user after reset: progress=%d completed=%v resetAt=%v, want unchanged 2/false/%v", progress, completed, resetAt, tomorrow)
	}
}

// TestApplyProgress_LazilyResetsAnAlreadyDueRow confirms the lazy-reset
// path inside applyProgress itself (not just the bulk background job): a
// user whose reset_at has already elapsed gets a fresh period the moment
// they next act, even if the reset job hasn't ticked yet.
func TestApplyProgress_LazilyResetsAnAlreadyDueRow(t *testing.T) {
	pool := testPool(t)
	svc := New(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool, "HUMAN", 100_000)
	questID := questIDForType(t, pool, string(domain.QuestTypeMakeTrades))

	now := time.Now().UTC()
	yesterday := now.Add(-1 * time.Hour)
	// Simulate a user who completed yesterday's quest and whose row hasn't
	// been swept by the background reset job yet.
	insertUserQuest(t, pool, userID, questID, 3, &now, yesterday)

	if err := svc.RecordTrade(ctx, userID); err != nil {
		t.Fatalf("RecordTrade: %v", err)
	}

	progress, completed, resetAt, found := getUserQuestRow(t, pool, userID, questID)
	if !found {
		t.Fatal("expected a user_quests row")
	}
	if progress != 1 {
		t.Fatalf("progress after one trade in a fresh period = %d, want 1 (lazy reset then +1)", progress)
	}
	if completed {
		t.Fatal("expected not completed after just one trade in the new period")
	}
	if !resetAt.After(time.Now().UTC()) {
		t.Fatalf("reset_at = %v, want a future time after lazy reset", resetAt)
	}
	if n := countTransactions(t, pool, userID, "QUEST_REWARD"); n != 0 {
		t.Fatalf("QUEST_REWARD transactions = %d, want 0 (not completed this period)", n)
	}
}
