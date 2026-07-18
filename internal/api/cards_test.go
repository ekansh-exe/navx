package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestLaunchCardFlow(t *testing.T) {
	r, pool := testRouter(t)

	username := "launcher_" + uuid.NewString()[:8]
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, "DELETE FROM holdings WHERE user_id IN (SELECT id FROM users WHERE username = $1)", username)
		pool.Exec(ctx, "DELETE FROM transactions WHERE user_id IN (SELECT id FROM users WHERE username = $1)", username)
		pool.Exec(ctx, "DELETE FROM cards WHERE creator_user_id IN (SELECT id FROM users WHERE username = $1)", username)
		pool.Exec(ctx, "DELETE FROM users WHERE username = $1", username)
	})

	doJSON(t, r, http.MethodPost, "/api/auth/register", registerRequest{Username: username, Password: "correct-password"})
	loginRec := doJSON(t, r, http.MethodPost, "/api/auth/login", loginRequest{Username: username, Password: "correct-password"})
	var loginResp loginResponse
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	desc := "an api-launched card"
	launchRec := doAuthJSON(t, r, http.MethodPost, "/api/cards", loginResp.Token, launchCardRequest{
		Symbol:          "APILNCH_" + uuid.NewString()[:6],
		Name:            "API Launch Co.",
		Description:     &desc,
		TotalSupply:     10_000,
		RetainedPercent: 0.3,
		IdempotencyKey:  uuid.NewString(),
	})
	if launchRec.Code != http.StatusCreated {
		t.Fatalf("launch status = %d, body = %s", launchRec.Code, launchRec.Body.String())
	}
	var launchResp launchCardResponse
	if err := json.Unmarshal(launchRec.Body.Bytes(), &launchResp); err != nil {
		t.Fatalf("decode launch response: %v", err)
	}

	if launchResp.Card.CardType != "USER_CREATED" {
		t.Fatalf("card_type = %s, want USER_CREATED", launchResp.Card.CardType)
	}
	wantRetained := int64(3000)
	if launchResp.Card.CreatorRetainedShares != wantRetained {
		t.Fatalf("creator_retained_shares = %d, want %d", launchResp.Card.CreatorRetainedShares, wantRetained)
	}
	if launchResp.User.CurrencyBalance != loginResp.User.CurrencyBalance-10_000 {
		t.Fatalf("balance after launch = %d, want %d", launchResp.User.CurrencyBalance, loginResp.User.CurrencyBalance-10_000)
	}

	// Attempting to sell any retained shares immediately must fail — still vesting.
	sellRec := doAuthJSON(t, r, http.MethodPost, "/api/trades/execute", loginResp.Token, tradeExecuteRequest{
		CardID: launchResp.Card.ID, Type: "SELL", Shares: 1, IdempotencyKey: uuid.NewString(),
	})
	if sellRec.Code != http.StatusConflict {
		t.Fatalf("sell-immediately-after-launch status = %d, body = %s, want %d", sellRec.Code, sellRec.Body.String(), http.StatusConflict)
	}
}

func TestLaunchCard_RequiresAuth(t *testing.T) {
	r, _ := testRouter(t)

	req := doJSON(t, r, http.MethodPost, "/api/cards", launchCardRequest{
		Symbol: "NOAUTH", Name: "x", TotalSupply: 1000, RetainedPercent: 0.1, IdempotencyKey: uuid.NewString(),
	})
	if req.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", req.Code, http.StatusUnauthorized)
	}
}
