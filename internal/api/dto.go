package api

import (
	"time"

	"github.com/google/uuid"

	"github.com/ekansh-exe/navx/internal/domain"
)

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userDTO struct {
	ID               uuid.UUID  `json:"id"`
	Username         string     `json:"username"`
	UserType         string     `json:"user_type"`
	CurrencyBalance  int64      `json:"currency_balance"`
	LoginStreakCount int        `json:"login_streak_count"`
	LastLoginAt      *time.Time `json:"last_login_at"`
	CreatedAt        time.Time  `json:"created_at"`
}

func toUserDTO(u *domain.User) userDTO {
	return userDTO{
		ID:               u.ID,
		Username:         u.Username,
		UserType:         string(u.UserType),
		CurrencyBalance:  u.CurrencyBalance,
		LoginStreakCount: u.LoginStreakCount,
		LastLoginAt:      u.LastLoginAt,
		CreatedAt:        u.CreatedAt,
	}
}

type loginResponse struct {
	Token         string  `json:"token"`
	User          userDTO `json:"user"`
	RewardGranted bool    `json:"reward_granted"`
	RewardAmount  int64   `json:"reward_amount"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type cardDTO struct {
	ID                uuid.UUID `json:"id"`
	Symbol            string    `json:"symbol"`
	Name              string    `json:"name"`
	CardType          string    `json:"card_type"`
	SupplyModel       string    `json:"supply_model"`
	TotalSupply       *int64    `json:"total_supply"`
	CirculatingSupply int64     `json:"circulating_supply"`
	CurrentPrice      int64     `json:"current_price"`
	Status            string    `json:"status"`
}

func toCardDTO(c *domain.Card) cardDTO {
	return cardDTO{
		ID:                c.ID,
		Symbol:            c.Symbol,
		Name:              c.Name,
		CardType:          string(c.Type),
		SupplyModel:       string(c.SupplyModel),
		TotalSupply:       c.TotalSupply,
		CirculatingSupply: c.CirculatingSupply,
		CurrentPrice:      c.CurrentPrice,
		Status:            string(c.Status),
	}
}

type transactionDTO struct {
	ID                 uuid.UUID  `json:"id"`
	Type               string     `json:"type"`
	CardID             *uuid.UUID `json:"card_id"`
	Shares             *int64     `json:"shares"`
	PricePerShare      *int64     `json:"price_per_share"`
	TotalCurrencyDelta int64      `json:"total_currency_delta"`
	ResultingBalance   int64      `json:"resulting_balance"`
	CreatedAt          time.Time  `json:"created_at"`
}

func toTransactionDTO(t *domain.Transaction) transactionDTO {
	return transactionDTO{
		ID:                 t.ID,
		Type:               string(t.Type),
		CardID:             t.CardID,
		Shares:             t.Shares,
		PricePerShare:      t.PricePerShare,
		TotalCurrencyDelta: t.TotalCurrencyDelta,
		ResultingBalance:   t.ResultingBalance,
		CreatedAt:          t.CreatedAt,
	}
}

type quoteRequest struct {
	CardID uuid.UUID `json:"card_id"`
	Type   string    `json:"type"`
	Shares int64     `json:"shares"`
}

type quoteResponse struct {
	Card                   cardDTO `json:"card"`
	Type                   string  `json:"type"`
	Shares                 int64   `json:"shares"`
	EstimatedCost          int64   `json:"estimated_cost"` // positive=buyer would pay, negative=seller would receive
	EstimatedFee           int64   `json:"estimated_fee"`
	EstimatedPricePerShare int64   `json:"estimated_price_per_share"`
}

type tradeExecuteRequest struct {
	CardID         uuid.UUID `json:"card_id"`
	Type           string    `json:"type"`
	Shares         int64     `json:"shares"`
	IdempotencyKey string    `json:"idempotency_key"`
}

type tradeExecuteResponse struct {
	Transaction    transactionDTO `json:"transaction"`
	FeeTransaction transactionDTO `json:"fee_transaction"`
	User           userDTO        `json:"user"`
	Card           cardDTO        `json:"card"`
}
