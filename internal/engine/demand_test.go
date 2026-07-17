package engine

import "testing"

func TestUpdateDemandEMA_DecaysTowardZeroWithNoSignal(t *testing.T) {
	ema := 0.8
	prevAbs := ema
	for i := 0; i < 20; i++ {
		ema = UpdateDemandEMA(ema, 0, 0.3)
		abs := ema
		if abs < 0 {
			abs = -abs
		}
		if abs >= prevAbs {
			t.Fatalf("tick %d: |ema| = %v, want strictly less than previous %v (should decay toward 0)", i, abs, prevAbs)
		}
		prevAbs = abs
	}
	if prevAbs > 0.01 {
		t.Fatalf("ema = %v after 20 zero-signal ticks, want within 0.01 of 0", ema)
	}
}

func TestUpdateDemandEMA_TracksHeldConstantSignal(t *testing.T) {
	ema := 0.0
	const signal = 0.6
	for i := 0; i < 50; i++ {
		ema = UpdateDemandEMA(ema, signal, 0.2)
	}
	diff := ema - signal
	if diff < 0 {
		diff = -diff
	}
	if diff > 0.01 {
		t.Fatalf("ema = %v after 50 ticks of a held signal %v, want within 0.01", ema, signal)
	}
}

func TestDemandModifier_NeutralAtZeroEMA(t *testing.T) {
	if got := DemandModifier(0, -0.5, 0.5); got != 1 {
		t.Fatalf("DemandModifier(0, ...) = %v, want 1", got)
	}
}

func TestDemandModifier_ClampsExtremeEMA(t *testing.T) {
	if got := DemandModifier(10, -0.5, 0.5); got != 1.5 {
		t.Fatalf("DemandModifier(10, -0.5, 0.5) = %v, want 1.5 (clamped)", got)
	}
	if got := DemandModifier(-10, -0.5, 0.5); got != 0.5 {
		t.Fatalf("DemandModifier(-10, -0.5, 0.5) = %v, want 0.5 (clamped)", got)
	}
}
