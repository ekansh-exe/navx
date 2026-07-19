package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestListNews_MostRecentFirstAndRespectsLimit(t *testing.T) {
	r, pool := testRouter(t)
	ctx := context.Background()

	headlines := []string{
		"Flood in Endia affects Food markets",
		"War in Eran affects Oil Gas markets",
		"Strike in Straya affects Shipping markets",
	}
	var ids []uuid.UUID
	for _, h := range headlines {
		var id uuid.UUID
		if err := pool.QueryRow(ctx, "INSERT INTO news_events (headline) VALUES ($1) RETURNING id", h).Scan(&id); err != nil {
			t.Fatalf("seed news event: %v", err)
		}
		ids = append(ids, id)
	}
	t.Cleanup(func() {
		for _, id := range ids {
			pool.Exec(context.Background(), "DELETE FROM news_events WHERE id = $1", id)
		}
	})

	rec := doJSON(t, r, http.MethodGet, "/api/news?limit=2&offset=0", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp newsListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Limit != 2 || resp.Offset != 0 {
		t.Fatalf("limit/offset = %d/%d, want 2/0", resp.Limit, resp.Offset)
	}
	if len(resp.News) != 2 {
		t.Fatalf("expected 2 news items, got %d", len(resp.News))
	}
	if resp.News[0].Headline != headlines[len(headlines)-1] {
		t.Fatalf("News[0].Headline = %q, want the most recently created (%q)", resp.News[0].Headline, headlines[len(headlines)-1])
	}
}

func TestListNews_DefaultsAndCapsLimit(t *testing.T) {
	r, _ := testRouter(t)

	rec := doJSON(t, r, http.MethodGet, "/api/news", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp newsListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Limit != defaultNewsPageSize {
		t.Fatalf("default limit = %d, want %d", resp.Limit, defaultNewsPageSize)
	}

	recCapped := doJSON(t, r, http.MethodGet, "/api/news?limit=99999", nil)
	var respCapped newsListResponse
	if err := json.Unmarshal(recCapped.Body.Bytes(), &respCapped); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if respCapped.Limit != maxNewsPageSize {
		t.Fatalf("capped limit = %d, want %d", respCapped.Limit, maxNewsPageSize)
	}
}
