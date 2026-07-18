package api

import (
	"net/http"

	"github.com/ekansh-exe/navx/internal/leaderboard"
)

// Leaderboard is a public read (§8): GET /api/leaderboard. Always serves
// whatever's cached in Redis — computed on a scheduled job, never live —
// returning an empty list rather than an error if nothing's cached yet
// (e.g. the first moments after boot).
func (h *Handler) Leaderboard(w http.ResponseWriter, r *http.Request) {
	entries, err := leaderboard.ReadCached(r.Context(), h.redisClient)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	dtos := make([]leaderboardEntryDTO, len(entries))
	for i, e := range entries {
		dtos[i] = toLeaderboardEntryDTO(e)
	}

	writeJSON(w, http.StatusOK, leaderboardResponse{Leaderboard: dtos})
}
