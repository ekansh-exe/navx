package domain

import (
	"time"

	"github.com/google/uuid"
)

// Quest is a daily (or, later, weekly) objective players can complete for a
// small currency reward (§7).
type Quest struct {
	ID             uuid.UUID
	Title          string
	Description    *string
	Type           QuestType
	TargetValue    int32
	RewardCurrency int32
	ResetTime      QuestResetTime
	CreatedAt      time.Time
}

// UserQuest is one user's progress toward one Quest.
type UserQuest struct {
	UserID      uuid.UUID
	QuestID     uuid.UUID
	Progress    int32
	CompletedAt *time.Time
	ResetAt     time.Time
}
