package domain

import "github.com/google/uuid"

// Holding is a user's position in a card, keyed by (UserID, CardID) (§2).
type Holding struct {
	UserID       uuid.UUID
	CardID       uuid.UUID
	SharesOwned  int64 // never negative
	AvgCostBasis int64
}
