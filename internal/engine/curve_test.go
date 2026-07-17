package engine

import (
	"math"
	"testing"
)

func TestCurveMultiplier_Monotonic(t *testing.T) {
	supplies := []int64{0, 1, 10, 1000, 1_000_000, maxSupply}
	const scale = 1000.0

	prev := -math.MaxFloat64
	for _, s := range supplies {
		mult, err := CurveMultiplier(s, scale)
		if err != nil {
			t.Fatalf("CurveMultiplier(%d, %v): %v", s, scale, err)
		}
		if mult <= prev {
			t.Fatalf("CurveMultiplier(%d) = %v, want > previous %v (not monotonic)", s, mult, prev)
		}
		prev = mult
	}
}

func TestCurveMultiplier_ZeroSupplyIsPositive(t *testing.T) {
	mult, err := CurveMultiplier(0, 1.0)
	if err != nil {
		t.Fatalf("CurveMultiplier(0, 1): %v", err)
	}
	if mult <= 0 || math.IsNaN(mult) || math.IsInf(mult, 0) {
		t.Fatalf("CurveMultiplier(0, 1) = %v, want a positive finite value", mult)
	}
}

func TestCurveMultiplier_InvalidInputs(t *testing.T) {
	cases := []struct {
		name   string
		supply int64
		scale  float64
	}{
		{"negative supply", -1, 1.0},
		{"supply over sanity bound", maxSupply + 1, 1.0},
		{"zero scale", 100, 0},
		{"negative scale", 100, -5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := CurveMultiplier(c.supply, c.scale); err == nil {
				t.Fatalf("CurveMultiplier(%d, %v): expected an error, got none", c.supply, c.scale)
			}
		})
	}
}

func TestSpotPrice_HandComputedValues(t *testing.T) {
	cases := []struct {
		name   string
		supply int64
		params CurveParams
		want   int64
	}{
		{
			// (99+1)/1 = 100, sqrt(100) = 10, 10*10*1*1 = 100 exactly.
			name:   "clean multiplier, neutral modifiers",
			supply: 99,
			params: CurveParams{BasePrice: 10, Scale: 1, DemandModifier: 1, DriftFactor: 1},
			want:   100,
		},
		{
			// (17+1)/2 = 9, sqrt(9) = 3, 7*3*1.1*0.9 = 20.79 -> rounds to 21.
			name:   "fractional result rounds to nearest",
			supply: 17,
			params: CurveParams{BasePrice: 7, Scale: 2, DemandModifier: 1.1, DriftFactor: 0.9},
			want:   21,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := SpotPrice(c.supply, c.params)
			if err != nil {
				t.Fatalf("SpotPrice: %v", err)
			}
			if got != c.want {
				t.Fatalf("SpotPrice(%d, %+v) = %d, want %d", c.supply, c.params, got, c.want)
			}
		})
	}
}

func TestSpotPrice_RejectsNaNRatherThanRoundingGarbage(t *testing.T) {
	params := CurveParams{BasePrice: math.NaN(), Scale: 1, DemandModifier: 1, DriftFactor: 1}
	if _, err := SpotPrice(10, params); err == nil {
		t.Fatal("SpotPrice with a NaN BasePrice: expected an error, got none")
	}
}

func TestSpotPrice_PropagatesCurveMultiplierError(t *testing.T) {
	params := CurveParams{BasePrice: 10, Scale: 0, DemandModifier: 1, DriftFactor: 1}
	if _, err := SpotPrice(10, params); err == nil {
		t.Fatal("SpotPrice with scale=0: expected an error, got none")
	}
}
