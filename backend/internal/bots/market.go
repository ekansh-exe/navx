package bots

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/store/db"
)

// appendHistory records price at cardID in history, keeping only the most
// recent historyWindow observations (unbounded growth isn't acceptable for a
// bot goroutine meant to run for the life of the process).
func appendHistory(history map[uuid.UUID][]int64, cardID uuid.UUID, price int64) []int64 {
	h := append(history[cardID], price)
	if len(h) > historyWindow {
		h = h[len(h)-historyWindow:]
	}
	history[cardID] = h
	return h
}

// fetchSnapshots is a bot's view of the market this tick: every active
// card's live price (folded into the bot's own rolling history) plus the
// bot's own holding of each. Read-only — no locks, no mutation; all trading
// still goes through Ledger.ExecuteTrade.
func fetchSnapshots(ctx context.Context, q *db.Queries, botUserID uuid.UUID, history map[uuid.UUID][]int64) ([]MarketSnapshot, error) {
	cards, err := q.ListActiveCards(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active cards: %w", err)
	}
	holdings, err := q.ListHoldingsByUser(ctx, botUserID)
	if err != nil {
		return nil, fmt.Errorf("list holdings: %w", err)
	}
	owned := make(map[uuid.UUID]int64, len(holdings))
	for _, h := range holdings {
		owned[h.CardID] = h.SharesOwned
	}

	snapshots := make([]MarketSnapshot, 0, len(cards))
	for _, c := range cards {
		hist := appendHistory(history, c.ID, c.CurrentPrice)
		histCopy := make([]int64, len(hist))
		copy(histCopy, hist)
		snapshots = append(snapshots, MarketSnapshot{
			CardID:            c.ID,
			Symbol:            c.Symbol,
			CardType:          domain.CardType(c.CardType),
			Sector:            c.Sector,
			CurrentPrice:      c.CurrentPrice,
			CirculatingSupply: c.CirculatingSupply,
			History:           histCopy,
			SharesOwned:       owned[c.ID],
		})
	}
	return snapshots, nil
}

// computeDerivedPrices is §5.2's index derived price: the weighted sum of an
// INDEX card's components' current prices, keyed by index card ID. Skips
// (rather than errors on) an index with no components or an unparseable
// weight, since a market-scan tick shouldn't fail outright over one bad row.
func computeDerivedPrices(ctx context.Context, q *db.Queries, indexCardIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	derived := make(map[uuid.UUID]int64, len(indexCardIDs))
	for _, id := range indexCardIDs {
		rows, err := q.ListIndexComponentPrices(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("list index component prices for %s: %w", id, err)
		}
		if len(rows) == 0 {
			continue
		}
		var weighted float64
		for _, r := range rows {
			w, err := r.Weight.Float64Value()
			if err != nil || !w.Valid {
				continue
			}
			weighted += w.Float64 * float64(r.CurrentPrice)
		}
		derived[id] = int64(math.Round(weighted))
	}
	return derived, nil
}
