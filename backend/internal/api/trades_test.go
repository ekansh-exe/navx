package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func createTestTradeCard(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO cards (card_type, symbol, name, supply_model, total_supply, circulating_supply, base_price, scale, current_price, status)
		VALUES ('SYSTEM_COMPANY', $1, $1, 'FIXED', 1000000, 1000, 10, 1000000, 1, 'ACTIVE')
		RETURNING id`,
		"APITST_"+uuid.NewString()[:8],
	).Scan(&id)
	if err != nil {
		t.Fatalf("create test card: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		pool.Exec(cleanupCtx, "DELETE FROM holdings WHERE card_id = $1", id)
		pool.Exec(cleanupCtx, "DELETE FROM transactions WHERE card_id = $1", id)
		pool.Exec(cleanupCtx, "DELETE FROM cards WHERE id = $1", id)
	})
	return id
}

// doAuthJSON is doJSON plus a Bearer token — the trade endpoints are all
// behind RequireAuth, unlike register/login.
func doAuthJSON(t *testing.T, r http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestQuoteAndExecuteTradeFlow(t *testing.T) {
	r, pool := testRouter(t)
	cardID := createTestTradeCard(t, pool)

	username := "trade_" + uuid.NewString()[:8]
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, "DELETE FROM transactions WHERE user_id IN (SELECT id FROM users WHERE username = $1)", username)
		pool.Exec(ctx, "DELETE FROM holdings WHERE user_id IN (SELECT id FROM users WHERE username = $1)", username)
		pool.Exec(ctx, "DELETE FROM users WHERE username = $1", username)
	})

	doJSON(t, r, http.MethodPost, "/api/auth/register", registerRequest{Username: username, Password: "correct-password"})
	loginRec := doJSON(t, r, http.MethodPost, "/api/auth/login", loginRequest{Username: username, Password: "correct-password"})
	var loginResp loginResponse
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	// Quote: non-binding, no mutation.
	quoteRec := doAuthJSON(t, r, http.MethodPost, "/api/trades/quote", loginResp.Token, quoteRequest{CardID: cardID, Type: "BUY", Shares: 50})
	if quoteRec.Code != http.StatusOK {
		t.Fatalf("quote status = %d, body = %s", quoteRec.Code, quoteRec.Body.String())
	}
	var quoteResp quoteResponse
	if err := json.Unmarshal(quoteRec.Body.Bytes(), &quoteResp); err != nil {
		t.Fatalf("decode quote response: %v", err)
	}
	if quoteResp.EstimatedCost <= 0 {
		t.Fatalf("estimated cost = %d, want > 0 for a buy", quoteResp.EstimatedCost)
	}

	var supplyAfterQuote int64
	if err := pool.QueryRow(context.Background(), "SELECT circulating_supply FROM cards WHERE id = $1", cardID).Scan(&supplyAfterQuote); err != nil {
		t.Fatalf("read supply after quote: %v", err)
	}
	if supplyAfterQuote != 1000 {
		t.Fatalf("supply after quote = %d, want unchanged 1000 (quote must not mutate)", supplyAfterQuote)
	}

	// Execute: a real, idempotency-keyed buy.
	execRec := doAuthJSON(t, r, http.MethodPost, "/api/trades/execute", loginResp.Token, tradeExecuteRequest{
		CardID: cardID, Type: "BUY", Shares: 50, IdempotencyKey: uuid.NewString(),
	})
	if execRec.Code != http.StatusOK {
		t.Fatalf("execute status = %d, body = %s", execRec.Code, execRec.Body.String())
	}
	var execResp tradeExecuteResponse
	if err := json.Unmarshal(execRec.Body.Bytes(), &execResp); err != nil {
		t.Fatalf("decode execute response: %v", err)
	}

	if execResp.Card.CirculatingSupply != 1050 {
		t.Fatalf("circulating supply after execute = %d, want 1050", execResp.Card.CirculatingSupply)
	}
	wantDrop := -execResp.Transaction.TotalCurrencyDelta - execResp.FeeTransaction.TotalCurrencyDelta
	wantBalance := loginResp.User.CurrencyBalance - wantDrop
	if execResp.User.CurrencyBalance != wantBalance {
		t.Fatalf("balance after execute = %d, want %d", execResp.User.CurrencyBalance, wantBalance)
	}

	var sharesOwned int64
	if err := pool.QueryRow(context.Background(),
		"SELECT shares_owned FROM holdings WHERE card_id = $1", cardID,
	).Scan(&sharesOwned); err != nil {
		t.Fatalf("read holding: %v", err)
	}
	if sharesOwned != 50 {
		t.Fatalf("shares_owned = %d, want 50", sharesOwned)
	}
}

func TestExecuteTrade_RequiresAuth(t *testing.T) {
	r, pool := testRouter(t)
	cardID := createTestTradeCard(t, pool)

	req := httptest.NewRequest(http.MethodPost, "/api/trades/execute", bytes.NewReader(mustJSON(t, tradeExecuteRequest{
		CardID: cardID, Type: "BUY", Shares: 10, IdempotencyKey: uuid.NewString(),
	})))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}
