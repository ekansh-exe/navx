package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/ekansh-exe/navx/internal/leaderboard"
)

func TestGetLeaderboard_ServesWhateverIsCached(t *testing.T) {
	r, _ := testRouter(t)
	redisClient := testRedisClient(t)
	ctx := context.Background()

	change := int64(500)
	want := []leaderboard.Entry{
		{Rank: 1, UserID: uuid.New(), Username: "top_dog", NetWorth: 1_000_000, ChangeFromLastRefresh: &change},
		{Rank: 2, UserID: uuid.New(), Username: "runner_up", NetWorth: 500_000},
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if err := redisClient.Set(ctx, leaderboard.CacheKey, data, 0).Err(); err != nil {
		t.Fatalf("seed redis cache: %v", err)
	}
	t.Cleanup(func() { redisClient.Del(context.Background(), leaderboard.CacheKey) })

	rec := doJSON(t, r, http.MethodGet, "/api/leaderboard", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp leaderboardResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Leaderboard) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(resp.Leaderboard))
	}
	if resp.Leaderboard[0].Username != "top_dog" || resp.Leaderboard[0].NetWorth != 1_000_000 {
		t.Fatalf("Leaderboard[0] = %+v, want top_dog/1000000", resp.Leaderboard[0])
	}
	if resp.Leaderboard[0].ChangeFromLastRefresh == nil || *resp.Leaderboard[0].ChangeFromLastRefresh != 500 {
		t.Fatalf("Leaderboard[0].ChangeFromLastRefresh = %v, want 500", resp.Leaderboard[0].ChangeFromLastRefresh)
	}
	if resp.Leaderboard[1].ChangeFromLastRefresh != nil {
		t.Fatalf("Leaderboard[1].ChangeFromLastRefresh = %v, want nil", *resp.Leaderboard[1].ChangeFromLastRefresh)
	}
}

func TestGetLeaderboard_EmptyWhenCacheEmpty(t *testing.T) {
	r, _ := testRouter(t)
	redisClient := testRedisClient(t)
	redisClient.Del(context.Background(), leaderboard.CacheKey)

	rec := doJSON(t, r, http.MethodGet, "/api/leaderboard", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 even with an empty cache, body = %s", rec.Code, rec.Body.String())
	}
	var resp leaderboardResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Leaderboard == nil {
		t.Fatal("expected a non-nil (possibly empty) leaderboard slice, got nil (would serialize as JSON null, not [])")
	}
}
