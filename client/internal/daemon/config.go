package daemon

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerURL         string
	MachineID         string
	BufferPath        string
	HeartbeatWindow   time.Duration
	SecondBucket      time.Duration
	RequestTimeout    time.Duration
	OfflineFlushEvery time.Duration
}

func LoadConfigFromEnv() Config {
	windowSeconds := envInt("ATK_HEARTBEAT_WINDOW_SECONDS", 300)
	return Config{
		ServerURL:         envOrDefault("ATK_SERVER_URL", "http://127.0.0.1:8080"),
		MachineID:         envOrDefault("ATK_MACHINE_ID", "unknown-machine"),
		BufferPath:        envOrDefault("ATK_BUFFER_PATH", "/var/lib/atk-tracker/buffer.db"),
		HeartbeatWindow:   time.Duration(windowSeconds) * time.Second,
		SecondBucket:      1 * time.Second,
		RequestTimeout:    time.Duration(envInt("ATK_REQUEST_TIMEOUT_SECONDS", 10)) * time.Second,
		OfflineFlushEvery: time.Duration(envInt("ATK_FLUSH_INTERVAL_SECONDS", 30)) * time.Second,
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
