package engine

import (
	"math"
	"testing"
)

// naiveExecutionCost computes the same integral as ExecutionCost but via
// direct subtraction of the two powers (no cancellation-safe identity), so
// tests can confirm the two formulations agree.
func naiveExecutionCost(startSupply, deltaShares int64, p CurveParams) float64 {
	endSupply := startSupply + deltaShares
	a := float64(endSupply + 1)
	b := float64(startSupply + 1)
	return p.BasePrice * p.DemandModifier * p.DriftFactor * (2.0 / 3.0) * (math.Pow(a, 1.5) - math.Pow(b, 1.5)) / math.Sqrt(p.Scale)
}

func TestExecutionCost_BuyPositiveSellNegative(t *testing.T) {
	params := CurveParams{BasePrice: 5, Scale: 1000, DemandModifier: 1, DriftFactor: 1}
	const startSupply = 100_000
	const size = 500

	buyCost, err := ExecutionCost(startSupply, size, params)
	if err != nil {
		t.Fatalf("buy: %v", err)
	}
	if buyCost <= 0 {
		t.Fatalf("buy cost = %d, want > 0", buyCost)
	}

	sellCost, err := ExecutionCost(startSupply, -size, params)
	if err != nil {
		t.Fatalf("sell: %v", err)
	}
	if sellCost >= 0 {
		t.Fatalf("sell cost = %d, want < 0 (proceeds)", sellCost)
	}

	// The curve is increasing, so buying (integrating [start, start+size])
	// covers strictly higher prices than selling the same size (integrating
	// [start-size, start]) — the buyer pays more than the seller receives.
	if buyCost <= -sellCost {
		t.Fatalf("expected buy cost (%d) > sell proceeds (%d)", buyCost, -sellCost)
	}
}

func TestExecutionCost_ZeroDeltaIsFreeNoOp(t *testing.T) {
	params := CurveParams{BasePrice: 5, Scale: 1000, DemandModifier: 1, DriftFactor: 1}
	got, err := ExecutionCost(1000, 0, params)
	if err != nil {
		t.Fatalf("ExecutionCost with deltaShares=0: %v", err)
	}
	if got != 0 {
		t.Fatalf("ExecutionCost with deltaShares=0 = %d, want 0", got)
	}
}

func TestExecutionCost_InvalidSupplyRange(t *testing.T) {
	params := CurveParams{BasePrice: 5, Scale: 1000, DemandModifier: 1, DriftFactor: 1}

	if _, err := ExecutionCost(-1, 10, params); err == nil {
		t.Fatal("negative startSupply: expected an error, got none")
	}
	if _, err := ExecutionCost(100, -200, params); err == nil {
		t.Fatal("selling past zero circulating supply: expected an error, got none")
	}
	if _, err := ExecutionCost(maxSupply, 1, params); err == nil {
		t.Fatal("endSupply over the sanity bound: expected an error, got none")
	}
}

func TestExecutionCost_InvalidParams(t *testing.T) {
	if _, err := ExecutionCost(1000, 100, CurveParams{Scale: 0}); err == nil {
		t.Fatal("scale=0: expected an error, got none")
	}
}

func TestExecutionCost_MatchesNaiveFormulaAtModerateSupply(t *testing.T) {
	params := CurveParams{BasePrice: 3, Scale: 50, DemandModifier: 1.05, DriftFactor: 0.97}
	const startSupply = 1000
	const size = 200

	got, err := ExecutionCost(startSupply, size, params)
	if err != nil {
		t.Fatalf("ExecutionCost: %v", err)
	}
	want, err := roundToInt64(naiveExecutionCost(startSupply, size, params))
	if err != nil {
		t.Fatalf("naive formula: %v", err)
	}
	if got != want {
		t.Fatalf("cancellation-safe formula = %d, naive formula = %d, expected them to agree", got, want)
	}
}

// TestExecutionCost_ConvergesToSpotPriceAtLargeSupply checks the calculus
// property that a one-share trade's average price approaches the spot
// price as supply grows — but only at large supply. At small supply the
// curve's concavity means a 1-share step's average is measurably biased
// toward the endpoint (the gap is roughly 0.25/sqrt(scale*(supply+1))
// relative to the multiplier — ~25% at supply=0, ~0.02% at supply=2,000,000
// for scale=1), so asserting near-equality there would be testing a
// property that provably doesn't hold yet, not catching a real bug.
func TestExecutionCost_ConvergesToSpotPriceAtLargeSupply(t *testing.T) {
	params := CurveParams{BasePrice: 100, Scale: 1, DemandModifier: 1, DriftFactor: 1}
	const largeSupply = 10_000_000

	spot, err := SpotPrice(largeSupply, params)
	if err != nil {
		t.Fatalf("SpotPrice: %v", err)
	}
	oneShareCost, err := ExecutionCost(largeSupply, 1, params)
	if err != nil {
		t.Fatalf("ExecutionCost: %v", err)
	}

	relDiff := math.Abs(float64(oneShareCost-spot)) / float64(spot)
	const tolerance = 0.001 // 0.1%
	if relDiff > tolerance {
		t.Fatalf("one-share execution cost %d vs spot price %d: relative diff %.6f exceeds tolerance %v", oneShareCost, spot, relDiff, tolerance)
	}
}
