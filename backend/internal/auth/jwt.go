package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ErrInvalidToken is returned by ValidateToken for any malformed, expired,
// or badly-signed token.
var ErrInvalidToken = errors.New("invalid or expired token")

type claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// IssueToken issues an HS256 JWT carrying userID, expiring after ttl.
func IssueToken(secret []byte, userID uuid.UUID, ttl time.Duration) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	})
	return token.SignedString(secret)
}

// ValidateToken validates an HS256 JWT and returns the userID it carries.
func ValidateToken(secret []byte, tokenString string) (uuid.UUID, error) {
	var c claims
	token, err := jwt.ParseWithClaims(tokenString, &c, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(c.UserID)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	return userID, nil
}
