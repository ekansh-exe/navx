package quests

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/leaderboard"
)

// This test writes leaderboard entries directly to the same Redis key/shape
// internal/leaderboard's own refresh job writes, without running that job
// — this package only ever reads that cache (leaderboard.ReadCached), so
// the test exercises exactly that read path in isolation.
func TestCheckRankQuests_GrantsAtOrUnderThresholdOnly(t *testing.T) {
	pool := testPool(t)
	redisClient := testRedis(t)
	svc := New(pool)
	ctx := context.Background()

	qualifyingUser := createTestUser(t, pool, "HUMAN", 100_000)
	nonQualifyingUser := createTestUser(t, pool, "HUMAN", 100_000)

	entries := []leaderboard.Entry{
		{Rank: 10, UserID: qualifyingUser, Username: "a", NetWorth: 1000},
		{Rank: 51, UserID: nonQualifyingUser, Username: "b", NetWorth: 500},
	}
	data, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshal entries: %v", err)
	}
	if err := redisClient.Set(ctx, leaderboard.CacheKey, data, 0).Err(); err != nil {
		t.Fatalf("seed leaderboard cache: %v", err)
	}
	t.Cleanup(func() { redisClient.Del(context.Background(), leaderboard.CacheKey) })

	if err := svc.CheckRankQuests(ctx, redisClient); err != nil {
		t.Fatalf("CheckRankQuests: %v", err)
	}

	questID := questIDForType(t, pool, string(domain.QuestTypeReachRank))

	_, completed, _, found := getUserQuestRow(t, pool, qualifyingUser, questID)
	if !found || !completed {
		t.Fatal("expected rank-10 user's REACH_RANK quest to be completed")
	}
	if n := countTransactions(t, pool, qualifyingUser, "QUEST_REWARD"); n != 1 {
		t.Fatalf("QUEST_REWARD transactions for rank-10 user = %d, want 1", n)
	}

	if n := countTransactions(t, pool, nonQualifyingUser, "QUEST_REWARD"); n != 0 {
		t.Fatalf("QUEST_REWARD transactions for rank-51 user = %d, want 0", n)
	}
}
