package domain

import "github.com/google/uuid"

// IndexComponent is one constituent of the NAV5 index card (§2, §5.2).
// Membership and weights are recomputed on a schedule, not every tick.
type IndexComponent struct {
	IndexCardID     uuid.UUID
	ComponentCardID uuid.UUID
	Weight          float64 // market-cap-weighted, normalized so all components sum to 1
}
