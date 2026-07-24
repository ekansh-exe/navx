package bots

import (
	"context"
	"errors"
	"hash/fnv"
	"log/slog"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/ledger"
	"github.com/ekansh-exe/navx/internal/safety"
	"github.com/ekansh-exe/navx/internal/store/db"
)

// minTickInterval/maxTickInterval back §4.5's "a ticker every few seconds
// per bot, jittered so they don't all fire in lockstep" — a fresh random
// duration in this range is chosen before every tick, not a fixed period
// plus noise.
const (
	minTickInterval = 2 * time.Second
	maxTickInterval = 5 * time.Second
)

// Bot is a single running persona instance — one goroutine, one
// user_type=BOT account, trading exclusively through Ledger.ExecuteTrade
// (§4.5: "not a raw database write, not a bypass of any check").
type Bot struct {
	UserID   uuid.UUID
	Username string
	Persona  Persona

	ledger  *ledger.Ledger
	queries *db.Queries
	rng     *rand.Rand
	history map[uuid.UUID][]int64

	// newsCursor*/newsQueue back PersonaNewsReactive only (news_reactive.go)
	// — every other persona leaves them at their zero value.
	newsCursorSeen     bool
	newsCursorLastSeen time.Time
	newsQueue          []Decision
}

func newBot(userID uuid.UUID, username string, persona Persona, ledgerSvc *ledger.Ledger, queries *db.Queries) *Bot {
	h := fnv.New64a()
	h.Write([]byte(username))
	seed := int64(h.Sum64()) ^ time.Now().UnixNano()
	return &Bot{
		UserID:   userID,
		Username: username,
		Persona:  persona,
		ledger:   ledgerSvc,
		queries:  queries,
		rng:      rand.New(rand.NewSource(seed)),
		history:  make(map[uuid.UUID][]int64),
	}
}

// Run drives this bot's trading loop until ctx is cancelled.
func (b *Bot) Run(ctx context.Context) {
	for {
		timer := time.NewTimer(jitteredInterval(b.rng))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			b.tick(ctx)
		}
	}
}

func jitteredInterval(rng *rand.Rand) time.Duration {
	span := maxTickInterval - minTickInterval
	return minTickInterval + time.Duration(rng.Int63n(int64(span)))
}

// tick fetches the current market, asks this bot's persona for a decision,
// and — if there is one — executes it through the same trade-execution
// path a human's HTTP request uses. Never panics, never exits the loop:
// anything that goes wrong is logged and the bot tries again next tick.
func (b *Bot) tick(ctx context.Context) {
	// One bot's tick panicking must never crash the other 9 bots, the HTTP
	// server, or every open WebSocket connection with it — see
	// internal/safety's doc comment.
	defer safety.Recover("bot_tick:" + b.Username)

	snapshots, err := fetchSnapshots(ctx, b.queries, b.UserID, b.history)
	if err != nil {
		slog.Error("BOT_TICK_ERROR", "bot", b.Username, "persona", b.Persona, "error", err)
		return
	}

	decision := b.decide(ctx, snapshots)
	if decision == nil {
		return
	}

	// Personas size trades by a percentage of the target card's supply, which
	// far exceeds what a bot's balance can pay for on a large, expensive card.
	// Cap every BUY to what this bot can actually afford right now (SELLs pass
	// through) so trades execute and prices move, instead of being rejected as
	// insufficient-balance. Fetch the live balance here rather than from the
	// snapshot read so it reflects any trade the bot committed last tick.
	user, err := b.queries.GetUserByID(ctx, b.UserID)
	if err != nil {
		slog.Error("BOT_TICK_ERROR", "bot", b.Username, "persona", b.Persona, "error", err)
		return
	}
	decision = capBuyToBalance(decision, snapshots, user.CurrencyBalance)
	if decision == nil {
		return
	}

	result, err := b.ledger.ExecuteTrade(ctx, ledger.TradeParams{
		UserID:         b.UserID,
		CardID:         decision.CardID,
		Type:           decision.Type,
		Shares:         decision.Shares,
		IdempotencyKey: uuid.NewString(),
	})
	if err != nil {
		if isSkippableTradeError(err) {
			slog.Info("BOT_TRADE_SKIPPED",
				"bot", b.Username, "persona", b.Persona,
				"type", decision.Type, "shares", decision.Shares, "reason", err.Error())
			return
		}
		slog.Error("BOT_TRADE_ERROR", "bot", b.Username, "persona", b.Persona, "error", err)
		return
	}

	slog.Info("BOT_TRADE_EXECUTED",
		"bot", b.Username, "persona", b.Persona,
		"symbol", result.Card.Symbol, "type", result.Transaction.Type,
		"shares", *result.Transaction.Shares, "price_per_share", *result.Transaction.PricePerShare,
		"fee", -result.FeeTransaction.TotalCurrencyDelta, "balance", result.User.CurrencyBalance)
}

func (b *Bot) decide(ctx context.Context, snapshots []MarketSnapshot) *Decision {
	switch b.Persona {
	case PersonaMomentum:
		return decideMomentum(b.rng, snapshots)
	case PersonaContrarian:
		return decideContrarian(b.rng, snapshots)
	case PersonaRandomWalker:
		return decideRandomWalker(b.rng, snapshots)
	case PersonaNewsReactive:
		return b.decideNewsReactive(ctx, snapshots)
	case PersonaIndexTracker:
		var indexCardIDs []uuid.UUID
		for _, s := range snapshots {
			if s.CardType == domain.CardTypeIndex {
				indexCardIDs = append(indexCardIDs, s.CardID)
			}
		}
		if len(indexCardIDs) == 0 {
			return nil
		}
		derived, err := computeDerivedPrices(ctx, b.queries, indexCardIDs)
		if err != nil {
			slog.Error("BOT_TICK_ERROR", "bot", b.Username, "persona", b.Persona, "error", err)
			return nil
		}
		return decideIndexTracker(b.rng, snapshots, derived)
	default:
		return nil
	}
}

// isSkippableTradeError reports whether err is one of the expected
// business-rule rejections a bot should quietly back off from — exactly
// the same errors a human's HTTP request can hit — rather than a genuine
// system failure worth surfacing loudly.
func isSkippableTradeError(err error) bool {
	for _, sentinel := range []error{
		ledger.ErrCardNotTradable,
		ledger.ErrCircuitBreakerActive,
		ledger.ErrPositionCapExceeded,
		ledger.ErrInsufficientSupply,
		ledger.ErrInsufficientBalance,
		ledger.ErrInsufficientShares,
	} {
		if errors.Is(err, sentinel) {
			return true
		}
	}
	return false
}
