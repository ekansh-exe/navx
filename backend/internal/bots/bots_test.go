package bots

import (
	"context"
	"testing"
	"time"

	"github.com/ekansh-exe/navx/internal/ledger"
)

// TestRun_StartsWithoutPanicAndRespectsContextCancellation is a smoke test:
// it doesn't wait long enough for any bot's jittered timer (2-5s) to fire —
// tick-level behavior is covered directly in bot_test.go — it only confirms
// Run wires up account lookups and goroutines without panicking, and that
// everything exits promptly once ctx is cancelled.
func TestRun_StartsWithoutPanicAndRespectsContextCancellation(t *testing.T) {
	pool := testPool(t)
	ledgerSvc := ledger.New(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	Run(ctx, pool, ledgerSvc, time.Hour)

	<-ctx.Done()
	time.Sleep(50 * time.Millisecond)
}
