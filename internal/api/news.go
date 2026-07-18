package api

import (
	"net/http"
	"strconv"
)

const (
	defaultNewsPageSize = 20
	maxNewsPageSize     = 100
)

// ListNews is a public, non-binding read (§9/§10): GET /api/news?limit=&offset=,
// most recent first.
func (h *Handler) ListNews(w http.ResponseWriter, r *http.Request) {
	limit := parsePageParam(r.URL.Query().Get("limit"), defaultNewsPageSize, maxNewsPageSize)
	offset := parsePageParam(r.URL.Query().Get("offset"), 0, 0)

	events, err := h.newsReader.List(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	dtos := make([]newsEventDTO, len(events))
	for i, e := range events {
		dtos[i] = toNewsEventDTO(e)
	}

	writeJSON(w, http.StatusOK, newsListResponse{News: dtos, Limit: limit, Offset: offset})
}

// parsePageParam parses a query param as a non-negative int32, falling back
// to def on anything invalid or absent. If max > 0, the result is also
// capped at max (used for limit; offset has no upper bound).
func parsePageParam(raw string, def, max int32) int32 {
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return def
	}
	if max > 0 && int32(v) > max {
		return max
	}
	return int32(v)
}
