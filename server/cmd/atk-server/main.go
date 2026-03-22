package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"atk-tracker/server/internal/api"
	"atk-tracker/server/internal/db"
	"atk-tracker/server/internal/live"
	"atk-tracker/server/internal/rollup"
)

func main() {
	dsn := envOrDefault("ATK_DATABASE_URL", "postgres://localhost:5432/atk_tracker?sslmode=disable")
	addr := envOrDefault("ATK_SERVER_ADDR", ":8080")

	store, err := db.NewStore(context.Background(), dsn)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer store.Close()

	liveTracker := live.NewTracker(10 * time.Minute)
	handler := api.NewHandler(store, liveTracker)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	worker := rollup.NewWorker(store, 24*time.Hour)
	go worker.Start(ctx)

	srv := &http.Server{
		Addr:    addr,
		Handler: handler.Routes(),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("server listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
