package store

import (
	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/store/db"
)

// ToDomainUser maps a store-layer row to the framework-free domain type,
// deliberately dropping PasswordHash so it can never leak past this layer.
func ToDomainUser(u db.User) *domain.User {
	return &domain.User{
		ID:               u.ID,
		Username:         u.Username,
		UserType:         domain.UserType(u.UserType),
		CurrencyBalance:  u.CurrencyBalance,
		LoginStreakCount: int(u.LoginStreakCount),
		LastLoginAt:      u.LastLoginAt,
		CreatedAt:        u.CreatedAt,
	}
}
