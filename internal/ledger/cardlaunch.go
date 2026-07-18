package ledger

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/engine"
	"github.com/ekansh-exe/navx/internal/store"
	"github.com/ekansh-exe/navx/internal/store/db"
)

// cardsSymbolUniqueConstraint is Postgres's default auto-generated name for
// the inline `symbol TEXT UNIQUE` constraint on cards (migration 000001) —
// used to tell a real symbol collision apart from an idempotency-key
// collision, since both surface as the same 23505 code.
const cardsSymbolUniqueConstraint = "cards_symbol_key"

const (
	// LaunchThreshold is the minimum currency_balance required before
	// "Launch a Card" unlocks (§6, step 1: "Player crosses currency
	// threshold"). Set comfortably above LaunchCost so meeting the
	// threshold always implies being able to afford the launch too.
	LaunchThreshold int64 = 50_000

	// LaunchCost is the currency sink deducted on every launch (§6, step
	// 3: "a currency cost to launch... to control spam").
	LaunchCost int64 = 10_000

	// MaxRetainedPercent is §4.3's own suggested cap ("e.g. cap at 40%"),
	// adopted directly rather than picked freely.
	MaxRetainedPercent float64 = 0.40

	// launchBasePrice/launchScale are the fixed curve constants assigned
	// to every user-launched card, mirroring how Phase 3 anchored the 30
	// seeded companies (base_price/scale aren't player-set fields per §6's
	// flow — the server decides them). scale = total_supply, same as the
	// FIXED-company backfill in migration 000005.
	launchBasePrice float64 = 10
)

var (
	ErrBelowLaunchThreshold   = errors.New("currency balance is below the card-launch threshold")
	ErrInvalidTotalSupply     = errors.New("total_supply must be a positive number")
	ErrInvalidRetainedPercent = errors.New("retained_percent must be between 0 and the maximum allowed")
	ErrSymbolTaken            = errors.New("symbol is already in use")
	ErrRetainedSharesLocked   = errors.New("cannot sell that many retained shares yet — still vesting")
)

// LaunchCardParams describes a user-generated card launch (§6). Only FIXED
// supply is supported for user-created cards this phase — UNLIMITED
// user-cards raise unresolved questions (e.g. what "retained percent" is a
// percent *of* with no total_supply) that the spec doesn't answer, so
// that's deferred rather than guessed at.
type LaunchCardParams struct {
	CreatorUserID   uuid.UUID
	Symbol          string
	Name            string
	Description     *string
	ImageURL        *string
	TotalSupply     int64
	RetainedPercent float64
	IdempotencyKey  string
}

type LaunchCardResult struct {
	Card        *domain.Card
	Transaction *domain.Transaction
	User        *domain.User
}

// LaunchCard creates a new USER_CREATED card, mints CreatorRetainedShares
// into the creator's holdings, and deducts LaunchCost — all as one atomic
// transaction, idempotent on IdempotencyKey (same pattern as ExecuteTrade).
func (l *Ledger) LaunchCard(ctx context.Context, params LaunchCardParams) (*LaunchCardResult, error) {
	if params.TotalSupply <= 0 {
		return nil, ErrInvalidTotalSupply
	}
	if params.RetainedPercent < 0 || params.RetainedPercent > MaxRetainedPercent {
		return nil, ErrInvalidRetainedPercent
	}

	if existing, err := l.queries.GetTransactionByIdempotencyKey(ctx, &params.IdempotencyKey); err == nil {
		return l.buildLaunchReplayResult(ctx, existing)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check idempotency key: %w", err)
	}

	result, err := l.launchCardTx(ctx, params)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == idempotencyKeyUniqueViolation {
			// Distinguish by constraint name, not just the shared 23505
			// code: a symbol collision (real "this name is taken") and an
			// idempotency-key collision (a race with a concurrent replay)
			// need different responses.
			if pgErr.ConstraintName == cardsSymbolUniqueConstraint {
				return nil, ErrSymbolTaken
			}
			existing, ferr := l.queries.GetTransactionByIdempotencyKey(ctx, &params.IdempotencyKey)
			if ferr != nil {
				return nil, fmt.Errorf("fetch after idempotency conflict: %w", ferr)
			}
			return l.buildLaunchReplayResult(ctx, existing)
		}
		return nil, err
	}
	return result, nil
}

func (l *Ledger) launchCardTx(ctx context.Context, params LaunchCardParams) (*LaunchCardResult, error) {
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := l.queries.WithTx(tx)

	// Only one row is locked here (the creator's), so there's no ordering
	// conflict with ExecuteTrade's card->user->holdings order — a launch
	// creates a brand new card row, there's nothing existing to lock.
	user, err := qtx.GetUserForUpdate(ctx, params.CreatorUserID)
	if err != nil {
		return nil, fmt.Errorf("get user for update: %w", err)
	}
	if user.CurrencyBalance < LaunchThreshold {
		return nil, ErrBelowLaunchThreshold
	}
	newBalance := user.CurrencyBalance - LaunchCost
	if newBalance < 0 {
		return nil, ErrInsufficientBalance
	}

	retainedShares := int64(math.Round(float64(params.TotalSupply) * params.RetainedPercent))
	scale := float64(params.TotalSupply)
	curveParams := engine.CurveParams{BasePrice: launchBasePrice, Scale: scale, DemandModifier: 1, DriftFactor: 1}
	startPrice, err := engine.SpotPrice(retainedShares, curveParams)
	if err != nil {
		return nil, fmt.Errorf("compute launch price: %w", err)
	}

	card, err := qtx.CreateUserCard(ctx, db.CreateUserCardParams{
		CreatorUserID:     &params.CreatorUserID,
		Symbol:            params.Symbol,
		Name:              params.Name,
		Description:       params.Description,
		ImageUrl:          params.ImageURL,
		SupplyModel:       string(domain.SupplyModelFixed),
		TotalSupply:       &params.TotalSupply,
		CirculatingSupply: retainedShares,
		BasePrice:         launchBasePrice,
		Scale:             scale,
		CurrentPrice:      startPrice,
	})
	if err != nil {
		// A symbol 23505 here is handled by the caller (LaunchCard).
		return nil, err
	}

	if _, err := qtx.UpsertHolding(ctx, db.UpsertHoldingParams{
		UserID:       params.CreatorUserID,
		CardID:       card.ID,
		SharesOwned:  retainedShares,
		AvgCostBasis: 0, // granted at launch, not purchased at a per-share price
	}); err != nil {
		return nil, fmt.Errorf("mint retained holding: %w", err)
	}

	updatedUser, err := qtx.ApplyBalanceDelta(ctx, db.ApplyBalanceDeltaParams{
		ID:              params.CreatorUserID,
		CurrencyBalance: -LaunchCost,
	})
	if err != nil {
		return nil, fmt.Errorf("deduct launch cost: %w", err)
	}

	idempotencyKey := params.IdempotencyKey
	launchTxn, err := qtx.CreateTransaction(ctx, db.CreateTransactionParams{
		UserID:             params.CreatorUserID,
		CardID:             &card.ID,
		Type:               string(domain.TransactionTypeCardLaunch),
		Shares:             &retainedShares,
		TotalCurrencyDelta: -LaunchCost,
		ResultingBalance:   newBalance,
		IdempotencyKey:     &idempotencyKey,
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &LaunchCardResult{
		Card:        store.ToDomainCard(card),
		Transaction: store.ToDomainTransaction(launchTxn),
		User:        store.ToDomainUser(updatedUser),
	}, nil
}

func (l *Ledger) buildLaunchReplayResult(ctx context.Context, existing db.Transaction) (*LaunchCardResult, error) {
	if existing.CardID == nil {
		return nil, fmt.Errorf("launch transaction %s has no card_id", existing.ID)
	}
	card, err := l.queries.GetCardByID(ctx, *existing.CardID)
	if err != nil {
		return nil, fmt.Errorf("fetch card: %w", err)
	}
	user, err := l.queries.GetUserByID(ctx, existing.UserID)
	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", err)
	}
	return &LaunchCardResult{
		Card:        store.ToDomainCard(card),
		Transaction: store.ToDomainTransaction(existing),
		User:        store.ToDomainUser(user),
	}, nil
}
