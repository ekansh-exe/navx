package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestGetTrades_RequiresAuth(t *testing.T) {
	r, _ := testRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/users/me/trades", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// TestGetTrades_NewestFirstWithPairedFee exercises the real HTTP surface:
// a fresh user's trade history starts empty, and after two trades it comes
// back newest-first, each row carrying its own fee transaction and card —
// exactly what the portfolio page's "recent trades" needs to survive a
// refresh (previously session-only client state, never persisted at all).
func TestGetTrades_NewestFirstWithPairedFee(t *testing.T) {
	r, pool := testRouter(t)
	cardID := createTestHoldingsCard(t, pool)

	username := "trades_" + uuid.NewString()[:8]
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

	emptyRec := doAuthJSON(t, r, http.MethodGet, "/api/users/me/trades", loginResp.Token, nil)
	if emptyRec.Code != http.StatusOK {
		t.Fatalf("trades status = %d, body = %s", emptyRec.Code, emptyRec.Body.String())
	}
	var emptyResp tradeHistoryResponse
	if err := json.Unmarshal(emptyRec.Body.Bytes(), &emptyResp); err != nil {
		t.Fatalf("decode empty trades response: %v", err)
	}
	if len(emptyResp.Trades) != 0 {
		t.Fatalf("trades for a brand-new user = %+v, want empty", emptyResp.Trades)
	}

	for _, shares := range []int64{10, 5} {
		rec := doAuthJSON(t, r, http.MethodPost, "/api/trades/execute", loginResp.Token, tradeExecuteRequest{
			CardID: cardID, Type: "BUY", Shares: shares, IdempotencyKey: uuid.NewString(),
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("execute (shares=%d) status = %d, body = %s", shares, rec.Code, rec.Body.String())
		}
	}

	tradesRec := doAuthJSON(t, r, http.MethodGet, "/api/users/me/trades", loginResp.Token, nil)
	var tradesResp tradeHistoryResponse
	if err := json.Unmarshal(tradesRec.Body.Bytes(), &tradesResp); err != nil {
		t.Fatalf("decode trades response: %v", err)
	}
	if len(tradesResp.Trades) != 2 {
		t.Fatalf("trades after 2 executions = %+v, want 2 entries", tradesResp.Trades)
	}
	// Newest first: the second (5-share) trade should lead.
	if tradesResp.Trades[0].Transaction.Shares == nil || *tradesResp.Trades[0].Transaction.Shares != 5 {
		t.Fatalf("Trades[0].Transaction.Shares = %v, want 5 (newest first)", tradesResp.Trades[0].Transaction.Shares)
	}
	if tradesResp.Trades[1].Transaction.Shares == nil || *tradesResp.Trades[1].Transaction.Shares != 10 {
		t.Fatalf("Trades[1].Transaction.Shares = %v, want 10", tradesResp.Trades[1].Transaction.Shares)
	}
	for i, entry := range tradesResp.Trades {
		if entry.FeeTransaction.Type != "FEE" {
			t.Fatalf("Trades[%d].FeeTransaction.Type = %q, want FEE", i, entry.FeeTransaction.Type)
		}
		if entry.Card.ID != cardID {
			t.Fatalf("Trades[%d].Card.ID = %s, want %s", i, entry.Card.ID, cardID)
		}
	}
}
