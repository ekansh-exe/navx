package quests

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/store/db"
)

// RecordTrade increments the MAKE_TRADES quest's progress by one (§7:
// "make 3 trades today"). Called once per successfully executed trade —
// the caller decides which trades count (see internal/api's hook, which
// only calls this for human-initiated trades; bot trades never reach it,
// mirroring §4.5/§8's bot-exclusion-from-human-facing-systems precedent).
func (s *Service) RecordTrade(ctx context.Context, userID uuid.UUID) error {
	return s.applyProgress(ctx, userID, domain.QuestTypeMakeTrades, func(current int32) int32 {
		return current + 1
	})
}

// completeIfConditionMet is for quest types that are boolean conditions
// rather than incrementing counters (HOLD_CARD, REACH_RANK, both seeded
// with target_value=1). Only called once the condition is confirmed true,
// and applyProgress already short-circuits once completed, so progress is
// always exactly 0 going in — +1 reaches target_value=1 and completes it.
func (s *Service) completeIfConditionMet(ctx context.Context, userID uuid.UUID, questType domain.QuestType) error {
	return s.applyProgress(ctx, userID, questType, func(current int32) int32 {
		return current + 1
	})
}

// applyProgress is the shared core for every quest-completion path: fetch
// the quest, get-or-create-and-lock the user's progress row, lazily reset
// it first if its period has already elapsed, skip entirely if already
// completed this period (so re-meeting the condition never double-grants),
// then apply computeNewProgress and atomically grant the reward in the
// same transaction if that reaches the quest's target.
//
// computeNewProgress receives the row's progress *after* any lazy reset
// and returns the progress to write this round.
func (s *Service) applyProgress(ctx context.Context, userID uuid.UUID, questType domain.QuestType, computeNewProgress func(currentProgress int32) int32) error {
	now := time.Now().UTC()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := s.queries.WithTx(tx)

	quest, err := qtx.GetQuestByType(ctx, string(questType))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // this quest type isn't seeded — nothing to track
		}
		return fmt.Errorf("get quest by type %q: %w", questType, err)
	}

	row, err := qtx.GetOrCreateUserQuestForUpdate(ctx, db.GetOrCreateUserQuestForUpdateParams{
		UserID:  userID,
		QuestID: quest.ID,
		ResetAt: nextMidnightUTC(now),
	})
	if err != nil {
		return fmt.Errorf("get or create user quest: %w", err)
	}

	progress, completedAt, resetAt := row.Progress, row.CompletedAt, row.ResetAt
	if !resetAt.After(now) {
		// This quest's period has already elapsed — reset before applying
		// anything new rather than building on stale progress. The bulk
		// reset job will catch this row too; no need to wait on it.
		progress, completedAt, resetAt = 0, nil, nextMidnightUTC(now)
	}

	if completedAt != nil {
		// Already completed this period: nothing further to do, not even a
		// write — this is what keeps a 4th trade (etc.) from double-granting.
		return tx.Commit(ctx)
	}

	newProgress := computeNewProgress(progress)
	if newProgress > quest.TargetValue {
		newProgress = quest.TargetValue
	}

	var newCompletedAt *time.Time
	justCompleted := newProgress >= quest.TargetValue
	if justCompleted {
		newCompletedAt = &now
	}

	if _, err := qtx.SetUserQuestState(ctx, db.SetUserQuestStateParams{
		UserID:      userID,
		QuestID:     quest.ID,
		Progress:    newProgress,
		CompletedAt: newCompletedAt,
		ResetAt:     resetAt,
	}); err != nil {
		return fmt.Errorf("set user quest state: %w", err)
	}

	if justCompleted {
		if err := grantReward(ctx, qtx, userID, quest, resetAt); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// grantReward credits a completed quest's reward_currency and writes the
// matching QUEST_REWARD transaction, atomically within the caller's
// already-open transaction — mirrors ledger.GrantDailyReward/
// RebalanceBotBalance's exact shape (lock user row, apply delta, write
// transaction with a deterministic idempotency key as a defense-in-depth
// backstop on top of the row lock).
func grantReward(ctx context.Context, qtx *db.Queries, userID uuid.UUID, quest db.Quest, periodResetAt time.Time) error {
	if _, err := qtx.GetUserForUpdate(ctx, userID); err != nil {
		return fmt.Errorf("get user for update: %w", err)
	}

	reward := int64(quest.RewardCurrency)
	updated, err := qtx.ApplyBalanceDelta(ctx, db.ApplyBalanceDeltaParams{ID: userID, CurrencyBalance: reward})
	if err != nil {
		return fmt.Errorf("apply balance delta: %w", err)
	}

	idempotencyKey := fmt.Sprintf("quest_reward:%s:%s:%s", userID, quest.ID, periodResetAt.Format(time.RFC3339))
	if _, err := qtx.CreateTransaction(ctx, db.CreateTransactionParams{
		UserID:             userID,
		Type:               string(domain.TransactionTypeQuestReward),
		TotalCurrencyDelta: reward,
		ResultingBalance:   updated.CurrencyBalance,
		IdempotencyKey:     &idempotencyKey,
	}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == idempotencyKeyUniqueViolation {
			// Already granted for this exact quest+period by a concurrent
			// caller — treat as done rather than surfacing a raw conflict.
			return nil
		}
		return fmt.Errorf("create quest reward transaction: %w", err)
	}
	return nil
}
