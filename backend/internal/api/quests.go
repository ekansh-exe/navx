package api

import "net/http"

// Quests is a per-user read (§7): GET /api/quests, returning every quest
// with the authenticated user's current progress against it.
func (h *Handler) Quests(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing authenticated user")
		return
	}

	views, err := h.quests.ListUserQuests(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	dtos := make([]questDTO, len(views))
	for i, v := range views {
		dtos[i] = toQuestDTO(v)
	}

	writeJSON(w, http.StatusOK, questsResponse{Quests: dtos})
}
