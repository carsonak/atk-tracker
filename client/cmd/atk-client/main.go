package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"atk-tracker/client/internal/daemon"
)

func main() {
	cfg := daemon.LoadConfigFromEnv()
	d, err := daemon.New(cfg)
	if err != nil {
		log.Fatalf("failed to initialize daemon: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	defer stop()

	if err := d.Run(ctx); err != nil {
		log.Fatalf("daemon stopped with error: %v", err)
	}
}
