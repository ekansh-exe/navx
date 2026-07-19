package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// createTestHoldingsCard mirrors createTestTradeCard but with a circulating
// supply/scale large enough that a 10-share buy-then-sell round trip barely
// moves the rounded integer price — createTestTradeCard's tiny supply (1000)
// rounds to a price of 0 at this position size, which confuses the circuit
// breaker's percentage-move math on the very next trade. Unrelated to
// anything this test is actually exercising, so sidestepped here rather than
// chasing that edge case.
func createTestHoldingsCard(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO cards (card_type, symbol, name, supply_model, total_supply, circulating_supply, base_price, scale, current_price, status)
		VALUES ('SYSTEM_COMPANY', $1, $1, 'FIXED', 2000000, 500000, 1000, 1000000000, 22, 'ACTIVE')
		RETURNING id`,
		"APIHLD_"+uuid.NewString()[:8],
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

func TestGetHoldings_RequiresAuth(t *testing.T) {
	r, _ := testRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/users/me/holdings", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// TestGetHoldings_ReflectsBuyThenExcludesClosedPosition exercises the real
// HTTP surface: a freshly registered user starts with no holdings, a BUY
// makes the position appear with the right share count, and selling back
// down to 0 shares makes it disappear again (a closed-out position isn't a
// "holding" from the portfolio page's perspective — see
// ledger.ListHoldingsByUser).
func TestGetHoldings_ReflectsBuyThenExcludesClosedPosition(t *testing.T) {
	r, pool := testRouter(t)
	cardID := createTestHoldingsCard(t, pool)

	username := "holdings_" + uuid.NewString()[:8]
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

	emptyRec := doAuthJSON(t, r, http.MethodGet, "/api/users/me/holdings", loginResp.Token, nil)
	if emptyRec.Code != http.StatusOK {
		t.Fatalf("holdings status = %d, body = %s", emptyRec.Code, emptyRec.Body.String())
	}
	var emptyResp holdingsResponse
	if err := json.Unmarshal(emptyRec.Body.Bytes(), &emptyResp); err != nil {
		t.Fatalf("decode empty holdings response: %v", err)
	}
	if len(emptyResp.Holdings) != 0 {
		t.Fatalf("holdings for a brand-new user = %+v, want empty", emptyResp.Holdings)
	}

	execRec := doAuthJSON(t, r, http.MethodPost, "/api/trades/execute", loginResp.Token, tradeExecuteRequest{
		CardID: cardID, Type: "BUY", Shares: 10, IdempotencyKey: uuid.NewString(),
	})
	if execRec.Code != http.StatusOK {
		t.Fatalf("execute status = %d, body = %s", execRec.Code, execRec.Body.String())
	}

	afterBuyRec := doAuthJSON(t, r, http.MethodGet, "/api/users/me/holdings", loginResp.Token, nil)
	var afterBuyResp holdingsResponse
	if err := json.Unmarshal(afterBuyRec.Body.Bytes(), &afterBuyResp); err != nil {
		t.Fatalf("decode post-buy holdings response: %v", err)
	}
	if len(afterBuyResp.Holdings) != 1 {
		t.Fatalf("holdings after buying = %+v, want exactly 1 position", afterBuyResp.Holdings)
	}
	if afterBuyResp.Holdings[0].CardID != cardID || afterBuyResp.Holdings[0].SharesOwned != 10 {
		t.Fatalf("holding = %+v, want card_id=%s shares_owned=10", afterBuyResp.Holdings[0], cardID)
	}

	sellRec := doAuthJSON(t, r, http.MethodPost, "/api/trades/execute", loginResp.Token, tradeExecuteRequest{
		CardID: cardID, Type: "SELL", Shares: 10, IdempotencyKey: uuid.NewString(),
	})
	if sellRec.Code != http.StatusOK {
		t.Fatalf("sell status = %d, body = %s", sellRec.Code, sellRec.Body.String())
	}

	afterSellRec := doAuthJSON(t, r, http.MethodGet, "/api/users/me/holdings", loginResp.Token, nil)
	var afterSellResp holdingsResponse
	if err := json.Unmarshal(afterSellRec.Body.Bytes(), &afterSellResp); err != nil {
		t.Fatalf("decode post-sell holdings response: %v", err)
	}
	if len(afterSellResp.Holdings) != 0 {
		t.Fatalf("holdings after selling everything = %+v, want empty (closed positions are excluded)", afterSellResp.Holdings)
	}
}
