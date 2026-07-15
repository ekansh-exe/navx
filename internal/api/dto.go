package api

import (
	"time"

	"github.com/google/uuid"

	"github.com/ekansh-exe/navx/internal/domain"
)

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userDTO struct {
	ID               uuid.UUID  `json:"id"`
	Username         string     `json:"username"`
	UserType         string     `json:"user_type"`
	CurrencyBalance  int64      `json:"currency_balance"`
	LoginStreakCount int        `json:"login_streak_count"`
	LastLoginAt      *time.Time `json:"last_login_at"`
	CreatedAt        time.Time  `json:"created_at"`
}

func toUserDTO(u *domain.User) userDTO {
	return userDTO{
		ID:               u.ID,
		Username:         u.Username,
		UserType:         string(u.UserType),
		CurrencyBalance:  u.CurrencyBalance,
		LoginStreakCount: u.LoginStreakCount,
		LastLoginAt:      u.LastLoginAt,
		CreatedAt:        u.CreatedAt,
	}
}

type loginResponse struct {
	Token         string  `json:"token"`
	User          userDTO `json:"user"`
	RewardGranted bool    `json:"reward_granted"`
	RewardAmount  int64   `json:"reward_amount"`
}

type errorResponse struct {
	Error string `json:"error"`
}
