package quests

import (
	"context"
	"log/slog"
	"time"

	"github.com/ekansh-exe/navx/internal/safety"
	"github.com/ekansh-exe/navx/internal/store/db"
)

// defaultQuestResetCheckInterval is how often the reset job looks for rows
// whose reset window has passed. This isn't a literal "runs once a day at
// midnight" scheduler (which would need precise sleep-until-midnight logic
// and wouldn't self-heal after any downtime) — each row only actually
// resets once every 24h regardless of how often this ticks, since the
// bulk update only ever touches rows where reset_at <= now.
const defaultQuestResetCheckInterval = 1 * time.Hour

// ResetDueQuests bulk-resets every user_quests row whose reset window has
// already elapsed — progress=0, completed_at=NULL, reset_at=next UTC
// midnight — regardless of whether that quest was completed, so "make 3
// trades today" is a fresh target for every user each day, not just the
// ones who finished it. Returns the number of rows reset.
func (s *Service) ResetDueQuests(ctx context.Context, now time.Time) (int64, error) {
	return s.queries.ResetDueUserQuests(ctx, db.ResetDueUserQuestsParams{
		ResetAt:   now,
		ResetAt_2: nextMidnightUTC(now),
	})
}

// RunResetJob periodically resets any due user_quests rows until ctx is
// cancelled — see ResetDueQuests and defaultQuestResetCheckInterval's doc
// comments for why this polls rather than firing exactly at midnight.
func (s *Service) RunResetJob(ctx context.Context, interval time.Duration) {
	resetOnce := func() {
		defer safety.Recover("quest_reset")
		now := time.Now().UTC()
		n, err := s.ResetDueQuests(ctx, now)
		if err != nil {
			slog.Error("QUEST_RESET_ERROR", "error", err)
			return
		}
		if n > 0 {
			slog.Info("QUESTS_RESET", "rows", n)
		}
	}

	resetOnce()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			resetOnce()
		}
	}
}
