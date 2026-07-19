package engine

// UpdateDemandEMA advances the demand EMA by one tick using standard
// exponential smoothing: alpha in (0,1] controls how quickly the EMA
// reacts to new signal vs. its own history.
//
// netDirection is normalized net buy volume for this tick, expected in
// [-1, 1] (e.g. (buyVolume-sellVolume)/(buyVolume+sellVolume), 0 on an idle
// tick) — not just a ±1 direction flag, since §4.1's "based on recent net
// buy volume" implies magnitude matters, not only sign.
//
// Contract: call this once per tick (with netDirection=0 on ticks with no
// trades), not once per trade — "decays over time" (§4.1) only holds if
// idle ticks are fed through too; skipping them freezes the EMA instead of
// decaying it. Initialize prevEMA to 0 before the first tick.
func UpdateDemandEMA(prevEMA, netDirection, alpha float64) float64 {
	return alpha*netDirection + (1-alpha)*prevEMA
}

// DemandModifier converts an EMA value into the multiplicative factor used
// in SpotPrice/ExecutionCost, clamped to [1+minEMA, 1+maxEMA] so demand
// can't push the multiplier to zero or to an extreme multiple. Neutral
// (==1) at ema=0.
func DemandModifier(ema, minEMA, maxEMA float64) float64 {
	return 1 + clampFloat(ema, minEMA, maxEMA)
}
