package ledger

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/engine"
	"github.com/ekansh-exe/navx/internal/store"
	"github.com/ekansh-exe/navx/internal/store/db"
)

// feeRate and minFee implement §4.2.4's "0.5-1% fee on both buy and sell" —
// 1% (the stronger anti-wash-trading deterrent) with a floor so a fee can
// never round down to zero and let a small trade round-trip for free.
const (
	feeRate = 0.01
	minFee  = int64(1)
)

var (
	ErrInvalidShares          = errors.New("shares must be a positive number")
	ErrInvalidTradeType       = errors.New("trade type must be BUY or SELL")
	ErrCardNotFound           = errors.New("card not found")
	ErrCardNotTradable        = errors.New("card is not currently active")
	ErrInsufficientSupply     = errors.New("not enough remaining supply for this card")
	ErrInsufficientShares     = errors.New("cannot sell more shares than you own")
	ErrInsufficientBalance    = errors.New("insufficient balance for this trade")
	ErrIdempotencyKeyMismatch = errors.New("idempotency key was already used for a different trade")
)

// TradeParams describes a single buy or sell request (§4.2). Shares is
// always positive; Type carries the direction.
type TradeParams struct {
	UserID         uuid.UUID
	CardID         uuid.UUID
	Type           domain.TransactionType
	Shares         int64
	IdempotencyKey string
}

// TradeResult is what a successful (or idempotently-replayed) trade returns.
type TradeResult struct {
	Transaction    *domain.Transaction
	FeeTransaction *domain.Transaction
	User           *domain.User
	Card           *domain.Card
}

// QuoteParams/QuoteResult back the non-binding preview endpoint (§10).
type QuoteParams struct {
	CardID uuid.UUID
	Type   domain.TransactionType
	Shares int64
}

type QuoteResult struct {
	Card          *domain.Card
	Cost          int64 // signed per engine.ExecutionCost's convention: positive=buyer pays, negative=seller receives
	Fee           int64
	PricePerShare int64
}

func validateTradeShapeAndType(shares int64, tradeType domain.TransactionType) error {
	if shares <= 0 {
		return ErrInvalidShares
	}
	if tradeType != domain.TransactionTypeBuy && tradeType != domain.TransactionTypeSell {
		return ErrInvalidTradeType
	}
	return nil
}

func deltaSharesFor(tradeType domain.TransactionType, shares int64) int64 {
	if tradeType == domain.TransactionTypeSell {
		return -shares
	}
	return shares
}

// computeFee is shared by Quote and ExecuteTrade so the fee formula can
// never drift between the preview and the real execution.
func computeFee(cost int64) int64 {
	fee := int64(math.Round(math.Abs(float64(cost)) * feeRate))
	if fee < minFee {
		fee = minFee
	}
	return fee
}

func roundedPricePerShare(cost, shares int64) int64 {
	return int64(math.Round(math.Abs(float64(cost)) / float64(shares)))
}

// Quote is a non-binding preview (§10) — a plain unlocked read, no mutation.
func (l *Ledger) Quote(ctx context.Context, params QuoteParams) (*QuoteResult, error) {
	if err := validateTradeShapeAndType(params.Shares, params.Type); err != nil {
		return nil, err
	}

	card, err := l.queries.GetCardByID(ctx, params.CardID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCardNotFound
	} else if err != nil {
		return nil, fmt.Errorf("get card: %w", err)
	}

	deltaShares := deltaSharesFor(params.Type, params.Shares)
	curveParams := engine.CurveParams{BasePrice: card.BasePrice, Scale: card.Scale, DemandModifier: 1, DriftFactor: 1}

	cost, err := engine.ExecutionCost(card.CirculatingSupply, deltaShares, curveParams)
	if err != nil {
		return nil, fmt.Errorf("compute execution cost: %w", err)
	}

	return &QuoteResult{
		Card:          store.ToDomainCard(card),
		Cost:          cost,
		Fee:           computeFee(cost),
		PricePerShare: roundedPricePerShare(cost, params.Shares),
	}, nil
}

// ExecuteTrade runs a real buy or sell (§4.2): recomputes price at
// execution time inside a single locked DB transaction, applies slippage
// (via engine.ExecutionCost's curve integration) and the fee, and writes
// the ledger atomically. Idempotent on IdempotencyKey.
func (l *Ledger) ExecuteTrade(ctx context.Context, params TradeParams) (*TradeResult, error) {
	if err := validateTradeShapeAndType(params.Shares, params.Type); err != nil {
		return nil, err
	}

	// Fast path: unlocked idempotent replay check.
	if existing, err := l.queries.GetTransactionByIdempotencyKey(ctx, &params.IdempotencyKey); err == nil {
		return l.buildReplayResult(ctx, existing, params)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check idempotency key: %w", err)
	}

	result, err := l.executeTradeTx(ctx, params)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == idempotencyKeyUniqueViolation {
			// A concurrent request with the same key committed first; this
			// attempt's mutations were rolled back by the transaction abort
			// (never salvage anything computed here — refetch fresh state).
			existing, ferr := l.queries.GetTransactionByIdempotencyKey(ctx, &params.IdempotencyKey)
			if ferr != nil {
				return nil, fmt.Errorf("fetch after idempotency conflict: %w", ferr)
			}
			return l.buildReplayResult(ctx, existing, params)
		}
		return nil, err
	}
	return result, nil
}

func (l *Ledger) executeTradeTx(ctx context.Context, params TradeParams) (*TradeResult, error) {
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := l.queries.WithTx(tx)

	now := time.Now().UTC()

	// Lock order is fixed globally across every ledger function that ever
	// touches more than one of these rows: card, then user, then holdings.
	// This is what prevents deadlocks between concurrent trades — it must
	// stay this order for any future function (e.g. CARD_LAUNCH) that also
	// locks more than one of these.
	card, err := qtx.GetCardForUpdate(ctx, params.CardID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCardNotFound
	} else if err != nil {
		return nil, fmt.Errorf("get card for update: %w", err)
	}

	// Circuit breaker (§4.3): halts both BUY and SELL while active, checked
	// before anything else so a halted card short-circuits immediately.
	if card.CircuitBreakerHaltedUntil != nil && now.Before(*card.CircuitBreakerHaltedUntil) {
		return nil, ErrCircuitBreakerActive
	}

	// BUY requires an ACTIVE card; SELL is always allowed regardless of
	// status so a user is never trapped holding a position with no way to
	// exit (a deliberate call — nothing sets a non-ACTIVE status yet).
	if params.Type == domain.TransactionTypeBuy && card.Status != string(domain.CardStatusActive) {
		return nil, ErrCardNotTradable
	}

	user, err := qtx.GetUserForUpdate(ctx, params.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user for update: %w", err)
	}

	deltaShares := deltaSharesFor(params.Type, params.Shares)
	newSupply := card.CirculatingSupply + deltaShares
	if params.Type == domain.TransactionTypeBuy && card.TotalSupply != nil && newSupply > *card.TotalSupply {
		return nil, ErrInsufficientSupply
	}

	holding, err := qtx.GetHoldingForUpdate(ctx, db.GetHoldingForUpdateParams{UserID: params.UserID, CardID: params.CardID})
	sharesOwned, avgCostBasis := int64(0), int64(0)
	if err == nil {
		sharesOwned, avgCostBasis = holding.SharesOwned, holding.AvgCostBasis
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get holding for update: %w", err)
	}
	if params.Type == domain.TransactionTypeSell && params.Shares > sharesOwned {
		return nil, ErrInsufficientShares
	}

	// Position cap (§4.3): BUY-only, since selling only ever reduces a
	// position and can never breach a max-ownership cap.
	if params.Type == domain.TransactionTypeBuy {
		newSharesOwned := sharesOwned + deltaShares
		if positionCapBreached(newSharesOwned, newSupply) {
			return nil, ErrPositionCapExceeded
		}
	}

	// Creator vesting (§4.3, §6 step 5): if the seller is this card's
	// creator, cap how much of their still-restricted retained allocation
	// they can sell right now — tighter than the general ownership check
	// above whenever the card hasn't fully vested yet.
	var creatorRetainedSharesSoldDelta int64
	if params.Type == domain.TransactionTypeSell && card.CreatorUserID != nil && *card.CreatorUserID == params.UserID {
		maxSellable, unlockedFromRestricted := creatorSellLimit(card, sharesOwned, now)
		if params.Shares > maxSellable {
			return nil, ErrRetainedSharesLocked
		}
		creatorRetainedSharesSoldDelta = params.Shares
		if creatorRetainedSharesSoldDelta > unlockedFromRestricted {
			creatorRetainedSharesSoldDelta = unlockedFromRestricted
		}
	}

	// No live drift ticker or demand tracking exists yet (later phases), so
	// both modifiers stay neutral — every field used here comes from this
	// single locked read, never a stale pre-lock fetch.
	curveParams := engine.CurveParams{BasePrice: card.BasePrice, Scale: card.Scale, DemandModifier: 1, DriftFactor: 1}

	cost, err := engine.ExecutionCost(card.CirculatingSupply, deltaShares, curveParams)
	if err != nil {
		return nil, fmt.Errorf("compute execution cost: %w", err)
	}

	// Wash-trade detection (§4.3): a punitive fee multiplier on the flagged
	// leg rather than outright rejection — it still executes.
	lastTrade, err := qtx.GetLastTradeByUserAndCard(ctx, db.GetLastTradeByUserAndCardParams{UserID: params.UserID, CardID: &params.CardID})
	var lastTradePtr *db.Transaction
	if err == nil {
		lastTradePtr = &lastTrade
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get last trade: %w", err)
	}
	fee := int64(math.Round(float64(computeFee(cost)) * washTradeMultiplier(lastTradePtr, params.Type, now)))

	// engine.ExecutionCost: positive cost = buyer pays, negative = seller
	// receives. tradeDelta = -cost correctly debits a buy and credits a
	// sell with one formula; the fee is always an additional debit.
	tradeDelta := -cost
	feeDelta := -fee
	newBalance := user.CurrencyBalance + tradeDelta + feeDelta
	if newBalance < 0 {
		return nil, ErrInsufficientBalance
	}

	newPrice, err := engine.SpotPrice(newSupply, curveParams)
	if err != nil {
		return nil, fmt.Errorf("compute new spot price: %w", err)
	}

	// Circuit breaker (§4.3): decide whether this trade's price move rolls
	// or trips the rolling window, persisted atomically with the rest of
	// the trade below.
	windowStartedAt, windowStartPrice, haltedUntil, triggered := nextCircuitBreakerState(
		card.CircuitBreakerWindowStartedAt, card.CircuitBreakerWindowStartPrice, card.CurrentPrice, newPrice, now,
	)
	if triggered {
		slog.Warn("CIRCUIT_BREAKER_TRIGGERED",
			"card_id", card.ID,
			"window_start_price", windowStartPrice,
			"new_price", newPrice,
			"halted_until", haltedUntil,
		)
	}

	updatedCard, err := qtx.UpdateCardAfterTrade(ctx, db.UpdateCardAfterTradeParams{
		ID:                             params.CardID,
		CirculatingSupply:              newSupply,
		CurrentPrice:                   newPrice,
		CreatorRetainedSharesSold:      creatorRetainedSharesSoldDelta,
		CircuitBreakerWindowStartedAt:  windowStartedAt,
		CircuitBreakerWindowStartPrice: windowStartPrice,
		CircuitBreakerHaltedUntil:      haltedUntil,
	})
	if err != nil {
		return nil, fmt.Errorf("update card: %w", err)
	}

	updatedUser, err := qtx.ApplyBalanceDelta(ctx, db.ApplyBalanceDeltaParams{
		ID:              params.UserID,
		CurrencyBalance: tradeDelta + feeDelta,
	})
	if err != nil {
		return nil, fmt.Errorf("update user balance: %w", err)
	}

	newSharesOwned := sharesOwned + deltaShares
	newAvgCostBasis := avgCostBasis
	if params.Type == domain.TransactionTypeBuy {
		totalCostIncludingFee := cost + fee
		newAvgCostBasis = int64(math.Round(
			(float64(sharesOwned)*float64(avgCostBasis) + float64(totalCostIncludingFee)) / float64(newSharesOwned),
		))
	}
	// SELL leaves avg_cost_basis unchanged — selling doesn't change the
	// remaining shares' cost basis, even if newSharesOwned reaches 0.
	if _, err := qtx.UpsertHolding(ctx, db.UpsertHoldingParams{
		UserID:       params.UserID,
		CardID:       params.CardID,
		SharesOwned:  newSharesOwned,
		AvgCostBasis: newAvgCostBasis,
	}); err != nil {
		return nil, fmt.Errorf("upsert holding: %w", err)
	}

	pricePerShare := roundedPricePerShare(cost, params.Shares)
	idempotencyKey := params.IdempotencyKey
	tradeTxn, err := qtx.CreateTransaction(ctx, db.CreateTransactionParams{
		UserID:             params.UserID,
		CardID:             &params.CardID,
		Type:               string(params.Type),
		Shares:             &params.Shares,
		PricePerShare:      &pricePerShare,
		TotalCurrencyDelta: tradeDelta,
		ResultingBalance:   user.CurrencyBalance + tradeDelta,
		IdempotencyKey:     &idempotencyKey,
	})
	if err != nil {
		// A 23505 here is handled by the caller (ExecuteTrade), which
		// re-fetches fresh state after this transaction rolls back —
		// returned as-is, not wrapped, so errors.As still finds it.
		return nil, err
	}

	feeTxn, err := qtx.CreateTransaction(ctx, db.CreateTransactionParams{
		UserID:               params.UserID,
		CardID:               &params.CardID,
		Type:                 string(domain.TransactionTypeFee),
		TotalCurrencyDelta:   feeDelta,
		ResultingBalance:     newBalance,
		RelatedTransactionID: &tradeTxn.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("create fee transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &TradeResult{
		Transaction:    store.ToDomainTransaction(tradeTxn),
		FeeTransaction: store.ToDomainTransaction(feeTxn),
		User:           store.ToDomainUser(updatedUser),
		Card:           store.ToDomainCard(updatedCard),
	}, nil
}

// buildReplayResult reconstructs a TradeResult from already-committed rows
// for an idempotent retry — no locks, no mutation, never anything computed
// inside a transaction that hit a duplicate-key conflict and rolled back.
func (l *Ledger) buildReplayResult(ctx context.Context, existing db.Transaction, params TradeParams) (*TradeResult, error) {
	if existing.CardID == nil || *existing.CardID != params.CardID ||
		existing.Shares == nil || *existing.Shares != params.Shares ||
		existing.Type != string(params.Type) {
		return nil, ErrIdempotencyKeyMismatch
	}

	feeTxn, err := l.queries.GetRelatedFeeTransaction(ctx, &existing.ID)
	if err != nil {
		return nil, fmt.Errorf("fetch related fee transaction: %w", err)
	}
	user, err := l.queries.GetUserByID(ctx, params.UserID)
	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", err)
	}
	card, err := l.queries.GetCardByID(ctx, params.CardID)
	if err != nil {
		return nil, fmt.Errorf("fetch card: %w", err)
	}

	return &TradeResult{
		Transaction:    store.ToDomainTransaction(existing),
		FeeTransaction: store.ToDomainTransaction(feeTxn),
		User:           store.ToDomainUser(user),
		Card:           store.ToDomainCard(card),
	}, nil
}
