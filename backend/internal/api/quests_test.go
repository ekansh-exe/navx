package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestGetQuests_RequiresAuth(t *testing.T) {
	r, _ := testRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/quests", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// TestGetQuests_DefaultsThenReflectsTradeProgress exercises the real HTTP
// surface end to end: a freshly registered user sees all seeded quests at
// 0 progress, and after one executed trade the MAKE_TRADES quest's
// progress updates — proving internal/api's ExecuteTrade hook actually
// reaches internal/quests, not just the package-level unit tests.
func TestGetQuests_DefaultsThenReflectsTradeProgress(t *testing.T) {
	r, pool := testRouter(t)
	cardID := createTestTradeCard(t, pool)

	username := "quests_" + uuid.NewString()[:8]
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, "DELETE FROM user_quests WHERE user_id IN (SELECT id FROM users WHERE username = $1)", username)
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

	questsRec := doAuthJSON(t, r, http.MethodGet, "/api/quests", loginResp.Token, nil)
	if questsRec.Code != http.StatusOK {
		t.Fatalf("quests status = %d, body = %s", questsRec.Code, questsRec.Body.String())
	}
	var questsResp questsResponse
	if err := json.Unmarshal(questsRec.Body.Bytes(), &questsResp); err != nil {
		t.Fatalf("decode quests response: %v", err)
	}
	if len(questsResp.Quests) != 3 {
		t.Fatalf("len(quests) = %d, want 3", len(questsResp.Quests))
	}
	for _, q := range questsResp.Quests {
		if q.Progress != 0 || q.Completed {
			t.Fatalf("quest %q for a brand-new user: progress=%d completed=%v, want 0/false", q.Title, q.Progress, q.Completed)
		}
	}

	execRec := doAuthJSON(t, r, http.MethodPost, "/api/trades/execute", loginResp.Token, tradeExecuteRequest{
		CardID: cardID, Type: "BUY", Shares: 10, IdempotencyKey: uuid.NewString(),
	})
	if execRec.Code != http.StatusOK {
		t.Fatalf("execute status = %d, body = %s", execRec.Code, execRec.Body.String())
	}

	questsRec2 := doAuthJSON(t, r, http.MethodGet, "/api/quests", loginResp.Token, nil)
	var questsResp2 questsResponse
	if err := json.Unmarshal(questsRec2.Body.Bytes(), &questsResp2); err != nil {
		t.Fatalf("decode second quests response: %v", err)
	}
	found := false
	for _, q := range questsResp2.Quests {
		if q.Title == "Make 3 trades today" {
			found = true
			if q.Progress != 1 {
				t.Fatalf("MAKE_TRADES progress after one trade = %d, want 1", q.Progress)
			}
		}
	}
	if !found {
		t.Fatal(`expected "Make 3 trades today" among the listed quests`)
	}
}
