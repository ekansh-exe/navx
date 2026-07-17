// Package engine implements the bonding-curve pricing math (§4.1) as pure
// functions — no DB, no HTTP. Phase 3 wires these into real trade execution
// (row-locking, persistence); this package only computes numbers.
package engine

import (
	"fmt"
	"math"
)

// maxSupply is a sanity bound on circulating supply, guarding against
// int64 overflow (supply+1, supply*something) before values ever reach
// float math. Generous relative to today's seeded companies (~2M shares)
// but still a real ceiling — an UNLIMITED card is mintable-on-buy, not
// literally infinite (§4.4), so a bound is appropriate.
const maxSupply = int64(1e15)

// CurveParams bundles the per-card constants and current modifiers needed
// to price a card (§4.1): price = BasePrice * curve_multiplier(supply) *
// DemandModifier * DriftFactor. How BasePrice/Scale are sourced or stored
// per card (the schema has no base_price column yet) is a Phase 3 decision,
// not resolved here.
type CurveParams struct {
	BasePrice      float64
	Scale          float64
	DemandModifier float64
	DriftFactor    float64
}

func (p CurveParams) validate() error {
	if p.Scale <= 0 {
		return fmt.Errorf("scale must be > 0, got %v", p.Scale)
	}
	return nil
}

func validateSupply(supply int64) error {
	if supply < 0 || supply > maxSupply {
		return fmt.Errorf("supply out of range [0, %d]: %d", maxSupply, supply)
	}
	return nil
}

// CurveMultiplier returns a monotonically increasing multiplier for the
// given circulating supply using a square-root curve: sqrt((supply+1)/scale).
// The +1 offset keeps the multiplier positive (not zero) at zero supply, so
// price never collapses to zero before any shares are in circulation.
func CurveMultiplier(circulatingSupply int64, scale float64) (float64, error) {
	if err := validateSupply(circulatingSupply); err != nil {
		return 0, err
	}
	if scale <= 0 {
		return 0, fmt.Errorf("scale must be > 0, got %v", scale)
	}
	return math.Sqrt(float64(circulatingSupply+1) / scale), nil
}

// SpotPrice is BasePrice * curve_multiplier(supply) * DemandModifier *
// DriftFactor, rounded to the nearest smallest currency unit.
func SpotPrice(circulatingSupply int64, p CurveParams) (int64, error) {
	mult, err := CurveMultiplier(circulatingSupply, p.Scale)
	if err != nil {
		return 0, err
	}
	return roundToInt64(p.BasePrice * mult * p.DemandModifier * p.DriftFactor)
}

// roundToInt64 is the boundary where a computed price/cost becomes a
// chargeable amount. Go's float->int conversion is implementation-defined
// (not a panic) for NaN, ±Inf, and out-of-range values — unacceptable on a
// money path — so those are rejected explicitly instead of silently
// producing a garbage int64.
func roundToInt64(raw float64) (int64, error) {
	if math.IsNaN(raw) || math.IsInf(raw, 0) {
		return 0, fmt.Errorf("invalid price/cost value: %v", raw)
	}
	rounded := math.Round(raw)
	if rounded > float64(math.MaxInt64) || rounded < float64(math.MinInt64) {
		return 0, fmt.Errorf("price/cost value out of int64 range: %v", rounded)
	}
	return int64(rounded), nil
}
