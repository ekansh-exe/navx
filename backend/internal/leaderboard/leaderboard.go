// Package leaderboard implements §8: ranking human users by net worth
// (currency_balance + the value of their holdings at current card prices),
// computed on a scheduled job rather than per-request, and cached in Redis
// so GET /api/leaderboard never has to join holdings x cards x users live.
package leaderboard

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/ekansh-exe/navx/internal/store/db"
)

// CacheKey is the single Redis key the refresh job writes and the API
// handler reads — a plain JSON-array string value (not a native Redis
// sorted-set), per this phase's explicit design.
const CacheKey = "leaderboard:top100"

// Entry is one ranked row, shared by both the writer (refreshOnce) and the
// reader (ReadCached / the API handler) so the JSON shape only ever lives
// in one place.
type Entry struct {
	Rank                  int       `json:"rank"`
	UserID                uuid.UUID `json:"user_id"`
	Username              string    `json:"username"`
	NetWorth              int64     `json:"net_worth"`
	ChangeFromLastRefresh *int64    `json:"change_from_last_refresh,omitempty"`

	// IsGoat/NetWorthDisplay back the GOAT tribute row (see GoatEntry) — a
	// fixed decoration, not real ranking data, so both are omitted entirely
	// for every normal entry.
	IsGoat          bool   `json:"is_goat,omitempty"`
	NetWorthDisplay string `json:"net_worth_display,omitempty"`
}

// goatUsername/goatNetWorthDisplay are the fixed tribute row's display
// values — never computed, never changing.
const (
	goatUsername        = "Nav28"
	goatNetWorthDisplay = "Arc Reactor of Developer"
)

// GoatEntry is a fixed, hardcoded tribute row — not a real user, not backed
// by any database row, and never included in ComputeLeaderboard, the Redis
// cache, or the rank-change diff in buildEntries. UserID is the nil UUID
// (never matches a real user) purely so Entry's JSON shape stays uniform.
func GoatEntry() Entry {
	return Entry{
		IsGoat:          true,
		Username:        goatUsername,
		NetWorthDisplay: goatNetWorthDisplay,
	}
}

// PrependGoat puts the GOAT tribute row first, ahead of every real ranked
// entry — called at the point entries leave this package (the REST handler,
// the WS broadcast) so the cached/diffed data itself never has to account for
// it. Always returns at least the GOAT entry, even when entries is empty.
func PrependGoat(entries []Entry) []Entry {
	return append([]Entry{GoatEntry()}, entries...)
}

// Observer receives the freshly-computed leaderboard after each successful
// refresh so the real-time push layer can broadcast it (§7). Optional and
// nil-safe — the refresh job works with no observer wired at all.
type Observer func([]Entry)

// Run drives the leaderboard-refresh job until ctx is cancelled: one
// immediate computation on startup, then one every interval thereafter —
// mirrors internal/bots's RunRebalanceJob and internal/news's Run. observer,
// if non-nil, is invoked with the entries produced by each successful refresh.
func Run(ctx context.Context, pool *pgxpool.Pool, redisClient *redis.Client, interval time.Duration, observer Observer) {
	queries := db.New(pool)

	notify(refreshOnce(ctx, queries, redisClient), observer)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			notify(refreshOnce(ctx, queries, redisClient), observer)
		}
	}
}

func notify(entries []Entry, observer Observer) {
	if observer != nil && entries != nil {
		observer(entries)
	}
}

// refreshOnce computes the current leaderboard and overwrites the cache —
// no TTL, so a stalled job leaves the last-known leaderboard in place
// rather than making it vanish. Returns the computed entries on success, or
// nil if the refresh failed before the cache was updated.
func refreshOnce(ctx context.Context, queries *db.Queries, redisClient *redis.Client) []Entry {
	prev, err := ReadCached(ctx, redisClient)
	if err != nil {
		slog.Error("LEADERBOARD_REFRESH_ERROR", "step", "read previous cache", "error", err)
		prev = nil
	}

	rows, err := queries.ComputeLeaderboard(ctx)
	if err != nil {
		slog.Error("LEADERBOARD_REFRESH_ERROR", "step", "compute", "error", err)
		return nil
	}

	entries := buildEntries(rows, prev)

	data, err := json.Marshal(entries)
	if err != nil {
		slog.Error("LEADERBOARD_REFRESH_ERROR", "step", "marshal", "error", err)
		return nil
	}
	if err := redisClient.Set(ctx, CacheKey, data, 0).Err(); err != nil {
		slog.Error("LEADERBOARD_REFRESH_ERROR", "step", "redis set", "error", err)
		return nil
	}
	slog.Info("LEADERBOARD_REFRESHED", "entries", len(entries))
	return entries
}

// buildEntries assigns 1-based rank in the SQL's already-sorted order and
// diffs each user's net_worth against its value in prev (nil if the user
// wasn't present last refresh). Pure — no DB/Redis calls — for direct unit
// testing.
func buildEntries(rows []db.ComputeLeaderboardRow, prev []Entry) []Entry {
	prevNetWorth := make(map[uuid.UUID]int64, len(prev))
	for _, p := range prev {
		prevNetWorth[p.UserID] = p.NetWorth
	}

	entries := make([]Entry, len(rows))
	for i, row := range rows {
		entry := Entry{
			Rank:     i + 1,
			UserID:   row.UserID,
			Username: row.Username,
			NetWorth: row.NetWorth,
		}
		if last, ok := prevNetWorth[row.UserID]; ok {
			change := row.NetWorth - last
			entry.ChangeFromLastRefresh = &change
		}
		entries[i] = entry
	}
	return entries
}

// ReadCached returns the cached leaderboard, or an empty (non-nil) slice if
// nothing has been cached yet — never an error just for "cache empty",
// since that's an expected state in the first moments after boot.
func ReadCached(ctx context.Context, redisClient *redis.Client) ([]Entry, error) {
	data, err := redisClient.Get(ctx, CacheKey).Bytes()
	if errors.Is(err, redis.Nil) {
		return []Entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}
