package ledger

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/store"
	"github.com/ekansh-exe/navx/internal/store/db"
)

// BotStartingBalance is the target a bot's currency_balance is reset to by
// RebalanceBotBalance (§4.5's nightly rebalancing job) — chosen well above
// the human default so a bot has room to make markets across every seeded
// card without a bad run of trades permanently draining it.
const BotStartingBalance int64 = 1_000_000

// RebalanceBotBalance resets a single bot's balance back to
// BotStartingBalance so one bot's bad luck can't permanently distort a
// card's price (§4.5). This is a currency-affecting operation, so — like
// every other one — it goes through this package and is recorded as a real
// REBALANCE transaction rather than a raw balance overwrite; it is not a
// trade and is not subject to trade-execution rules (fees, slippage,
// position caps, circuit breaker), since it isn't buying or selling
// anything. A no-op (no transaction written) if the bot is already exactly
// at the target.
func (l *Ledger) RebalanceBotBalance(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := l.queries.WithTx(tx)

	user, err := qtx.GetUserForUpdate(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user for update: %w", err)
	}

	delta := BotStartingBalance - user.CurrencyBalance
	if delta == 0 {
		return store.ToDomainUser(user), nil
	}

	updated, err := qtx.ApplyBalanceDelta(ctx, db.ApplyBalanceDeltaParams{ID: userID, CurrencyBalance: delta})
	if err != nil {
		return nil, fmt.Errorf("apply balance delta: %w", err)
	}

	now := time.Now().UTC()
	idempotencyKey := fmt.Sprintf("bot_rebalance:%s:%s", userID, now.Format(time.RFC3339Nano))
	if _, err := qtx.CreateTransaction(ctx, db.CreateTransactionParams{
		UserID:             userID,
		Type:               string(domain.TransactionTypeRebalance),
		TotalCurrencyDelta: delta,
		ResultingBalance:   updated.CurrencyBalance,
		IdempotencyKey:     &idempotencyKey,
	}); err != nil {
		return nil, fmt.Errorf("create rebalance transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return store.ToDomainUser(updated), nil
}
