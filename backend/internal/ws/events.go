package ws

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/leaderboard"
)

// Topic names — the path suffixes the frontend connects to (/ws/<topic>) and
// the keys the hub fans out on.
const (
	TopicPrices      = "prices"
	TopicNews        = "news"
	TopicLeaderboard = "leaderboard"
)

// envelope is the wire shape every pushed message shares — a discriminated
// union the frontend switches on by `type` (see frontend/src/types/ws.ts).
type envelope struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// priceTickData mirrors PriceTickMessage.data in frontend/src/types/ws.ts —
// field names, casing, and integer-currency units match the REST cardDTO.
type priceTickData struct {
	CardID        uuid.UUID `json:"card_id"`
	Symbol        string    `json:"symbol"`
	Price         int64     `json:"price"`
	PreviousPrice int64     `json:"previous_price"`
	Volume        int64     `json:"volume"`
	TS            time.Time `json:"ts"`
}

// newsData mirrors the REST newsEventDTO (which NewsPublishedMessage.data is
// typed as on the frontend), kept in sync field-for-field.
type newsData struct {
	ID            uuid.UUID  `json:"id"`
	Headline      string     `json:"headline"`
	Body          *string    `json:"body"`
	Category      *string    `json:"category"`
	RelatedCardID *uuid.UUID `json:"related_card_id"`
	CreatedAt     time.Time  `json:"created_at"`
}

// PublishPriceTick broadcasts a single price change to /ws/prices. Called on
// every executed trade — human or bot — so previousPrice is the card's price
// just before this trade and price is the recomputed spot price after it.
func (h *Hub) PublishPriceTick(cardID uuid.UUID, symbol string, price, previousPrice, volume int64, ts time.Time) {
	payload, err := json.Marshal(envelope{
		Type: "price_tick",
		Data: priceTickData{
			CardID:        cardID,
			Symbol:        symbol,
			Price:         price,
			PreviousPrice: previousPrice,
			Volume:        volume,
			TS:            ts,
		},
	})
	if err != nil {
		slog.Error("WS_MARSHAL_ERROR", "topic", TopicPrices, "error", err)
		return
	}
	h.Publish(TopicPrices, payload)
}

// leaderboardData mirrors LeaderboardUpdateMessage.data in
// frontend/src/types/ws.ts. leaderboard.Entry's own JSON tags already match
// the REST leaderboardEntryDTO, so entries serialize identically here.
type leaderboardData struct {
	Leaderboard []leaderboard.Entry `json:"leaderboard"`
}

// PublishLeaderboard broadcasts a refreshed leaderboard to /ws/leaderboard.
func (h *Hub) PublishLeaderboard(entries []leaderboard.Entry) {
	payload, err := json.Marshal(envelope{
		Type: "leaderboard_update",
		Data: leaderboardData{Leaderboard: entries},
	})
	if err != nil {
		slog.Error("WS_MARSHAL_ERROR", "topic", TopicLeaderboard, "error", err)
		return
	}
	h.Publish(TopicLeaderboard, payload)
}

// PublishNews broadcasts a newly published news event to /ws/news.
func (h *Hub) PublishNews(e *domain.NewsEvent) {
	payload, err := json.Marshal(envelope{
		Type: "news_published",
		Data: newsData{
			ID:            e.ID,
			Headline:      e.Headline,
			Body:          e.Body,
			Category:      e.Category,
			RelatedCardID: e.RelatedCardID,
			CreatedAt:     e.CreatedAt,
		},
	})
	if err != nil {
		slog.Error("WS_MARSHAL_ERROR", "topic", TopicNews, "error", err)
		return
	}
	h.Publish(TopicNews, payload)
}
