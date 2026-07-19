package quests

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// View is one quest's state as a specific user should see it — the
// read-side shape GET /api/quests serializes.
type View struct {
	ID             uuid.UUID
	Title          string
	Progress       int32
	TargetValue    int32
	RewardCurrency int32
	Completed      bool
	ResetAt        time.Time
}

// ListUserQuests returns every quest with userID's current progress
// against it. A quest the user has never made progress on yet (no
// user_quests row) is presented as fresh (progress 0, not completed).
//
// If a row's reset_at has already elapsed but the periodic reset job
// hasn't physically reset it yet, this presents it as fresh too — a plain
// read should never show stale prior-period progress just because of
// timing between polls.
func (s *Service) ListUserQuests(ctx context.Context, userID uuid.UUID) ([]View, error) {
	rows, err := s.queries.ListUserQuestsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	views := make([]View, 0, len(rows))
	for _, r := range rows {
		view := View{
			ID:             r.QuestID,
			Title:          r.Title,
			TargetValue:    r.TargetValue,
			RewardCurrency: r.RewardCurrency,
			ResetAt:        nextMidnightUTC(now),
		}
		if r.Progress != nil && r.ResetAt != nil && r.ResetAt.After(now) {
			view.Progress = *r.Progress
			view.Completed = r.CompletedAt != nil
			view.ResetAt = *r.ResetAt
		}
		views = append(views, view)
	}
	return views, nil
}
