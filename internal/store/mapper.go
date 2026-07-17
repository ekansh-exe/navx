package store

import (
	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/store/db"
)

// ToDomainUser maps a store-layer row to the framework-free domain type,
// deliberately dropping PasswordHash so it can never leak past this layer.
func ToDomainUser(u db.User) *domain.User {
	return &domain.User{
		ID:               u.ID,
		Username:         u.Username,
		UserType:         domain.UserType(u.UserType),
		CurrencyBalance:  u.CurrencyBalance,
		LoginStreakCount: int(u.LoginStreakCount),
		LastLoginAt:      u.LastLoginAt,
		CreatedAt:        u.CreatedAt,
	}
}

// ToDomainCard maps a store-layer row to the framework-free domain type.
func ToDomainCard(c db.Card) *domain.Card {
	return &domain.Card{
		ID:                    c.ID,
		CreatorUserID:         c.CreatorUserID,
		Type:                  domain.CardType(c.CardType),
		Sector:                c.Sector,
		Symbol:                c.Symbol,
		Name:                  c.Name,
		SupplyModel:           domain.SupplyModel(c.SupplyModel),
		TotalSupply:           c.TotalSupply,
		CirculatingSupply:     c.CirculatingSupply,
		CreatorRetainedShares: c.CreatorRetainedShares,
		BasePrice:             c.BasePrice,
		Scale:                 c.Scale,
		CurrentPrice:          c.CurrentPrice,
		Status:                domain.CardStatus(c.Status),
		CreatedAt:             c.CreatedAt,
	}
}

// ToDomainTransaction maps a store-layer row to the framework-free domain type.
func ToDomainTransaction(t db.Transaction) *domain.Transaction {
	return &domain.Transaction{
		ID:                   t.ID,
		UserID:               t.UserID,
		CardID:               t.CardID,
		Type:                 domain.TransactionType(t.Type),
		Shares:               t.Shares,
		PricePerShare:        t.PricePerShare,
		TotalCurrencyDelta:   t.TotalCurrencyDelta,
		ResultingBalance:     t.ResultingBalance,
		IdempotencyKey:       t.IdempotencyKey,
		RelatedTransactionID: t.RelatedTransactionID,
		CreatedAt:            t.CreatedAt,
	}
}
