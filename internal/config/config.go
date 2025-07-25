package config

import (
	"os"
	"strconv"
)

type Config struct {
	RedisURL             string
	ListenAddr           string
	ProcessorDefaultURL  string
	ProcessorFallbackURL string
	Workers              int
}

func Load() Config {
	workers := 2
	if w := os.Getenv("WORKERS"); w != "" {
		if n, err := strconv.Atoi(w); err == nil {
			workers = n
		}
	}
	return Config{
		RedisURL:             getenv("REDIS_URL", "redis:6379"),
		ListenAddr:           getenv("LISTEN_ADDR", ":8080"),
		ProcessorDefaultURL:  getenv("PROCESSOR_DEFAULT_URL", "http://payment-processor-default:8080"),
		ProcessorFallbackURL: getenv("PROCESSOR_FALLBACK_URL", "http://payment-processor-fallback:8080"),
		Workers:              workers,
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
