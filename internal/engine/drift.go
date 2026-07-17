package engine

// NextDriftFactor advances the drift factor by one tick: a small bounded
// random walk (§4.1) with gentle mean reversion toward 1.0 (neutral). A
// pure random walk confined to [min,max] has no restoring force and tends
// to drift to and sit near the bounds rather than "hover near 1.0" the way
// §4.1's prose implies — the reversionRate term corrects for that.
//
// randUnit must be in [-1, 1]; the caller supplies randomness (e.g. via
// math/rand) so this function itself stays pure and testable. maxStepPercent
// bounds the random step size per tick; reversionRate (small, e.g.
// 0.01-0.05) pulls the value back toward 1.0 each tick independent of the
// random shock; the result is clamped to [min, max].
//
// Initialize prev to 1.0 before the first tick (neutral, no drift applied
// yet).
func NextDriftFactor(prev, randUnit, maxStepPercent, reversionRate, min, max float64) float64 {
	next := prev + randUnit*maxStepPercent*prev - reversionRate*(prev-1.0)
	return clampFloat(next, min, max)
}

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
