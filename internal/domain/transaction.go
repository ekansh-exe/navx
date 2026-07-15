package domain

import (
	"time"

	"github.com/google/uuid"
)

// Transaction is the append-only ledger row — the actual source of truth for
// a user's balance (§2). Every currency-affecting event (trade, reward, fee)
// must go through this one path; user.currency_balance must always equal the
// sum of a user's transaction deltas.
type Transaction struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	CardID             *uuid.UUID
	Type               TransactionType
	Shares             *int64
	PricePerShare      *int64
	TotalCurrencyDelta int64
	ResultingBalance   int64
	IdempotencyKey     *string
	CreatedAt          time.Time
}
