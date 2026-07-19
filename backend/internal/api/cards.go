package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ekansh-exe/navx/internal/ledger"
)

const (
	defaultCardsPageSize = 100
	maxCardsPageSize     = 200
)

// ListCards is a public read (§10): GET /api/cards returns every ACTIVE card —
// the ~30 system companies plus the NAV5 index — ordered by symbol. limit/
// offset are supported for parity with the other list endpoints; the defaults
// return the whole board in one page.
func (h *Handler) ListCards(w http.ResponseWriter, r *http.Request) {
	limit := parsePageParam(r.URL.Query().Get("limit"), defaultCardsPageSize, maxCardsPageSize)
	offset := parsePageParam(r.URL.Query().Get("offset"), 0, 0)

	cards, err := h.ledger.ListActiveCards(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	page := paginate(cards, limit, offset)
	dtos := make([]cardDTO, len(page))
	for i, c := range page {
		dtos[i] = toCardDTO(c)
	}

	writeJSON(w, http.StatusOK, cardListResponse{Cards: dtos, Limit: limit, Offset: offset})
}

// GetCard is a public read (§10): GET /api/cards/{id} returns one card, or 404
// if no card has that ID.
func (h *Handler) GetCard(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	card, err := h.ledger.GetCard(r.Context(), id)
	if errors.Is(err, ledger.ErrCardNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, toCardDTO(card))
}

// paginate applies the same non-negative limit/offset semantics as the other
// list endpoints to an already-fetched slice: offset past the end yields an
// empty page, and the page is capped at limit.
func paginate[T any](items []T, limit, offset int32) []T {
	if int(offset) >= len(items) {
		return nil
	}
	end := int(offset) + int(limit)
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

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
