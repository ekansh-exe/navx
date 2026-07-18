package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ekansh-exe/navx/internal/ledger"
)

// LaunchCard requires an idempotency_key (§10, same contract as trade execution).
func (h *Handler) LaunchCard(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing authenticated user")
		return
	}

	var req launchCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.IdempotencyKey == "" {
		writeError(w, http.StatusBadRequest, "idempotency_key is required")
		return
	}

	result, err := h.ledger.LaunchCard(r.Context(), ledger.LaunchCardParams{
		CreatorUserID:   userID,
		Symbol:          req.Symbol,
		Name:            req.Name,
		Description:     req.Description,
		ImageURL:        req.ImageURL,
		TotalSupply:     req.TotalSupply,
		RetainedPercent: req.RetainedPercent,
		IdempotencyKey:  req.IdempotencyKey,
	})
	if err != nil {
		writeLaunchError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, launchCardResponse{
		Card:        toCardDTO(result.Card),
		Transaction: toTransactionDTO(result.Transaction),
		User:        toUserDTO(result.User),
	})
}

func writeLaunchError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ledger.ErrInvalidTotalSupply), errors.Is(err, ledger.ErrInvalidRetainedPercent):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ledger.ErrSymbolTaken):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ledger.ErrBelowLaunchThreshold), errors.Is(err, ledger.ErrInsufficientBalance):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
