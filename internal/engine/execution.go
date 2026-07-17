package engine

import "math"

// ExecutionCost integrates the curve across a trade instead of pricing the
// whole order at one snapshot price (§4.2.3) — the continuous-limit
// approximation of "price each unit share at its own supply level and sum".
// Both are valid models but give slightly different numbers; don't
// "simplify" this into a discrete per-share loop expecting an identical
// result.
//
// deltaShares > 0 is a buy (supply increases); < 0 is a sell (supply
// decreases). The returned cost is negative for a sell — that's proceeds
// owed to the seller, not a charge:
//
//	ExecutionCost(1000, 50, p)  -> positive: currency the buyer pays
//	ExecutionCost(1000, -50, p) -> negative: currency credited to the seller
//
// Callers must apply the sign as-is (e.g. balance -= cost works for both
// directions) rather than assuming the result is always a debit.
func ExecutionCost(startSupply, deltaShares int64, p CurveParams) (int64, error) {
	if err := p.validate(); err != nil {
		return 0, err
	}
	endSupply := startSupply + deltaShares
	if err := validateSupply(startSupply); err != nil {
		return 0, err
	}
	if err := validateSupply(endSupply); err != nil {
		return 0, err
	}
	if deltaShares == 0 {
		return 0, nil
	}

	// curve_multiplier(s) = sqrt((s+1)/scale) has antiderivative
	// (2/3)*sqrt(1/scale)*(s+1)^1.5. Computing (end+1)^1.5 - (start+1)^1.5
	// directly loses precision at extreme supply (subtracting two large,
	// nearly-equal powers). Using a=end+1, b=start+1 and the identity
	// a^1.5-b^1.5 = (a-b)*(a+sqrt(a*b)+b)/(sqrt(a)+sqrt(b)) — where a-b is
	// exactly deltaShares, an integer, computed with no cancellation at
	// all — keeps this exact at any supply magnitude.
	a := float64(endSupply + 1)
	b := float64(startSupply + 1)
	sqrtA := math.Sqrt(a)
	sqrtB := math.Sqrt(b)
	diffPow := float64(deltaShares) * (a + sqrtA*sqrtB + b) / (sqrtA + sqrtB)

	raw := p.BasePrice * p.DemandModifier * p.DriftFactor * (2.0 / 3.0) * diffPow / math.Sqrt(p.Scale)
	return roundToInt64(raw)
}
