package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIssueAndValidateToken(t *testing.T) {
	secret := []byte("test-secret")
	userID := uuid.New()

	token, err := IssueToken(secret, userID, time.Hour)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	got, err := ValidateToken(secret, token)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if got != userID {
		t.Fatalf("validated userID = %s, want %s", got, userID)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	token, err := IssueToken([]byte("secret-a"), uuid.New(), time.Hour)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	if _, err := ValidateToken([]byte("secret-b"), token); err == nil {
		t.Fatal("expected validation to fail with the wrong secret")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	secret := []byte("test-secret")
	token, err := IssueToken(secret, uuid.New(), -time.Hour) // already expired
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	if _, err := ValidateToken(secret, token); err == nil {
		t.Fatal("expected validation to fail for an expired token")
	}
}
