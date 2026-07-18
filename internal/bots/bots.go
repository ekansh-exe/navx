package bots

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ekansh-exe/navx/internal/ledger"
	"github.com/ekansh-exe/navx/internal/store/db"
)

// Run starts one goroutine per seeded bot account (SeedAccounts) plus the
// nightly rebalance job, and returns immediately — matching the existing
// fire-and-forget goroutine pattern main.go already uses for the HTTP
// listener. Every goroutine selects on ctx.Done() and exits when the
// caller's context is cancelled (e.g. on SIGINT/SIGTERM). A seed account
// missing from the users table (e.g. migrations not yet applied) is logged
// and skipped rather than failing startup.
func Run(ctx context.Context, pool *pgxpool.Pool, ledgerSvc *ledger.Ledger, rebalanceInterval time.Duration) {
	queries := db.New(pool)

	started := 0
	for _, acct := range SeedAccounts {
		user, err := queries.GetUserByUsername(ctx, acct.Username)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				slog.Warn("BOT_ACCOUNT_MISSING", "username", acct.Username, "persona", acct.Persona)
			} else {
				slog.Error("BOT_ACCOUNT_LOOKUP_ERROR", "username", acct.Username, "error", err)
			}
			continue
		}
		bot := newBot(user.ID, user.Username, acct.Persona, ledgerSvc, queries)
		go bot.Run(ctx)
		started++
	}
	slog.Info("BOTS_STARTED", "count", started, "of", len(SeedAccounts))

	go RunRebalanceJob(ctx, ledgerSvc, queries, rebalanceInterval)
}
