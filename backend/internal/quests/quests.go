// Package quests implements §7's daily quests: small currency rewards for
// hitting simple daily objectives (make N trades, hold a card for 24h,
// reach a leaderboard rank), issued through the same Transaction ledger as
// every other currency-affecting event in the system — never a separate
// ad-hoc balance update path.
package quests

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ekansh-exe/navx/internal/store/db"
)

const idempotencyKeyUniqueViolation = "23505"

// Service is the one type every quest-progress/read/reset path goes
// through — same shape as ledger.Ledger, news.Reader, leaderboard's job
// (constructs its own *db.Queries from the shared pool).
type Service struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func New(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, queries: db.New(pool)}
}

// nextMidnightUTC returns the next UTC midnight strictly after now — the
// reset boundary for every DAILY quest.
func nextMidnightUTC(now time.Time) time.Time {
	now = now.UTC()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return midnight.AddDate(0, 0, 1)
}
