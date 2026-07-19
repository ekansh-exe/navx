package quests

import (
	"context"
	"log/slog"
	"time"

	"github.com/ekansh-exe/navx/internal/domain"
)

// holdCardQuestDuration is §7's "hold a card for 24h".
const holdCardQuestDuration = 24 * time.Hour

// CheckHoldCardQuests finds every HUMAN user currently holding a nonzero
// position first bought at least 24h ago, and completes their HOLD_CARD
// quest for this period. Safe to call repeatedly — applyProgress no-ops
// once a user's quest is already completed for the current period.
func (s *Service) CheckHoldCardQuests(ctx context.Context) error {
	cutoff := time.Now().UTC().Add(-holdCardQuestDuration)
	userIDs, err := s.queries.ListHumanUsersWithQualifyingHolding(ctx, &cutoff)
	if err != nil {
		return err
	}
	for _, userID := range userIDs {
		if err := s.completeIfConditionMet(ctx, userID, domain.QuestTypeHoldCard); err != nil {
			slog.Error("QUEST_HOLD_CARD_CHECK_ERROR", "user_id", userID, "error", err)
		}
	}
	return nil
}

// RunHoldCardCheckJob periodically evaluates the HOLD_CARD quest for every
// eligible user until ctx is cancelled. This polls rather than spawning a
// literal per-user "24h timer" goroutine (which wouldn't survive a process
// restart and doesn't scale past a handful of users) — matching this
// codebase's existing ticker-based background job pattern (bots'
// RunRebalanceJob, leaderboard's Run, news' Run).
func (s *Service) RunHoldCardCheckJob(ctx context.Context, interval time.Duration) {
	if err := s.CheckHoldCardQuests(ctx); err != nil {
		slog.Error("QUEST_HOLD_CARD_CHECK_ERROR", "error", err)
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.CheckHoldCardQuests(ctx); err != nil {
				slog.Error("QUEST_HOLD_CARD_CHECK_ERROR", "error", err)
			}
		}
	}
}
