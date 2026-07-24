package api

import "net/http"

const (
	defaultTradesPageSize = 20
	maxTradesPageSize     = 100
)

// Holdings is a per-user read: GET /api/users/me/holdings, returning the
// authenticated user's current nonzero positions across every card.
func (h *Handler) Holdings(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing authenticated user")
		return
	}

	holdings, err := h.ledger.ListHoldingsByUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	dtos := make([]holdingDTO, len(holdings))
	for i, hd := range holdings {
		dtos[i] = toHoldingDTO(hd)
	}

	writeJSON(w, http.StatusOK, holdingsResponse{Holdings: dtos})
}

// Trades is a per-user read: GET /api/users/me/trades?limit=&offset=,
// returning the authenticated user's own BUY/SELL trade history, newest
// first, each paired with its fee and the card it was against. This is the
// persisted history behind the portfolio's "recent trades" — unlike the
// frontend's previous session-only client state, it survives a refresh or a
// fresh login.
func (h *Handler) Trades(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing authenticated user")
		return
	}

	limit := parsePageParam(r.URL.Query().Get("limit"), defaultTradesPageSize, maxTradesPageSize)
	offset := parsePageParam(r.URL.Query().Get("offset"), 0, 0)

	trades, err := h.ledger.ListRecentTradesByUser(r.Context(), userID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	dtos := make([]tradeHistoryEntryDTO, len(trades))
	for i, t := range trades {
		dtos[i] = toTradeHistoryEntryDTO(t)
	}

	writeJSON(w, http.StatusOK, tradeHistoryResponse{Trades: dtos, Limit: limit, Offset: offset})
}
