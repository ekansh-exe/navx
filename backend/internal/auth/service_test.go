package auth

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ekansh-exe/navx/internal/ledger"
)

func testService(t *testing.T) (*Service, *pgxpool.Pool) {
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

	return NewService(pool, ledger.New(pool), []byte("test-secret"), time.Hour), pool
}

func cleanupUser(t *testing.T, pool *pgxpool.Pool, username string) {
	t.Helper()
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, "DELETE FROM transactions WHERE user_id IN (SELECT id FROM users WHERE username = $1)", username)
		pool.Exec(ctx, "DELETE FROM users WHERE username = $1", username)
	})
}

func TestRegister_HappyPathAndDuplicateUsername(t *testing.T) {
	svc, pool := testService(t)
	ctx := context.Background()
	username := "reg_" + uuid.NewString()[:8]
	cleanupUser(t, pool, username)

	user, err := svc.Register(ctx, username, "correct-password")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Username != username {
		t.Fatalf("username = %q, want %q", user.Username, username)
	}
	if user.CurrencyBalance != 100000 {
		t.Fatalf("currency_balance = %d, want 100000", user.CurrencyBalance)
	}

	if _, err := svc.Register(ctx, username, "another-password"); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("expected ErrUsernameTaken for duplicate username, got %v", err)
	}
}

func TestRegister_InvalidInput(t *testing.T) {
	svc, _ := testService(t)
	ctx := context.Background()

	if _, err := svc.Register(ctx, "ab", "goodpassword"); !errors.Is(err, ErrInvalidUsername) {
		t.Fatalf("expected ErrInvalidUsername for a too-short username, got %v", err)
	}
	if _, err := svc.Register(ctx, "validname", "short"); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("expected ErrInvalidPassword for a too-short password, got %v", err)
	}
}

func TestLogin_HappyPathAndWrongPassword(t *testing.T) {
	svc, pool := testService(t)
	ctx := context.Background()
	username := "login_" + uuid.NewString()[:8]
	cleanupUser(t, pool, username)

	if _, err := svc.Register(ctx, username, "correct-password"); err != nil {
		t.Fatalf("register: %v", err)
	}

	user, token, granted, err := svc.Login(ctx, username, "correct-password")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if token == "" {
		t.Fatal("expected a non-empty token")
	}
	if !granted {
		t.Fatal("expected the first login to grant the daily reward")
	}
	if user.CurrencyBalance != 100005 {
		t.Fatalf("currency_balance = %d, want 100005", user.CurrencyBalance)
	}

	if _, _, _, err := svc.Login(ctx, username, "wrong-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for wrong password, got %v", err)
	}

	if _, _, _, err := svc.Login(ctx, "no-such-user", "whatever"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for unknown username, got %v", err)
	}

	// Second correct login same day must not grant again.
	_, _, granted2, err := svc.Login(ctx, username, "correct-password")
	if err != nil {
		t.Fatalf("second login: %v", err)
	}
	if granted2 {
		t.Fatal("expected second same-day login to not grant the reward again")
	}
}

// TestLoginReward_Idempotent is a focused, dedicated check (§7) that the
// same user logging in twice on the same UTC calendar day grants the +5
// daily reward exactly once, not twice — the balance after the second
// login must equal the balance after the first, not first+5 again.
func TestLoginReward_Idempotent(t *testing.T) {
	svc, pool := testService(t)
	ctx := context.Background()
	username := "loginidem_" + uuid.NewString()[:8]
	cleanupUser(t, pool, username)

	if _, err := svc.Register(ctx, username, "correct-password"); err != nil {
		t.Fatalf("register: %v", err)
	}

	first, _, granted1, err := svc.Login(ctx, username, "correct-password")
	if err != nil {
		t.Fatalf("first login: %v", err)
	}
	if !granted1 {
		t.Fatal("expected the first login of the day to grant the reward")
	}

	second, _, granted2, err := svc.Login(ctx, username, "correct-password")
	if err != nil {
		t.Fatalf("second login: %v", err)
	}
	if granted2 {
		t.Fatal("expected the second same-day login to not grant the reward again")
	}
	if second.CurrencyBalance != first.CurrencyBalance {
		t.Fatalf("balance after second login = %d, want unchanged from first login's %d", second.CurrencyBalance, first.CurrencyBalance)
	}

	third, _, granted3, err := svc.Login(ctx, username, "correct-password")
	if err != nil {
		t.Fatalf("third login: %v", err)
	}
	if granted3 || third.CurrencyBalance != first.CurrencyBalance {
		t.Fatalf("expected a third same-day login to also not grant again (balance %d, want %d, granted=%v)", third.CurrencyBalance, first.CurrencyBalance, granted3)
	}
}
