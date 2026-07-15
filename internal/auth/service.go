package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/ledger"
	"github.com/ekansh-exe/navx/internal/store"
	"github.com/ekansh-exe/navx/internal/store/db"
)

const uniqueViolationCode = "23505"

// Service implements registration and login (§10, §11.1). It never returns
// password_hash to callers — that field stops at the store layer.
type Service struct {
	queries   *db.Queries
	ledger    *ledger.Ledger
	jwtSecret []byte
	jwtTTL    time.Duration
}

func NewService(pool *pgxpool.Pool, ledger *ledger.Ledger, jwtSecret []byte, jwtTTL time.Duration) *Service {
	return &Service{
		queries:   db.New(pool),
		ledger:    ledger,
		jwtSecret: jwtSecret,
		jwtTTL:    jwtTTL,
	}
}

// Register creates a new HUMAN user. It does not grant the daily login
// reward — the reward path starts on the user's first explicit Login call.
func (s *Service) Register(ctx context.Context, username, password string) (*domain.User, error) {
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	created, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Username:     username,
		PasswordHash: &hash,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return nil, ErrUsernameTaken
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return store.ToDomainUser(created), nil
}

// Login verifies credentials, grants the daily reward if not already
// granted today, and issues a JWT. rewardGranted reports whether this call
// actually granted today's reward (false on a second login the same day).
func (s *Service) Login(ctx context.Context, username, password string) (user *domain.User, token string, rewardGranted bool, err error) {
	found, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, "", false, ErrInvalidCredentials
	}

	if found.PasswordHash == nil || !verifyPassword(*found.PasswordHash, password) {
		return nil, "", false, ErrInvalidCredentials
	}

	updated, granted, err := s.ledger.GrantDailyReward(ctx, found.ID)
	if err != nil {
		return nil, "", false, fmt.Errorf("grant daily reward: %w", err)
	}

	tok, err := IssueToken(s.jwtSecret, updated.ID, s.jwtTTL)
	if err != nil {
		return nil, "", false, fmt.Errorf("issue token: %w", err)
	}

	return updated, tok, granted, nil
}

// GetUser fetches a user by ID (used by the JWT-protected /me endpoint).
func (s *Service) GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	found, err := s.queries.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return store.ToDomainUser(found), nil
}

func validateUsername(username string) error {
	if len(username) < 3 || len(username) > 32 {
		return ErrInvalidUsername
	}
	return nil
}

func validatePassword(password string) error {
	// bcrypt hard-errors above 72 bytes, so this bound is enforced here
	// rather than surfacing a raw bcrypt error to the caller.
	if len(password) < 8 || len(password) > 72 {
		return ErrInvalidPassword
	}
	return nil
}
