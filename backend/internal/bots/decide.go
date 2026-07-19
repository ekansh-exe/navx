package bots

import (
	"math/rand"

	"github.com/google/uuid"

	"github.com/ekansh-exe/navx/internal/domain"
)

// historyWindow is how many of a bot's own rolling price observations are
// needed before momentum/contrarian bots trust a trend signal — no
// price_ticks time series is wired up anywhere in the DB yet, so each bot
// tracks its own small in-memory window instead (see market.go).
const historyWindow = 5

// momentumThreshold is the minimum fractional move across historyWindow
// observations for momentum/contrarian bots to treat it as a real trend
// rather than noise.
const momentumThreshold = 0.002

// indexTrackerThreshold is how far NAV5 (or any INDEX card) must diverge
// from its derived price (§5.2) before the index-tracker nudges it back.
const indexTrackerThreshold = 0.02

// Trade sizes are a percentage of the target card's own circulating supply,
// not a fixed share count — seeded companies range from ~500K to 2M+
// shares outstanding, and the sqrt bonding curve (§4.1) means a fixed
// handful of shares (e.g. 1-20) moves the rounded integer price by exactly
// zero on a card that size. Scaling to supply keeps price impact visible
// regardless of a card's absolute scale. minTradeShares/maxTradeShares are
// sanity bounds (the position cap and balance checks in ExecuteTrade are
// still what actually keeps any of this safe).
const (
	minTradePercentOfSupply = 0.0010
	maxTradePercentOfSupply = 0.0080
	minTradeShares          = 1
	// maxTradeShares is a broad sanity ceiling, not a per-card tuning knob —
	// it must stay comfortably above what maxTradePercentOfSupply/burst
	// produce for any realistically-sized seeded card (up to a few million
	// shares outstanding), or it silently flattens the percentage scaling
	// this whole scheme exists for.
	maxTradeShares = 100_000
)

// buyBudgetFraction caps how much of its currency balance a bot commits to a
// single BUY. It serves two purposes: (1) it keeps orders affordable — the
// supply-percentage sizing above routinely produces orders costing many times
// a bot's whole balance (a ~1% slice of a ~1M-share card priced in the
// thousands dwarfs the 1M starting balance), which is why untuned bots skip
// nearly every buy with "insufficient balance"; and (2) spending only a
// fraction per trade lets a bot keep trading over many ticks instead of
// depleting itself in one shot, so prices keep moving. The fraction (rather
// than the full balance) also leaves headroom for the 1% fee and the buy's own
// curve slippage (its average fill price sits above the current spot).
const buyBudgetFraction = 0.25

// affordableBuyShares is the largest BUY, in shares, a bot with balance can
// make on a card at price without exceeding its per-trade budget. Zero when
// the price is unknown/zero or the budget can't cover a single share.
func affordableBuyShares(balance, price int64) int64 {
	if price <= 0 {
		return 0
	}
	return int64(float64(balance) * buyBudgetFraction / float64(price))
}

// capBuyToBalance shrinks a BUY decision to what the bot can actually afford at
// the card's current price, returning nil when it can't afford even the
// minimum order (so the bot skips quietly rather than firing a doomed trade
// the ledger will reject). SELLs pass through unchanged — they're already
// bounded by shares owned and only cost the fee. The card's price is looked up
// from snapshots by ID.
func capBuyToBalance(decision *Decision, snapshots []MarketSnapshot, balance int64) *Decision {
	if decision == nil || decision.Type != domain.TransactionTypeBuy {
		return decision
	}

	var price int64
	for i := range snapshots {
		if snapshots[i].CardID == decision.CardID {
			price = snapshots[i].CurrentPrice
			break
		}
	}

	affordable := affordableBuyShares(balance, price)
	if affordable < minTradeShares {
		return nil
	}
	if decision.Shares > affordable {
		decision.Shares = affordable
	}
	return decision
}

// MarketSnapshot is one card's current state as a bot sees it: live price,
// its own rolling price history (oldest first, latest == CurrentPrice), and
// its own holding of that card.
type MarketSnapshot struct {
	CardID            uuid.UUID
	Symbol            string
	CardType          domain.CardType
	Sector            *string
	CurrentPrice      int64
	CirculatingSupply int64
	History           []int64
	SharesOwned       int64
}

// Decision is what a persona wants to do this tick — nil means skip.
type Decision struct {
	CardID uuid.UUID
	Type   domain.TransactionType
	Shares int64
}

// trendPercent reports the fractional price move across the most recent
// historyWindow observations, and whether there's enough history yet to
// trust the signal at all.
func trendPercent(history []int64) (float64, bool) {
	if len(history) < historyWindow {
		return 0, false
	}
	window := history[len(history)-historyWindow:]
	oldest, latest := window[0], window[len(window)-1]
	if oldest == 0 {
		return 0, false
	}
	return float64(latest-oldest) / float64(oldest), true
}

// sizeTrade picks an order size as a random percentage of circulatingSupply
// (see the doc comment on the percent constants above), clamped to a
// sane absolute range.
func sizeTrade(rng *rand.Rand, circulatingSupply int64) int64 {
	pct := minTradePercentOfSupply + rng.Float64()*(maxTradePercentOfSupply-minTradePercentOfSupply)
	return clampShares(int64(float64(circulatingSupply) * pct))
}

func clampShares(shares int64) int64 {
	if shares < minTradeShares {
		return minTradeShares
	}
	if shares > maxTradeShares {
		return maxTradeShares
	}
	return shares
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// decideMomentum buys the strongest uptrending card and sells the strongest
// downtrending one it holds (§4.5: "amplifies moves, creates trends"). A
// buy candidate always wins over a sell candidate when both exist — this
// bot's whole point is piling onto strength, not risk-managing a portfolio.
func decideMomentum(rng *rand.Rand, snapshots []MarketSnapshot) *Decision {
	var buyCandidate, sellCandidate *MarketSnapshot
	var buyTrend, sellTrend float64

	for i := range snapshots {
		s := &snapshots[i]
		trend, ok := trendPercent(s.History)
		if !ok {
			continue
		}
		if trend > momentumThreshold && (buyCandidate == nil || trend > buyTrend) {
			buyCandidate, buyTrend = s, trend
		}
		if trend < -momentumThreshold && s.SharesOwned > 0 && (sellCandidate == nil || trend < sellTrend) {
			sellCandidate, sellTrend = s, trend
		}
	}

	switch {
	case buyCandidate != nil:
		return &Decision{CardID: buyCandidate.CardID, Type: domain.TransactionTypeBuy, Shares: sizeTrade(rng, buyCandidate.CirculatingSupply)}
	case sellCandidate != nil:
		return &Decision{CardID: sellCandidate.CardID, Type: domain.TransactionTypeSell, Shares: minInt64(sellCandidate.SharesOwned, sizeTrade(rng, sellCandidate.CirculatingSupply))}
	default:
		return nil
	}
}

// decideContrarian is decideMomentum with the signal inverted: buys dips,
// sells rallies it holds (§4.5: "dampens moves, creates support/resistance").
func decideContrarian(rng *rand.Rand, snapshots []MarketSnapshot) *Decision {
	var buyCandidate, sellCandidate *MarketSnapshot
	var buyTrend, sellTrend float64

	for i := range snapshots {
		s := &snapshots[i]
		trend, ok := trendPercent(s.History)
		if !ok {
			continue
		}
		if trend < -momentumThreshold && (buyCandidate == nil || trend < buyTrend) {
			buyCandidate, buyTrend = s, trend
		}
		if trend > momentumThreshold && s.SharesOwned > 0 && (sellCandidate == nil || trend > sellTrend) {
			sellCandidate, sellTrend = s, trend
		}
	}

	switch {
	case buyCandidate != nil:
		return &Decision{CardID: buyCandidate.CardID, Type: domain.TransactionTypeBuy, Shares: sizeTrade(rng, buyCandidate.CirculatingSupply)}
	case sellCandidate != nil:
		return &Decision{CardID: sellCandidate.CardID, Type: domain.TransactionTypeSell, Shares: minInt64(sellCandidate.SharesOwned, sizeTrade(rng, sellCandidate.CirculatingSupply))}
	default:
		return nil
	}
}

// decideRandomWalker is pure background noise (§4.5): a random active card,
// a small random order, sector-agnostic. Only sells if it already holds
// something, and even then only about half the time.
func decideRandomWalker(rng *rand.Rand, snapshots []MarketSnapshot) *Decision {
	if len(snapshots) == 0 {
		return nil
	}
	s := snapshots[rng.Intn(len(snapshots))]
	if s.SharesOwned > 0 && rng.Intn(2) == 0 {
		return &Decision{CardID: s.CardID, Type: domain.TransactionTypeSell, Shares: minInt64(s.SharesOwned, sizeTrade(rng, s.CirculatingSupply))}
	}
	return &Decision{CardID: s.CardID, Type: domain.TransactionTypeBuy, Shares: sizeTrade(rng, s.CirculatingSupply)}
}

// decideNewsReactive (news_reactive.go) is this persona's real strategy —
// it lives on *Bot rather than here since it needs DB access (to poll
// news_events) that the other pure decide* functions don't.

// decideIndexTracker keeps an INDEX card's tradable liquidity healthy by
// nudging it toward its derived price (§5.2, §4.5) — the weighted sum of its
// components' current prices, computed by computeDerivedPrices in market.go.
func decideIndexTracker(rng *rand.Rand, snapshots []MarketSnapshot, derivedPrices map[uuid.UUID]int64) *Decision {
	for _, s := range snapshots {
		if s.CardType != domain.CardTypeIndex || s.CurrentPrice == 0 {
			continue
		}
		derived, ok := derivedPrices[s.CardID]
		if !ok {
			continue
		}
		diff := float64(derived-s.CurrentPrice) / float64(s.CurrentPrice)
		if diff > indexTrackerThreshold {
			return &Decision{CardID: s.CardID, Type: domain.TransactionTypeBuy, Shares: sizeTrade(rng, s.CirculatingSupply)}
		}
		if diff < -indexTrackerThreshold && s.SharesOwned > 0 {
			return &Decision{CardID: s.CardID, Type: domain.TransactionTypeSell, Shares: minInt64(s.SharesOwned, sizeTrade(rng, s.CirculatingSupply))}
		}
	}
	return nil
}
