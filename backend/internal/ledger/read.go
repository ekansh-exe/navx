package ledger

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/store"
)

// ListActiveCards returns every ACTIVE card (the ~30 system companies plus the
// NAV5 index), ordered by symbol — the read behind GET /api/cards. A plain
// unlocked read; no currency effect.
func (l *Ledger) ListActiveCards(ctx context.Context) ([]*domain.Card, error) {
	rows, err := l.queries.ListActiveCards(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active cards: %w", err)
	}
	cards := make([]*domain.Card, len(rows))
	for i, row := range rows {
		cards[i] = store.ToDomainCard(row)
	}
	return cards, nil
}

// GetCard returns a single card by ID — the read behind GET /api/cards/{id}.
// Returns ErrCardNotFound (the same sentinel the trade path uses) when no such
// card exists, so the handler can map it to a 404.
func (l *Ledger) GetCard(ctx context.Context, id uuid.UUID) (*domain.Card, error) {
	card, err := l.queries.GetCardByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCardNotFound
	} else if err != nil {
		return nil, fmt.Errorf("get card: %w", err)
	}
	return store.ToDomainCard(card), nil
}
