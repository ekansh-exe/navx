package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/ekansh-exe/navx/internal/auth"
	"github.com/ekansh-exe/navx/internal/ledger"
	"github.com/ekansh-exe/navx/internal/news"
	"github.com/ekansh-exe/navx/internal/quests"
)

// Handler holds thin HTTP handlers (§11.1) — all business logic lives in
// /internal/auth, /internal/ledger, /internal/news, /internal/leaderboard,
// and /internal/quests; handlers only decode requests, call them, and
// encode responses.
type Handler struct {
	auth        *auth.Service
	ledger      *ledger.Ledger
	newsReader  *news.Reader
	redisClient *redis.Client
	quests      *quests.Service
}

func NewHandler(authSvc *auth.Service, ledgerSvc *ledger.Ledger, newsReader *news.Reader, redisClient *redis.Client, questsSvc *quests.Service) *Handler {
	return &Handler{auth: authSvc, ledger: ledgerSvc, newsReader: newsReader, redisClient: redisClient, quests: questsSvc}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.auth.Register(r.Context(), req.Username, req.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toUserDTO(user))
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, token, granted, err := h.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	resp := loginResponse{
		Token:         token,
		User:          toUserDTO(user),
		RewardGranted: granted,
	}
	if granted {
		resp.RewardAmount = ledger.DailyRewardAmount
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing authenticated user")
		return
	}

	user, err := h.auth.GetUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, toUserDTO(user))
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrUsernameTaken):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, auth.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, auth.ErrInvalidUsername), errors.Is(err, auth.ErrInvalidPassword):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
