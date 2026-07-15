package domain

import (
	"time"

	"github.com/google/uuid"
)

// PriceTick is a historical price/volume sample for charting (§2).
type PriceTick struct {
	CardID    uuid.UUID
	Price     int64
	Volume    int64
	Timestamp time.Time
}
