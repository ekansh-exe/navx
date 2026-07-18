package bots

import (
	"math"
	"math/rand"
	"testing"

	"github.com/google/uuid"

	"github.com/ekansh-exe/navx/internal/domain"
)

func TestTrendPercent(t *testing.T) {
	tests := []struct {
		name      string
		history   []int64
		wantOK    bool
		wantTrend float64
	}{
		{"not enough history", []int64{100, 105}, false, 0},
		{"flat", []int64{100, 100, 100, 100, 100}, true, 0},
		{"up 20%", []int64{100, 105, 110, 115, 120}, true, 0.20},
		{"down 20%", []int64{100, 95, 90, 85, 80}, true, -0.20},
		{"only the trailing window counts", []int64{9999, 100, 105, 110, 115, 120}, true, 0.20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := trendPercent(tt.history)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && math.Abs(got-tt.wantTrend) > 1e-9 {
				t.Fatalf("trend = %v, want %v", got, tt.wantTrend)
			}
		})
	}
}

func TestDecideMomentum(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	uptrend := uuid.New()
	downtrend := uuid.New()
	flat := uuid.New()

	t.Run("buys the strongest uptrend", func(t *testing.T) {
		snapshots := []MarketSnapshot{
			{CardID: flat, History: []int64{100, 100, 100, 100, 100}},
			{CardID: uptrend, History: []int64{100, 105, 110, 115, 120}},
		}
		d := decideMomentum(rng, snapshots)
		if d == nil || d.CardID != uptrend || d.Type != domain.TransactionTypeBuy {
			t.Fatalf("decision = %+v, want BUY on uptrend card", d)
		}
	})

	t.Run("sells a held downtrend when no uptrend exists", func(t *testing.T) {
		snapshots := []MarketSnapshot{
			{CardID: downtrend, History: []int64{100, 95, 90, 85, 80}, SharesOwned: 50},
		}
		d := decideMomentum(rng, snapshots)
		if d == nil || d.CardID != downtrend || d.Type != domain.TransactionTypeSell {
			t.Fatalf("decision = %+v, want SELL on downtrend card", d)
		}
		if d.Shares > 50 {
			t.Fatalf("sell shares %d exceeds owned 50", d.Shares)
		}
	})

	t.Run("skips a downtrend it doesn't own", func(t *testing.T) {
		snapshots := []MarketSnapshot{
			{CardID: downtrend, History: []int64{100, 95, 90, 85, 80}, SharesOwned: 0},
		}
		if d := decideMomentum(rng, snapshots); d != nil {
			t.Fatalf("expected nil decision, got %+v", d)
		}
	})

	t.Run("skips when history is too short", func(t *testing.T) {
		snapshots := []MarketSnapshot{{CardID: uptrend, History: []int64{100, 120}}}
		if d := decideMomentum(rng, snapshots); d != nil {
			t.Fatalf("expected nil decision, got %+v", d)
		}
	})
}

func TestDecideContrarian(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	uptrend := uuid.New()
	downtrend := uuid.New()

	t.Run("buys the dip", func(t *testing.T) {
		snapshots := []MarketSnapshot{
			{CardID: downtrend, History: []int64{100, 95, 90, 85, 80}},
		}
		d := decideContrarian(rng, snapshots)
		if d == nil || d.CardID != downtrend || d.Type != domain.TransactionTypeBuy {
			t.Fatalf("decision = %+v, want BUY on the dip", d)
		}
	})

	t.Run("sells a held rally", func(t *testing.T) {
		snapshots := []MarketSnapshot{
			{CardID: uptrend, History: []int64{100, 105, 110, 115, 120}, SharesOwned: 30},
		}
		d := decideContrarian(rng, snapshots)
		if d == nil || d.CardID != uptrend || d.Type != domain.TransactionTypeSell {
			t.Fatalf("decision = %+v, want SELL on the rally", d)
		}
		if d.Shares > 30 {
			t.Fatalf("sell shares %d exceeds owned 30", d.Shares)
		}
	})

	t.Run("skips a rally it doesn't own", func(t *testing.T) {
		snapshots := []MarketSnapshot{
			{CardID: uptrend, History: []int64{100, 105, 110, 115, 120}, SharesOwned: 0},
		}
		if d := decideContrarian(rng, snapshots); d != nil {
			t.Fatalf("expected nil decision, got %+v", d)
		}
	})
}

func TestDecideRandomWalker(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	cardA := uuid.New()

	t.Run("nil when no active cards", func(t *testing.T) {
		if d := decideRandomWalker(rng, nil); d != nil {
			t.Fatalf("expected nil, got %+v", d)
		}
	})

	t.Run("buys when it owns nothing", func(t *testing.T) {
		snapshots := []MarketSnapshot{{CardID: cardA, SharesOwned: 0, CirculatingSupply: 100_000}}
		d := decideRandomWalker(rng, snapshots)
		if d == nil || d.Type != domain.TransactionTypeBuy || d.CardID != cardA {
			t.Fatalf("decision = %+v, want BUY on the only card", d)
		}
		if d.Shares < minTradeShares || d.Shares > maxTradeShares {
			t.Fatalf("shares %d out of expected range [%d,%d]", d.Shares, minTradeShares, maxTradeShares)
		}
	})

	t.Run("sell size never exceeds owned shares", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			snapshots := []MarketSnapshot{{CardID: cardA, SharesOwned: 3}}
			d := decideRandomWalker(rng, snapshots)
			if d == nil {
				t.Fatal("expected a decision")
			}
			if d.Type == domain.TransactionTypeSell && d.Shares > 3 {
				t.Fatalf("sell shares %d exceeds owned 3", d.Shares)
			}
		}
	})
}

func TestDecideNewsReactive(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	cardA := uuid.New()
	const supply = 1_000_000

	for i := 0; i < 200; i++ {
		snapshots := []MarketSnapshot{{CardID: cardA, SharesOwned: 5, CirculatingSupply: supply}}
		d := decideNewsReactive(rng, snapshots)
		if d == nil {
			t.Fatal("expected a decision, got nil")
		}
		if d.Type == domain.TransactionTypeSell && d.Shares > 5 {
			t.Fatalf("sell shares %d exceeds owned 5", d.Shares)
		}
		if d.Shares < minTradeShares || d.Shares > maxTradeShares {
			t.Fatalf("shares %d out of any expected range", d.Shares)
		}
	}
}

func TestSizeTrade_WithinPercentRange(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	const supply = 1_000_000
	wantMin := int64(supply * minTradePercentOfSupply)
	wantMax := int64(supply * maxTradePercentOfSupply)
	for i := 0; i < 200; i++ {
		got := sizeTrade(rng, supply)
		if got < wantMin || got > wantMax {
			t.Fatalf("sizeTrade = %d, want within [%d,%d]", got, wantMin, wantMax)
		}
	}
}

func TestBurstShares_LargerThanNormalTradeRange(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	const supply = 1_000_000
	normalMax := int64(supply * maxTradePercentOfSupply)
	burstMin := int64(supply * newsBurstMinPercentOfSupply)
	if burstMin <= normalMax {
		t.Fatalf("burst range (min %d) should start above the normal trade range (max %d) for a clear distinction", burstMin, normalMax)
	}
	for i := 0; i < 200; i++ {
		got := burstShares(rng, supply)
		if got < burstMin || got > int64(supply*newsBurstMaxPercentOfSupply) {
			t.Fatalf("burstShares = %d, out of expected burst range", got)
		}
	}
}

func TestSizeTrade_ClampsToAbsoluteBounds(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	// A tiny supply should still produce at least minTradeShares.
	if got := sizeTrade(rng, 1); got < minTradeShares {
		t.Fatalf("sizeTrade with supply=1 = %d, want at least %d", got, minTradeShares)
	}
	// A huge supply should still be capped at maxTradeShares.
	if got := sizeTrade(rng, 1_000_000_000); got > maxTradeShares {
		t.Fatalf("sizeTrade with supply=1e9 = %d, want at most %d", got, maxTradeShares)
	}
}

func TestDecideIndexTracker(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	indexCard := uuid.New()
	nonIndexCard := uuid.New()

	t.Run("buys when undervalued vs derived price", func(t *testing.T) {
		snapshots := []MarketSnapshot{
			{CardID: indexCard, CardType: domain.CardTypeIndex, CurrentPrice: 100},
			{CardID: nonIndexCard, CardType: domain.CardTypeSystemCompany, CurrentPrice: 50},
		}
		derived := map[uuid.UUID]int64{indexCard: 110}
		d := decideIndexTracker(rng, snapshots, derived)
		if d == nil || d.CardID != indexCard || d.Type != domain.TransactionTypeBuy {
			t.Fatalf("decision = %+v, want BUY on index card", d)
		}
	})

	t.Run("sells when overvalued and holding shares", func(t *testing.T) {
		snapshots := []MarketSnapshot{
			{CardID: indexCard, CardType: domain.CardTypeIndex, CurrentPrice: 100, SharesOwned: 20},
		}
		derived := map[uuid.UUID]int64{indexCard: 90}
		d := decideIndexTracker(rng, snapshots, derived)
		if d == nil || d.Type != domain.TransactionTypeSell {
			t.Fatalf("decision = %+v, want SELL", d)
		}
	})

	t.Run("skips when within threshold", func(t *testing.T) {
		snapshots := []MarketSnapshot{
			{CardID: indexCard, CardType: domain.CardTypeIndex, CurrentPrice: 100},
		}
		derived := map[uuid.UUID]int64{indexCard: 101}
		if d := decideIndexTracker(rng, snapshots, derived); d != nil {
			t.Fatalf("expected nil, got %+v", d)
		}
	})

	t.Run("skips non-index cards entirely", func(t *testing.T) {
		snapshots := []MarketSnapshot{
			{CardID: nonIndexCard, CardType: domain.CardTypeSystemCompany, CurrentPrice: 50},
		}
		derived := map[uuid.UUID]int64{nonIndexCard: 1000}
		if d := decideIndexTracker(rng, snapshots, derived); d != nil {
			t.Fatalf("expected nil (not an index card), got %+v", d)
		}
	})
}
