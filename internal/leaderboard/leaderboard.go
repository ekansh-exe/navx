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
}

// Run drives the leaderboard-refresh job until ctx is cancelled: one
// immediate computation on startup, then one every interval thereafter —
// mirrors internal/bots's RunRebalanceJob and internal/news's Run.
func Run(ctx context.Context, pool *pgxpool.Pool, redisClient *redis.Client, interval time.Duration) {
	queries := db.New(pool)

	refreshOnce(ctx, queries, redisClient)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refreshOnce(ctx, queries, redisClient)
		}
	}
}

// refreshOnce computes the current leaderboard and overwrites the cache —
// no TTL, so a stalled job leaves the last-known leaderboard in place
// rather than making it vanish.
func refreshOnce(ctx context.Context, queries *db.Queries, redisClient *redis.Client) {
	prev, err := ReadCached(ctx, redisClient)
	if err != nil {
		slog.Error("LEADERBOARD_REFRESH_ERROR", "step", "read previous cache", "error", err)
		prev = nil
	}

	rows, err := queries.ComputeLeaderboard(ctx)
	if err != nil {
		slog.Error("LEADERBOARD_REFRESH_ERROR", "step", "compute", "error", err)
		return
	}

	entries := buildEntries(rows, prev)

	data, err := json.Marshal(entries)
	if err != nil {
		slog.Error("LEADERBOARD_REFRESH_ERROR", "step", "marshal", "error", err)
		return
	}
	if err := redisClient.Set(ctx, CacheKey, data, 0).Err(); err != nil {
		slog.Error("LEADERBOARD_REFRESH_ERROR", "step", "redis set", "error", err)
		return
	}
	slog.Info("LEADERBOARD_REFRESHED", "entries", len(entries))
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
