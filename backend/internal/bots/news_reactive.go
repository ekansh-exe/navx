package bots

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ekansh-exe/navx/internal/bots/newsreactive"
	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/news"
	"github.com/ekansh-exe/navx/internal/store/db"
)

// newsPollLimit is how many of the most recent news_events rows a
// news-reactive bot inspects each time it looks for something new — a small
// window is enough since headlines are generated far less often than a
// bot's own 2-5s tick.
const newsPollLimit = 10

// pollNewNews returns news events created since this bot's cursor, oldest
// first, and advances the cursor. On the very first call it seeds the
// cursor to the latest existing event's CreatedAt without returning
// anything, so a freshly-started bot reacts only to headlines from here on,
// not the entire pre-existing backlog.
func (b *Bot) pollNewNews(ctx context.Context) ([]db.NewsEvent, error) {
	rows, err := b.queries.ListNewsEvents(ctx, db.ListNewsEventsParams{Limit: newsPollLimit, Offset: 0})
	if err != nil {
		return nil, fmt.Errorf("list news events: %w", err)
	}

	// Seed on the very first poll regardless of whether any rows exist yet
	// — if the table's empty right now, leaving the cursor at its zero
	// value means the next real event (whenever it lands) is correctly
	// seen as fresh, rather than requiring a row to already exist to mark
	// the cursor seeded at all.
	if !b.newsCursorSeen {
		b.newsCursorSeen = true
		if len(rows) > 0 {
			b.newsCursorLastSeen = rows[0].CreatedAt
		}
		return nil, nil
	}

	if len(rows) == 0 {
		return nil, nil
	}

	// rows is newest-first; walk backwards to return oldest-first and find
	// everything after the cursor.
	var fresh []db.NewsEvent
	for i := len(rows) - 1; i >= 0; i-- {
		if rows[i].CreatedAt.After(b.newsCursorLastSeen) {
			fresh = append(fresh, rows[i])
		}
	}
	if len(fresh) > 0 {
		b.newsCursorLastSeen = fresh[len(fresh)-1].CreatedAt
	}
	return fresh, nil
}

// decideNewsReactive is §4.5's news-reactive trader: it subscribes to the
// same news events the newspaper does (by polling news_events, since there
// is no pubsub), parses each new headline's event type and affected
// sector(s), and nudges small, predictable trades toward (buy, positive
// events) or away from (sell, negative events) the SYSTEM_COMPANY cards in
// those sectors. One tick executes at most one trade, so trades queue up
// across sectors/cards and drain one per tick until empty.
func (b *Bot) decideNewsReactive(ctx context.Context, snapshots []MarketSnapshot) *Decision {
	if len(b.newsQueue) == 0 {
		events, err := b.pollNewNews(ctx)
		if err != nil {
			slog.Error("BOT_NEWS_POLL_ERROR", "bot", b.Username, "error", err)
			return nil
		}
		for _, e := range events {
			b.enqueueNewsTrades(snapshots, e)
		}
	}
	if len(b.newsQueue) == 0 {
		return nil
	}
	next := b.newsQueue[0]
	b.newsQueue = b.newsQueue[1:]
	return &next
}

// enqueueNewsTrades parses one news event's headline and, for every
// affected sector it recognizes, queues a small trade on every active
// SYSTEM_COMPANY card in that sector. Unrecognized headlines (parse
// failure) and events this persona takes no stance on (STRIKE) are logged
// and skipped, never treated as an error.
func (b *Bot) enqueueNewsTrades(snapshots []MarketSnapshot, e db.NewsEvent) {
	event, sectors, err := newsreactive.ParseHeadline(e.Headline)
	if err != nil {
		slog.Warn("BOT_NEWS_PARSE_ERROR", "bot", b.Username, "headline", e.Headline, "error", err)
		return
	}
	txType, ok := newsreactive.Direction(event)
	if !ok {
		slog.Info("BOT_NEWS_IGNORED", "bot", b.Username, "headline", e.Headline, "event", event)
		return
	}

	queued := 0
	for _, s := range snapshots {
		if s.CardType != domain.CardTypeSystemCompany || s.Sector == nil {
			continue
		}
		if !sectorAffected(*s.Sector, sectors) {
			continue
		}

		shares := sizeTrade(b.rng, s.CirculatingSupply)
		if txType == domain.TransactionTypeSell {
			shares = minInt64(s.SharesOwned, shares)
			if shares <= 0 {
				continue
			}
		}
		b.newsQueue = append(b.newsQueue, Decision{CardID: s.CardID, Type: txType, Shares: shares})
		queued++
	}
	slog.Info("BOT_NEWS_REACTION_QUEUED",
		"bot", b.Username, "headline", e.Headline, "event", event, "type", txType, "cards_queued", queued)
}

func sectorAffected(cardSector string, affected []news.Sector) bool {
	for _, s := range affected {
		if strings.EqualFold(cardSector, string(s)) {
			return true
		}
	}
	return false
}
