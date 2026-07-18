package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/ekansh-exe/navx/internal/api"
	"github.com/ekansh-exe/navx/internal/auth"
	"github.com/ekansh-exe/navx/internal/bots"
	"github.com/ekansh-exe/navx/internal/leaderboard"
	"github.com/ekansh-exe/navx/internal/ledger"
	"github.com/ekansh-exe/navx/internal/news"
)

const jwtTTL = 24 * time.Hour

// defaultBotRebalanceInterval matches §4.5's "nightly" cadence; overridable
// via BOT_REBALANCE_INTERVAL (e.g. "2m") for local testing/demoing.
const defaultBotRebalanceInterval = 24 * time.Hour

// defaultNewsGenerationInterval/defaultLeaderboardRefreshInterval match §9
// (unspecified, chosen for this phase) and §8's explicit "e.g. every 60s".
// Both overridable the same way (NEWS_GENERATION_INTERVAL,
// LEADERBOARD_REFRESH_INTERVAL) for local testing/demoing.
const (
	defaultNewsGenerationInterval     = 6 * time.Hour
	defaultLeaderboardRefreshInterval = 60 * time.Second
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("REDIS_URL is required")
	}

	rebalanceInterval := defaultBotRebalanceInterval
	if raw := os.Getenv("BOT_REBALANCE_INTERVAL"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			log.Fatalf("invalid BOT_REBALANCE_INTERVAL %q: %v", raw, err)
		}
		rebalanceInterval = parsed
	}

	newsInterval := defaultNewsGenerationInterval
	if raw := os.Getenv("NEWS_GENERATION_INTERVAL"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			log.Fatalf("invalid NEWS_GENERATION_INTERVAL %q: %v", raw, err)
		}
		newsInterval = parsed
	}

	leaderboardInterval := defaultLeaderboardRefreshInterval
	if raw := os.Getenv("LEADERBOARD_REFRESH_INTERVAL"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			log.Fatalf("invalid LEADERBOARD_REFRESH_INTERVAL %q: %v", raw, err)
		}
		leaderboardInterval = parsed
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to create db pool: %v", err)
	}
	defer pool.Close()

	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("invalid REDIS_URL: %v", err)
	}
	redisClient := redis.NewClient(redisOpts)
	defer redisClient.Close()

	ledgerSvc := ledger.New(pool)
	authSvc := auth.NewService(pool, ledgerSvc, []byte(jwtSecret), jwtTTL)
	newsReader := news.NewReader(pool)
	apiHandler := api.NewHandler(authSvc, ledgerSvc, newsReader, redisClient)

	// Market bots (§4.5): background goroutines trading through the exact
	// same ledgerSvc.ExecuteTrade the HTTP handlers above call — no special
	// access. Started here rather than blocking main(), and shares ctx so
	// they stop cleanly on the same SIGINT/SIGTERM as the HTTP server.
	bots.Run(ctx, pool, ledgerSvc, rebalanceInterval)

	// News generation (§9) and the leaderboard refresh job (§8) — same
	// fire-and-forget goroutine pattern, same shared ctx for shutdown.
	go news.Run(ctx, pool, newsInterval)
	go leaderboard.Run(ctx, pool, redisClient, leaderboardInterval)

	r := chi.NewRouter()
	r.Get("/health", healthHandler(pool))
	r.Post("/api/auth/register", apiHandler.Register)
	r.Post("/api/auth/login", apiHandler.Login)
	r.Get("/api/news", apiHandler.ListNews)
	r.Get("/api/leaderboard", apiHandler.Leaderboard)
	r.Group(func(r chi.Router) {
		r.Use(api.RequireAuth([]byte(jwtSecret)))
		r.Get("/api/users/me", apiHandler.Me)
		r.Post("/api/trades/quote", apiHandler.Quote)
		r.Post("/api/trades/execute", apiHandler.ExecuteTrade)
		r.Post("/api/cards", apiHandler.LaunchCard)
	})

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("navx server listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func healthHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		w.Header().Set("Content-Type", "application/json")

		if err := pool.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "error",
				"db":     err.Error(),
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"db":     "ok",
		})
	}
}
