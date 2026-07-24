package api

import (
	"time"

	"github.com/google/uuid"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/leaderboard"
	"github.com/ekansh-exe/navx/internal/ledger"
	"github.com/ekansh-exe/navx/internal/quests"
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
	ID                        uuid.UUID  `json:"id"`
	CreatorUserID             *uuid.UUID `json:"creator_user_id"`
	Symbol                    string     `json:"symbol"`
	Name                      string     `json:"name"`
	Description               *string    `json:"description"`
	ImageURL                  *string    `json:"image_url"`
	CardType                  string     `json:"card_type"`
	SupplyModel               string     `json:"supply_model"`
	TotalSupply               *int64     `json:"total_supply"`
	CirculatingSupply         int64      `json:"circulating_supply"`
	CreatorRetainedShares     int64      `json:"creator_retained_shares"`
	CreatorRetainedSharesSold int64      `json:"creator_retained_shares_sold"`
	CurrentPrice              int64      `json:"current_price"`
	Status                    string     `json:"status"`
	CreatedAt                 time.Time  `json:"created_at"`
}

func toCardDTO(c *domain.Card) cardDTO {
	return cardDTO{
		ID:                        c.ID,
		CreatorUserID:             c.CreatorUserID,
		Symbol:                    c.Symbol,
		Name:                      c.Name,
		Description:               c.Description,
		ImageURL:                  c.ImageURL,
		CardType:                  string(c.Type),
		SupplyModel:               string(c.SupplyModel),
		TotalSupply:               c.TotalSupply,
		CirculatingSupply:         c.CirculatingSupply,
		CreatorRetainedShares:     c.CreatorRetainedShares,
		CreatorRetainedSharesSold: c.CreatorRetainedSharesSold,
		CurrentPrice:              c.CurrentPrice,
		Status:                    string(c.Status),
		CreatedAt:                 c.CreatedAt,
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

type cardListResponse struct {
	Cards  []cardDTO `json:"cards"`
	Limit  int32     `json:"limit"`
	Offset int32     `json:"offset"`
}

type launchCardRequest struct {
	Symbol          string  `json:"symbol"`
	Name            string  `json:"name"`
	Description     *string `json:"description"`
	ImageURL        *string `json:"image_url"`
	TotalSupply     int64   `json:"total_supply"`
	RetainedPercent float64 `json:"retained_percent"`
	IdempotencyKey  string  `json:"idempotency_key"`
}

type launchCardResponse struct {
	Card        cardDTO        `json:"card"`
	Transaction transactionDTO `json:"transaction"`
	User        userDTO        `json:"user"`
}

type newsEventDTO struct {
	ID            uuid.UUID  `json:"id"`
	Headline      string     `json:"headline"`
	Body          *string    `json:"body"`
	Category      *string    `json:"category"`
	RelatedCardID *uuid.UUID `json:"related_card_id"`
	CreatedAt     time.Time  `json:"created_at"`
}

func toNewsEventDTO(n *domain.NewsEvent) newsEventDTO {
	return newsEventDTO{
		ID:            n.ID,
		Headline:      n.Headline,
		Body:          n.Body,
		Category:      n.Category,
		RelatedCardID: n.RelatedCardID,
		CreatedAt:     n.CreatedAt,
	}
}

type newsListResponse struct {
	News   []newsEventDTO `json:"news"`
	Limit  int32          `json:"limit"`
	Offset int32          `json:"offset"`
}

type leaderboardEntryDTO struct {
	Rank                  int       `json:"rank"`
	UserID                uuid.UUID `json:"user_id"`
	Username              string    `json:"username"`
	NetWorth              int64     `json:"net_worth"`
	ChangeFromLastRefresh *int64    `json:"change_from_last_refresh,omitempty"`
	IsGoat                bool      `json:"is_goat,omitempty"`
	NetWorthDisplay       string    `json:"net_worth_display,omitempty"`
}

func toLeaderboardEntryDTO(e leaderboard.Entry) leaderboardEntryDTO {
	return leaderboardEntryDTO{
		Rank:                  e.Rank,
		UserID:                e.UserID,
		Username:              e.Username,
		NetWorth:              e.NetWorth,
		ChangeFromLastRefresh: e.ChangeFromLastRefresh,
		IsGoat:                e.IsGoat,
		NetWorthDisplay:       e.NetWorthDisplay,
	}
}

type leaderboardResponse struct {
	Leaderboard []leaderboardEntryDTO `json:"leaderboard"`
}

type holdingDTO struct {
	CardID        uuid.UUID  `json:"card_id"`
	SharesOwned   int64      `json:"shares_owned"`
	AvgCostBasis  int64      `json:"avg_cost_basis"`
	FirstBoughtAt *time.Time `json:"first_bought_at"`
}

func toHoldingDTO(h *domain.Holding) holdingDTO {
	return holdingDTO{
		CardID:        h.CardID,
		SharesOwned:   h.SharesOwned,
		AvgCostBasis:  h.AvgCostBasis,
		FirstBoughtAt: h.FirstBoughtAt,
	}
}

type holdingsResponse struct {
	Holdings []holdingDTO `json:"holdings"`
}

type tradeHistoryEntryDTO struct {
	Transaction    transactionDTO `json:"transaction"`
	FeeTransaction transactionDTO `json:"fee_transaction"`
	Card           cardDTO        `json:"card"`
}

func toTradeHistoryEntryDTO(e *ledger.TradeHistoryEntry) tradeHistoryEntryDTO {
	return tradeHistoryEntryDTO{
		Transaction:    toTransactionDTO(e.Transaction),
		FeeTransaction: toTransactionDTO(e.FeeTransaction),
		Card:           toCardDTO(e.Card),
	}
}

type tradeHistoryResponse struct {
	Trades []tradeHistoryEntryDTO `json:"trades"`
	Limit  int32                  `json:"limit"`
	Offset int32                  `json:"offset"`
}

type questDTO struct {
	ID             uuid.UUID `json:"id"`
	Title          string    `json:"title"`
	Progress       int32     `json:"progress"`
	TargetValue    int32     `json:"target_value"`
	RewardCurrency int32     `json:"reward_currency"`
	Completed      bool      `json:"completed"`
	ResetAt        time.Time `json:"reset_at"`
}

func toQuestDTO(v quests.View) questDTO {
	return questDTO{
		ID:             v.ID,
		Title:          v.Title,
		Progress:       v.Progress,
		TargetValue:    v.TargetValue,
		RewardCurrency: v.RewardCurrency,
		Completed:      v.Completed,
		ResetAt:        v.ResetAt,
	}
}

type questsResponse struct {
	Quests []questDTO `json:"quests"`
}
