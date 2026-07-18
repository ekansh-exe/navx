package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ekansh-exe/navx/internal/auth"
	"github.com/ekansh-exe/navx/internal/ledger"
)

func testRouter(t *testing.T) (chi.Router, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect to test db: %v", err)
	}
	t.Cleanup(pool.Close)

	secret := []byte("test-secret")
	ledgerSvc := ledger.New(pool)
	h := NewHandler(auth.NewService(pool, ledgerSvc, secret, time.Hour), ledgerSvc)

	r := chi.NewRouter()
	r.Post("/api/auth/register", h.Register)
	r.Post("/api/auth/login", h.Login)
	r.Group(func(r chi.Router) {
		r.Use(RequireAuth(secret))
		r.Get("/api/users/me", h.Me)
		r.Post("/api/trades/quote", h.Quote)
		r.Post("/api/trades/execute", h.ExecuteTrade)
		r.Post("/api/cards", h.LaunchCard)
	})
	return r, pool
}

func doJSON(t *testing.T, r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestRegisterLoginMeFlow(t *testing.T) {
	r, pool := testRouter(t)
	username := "flow_" + uuid.NewString()[:8]
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, "DELETE FROM transactions WHERE user_id IN (SELECT id FROM users WHERE username = $1)", username)
		pool.Exec(ctx, "DELETE FROM users WHERE username = $1", username)
	})

	registerRec := doJSON(t, r, http.MethodPost, "/api/auth/register", registerRequest{
		Username: username,
		Password: "correct-password",
	})
	if registerRec.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", registerRec.Code, registerRec.Body.String())
	}

	loginRec := doJSON(t, r, http.MethodPost, "/api/auth/login", loginRequest{
		Username: username,
		Password: "correct-password",
	})
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", loginRec.Code, loginRec.Body.String())
	}
	var loginResp loginResponse
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if !loginResp.RewardGranted {
		t.Fatal("expected first login to grant the daily reward")
	}
	if loginResp.RewardAmount != ledger.DailyRewardAmount {
		t.Fatalf("reward_amount = %d, want %d", loginResp.RewardAmount, ledger.DailyRewardAmount)
	}
	if loginResp.Token == "" {
		t.Fatal("expected a non-empty token")
	}

	// Second login same day: reward must not grant again.
	loginRec2 := doJSON(t, r, http.MethodPost, "/api/auth/login", loginRequest{
		Username: username,
		Password: "correct-password",
	})
	var loginResp2 loginResponse
	if err := json.Unmarshal(loginRec2.Body.Bytes(), &loginResp2); err != nil {
		t.Fatalf("decode second login response: %v", err)
	}
	if loginResp2.RewardGranted {
		t.Fatal("expected second same-day login to not grant the reward again")
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
	meRec := httptest.NewRecorder()
	r.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me status = %d, body = %s", meRec.Code, meRec.Body.String())
	}
	var meResp userDTO
	if err := json.Unmarshal(meRec.Body.Bytes(), &meResp); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	if meResp.Username != username {
		t.Fatalf("me username = %q, want %q", meResp.Username, username)
	}
	if meResp.CurrencyBalance != loginResp2.User.CurrencyBalance {
		t.Fatalf("me balance = %d, want %d", meResp.CurrencyBalance, loginResp2.User.CurrencyBalance)
	}
}

func TestMe_RequiresAuth(t *testing.T) {
	r, _ := testRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
