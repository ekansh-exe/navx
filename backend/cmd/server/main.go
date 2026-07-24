package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/ekansh-exe/navx/internal/api"
	"github.com/ekansh-exe/navx/internal/auth"
	"github.com/ekansh-exe/navx/internal/bots"
	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/leaderboard"
	"github.com/ekansh-exe/navx/internal/ledger"
	"github.com/ekansh-exe/navx/internal/migrate"
	"github.com/ekansh-exe/navx/internal/news"
	"github.com/ekansh-exe/navx/internal/quests"
	"github.com/ekansh-exe/navx/internal/ws"
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

// defaultQuestResetCheckInterval/defaultQuestCheckInterval back §7's daily
// quests (Phase 9). The reset job only needs to notice a due row sometime
// within its window, not fire at exactly midnight (see quests.RunResetJob's
// doc comment); the hold-card/rank checks share the leaderboard's own ~60s
// cadence since REACH_RANK is only as fresh as the leaderboard cache anyway.
// Both overridable the same way as the intervals above.
const (
	defaultQuestResetCheckInterval = 1 * time.Hour
	defaultQuestCheckInterval      = 60 * time.Second
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

	questResetCheckInterval := defaultQuestResetCheckInterval
	if raw := os.Getenv("QUEST_RESET_CHECK_INTERVAL"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			log.Fatalf("invalid QUEST_RESET_CHECK_INTERVAL %q: %v", raw, err)
		}
		questResetCheckInterval = parsed
	}

	questCheckInterval := defaultQuestCheckInterval
	if raw := os.Getenv("QUEST_CHECK_INTERVAL"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			log.Fatalf("invalid QUEST_CHECK_INTERVAL %q: %v", raw, err)
		}
		questCheckInterval = parsed
	}

	// Local dev default: the Vite dev server's origin. Override for any
	// other frontend origin (a different port, a deployed URL, etc).
	corsOrigins := []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	if raw := os.Getenv("CORS_ALLOWED_ORIGINS"); raw != "" {
		corsOrigins = strings.Split(raw, ",")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Applies every embedded migration (migrations/*.sql) before anything
	// else touches the database — so a fresh deploy (Render or otherwise)
	// never depends on a human running `migrate ... up` by hand first. A
	// no-op if the schema's already current.
	if err := migrate.Up(dbURL); err != nil {
		log.Fatalf("failed to apply migrations: %v", err)
	}

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
	questsSvc := quests.New(pool)
	apiHandler := api.NewHandler(authSvc, ledgerSvc, newsReader, redisClient, questsSvc)

	// Real-time push layer (§7): one in-process hub fed by nil-safe observer
	// hooks. Every committed trade — human HTTP request or bot goroutine, both
	// go through ledgerSvc.ExecuteTrade — emits a price tick; every generated
	// news event emits a news_published. The hub authorizes browser upgrades
	// against the same origins as CORS.
	hub := ws.NewHub(originHosts(corsOrigins))
	ledgerSvc.SetTradeObserver(func(e ledger.TradeEvent) {
		hub.PublishPriceTick(e.CardID, e.Symbol, e.Price, e.PreviousPrice, e.Volume, e.Timestamp)
	})

	// Market bots (§4.5): background goroutines trading through the exact
	// same ledgerSvc.ExecuteTrade the HTTP handlers above call — no special
	// access. Started here rather than blocking main(), and shares ctx so
	// they stop cleanly on the same SIGINT/SIGTERM as the HTTP server.
	bots.Run(ctx, pool, ledgerSvc, rebalanceInterval)

	// News generation (§9), the leaderboard refresh job (§8), and daily
	// quests' background jobs (§7, Phase 9) — same fire-and-forget
	// goroutine pattern, same shared ctx for shutdown.
	go news.Run(ctx, pool, newsInterval, func(e *domain.NewsEvent) { hub.PublishNews(e) })
	go leaderboard.Run(ctx, pool, redisClient, leaderboardInterval, func(e []leaderboard.Entry) {
		hub.PublishLeaderboard(leaderboard.PrependGoat(e))
	})
	go questsSvc.RunResetJob(ctx, questResetCheckInterval)
	go questsSvc.RunHoldCardCheckJob(ctx, questCheckInterval)
	go questsSvc.RunRankCheckJob(ctx, redisClient, questCheckInterval)

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Get("/health", healthHandler(pool))
	r.Post("/api/auth/register", apiHandler.Register)
	r.Post("/api/auth/login", apiHandler.Login)
	r.Get("/api/news", apiHandler.ListNews)
	r.Get("/api/leaderboard", apiHandler.Leaderboard)
	r.Get("/api/cards", apiHandler.ListCards)
	r.Get("/api/cards/{id}", apiHandler.GetCard)

	// Real-time push channels (§7) — public, upgrade-only, no auth.
	r.Get("/ws/prices", hub.ServeWS(ws.TopicPrices))
	r.Get("/ws/news", hub.ServeWS(ws.TopicNews))
	r.Get("/ws/leaderboard", hub.ServeWS(ws.TopicLeaderboard))
	r.Group(func(r chi.Router) {
		r.Use(api.RequireAuth([]byte(jwtSecret)))
		r.Get("/api/users/me", apiHandler.Me)
		r.Get("/api/users/me/holdings", apiHandler.Holdings)
		r.Get("/api/users/me/trades", apiHandler.Trades)
		r.Post("/api/trades/quote", apiHandler.Quote)
		r.Post("/api/trades/execute", apiHandler.ExecuteTrade)
		r.Post("/api/cards", apiHandler.LaunchCard)
		r.Get("/api/quests", apiHandler.Quests)
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

// originHosts strips the scheme from each CORS origin URL, yielding the
// host[:port] patterns websocket.Accept matches the browser's Origin header
// against (e.g. "http://localhost:5173" -> "localhost:5173").
func originHosts(origins []string) []string {
	hosts := make([]string, 0, len(origins))
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if i := strings.Index(o, "://"); i >= 0 {
			o = o[i+3:]
		}
		o = strings.TrimSuffix(o, "/")
		if o != "" {
			hosts = append(hosts, o)
		}
	}
	return hosts
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
