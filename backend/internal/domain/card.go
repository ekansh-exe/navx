package domain

import (
	"time"

	"github.com/google/uuid"
)

// Card covers system companies, the NAV5 index, and user-created cards — one
// table/type distinguished by Type (§2).
type Card struct {
	ID                        uuid.UUID
	CreatorUserID             *uuid.UUID // nil for SYSTEM_COMPANY and INDEX cards
	Type                      CardType
	Sector                    *string // nil for user-created and the INDEX card
	Symbol                    string
	Name                      string
	Description               *string // nil for the 30 seeded companies and NAV5; set at launch for USER_CREATED cards
	ImageURL                  *string
	SupplyModel               SupplyModel
	TotalSupply               *int64 // nil when SupplyModel is UNLIMITED
	CirculatingSupply         int64
	CreatorRetainedShares     int64
	CreatorRetainedSharesSold int64   // how much of CreatorRetainedShares the creator has sold so far (§4.3 vesting)
	BasePrice                 float64 // engine.CurveParams anchor (§4.1) — a pricing parameter, not a currency amount
	Scale                     float64 // engine.CurveParams anchor (§4.1) — a pricing parameter, not a currency amount
	CurrentPrice              int64   // denormalized cache; price engine (Phase 2+) is the source of truth
	Status                    CardStatus
	CreatedAt                 time.Time
}
