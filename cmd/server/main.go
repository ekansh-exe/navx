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

	"github.com/ekansh-exe/navx/internal/api"
	"github.com/ekansh-exe/navx/internal/auth"
	"github.com/ekansh-exe/navx/internal/ledger"
)

const jwtTTL = 24 * time.Hour

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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to create db pool: %v", err)
	}
	defer pool.Close()

	ledgerSvc := ledger.New(pool)
	authSvc := auth.NewService(pool, ledgerSvc, []byte(jwtSecret), jwtTTL)
	apiHandler := api.NewHandler(authSvc, ledgerSvc)

	r := chi.NewRouter()
	r.Get("/health", healthHandler(pool))
	r.Post("/api/auth/register", apiHandler.Register)
	r.Post("/api/auth/login", apiHandler.Login)
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
