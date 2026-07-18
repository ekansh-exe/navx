package news

import (
	"context"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ekansh-exe/navx/internal/store/db"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect to test db: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestGenerateOnce_WritesWellFormedNewsEvent(t *testing.T) {
	pool := testPool(t)
	queries := db.New(pool)
	ctx := context.Background()

	rng := rand.New(rand.NewSource(1))
	generateOnce(ctx, queries, rng)

	rows, err := queries.ListNewsEvents(ctx, db.ListNewsEventsParams{Limit: 1, Offset: 0})
	if err != nil {
		t.Fatalf("list news events: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected at least one news event, got %d", len(rows))
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM news_events WHERE id = $1", rows[0].ID)
	})

	event := rows[0]
	if !strings.Contains(event.Headline, " in ") || !strings.Contains(event.Headline, " affects ") || !strings.Contains(event.Headline, " markets") {
		t.Fatalf("headline %q doesn't match the expected format", event.Headline)
	}
	if event.Category == nil || *event.Category == "" {
		t.Fatal("expected a non-empty category")
	}
	if event.RelatedCardID != nil {
		t.Fatalf("expected related_card_id to be nil for a sector-wide event, got %v", *event.RelatedCardID)
	}

	foundCountry := false
	for _, c := range Countries {
		if strings.Contains(event.Headline, c.Name) {
			foundCountry = true
			break
		}
	}
	if !foundCountry {
		t.Fatalf("headline %q doesn't mention any known fictional country", event.Headline)
	}
}

func TestReaderList_MostRecentFirstAndRespectsLimit(t *testing.T) {
	pool := testPool(t)
	queries := db.New(pool)
	reader := NewReader(pool)
	ctx := context.Background()

	headlines := []string{"Flood in Endia affects Food markets", "War in Eran affects Oil Gas markets", "Strike in Straya affects Shipping markets"}
	for _, h := range headlines {
		row, err := queries.CreateNewsEvent(ctx, db.CreateNewsEventParams{Headline: h})
		if err != nil {
			t.Fatalf("seed news event: %v", err)
		}
		id := row.ID
		t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM news_events WHERE id = $1", id) })
	}

	events, err := reader.List(ctx, 2, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events (limit=2), got %d", len(events))
	}
	// Most recent first: the last-inserted headline should come back first.
	if events[0].Headline != headlines[len(headlines)-1] {
		t.Fatalf("events[0].Headline = %q, want the most recently created (%q)", events[0].Headline, headlines[len(headlines)-1])
	}
}
