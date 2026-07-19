package api

import "net/http"

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
