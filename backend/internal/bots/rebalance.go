package bots

import (
	"context"
	"log/slog"
	"time"

	"github.com/ekansh-exe/navx/internal/ledger"
	"github.com/ekansh-exe/navx/internal/store/db"
)

// RunRebalanceJob periodically resets every BOT-type user's balance back to
// ledger.BotStartingBalance (§4.5: "so a bot's strategy going wrong can't
// permanently distort a card's price or drain toward one side"), logging
// each non-zero reset. Runs until ctx is cancelled.
func RunRebalanceJob(ctx context.Context, ledgerSvc *ledger.Ledger, queries *db.Queries, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rebalanceOnce(ctx, ledgerSvc, queries)
		}
	}
}

func rebalanceOnce(ctx context.Context, ledgerSvc *ledger.Ledger, queries *db.Queries) {
	botUsers, err := queries.ListUsersByType(ctx, "BOT")
	if err != nil {
		slog.Error("BOT_REBALANCE_ERROR", "error", err)
		return
	}

	var resetCount int
	for _, u := range botUsers {
		before := u.CurrencyBalance
		updated, err := ledgerSvc.RebalanceBotBalance(ctx, u.ID)
		if err != nil {
			slog.Error("BOT_REBALANCE_ERROR", "bot", u.Username, "error", err)
			continue
		}
		if updated.CurrencyBalance != before {
			resetCount++
			slog.Info("BOT_REBALANCED", "bot", u.Username, "balance_before", before, "balance_after", updated.CurrencyBalance)
		}
	}
	slog.Info("BOT_REBALANCE_CYCLE_COMPLETE", "bots_checked", len(botUsers), "bots_reset", resetCount)
}
