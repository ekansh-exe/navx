package news

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/store"
	"github.com/ekansh-exe/navx/internal/store/db"
)

// Run drives the news-generation job until ctx is cancelled: one immediate
// generation on startup (so there's something to show right away, not
// after a full interval), then one every interval thereafter. Mirrors
// internal/bots's RunRebalanceJob ticker shape.
func Run(ctx context.Context, pool *pgxpool.Pool, interval time.Duration) {
	queries := db.New(pool)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	generateOnce(ctx, queries, rng)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			generateOnce(ctx, queries, rng)
		}
	}
}

// generateOnce picks a random country + event type + affected sector(s),
// composes a headline, and writes it as a news_events row. related_card_id
// and body are left unset — these are sector-wide macro events, not tied to
// one card, and only a headline was asked for this phase.
func generateOnce(ctx context.Context, queries *db.Queries, rng *rand.Rand) {
	country := randomCountry(rng)
	event := randomEventType(rng)
	sectors := sectorsFor(event)
	headline := composeHeadline(event, country, sectors)
	category := string(event)

	created, err := queries.CreateNewsEvent(ctx, db.CreateNewsEventParams{
		Headline: headline,
		Category: &category,
	})
	if err != nil {
		slog.Error("NEWS_GENERATION_ERROR", "error", err)
		return
	}
	slog.Info("NEWS_GENERATED", "id", created.ID, "headline", headline)
}

// Reader is the read side used by the API layer, mirroring ledger.New(pool)'s
// shape (constructs its own *db.Queries internally).
type Reader struct {
	queries *db.Queries
}

func NewReader(pool *pgxpool.Pool) *Reader {
	return &Reader{queries: db.New(pool)}
}

// List returns the most recent news events first. limit/offset are the
// caller's responsibility to bound sanely (the API handler does this).
func (r *Reader) List(ctx context.Context, limit, offset int32) ([]*domain.NewsEvent, error) {
	rows, err := r.queries.ListNewsEvents(ctx, db.ListNewsEventsParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	events := make([]*domain.NewsEvent, len(rows))
	for i, row := range rows {
		events[i] = store.ToDomainNewsEvent(row)
	}
	return events, nil
}
