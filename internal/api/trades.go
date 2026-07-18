package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/ledger"
)

// Quote is a non-binding preview (§10) — no mutation, no idempotency key.
func (h *Handler) Quote(w http.ResponseWriter, r *http.Request) {
	var req quoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.ledger.Quote(r.Context(), ledger.QuoteParams{
		CardID: req.CardID,
		Type:   domain.TransactionType(req.Type),
		Shares: req.Shares,
	})
	if err != nil {
		writeTradeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, quoteResponse{
		Card:                   toCardDTO(result.Card),
		Type:                   req.Type,
		Shares:                 req.Shares,
		EstimatedCost:          result.Cost,
		EstimatedFee:           result.Fee,
		EstimatedPricePerShare: result.PricePerShare,
	})
}

// ExecuteTrade requires an idempotency_key (§10) — a real buy/sell.
func (h *Handler) ExecuteTrade(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing authenticated user")
		return
	}

	var req tradeExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.IdempotencyKey == "" {
		writeError(w, http.StatusBadRequest, "idempotency_key is required")
		return
	}

	result, err := h.ledger.ExecuteTrade(r.Context(), ledger.TradeParams{
		UserID:         userID,
		CardID:         req.CardID,
		Type:           domain.TransactionType(req.Type),
		Shares:         req.Shares,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		writeTradeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tradeExecuteResponse{
		Transaction:    toTransactionDTO(result.Transaction),
		FeeTransaction: toTransactionDTO(result.FeeTransaction),
		User:           toUserDTO(result.User),
		Card:           toCardDTO(result.Card),
	})
}

func writeTradeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ledger.ErrInvalidShares), errors.Is(err, ledger.ErrInvalidTradeType):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ledger.ErrCardNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ledger.ErrCardNotTradable),
		errors.Is(err, ledger.ErrInsufficientSupply),
		errors.Is(err, ledger.ErrInsufficientShares),
		errors.Is(err, ledger.ErrInsufficientBalance),
		errors.Is(err, ledger.ErrIdempotencyKeyMismatch),
		errors.Is(err, ledger.ErrRetainedSharesLocked),
		errors.Is(err, ledger.ErrPositionCapExceeded),
		errors.Is(err, ledger.ErrCircuitBreakerActive):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
