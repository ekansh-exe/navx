package domain

import (
	"time"

	"github.com/google/uuid"
)

// User is a player account (§2). NetWorth is derived (cash + holdings valued
// at current price) and is never stored on this struct — it's recomputed by
// the leaderboard job (§8) from CurrencyBalance plus Holding rows.
type User struct {
	ID               uuid.UUID
	Username         string
	UserType         UserType
	CurrencyBalance  int64 // smallest currency unit, never negative
	LoginStreakCount int
	LastLoginAt      *time.Time
	CreatedAt        time.Time
}
