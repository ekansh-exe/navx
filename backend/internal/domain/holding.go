package domain

import (
	"time"

	"github.com/google/uuid"
)

// Holding is a user's position in a card, keyed by (UserID, CardID) (§2).
type Holding struct {
	UserID       uuid.UUID
	CardID       uuid.UUID
	SharesOwned  int64 // never negative
	AvgCostBasis int64
	// FirstBoughtAt is set once, the first time this position went from
	// zero/nonexistent to nonzero, and never overwritten afterward — backs
	// the HOLD_CARD quest (§7). Nil if never bought (row wouldn't exist) or
	// for pre-migration rows that haven't traded since first_bought_at was
	// added.
	FirstBoughtAt *time.Time
}
