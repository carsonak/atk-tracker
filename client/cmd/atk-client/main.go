package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"atk-tracker/client/internal/daemon"
)

func main() {
	devInput, err := os.Open("/dev/input")
	if err != nil && errors.Is(err, os.ErrPermission) {
		log.Fatalf("FATAL: Insufficient permissions to read /dev/input/. Ensure daemon runs as root or is in the 'input' group.")
	}
	if devInput != nil {
		_ = devInput.Close()
	}

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
