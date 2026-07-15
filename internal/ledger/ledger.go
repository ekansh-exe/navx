package ledger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/store"
	"github.com/ekansh-exe/navx/internal/store/db"
)

const idempotencyKeyUniqueViolation = "23505"

// DailyRewardAmount is the flat daily login reward (§7).
const DailyRewardAmount int64 = 5

// Ledger is the one package every currency-affecting operation goes through
// (§7, §11.1) — trade execution (Phase 3) will live here too, reusing the
// same locked-transaction shape as GrantDailyReward.
type Ledger struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func New(pool *pgxpool.Pool) *Ledger {
	return &Ledger{pool: pool, queries: db.New(pool)}
}

// GrantDailyReward grants the flat +5 daily login reward (§7) if the user
// hasn't already received it for today's UTC calendar date. Idempotent —
// safe to call on every login. The user row is locked for the duration of
// the check-and-grant (§4.2's atomic pattern) so concurrent logins can't
// double-grant; the transaction's idempotency_key is a defense-in-depth
// backstop on top of that lock, not the primary serialization mechanism.
func (l *Ledger) GrantDailyReward(ctx context.Context, userID uuid.UUID) (*domain.User, bool, error) {
	now := time.Now().UTC()

	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := l.queries.WithTx(tx)

	user, err := qtx.GetUserForUpdate(ctx, userID)
	if err != nil {
		return nil, false, fmt.Errorf("get user for update: %w", err)
	}

	newStreak, alreadyGranted := nextLoginState(user.LastLoginAt, user.LoginStreakCount, now)
	if alreadyGranted {
		return store.ToDomainUser(user), false, nil
	}

	updated, err := qtx.ApplyDailyReward(ctx, db.ApplyDailyRewardParams{
		ID:               userID,
		CurrencyBalance:  DailyRewardAmount,
		LastLoginAt:      &now,
		LoginStreakCount: newStreak,
	})
	if err != nil {
		return nil, false, fmt.Errorf("apply daily reward: %w", err)
	}

	idempotencyKey := fmt.Sprintf("daily_reward:%s:%s", userID, now.Format("2006-01-02"))
	if _, err := qtx.CreateTransaction(ctx, db.CreateTransactionParams{
		UserID:             userID,
		Type:               string(domain.TransactionTypeDailyReward),
		TotalCurrencyDelta: DailyRewardAmount,
		ResultingBalance:   updated.CurrencyBalance,
		IdempotencyKey:     &idempotencyKey,
	}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == idempotencyKeyUniqueViolation {
			// A DAILY_REWARD transaction for today's date already exists
			// even though last_login_at said otherwise (e.g. the two got
			// out of sync some other way) — treat as already-granted
			// rather than surfacing a raw constraint error.
			return store.ToDomainUser(user), false, nil
		}
		return nil, false, fmt.Errorf("create transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("commit tx: %w", err)
	}

	return store.ToDomainUser(updated), true, nil
}

// nextLoginState is the pure decision logic behind GrantDailyReward, kept
// separate so it's directly unit-testable with arbitrary dates rather than
// only through a database (real UTC "today" can't be fast-forwarded in a
// test). Returns the streak GrantDailyReward should persist, and whether the
// user has already been granted today's reward (in which case no write
// should happen at all).
func nextLoginState(lastLoginAt *time.Time, currentStreak int32, now time.Time) (streak int32, alreadyGranted bool) {
	today := truncateToDate(now)

	if lastLoginAt != nil && truncateToDate(*lastLoginAt).Equal(today) {
		return currentStreak, true
	}
	if lastLoginAt != nil && truncateToDate(*lastLoginAt).Equal(today.AddDate(0, 0, -1)) {
		return currentStreak + 1, false
	}
	return 1, false
}

func truncateToDate(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
