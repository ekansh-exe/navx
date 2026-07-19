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

// ListHoldingsByUser returns the user's current nonzero positions — the read
// behind GET /api/users/me/holdings. A holdings row survives at 0 shares
// after a full sell (UpsertHolding never deletes it, since avg_cost_basis and
// first_bought_at still matter for quests/history), but a closed-out
// position isn't a "holding" from the portfolio page's perspective, so it's
// filtered out here rather than pushed onto every caller. A plain unlocked
// read; no currency effect.
func (l *Ledger) ListHoldingsByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Holding, error) {
	rows, err := l.queries.ListHoldingsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list holdings by user: %w", err)
	}
	holdings := make([]*domain.Holding, 0, len(rows))
	for _, row := range rows {
		if row.SharesOwned == 0 {
			continue
		}
		holdings = append(holdings, store.ToDomainHolding(row))
	}
	return holdings, nil
}
