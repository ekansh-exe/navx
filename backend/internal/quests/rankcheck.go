package quests

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/leaderboard"
	"github.com/ekansh-exe/navx/internal/safety"
)

// reachRankQuestThreshold is §7's "reach rank 50 on the leaderboard".
const reachRankQuestThreshold = 50

// CheckRankQuests reads the leaderboard's cache (populated independently by
// internal/leaderboard's own refresh job — this package never touches that
// job's logic, only its already-exported read path) and completes the
// REACH_RANK quest for every user at or better than rank 50. Bots are
// already excluded from the leaderboard itself (§4.5/§8), so nothing extra
// is needed here to keep them out of this quest either.
func (s *Service) CheckRankQuests(ctx context.Context, redisClient *redis.Client) error {
	defer safety.Recover("quest_rank_check")

	entries, err := leaderboard.ReadCached(ctx, redisClient)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Rank > reachRankQuestThreshold {
			continue
		}
		if err := s.completeIfConditionMet(ctx, e.UserID, domain.QuestTypeReachRank); err != nil {
			slog.Error("QUEST_RANK_CHECK_ERROR", "user_id", e.UserID, "error", err)
		}
	}
	return nil
}

// RunRankCheckJob polls the leaderboard cache on its own schedule instead
// of hooking into leaderboard.refreshOnce directly, so internal/leaderboard
// never has to change for this feature. In practice this trails a real
// refresh by at most one poll interval — imperceptible for a daily quest.
func (s *Service) RunRankCheckJob(ctx context.Context, redisClient *redis.Client, interval time.Duration) {
	if err := s.CheckRankQuests(ctx, redisClient); err != nil {
		slog.Error("QUEST_RANK_CHECK_ERROR", "error", err)
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.CheckRankQuests(ctx, redisClient); err != nil {
				slog.Error("QUEST_RANK_CHECK_ERROR", "error", err)
			}
		}
	}
}
