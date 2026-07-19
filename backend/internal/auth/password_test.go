package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := hashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if !verifyPassword(hash, "correct horse battery staple") {
		t.Fatal("expected verifyPassword to succeed with the correct password")
	}
	if verifyPassword(hash, "wrong password") {
		t.Fatal("expected verifyPassword to fail with the wrong password")
	}
}
