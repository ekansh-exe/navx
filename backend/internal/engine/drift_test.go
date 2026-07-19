package engine

import "testing"

func TestNextDriftFactor_StaysWithinClampBounds(t *testing.T) {
	const min, max = 0.8, 1.2
	prev := 1.0
	inputs := []float64{1, -1, 1, 1, -1, -1, 1, -1, 1, 1}
	for i, randUnit := range inputs {
		prev = NextDriftFactor(prev, randUnit, 0.5, 0.02, min, max)
		if prev < min || prev > max {
			t.Fatalf("tick %d: drift factor = %v, want within [%v, %v]", i, prev, min, max)
		}
	}
}

func TestNextDriftFactor_MeanRevertsTowardOneWithNoRandomShock(t *testing.T) {
	const min, max = 0.5, 1.5
	prev := max // start as far from neutral as the clamp allows
	initialDistance := prev - 1.0

	for i := 0; i < 200; i++ {
		prev = NextDriftFactor(prev, 0, 0.5, 0.05, min, max)
	}

	finalDistance := prev - 1.0
	if finalDistance < 0 {
		finalDistance = -finalDistance
	}
	if finalDistance >= initialDistance {
		t.Fatalf("expected mean reversion to shrink distance from 1.0 (started at %v), got %v after 200 ticks", initialDistance, prev)
	}
	// With no random shock at all, reversion alone should pull it very
	// close to neutral after this many ticks.
	if finalDistance > 0.01 {
		t.Fatalf("drift factor = %v after 200 zero-shock ticks, want within 0.01 of 1.0", prev)
	}
}
